
import React, { useState } from 'react';

import { useForm } from 'react-hook-form';

import { useCreateDevice } from '../../hooks/useDevices';

import { saveDeviceCredentials, detectProtocol } from '../../services/deviceApi';

import { Device, ProtocolDetectionResult } from '../../types/device';
import { useToast } from '../../contexts/ToastContext';



export interface DeviceOption {

  value: string;

  label: string;

  vendor: string;

  mgmtController: string;

  protocol: 'redfish' | 'ipmi' | 'snmp' | 'ssh' | 'proxmox' | 'agent';

  defaultPort: number;

}



export const DEVICE_OPTIONS: DeviceOption[] = [

  // Dell

  { value: 'dell_idrac9_redfish',  label: 'Dell iDRAC9 (Redfish)',  vendor: 'Dell',      mgmtController: 'iDRAC9',   protocol: 'redfish', defaultPort: 443 },

  { value: 'dell_idrac8_ipmi',     label: 'Dell iDRAC8 (IPMI)',     vendor: 'Dell',      mgmtController: 'iDRAC8',   protocol: 'ipmi',    defaultPort: 623 },

  { value: 'dell_idrac7_ipmi',     label: 'Dell iDRAC7 (IPMI)',     vendor: 'Dell',      mgmtController: 'iDRAC7',   protocol: 'ipmi',    defaultPort: 623 },

  // HPE

  { value: 'hpe_ilo5_redfish',     label: 'HPE iLO5 (Redfish)',     vendor: 'HPE',       mgmtController: 'iLO5',     protocol: 'redfish', defaultPort: 443 },

  { value: 'hpe_ilo4_ipmi',        label: 'HPE iLO4 (IPMI)',        vendor: 'HPE',       mgmtController: 'iLO4',     protocol: 'ipmi',    defaultPort: 623 },

  { value: 'hpe_ilo3_ipmi',        label: 'HPE iLO3 (IPMI)',        vendor: 'HPE',       mgmtController: 'iLO3',     protocol: 'ipmi',    defaultPort: 623 },

  // Lenovo

  { value: 'lenovo_xcc_redfish',   label: 'Lenovo XClarity (Redfish)', vendor: 'Lenovo', mgmtController: 'XCC',      protocol: 'redfish', defaultPort: 443 },

  { value: 'lenovo_xcc_ipmi',      label: 'Lenovo XClarity (IPMI)',    vendor: 'Lenovo', mgmtController: 'XCC',      protocol: 'ipmi',    defaultPort: 623 },

  // Cisco

  { value: 'cisco_cimc_redfish',   label: 'Cisco CIMC (Redfish)',   vendor: 'Cisco',     mgmtController: 'CIMC',     protocol: 'redfish', defaultPort: 443 },

  { value: 'cisco_cimc_ipmi',      label: 'Cisco CIMC (IPMI)',      vendor: 'Cisco',     mgmtController: 'CIMC',     protocol: 'ipmi',    defaultPort: 623 },

  // Huawei

  { value: 'huawei_ibmc_redfish',  label: 'Huawei iBMC (Redfish)',  vendor: 'Huawei',    mgmtController: 'iBMC',     protocol: 'redfish', defaultPort: 443 },

  { value: 'huawei_ibmc_ipmi',     label: 'Huawei iBMC (IPMI)',     vendor: 'Huawei',    mgmtController: 'iBMC',     protocol: 'ipmi',    defaultPort: 623 },

  // Ericsson

  { value: 'ericsson_cru_redfish', label: 'Ericsson CRU (Redfish)', vendor: 'Ericsson',  mgmtController: 'CRU',      protocol: 'redfish', defaultPort: 443 },

  { value: 'ericsson_bmc_ipmi',    label: 'Ericsson BMC (IPMI)',    vendor: 'Ericsson',  mgmtController: 'BMC',      protocol: 'ipmi',    defaultPort: 623 },

  { value: 'ericsson_ssh',         label: 'Ericsson (SSH)',         vendor: 'Ericsson',  mgmtController: 'SSH',      protocol: 'ssh',     defaultPort: 22 },

  // Supermicro

  { value: 'supermicro_redfish',   label: 'Supermicro BMC (Redfish)', vendor: 'Supermicro', mgmtController: 'BMC',   protocol: 'redfish', defaultPort: 443 },

  { value: 'supermicro_ipmi',      label: 'Supermicro IPMI',          vendor: 'Supermicro', mgmtController: 'BMC',   protocol: 'ipmi',    defaultPort: 623 },

  // Fujitsu

  { value: 'fujitsu_irmc_redfish', label: 'Fujitsu iRMC (Redfish)', vendor: 'Fujitsu',   mgmtController: 'iRMC',    protocol: 'redfish', defaultPort: 443 },

  { value: 'fujitsu_irmc_ipmi',    label: 'Fujitsu iRMC (IPMI)',    vendor: 'Fujitsu',   mgmtController: 'iRMC',    protocol: 'ipmi',    defaultPort: 623 },

  // ASUS

  { value: 'asus_asmb_ipmi',       label: 'ASUS ASMB (IPMI)',       vendor: 'ASUS',      mgmtController: 'ASMB',    protocol: 'ipmi',    defaultPort: 623 },

  // Gigabyte

  { value: 'gigabyte_bmc_redfish', label: 'Gigabyte BMC (Redfish)', vendor: 'Gigabyte',  mgmtController: 'BMC',     protocol: 'redfish', defaultPort: 443 },

  { value: 'gigabyte_bmc_ipmi',    label: 'Gigabyte BMC (IPMI)',    vendor: 'Gigabyte',  mgmtController: 'BMC',     protocol: 'ipmi',    defaultPort: 623 },

  // IEIT

  { value: 'ieit_bmc_ipmi',        label: 'IEIT BMC (IPMI)',        vendor: 'IEIT',      mgmtController: 'BMC',     protocol: 'ipmi',    defaultPort: 623 },

  // Generic Server
  { value: 'generic_ipmi',         label: 'Generic IPMI / Legacy BMC', vendor: 'Generic',   mgmtController: 'BMC',     protocol: 'ipmi',    defaultPort: 623 },
  { value: 'generic_ssh',          label: 'Generic Server (SSH)',   vendor: 'Generic',   mgmtController: 'SSH',     protocol: 'ssh',     defaultPort: 22 },

  // UPS

  { value: 'apc_ups_snmp',         label: 'APC UPS (SNMP)',         vendor: 'APC',       mgmtController: 'SNMP',    protocol: 'snmp',    defaultPort: 161 },

  { value: 'eaton_ups_snmp',       label: 'Eaton UPS (SNMP)',       vendor: 'Eaton',     mgmtController: 'SNMP',    protocol: 'snmp',    defaultPort: 161 },

  { value: 'generic_ups_snmp',     label: 'Generic UPS (SNMP)',     vendor: 'Generic',   mgmtController: 'SNMP',    protocol: 'snmp',    defaultPort: 161 },

  // Virtualization

  { value: 'proxmox',              label: 'Proxmox Node',           vendor: 'Proxmox',   mgmtController: 'Proxmox', protocol: 'proxmox', defaultPort: 8006 },

  // OS Agent

  { value: 'linux_agent',          label: 'Linux (node_exporter)',  vendor: 'Linux',     mgmtController: 'Agent',   protocol: 'agent',   defaultPort: 9100 },

  { value: 'windows_agent',        label: 'Windows (win_exporter)', vendor: 'Windows',   mgmtController: 'Agent',   protocol: 'agent',   defaultPort: 9182 },

];



