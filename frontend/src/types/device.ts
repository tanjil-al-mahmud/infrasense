export type DeviceStatus = 'healthy' | 'warning' | 'critical' | 'unavailable' | 'unknown';

export type DeviceType =
  | 'redfish'
  | 'ipmi'
  | 'snmp'
  | 'proxmox'
  | 'linux_agent'
  | 'windows_agent';

export interface Device {
  id: string;
  hostname: string;
  ip_address: string;
  bmc_ip_address?: string;
  device_type: string;
  vendor?: string;
  management_controller?: string;
  protocol?: string;
  polling_interval?: number;
  ssl_verify?: boolean;
  location?: string;
  description?: string;
  tags?: string[];
  status: DeviceStatus;
  last_seen?: string;
  last_sync_at?: string;
  connection_status: 'connected' | 'failed' | 'unknown';
  connection_error?: string;
  created_at: string;
  updated_at: string;
}

export interface CreateDeviceRequest {
  hostname: string;
  ip_address: string;
  bmc_ip_address?: string;
  device_type: string;
  vendor?: string;
  management_controller?: string;
  protocol?: string;
  polling_interval?: number;
  ssl_verify?: boolean;
  location?: string;
  description?: string;
  tags?: string[];
}

export interface UpdateDeviceRequest {
  hostname?: string;
  ip_address?: string;
  bmc_ip_address?: string;
  device_type?: string;
  vendor?: string;
  management_controller?: string;
  protocol?: string;
  polling_interval?: number;
  ssl_verify?: boolean;
  location?: string;
  description?: string;
  tags?: string[];
}

export interface DeviceListParams {
  page?: number;
  page_size?: number;
  device_type?: DeviceType;
  status?: DeviceStatus;
  location?: string;
  search?: string;
  tags?: string[];
}

export interface DeviceCredentialRequest {
  protocol: string;
  username?: string;
  password?: string;
  community_string?: string;
  auth_protocol?: string;
  priv_protocol?: string;
  port?: number;
  http_scheme?: 'https' | 'http';
  ssl_verify?: boolean;
  polling_interval?: number;
  timeout_seconds?: number;
  retry_attempts?: number;
}

export interface ConnectionTestResult {
  success: boolean;
  message: string;
  redfish_version?: string;
  error_code?: string;
}

export interface ProcessorInfo {
  name: string;
  model?: string;
  manufacturer?: string;
  socket?: string;
  cores?: number;
  threads?: number;
  speed_mhz?: number;
  max_speed_mhz?: number;
  cache_size_mib?: number;
  temperature_celsius?: number | null;
  health?: string;
}

export interface MemoryInfo {
  name: string;
  capacity_gb?: number;
  manufacturer?: string;
  part_number?: string;
  serial_number?: string;
  location?: string;
  speed_mhz?: number;
  memory_type?: string;
  ecc_enabled?: boolean | null;
  health?: string;
}

export interface StorageControllerInfo {
  name: string;
  model?: string;
  manufacturer?: string;
  health?: string;
  firmware_version?: string;
  battery_health?: string;
}

export interface DriveInfo {
  name: string;
  model?: string;
  manufacturer?: string;
  serial_number?: string;
  capacity_gb?: number;
  media_type?: string;
  bus_protocol?: string;
  temperature_celsius?: number | null;
  read_policy?: string;
  write_policy?: string;
  health?: string;
  status?: string;
}

export interface VirtualDiskInfo {
  name: string;
  raid_level?: string;
  capacity_gb?: number;
  read_policy?: string;
  write_policy?: string;
  cache_policy?: string;
  health?: string;
  status?: string;
}

export interface StorageEnclosureInfo {
  name: string;
  backplane_id?: string;
  controller?: string;
  health?: string;
}

export interface TemperatureReading {
  name: string;
  reading_celsius?: number | null;
  upper_threshold_warn?: number;
  upper_threshold_crit?: number;
  health?: string;
}

export interface FanReading {
  name: string;
  reading_rpm?: number | null;
  lower_threshold_warn?: number;
  health?: string;
}

export interface PowerSupplyInfo {
  name: string;
  power_input_watts?: number;
  power_output_watts?: number;
  last_power_output_watts?: number;
  health?: string;
  status?: string;
  redundancy?: string;
}

export interface VoltageReading {
  name: string;
  reading_volts?: number | null;
  upper_threshold_warn?: number;
  lower_threshold_warn?: number;
  health?: string;
}

export interface FullNICInfo {
  name: string;
  model?: string;
  firmware_version?: string;
  mac_address: string;
  ipv4_address?: string;
  ipv6_address?: string;
  link_status?: string;
  speed_mbps?: number;
  health?: string;
}

export interface AcceleratorInfo {
  name: string;
  model?: string;
  manufacturer?: string;
  device_class?: string;
  health?: string;
}

export interface PCIeSlotInfo {
  name: string;
  slot_type?: string;
  pcie_type?: string;
  status?: string;
  device_name?: string;
}

export interface SELEntry {
  id: string;
  severity: string;
  message: string;
  created?: string;
}

export interface LifecycleLogEntry {
  id: string;
  severity: string;
  message: string;
  category?: string;
  created?: string;
}

export interface SyncStep {
  name: string;
  status: 'ok' | 'error' | 'skipped';
  message?: string;
}

