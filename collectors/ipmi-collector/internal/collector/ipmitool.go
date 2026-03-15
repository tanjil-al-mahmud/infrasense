package collector

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Metric struct {
	Name      string
	Value     float64
	Labels    map[string]string
	Timestamp time.Time
}

type IPMIData struct {
	Metrics []Metric
	SELLogs []string
}

// SDRRecord holds sensor metadata from `ipmitool sdr` output.
// Used to enrich sensor readings with type classification.
type SDRRecord struct {
	SensorName string
	SensorID   string
	SensorType string
	EntityID   string
	Reading    string
	Status     string
}

// ExecuteIPMITool executes ipmitool command with proper argument escaping
func ExecuteIPMITool(ctx context.Context, host, username, password string, args ...string) (string, error) {
	// Validate inputs to prevent command injection
	if err := validateIPMIInput(host); err != nil {
		return "", fmt.Errorf("invalid host: %w", err)
	}
	if err := validateIPMIInput(username); err != nil {
		return "", fmt.Errorf("invalid username: %w", err)
	}

	// Build ipmitool command
	cmdArgs := []string{
		"-I", "lanplus",
		"-H", host,
		"-U", username,
		"-P", password,
	}
	cmdArgs = append(cmdArgs, args...)

	cmd := exec.CommandContext(ctx, "ipmitool", cmdArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("ipmitool command failed: %w (output: %s)", err, string(output))
	}

	return string(output), nil
}

// validateIPMIInput validates input to prevent command injection
func validateIPMIInput(input string) error {
	// Allow alphanumeric, dots, hyphens, underscores
	validPattern := regexp.MustCompile(`^[a-zA-Z0-9.\-_]+$`)
	if !validPattern.MatchString(input) {
		return fmt.Errorf("input contains invalid characters")
	}
	return nil
}

// runChassisStatus executes `ipmitool chassis status` and returns a power state metric.
// Emits infrasense_ipmi_chassis_power_state with value 1 (on) or 0 (off).
// On non-zero exit code, logs command, exit code, and stderr.
func runChassisStatus(ctx context.Context, device Device) ([]Metric, error) {
	cmdArgs := []string{
		"-I", "lanplus",
		"-H", device.IPAddress,
		"-U", device.Username,
		"-P", device.Password,
		"chassis", "status",
	}

	cmd := exec.CommandContext(ctx, "ipmitool", cmdArgs...)
	stdout, err := cmd.Output()
	if err != nil {
		exitCode := -1
		stderr := ""
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
			stderr = string(exitErr.Stderr)
		}
		slog.Error("ipmitool command failed",
			"event", "ipmitool_error",
			"command", "ipmitool chassis status",
			"device_id", device.ID,
			"hostname", device.Hostname,
			"exit_code", exitCode,
			"stderr", stderr,
		)
		return nil, fmt.Errorf("ipmitool chassis status failed (exit %d): %s", exitCode, stderr)
	}

	powerState := parseChassisPowerState(string(stdout))

	metric := Metric{
		Name:  "infrasense_ipmi_chassis_power_state",
		Value: powerState,
		Labels: map[string]string{
			"device_id": fmt.Sprintf("%d", device.ID),
			"hostname":  device.Hostname,
		},
		Timestamp: time.Now(),
	}

	return []Metric{metric}, nil
}

// parseChassisPowerState parses "System Power         : on/off" from ipmitool chassis status output.
// Returns 1.0 if power is on, 0.0 otherwise.
func parseChassisPowerState(output string) float64 {
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "System Power") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				state := strings.TrimSpace(strings.ToLower(parts[1]))
				if state == "on" {
					return 1.0
				}
				return 0.0
			}
		}
	}
	return 0.0
}

// runSensorList executes `ipmitool sensor` and returns sensor value metrics.
// Parses pipe-delimited CSV output and emits infrasense_ipmi_sensor_value
// with labels sensor_name, unit, device_id, hostname.
// Sensors with value "na" or "N/A" are skipped.
// On non-zero exit code, logs command, exit code, and stderr.
func runSensorList(ctx context.Context, device Device) ([]Metric, error) {
	cmdArgs := []string{
		"-I", "lanplus",
		"-H", device.IPAddress,
		"-U", device.Username,
		"-P", device.Password,
		"sensor",
	}

	cmd := exec.CommandContext(ctx, "ipmitool", cmdArgs...)
	stdout, err := cmd.Output()
	if err != nil {
		exitCode := -1
		stderr := ""
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
			stderr = string(exitErr.Stderr)
		}
		slog.Error("ipmitool command failed",
			"event", "ipmitool_error",
			"command", "ipmitool sensor",
			"device_id", device.ID,
			"hostname", device.Hostname,
			"exit_code", exitCode,
			"stderr", stderr,
		)
		return nil, fmt.Errorf("ipmitool sensor failed (exit %d): %s", exitCode, stderr)
	}

	return parseSensorList(string(stdout), device)
}

