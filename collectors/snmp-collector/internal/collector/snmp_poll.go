package collector

import (
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/gosnmp/gosnmp"
)

// Metric is a single time-series data point emitted by a poll function.
type Metric struct {
	Name      string
	Value     float64
	Labels    map[string]string
	Timestamp time.Time
}

// Standard MIB OIDs
const (
	// System MIB (1.3.6.1.2.1.1)
	oidSysDescr    = "1.3.6.1.2.1.1.1.0"
	oidSysUptime   = "1.3.6.1.2.1.1.3.0"
	oidSysName     = "1.3.6.1.2.1.1.5.0"
	oidSysContact  = "1.3.6.1.2.1.1.4.0"
	oidSysLocation = "1.3.6.1.2.1.1.6.0"

	// UCD-SNMP CPU (1.3.6.1.4.1.2021.11)
	oidCPUUser   = "1.3.6.1.4.1.2021.11.9.0"
	oidCPUSystem = "1.3.6.1.4.1.2021.11.10.0"
	oidCPUIdle   = "1.3.6.1.4.1.2021.11.11.0"

	// UCD-SNMP Memory (1.3.6.1.4.1.2021.4)
	oidMemTotal  = "1.3.6.1.4.1.2021.4.5.0"
	oidMemAvail  = "1.3.6.1.4.1.2021.4.6.0"
	oidMemFree   = "1.3.6.1.4.1.2021.4.11.0"
	oidMemCached = "1.3.6.1.4.1.2021.4.15.0"

	// UCD-SNMP Disk (1.3.6.1.4.1.2021.9)
	oidDiskPath    = "1.3.6.1.4.1.2021.9.1.2"
	oidDiskTotal   = "1.3.6.1.4.1.2021.9.1.6"
	oidDiskAvail   = "1.3.6.1.4.1.2021.9.1.7"
	oidDiskUsed    = "1.3.6.1.4.1.2021.9.1.8"
	oidDiskUsedPct = "1.3.6.1.4.1.2021.9.1.9"

	// IF-MIB Network (1.3.6.1.2.1.2)
	oidIfDescr      = "1.3.6.1.2.1.2.2.1.2"
	oidIfOperStatus = "1.3.6.1.2.1.2.2.1.8"
	oidIfSpeed      = "1.3.6.1.2.1.2.2.1.5"
	oidIfInOctets   = "1.3.6.1.2.1.2.2.1.10"
	oidIfOutOctets  = "1.3.6.1.2.1.2.2.1.16"

	// Dell vendor OIDs (1.3.6.1.4.1.674)
	oidDellTempProbeStatus  = "1.3.6.1.4.1.674.10892.5.4.700.20.1.5"
	oidDellTempProbeReading = "1.3.6.1.4.1.674.10892.5.4.700.20.1.6"
	oidDellFanStatus        = "1.3.6.1.4.1.674.10892.5.4.700.12.1.5"
	oidDellFanReading       = "1.3.6.1.4.1.674.10892.5.4.700.12.1.6"
	oidDellPSUStatus        = "1.3.6.1.4.1.674.10892.5.4.600.12.1.5"
	oidDellRAIDStatus       = "1.3.6.1.4.1.674.10892.5.5.1.20.130.1.1.38"

	// Dell iDRAC / OMSA hardware health OIDs (used by pollDellVendorOIDs)
	oidDellEnterprisePrefix       = "1.3.6.1.4.1.674"
	oidSysObjectID                = "1.3.6.1.2.1.1.2.0"
	oidDellGlobalSystemStatus     = "1.3.6.1.4.1.674.10892.5.4.200.10.1.2"
	oidDellSystemLCDStatus        = "1.3.6.1.4.1.674.10892.5.4.200.10.1.4"
	oidDellPowerSupplyStatus      = "1.3.6.1.4.1.674.10892.5.4.300.10.1.5"
	oidDellTemperatureProbeStatus = "1.3.6.1.4.1.674.10892.5.4.600.12.1.5"
	oidDellFanDeviceStatus        = "1.3.6.1.4.1.674.10892.5.4.600.20.1.5"
)

