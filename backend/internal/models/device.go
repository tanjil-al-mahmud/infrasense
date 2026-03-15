package models

import (
	"time"

	"github.com/google/uuid"
)

// ── Device core model ─────────────────────────────────────────────────────────

type Device struct {
	ID                   uuid.UUID  `json:"id" db:"id"`
	Hostname             string     `json:"hostname" db:"hostname"`
	IPAddress            string     `json:"ip_address" db:"ip_address"`
	BMCIPAddress         *string    `json:"bmc_ip_address,omitempty" db:"bmc_ip_address"`
	DeviceType           string     `json:"device_type" db:"device_type"`
	Vendor               *string    `json:"vendor,omitempty" db:"vendor"`
	ManagementController *string    `json:"management_controller,omitempty" db:"management_controller"`
	Protocol             *string    `json:"protocol,omitempty" db:"protocol"`
	PollingInterval      *int       `json:"polling_interval,omitempty" db:"polling_interval"`
	SSLVerify            *bool      `json:"ssl_verify,omitempty" db:"ssl_verify"`
	Location             *string    `json:"location,omitempty" db:"location"`
	Tags                 []string   `json:"tags,omitempty" db:"tags"`
	Status               string     `json:"status" db:"status"`
	LastSeen             *time.Time `json:"last_seen,omitempty" db:"last_seen"`
	LastSyncAt           *time.Time `json:"last_sync_at,omitempty" db:"last_sync_at"`
	ConnectionStatus     string     `json:"connection_status" db:"connection_status"`
	ConnectionError      *string    `json:"connection_error,omitempty" db:"connection_error"`
	CreatedAt            time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at" db:"updated_at"`
}

type DeviceCreateRequest struct {
	Hostname             string   `json:"hostname" binding:"required"`
	IPAddress            string   `json:"ip_address" binding:"required"`
	BMCIPAddress         *string  `json:"bmc_ip_address,omitempty"`
	DeviceType           string   `json:"device_type" binding:"required"`
	Vendor               *string  `json:"vendor,omitempty"`
	ManagementController *string  `json:"management_controller,omitempty"`
	Protocol             *string  `json:"protocol,omitempty"`
	PollingInterval      *int     `json:"polling_interval,omitempty"`
	SSLVerify            *bool    `json:"ssl_verify,omitempty"`
	Location             *string  `json:"location,omitempty"`
	Tags                 []string `json:"tags,omitempty"`
}

type DeviceUpdateRequest struct {
	Hostname             *string  `json:"hostname,omitempty"`
	IPAddress            *string  `json:"ip_address,omitempty"`
	BMCIPAddress         *string  `json:"bmc_ip_address,omitempty"`
	DeviceType           *string  `json:"device_type,omitempty"`
	Vendor               *string  `json:"vendor,omitempty"`
	ManagementController *string  `json:"management_controller,omitempty"`
	Protocol             *string  `json:"protocol,omitempty"`
	PollingInterval      *int     `json:"polling_interval,omitempty"`
	SSLVerify            *bool    `json:"ssl_verify,omitempty"`
	Location             *string  `json:"location,omitempty"`
	Tags                 []string `json:"tags,omitempty"`
	Status               *string  `json:"status,omitempty"`
}

type DeviceListFilter struct {
	DeviceType *string
	Status     *string
	Location   *string
	Tags       []string
	Search     *string
	Page       int
	PageSize   int
}

type DeviceListResponse struct {
	Data []Device       `json:"data"`
	Meta PaginationMeta `json:"meta"`
}

type PaginationMeta struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
	Total    int `json:"total"`
}

// ── Power control ─────────────────────────────────────────────────────────────

// PowerControlRequest is the body for POST /devices/:id/power
type PowerControlRequest struct {
	// ResetType: On | ForceOff | GracefulShutdown | ForceRestart | PowerCycle | GracefulRestart
	ResetType string `json:"reset_type" binding:"required"`
}

// PowerControlResult is returned by the power control endpoint.
type PowerControlResult struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	ResetType string `json:"reset_type"`
}

