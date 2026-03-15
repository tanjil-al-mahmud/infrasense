/**
 * Device API Service
 * 
 * API functions for device management operations.
 * All functions use the configured apiClient with JWT authentication.
 */

import apiClient from './api';
import {
  Device,
  CreateDeviceRequest,
  UpdateDeviceRequest,
  DeviceListParams,
  DeviceCredentialRequest,
  PaginatedResponse,
  ApiResponse,
  DeviceInventory,
  DeviceMetrics,
  DeviceMetricPoint,
  NetworkInterfaceMetrics,
  ConnectionTestResult,
  DeviceSyncResult,
  PowerControlRequest,
  PowerControlResult,
  BootControlRequest,
  BootControlResult,
  ProtocolDetectionResult,
  DeviceLogsResponse,
  DeviceLogsParams,
} from '../types/device';

/**
 * Fetch paginated list of devices with optional filtering
 * GET /api/v1/devices
 */
export const fetchDevices = async (
  params?: DeviceListParams
): Promise<PaginatedResponse<Device>> => {
  const response = await apiClient.get<PaginatedResponse<Device>>('/devices', {
    params,
  });
  return response.data || { data: [], meta: { page: 1, page_size: 20, total: 0 } };
};

/**
 * Fetch a single device by ID
 * GET /api/v1/devices/{id}
 */
export const fetchDevice = async (id: string): Promise<Device> => {
  const response = await apiClient.get<ApiResponse<Device>>(`/devices/${id}`);
  return response.data.data;
};

/**
 * Create a new device
 * POST /api/v1/devices
 */
export const createDevice = async (
  device: CreateDeviceRequest
): Promise<Device> => {
  const response = await apiClient.post<ApiResponse<Device>>('/devices', device);
  return response.data.data;
};

/**
 * Update an existing device
 * PUT /api/v1/devices/{id}
 */
export const updateDevice = async (
  id: string,
  device: UpdateDeviceRequest
): Promise<Device> => {
  const response = await apiClient.put<ApiResponse<Device>>(
    `/devices/${id}`,
    device
  );
  return response.data.data;
};

/**
 * Delete a device
 * DELETE /api/v1/devices/{id}
 */
export const deleteDevice = async (id: string): Promise<void> => {
  await apiClient.delete(`/devices/${id}`);
};

/**
 * Fetch hardware inventory for a device
 * GET /api/v1/devices/{id}/inventory
 */
export const fetchDeviceInventory = async (id: string): Promise<DeviceInventory> => {
  const response = await apiClient.get<ApiResponse<DeviceInventory>>(
    `/devices/${id}/inventory`
  );
  return response.data.data;
};

/**
 * Fetch recent metrics for a device
 * GET /api/v1/devices/{id}/metrics
 * Backend returns VictoriaMetrics query_range data keyed by metric name.
 * We transform it into the DeviceMetrics shape expected by the UI.
 */
export const fetchDeviceMetrics = async (id: string): Promise<DeviceMetrics> => {
  const response = await apiClient.get<ApiResponse<Record<string, any>>>(
    `/devices/${id}/metrics`
  );
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const raw = (response.data as any).data ?? response.data ?? {};

  // Helper: extract [{timestamp, value}] from VictoriaMetrics query_range result
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const extractPoints = (metricData: any): DeviceMetricPoint[] => {
    if (!metricData) return [];
    const result = metricData?.result ?? [];
    const points: DeviceMetricPoint[] = [];
    for (const series of result) {
      for (const [ts, val] of series?.values ?? []) {
        const v = parseFloat(val);
        if (!isNaN(v)) points.push({ timestamp: Number(ts), value: v });
      }
    }
    // Sort by timestamp and deduplicate
    return points.sort((a, b) => a.timestamp - b.timestamp);
  };

  return {
    temperature: extractPoints(raw['infrasense_redfish_temperature_celsius']),
    fan_speed: extractPoints(raw['infrasense_redfish_fan_speed_rpm']),
    power_consumption: extractPoints(raw['infrasense_redfish_psu_power_watts']),
    network_interfaces: extractNetworkInterfaces(raw),
  };
};

