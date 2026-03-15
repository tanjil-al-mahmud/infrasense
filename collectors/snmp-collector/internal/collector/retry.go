package collector

import (
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"github.com/gosnmp/gosnmp"
)

// isTimeoutError returns true if the error looks like an SNMP timeout.
func isTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "timeout") || strings.Contains(msg, "timed out") || strings.Contains(msg, "i/o timeout")
}

const (
	maxBackoffDuration = 10 * time.Minute
	initialBackoff     = 1 * time.Second
)

type RetryState struct {
	deviceID     int64
	hostname     string
	failureCount int
	lastAttempt  time.Time
	nextAttempt  time.Time
}

type RetryManager struct {
	states map[int64]*RetryState
}

func NewRetryManager() *RetryManager {
	return &RetryManager{
		states: make(map[int64]*RetryState),
	}
}

func (rm *RetryManager) ShouldRetry(deviceID int64, hostname string) bool {
	state, exists := rm.states[deviceID]
	if !exists {
		// First attempt
		return true
	}

	// Check if enough time has passed for next retry
	return time.Now().After(state.nextAttempt)
}

func (rm *RetryManager) RecordFailure(deviceID int64, hostname string) time.Duration {
	state, exists := rm.states[deviceID]
	if !exists {
		state = &RetryState{
			deviceID:     deviceID,
			hostname:     hostname,
			failureCount: 0,
		}
		rm.states[deviceID] = state
	}

	state.failureCount++
	state.lastAttempt = time.Now()

	// Calculate exponential backoff: 1s, 2s, 4s, 8s, ..., max 10 minutes
	backoff := time.Duration(math.Pow(2, float64(state.failureCount-1))) * initialBackoff
	if backoff > maxBackoffDuration {
		backoff = maxBackoffDuration
	}

	state.nextAttempt = time.Now().Add(backoff)

	log.Printf("Device %s failed %d times, next retry in %v", hostname, state.failureCount, backoff)

	return backoff
}

func (rm *RetryManager) RecordSuccess(deviceID int64) {
	// Reset failure count on success
	delete(rm.states, deviceID)
}

func (rm *RetryManager) GetFailureCount(deviceID int64) int {
	state, exists := rm.states[deviceID]
	if !exists {
		return 0
	}
	return state.failureCount
}

// PollDeviceWithRetry polls a device with exponential backoff retry logic
func (c *SNMPCollector) PollDeviceWithRetry(device Device) {
	// Check if we should retry this device
	if !c.retryManager.ShouldRetry(device.ID, device.Hostname) {
		return
	}

	log.Printf("Polling device %s (%s)", device.Hostname, device.IPAddress)

	// Enforce 10-second SNMP timeout per device poll (requirement 8.8)
	snmpTimeout := 10 * time.Second
	if c.timeout > 0 && c.timeout < snmpTimeout {
		snmpTimeout = c.timeout
	}

	// Create SNMP client
	snmpClient := &gosnmp.GoSNMP{
		Target:    device.IPAddress,
		Port:      161,
		Transport: "udp",
		Timeout:   snmpTimeout,
		Retries:   2,
	}

	// Configure SNMP version and authentication
	if device.SNMPVersion == "snmp_v3" {
		snmpClient.Version = gosnmp.Version3
		snmpClient.SecurityModel = gosnmp.UserSecurityModel
		snmpClient.MsgFlags = gosnmp.AuthPriv

		// Set authentication
		switch device.AuthProtocol {
		case "MD5":
			snmpClient.SecurityParameters = &gosnmp.UsmSecurityParameters{
				UserName:                 device.Username,
				AuthenticationProtocol:   gosnmp.MD5,
				AuthenticationPassphrase: device.AuthPassword,
				PrivacyProtocol:          gosnmp.DES,
				PrivacyPassphrase:        device.PrivPassword,
			}
		case "SHA":
			snmpClient.SecurityParameters = &gosnmp.UsmSecurityParameters{
				UserName:                 device.Username,
				AuthenticationProtocol:   gosnmp.SHA,
				AuthenticationPassphrase: device.AuthPassword,
				PrivacyProtocol:          gosnmp.AES,
				PrivacyPassphrase:        device.PrivPassword,
			}
		}
	} else {
		// Default to v2c
		snmpClient.Version = gosnmp.Version2c
		snmpClient.Community = device.Community
	}

	// Connect to device
	err := snmpClient.Connect()
	if err != nil {
		if isTimeoutError(err) {
			log.Printf("SNMP timeout connecting to device %s (id=%d, ip=%s): %v", device.Hostname, device.ID, device.IPAddress, err)
		} else {
			log.Printf("Failed to connect to device %s (%s): %v", device.Hostname, device.IPAddress, err)
		}

		// Record failure and get backoff duration
		backoff := c.retryManager.RecordFailure(device.ID, device.Hostname)

		// Update device status to unavailable
		c.updateDeviceStatus(device.ID, "unavailable", fmt.Sprintf("Connection failed: %v (retry in %v)", err, backoff))

		return
	}
	defer snmpClient.Conn.Close()

	timestamp := time.Now()

	// Poll all SNMP metrics
	if err := c.PollAllMetrics(snmpClient, device, timestamp); err != nil {
		if isTimeoutError(err) {
			log.Printf("SNMP timeout polling device %s (id=%d, ip=%s): %v", device.Hostname, device.ID, device.IPAddress, err)
			backoff := c.retryManager.RecordFailure(device.ID, device.Hostname)
			c.updateDeviceStatus(device.ID, "unavailable", fmt.Sprintf("SNMP timeout: %v (retry in %v)", err, backoff))
			return
		}
		log.Printf("Warning: partial poll errors for device %s: %v", device.Hostname, err)
		// Don't fail on partial errors — some OIDs may not be supported
	}

	// Record success (resets failure count)
	c.retryManager.RecordSuccess(device.ID)

	// Update device status to healthy
	c.updateDeviceStatus(device.ID, "healthy", "")

	// Update collector status with success
	c.updateCollectorStatusSuccess(device.ID)

	log.Printf("Successfully polled device %s (%s)", device.Hostname, device.IPAddress)
}