// BootControlRequest is the body for POST /devices/:id/boot
type BootControlRequest struct {
	// Target: Pxe | Cd | Hdd | BiosSetup | None
	Target string `json:"target" binding:"required"`
	Once   bool   `json:"once"`
}

// BootControlResult is returned by the boot control endpoint.
type BootControlResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Target  string `json:"target"`
}

// ── Hardware telemetry types ──────────────────────────────────────────────────

type ProcessorInfo struct {
	Name               string   `json:"name"`
	Model              string   `json:"model,omitempty"`
	Manufacturer       string   `json:"manufacturer,omitempty"`
	Socket             string   `json:"socket,omitempty"`
	Cores              int      `json:"cores,omitempty"`
	Threads            int      `json:"threads,omitempty"`
	SpeedMHz           int      `json:"speed_mhz,omitempty"`
	MaxSpeedMHz        int      `json:"max_speed_mhz,omitempty"`
	CacheSizeMiB       int      `json:"cache_size_mib,omitempty"`
	TemperatureCelsius *float64 `json:"temperature_celsius,omitempty"`
	Health             string   `json:"health,omitempty"`
}

type MemoryInfo struct {
	Name         string  `json:"name"`
	CapacityGB   float64 `json:"capacity_gb,omitempty"`
	Manufacturer string  `json:"manufacturer,omitempty"`
	PartNumber   string  `json:"part_number,omitempty"`
	SerialNumber string  `json:"serial_number,omitempty"`
	Location     string  `json:"location,omitempty"`
	SpeedMHz     int     `json:"speed_mhz,omitempty"`
	MemoryType   string  `json:"memory_type,omitempty"`
	ECCEnabled   *bool   `json:"ecc_enabled,omitempty"`
	Health       string  `json:"health,omitempty"`
}

type StorageControllerInfo struct {
	Name          string `json:"name"`
	Model         string `json:"model,omitempty"`
	Manufacturer  string `json:"manufacturer,omitempty"`
	Health        string `json:"health,omitempty"`
	FirmwareVer   string `json:"firmware_version,omitempty"`
	BatteryHealth string `json:"battery_health,omitempty"`
}

type DriveInfo struct {
	Name               string   `json:"name"`
	Model              string   `json:"model,omitempty"`
	Manufacturer       string   `json:"manufacturer,omitempty"`
	SerialNumber       string   `json:"serial_number,omitempty"`
	CapacityGB         float64  `json:"capacity_gb,omitempty"`
	MediaType          string   `json:"media_type,omitempty"`
	BusProtocol        string   `json:"bus_protocol,omitempty"`
	TemperatureCelsius *float64 `json:"temperature_celsius,omitempty"`
	ReadPolicy         string   `json:"read_policy,omitempty"`
	WritePolicy        string   `json:"write_policy,omitempty"`
	Health             string   `json:"health,omitempty"`
	Status             string   `json:"status,omitempty"`
}

type VirtualDiskInfo struct {
	Name        string  `json:"name"`
	RAIDLevel   string  `json:"raid_level,omitempty"`
	CapacityGB  float64 `json:"capacity_gb,omitempty"`
	ReadPolicy  string  `json:"read_policy,omitempty"`
	WritePolicy string  `json:"write_policy,omitempty"`
	CachePolicy string  `json:"cache_policy,omitempty"`
	Health      string  `json:"health,omitempty"`
	Status      string  `json:"status,omitempty"`
}

type TemperatureReading struct {
	Name            string   `json:"name"`
	ReadingCelsius  *float64 `json:"reading_celsius"`
	UpperThreshWarn *float64 `json:"upper_threshold_warn,omitempty"`
	UpperThreshCrit *float64 `json:"upper_threshold_crit,omitempty"`
	Health          string   `json:"health,omitempty"`
}

type FanReading struct {
	Name        string   `json:"name"`
	FanID       string   `json:"fan_id,omitempty"`
	ReadingRPM  *float64 `json:"reading_rpm"`
	LowerThresh *float64 `json:"lower_threshold_warn,omitempty"`
	Health      string   `json:"health,omitempty"`
}