const VENDORS = [...new Set(DEVICE_OPTIONS.map((d) => d.vendor))];



interface FormValues {

  name: string; ip_address: string; device_option: string; location: string; description: string;

  bmc_ip: string; bmc_username: string; bmc_password: string;

  redfish_port: string; redfish_scheme: 'https' | 'http'; skip_tls: boolean;

  ipmi_port: string;

  ssh_port: string; ssh_username: string; ssh_password: string; ssh_key: string;

  polling_interval: string; timeout_seconds: string; retry_attempts: string;

  snmp_version: 'v2c' | 'v3'; community_string: string; snmp_username: string;

  auth_protocol: string; auth_password: string; priv_protocol: string; priv_password: string; snmp_port: string;

  proxmox_ip: string; proxmox_token_id: string; proxmox_token_secret: string; proxmox_port: string; proxmox_node: string;

  agent_ip: string; agent_port: string;

}



interface AddDeviceModalProps { onClose: () => void; onSuccess: (device: Device) => void }



const AddDeviceModal: React.FC<AddDeviceModalProps> = ({ onClose, onSuccess }) => {

  const createDevice = useCreateDevice();
  const { showToast } = useToast();
  const [isDetecting, setIsDetecting] = useState(false);
  const [detectionResult, setDetectionResult] = useState<ProtocolDetectionResult | null>(null);
  const [manualMode, setManualMode] = useState(false);
  const [scanUsername, setScanUsername] = useState('');
  const [scanPassword, setScanPassword] = useState('');
  const [selectedProtocol, setSelectedProtocol] = useState('');

  const [showBmcPw, setShowBmcPw] = useState(false);

  const [showSshPw, setShowSshPw] = useState(false);

  const [showProxmoxSecret, setShowProxmoxSecret] = useState(false);



  const { register, handleSubmit, watch, setValue, formState: { errors, isSubmitting }, setError } = useForm<FormValues>({

    defaultValues: {

      name: '', ip_address: '', device_option: '', location: '', description: '',

      bmc_ip: '', bmc_username: '', bmc_password: '',

      redfish_port: '443', redfish_scheme: 'https', skip_tls: true,

      ipmi_port: '623',

      ssh_port: '22', ssh_username: '', ssh_password: '', ssh_key: '',

      polling_interval: '60', timeout_seconds: '30', retry_attempts: '3',

      snmp_version: 'v2c', community_string: 'public', snmp_username: '',

      auth_protocol: 'SHA', auth_password: '', priv_protocol: 'AES', priv_password: '', snmp_port: '161',

      proxmox_ip: '', proxmox_token_id: '', proxmox_token_secret: '', proxmox_port: '8006', proxmox_node: '',

      agent_ip: '', agent_port: '',

    },

  });



  const selectedOption = watch('device_option');

  const snmpVersion = watch('snmp_version');

  const deviceInfo = DEVICE_OPTIONS.find((d) => d.value === selectedOption);

  const activeProtocol = (detectionResult && !manualMode) ? selectedProtocol : deviceInfo?.protocol;
  const protocol = activeProtocol; // Alias for JSX compatibility

  const defaultAgentPort = selectedOption === 'windows_agent' ? '9182' : '9100';



  const handleDetect = async () => {
    const ip = watch('ip_address');
    if (!ip) {
      setError('ip_address', { type: 'manual', message: 'Required for detection' });
      return;
    }
    setIsDetecting(true);
    setDetectionResult(null);
    try {
      const res = await detectProtocol(ip, scanUsername, scanPassword);
      setDetectionResult(res);
      if (res.recommended_protocol && res.recommended_protocol !== 'unknown') {
        setSelectedProtocol(res.recommended_protocol);
        setValue('bmc_ip', ip);
        if (scanUsername) setValue('bmc_username', scanUsername);
        if (scanPassword) setValue('bmc_password', scanPassword);
      }
      showToast('Device detection completed.', 'success');
    } catch (err: any) {
      showToast(err.message || 'Detection failed. Entering manual mode.', 'error');
      setManualMode(true);
    } finally {
      setIsDetecting(false);
    }
  };

  const onSubmit = async (data: FormValues) => {

    if (manualMode && !selectedOption) { setError('device_option', { message: 'Please select a device type' }); return; }
    if (!manualMode && detectionResult && !selectedProtocol) { setError('root', { message: 'Please select a protocol' }); return; }

    const isBMC = activeProtocol === 'redfish' || activeProtocol === 'ipmi';

    let primaryIP = data.ip_address;

    if (isBMC) primaryIP = data.bmc_ip || data.ip_address;

    else if (activeProtocol === 'proxmox') primaryIP = data.proxmox_ip || data.ip_address;

    else if (activeProtocol === 'agent') primaryIP = data.agent_ip || data.ip_address;

    else if (activeProtocol === 'ssh') primaryIP = data.ip_address;



    try {

      const device = await createDevice.mutateAsync({

        hostname: data.name,

        ip_address: primaryIP || data.ip_address,

        bmc_ip_address: isBMC ? (data.bmc_ip || undefined) : undefined,

        device_type: (detectionResult && !manualMode && activeProtocol) ? `${detectionResult.vendor || 'auto'}_${activeProtocol}`.toLowerCase() : selectedOption,

        vendor: (detectionResult && !manualMode) ? detectionResult.vendor : deviceInfo?.vendor,

        management_controller: (detectionResult && !manualMode) ? detectionResult.bmc_type : deviceInfo?.mgmtController,

        protocol: activeProtocol,

        polling_interval: data.polling_interval ? parseInt(data.polling_interval, 10) : 60,

        ssl_verify: activeProtocol === 'redfish' ? !data.skip_tls : undefined,

        location: data.location || undefined,

      });



      if (protocol === 'redfish') {

        await saveDeviceCredentials(device.id, {

          protocol: 'redfish', username: data.bmc_username || undefined, password: data.bmc_password || undefined,

          port: data.redfish_port ? parseInt(data.redfish_port, 10) : 443,

          http_scheme: data.redfish_scheme || 'https', ssl_verify: !data.skip_tls,

          polling_interval: parseInt(data.polling_interval, 10) || 60,

          timeout_seconds: parseInt(data.timeout_seconds, 10) || 30,

          retry_attempts: parseInt(data.retry_attempts, 10) || 3,

        });

      } else if (protocol === 'ipmi') {

        await saveDeviceCredentials(device.id, {

          protocol: 'ipmi', username: data.bmc_username || undefined, password: data.bmc_password || undefined,

          port: data.ipmi_port ? parseInt(data.ipmi_port, 10) : 623,

          polling_interval: parseInt(data.polling_interval, 10) || 60,

          timeout_seconds: parseInt(data.timeout_seconds, 10) || 30,

          retry_attempts: parseInt(data.retry_attempts, 10) || 3,

        });

      } else if (protocol === 'ssh') {

        await saveDeviceCredentials(device.id, {

          protocol: 'ssh', username: data.ssh_username || undefined, password: data.ssh_password || undefined,

          port: data.ssh_port ? parseInt(data.ssh_port, 10) : 22,

        });

      } else if (protocol === 'snmp') {

        if (data.snmp_version === 'v2c') {

          await saveDeviceCredentials(device.id, { protocol: 'snmp_v2c', community_string: data.community_string || undefined });

        } else {

          await saveDeviceCredentials(device.id, {

            protocol: 'snmp_v3', username: data.snmp_username || undefined, password: data.auth_password || undefined,

            auth_protocol: data.auth_protocol || undefined, priv_protocol: data.priv_protocol || undefined,

          });

        }

      } else if (protocol === 'proxmox') {

        await saveDeviceCredentials(device.id, {

          protocol: 'proxmox', username: data.proxmox_token_id || undefined, password: data.proxmox_token_secret || undefined,

        });

      }

      onSuccess(device);

    } catch (err) {

      setError('root', { message: err instanceof Error ? err.message : 'Failed to add device' });

    }

  };



  return (

    <div style={s.overlay} role="dialog" aria-modal="true" aria-label="Add Device">

      <div style={s.modal}>

        <div style={s.header}>

          <h2 style={s.title}>Add Device</h2>

          <button onClick={onClose} style={s.closeBtn} aria-label="Close">✕</button>

        </div>

        <form onSubmit={handleSubmit(onSubmit)} noValidate style={s.form}>

          {errors.root && <div style={s.errorBanner} role="alert">{errors.root.message}</div>}



          <div style={s.row2}>

            <Field label="Device Name *" error={errors.name?.message}>

              <input style={inp(!!errors.name)} {...register('name', { required: 'Required' })} placeholder="e.g. prod-server-01" />

            </Field>

            <Field label="Hostname / IP Address *" error={errors.ip_address?.message}>

              <input style={inp(!!errors.ip_address)} {...register('ip_address', { required: 'Required' })} placeholder="e.g. 192.168.1.10" />

            </Field>

          </div>



          {/* DETECTION BLOCK */}
          <div style={{ backgroundColor: '#1e293b', padding: '1rem', borderRadius: '8px', border: '1px solid #334155' }}>
            <h3 style={{ margin: '0 0 0.75rem', fontSize: '0.875rem', color: '#e2e8f0', fontWeight: 600 }}>Auto Detect Capabilities</h3>
            {!detectionResult && !manualMode && (
              <>
                <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '0.75rem', marginBottom: '0.75rem' }}>
                  <input style={inp(false)} placeholder="BMC Username (optional)" value={scanUsername} onChange={e => setScanUsername(e.target.value)} />
                  <input type="password" style={inp(false)} placeholder="BMC Password (optional)" value={scanPassword} onChange={e => setScanPassword(e.target.value)} />
                </div>
                <div style={{ display: 'flex', gap: '0.75rem', alignItems: 'center' }}>
                  <button type="button" onClick={handleDetect} disabled={isDetecting} style={{ ...s.submitBtn, backgroundColor: isDetecting ? '#475569' : '#10b981' }}>
                    {isDetecting ? 'Detecting...' : 'Detect Device'}
                  </button>
                  <button type="button" onClick={() => setManualMode(true)} style={{ background: 'none', border: 'none', color: '#94a3b8', fontSize: '0.8rem', cursor: 'pointer', textDecoration: 'underline' }}>
                    Skip detection (Manual Setup)
                  </button>
                </div>
              </>
            )}

            {detectionResult && !manualMode && (
              <div style={{ display: 'flex', flexDirection: 'column', gap: '0.5rem' }}>
                <div style={{ display: 'flex', gap: '1rem', flexWrap: 'wrap' }}>
                  <span style={{ color: '#f8fafc', fontSize: '0.875rem' }}><strong>Vendor:</strong> {detectionResult.vendor || 'Unknown'}</span>
                  <span style={{ color: '#f8fafc', fontSize: '0.875rem' }}><strong>Model:</strong> {detectionResult.model || 'Unknown'}</span>
                  <span style={{ color: '#f8fafc', fontSize: '0.875rem' }}><strong>BMC:</strong> {detectionResult.bmc_type || 'Unknown'}</span>
                </div>
                <div style={{ fontSize: '0.875rem', color: '#cbd5e1' }}>
                  <strong>Supported Protocols:</strong> {detectionResult.supported_protocols?.map(p => <span key={p} style={{ marginRight: '8px', color: '#4ade80' }}>✓ {p.toUpperCase()}</span>)}
                </div>
                
                <div style={{ display: 'flex', gap: '0.75rem', alignItems: 'center', marginTop: '0.5rem' }}>
                  <span style={{ color: '#94a3b8', fontSize: '0.875rem' }}>Select Protocol:</span>
                  <select 
                    style={{ ...inp(false), width: 'auto' }} 
                    value={selectedProtocol} 
                    onChange={e => setSelectedProtocol(e.target.value)}
                  >
                    <option value="">-- select --</option>
                    {detectionResult.supported_protocols?.map(p => (
                      <option key={p} value={p}>{p.toUpperCase()}</option>
                    ))}
                  </select>
                  <button type="button" onClick={() => { setDetectionResult(null); setManualMode(true); }} style={s.cancelBtn}>Manual Setup Config</button>
                </div>
              </div>
            )}
            
            {manualMode && (
              <div style={{ display: 'flex', gap: '0.75rem', alignItems: 'center' }}>
                <span style={{ color: '#f87171', fontSize: '0.875rem', fontWeight: 600 }}>Manual Setup Active</span>
                <button type="button" onClick={() => { setManualMode(false); }} style={{ background: 'none', border: 'none', color: '#94a3b8', fontSize: '0.8rem', cursor: 'pointer', textDecoration: 'underline' }}>
                  Return to Auto-Detect
                </button>
              </div>
            )}
          </div>

          {manualMode && (
          <Field label="Device Type *" error={errors.device_option?.message}>

            <select style={inp(!!errors.device_option)} {...register('device_option', { required: !detectionResult || manualMode ? 'Required' : false })}>

              <option value="">— Select device type —</option>

              {VENDORS.map((vendor) => (

                <optgroup key={vendor} label={`— ${vendor} —`}>

                  {DEVICE_OPTIONS.filter((d) => d.vendor === vendor).map((d) => (

                    <option key={d.value} value={d.value}>{d.label}</option>

                  ))}

                </optgroup>

              ))}

            </select>

          </Field>
          )}

          {manualMode && deviceInfo && (

            <div style={{ display: 'flex', gap: '0.5rem', flexWrap: 'wrap' }}>

              <span style={s.badge}>Vendor: {deviceInfo.vendor}</span>

              <span style={s.badge}>Controller: {deviceInfo.mgmtController}</span>

              <span style={s.badge}>Protocol: {deviceInfo.protocol?.toUpperCase()}</span>

              <span style={s.badge}>Default Port: {deviceInfo.defaultPort}</span>

            </div>

          )}



          <div style={s.row2}>

            <Field label="Location">

              <input style={inp(false)} {...register('location')} placeholder="e.g. Rack A, DC1" />

            </Field>

            <Field label="Description">

              <input style={inp(false)} {...register('description')} placeholder="Optional description" />

            </Field>

          </div>



          {/* ── Redfish ── */}

          {protocol === 'redfish' && (

            <fieldset style={s.fieldset}>

              <legend style={s.legend}>Redfish / BMC Credentials</legend>

              <div style={s.row2}>

                <Field label="BMC IP Address *" error={errors.bmc_ip?.message}>

                  <input style={inp(!!errors.bmc_ip)} {...register('bmc_ip', { required: 'Required' })} placeholder="e.g. 192.168.1.20" />

                </Field>

                <Field label="Port"><input style={inp(false)} {...register('redfish_port')} placeholder="443" /></Field>

              </div>

              <div style={s.row2}>

                <Field label="Scheme">

                  <select style={inp(false)} {...register('redfish_scheme')}>

                    <option value="https">HTTPS</option>

                    <option value="http">HTTP</option>

                  </select>

                </Field>

                <Field label="">

                  <label style={{ ...s.checkLabel, marginTop: '1.5rem' }}>

                    <input type="checkbox" {...register('skip_tls')} style={{ marginRight: '0.4rem' }} />

                    Skip TLS Verify (self-signed certs)

                  </label>

                </Field>

              </div>

              <div style={s.row2}>

                <Field label="BMC Username *" error={errors.bmc_username?.message}>

                  <input style={inp(!!errors.bmc_username)} {...register('bmc_username', { required: 'Required' })} autoComplete="off" />

                </Field>

                <Field label="BMC Password *" error={errors.bmc_password?.message}>

                  <div style={s.pwWrap}>

                    <input type={showBmcPw ? 'text' : 'password'} style={{ ...inp(!!errors.bmc_password), flex: 1 }}

                      {...register('bmc_password', { required: 'Required' })} autoComplete="new-password" />

                    <button type="button" style={s.eyeBtn} onClick={() => setShowBmcPw((v) => !v)} aria-label="Toggle">{showBmcPw ? '🙈' : '👁'}</button>

                  </div>

                </Field>

              </div>

              <div style={s.row3}>

                <Field label="Polling (s)"><input style={inp(false)} {...register('polling_interval')} placeholder="60" /></Field>

                <Field label="Timeout (s)"><input style={inp(false)} {...register('timeout_seconds')} placeholder="30" /></Field>

                <Field label="Retries"><input style={inp(false)} {...register('retry_attempts')} placeholder="3" /></Field>

              </div>

            </fieldset>

          )}



          {/* ── IPMI ── */}

          {protocol === 'ipmi' && (

            <fieldset style={s.fieldset}>

              <legend style={s.legend}>IPMI Credentials</legend>

              <div style={s.row2}>

                <Field label="BMC IP Address *" error={errors.bmc_ip?.message}>

                  <input style={inp(!!errors.bmc_ip)} {...register('bmc_ip', { required: 'Required' })} placeholder="e.g. 192.168.1.20" />

                </Field>

                <Field label="IPMI Port"><input style={inp(false)} {...register('ipmi_port')} placeholder="623" /></Field>

              </div>

              <div style={s.row2}>

                <Field label="BMC Username *" error={errors.bmc_username?.message}>

                  <input style={inp(!!errors.bmc_username)} {...register('bmc_username', { required: 'Required' })} autoComplete="off" />

                </Field>

                <Field label="BMC Password *" error={errors.bmc_password?.message}>

                  <div style={s.pwWrap}>

                    <input type={showBmcPw ? 'text' : 'password'} style={{ ...inp(!!errors.bmc_password), flex: 1 }}

                      {...register('bmc_password', { required: 'Required' })} autoComplete="new-password" />

                    <button type="button" style={s.eyeBtn} onClick={() => setShowBmcPw((v) => !v)} aria-label="Toggle">{showBmcPw ? '🙈' : '👁'}</button>

                  </div>

                </Field>

              </div>

              <div style={s.row3}>

                <Field label="Polling (s)"><input style={inp(false)} {...register('polling_interval')} placeholder="60" /></Field>

                <Field label="Timeout (s)"><input style={inp(false)} {...register('timeout_seconds')} placeholder="30" /></Field>

                <Field label="Retries"><input style={inp(false)} {...register('retry_attempts')} placeholder="3" /></Field>

              </div>

            </fieldset>

          )}



          {/* ── SSH ── */}

          {protocol === 'ssh' && (

            <fieldset style={s.fieldset}>

              <legend style={s.legend}>SSH Credentials</legend>

              <div style={s.row2}>

                <Field label="SSH Port"><input style={inp(false)} {...register('ssh_port')} placeholder="22" /></Field>

                <Field label="Username *" error={errors.ssh_username?.message}>

                  <input style={inp(!!errors.ssh_username)} {...register('ssh_username', { required: 'Required' })} autoComplete="off" />

                </Field>

              </div>

              <Field label="Password">

                <div style={s.pwWrap}>

                  <input type={showSshPw ? 'text' : 'password'} style={{ ...inp(false), flex: 1 }}

                    {...register('ssh_password')} autoComplete="new-password" />

                  <button type="button" style={s.eyeBtn} onClick={() => setShowSshPw((v) => !v)} aria-label="Toggle">{showSshPw ? '🙈' : '👁'}</button>

                </div>

              </Field>

              <Field label="SSH Private Key (optional)">

                <textarea style={{ ...inp(false), minHeight: 80, resize: 'vertical', fontFamily: 'monospace', fontSize: '0.75rem' }}

                  {...register('ssh_key')} placeholder="-----BEGIN OPENSSH PRIVATE KEY-----" />

              </Field>

              <div style={s.row2}>

                <Field label="Polling (s)"><input style={inp(false)} {...register('polling_interval')} placeholder="60" /></Field>

                <Field label="Timeout (s)"><input style={inp(false)} {...register('timeout_seconds')} placeholder="30" /></Field>

              </div>

            </fieldset>

          )}



          {/* ── SNMP ── */}

          {protocol === 'snmp' && (

            <fieldset style={s.fieldset}>

              <legend style={s.legend}>SNMP Configuration</legend>

              <div style={s.row2}>

                <Field label="Port"><input style={inp(false)} {...register('snmp_port')} placeholder="161" /></Field>

                <Field label="SNMP Version">

                  <select style={inp(false)} {...register('snmp_version')}>

                    <option value="v2c">v2c</option>

                    <option value="v3">v3</option>

                  </select>

                </Field>

              </div>

              {snmpVersion === 'v2c' && (

                <Field label="Community String"><input style={inp(false)} {...register('community_string')} placeholder="public" /></Field>

              )}

              {snmpVersion === 'v3' && (

                <>

                  <Field label="SNMP Username *" error={errors.snmp_username?.message}>

                    <input style={inp(!!errors.snmp_username)} {...register('snmp_username', { required: 'Required for v3' })} />

                  </Field>

                  <div style={s.row2}>

                    <Field label="Auth Protocol">

                      <select style={inp(false)} {...register('auth_protocol')}>

                        <option value="MD5">MD5</option><option value="SHA">SHA</option>

                      </select>

                    </Field>

                    <Field label="Auth Password"><input type="password" style={inp(false)} {...register('auth_password')} autoComplete="new-password" /></Field>

                  </div>

                  <div style={s.row2}>

                    <Field label="Privacy Protocol">

                      <select style={inp(false)} {...register('priv_protocol')}>

                        <option value="DES">DES</option><option value="AES">AES</option>

                      </select>

                    </Field>

                    <Field label="Privacy Password"><input type="password" style={inp(false)} {...register('priv_password')} autoComplete="new-password" /></Field>

                  </div>

                </>

              )}

            </fieldset>

          )}



          {/* ── Proxmox ── */}

          {protocol === 'proxmox' && (

            <fieldset style={s.fieldset}>

              <legend style={s.legend}>Proxmox Configuration</legend>

              <div style={s.row2}>

                <Field label="Proxmox IP *" error={errors.proxmox_ip?.message}>

                  <input style={inp(!!errors.proxmox_ip)} {...register('proxmox_ip', { required: 'Required' })} placeholder="e.g. 192.168.1.50" />

                </Field>

                <Field label="Port"><input style={inp(false)} {...register('proxmox_port')} placeholder="8006" /></Field>

              </div>

              <Field label="Node Name *" error={errors.proxmox_node?.message}>

                <input style={inp(!!errors.proxmox_node)} {...register('proxmox_node', { required: 'Required' })} placeholder="e.g. pve" />

              </Field>

              <Field label="API Token ID *" error={errors.proxmox_token_id?.message}>

                <input style={inp(!!errors.proxmox_token_id)} {...register('proxmox_token_id', { required: 'Required' })} placeholder="user@realm!tokenname" />

              </Field>

              <Field label="API Token Secret *" error={errors.proxmox_token_secret?.message}>

                <div style={s.pwWrap}>

                  <input type={showProxmoxSecret ? 'text' : 'password'} style={{ ...inp(!!errors.proxmox_token_secret), flex: 1 }}

                    {...register('proxmox_token_secret', { required: 'Required' })} autoComplete="new-password" />

                  <button type="button" style={s.eyeBtn} onClick={() => setShowProxmoxSecret((v) => !v)} aria-label="Toggle">{showProxmoxSecret ? '🙈' : '👁'}</button>

                </div>

              </Field>

            </fieldset>

          )}



          {/* ── Agent ── */}

          {protocol === 'agent' && (

            <fieldset style={s.fieldset}>

              <legend style={s.legend}>Agent Configuration</legend>

              <div style={s.row2}>

                <Field label="Server IP *" error={errors.agent_ip?.message}>

                  <input style={inp(!!errors.agent_ip)} {...register('agent_ip', { required: 'Required' })} placeholder="e.g. 192.168.1.100" />

                </Field>

                <Field label={`Exporter Port (default: ${defaultAgentPort})`}>

                  <input style={inp(false)} {...register('agent_port')} placeholder={defaultAgentPort} />

                </Field>

              </div>

            </fieldset>

          )}



          <div style={s.actions}>

            <button type="button" onClick={onClose} style={s.cancelBtn} disabled={isSubmitting}>Cancel</button>

            <button type="submit" style={{ ...s.submitBtn, ...(isSubmitting ? s.disabled : {}) }} disabled={isSubmitting}>

              {isSubmitting ? 'Adding...' : 'Add Device'}

            </button>

          </div>

        </form>

      </div>

    </div>

  );

};



