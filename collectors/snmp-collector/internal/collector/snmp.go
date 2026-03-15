package collector

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/gosnmp/gosnmp"
	_ "github.com/lib/pq"
)

type Device struct {
	ID           int64
	Hostname     string
	IPAddress    string
	SNMPVersion  string // v2c or v3
	Community    string // For v2c
	Username     string // For v3
	AuthProtocol string // For v3
	AuthPassword string // For v3
	PrivProtocol string // For v3
	PrivPassword string // For v3
	Status       string
}

type SNMPCollector struct {
	db              *sql.DB
	metricsWriter   MetricsWriter
	devices         []Device
	devicesMutex    sync.RWMutex
	pollingInterval time.Duration
	reloadInterval  time.Duration
	maxConcurrent   int
	timeout         time.Duration
	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
	retryManager    *RetryManager
}

type MetricsWriter interface {
	WriteMetric(name string, value float64, labels map[string]string, timestamp time.Time) error
}

func NewSNMPCollector(db *sql.DB, metricsWriter MetricsWriter, pollingInterval, reloadInterval time.Duration, maxConcurrent int, timeout time.Duration) *SNMPCollector {
	ctx, cancel := context.WithCancel(context.Background())
	return &SNMPCollector{
		db:              db,
		metricsWriter:   metricsWriter,
		devices:         make([]Device, 0),
		pollingInterval: pollingInterval,
		reloadInterval:  reloadInterval,
		maxConcurrent:   maxConcurrent,
		timeout:         timeout,
		ctx:             ctx,
		cancel:          cancel,
		retryManager:    NewRetryManager(),
	}
}

func (c *SNMPCollector) Start() error {
	// Initial device load
	if err := c.loadDevices(); err != nil {
		return fmt.Errorf("failed to load devices: %w", err)
	}

	log.Printf("Loaded %d SNMP devices", len(c.devices))

	// Start device reload goroutine
	c.wg.Add(1)
	go c.deviceReloadLoop()

	// Start polling goroutine
	c.wg.Add(1)
	go c.pollingLoop()

	// Start SNMP trap listener on UDP port 162
	go func() {
		if err := c.startTrapListener(162); err != nil {
			log.Printf("SNMP trap listener stopped: %v", err)
		}
	}()

	return nil
}

func (c *SNMPCollector) Stop() {
	log.Println("Stopping SNMP collector...")
	c.cancel()
	c.wg.Wait()
	log.Println("SNMP collector stopped")
}

func (c *SNMPCollector) loadDevices() error {
	query := `
		SELECT 
			d.id, 
			d.hostname, 
			d.ip_address,
			COALESCE(dc.protocol, 'snmp_v2c') as snmp_version,
			COALESCE(dc.community_string, '') as community,
			COALESCE(dc.username, '') as username,
			COALESCE(dc.auth_protocol, '') as auth_protocol,
			COALESCE(dc.auth_password, '') as auth_password,
			COALESCE(dc.priv_protocol, '') as priv_protocol,
			COALESCE(dc.priv_password, '') as priv_password,
			d.status
		FROM devices d
		LEFT JOIN device_credentials dc ON d.id = dc.device_id
		WHERE d.protocol = 'snmp' AND d.status != 'deleted'
	`

	rows, err := c.db.Query(query)
	if err != nil {
		return fmt.Errorf("failed to query devices: %w", err)
	}
	defer rows.Close()

	devices := make([]Device, 0)
	for rows.Next() {
		var d Device
		if err := rows.Scan(&d.ID, &d.Hostname, &d.IPAddress, &d.SNMPVersion, &d.Community, &d.Username, &d.AuthProtocol, &d.AuthPassword, &d.PrivProtocol, &d.PrivPassword, &d.Status); err != nil {
			log.Printf("Error scanning device row: %v", err)
			continue
		}
		devices = append(devices, d)
	}

	c.devicesMutex.Lock()
	c.devices = devices
	c.devicesMutex.Unlock()

	return nil
}

func (c *SNMPCollector) deviceReloadLoop() {
	defer c.wg.Done()

	ticker := time.NewTicker(c.reloadInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			if err := c.loadDevices(); err != nil {
				log.Printf("Error reloading devices: %v", err)
			} else {
				c.devicesMutex.RLock()
				deviceCount := len(c.devices)
				c.devicesMutex.RUnlock()
				log.Printf("Reloaded %d SNMP devices", deviceCount)
			}
		}
	}
}

func (c *SNMPCollector) pollingLoop() {
	defer c.wg.Done()

	ticker := time.NewTicker(c.pollingInterval)
	defer ticker.Stop()

	// Poll immediately on start
	c.pollAllDevices()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.pollAllDevices()
		}
	}
}

func (c *SNMPCollector) pollAllDevices() {
	c.devicesMutex.RLock()
	devices := make([]Device, len(c.devices))
	copy(devices, c.devices)
	c.devicesMutex.RUnlock()

	if len(devices) == 0 {
		return
	}

	log.Printf("Starting poll cycle for %d devices", len(devices))

	// Create semaphore for concurrent polling
	sem := make(chan struct{}, c.maxConcurrent)
	var wg sync.WaitGroup

	for _, device := range devices {
		wg.Add(1)
		go func(d Device) {
			defer wg.Done()

			// Acquire semaphore
			sem <- struct{}{}
			defer func() { <-sem }()

			c.PollDeviceWithRetry(d)
		}(device)
	}

	wg.Wait()
	log.Printf("Poll cycle completed for %d devices", len(devices))
}