type PowerSupplyInfo struct {
	Name             string   `json:"name"`
	PowerInputWatts  *float64 `json:"power_input_watts,omitempty"`
	PowerOutputWatts *float64 `json:"power_output_watts,omitempty"`
	LastPowerOutputW *float64 `json:"last_power_output_watts,omitempty"`
	PowerCapWatts    *float64 `json:"power_cap_watts,omitempty"`
	Health           string   `json:"health,omitempty"`
	Status           string   `json:"status,omitempty"`
	Redundancy       string   `json:"redundancy,omitempty"`
}

type VoltageReading struct {
	Name            string   `json:"name"`
	ReadingVolts    *float64 `json:"reading_volts"`
	UpperThreshWarn *float64 `json:"upper_threshold_warn,omitempty"`
	LowerThreshWarn *float64 `json:"lower_threshold_warn,omitempty"`
	Health          string   `json:"health,omitempty"`
}

// FullNICInfo includes port-level detail and firmware version.
type FullNICInfo struct {
	Name            string `json:"name"`
	Model           string `json:"model,omitempty"`
	FirmwareVersion string `json:"firmware_version,omitempty"`
	MACAddress      string `json:"mac_address"`
	IPv4Address     string `json:"ipv4_address,omitempty"`
	IPv6Address     string `json:"ipv6_address,omitempty"`
	LinkStatus      string `json:"link_status,omitempty"`
	SpeedMbps       int    `json:"speed_mbps,omitempty"`
	Health          string `json:"health,omitempty"`
}

type AcceleratorInfo struct {
	Name         string `json:"name"`
	Model        string `json:"model,omitempty"`
	Manufacturer string `json:"manufacturer,omitempty"`
	DeviceClass  string `json:"device_class,omitempty"`
	Health       string `json:"health,omitempty"`
}

type PCIeSlotInfo struct {
	Name       string `json:"name"`
	SlotType   string `json:"slot_type,omitempty"`
	PCIeType   string `json:"pcie_type,omitempty"`
	Status     string `json:"status,omitempty"`
	DeviceName string `json:"device_name,omitempty"`
}

type SELEntry struct {
	ID       string `json:"id"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
	Created  string `json:"created,omitempty"`
}

type LifecycleLogEntry struct {
	ID       string `json:"id"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
	Category string `json:"category,omitempty"`
	Created  string `json:"created,omitempty"`
}

type SyncStep struct {
	Name    string `json:"name"`
	Status  string `json:"status"` // "ok", "error", "skipped"
	Message string `json:"message,omitempty"`
}

// ConnectionTestResult is returned by the test-connection endpoint.
type ConnectionTestResult struct {
	Success        bool   `json:"success"`
	Message        string `json:"message"`
	RedfishVersion string `json:"redfish_version,omitempty"`
	ErrorCode      string `json:"error_code,omitempty"`
}

// StorageEnclosureInfo represents a storage enclosure/backplane.
type StorageEnclosureInfo struct {
	Name        string `json:"name"`
	BackplaneID string `json:"backplane_id,omitempty"`
	Controller  string `json:"controller,omitempty"`
	Health      string `json:"health,omitempty"`
}

// OSInfo holds operating system details collected during sync.
type OSInfo struct {
	Name          string `json:"name,omitempty"`
	Version       string `json:"version,omitempty"`
	Kernel        string `json:"kernel,omitempty"`
	Hostname      string `json:"hostname,omitempty"`
	UptimeSeconds int64  `json:"uptime_seconds,omitempty"`
}

// NICInfo is the legacy per-row NIC record (device_nics table).
type NICInfo struct {
	Model       string     `json:"model,omitempty"`
	MACAddress  string     `json:"mac_address"`
	CollectedAt *time.Time `json:"collected_at,omitempty"`
}

// DiskInfo is the legacy per-row disk record (device_disks table).
type DiskInfo struct {
	Model        string     `json:"model,omitempty"`
	CapacityGB   int        `json:"capacity_gb,omitempty"`
	SerialNumber string     `json:"serial_number,omitempty"`
	CollectedAt  *time.Time `json:"collected_at,omitempty"`
}

