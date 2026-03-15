package collector

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"time"
)

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

	slog.Warn("device poll failed, scheduling retry",
		"event", "device_poll_failure",
		"device_id", deviceID,
		"hostname", hostname,
		"failure_count", state.failureCount,
		"retry_in_seconds", backoff.Seconds())

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

// PollDevice polls a single device by running all 5 ipmitool commands with a 30-second
// context timeout. On any command failure it logs the command name, exit code, and stderr,
// marks the device unavailable, and returns false. On full success it pushes all collected
// metrics to VictoriaMetrics, marks the device healthy, and returns true.
func (c *IPMICollector) PollDevice(device Device) bool {
	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()

	var allMetrics []Metric

	// 1. chassis status → infrasense_ipmi_chassis_power_state
	chassisMetrics, err := runChassisStatus(ctx, device)
	if err != nil {
		c.updateDeviceStatus(device.ID, "unavailable", fmt.Sprintf("chassis status failed: %v", err))
		return false
	}
	allMetrics = append(allMetrics, chassisMetrics...)

	// 2. sensor list → infrasense_ipmi_sensor_value
	sensorMetrics, err := runSensorList(ctx, device)
	if err != nil {
		c.updateDeviceStatus(device.ID, "unavailable", fmt.Sprintf("sensor list failed: %v", err))
		return false
	}
	allMetrics = append(allMetrics, sensorMetrics...)

	// 3. sdr → sensor metadata enrichment (no metrics emitted directly)
	_, err = runSDR(ctx, device)
	if err != nil {
		c.updateDeviceStatus(device.ID, "unavailable", fmt.Sprintf("sdr failed: %v", err))
		return false
	}

	// 4. sel list → infrasense_ipmi_sel_entries_total{severity}
	selMetrics, err := runSELList(ctx, device)
	if err != nil {
		c.updateDeviceStatus(device.ID, "unavailable", fmt.Sprintf("sel list failed: %v", err))
		return false
	}
	allMetrics = append(allMetrics, selMetrics...)

	// 5. fru → upsert manufacturer/product/serial into device_inventory
	if err := runFRU(ctx, device, c.db); err != nil {
		c.updateDeviceStatus(device.ID, "unavailable", fmt.Sprintf("fru failed: %v", err))
		return false
	}

	// Push all collected metrics to VictoriaMetrics
	for _, metric := range allMetrics {
		if err := c.metricsWriter.WriteMetric(metric.Name, metric.Value, metric.Labels, metric.Timestamp); err != nil {
			slog.Error("metric write error",
				"event", "metric_write_error",
				"device_id", device.ID,
				"hostname", device.Hostname,
				"error", err.Error())
		}
	}

	return true
}

// PollDeviceWithRetry polls a device with exponential backoff retry logic
func (c *IPMICollector) PollDeviceWithRetry(device Device) {
	// Check if we should retry this device
	if !c.retryManager.ShouldRetry(device.ID, device.Hostname) {
		return
	}

	timestamp := time.Now()

	slog.Info("polling device",
		"event", "poll_attempt",
		"device_id", device.ID,
		"hostname", device.Hostname,
		"timestamp", timestamp.Format(time.RFC3339))

	ok := c.PollDevice(device)
	if !ok {
		slog.Error("device poll failed",
			"event", "poll_attempt",
			"device_id", device.ID,
			"hostname", device.Hostname,
			"timestamp", timestamp.Format(time.RFC3339),
			"result", "error")

		// Record failure and get backoff duration
		backoff := c.retryManager.RecordFailure(device.ID, device.Hostname)

		slog.Warn("device marked unavailable, scheduling retry",
			"event", "device_unavailable",
			"device_id", device.ID,
			"hostname", device.Hostname,
			"retry_in_seconds", backoff.Seconds())

		return
	}

	slog.Info("device poll successful",
		"event", "poll_attempt",
		"device_id", device.ID,
		"hostname", device.Hostname,
		"timestamp", timestamp.Format(time.RFC3339),
		"result", "success")

	// Record success (resets failure count)
	c.retryManager.RecordSuccess(device.ID)

	// Update device status to healthy
	c.updateDeviceStatus(device.ID, "healthy", "")

	// Update collector status with success
	c.updateCollectorStatusSuccess(device.ID)
}