func (c *SNMPCollector) pollUPSMetrics(snmpClient *gosnmp.GoSNMP, device Device, timestamp time.Time) error {
	// UPS OIDs (standard UPS-MIB)
	oids := map[string]string{
		"battery_charge":  "1.3.6.1.2.1.33.1.2.4.0",     // upsBatteryStatus
		"input_voltage":   "1.3.6.1.2.1.33.1.3.3.1.3.1", // upsInputVoltage
		"output_voltage":  "1.3.6.1.2.1.33.1.4.4.1.2.1", // upsOutputVoltage
		"load_percent":    "1.3.6.1.2.1.33.1.4.4.1.5.1", // upsOutputPercentLoad
		"runtime_minutes": "1.3.6.1.2.1.33.1.2.3.0",     // upsEstimatedMinutesRemaining
		"battery_status":  "1.3.6.1.2.1.33.1.2.1.0",     // upsBatteryStatus
	}

	labels := map[string]string{
		"device_id": fmt.Sprintf("%d", device.ID),
		"hostname":  device.Hostname,
	}

	// Poll each OID
	for metricName, oid := range oids {
		result, err := snmpClient.Get([]string{oid})
		if err != nil {
			log.Printf("Error polling OID %s for device %s: %v", oid, device.Hostname, err)
			continue
		}

		if len(result.Variables) > 0 {
			variable := result.Variables[0]
			var value float64

			switch variable.Type {
			case gosnmp.Integer:
				value = float64(variable.Value.(int))
			case gosnmp.Gauge32:
				value = float64(variable.Value.(uint))
			case gosnmp.Counter32:
				value = float64(variable.Value.(uint))
			case gosnmp.Counter64:
				value = float64(variable.Value.(uint64))
			default:
				log.Printf("Unsupported SNMP type %v for OID %s", variable.Type, oid)
				continue
			}

			// Write metric to VictoriaMetrics
			metricFullName := fmt.Sprintf("infrasense_snmp_ups_%s", metricName)
			if err := c.metricsWriter.WriteMetric(metricFullName, value, labels, timestamp); err != nil {
				log.Printf("Error writing metric %s: %v", metricFullName, err)
			}
		}
	}

	return nil
}

func (c *SNMPCollector) updateDeviceStatus(deviceID int64, status string, errorMsg string) {
	query := `UPDATE devices SET status = $1, updated_at = NOW() WHERE id = $2`
	if _, err := c.db.Exec(query, status, deviceID); err != nil {
		log.Printf("Error updating device status: %v", err)
		return
	}

	// Log the status update with device_id, timestamp, and error message
	log.Printf("Updated device %d status to %s at %s: %s", deviceID, status, time.Now().Format(time.RFC3339), errorMsg)

	// Update collector_status table
	statusQuery := `
		INSERT INTO collector_status (collector_name, device_id, last_poll_time, last_error)
		VALUES ('snmp-collector', $1, NOW(), $2)
		ON CONFLICT (collector_name, device_id) 
		DO UPDATE SET 
			last_poll_time = NOW(),
			last_error = $2
	`
	if _, err := c.db.Exec(statusQuery, deviceID, errorMsg); err != nil {
		log.Printf("Error updating collector status: %v", err)
	}
}

func (c *SNMPCollector) updateCollectorStatusSuccess(deviceID int64) {
	statusQuery := `
		INSERT INTO collector_status (collector_name, device_id, last_poll_time, last_success_time, last_error)
		VALUES ('snmp-collector', $1, NOW(), NOW(), '')
		ON CONFLICT (collector_name, device_id) 
		DO UPDATE SET 
			last_poll_time = NOW(),
			last_success_time = NOW(),
			last_error = ''
	`
	if _, err := c.db.Exec(statusQuery, deviceID); err != nil {
		log.Printf("Error updating collector status: %v", err)
	}
}

func (c *SNMPCollector) GetDeviceCount() int {
	c.devicesMutex.RLock()
	defer c.devicesMutex.RUnlock()
	return len(c.devices)
}

// startTrapListener starts a UDP SNMP trap listener on the given port.
// On receiving a trap it extracts the source IP, looks up the matching device,
// and emits an infrasense_snmp_trap_received counter metric with trap_oid and device_id labels.
func (c *SNMPCollector) startTrapListener(port int) error {
	tl := gosnmp.NewTrapListener()
	tl.Params = gosnmp.Default

	tl.OnNewTrap = func(packet *gosnmp.SnmpPacket, addr *net.UDPAddr) {
		timestamp := time.Now()

		// Extract source IP (strip port if present)
		srcIP := ""
		if addr != nil {
			srcIP = addr.IP.String()
		}

		// Look up device by IP address
		c.devicesMutex.RLock()
		var device *Device
		for i := range c.devices {
			if c.devices[i].IPAddress == srcIP {
				device = &c.devices[i]
				break
			}
		}
		c.devicesMutex.RUnlock()

		deviceID := "unknown"
		if device != nil {
			deviceID = fmt.Sprintf("%d", device.ID)
		}

		// Emit one counter metric per trap variable OID
		for _, v := range packet.Variables {
			trapOID := strings.TrimPrefix(v.Name, ".")
			labels := map[string]string{
				"trap_oid":  trapOID,
				"device_id": deviceID,
			}
			if err := c.metricsWriter.WriteMetric("infrasense_snmp_trap_received", 1, labels, timestamp); err != nil {
				log.Printf("Error writing trap metric for OID %s from %s: %v", trapOID, srcIP, err)
			}
		}

		log.Printf("SNMP trap received from %s (device_id=%s): %d variables", srcIP, deviceID, len(packet.Variables))
	}

	addr := fmt.Sprintf("0.0.0.0:%d", port)
	log.Printf("Starting SNMP trap listener on UDP %s", addr)
	if err := tl.Listen(addr); err != nil {
		return fmt.Errorf("SNMP trap listener on %s failed: %w", addr, err)
	}
	return nil
}