// parseSensorList parses ipmitool sensor pipe-delimited CSV output.
// Format: sensor_name | value | unit | status | lower_non_recoverable | lower_critical | lower_non_critical | upper_non_critical | upper_critical | upper_non_recoverable
func parseSensorList(output string, device Device) ([]Metric, error) {
	var metrics []Metric
	scanner := bufio.NewScanner(strings.NewReader(output))

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		fields := strings.Split(line, "|")
		if len(fields) < 3 {
			continue
		}

		sensorName := strings.TrimSpace(fields[0])
		valueStr := strings.TrimSpace(fields[1])
		unit := strings.TrimSpace(fields[2])

		// Skip unavailable sensors
		valueLower := strings.ToLower(valueStr)
		if valueLower == "na" || valueLower == "n/a" || valueStr == "" {
			continue
		}

		value, err := strconv.ParseFloat(valueStr, 64)
		if err != nil {
			continue
		}

		metrics = append(metrics, Metric{
			Name:  "infrasense_ipmi_sensor_value",
			Value: value,
			Labels: map[string]string{
				"sensor_name": sensorName,
				"unit":        unit,
				"device_id":   fmt.Sprintf("%d", device.ID),
				"hostname":    device.Hostname,
			},
			Timestamp: time.Now(),
		})
	}

	return metrics, nil
}

// runSDR executes `ipmitool sdr` and returns SDR records for sensor metadata enrichment.
// Each record captures sensor name, type, entity ID, reading, and status.
// On non-zero exit code, logs command, exit code, and stderr.
func runSDR(ctx context.Context, device Device) ([]SDRRecord, error) {
	cmdArgs := []string{
		"-I", "lanplus",
		"-H", device.IPAddress,
		"-U", device.Username,
		"-P", device.Password,
		"sdr",
	}

	cmd := exec.CommandContext(ctx, "ipmitool", cmdArgs...)
	stdout, err := cmd.Output()
	if err != nil {
		exitCode := -1
		stderr := ""
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
			stderr = string(exitErr.Stderr)
		}
		slog.Error("ipmitool command failed",
			"event", "ipmitool_error",
			"command", "ipmitool sdr",
			"device_id", device.ID,
			"hostname", device.Hostname,
			"exit_code", exitCode,
			"stderr", stderr,
		)
		return nil, fmt.Errorf("ipmitool sdr failed (exit %d): %s", exitCode, stderr)
	}

	return parseSDRRecords(string(stdout)), nil
}

// parseSDRRecords parses ipmitool sdr pipe-delimited output.
// Format: sensor_name | reading | status
// Returns a slice of SDRRecord with sensor metadata for cross-referencing.
func parseSDRRecords(output string) []SDRRecord {
	var records []SDRRecord
	scanner := bufio.NewScanner(strings.NewReader(output))

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		fields := strings.Split(line, "|")
		if len(fields) < 3 {
			continue
		}

		sensorName := strings.TrimSpace(fields[0])
		reading := strings.TrimSpace(fields[1])
		status := strings.TrimSpace(fields[2])

		if sensorName == "" {
			continue
		}

		records = append(records, SDRRecord{
			SensorName: sensorName,
			SensorType: determineSensorType(sensorName, ""),
			Reading:    reading,
			Status:     status,
		})
	}

	return records
}

