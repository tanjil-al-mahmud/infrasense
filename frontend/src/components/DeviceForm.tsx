import React, { useState } from 'react';
import { useForm, SubmitHandler, FieldValues } from 'react-hook-form';
import { useNavigate } from 'react-router-dom';
import { useCreateDevice, useUpdateDevice } from '../hooks/useDevices';
import { detectProtocol } from '../services/deviceApi';
import { Device, CreateDeviceRequest, UpdateDeviceRequest, DeviceType, ProtocolDetectionResult } from '../types/device';

const DEVICE_TYPE_LABELS: Record<DeviceType, string> = {
  ipmi: 'IPMI',
  redfish: 'Redfish',
  snmp: 'SNMP',
  proxmox: 'Proxmox',
  linux_agent: 'Linux Agent',
  windows_agent: 'Windows Agent',
};

const ALL_DEVICE_TYPES: DeviceType[] = ['redfish', 'ipmi', 'snmp', 'proxmox', 'linux_agent', 'windows_agent'];

const IP_PATTERN = /^((\d{1,3}\.){3}\d{1,3}|([\da-fA-F]{0,4}:){2,7}[\da-fA-F]{0,4}|::[\da-fA-F]{0,4})$/;

interface FormValues extends FieldValues {
  hostname: string;
  ip_address: string;
  bmc_ip_address?: string;
  device_type: DeviceType;
  location?: string;
  tags?: string;
}

export interface DeviceFormProps {
  device?: Device;
  onSuccess?: (device: Device) => void;
  onCancel?: () => void;
}