// DeviceSyncResult holds all data collected during a Redfish sync.
type DeviceSyncResult struct {
	Success                    bool                    `json:"success"`
	Message                    string                  `json:"message,omitempty"`
	Steps                      []SyncStep              `json:"steps,omitempty"`
	Manufacturer               *string                 `json:"manufacturer,omitempty"`
	Model                      *string                 `json:"model,omitempty"`
	SerialNumber               *string                 `json:"serial_number,omitempty"`
	ServiceTag                 *string                 `json:"service_tag,omitempty"`
	AssetTag                   *string                 `json:"asset_tag,omitempty"`
	PowerState                 *string                 `json:"power_state,omitempty"`
	HealthStatus               *string                 `json:"health_status,omitempty"`
	BIOSVersion                *string                 `json:"bios_version,omitempty"`
	SystemUUID                 *string                 `json:"system_uuid,omitempty"`
	SystemRevision             *string                 `json:"system_revision,omitempty"`
	SystemUptimeSeconds        *int64                  `json:"system_uptime_seconds,omitempty"`
	BootMode                   *string                 `json:"boot_mode,omitempty"`
	LifecycleControllerVersion *string                 `json:"lifecycle_controller_version,omitempty"`
	IntrusionStatus            *string                 `json:"intrusion_status,omitempty"`
	BMCFirmware                *string                 `json:"bmc_firmware,omitempty"`
	BMCName                    *string                 `json:"bmc_name,omitempty"`
	BMCHardwareVersion         *string                 `json:"bmc_hardware_version,omitempty"`
	BMCMACAddress              *string                 `json:"bmc_mac_address,omitempty"`
	BMCDNSName                 *string                 `json:"bmc_dns_name,omitempty"`
	BMCLicense                 *string                 `json:"bmc_license,omitempty"`
	TotalPowerWatts            *float64                `json:"total_power_watts,omitempty"`
	MemoryTotalGB              *float64                `json:"memory_total_gb,omitempty"`
	OS                         *OSInfo                 `json:"os,omitempty"`
	Processors                 []ProcessorInfo         `json:"processors,omitempty"`
	MemoryModules              []MemoryInfo            `json:"memory_modules,omitempty"`
	StorageControllers         []StorageControllerInfo `json:"storage_controllers,omitempty"`
	Drives                     []DriveInfo             `json:"drives,omitempty"`
	VirtualDisks               []VirtualDiskInfo       `json:"virtual_disks,omitempty"`
	StorageEnclosures          []StorageEnclosureInfo  `json:"storage_enclosures,omitempty"`
	Temperatures               []TemperatureReading    `json:"temperatures,omitempty"`
	Fans                       []FanReading            `json:"fans,omitempty"`
	PowerSupplies              []PowerSupplyInfo       `json:"power_supplies,omitempty"`
	Voltages                   []VoltageReading        `json:"voltages,omitempty"`
	NICs                       []FullNICInfo           `json:"nics,omitempty"`
	Accelerators               []AcceleratorInfo       `json:"accelerators,omitempty"`
	PCIeSlots                  []PCIeSlotInfo          `json:"pcie_slots,omitempty"`
	SELEntries                 []SELEntry              `json:"sel_entries,omitempty"`
	LifecycleLogs              []LifecycleLogEntry     `json:"lifecycle_logs,omitempty"`
}