// runSELList executes `ipmitool sel list` and returns SEL entry count metrics
// grouped by severity. Emits infrasense_ipmi_sel_entries_total{severity, device_id, hostname}.
// Severity is classified as "critical", "warning", or "info" based on event description keywords.
// On non-zero exit code, logs command, exit code, and stderr, then returns an error.
func runSELList(ctx context.Context, device Device) ([]Metric, error) {
	cmdArgs := []string{
		"-I", "lanplus",
		"-H", device.IPAddress,
		"-U", device.Username,
		"-P", device.Password,
		"sel", "list",
	}

	cmd := exec.CommandContext(ctx, "ipmitool", cmdArgs...)
	stdout, err := cmd.Output()
	if err != nil {
		exitCode := -1
		stderr := ""
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
			stderr = string(exitErr.Stderr)
		}
		slog.Error("ipmitool command failed",
			"event", "ipmitool_error",
			"command", "ipmitool sel list",
			"device_id", device.ID,
			"hostname", device.Hostname,
			"exit_code", exitCode,
			"stderr", stderr,
		)
		return nil, fmt.Errorf("ipmitool sel list failed (exit %d): %s", exitCode, stderr)
	}

	return parseSELEntries(string(stdout), device)
}

// parseSELEntries counts SEL entries by severity and returns one metric per severity level.
// SEL line format: <id> | <date> | <time> | <sensor> | <description> | <assertion>
// Severity classification:
//   - "critical" — description contains "critical", "non-recoverable", or "failure"
//   - "warning"  — description contains "warning", "non-critical", or "degraded"
//   - "info"     — all other entries
func parseSELEntries(output string, device Device) ([]Metric, error) {
	counts := map[string]float64{
		"critical": 0,
		"warning":  0,
		"info":     0,
	}

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "SEL has no entries") {
			continue
		}

		// Each SEL entry has at least 5 pipe-delimited fields.
		fields := strings.Split(line, "|")
		if len(fields) < 5 {
			continue
		}

		// The event description is in field index 4 (0-based).
		description := strings.ToLower(strings.TrimSpace(fields[4]))

		switch {
		case strings.Contains(description, "critical") ||
			strings.Contains(description, "non-recoverable") ||
			strings.Contains(description, "failure"):
			counts["critical"]++
		case strings.Contains(description, "warning") ||
			strings.Contains(description, "non-critical") ||
			strings.Contains(description, "degraded"):
			counts["warning"]++
		default:
			counts["info"]++
		}
	}

	var metrics []Metric
	for severity, count := range counts {
		metrics = append(metrics, Metric{
			Name:  "infrasense_ipmi_sel_entries_total",
			Value: count,
			Labels: map[string]string{
				"severity":  severity,
				"device_id": fmt.Sprintf("%d", device.ID),
				"hostname":  device.Hostname,
			},
			Timestamp: time.Now(),
		})
	}

	return metrics, nil
}

// runFRU executes `ipmitool fru` and upserts FRU inventory data into the device_inventory table.
// Parses Product Manufacturer, Product Name, and Product Serial from the output.
// On non-zero exit code, logs command, exit code, and stderr, then returns an error.
func runFRU(ctx context.Context, device Device, db *sql.DB) error {
	cmdArgs := []string{
		"-I", "lanplus",
		"-H", device.IPAddress,
		"-U", device.Username,
		"-P", device.Password,
		"fru",
	}

	cmd := exec.CommandContext(ctx, "ipmitool", cmdArgs...)
	stdout, err := cmd.Output()
	if err != nil {
		exitCode := -1
		stderr := ""
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
			stderr = string(exitErr.Stderr)
		}
		slog.Error("ipmitool command failed",
			"event", "ipmitool_error",
			"command", "ipmitool fru",
			"device_id", device.ID,
			"hostname", device.Hostname,
			"exit_code", exitCode,
			"stderr", stderr,
		)
		return fmt.Errorf("ipmitool fru failed (exit %d): %s", exitCode, stderr)
	}

	manufacturer, productName, serialNumber := parseFRUOutput(string(stdout))

	upsertQuery := `
		INSERT INTO device_inventory (device_id, manufacturer, system_model, service_tag, collected_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (device_id)
		DO UPDATE SET
			manufacturer  = EXCLUDED.manufacturer,
			system_model  = EXCLUDED.system_model,
			service_tag   = EXCLUDED.service_tag,
			collected_at  = EXCLUDED.collected_at
	`
	if _, err := db.ExecContext(ctx, upsertQuery, device.ID, manufacturer, productName, serialNumber); err != nil {
		return fmt.Errorf("failed to upsert FRU inventory for device %d: %w", device.ID, err)
	}

	slog.Info("FRU inventory upserted",
		"event", "fru_upserted",
		"device_id", device.ID,
		"hostname", device.Hostname,
		"manufacturer", manufacturer,
		"product_name", productName,
		"serial_number", serialNumber,
	)

	return nil
}