const Field: React.FC<{ label: string; error?: string; children: React.ReactNode }> = ({ label, error, children }) => (

  <div style={{ display: 'flex', flexDirection: 'column', gap: '0.25rem' }}>

    <label style={{ fontSize: '0.8125rem', fontWeight: 500, color: '#374151' }}>{label}</label>

    {children}

    {error && <p style={{ color: '#ef4444', fontSize: '0.75rem', margin: 0 }}>{error}</p>}

  </div>

);



const inp = (hasError: boolean): React.CSSProperties => ({

  border: `1px solid ${hasError ? '#ef4444' : '#d1d5db'}`, borderRadius: '4px',

  fontSize: '0.875rem', padding: '0.5rem 0.75rem', width: '100%', boxSizing: 'border-box', backgroundColor: '#fff',

});



const s: Record<string, React.CSSProperties> = {

  overlay: { position: 'fixed', inset: 0, backgroundColor: 'rgba(0,0,0,0.5)', display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 1000, padding: '1rem' },

  modal: { backgroundColor: '#fff', borderRadius: '8px', boxShadow: '0 20px 60px rgba(0,0,0,0.3)', maxWidth: '660px', width: '100%', maxHeight: '90vh', overflowY: 'auto', padding: '1.5rem' },

  header: { display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '1.25rem' },

  title: { fontSize: '1.125rem', fontWeight: 600, color: '#111827', margin: 0 },

  closeBtn: { background: 'none', border: 'none', color: '#6b7280', cursor: 'pointer', fontSize: '1rem', padding: '0.25rem' },

  form: { display: 'flex', flexDirection: 'column', gap: '1rem' },

  row2: { display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '0.75rem' },

  row3: { display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: '0.75rem' },

  fieldset: { border: '1px solid #e5e7eb', borderRadius: '6px', padding: '1rem', display: 'flex', flexDirection: 'column', gap: '0.75rem' },

  legend: { fontSize: '0.8125rem', fontWeight: 600, color: '#374151', padding: '0 0.25rem' },

  checkLabel: { display: 'flex', alignItems: 'center', fontSize: '0.875rem', color: '#374151', cursor: 'pointer' },

  pwWrap: { display: 'flex', alignItems: 'center', gap: '0.5rem' },

  eyeBtn: { background: 'none', border: 'none', cursor: 'pointer', fontSize: '1rem', padding: '0.25rem', flexShrink: 0 },

  actions: { display: 'flex', gap: '0.75rem', justifyContent: 'flex-end', marginTop: '0.5rem' },

  cancelBtn: { backgroundColor: '#f3f4f6', border: '1px solid #d1d5db', borderRadius: '4px', color: '#374151', cursor: 'pointer', fontSize: '0.875rem', fontWeight: 500, padding: '0.5rem 1.25rem' },

  submitBtn: { backgroundColor: '#2563eb', border: 'none', borderRadius: '4px', color: '#fff', cursor: 'pointer', fontSize: '0.875rem', fontWeight: 500, padding: '0.5rem 1.25rem' },

  disabled: { opacity: 0.6, cursor: 'not-allowed' },

  errorBanner: { backgroundColor: '#fee2e2', border: '1px solid #fca5a5', borderRadius: '4px', color: '#b91c1c', fontSize: '0.875rem', padding: '0.75rem 1rem' },

  badge: { background: '#f1f5f9', border: '1px solid #e2e8f0', borderRadius: '4px', color: '#475569', fontSize: '0.7rem', fontWeight: 600, padding: '2px 8px' },

};



export default AddDeviceModal;