export interface DeviceSyncResult {
  success: boolean;
  message: string;
  // System identity
  manufacturer?: string;
  model?: string;
  serial_number?: string;
  service_tag?: string;
  asset_tag?: string;
  system_uuid?: string;
  system_revision?: string;
  power_state?: string;
  health_status?: string;
  bios_version?: string;
  lifecycle_controller_version?: string;
  system_uptime_seconds?: number;
  boot_mode?: string;
  // BMC info
  bmc_firmware?: string;
  bmc_hardware_version?: string;
  bmc_mac_address?: string;
  bmc_dns_name?: string;
  bmc_license?: string;
  bmc_name?: string;
  // Hardware
  processors?: ProcessorInfo[];
  memory_total_gb?: number;
  memory_modules?: MemoryInfo[];
  storage_controllers?: StorageControllerInfo[];
  drives?: DriveInfo[];
  virtual_disks?: VirtualDiskInfo[];
  temperatures?: TemperatureReading[];
  fans?: FanReading[];
  power_supplies?: PowerSupplyInfo[];
  total_power_watts?: number;
  voltages?: VoltageReading[];
  nics?: FullNICInfo[];
  accelerators?: AcceleratorInfo[];
  pcie_slots?: PCIeSlotInfo[];
  intrusion_status?: string;
  sel_entries?: SELEntry[];
  lifecycle_logs?: LifecycleLogEntry[];
  steps?: SyncStep[];
}

export interface PaginatedResponse<T> {
  data: T[];
  meta: { page: number; page_size: number; total: number };
}

export interface ApiResponse<T> { data: T }
export interface ApiError { error: string; code?: string }

// Persisted inventory (from GET /devices/:id/inventory)
export interface DeviceInventory {
  device_id: string;
  manufacturer?: string;
  system_model?: string;
  service_tag?: string;
  asset_tag?: string;
  system_uuid?: string;
  system_revision?: string;
  power_state?: string;
  health_status?: string;
  system_uptime_seconds?: number;
  boot_mode?: string;
  cpu_model?: string;
  cpu_cores?: number;
  cpu_threads?: number;
  ram_total_gb?: number;
  firmware_bmc?: string;
  firmware_bios?: string;
  firmware_raid?: string;
  lifecycle_controller_version?: string;
  bmc_name?: string;
  bmc_mac_address?: string;
  bmc_dns_name?: string;
  bmc_license?: string;
  bmc_hardware_version?: string;
  total_power_watts?: number;
  processors?: ProcessorInfo[];
  memory_modules?: MemoryInfo[];
  storage_controllers?: StorageControllerInfo[];
  drives?: DriveInfo[];
  virtual_disks?: VirtualDiskInfo[];
  storage_enclosures?: StorageEnclosureInfo[];
  temperatures?: TemperatureReading[];
  fans?: FanReading[];
  power_supplies?: PowerSupplyInfo[];
  voltages?: VoltageReading[];
  nics?: FullNICInfo[];
  accelerators?: AcceleratorInfo[];
  pcie_slots?: PCIeSlotInfo[];
  intrusion_status?: string;
  sel_entries?: SELEntry[];
  lifecycle_logs?: LifecycleLogEntry[];
  collected_at: string;
}

export interface DeviceMetricPoint { timestamp: number; value: number }

export interface NetworkInterfaceMetrics {
  name: string;
  bytes_sent?: DeviceMetricPoint[];
  bytes_recv?: DeviceMetricPoint[];
  errors_in?: DeviceMetricPoint[];
  errors_out?: DeviceMetricPoint[];
}

export interface DeviceMetrics {
  temperature?: DeviceMetricPoint[];
  fan_speed?: DeviceMetricPoint[];
  power_consumption?: DeviceMetricPoint[];
  network_interfaces?: NetworkInterfaceMetrics[];
}

export interface PowerControlRequest {
  reset_type: 'On' | 'ForceOff' | 'GracefulShutdown' | 'ForceRestart' | 'PowerCycle' | 'GracefulRestart';
}

export interface PowerControlResult {
  success: boolean;
  message: string;
  reset_type: string;
}

export interface BootControlRequest {
  target: 'Pxe' | 'Cd' | 'Hdd' | 'BiosSetup' | 'None';
  once?: boolean;
}

export interface BootControlResult {
  success: boolean;
  message: string;
  target?: string;
  once?: boolean;
}

export interface OSInfo {
  name: string;
  version?: string;
  kernel?: string;
  hostname?: string;
  uptime_seconds?: number;
}

export interface ProtocolProbeResult {
  protocol: string;
  available: boolean;
  port: number;
  error?: string;
}

export interface ProtocolDetectionResult {
  bmc_ip_address: string;
  vendor: string;
  model: string;
  bmc_type: string;
  supported_protocols: string[];
  recommended_protocol: string;
  probes: ProtocolProbeResult[];
}

export type DeviceLogSeverity = 'Critical' | 'Warning' | 'Info';
export type DeviceLogSource = 'sel' | 'lifecycle';
export type DeviceLogSeverityFilter = 'critical' | 'warning' | 'all';

export interface DeviceLogEntry {
  id: string;
  source: DeviceLogSource;
  severity: DeviceLogSeverity;
  message: string;
  timestamp: string;
}

export interface DeviceLogsResponse {
  logs: DeviceLogEntry[];
  total: number;
}

export interface DeviceLogsParams {
  severity?: DeviceLogSeverityFilter;
  limit?: number;
}

export interface TelemetryEvent {
  device_id: string;
  timestamp: number;
  power_state?: string;
  health_status?: string;
  temperatures?: TemperatureReading[];
  fans?: FanReading[];
  power_supplies?: PowerSupplyInfo[];
  total_power_watts?: number;
  voltages?: VoltageReading[];
  error?: string;
}
