package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/infrasense/backend/internal/models"
	"github.com/lib/pq"
)

type DeviceRepository struct {
	db *DB
}

func NewDeviceRepository(db *DB) *DeviceRepository {
	return &DeviceRepository{db: db}
}

// DB returns the underlying database handle.
func (r *DeviceRepository) DB() *DB {
	return r.db
}

// Create creates a new device
func (r *DeviceRepository) Create(ctx context.Context, req models.DeviceCreateRequest) (*models.Device, error) {
	device := &models.Device{
		ID:                   uuid.New(),
		Hostname:             req.Hostname,
		IPAddress:            req.IPAddress,
		BMCIPAddress:         req.BMCIPAddress,
		DeviceType:           req.DeviceType,
		Vendor:               req.Vendor,
		ManagementController: req.ManagementController,
		Protocol:             req.Protocol,
		PollingInterval:      req.PollingInterval,
		SSLVerify:            req.SSLVerify,
		Location:             req.Location,
		Tags:                 req.Tags,
		Status:               "unknown",
		ConnectionStatus:     "unknown",
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
	}

	query := `
		INSERT INTO devices (id, hostname, ip_address, bmc_ip_address, device_type,
		                     vendor, management_controller, protocol, polling_interval, ssl_verify,
		                     location, tags, status, connection_status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		RETURNING id, hostname, ip_address, bmc_ip_address, device_type,
		          vendor, management_controller, protocol, polling_interval, ssl_verify,
		          location, tags, status, last_seen, last_sync_at, connection_status, connection_error, created_at, updated_at
	`

	err := r.db.conn.QueryRowContext(
		ctx, query,
		device.ID, device.Hostname, device.IPAddress, device.BMCIPAddress, device.DeviceType,
		device.Vendor, device.ManagementController, device.Protocol, device.PollingInterval, device.SSLVerify,
		device.Location, pq.Array(device.Tags), device.Status,
		device.ConnectionStatus, device.CreatedAt, device.UpdatedAt,
	).Scan(
		&device.ID, &device.Hostname, &device.IPAddress, &device.BMCIPAddress, &device.DeviceType,
		&device.Vendor, &device.ManagementController, &device.Protocol, &device.PollingInterval, &device.SSLVerify,
		&device.Location, pq.Array(&device.Tags), &device.Status,
		&device.LastSeen, &device.LastSyncAt, &device.ConnectionStatus, &device.ConnectionError,
		&device.CreatedAt, &device.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create device: %w", err)
	}

	return device, nil
}

// GetByID retrieves a device by ID
func (r *DeviceRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Device, error) {
	device := &models.Device{}

	query := `
		SELECT id, hostname, ip_address, bmc_ip_address, device_type,
		       vendor, management_controller, protocol, polling_interval, ssl_verify,
		       location, tags, status,
		       last_seen, last_sync_at, connection_status, connection_error, created_at, updated_at
		FROM devices
		WHERE id = $1
	`

	err := r.db.conn.QueryRowContext(ctx, query, id).Scan(
		&device.ID, &device.Hostname, &device.IPAddress, &device.BMCIPAddress, &device.DeviceType,
		&device.Vendor, &device.ManagementController, &device.Protocol, &device.PollingInterval, &device.SSLVerify,
		&device.Location, pq.Array(&device.Tags), &device.Status,
		&device.LastSeen, &device.LastSyncAt, &device.ConnectionStatus, &device.ConnectionError,
		&device.CreatedAt, &device.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("device not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get device: %w", err)
	}

	return device, nil
}