// Extract per-interface network metrics from SSH collector metric names:
// infrasense_ssh_net_bytes_sent{device_id, interface}
// infrasense_ssh_net_bytes_recv{device_id, interface}
// infrasense_ssh_net_errin{device_id, interface}
// infrasense_ssh_net_errout{device_id, interface}
// eslint-disable-next-line @typescript-eslint/no-explicit-any
function extractNetworkInterfaces(raw: Record<string, any>): NetworkInterfaceMetrics[] {
  const ifaceMap = new Map<string, NetworkInterfaceMetrics>();

  const metricKeys: Array<{ key: string; field: keyof NetworkInterfaceMetrics }> = [
    { key: 'infrasense_ssh_net_bytes_sent', field: 'bytes_sent' },
    { key: 'infrasense_ssh_net_bytes_recv', field: 'bytes_recv' },
    { key: 'infrasense_ssh_net_errin',      field: 'errors_in' },
    { key: 'infrasense_ssh_net_errout',     field: 'errors_out' },
  ];

  for (const { key, field } of metricKeys) {
    const metricData = raw[key];
    if (!metricData?.result) continue;
    for (const series of metricData.result) {
      const ifaceName: string = series?.metric?.interface ?? series?.metric?.device ?? 'unknown';
      if (!ifaceMap.has(ifaceName)) {
        ifaceMap.set(ifaceName, { name: ifaceName });
      }
      const entry = ifaceMap.get(ifaceName)!;
      const points: DeviceMetricPoint[] = (series?.values ?? [])
        .map(([ts, val]: [number, string]) => ({ timestamp: Number(ts), value: parseFloat(val) }))
        .filter((pt: DeviceMetricPoint) => !isNaN(pt.value))
        .sort((a: DeviceMetricPoint, b: DeviceMetricPoint) => a.timestamp - b.timestamp);
      if (points.length > 0) {
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        (entry as any)[field] = points;
      }
    }
  }

  return Array.from(ifaceMap.values()).sort((a, b) => a.name.localeCompare(b.name));
}

/**
 * Save credentials for a device
 * POST /api/v1/devices/{id}/credentials
 */
export const saveDeviceCredentials = async (
  id: string,
  cred: DeviceCredentialRequest
): Promise<void> => {
  await apiClient.post(`/devices/${id}/credentials`, cred);
};

/**
 * Test BMC connection for a device
 * POST /api/v1/devices/{id}/test-connection
 */
export const testDeviceConnection = async (id: string): Promise<ConnectionTestResult> => {
  const response = await apiClient.post<ApiResponse<ConnectionTestResult>>(`/devices/${id}/test-connection`);
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  return (response.data as any).data ?? response.data;
};

/**
 * Sync device hardware data from BMC
 * POST /api/v1/devices/{id}/sync
 */
export const syncDevice = async (id: string): Promise<DeviceSyncResult> => {
  const response = await apiClient.post<DeviceSyncResult>(`/devices/${id}/sync`);
  // Backend returns the result directly (not wrapped in {data: ...})
  return response.data;
};

/**
 * Send a power control action to the BMC
 * POST /api/v1/devices/{id}/power
 * resetType: On | ForceOff | GracefulShutdown | ForceRestart | PowerCycle | GracefulRestart
 */
export const powerControlDevice = async (
  id: string,
  req: PowerControlRequest
): Promise<PowerControlResult> => {
  const response = await apiClient.post<PowerControlResult>(`/devices/${id}/power`, req);
  return response.data;
};

/**
 * Send a boot override action to the BMC
 * POST /api/v1/devices/{id}/boot
 */
export const bootControlDevice = async (
  id: string,
  req: BootControlRequest
): Promise<BootControlResult> => {
  const response = await apiClient.post<BootControlResult>(`/devices/${id}/boot`, req);
  return response.data;
};

/**
 * Auto-detect supported protocols for a BMC IP
 * POST /api/v1/devices/detect-protocol
 */
export const detectProtocol = async (bmcIP: string, username?: string, password?: string): Promise<ProtocolDetectionResult> => {
  const response = await apiClient.post<ProtocolDetectionResult>('/devices/detect-protocol', { 
    bmc_ip: bmcIP,
    username,
    password
  });
  return response.data;
};

/**
 * Fetch device logs (SEL + lifecycle) with optional severity filter and limit
 * GET /api/v1/devices/{id}/logs
 */
export const fetchDeviceLogs = async (
  id: string,
  params?: DeviceLogsParams
): Promise<DeviceLogsResponse> => {
  const response = await apiClient.get<DeviceLogsResponse>(`/devices/${id}/logs`, {
    params,
  });
  return response.data;
};

/**
 * Open an SSE stream for real-time telemetry
 * GET /api/v1/devices/{id}/stream
 * Returns an EventSource — caller is responsible for closing it.
 */
export const streamDeviceTelemetry = (id: string, token: string): EventSource => {
  const base = (apiClient.defaults.baseURL ?? '').replace(/\/$/, '');
  return new EventSource(`${base}/devices/${id}/stream?token=${encodeURIComponent(token)}`);
};
