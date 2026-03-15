const fs = require('fs');

const path = 'c:/Projects/infrasense/frontend/src/components/devices/AddDeviceModal.tsx';
let content = fs.readFileSync(path, 'utf8');

// 1. Imports
content = content.replace(
  `import { saveDeviceCredentials } from '../../services/deviceApi';\n\nimport { Device } from '../../types/device';`,
  `import { saveDeviceCredentials, detectProtocol } from '../../services/deviceApi';\nimport { useToast } from '../../contexts/ToastContext';\n\nimport { Device, ProtocolDetectionResult } from '../../types/device';`
);

// 2. AddDeviceModal function setup
content = content.replace(
  `const AddDeviceModal: React.FC<AddDeviceModalProps> = ({ onClose, onSuccess }) => {\n\n  const createDevice = useCreateDevice();`,
  `const AddDeviceModal: React.FC<AddDeviceModalProps> = ({ onClose, onSuccess }) => {\n\n  const createDevice = useCreateDevice();\n  const { showToast } = useToast();\n  const [isDetecting, setIsDetecting] = useState(false);\n  const [detectionResult, setDetectionResult] = useState<ProtocolDetectionResult | null>(null);\n  const [manualMode, setManualMode] = useState(false);\n  const [scanUsername, setScanUsername] = useState('');\n  const [scanPassword, setScanPassword] = useState('');\n  const [selectedProtocol, setSelectedProtocol] = useState('');`
);

// 3. Fix useForm
content = content.replace(
  `const { register, handleSubmit, watch, formState: { errors, isSubmitting }, setError } = useForm<FormValues>`,
  `const { register, handleSubmit, watch, setValue, formState: { errors, isSubmitting }, setError } = useForm<FormValues>`
);

// 4. Update protocol resolution
content = content.replace(
  `const protocol = deviceInfo?.protocol;`,
  `const activeProtocol = (detectionResult && !manualMode) ? selectedProtocol : deviceInfo?.protocol;`
);

// 5. Update onSubmit payload
content = content.replace(
  `const isBMC = protocol === 'redfish' || protocol === 'ipmi';`,
  `const isBMC = activeProtocol === 'redfish' || activeProtocol === 'ipmi';`
);

// Replace "protocol === " with "activeProtocol === " inside onSubmit
content = content.replace(/protocol === /g, 'activeProtocol === ');

// Update createDevice payload inside onSubmit
content = content.replace(
  `        device_type: selectedOption,

        vendor: deviceInfo?.vendor,

        management_controller: deviceInfo?.mgmtController,

        protocol: protocol,`,
  `        device_type: (detectionResult && !manualMode && activeProtocol) ? \`\${detectionResult.vendor || 'auto'}_\${activeProtocol}\`.toLowerCase() : selectedOption,

        vendor: (detectionResult && !manualMode) ? detectionResult.vendor : deviceInfo?.vendor,

        management_controller: (detectionResult && !manualMode) ? detectionResult.bmc_type : deviceInfo?.mgmtController,

        protocol: activeProtocol,`
);

// Update error check for device_option
content = content.replace(
  `    if (!selectedOption) { setError('device_option', { message: 'Please select a device type' }); return; }`,
  `    if (manualMode && !selectedOption) { setError('device_option', { message: 'Please select a device type' }); return; }
    if (!manualMode && detectionResult && !selectedProtocol) { setError('root', { message: 'Please select a protocol' }); return; }`
);

// 6. Inject handleDetect
const handleDetectStr = `
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
        // Pre-fill the configured ip/credentials to the redfish/ipmi target form fields
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
`;
content = content.replace(
  `const onSubmit = async (data: FormValues) => {`,
  `${handleDetectStr}\n  const onSubmit = async (data: FormValues) => {`
);

// 7. Inject Detection UI block right after IP Field
const detectionUIMarkup = `
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
`;

content = content.replace(
  `          <Field label="Device Type *" error={errors.device_option?.message}>`,
  `          ${detectionUIMarkup}\n\n          {manualMode && (\n            <Field label="Device Type *" error={errors.device_option?.message}>`
);

content = content.replace(
  `              {DEVICE_OPTIONS.filter((d) => d.vendor === vendor).map((d) => (`,
  `              {DEVICE_OPTIONS.filter((d) => d.vendor === vendor).map((d) => (`
);

// We need to close the {manualMode && ( block just before <div style={s.row2}> location setup
content = content.replace(
  `          <div style={s.row2}>\n\n            <Field label="Location">`,
  `          )}\n\n          <div style={s.row2}>\n\n            <Field label="Location">`
);

// We also have the deviceInfo rendering badge:
content = content.replace(
  `          {deviceInfo && (`,
  `          {manualMode && deviceInfo && (`
);

// In the JSX, we replaced `protocol === ` with `activeProtocol === `. Since JSX has {protocol === 'redfish' && ..., we need to make sure this works.
// We replaced it globally inside onSubmit, but what about the JSX?
content = content.replace(
  `{protocol === 'redfish' && (`,
  `{activeProtocol === 'redfish' && (`
);
content = content.replace(
  `{protocol === 'ipmi' && (`,
  `{activeProtocol === 'ipmi' && (`
);
content = content.replace(
  `{protocol === 'ssh' && (`,
  `{activeProtocol === 'ssh' && (`
);
content = content.replace(
  `{protocol === 'snmp' && (`,
  `{activeProtocol === 'snmp' && (`
);
content = content.replace(
  `{protocol === 'proxmox' && (`,
  `{activeProtocol === 'proxmox' && (`
);
content = content.replace(
  `{protocol === 'agent' && (`,
  `{activeProtocol === 'agent' && (`
);

fs.writeFileSync(path, content, 'utf8');
console.log('Update Complete');