const DeviceForm: React.FC<DeviceFormProps> = ({ device, onSuccess, onCancel }) => {
  const navigate = useNavigate();
  const isEditMode = !!device;

  const createDevice = useCreateDevice();
  const updateDevice = useUpdateDevice();

  const [detecting, setDetecting] = useState(false);
  const [detectionResult, setDetectionResult] = useState<ProtocolDetectionResult | null>(null);
  const [detectionError, setDetectionError] = useState<string | null>(null);

  const {
    register,
    handleSubmit,
    watch,
    setValue,
    formState: { errors, isSubmitting },
    setError,
  } = useForm<FormValues>({
    defaultValues: {
      hostname: device?.hostname ?? '',
      ip_address: device?.ip_address ?? '',
      bmc_ip_address: device?.bmc_ip_address ?? '',
      device_type: (device?.device_type as DeviceType) ?? 'redfish',
      location: device?.location ?? '',
      tags: device?.tags?.join(', ') ?? '',
    },
  });

  const bmcIP = watch('bmc_ip_address');
  const ipAddress = watch('ip_address');

  const handleAutoDetect = async () => {
    const target = bmcIP || ipAddress;
    if (!target || !IP_PATTERN.test(target)) {
      setDetectionError('Enter a valid IP address or BMC IP first');
      return;
    }
    setDetecting(true);
    setDetectionResult(null);
    setDetectionError(null);
    try {
      const result = await detectProtocol(target);
      setDetectionResult(result);
      // Auto-select recommended protocol
      const recommended = result.recommended_protocol as DeviceType;
      if (recommended && ALL_DEVICE_TYPES.includes(recommended)) {
        setValue('device_type', recommended);
      }
    } catch (err) {
      setDetectionError(err instanceof Error ? err.message : 'Protocol detection failed');
    } finally {
      setDetecting(false);
    }
  };

  const parseTags = (raw: string): string[] =>
    raw.split(',').map(t => t.trim()).filter(Boolean);

  const onSubmit: SubmitHandler<FormValues> = async (data) => {
    const tags = parseTags(data.tags ?? '');
    try {
      if (isEditMode && device) {
        const payload: UpdateDeviceRequest = {
          hostname: data.hostname,
          ip_address: data.ip_address,
          bmc_ip_address: data.bmc_ip_address || undefined,
          device_type: data.device_type,
          location: data.location || undefined,
          tags: tags.length > 0 ? tags : undefined,
        };
        const updated = await updateDevice.mutateAsync({ id: device.id, data: payload });
        onSuccess?.(updated);
      } else {
        const payload: CreateDeviceRequest = {
          hostname: data.hostname,
          ip_address: data.ip_address,
          bmc_ip_address: data.bmc_ip_address || undefined,
          device_type: data.device_type,
          location: data.location || undefined,
          tags: tags.length > 0 ? tags : undefined,
        };
        const created = await createDevice.mutateAsync(payload);
        onSuccess?.(created);
        if (!onSuccess) navigate(`/devices/${created.id}`);
      }
    } catch (err) {
      setError('root', { message: err instanceof Error ? err.message : 'An unexpected error occurred' });
    }
  };

  const handleCancel = () => {
    if (onCancel) onCancel();
    else navigate(-1);
  };

  return (
    <form onSubmit={handleSubmit(onSubmit)} noValidate style={styles.form}>
      {errors.root && (
        <div style={styles.errorBanner} role="alert">{errors.root.message}</div>
      )}

      {/* Hostname */}
      <div style={styles.fieldGroup}>
        <label htmlFor="hostname" style={styles.label}>Hostname <span style={styles.required}>*</span></label>
        <input id="hostname" type="text" placeholder="e.g. server-01.example.com" disabled={isSubmitting}
          style={{ ...styles.input, ...(errors.hostname ? styles.inputError : {}) }}
          {...register('hostname', { required: 'Hostname is required' })} />
        {errors.hostname && <p style={styles.fieldError} role="alert">{errors.hostname.message}</p>}
      </div>

      {/* IP Address */}
      <div style={styles.fieldGroup}>
        <label htmlFor="ip_address" style={styles.label}>IP Address <span style={styles.required}>*</span></label>
        <input id="ip_address" type="text" placeholder="e.g. 192.168.1.10" disabled={isSubmitting}
          style={{ ...styles.input, ...(errors.ip_address ? styles.inputError : {}) }}
          {...register('ip_address', { required: 'IP address is required', pattern: { value: IP_PATTERN, message: 'Enter a valid IPv4 or IPv6 address' } })} />
        {errors.ip_address && <p style={styles.fieldError} role="alert">{errors.ip_address.message}</p>}
      </div>

      {/* BMC IP Address */}
      <div style={styles.fieldGroup}>
        <label htmlFor="bmc_ip_address" style={styles.label}>BMC IP Address</label>
        <input id="bmc_ip_address" type="text" placeholder="e.g. 192.168.1.11 (optional)" disabled={isSubmitting}
          style={{ ...styles.input, ...(errors.bmc_ip_address ? styles.inputError : {}) }}
          {...register('bmc_ip_address', { validate: val => !val || IP_PATTERN.test(val) || 'Enter a valid IPv4 or IPv6 address' })} />
        {errors.bmc_ip_address && <p style={styles.fieldError} role="alert">{errors.bmc_ip_address.message}</p>}
      </div>

      {/* Protocol Auto-Detection */}
      <div style={styles.fieldGroup}>
        <label style={styles.label}>Protocol Detection</label>
        <div style={{ display: 'flex', gap: '0.5rem', alignItems: 'center' }}>
          <button type="button" onClick={handleAutoDetect} disabled={detecting || isSubmitting}
            style={{ ...styles.detectBtn, ...(detecting ? styles.btnDisabled : {}) }}>
            {detecting ? '🔍 Detecting…' : '🔍 Auto-Detect Protocol'}
          </button>
          <span style={{ fontSize: '0.75rem', color: '#9ca3af' }}>
            Probes: Redfish → IPMI → SNMP → SSH
          </span>
        </div>

        {detectionError && (
          <div style={{ ...styles.errorBanner, marginTop: '0.5rem' }}>{detectionError}</div>
        )}

        {detectionResult && (
          <div style={styles.detectionResult}>
            <div style={{ fontSize: '0.8rem', fontWeight: 700, color: '#4ade80', marginBottom: '0.5rem' }}>
              ✅ Recommended: {detectionResult.recommended_protocol.toUpperCase()}
            </div>
            <div style={{ display: 'flex', flexWrap: 'wrap', gap: '0.4rem' }}>
              {detectionResult.probes.map(probe => (
                <span key={probe.protocol} style={{
                  fontSize: '0.7rem', fontWeight: 600, padding: '2px 8px', borderRadius: 9999,
                  background: probe.available ? '#052e16' : '#1e293b',
                  border: `1px solid ${probe.available ? '#16a34a' : '#334155'}`,
                  color: probe.available ? '#4ade80' : '#64748b',
                }}>
                  {probe.protocol.toUpperCase()} :{probe.port} {probe.available ? '✓' : '✗'}
                </span>
              ))}
            </div>
          </div>
        )}
      </div>

      {/* Device Type */}
      <div style={styles.fieldGroup}>
        <label htmlFor="device_type" style={styles.label}>Device Type <span style={styles.required}>*</span></label>
        <select id="device_type" disabled={isSubmitting}
          style={{ ...styles.input, ...(errors.device_type ? styles.inputError : {}) }}
          {...register('device_type', { required: 'Device type is required' })}>
          {ALL_DEVICE_TYPES.map(t => (
            <option key={t} value={t}>{DEVICE_TYPE_LABELS[t]}</option>
          ))}
        </select>
        {errors.device_type && <p style={styles.fieldError} role="alert">{errors.device_type.message}</p>}
        <p style={styles.hint}>Auto-detect will suggest the best protocol. You can override manually.</p>
      </div>

      {/* Location */}
      <div style={styles.fieldGroup}>
        <label htmlFor="location" style={styles.label}>Location</label>
        <input id="location" type="text" placeholder="e.g. Rack A, Row 3 (optional)" disabled={isSubmitting}
          style={styles.input} {...register('location')} />
      </div>

      {/* Tags */}
      <div style={styles.fieldGroup}>
        <label htmlFor="tags" style={styles.label}>Tags</label>
        <input id="tags" type="text" placeholder="e.g. production, web, critical (comma-separated)" disabled={isSubmitting}
          style={styles.input} {...register('tags')} />
        <p style={styles.hint}>Separate multiple tags with commas.</p>
      </div>

      {/* Actions */}
      <div style={styles.actions}>
        <button type="button" onClick={handleCancel} disabled={isSubmitting} style={styles.cancelBtn}>Cancel</button>
        <button type="submit" disabled={isSubmitting}
          style={{ ...styles.submitBtn, ...(isSubmitting ? styles.btnDisabled : {}) }}>
          {isSubmitting ? (isEditMode ? 'Saving...' : 'Registering...') : (isEditMode ? 'Save Changes' : 'Register Device')}
        </button>
      </div>
    </form>
  );
};