// parseFRUOutput parses ipmitool fru output and extracts Product Manufacturer,
// Product Name, and Product Serial values.
// Lines have the format: "  Product Manufacturer  : Dell Inc."
func parseFRUOutput(output string) (manufacturer, productName, serialNumber string) {
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "Product Manufacturer":
			manufacturer = value
		case "Product Name":
			productName = value
		case "Product Serial":
			serialNumber = value
		}
	}
	return
}

// CollectIPMIData collects all IPMI sensor data and SEL logs
func CollectIPMIData(ctx context.Context, device Device) (*IPMIData, error) {
	data := &IPMIData{
		Metrics: make([]Metric, 0),
		SELLogs: make([]string, 0),
	}

	// Collect chassis power state
	chassisMetrics, err := runChassisStatus(ctx, device)
	if err != nil {
		slog.Error("failed to collect chassis status",
			"event", "chassis_status_error",
			"device_id", device.ID,
			"hostname", device.Hostname,
			"error", err.Error(),
		)
		return nil, err
	}
	data.Metrics = append(data.Metrics, chassisMetrics...)

	// Collect sensor data - try 'sdr list full' first, fallback to 'sdr'
	sensorOutput, err := ExecuteIPMITool(ctx, device.IPAddress, device.Username, device.Password, "sdr", "list", "full")
	if err != nil {
		slog.Warn("ipmitool sdr list full failed, falling back to basic sdr", "host", device.IPAddress, "error", err)
		sensorOutput, err = ExecuteIPMITool(ctx, device.IPAddress, device.Username, device.Password, "sdr")
	}

	if err != nil {
		return nil, fmt.Errorf("failed to collect sensor data from both sdr list full and basic sdr: %w", err)
	}

	metrics, err := parseSensorData(sensorOutput, device.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse sensor data: %w", err)
	}
	data.Metrics = append(data.Metrics, metrics...)

	// Collect PSU status
	psuMetrics, err := collectPSUStatus(ctx, device)
	if err != nil {
		// Log error but continue with other metrics
		fmt.Printf("Warning: failed to collect PSU status for device %s: %v\n", device.Hostname, err)
	} else {
		data.Metrics = append(data.Metrics, psuMetrics...)
	}

	// Collect SEL logs
	selOutput, err := ExecuteIPMITool(ctx, device.IPAddress, device.Username, device.Password, "sel", "list")
	if err != nil {
		// Log error but continue
		fmt.Printf("Warning: failed to collect SEL logs for device %s: %v\n", device.Hostname, err)
	} else {
		data.SELLogs = parseSELLogs(selOutput)
	}

	return data, nil
}

// parseSensorData parses ipmitool sdr output
func parseSensorData(output string, deviceID int64) ([]Metric, error) {
	metrics := make([]Metric, 0)
	scanner := bufio.NewScanner(strings.NewReader(output))

	// Regular expression to parse sensor lines
	// Example: "CPU Temp        | 45 degrees C      | ok"
	sensorRegex := regexp.MustCompile(`^([^|]+)\s*\|\s*([^|]+)\s*\|\s*(.+)$`)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}
		matches := sensorRegex.FindStringSubmatch(line)
		if len(matches) != 4 {
			// Fallback: try split by | if regex fails
			parts := strings.Split(line, "|")
			if len(parts) >= 3 {
				sensorName := strings.TrimSpace(parts[0])
				valueStr := strings.TrimSpace(parts[1])
				status := strings.TrimSpace(parts[2])
				matches = []string{line, sensorName, valueStr, status}
			} else {
				continue
			}
		}

		sensorName := strings.TrimSpace(matches[1])
		valueStr := strings.TrimSpace(matches[2])
		status := strings.TrimSpace(matches[3])

		// Skip sensors that are definitely unavailable
		valLower := strings.ToLower(valueStr)
		if valLower == "na" || valLower == "n/a" || valLower == "none" {
			continue
		}

		// Extract numeric value and unit
		value, unit, err := parseValue(valueStr)
		if err != nil {
			// If not a number, map common discrete states
			if valLower == "ok" || valLower == "enabled" || valLower == "present" || valLower == "online" {
				value = 1.0
			} else if valLower == "failed" || valLower == "disabled" || valLower == "absent" || valLower == "offline" {
				value = 0.0
			} else {
				continue // Skip non-numeric and unknown discrete sensors
			}
		}

		// Determine sensor type
		sensorType := determineSensorType(sensorName, unit)

		metric := Metric{
			Name:  fmt.Sprintf("infrasense_ipmi_%s", sensorType),
			Value: value,
			Labels: map[string]string{
				"device_id":   fmt.Sprintf("%d", deviceID),
				"sensor_name": sensorName,
				"sensor_type": sensorType,
				"status":      status,
			},
			Timestamp: time.Now(),
		}

		metrics = append(metrics, metric)
	}

	return metrics, nil
}