// PollAllMetrics collects all SNMP metrics for a device
func (c *SNMPCollector) PollAllMetrics(snmpClient *gosnmp.GoSNMP, device Device, timestamp time.Time) error {
	var errs []string

	if err := c.pollSystemInfo(snmpClient, device, timestamp); err != nil {
		errs = append(errs, fmt.Sprintf("system: %v", err))
	}
	if err := c.pollCPUMetrics(snmpClient, device, timestamp); err != nil {
		errs = append(errs, fmt.Sprintf("cpu: %v", err))
	}
	if err := c.pollMemoryMetrics(snmpClient, device, timestamp); err != nil {
		errs = append(errs, fmt.Sprintf("memory: %v", err))
	}
	if err := c.pollDiskMetrics(snmpClient, device, timestamp); err != nil {
		errs = append(errs, fmt.Sprintf("disk: %v", err))
	}
	if err := c.pollDiskOIDs(snmpClient, device, timestamp); err != nil {
		errs = append(errs, fmt.Sprintf("disk_oids: %v", err))
	}
	if err := c.pollNetworkMetrics(snmpClient, device, timestamp); err != nil {
		errs = append(errs, fmt.Sprintf("network: %v", err))
	}
	if err := c.pollNetworkOIDs(snmpClient, device, timestamp); err != nil {
		errs = append(errs, fmt.Sprintf("network_oids: %v", err))
	}
	// Try Dell vendor OIDs (non-fatal if not Dell or OIDs unsupported)
	_ = c.pollDellVendorMetrics(snmpClient, device, timestamp)
	if _, err := c.pollDellVendorOIDs(snmpClient, device, timestamp); err != nil {
		log.Printf("Dell vendor OID poll skipped for device %s: %v", device.Hostname, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("partial poll errors: %s", strings.Join(errs, "; "))
	}
	return nil
}

func (c *SNMPCollector) pollSystemInfo(snmpClient *gosnmp.GoSNMP, device Device, timestamp time.Time) error {
	return c.pollSystemOIDs(snmpClient, device, timestamp)
}

// pollSystemOIDs polls system information OIDs under 1.3.6.1.2.1.1 and emits:
//   - infrasense_snmp_system_uptime_seconds (from sysUpTime .3.0)
//   - infrasense_snmp_system_info (gauge=1, label: description, from sysDescr .1.0)
//   - infrasense_snmp_system_name (gauge=1, label: name, from sysName .5.0)
func (c *SNMPCollector) pollSystemOIDs(snmpClient *gosnmp.GoSNMP, device Device, timestamp time.Time) error {
	oids := []string{oidSysDescr, oidSysUptime, oidSysName}
	result, err := snmpClient.Get(oids)
	if err != nil {
		return err
	}
	baseLabels := map[string]string{
		"device_id": fmt.Sprintf("%d", device.ID),
		"hostname":  device.Hostname,
	}
	for _, v := range result.Variables {
		oid := strings.TrimPrefix(v.Name, ".")
		switch oid {
		case oidSysUptime:
			if val, ok := toFloat64(v); ok {
				// sysUpTime is in hundredths of a second — convert to seconds
				_ = c.metricsWriter.WriteMetric("infrasense_snmp_system_uptime_seconds", val/100, baseLabels, timestamp)
			}
		case oidSysDescr:
			description := ""
			if s, ok := v.Value.(string); ok {
				description = s
			} else if b, ok := v.Value.([]byte); ok {
				description = string(b)
			}
			if description != "" {
				l := copyLabels(baseLabels)
				l["description"] = description
				_ = c.metricsWriter.WriteMetric("infrasense_snmp_system_info", 1, l, timestamp)
			}
		case oidSysName:
			name := ""
			if s, ok := v.Value.(string); ok {
				name = s
			} else if b, ok := v.Value.([]byte); ok {
				name = string(b)
			}
			if name != "" {
				l := copyLabels(baseLabels)
				l["name"] = name
				_ = c.metricsWriter.WriteMetric("infrasense_snmp_system_name", 1, l, timestamp)
			}
		}
	}
	return nil
}

// pollCPUOIDs polls CPU utilization OIDs under 1.3.6.1.4.1.2021.11 (UCD-SNMP-MIB) and emits:
//   - infrasense_snmp_cpu_user_percent   (from ssCpuUser   .9.0)
//   - infrasense_snmp_cpu_system_percent (from ssCpuSystem .10.0)
//   - infrasense_snmp_cpu_idle_percent   (from ssCpuIdle   .11.0)
func (c *SNMPCollector) pollCPUOIDs(snmpClient *gosnmp.GoSNMP, device Device, timestamp time.Time) error {
	oids := []string{oidCPUUser, oidCPUSystem, oidCPUIdle}
	result, err := snmpClient.Get(oids)
	if err != nil {
		return err
	}
	labels := map[string]string{
		"device_id": fmt.Sprintf("%d", device.ID),
		"hostname":  device.Hostname,
	}
	metricMap := map[string]string{
		oidCPUUser:   "infrasense_snmp_cpu_user_percent",
		oidCPUSystem: "infrasense_snmp_cpu_system_percent",
		oidCPUIdle:   "infrasense_snmp_cpu_idle_percent",
	}
	for _, v := range result.Variables {
		oid := strings.TrimPrefix(v.Name, ".")
		if name, ok := metricMap[oid]; ok {
			if val, ok := toFloat64(v); ok {
				_ = c.metricsWriter.WriteMetric(name, val, labels, timestamp)
			}
		}
	}
	return nil
}

func (c *SNMPCollector) pollCPUMetrics(snmpClient *gosnmp.GoSNMP, device Device, timestamp time.Time) error {
	return c.pollCPUOIDs(snmpClient, device, timestamp)
}

// pollMemoryOIDs polls memory OIDs under 1.3.6.1.4.1.2021.4 (UCD-SNMP-MIB) and emits:
//   - infrasense_snmp_memory_total_kb     (from memTotalReal .5.0)
//   - infrasense_snmp_memory_available_kb (from memAvailReal .6.0)
//   - infrasense_snmp_memory_free_kb      (from memTotalFree .11.0)
func (c *SNMPCollector) pollMemoryOIDs(snmpClient *gosnmp.GoSNMP, device Device, timestamp time.Time) error {
	oids := []string{oidMemTotal, oidMemAvail, oidMemFree}
	result, err := snmpClient.Get(oids)
	if err != nil {
		return err
	}
	labels := map[string]string{
		"device_id": fmt.Sprintf("%d", device.ID),
		"hostname":  device.Hostname,
	}
	metricMap := map[string]string{
		oidMemTotal: "infrasense_snmp_memory_total_kb",
		oidMemAvail: "infrasense_snmp_memory_available_kb",
		oidMemFree:  "infrasense_snmp_memory_free_kb",
	}
	for _, v := range result.Variables {
		oid := strings.TrimPrefix(v.Name, ".")
		if name, ok := metricMap[oid]; ok {
			if val, ok := toFloat64(v); ok {
				_ = c.metricsWriter.WriteMetric(name, val, labels, timestamp)
			}
		}
	}
	return nil
}

func (c *SNMPCollector) pollMemoryMetrics(snmpClient *gosnmp.GoSNMP, device Device, timestamp time.Time) error {
	return c.pollMemoryOIDs(snmpClient, device, timestamp)
}

// pollDiskOIDs walks the UCD-SNMP-MIB disk table under 1.3.6.1.4.1.2021.9 and emits:
//   - infrasense_snmp_disk_total_kb{mount_point} (from dskTotal .6)
//   - infrasense_snmp_disk_used_kb{mount_point}  (from dskUsed  .8)
func (c *SNMPCollector) pollDiskOIDs(snmpClient *gosnmp.GoSNMP, device Device, timestamp time.Time) error {
	mountPoints := make(map[string]string) // index -> mount path
	diskTotals := make(map[string]float64) // index -> total KB
	diskUsed := make(map[string]float64)   // index -> used KB

	// Walk dskPath (.2) to collect mount point labels
	_ = snmpClient.Walk(oidDiskPath, func(pdu gosnmp.SnmpPDU) error {
		idx := lastOIDComponent(pdu.Name)
		if s, ok := pdu.Value.(string); ok {
			mountPoints[idx] = s
		} else if b, ok := pdu.Value.([]byte); ok {
			mountPoints[idx] = string(b)
		}
		return nil
	})

	// Walk dskTotal (.6)
	_ = snmpClient.Walk(oidDiskTotal, func(pdu gosnmp.SnmpPDU) error {
		idx := lastOIDComponent(pdu.Name)
		if val, ok := toFloat64(pdu); ok {
			diskTotals[idx] = val
		}
		return nil
	})

	// Walk dskUsed (.8)
	_ = snmpClient.Walk(oidDiskUsed, func(pdu gosnmp.SnmpPDU) error {
		idx := lastOIDComponent(pdu.Name)
		if val, ok := toFloat64(pdu); ok {
			diskUsed[idx] = val
		}
		return nil
	})

	for idx, mountPath := range mountPoints {
		labels := map[string]string{
			"device_id":   fmt.Sprintf("%d", device.ID),
			"hostname":    device.Hostname,
			"mount_point": mountPath,
		}
		if v, ok := diskTotals[idx]; ok {
			_ = c.metricsWriter.WriteMetric("infrasense_snmp_disk_total_kb", v, labels, timestamp)
		}
		if v, ok := diskUsed[idx]; ok {
			_ = c.metricsWriter.WriteMetric("infrasense_snmp_disk_used_kb", v, labels, timestamp)
		}
	}
	return nil
}

func (c *SNMPCollector) pollDiskMetrics(snmpClient *gosnmp.GoSNMP, device Device, timestamp time.Time) error {
	// Walk disk table
	diskPaths := make(map[string]string)   // index -> path
	diskTotals := make(map[string]float64) // index -> total KB
	diskAvails := make(map[string]float64) // index -> avail KB
	diskUsedPct := make(map[string]float64)

	walkAndCollect := func(baseOID string, dest map[string]float64) {
		_ = snmpClient.Walk(baseOID, func(pdu gosnmp.SnmpPDU) error {
			idx := lastOIDComponent(pdu.Name)
			if val, ok := toFloat64(pdu); ok {
				dest[idx] = val
			}
			return nil
		})
	}

	_ = snmpClient.Walk(oidDiskPath, func(pdu gosnmp.SnmpPDU) error {
		idx := lastOIDComponent(pdu.Name)
		if s, ok := pdu.Value.(string); ok {
			diskPaths[idx] = s
		} else if b, ok := pdu.Value.([]byte); ok {
			diskPaths[idx] = string(b)
		}
		return nil
	})
	walkAndCollect(oidDiskTotal, diskTotals)
	walkAndCollect(oidDiskAvail, diskAvails)
	walkAndCollect(oidDiskUsedPct, diskUsedPct)

	for idx, path := range diskPaths {
		labels := map[string]string{
			"device_id": fmt.Sprintf("%d", device.ID),
			"hostname":  device.Hostname,
			"mount":     path,
		}
		if v, ok := diskTotals[idx]; ok {
			_ = c.metricsWriter.WriteMetric("infrasense_snmp_disk_total_kb", v, labels, timestamp)
		}
		if v, ok := diskAvails[idx]; ok {
			_ = c.metricsWriter.WriteMetric("infrasense_snmp_disk_available_kb", v, labels, timestamp)
		}
		if v, ok := diskUsedPct[idx]; ok {
			_ = c.metricsWriter.WriteMetric("infrasense_snmp_disk_used_percent", v, labels, timestamp)
		}
	}
	return nil
}

// pollNetworkOIDs walks IF-MIB under 1.3.6.1.2.1.2.* and emits bytes in/out per interface:
//   - infrasense_snmp_net_in_bytes{device_id, hostname, interface}  (from ifInOctets  1.3.6.1.2.1.2.2.1.10)
//   - infrasense_snmp_net_out_bytes{device_id, hostname, interface} (from ifOutOctets 1.3.6.1.2.1.2.2.1.16)
func (c *SNMPCollector) pollNetworkOIDs(snmpClient *gosnmp.GoSNMP, device Device, timestamp time.Time) error {
	ifNames := make(map[string]string)    // ifIndex -> interface name
	ifInOcts := make(map[string]float64)  // ifIndex -> in octets
	ifOutOcts := make(map[string]float64) // ifIndex -> out octets

	// Walk ifDescr to build ifIndex -> interface name map
	_ = snmpClient.Walk(oidIfDescr, func(pdu gosnmp.SnmpPDU) error {
		idx := lastOIDComponent(pdu.Name)
		if s, ok := pdu.Value.(string); ok {
			ifNames[idx] = s
		} else if b, ok := pdu.Value.([]byte); ok {
			ifNames[idx] = string(b)
		}
		return nil
	})

	// Walk ifInOctets
	_ = snmpClient.Walk(oidIfInOctets, func(pdu gosnmp.SnmpPDU) error {
		idx := lastOIDComponent(pdu.Name)
		if val, ok := toFloat64(pdu); ok {
			ifInOcts[idx] = val
		}
		return nil
	})

	// Walk ifOutOctets
	_ = snmpClient.Walk(oidIfOutOctets, func(pdu gosnmp.SnmpPDU) error {
		idx := lastOIDComponent(pdu.Name)
		if val, ok := toFloat64(pdu); ok {
			ifOutOcts[idx] = val
		}
		return nil
	})

	for idx, ifName := range ifNames {
		labels := map[string]string{
			"device_id": fmt.Sprintf("%d", device.ID),
			"hostname":  device.Hostname,
			"interface": ifName,
		}
		if v, ok := ifInOcts[idx]; ok {
			_ = c.metricsWriter.WriteMetric("infrasense_snmp_net_in_bytes", v, labels, timestamp)
		}
		if v, ok := ifOutOcts[idx]; ok {
			_ = c.metricsWriter.WriteMetric("infrasense_snmp_net_out_bytes", v, labels, timestamp)
		}
	}
	return nil
}

func (c *SNMPCollector) pollNetworkMetrics(snmpClient *gosnmp.GoSNMP, device Device, timestamp time.Time) error {
	ifNames := make(map[string]string)
	ifStatus := make(map[string]float64)
	ifSpeed := make(map[string]float64)
	ifInOctets := make(map[string]float64)
	ifOutOctets := make(map[string]float64)

	_ = snmpClient.Walk(oidIfDescr, func(pdu gosnmp.SnmpPDU) error {
		idx := lastOIDComponent(pdu.Name)
		if s, ok := pdu.Value.(string); ok {
			ifNames[idx] = s
		} else if b, ok := pdu.Value.([]byte); ok {
			ifNames[idx] = string(b)
		}
		return nil
	})

	walkFloat := func(baseOID string, dest map[string]float64) {
		_ = snmpClient.Walk(baseOID, func(pdu gosnmp.SnmpPDU) error {
			idx := lastOIDComponent(pdu.Name)
			if val, ok := toFloat64(pdu); ok {
				dest[idx] = val
			}
			return nil
		})
	}
	walkFloat(oidIfOperStatus, ifStatus)
	walkFloat(oidIfSpeed, ifSpeed)
	walkFloat(oidIfInOctets, ifInOctets)
	walkFloat(oidIfOutOctets, ifOutOctets)

	for idx, name := range ifNames {
		labels := map[string]string{
			"device_id": fmt.Sprintf("%d", device.ID),
			"hostname":  device.Hostname,
			"interface": name,
		}
		if v, ok := ifStatus[idx]; ok {
			_ = c.metricsWriter.WriteMetric("infrasense_snmp_interface_status", v, labels, timestamp)
		}
		if v, ok := ifSpeed[idx]; ok {
			_ = c.metricsWriter.WriteMetric("infrasense_snmp_interface_speed_bps", v, labels, timestamp)
		}
		if v, ok := ifInOctets[idx]; ok {
			_ = c.metricsWriter.WriteMetric("infrasense_snmp_interface_rx_bytes_total", v, labels, timestamp)
		}
		if v, ok := ifOutOctets[idx]; ok {
			_ = c.metricsWriter.WriteMetric("infrasense_snmp_interface_tx_bytes_total", v, labels, timestamp)
		}
	}
	return nil
}

// pollDellVendorOIDs polls Dell enterprise OIDs under 1.3.6.1.4.1.674.* for hardware health status.
// It first checks sysObjectID to confirm the device enterprise OID matches the Dell prefix
// (1.3.6.1.4.1.674). If the device is not Dell, it returns without error.
//
// Metrics emitted:
//   - infrasense_snmp_dell_global_system_status{device_id, hostname} — overall chassis health (OID .200.10.1.2)
//   - infrasense_snmp_dell_lcd_status{device_id, hostname}           — front-panel LCD status (OID .200.10.1.4)
//   - infrasense_snmp_dell_psu_status{device_id, hostname, psu_index}  — per-PSU status (table walk, OID .300.10.1.5)
//   - infrasense_snmp_dell_temp_status{device_id, hostname, probe_index} — per-temp-probe status (table walk, OID .600.12.1.5)
//   - infrasense_snmp_dell_fan_status{device_id, hostname, fan_index}  — per-fan status (table walk, OID .600.20.1.5)
func (c *SNMPCollector) pollDellVendorOIDs(snmpClient *gosnmp.GoSNMP, device Device, timestamp time.Time) ([]Metric, error) {
	// Check sysObjectID to confirm this is a Dell device
	result, err := snmpClient.Get([]string{oidSysObjectID})
	if err != nil {
		return nil, fmt.Errorf("failed to get sysObjectID: %w", err)
	}

	isDell := false
	for _, v := range result.Variables {
		var oidVal string
		switch val := v.Value.(type) {
		case string:
			oidVal = val
		case []byte:
			oidVal = string(val)
		}
		// sysObjectID is returned as an OID string; strip leading dot for comparison
		oidVal = strings.TrimPrefix(oidVal, ".")
		if strings.HasPrefix(oidVal, oidDellEnterprisePrefix) {
			isDell = true
			break
		}
	}

	if !isDell {
		return nil, nil
	}

	log.Printf("Device %s identified as Dell (enterprise OID matches %s), polling Dell hardware OIDs", device.Hostname, oidDellEnterprisePrefix)

	var metrics []Metric
	baseLabels := map[string]string{
		"device_id": fmt.Sprintf("%d", device.ID),
		"hostname":  device.Hostname,
	}

	// Poll scalar OIDs: Global System Status and LCD Status
	scalarResult, err := snmpClient.Get([]string{oidDellGlobalSystemStatus, oidDellSystemLCDStatus})
	if err != nil {
		log.Printf("Warning: failed to get Dell scalar OIDs for device %s: %v", device.Hostname, err)
	} else {
		for _, v := range scalarResult.Variables {
			oid := strings.TrimPrefix(v.Name, ".")
			if val, ok := toFloat64(v); ok {
				switch oid {
				case oidDellGlobalSystemStatus:
					l := copyLabels(baseLabels)
					_ = c.metricsWriter.WriteMetric("infrasense_snmp_dell_global_system_status", val, l, timestamp)
					metrics = append(metrics, Metric{Name: "infrasense_snmp_dell_global_system_status", Value: val, Labels: l, Timestamp: timestamp})
				case oidDellSystemLCDStatus:
					l := copyLabels(baseLabels)
					_ = c.metricsWriter.WriteMetric("infrasense_snmp_dell_lcd_status", val, l, timestamp)
					metrics = append(metrics, Metric{Name: "infrasense_snmp_dell_lcd_status", Value: val, Labels: l, Timestamp: timestamp})
				}
			}
		}
	}

	// Walk Power Supply Status table (1.3.6.1.4.1.674.10892.5.4.300.10.1.5)
	_ = snmpClient.Walk(oidDellPowerSupplyStatus, func(pdu gosnmp.SnmpPDU) error {
		idx := lastOIDComponent(pdu.Name)
		if val, ok := toFloat64(pdu); ok {
			l := copyLabels(baseLabels)
			l["psu_index"] = idx
			_ = c.metricsWriter.WriteMetric("infrasense_snmp_dell_psu_status", val, l, timestamp)
			metrics = append(metrics, Metric{Name: "infrasense_snmp_dell_psu_status", Value: val, Labels: l, Timestamp: timestamp})
		}
		return nil
	})

	// Walk Temperature Probe Status table (1.3.6.1.4.1.674.10892.5.4.600.12.1.5)
	_ = snmpClient.Walk(oidDellTemperatureProbeStatus, func(pdu gosnmp.SnmpPDU) error {
		idx := lastOIDComponent(pdu.Name)
		if val, ok := toFloat64(pdu); ok {
			l := copyLabels(baseLabels)
			l["probe_index"] = idx
			_ = c.metricsWriter.WriteMetric("infrasense_snmp_dell_temp_status", val, l, timestamp)
			metrics = append(metrics, Metric{Name: "infrasense_snmp_dell_temp_status", Value: val, Labels: l, Timestamp: timestamp})
		}
		return nil
	})

	// Walk Fan Status table (1.3.6.1.4.1.674.10892.5.4.600.20.1.5)
	_ = snmpClient.Walk(oidDellFanDeviceStatus, func(pdu gosnmp.SnmpPDU) error {
		idx := lastOIDComponent(pdu.Name)
		if val, ok := toFloat64(pdu); ok {
			l := copyLabels(baseLabels)
			l["fan_index"] = idx
			_ = c.metricsWriter.WriteMetric("infrasense_snmp_dell_fan_status", val, l, timestamp)
			metrics = append(metrics, Metric{Name: "infrasense_snmp_dell_fan_status", Value: val, Labels: l, Timestamp: timestamp})
		}
		return nil
	})

	return metrics, nil
}

func (c *SNMPCollector) pollDellVendorMetrics(snmpClient *gosnmp.GoSNMP, device Device, timestamp time.Time) error {
	labels := map[string]string{"device_id": fmt.Sprintf("%d", device.ID), "hostname": device.Hostname, "vendor": "dell"}

	// Temperature probes
	_ = snmpClient.Walk(oidDellTempProbeReading, func(pdu gosnmp.SnmpPDU) error {
		idx := lastOIDComponent(pdu.Name)
		if val, ok := toFloat64(pdu); ok {
			l := copyLabels(labels)
			l["probe_index"] = idx
			_ = c.metricsWriter.WriteMetric("infrasense_snmp_dell_temperature_celsius", val/10, l, timestamp)
		}
		return nil
	})

	// Fan readings
	_ = snmpClient.Walk(oidDellFanReading, func(pdu gosnmp.SnmpPDU) error {
		idx := lastOIDComponent(pdu.Name)
		if val, ok := toFloat64(pdu); ok {
			l := copyLabels(labels)
			l["fan_index"] = idx
			_ = c.metricsWriter.WriteMetric("infrasense_snmp_dell_fan_rpm", val, l, timestamp)
		}
		return nil
	})

	// PSU status
	_ = snmpClient.Walk(oidDellPSUStatus, func(pdu gosnmp.SnmpPDU) error {
		idx := lastOIDComponent(pdu.Name)
		if val, ok := toFloat64(pdu); ok {
			l := copyLabels(labels)
			l["psu_index"] = idx
			_ = c.metricsWriter.WriteMetric("infrasense_snmp_dell_psu_status", val, l, timestamp)
		}
		return nil
	})

	// RAID status
	_ = snmpClient.Walk(oidDellRAIDStatus, func(pdu gosnmp.SnmpPDU) error {
		idx := lastOIDComponent(pdu.Name)
		if val, ok := toFloat64(pdu); ok {
			l := copyLabels(labels)
			l["vdisk_index"] = idx
			_ = c.metricsWriter.WriteMetric("infrasense_snmp_dell_raid_status", val, l, timestamp)
		}
		return nil
	})

	return nil
}

// toFloat64 converts an SNMP PDU value to float64
func toFloat64(pdu gosnmp.SnmpPDU) (float64, bool) {
	switch v := pdu.Value.(type) {
	case int:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case uint:
		return float64(v), true
	case uint32:
		return float64(v), true
	case uint64:
		return float64(v), true
	case float32:
		return float64(v), true
	case float64:
		return v, true
	}
	return 0, false
}

// lastOIDComponent returns the last numeric component of an OID
func lastOIDComponent(oid string) string {
	parts := strings.Split(strings.TrimPrefix(oid, "."), ".")
	if len(parts) == 0 {
		return "0"
	}
	return parts[len(parts)-1]
}

func copyLabels(src map[string]string) map[string]string {
	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

// StartTrapListener starts an SNMP trap listener on UDP 162
func (c *SNMPCollector) StartTrapListener(port int) {
	tl := gosnmp.NewTrapListener()
	tl.OnNewTrap = func(packet *gosnmp.SnmpPacket, addr *net.UDPAddr) {
		log.Printf("SNMP trap received from %v: %d variables", addr, len(packet.Variables))
		timestamp := time.Now()
		srcAddr := ""
		if addr != nil {
			srcAddr = addr.String()
		}
		labels := map[string]string{"source": srcAddr, "trap_type": "snmp"}

		for _, v := range packet.Variables {
			oidStr := v.Name
			// Classify trap by OID prefix
			metricName := "infrasense_snmp_trap"
			if strings.HasPrefix(oidStr, "1.3.6.1.4.1.674") {
				metricName = "infrasense_snmp_trap_dell"
				labels["vendor"] = "dell"
			}
			if val, ok := toFloat64(v); ok {
				l := copyLabels(labels)
				l["oid"] = oidStr
				_ = c.metricsWriter.WriteMetric(metricName, val, l, timestamp)
			}
		}
	}
	tl.Params = gosnmp.Default

	addr := fmt.Sprintf("0.0.0.0:%d", port)
	log.Printf("Starting SNMP trap listener on %s", addr)
	if err := tl.Listen(addr); err != nil {
		log.Printf("SNMP trap listener error: %v", err)
	}
}