const styles: Record<string, React.CSSProperties> = {
  form: { display: 'flex', flexDirection: 'column', gap: '1rem' },
  errorBanner: { backgroundColor: '#fee2e2', border: '1px solid #fca5a5', borderRadius: '4px', color: '#b91c1c', fontSize: '0.875rem', padding: '0.75rem 1rem' },
  fieldGroup: { display: 'flex', flexDirection: 'column', gap: '0.25rem' },
  label: { fontSize: '0.875rem', fontWeight: 500, color: '#374151' },
  required: { color: '#ef4444' },
  input: { border: '1px solid #d1d5db', borderRadius: '4px', fontSize: '0.875rem', padding: '0.5rem 0.75rem', width: '100%', boxSizing: 'border-box' as const, backgroundColor: '#fff', outline: 'none' },
  inputError: { borderColor: '#ef4444' },
  fieldError: { color: '#ef4444', fontSize: '0.75rem', margin: 0 },
  hint: { color: '#9ca3af', fontSize: '0.75rem', margin: 0 },
  actions: { display: 'flex', gap: '0.75rem', justifyContent: 'flex-end', marginTop: '0.5rem' },
  cancelBtn: { backgroundColor: '#f3f4f6', border: '1px solid #d1d5db', borderRadius: '4px', color: '#374151', cursor: 'pointer', fontSize: '0.875rem', fontWeight: 500, padding: '0.5rem 1rem' },
  submitBtn: { backgroundColor: '#2563eb', border: 'none', borderRadius: '4px', color: '#fff', cursor: 'pointer', fontSize: '0.875rem', fontWeight: 600, padding: '0.5rem 1.25rem' },
  btnDisabled: { opacity: 0.6, cursor: 'not-allowed' },
  detectBtn: { backgroundColor: '#1e293b', border: '1px solid #3b82f6', borderRadius: '4px', color: '#60a5fa', cursor: 'pointer', fontSize: '0.8rem', fontWeight: 600, padding: '0.45rem 0.9rem', whiteSpace: 'nowrap' as const },
  detectionResult: { marginTop: '0.5rem', background: '#0f172a', border: '1px solid #16a34a', borderRadius: '6px', padding: '0.75rem' },
};

export default DeviceForm;