// parseValue extracts numeric value and unit from sensor reading
func parseValue(valueStr string) (float64, string, error) {
	// Handle different formats: "45 degrees C", "1200 RPM", "12.5 Volts", "250 Watts"
	parts := strings.Fields(valueStr)
	if len(parts) == 0 {
		return 0, "", fmt.Errorf("empty value")
	}

	// Try to parse first part as number
	value, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0, "", fmt.Errorf("not a number: %s", parts[0])
	}

	unit := ""
	if len(parts) > 1 {
		unit = strings.ToLower(parts[1])
	}

	return value, unit, nil
}

// determineSensorType determines the metric type based on sensor name and unit
func determineSensorType(sensorName, unit string) string {
	sensorNameLower := strings.ToLower(sensorName)

	// Temperature sensors
	if strings.Contains(sensorNameLower, "temp") || strings.Contains(unit, "degrees") {
		if strings.Contains(sensorNameLower, "cpu") {
			return "temperature_cpu_celsius"
		} else if strings.Contains(sensorNameLower, "inlet") {
			return "temperature_inlet_celsius"
		} else if strings.Contains(sensorNameLower, "exhaust") {
			return "temperature_exhaust_celsius"
		} else if strings.Contains(sensorNameLower, "system") {
			return "temperature_system_celsius"
		}
		return "temperature_celsius"
	}

	// Fan sensors
	if strings.Contains(sensorNameLower, "fan") || strings.Contains(unit, "rpm") {
		return "fan_speed_rpm"
	}

	// Voltage sensors
	if strings.Contains(sensorNameLower, "volt") || strings.Contains(unit, "volts") {
		if strings.Contains(sensorNameLower, "12v") {
			return "voltage_12v"
		} else if strings.Contains(sensorNameLower, "5v") {
			return "voltage_5v"
		} else if strings.Contains(sensorNameLower, "3.3v") {
			return "voltage_3v3"
		} else if strings.Contains(sensorNameLower, "vcore") || strings.Contains(sensorNameLower, "cpu") {
			return "voltage_vcore"
		}
		return "voltage"
	}

	// Power sensors
	if strings.Contains(sensorNameLower, "power") || strings.Contains(sensorNameLower, "watt") || strings.Contains(unit, "watts") {
		return "power_watts"
	}

	return "other"
}

// collectPSUStatus collects PSU status information
func collectPSUStatus(ctx context.Context, device Device) ([]Metric, error) {
	output, err := ExecuteIPMITool(ctx, device.IPAddress, device.Username, device.Password, "sdr", "type", "Power Supply")
	if err != nil {
		return nil, err
	}

	metrics := make([]Metric, 0)
	scanner := bufio.NewScanner(strings.NewReader(output))

	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "Power Supply") {
			// Parse PSU status
			// Example: "PS1 Status       | 0x01              | ok"
			parts := strings.Split(line, "|")
			if len(parts) >= 3 {
				psuName := strings.TrimSpace(parts[0])
				status := strings.TrimSpace(parts[2])

				// Convert status to numeric (1 = ok, 0 = failed)
				statusValue := 0.0
				if strings.Contains(strings.ToLower(status), "ok") {
					statusValue = 1.0
				}

				metric := Metric{
					Name:  "infrasense_ipmi_psu_status",
					Value: statusValue,
					Labels: map[string]string{
						"device_id":   fmt.Sprintf("%d", device.ID),
						"sensor_name": psuName,
						"sensor_type": "psu_status",
						"status":      status,
					},
					Timestamp: time.Now(),
				}
				metrics = append(metrics, metric)
			}
		}
	}

	return metrics, nil
}

// parseSELLogs parses System Event Log entries
func parseSELLogs(output string) []string {
	logs := make([]string, 0)
	scanner := bufio.NewScanner(strings.NewReader(output))

	for scanner.Scan() {
		line := scanner.Text()
		if line != "" && !strings.HasPrefix(line, "SEL has no entries") {
			logs = append(logs, line)
		}
	}

	return logs
}