// List retrieves devices with pagination and filtering
func (r *DeviceRepository) List(ctx context.Context, filter models.DeviceListFilter) ([]models.Device, int, error) {
	// Build WHERE clause
	whereClauses := []string{}
	args := []interface{}{}
	argPos := 1

	if filter.DeviceType != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("device_type = $%d", argPos))
		args = append(args, *filter.DeviceType)
		argPos++
	}

	if filter.Status != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("status = $%d", argPos))
		args = append(args, *filter.Status)
		argPos++
	}

	if filter.Location != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("location = $%d", argPos))
		args = append(args, *filter.Location)
		argPos++
	}

	if len(filter.Tags) > 0 {
		whereClauses = append(whereClauses, fmt.Sprintf("tags && $%d", argPos))
		args = append(args, pq.Array(filter.Tags))
		argPos++
	}

	if filter.Search != nil && *filter.Search != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("(hostname ILIKE $%d OR ip_address::text ILIKE $%d)", argPos, argPos))
		args = append(args, "%"+*filter.Search+"%")
		argPos++
	}

	whereClause := ""
	if len(whereClauses) > 0 {
		whereClause = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	// Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM devices %s", whereClause)
	var total int
	err := r.db.conn.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count devices: %w", err)
	}

	// Get paginated results
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 {
		filter.PageSize = 20
	}
	if filter.PageSize > 100 {
		filter.PageSize = 100
	}

	offset := (filter.Page - 1) * filter.PageSize

	query := fmt.Sprintf(`
		SELECT id, hostname, ip_address, bmc_ip_address, device_type,
		       vendor, management_controller, protocol, polling_interval, ssl_verify,
		       location, tags, status,
		       last_seen, last_sync_at, connection_status, connection_error, created_at, updated_at
		FROM devices
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argPos, argPos+1)

	args = append(args, filter.PageSize, offset)

	rows, err := r.db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list devices: %w", err)
	}
	defer rows.Close()

	devices := []models.Device{}
	for rows.Next() {
		device := models.Device{}
		err := rows.Scan(
			&device.ID, &device.Hostname, &device.IPAddress, &device.BMCIPAddress, &device.DeviceType,
			&device.Vendor, &device.ManagementController, &device.Protocol, &device.PollingInterval, &device.SSLVerify,
			&device.Location, pq.Array(&device.Tags), &device.Status,
			&device.LastSeen, &device.LastSyncAt, &device.ConnectionStatus, &device.ConnectionError,
			&device.CreatedAt, &device.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan device: %w", err)
		}
		devices = append(devices, device)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating devices: %w", err)
	}

	return devices, total, nil
}

// Update updates a device
func (r *DeviceRepository) Update(ctx context.Context, id uuid.UUID, req models.DeviceUpdateRequest) (*models.Device, error) {
	// Build SET clause dynamically
	setClauses := []string{"updated_at = $1"}
	args := []interface{}{time.Now()}
	argPos := 2

	if req.Hostname != nil {
		setClauses = append(setClauses, fmt.Sprintf("hostname = $%d", argPos))
		args = append(args, *req.Hostname)
		argPos++
	}

	if req.IPAddress != nil {
		setClauses = append(setClauses, fmt.Sprintf("ip_address = $%d", argPos))
		args = append(args, *req.IPAddress)
		argPos++
	}

	if req.BMCIPAddress != nil {
		setClauses = append(setClauses, fmt.Sprintf("bmc_ip_address = $%d", argPos))
		args = append(args, *req.BMCIPAddress)
		argPos++
	}

	if req.DeviceType != nil {
		setClauses = append(setClauses, fmt.Sprintf("device_type = $%d", argPos))
		args = append(args, *req.DeviceType)
		argPos++
	}

	if req.Vendor != nil {
		setClauses = append(setClauses, fmt.Sprintf("vendor = $%d", argPos))
		args = append(args, *req.Vendor)
		argPos++
	}

	if req.ManagementController != nil {
		setClauses = append(setClauses, fmt.Sprintf("management_controller = $%d", argPos))
		args = append(args, *req.ManagementController)
		argPos++
	}

	if req.Protocol != nil {
		setClauses = append(setClauses, fmt.Sprintf("protocol = $%d", argPos))
		args = append(args, *req.Protocol)
		argPos++
	}

	if req.PollingInterval != nil {
		setClauses = append(setClauses, fmt.Sprintf("polling_interval = $%d", argPos))
		args = append(args, *req.PollingInterval)
		argPos++
	}

	if req.SSLVerify != nil {
		setClauses = append(setClauses, fmt.Sprintf("ssl_verify = $%d", argPos))
		args = append(args, *req.SSLVerify)
		argPos++
	}

	if req.Location != nil {
		setClauses = append(setClauses, fmt.Sprintf("location = $%d", argPos))
		args = append(args, *req.Location)
		argPos++
	}

	if req.Tags != nil {
		setClauses = append(setClauses, fmt.Sprintf("tags = $%d", argPos))
		args = append(args, pq.Array(req.Tags))
		argPos++
	}

	if req.Status != nil {
		setClauses = append(setClauses, fmt.Sprintf("status = $%d", argPos))
		args = append(args, *req.Status)
		argPos++
	}

	args = append(args, id)

	query := fmt.Sprintf(`
		UPDATE devices
		SET %s
		WHERE id = $%d
		RETURNING id, hostname, ip_address, bmc_ip_address, device_type,
		          vendor, management_controller, protocol, polling_interval, ssl_verify,
		          location, tags, status,
		          last_seen, last_sync_at, connection_status, connection_error, created_at, updated_at
	`, strings.Join(setClauses, ", "), argPos)

	device := &models.Device{}
	err := r.db.conn.QueryRowContext(ctx, query, args...).Scan(
		&device.ID, &device.Hostname, &device.IPAddress, &device.BMCIPAddress, &device.DeviceType,
		&device.Vendor, &device.ManagementController, &device.Protocol, &device.PollingInterval, &device.SSLVerify,
		&device.Location, pq.Array(&device.Tags), &device.Status,
		&device.LastSeen, &device.LastSyncAt, &device.ConnectionStatus, &device.ConnectionError,
		&device.CreatedAt, &device.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("device not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to update device: %w", err)
	}

	return device, nil
}

// Delete deletes a device (cascade deletes credentials, alert rules, maintenance windows)
func (r *DeviceRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := "DELETE FROM devices WHERE id = $1"

	result, err := r.db.conn.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete device: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("device not found")
	}

	return nil
}

// MarkTimedOutAgentsUnavailable marks devices as unavailable if their last_seen timestamp
// is older than the specified timeout duration. Only applies to linux_agent and windows_agent device types.
// Returns the number of devices marked as unavailable.
func (r *DeviceRepository) MarkTimedOutAgentsUnavailable(ctx context.Context, timeout time.Duration) (int, error) {
	query := `
		UPDATE devices
		SET status = 'unavailable', updated_at = NOW()
		WHERE device_type IN ('linux_agent', 'windows_agent')
		  AND status != 'unavailable'
		  AND (last_seen IS NULL OR last_seen < NOW() - $1::interval)
		RETURNING id
	`

	rows, err := r.db.conn.QueryContext(ctx, query, timeout.String())
	if err != nil {
		return 0, fmt.Errorf("failed to mark timed-out agents as unavailable: %w", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return count, fmt.Errorf("failed to scan device id: %w", err)
		}
		count++
	}

	if err := rows.Err(); err != nil {
		return count, fmt.Errorf("error iterating timed-out devices: %w", err)
	}

	return count, nil
}

// UpdateConnectionStatus updates the connection_status and connection_error fields.
func (r *DeviceRepository) UpdateConnectionStatus(ctx context.Context, id uuid.UUID, status string, errMsg string) error {
	var errPtr *string
	if errMsg != "" {
		errPtr = &errMsg
	}
	query := `UPDATE devices SET connection_status = $1, connection_error = $2, updated_at = NOW() WHERE id = $3`
	_, err := r.db.conn.ExecContext(ctx, query, status, errPtr, id)
	if err != nil {
		return fmt.Errorf("failed to update connection status: %w", err)
	}
	return nil
}

// UpdateSyncResult updates last_sync_at, connection_status, device status, and optionally connection_error after a sync.
// When connectionStatus is "connected" the device status is set to "healthy"; on failure it is set to "critical".
func (r *DeviceRepository) UpdateSyncResult(ctx context.Context, id uuid.UUID, connectionStatus string, errMsg string) error {
	var errPtr *string
	if errMsg != "" {
		errPtr = &errMsg
	}
	deviceStatus := "healthy"
	if connectionStatus != "connected" {
		deviceStatus = "critical"
	}
	query := `
		UPDATE devices
		SET last_sync_at = NOW(), connection_status = $1, connection_error = $2,
		    status = $3, updated_at = NOW()
		WHERE id = $4
	`
	_, err := r.db.conn.ExecContext(ctx, query, connectionStatus, errPtr, deviceStatus, id)
	if err != nil {
		return fmt.Errorf("failed to update sync result: %w", err)
	}
	return nil
}

// GetInventory retrieves hardware inventory for a device including all JSONB telemetry.
func (r *DeviceRepository) GetInventory(ctx context.Context, id uuid.UUID) (*models.DeviceInventory, error) {
	inv := &models.DeviceInventory{DeviceID: id}

	var (
		procsJSON, memJSON, storCtrlJSON, drivesJSON []byte
		tempsJSON, fansJSON, psuJSON, nicsJSON       []byte
		selJSON, accelJSON, pcieJSON, voltJSON       []byte
		lcLogsJSON, vdisksJSON, enclosuresJSON       []byte
		legacyRAMTotalGB                             *float64 // ram_total_gb (INT) from migration 003
		bmcNameStr, sysUUIDStr, sysRevStr            string
	)

	err := r.db.conn.QueryRowContext(ctx, `
		SELECT cpu_model, cpu_cores, cpu_threads, ram_total_gb,
		       firmware_bmc, firmware_bios, firmware_raid, collected_at,
		       manufacturer, system_model, service_tag, asset_tag,
		       power_state, health_status, bmc_mac_address, bmc_dns_name,
		       bmc_license, bmc_hardware_version, lifecycle_controller_ver,
		       total_power_watts, memory_total_gb,
		       COALESCE(bmc_name, '') as bmc_name,
		       COALESCE(system_uuid, '') as system_uuid,
		       COALESCE(system_revision, '') as system_revision,
		       system_uptime_seconds, boot_mode,
		       os_name, os_version, os_kernel, os_hostname, os_uptime_seconds,
		       processors_json, memory_modules_json, storage_controllers_json,
		       drives_json, temperatures_json, fans_json, power_supplies_json,
		       nics_json, sel_entries_json, accelerators_json, pcie_slots_json, voltages_json,
		       lifecycle_logs_json, virtual_disks_json, storage_enclosures_json
		FROM device_inventory WHERE device_id = $1
	`, id).Scan(
		&inv.CPUModel, &inv.CPUCores, &inv.CPUThreads, &legacyRAMTotalGB,
		&inv.FirmwareBMC, &inv.FirmwareBIOS, &inv.FirmwareRAID, &inv.CollectedAt,
		&inv.Manufacturer, &inv.SystemModel, &inv.ServiceTag, &inv.AssetTag,
		&inv.PowerState, &inv.HealthStatus, &inv.BMCMACAddress, &inv.BMCDNSName,
		&inv.BMCLicense, &inv.BMCHardwareVersion, &inv.LifecycleControllerVer,
		&inv.TotalPowerWatts, &inv.RAMTotalGB,
		&bmcNameStr, &sysUUIDStr, &sysRevStr,
		&inv.SystemUptimeSeconds, &inv.BootMode,
		&inv.OSName, &inv.OSVersion, &inv.OSKernel, &inv.OSHostname, &inv.OSUptimeSeconds,
		&procsJSON, &memJSON, &storCtrlJSON,
		&drivesJSON, &tempsJSON, &fansJSON, &psuJSON,
		&nicsJSON, &selJSON, &accelJSON, &pcieJSON, &voltJSON,
		&lcLogsJSON, &vdisksJSON, &enclosuresJSON,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("inventory not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get inventory: %w", err)
	}
	// Prefer memory_total_gb (NUMERIC, from migration 011); fall back to legacy ram_total_gb (INT)
	if inv.RAMTotalGB == nil && legacyRAMTotalGB != nil {
		inv.RAMTotalGB = legacyRAMTotalGB
	}
	if bmcNameStr != "" {
		inv.BMCName = &bmcNameStr
	}
	if sysUUIDStr != "" {
		inv.SystemUUID = &sysUUIDStr
	}
	if sysRevStr != "" {
		inv.SystemRevision = &sysRevStr
	}

	// Unmarshal JSONB fields
	unmarshalJSON := func(data []byte, dest any) {
		if len(data) > 0 {
			_ = json.Unmarshal(data, dest)
		}
	}
	unmarshalJSON(procsJSON, &inv.Processors)
	unmarshalJSON(memJSON, &inv.MemoryModules)
	unmarshalJSON(storCtrlJSON, &inv.StorageControllers)
	unmarshalJSON(drivesJSON, &inv.Drives)
	unmarshalJSON(tempsJSON, &inv.Temperatures)
	unmarshalJSON(fansJSON, &inv.Fans)
	unmarshalJSON(psuJSON, &inv.PowerSupplies)
	unmarshalJSON(nicsJSON, &inv.NICs)
	unmarshalJSON(selJSON, &inv.SELEntries)
	unmarshalJSON(accelJSON, &inv.Accelerators)
	unmarshalJSON(pcieJSON, &inv.PCIeSlots)
	unmarshalJSON(voltJSON, &inv.Voltages)
	unmarshalJSON(lcLogsJSON, &inv.LifecycleLogs)
	unmarshalJSON(vdisksJSON, &inv.VirtualDisks)
	unmarshalJSON(enclosuresJSON, &inv.StorageEnclosures)

	// Legacy NIC rows
	nicRows, err := r.db.conn.QueryContext(ctx,
		`SELECT nic_model, mac_address::text, collected_at FROM device_nics WHERE device_id = $1`, id)
	if err == nil {
		defer nicRows.Close()
		for nicRows.Next() {
			var n models.NICInfo
			if err := nicRows.Scan(&n.Model, &n.MACAddress, &n.CollectedAt); err == nil {
				inv.LegacyNICs = append(inv.LegacyNICs, n)
			}
		}
	}

	// Legacy disk rows
	diskRows, err := r.db.conn.QueryContext(ctx,
		`SELECT disk_model, capacity_gb, serial_number, collected_at FROM device_disks WHERE device_id = $1`, id)
	if err == nil {
		defer diskRows.Close()
		for diskRows.Next() {
			var d models.DiskInfo
			if err := diskRows.Scan(&d.Model, &d.CapacityGB, &d.SerialNumber, &d.CollectedAt); err == nil {
				inv.LegacyDisks = append(inv.LegacyDisks, d)
			}
		}
	}

	return inv, nil
}

// UpsertInventory persists hardware inventory collected during a Redfish sync.
func (r *DeviceRepository) UpsertInventory(ctx context.Context, deviceID uuid.UUID, result *models.DeviceSyncResult) error {
	now := time.Now()

	var cpuModel *string
	var cpuCores, cpuThreads *int
	if len(result.Processors) > 0 {
		p := result.Processors[0]
		if p.Model != "" {
			cpuModel = &p.Model
		}
		if p.Cores > 0 {
			cpuCores = &p.Cores
		}
		if p.Threads > 0 {
			cpuThreads = &p.Threads
		}
	}

	// OS info fields
	var osName, osVersion, osKernel, osHostname *string
	var osUptime *int64
	if result.OS != nil {
		if result.OS.Name != "" {
			osName = &result.OS.Name
		}
		if result.OS.Version != "" {
			osVersion = &result.OS.Version
		}
		if result.OS.Kernel != "" {
			osKernel = &result.OS.Kernel
		}
		if result.OS.Hostname != "" {
			osHostname = &result.OS.Hostname
		}
		if result.OS.UptimeSeconds > 0 {
			osUptime = &result.OS.UptimeSeconds
		}
	}

	// Marshal JSONB fields
	toJSON := func(v any) []byte {
		b, _ := json.Marshal(v)
		return b
	}

	_, err := r.db.conn.ExecContext(ctx, `
		INSERT INTO device_inventory (
			id, device_id, cpu_model, cpu_cores, cpu_threads, ram_total_gb,
			firmware_bmc, firmware_bios, collected_at,
			manufacturer, system_model, service_tag, asset_tag,
			power_state, health_status, bmc_mac_address, bmc_dns_name,
			bmc_license, bmc_hardware_version, lifecycle_controller_ver,
			total_power_watts, memory_total_gb,
			bmc_name, system_uuid, system_revision, system_uptime_seconds, boot_mode,
			os_name, os_version, os_kernel, os_hostname, os_uptime_seconds,
			processors_json, memory_modules_json, storage_controllers_json,
			drives_json, temperatures_json, fans_json, power_supplies_json,
			nics_json, sel_entries_json, accelerators_json, pcie_slots_json, voltages_json,
			lifecycle_logs_json, virtual_disks_json, storage_enclosures_json
		) VALUES (
			gen_random_uuid(), $1, $2, $3, $4, $5,
			$6, $7, $8,
			$9, $10, $11, $12,
			$13, $14, $15, $16,
			$17, $18, $19,
			$20, $21,
			$22, $23, $24, $25, $26,
			$27, $28, $29, $30, $31,
			$32, $33, $34,
			$35, $36, $37, $38,
			$39, $40, $41, $42, $43,
			$44, $45, $46
		)
		ON CONFLICT (device_id) DO UPDATE SET
			cpu_model = EXCLUDED.cpu_model,
			cpu_cores = EXCLUDED.cpu_cores,
			cpu_threads = EXCLUDED.cpu_threads,
			ram_total_gb = EXCLUDED.ram_total_gb,
			firmware_bmc = EXCLUDED.firmware_bmc,
			firmware_bios = EXCLUDED.firmware_bios,
			collected_at = EXCLUDED.collected_at,
			manufacturer = EXCLUDED.manufacturer,
			system_model = EXCLUDED.system_model,
			service_tag = EXCLUDED.service_tag,
			asset_tag = EXCLUDED.asset_tag,
			power_state = EXCLUDED.power_state,
			health_status = EXCLUDED.health_status,
			bmc_mac_address = EXCLUDED.bmc_mac_address,
			bmc_dns_name = EXCLUDED.bmc_dns_name,
			bmc_license = EXCLUDED.bmc_license,
			bmc_hardware_version = EXCLUDED.bmc_hardware_version,
			lifecycle_controller_ver = EXCLUDED.lifecycle_controller_ver,
			total_power_watts = EXCLUDED.total_power_watts,
			memory_total_gb = EXCLUDED.memory_total_gb,
			bmc_name = EXCLUDED.bmc_name,
			system_uuid = EXCLUDED.system_uuid,
			system_revision = EXCLUDED.system_revision,
			system_uptime_seconds = EXCLUDED.system_uptime_seconds,
			boot_mode = EXCLUDED.boot_mode,
			os_name = EXCLUDED.os_name,
			os_version = EXCLUDED.os_version,
			os_kernel = EXCLUDED.os_kernel,
			os_hostname = EXCLUDED.os_hostname,
			os_uptime_seconds = EXCLUDED.os_uptime_seconds,
			processors_json = EXCLUDED.processors_json,
			memory_modules_json = EXCLUDED.memory_modules_json,
			storage_controllers_json = EXCLUDED.storage_controllers_json,
			drives_json = EXCLUDED.drives_json,
			temperatures_json = EXCLUDED.temperatures_json,
			fans_json = EXCLUDED.fans_json,
			power_supplies_json = EXCLUDED.power_supplies_json,
			nics_json = EXCLUDED.nics_json,
			sel_entries_json = EXCLUDED.sel_entries_json,
			accelerators_json = EXCLUDED.accelerators_json,
			pcie_slots_json = EXCLUDED.pcie_slots_json,
			voltages_json = EXCLUDED.voltages_json,
			lifecycle_logs_json = EXCLUDED.lifecycle_logs_json,
			virtual_disks_json = EXCLUDED.virtual_disks_json,
			storage_enclosures_json = EXCLUDED.storage_enclosures_json
	`,
		deviceID, cpuModel, cpuCores, cpuThreads, result.MemoryTotalGB,
		result.BMCFirmware, result.BIOSVersion, now,
		result.Manufacturer, result.Model, result.ServiceTag, result.AssetTag,
		result.PowerState, result.HealthStatus, result.BMCMACAddress, result.BMCDNSName,
		result.BMCLicense, result.BMCHardwareVersion, result.LifecycleControllerVersion,
		result.TotalPowerWatts, result.MemoryTotalGB,
		result.BMCName, result.SystemUUID, result.SystemRevision, result.SystemUptimeSeconds, result.BootMode,
		osName, osVersion, osKernel, osHostname, osUptime,
		toJSON(result.Processors), toJSON(result.MemoryModules), toJSON(result.StorageControllers),
		toJSON(result.Drives), toJSON(result.Temperatures), toJSON(result.Fans), toJSON(result.PowerSupplies),
		toJSON(result.NICs), toJSON(result.SELEntries), toJSON(result.Accelerators), toJSON(result.PCIeSlots), toJSON(result.Voltages),
		toJSON(result.LifecycleLogs), toJSON(result.VirtualDisks), toJSON(result.StorageEnclosures),
	)
	if err != nil {
		return fmt.Errorf("upsert inventory: %w", err)
	}

	// Replace legacy NIC rows
	if _, err := r.db.conn.ExecContext(ctx, `DELETE FROM device_nics WHERE device_id = $1`, deviceID); err != nil {
		return fmt.Errorf("delete nics: %w", err)
	}
	for _, nic := range result.NICs {
		if nic.MACAddress == "" {
			continue
		}
		model := nic.Model
		_, _ = r.db.conn.ExecContext(ctx,
			`INSERT INTO device_nics (id, device_id, nic_model, mac_address, collected_at) VALUES (gen_random_uuid(), $1, $2, $3::macaddr, $4)`,
			deviceID, model, nic.MACAddress, now,
		)
	}

	// Replace legacy disk rows
	if _, err := r.db.conn.ExecContext(ctx, `DELETE FROM device_disks WHERE device_id = $1`, deviceID); err != nil {
		return fmt.Errorf("delete disks: %w", err)
	}
	for _, d := range result.Drives {
		capGB := int(d.CapacityGB)
		var serial, model *string
		if d.SerialNumber != "" {
			serial = &d.SerialNumber
		}
		if d.Model != "" {
			model = &d.Model
		}
		_, _ = r.db.conn.ExecContext(ctx,
			`INSERT INTO device_disks (id, device_id, disk_model, capacity_gb, serial_number, collected_at) VALUES (gen_random_uuid(), $1, $2, $3, $4, $5)`,
			deviceID, model, capGB, serial, now,
		)
	}

	return nil
}