// DeviceInventory holds the full hardware inventory for a device as returned by GetInventory.
type DeviceInventory struct {
	DeviceID               uuid.UUID               `json:"device_id"`
	CPUModel               *string                 `json:"cpu_model,omitempty"`
	CPUCores               *int                    `json:"cpu_cores,omitempty"`
	CPUThreads             *int                    `json:"cpu_threads,omitempty"`
	RAMTotalGB             *float64                `json:"ram_total_gb,omitempty"`
	FirmwareBMC            *string                 `json:"firmware_bmc,omitempty"`
	FirmwareBIOS           *string                 `json:"firmware_bios,omitempty"`
	FirmwareRAID           *string                 `json:"firmware_raid,omitempty"`
	CollectedAt            *time.Time              `json:"collected_at,omitempty"`
	Manufacturer           *string                 `json:"manufacturer,omitempty"`
	SystemModel            *string                 `json:"system_model,omitempty"`
	ServiceTag             *string                 `json:"service_tag,omitempty"`
	AssetTag               *string                 `json:"asset_tag,omitempty"`
	PowerState             *string                 `json:"power_state,omitempty"`
	HealthStatus           *string                 `json:"health_status,omitempty"`
	BMCMACAddress          *string                 `json:"bmc_mac_address,omitempty"`
	BMCDNSName             *string                 `json:"bmc_dns_name,omitempty"`
	BMCLicense             *string                 `json:"bmc_license,omitempty"`
	BMCHardwareVersion     *string                 `json:"bmc_hardware_version,omitempty"`
	BMCName                *string                 `json:"bmc_name,omitempty"`
	LifecycleControllerVer *string                 `json:"lifecycle_controller_ver,omitempty"`
	TotalPowerWatts        *float64                `json:"total_power_watts,omitempty"`
	SystemUUID             *string                 `json:"system_uuid,omitempty"`
	SystemRevision         *string                 `json:"system_revision,omitempty"`
	SystemUptimeSeconds    *int64                  `json:"system_uptime_seconds,omitempty"`
	BootMode               *string                 `json:"boot_mode,omitempty"`
	OSName                 *string                 `json:"os_name,omitempty"`
	OSVersion              *string                 `json:"os_version,omitempty"`
	OSKernel               *string                 `json:"os_kernel,omitempty"`
	OSHostname             *string                 `json:"os_hostname,omitempty"`
	OSUptimeSeconds        *int64                  `json:"os_uptime_seconds,omitempty"`
	Processors             []ProcessorInfo         `json:"processors,omitempty"`
	MemoryModules          []MemoryInfo            `json:"memory_modules,omitempty"`
	StorageControllers     []StorageControllerInfo `json:"storage_controllers,omitempty"`
	Drives                 []DriveInfo             `json:"drives,omitempty"`
	Temperatures           []TemperatureReading    `json:"temperatures,omitempty"`
	Fans                   []FanReading            `json:"fans,omitempty"`
	PowerSupplies          []PowerSupplyInfo       `json:"power_supplies,omitempty"`
	NICs                   []FullNICInfo           `json:"nics,omitempty"`
	SELEntries             []SELEntry              `json:"sel_entries,omitempty"`
	Accelerators           []AcceleratorInfo       `json:"accelerators,omitempty"`
	PCIeSlots              []PCIeSlotInfo          `json:"pcie_slots,omitempty"`
	Voltages               []VoltageReading        `json:"voltages,omitempty"`
	LifecycleLogs          []LifecycleLogEntry     `json:"lifecycle_logs,omitempty"`
	VirtualDisks           []VirtualDiskInfo       `json:"virtual_disks,omitempty"`
	StorageEnclosures      []StorageEnclosureInfo  `json:"storage_enclosures,omitempty"`
	LegacyNICs             []NICInfo               `json:"legacy_nics,omitempty"`
	LegacyDisks            []DiskInfo              `json:"legacy_disks,omitempty"`
}
// ── Discovery types ──────────────────────────────────────────────────────────

type DiscoveryRequest struct {
	TargetCIDR  string `json:"target_cidr" binding:"required"`
	Credentials struct {
		Username string `json:"username,omitempty"`
		Password string `json:"password,omitempty"`
	} `json:"credentials,omitempty"`
}

type DiscoveredDevice struct {
	IPAddress    string `json:"ip_address"`
	Hostname     string `json:"hostname,omitempty"`
	Vendor       string `json:"vendor,omitempty"`
	Model        string `json:"model,omitempty"`
	Protocol     string `json:"protocol,omitempty"`
	PortsOpen    []int  `json:"ports_open"`
	Status       string `json:"status"` // "discovered", "needs_credentials", "error"
	ErrorMessage string `json:"error_message,omitempty"`
}

type DiscoveryResponse struct {
	Results []DiscoveredDevice `json:"results"`
}
