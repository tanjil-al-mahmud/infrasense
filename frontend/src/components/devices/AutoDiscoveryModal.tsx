import React, { useState } from 'react';
import { useForm } from 'react-hook-form';
import { useQueryClient } from '@tanstack/react-query';
import { deviceKeys } from '../../hooks/useDevices';
import { useToast } from '../../contexts/ToastContext';

// Assuming an HTTP client exists in services. If not, this is a placeholder for the actual API call.
const discoverDevices = async (payload: any) => {
  const token = localStorage.getItem('token');
  const response = await fetch('/api/v1/devices/discover', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${token}`
    },
    body: JSON.stringify(payload)
  });
  if (!response.ok) {
    const err = await response.json();
    throw new Error(err.error || 'Failed to discover devices');
  }
  return response.json();
};

const addDiscoveredDevices = async (payload: any) => {
  const token = localStorage.getItem('token');
  const response = await fetch('/api/v1/devices/bulk', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${token}`
    },
    body: JSON.stringify(payload)
  });
  if (!response.ok) {
    const err = await response.json();
    throw new Error(err.error || 'Failed to add devices');
  }
  return response.json();
};


interface DiscoveryResult {
  ip_address: string;
  hostname: string;
  vendor: string;
  model: string;
  protocol: string;
  ports_open: number[];
  status: string; // 'discovered', 'needs_credentials', 'error'
  error_message?: string;
}

interface FormValues {
  cidr: string;
  username?: string;
  password?: string;
}

interface AutoDiscoveryModalProps {
  onClose: () => void;
  onSuccess: (addedCount: number) => void;
}

const AutoDiscoveryModal: React.FC<AutoDiscoveryModalProps> = ({ onClose, onSuccess }) => {
  const { showToast } = useToast();
  const queryClient = useQueryClient();
  const [results, setResults] = useState<DiscoveryResult[]>([]);
  const [selectedIPs, setSelectedIPs] = useState<Set<string>>(new Set());
  const [isScanning, setIsScanning] = useState(false);
  const [showPassword, setShowPassword] = useState(false);

  const { register, handleSubmit, watch, formState: { errors } } = useForm<FormValues>({
    defaultValues: {
      cidr: '',
      username: '',
      password: '',
    }
  });

  const formData = watch();

  const handleScan = async (data: FormValues) => {
    setIsScanning(true);
    setResults([]);
    setSelectedIPs(new Set());
    try {
      const res = await discoverDevices({
        target_cidr: data.cidr,
        credentials: {
          username: data.username,
          password: data.password
        }
      });
      setResults(res.results || []);
      
      // Auto-select discovered hosts that don't have errors
      const validIPs = (res.results || [])
        .filter((r: DiscoveryResult) => r.status === 'discovered' || r.status === 'needs_credentials')
        .map((r: DiscoveryResult) => r.ip_address);
      setSelectedIPs(new Set(validIPs));

      if (res.results?.length > 0) {
        showToast(`Found ${res.results.length} device(s)`, 'success');
      } else {
        // Use 'info' or 'error' instead of 'warning'
        showToast('No BMC devices found in this range.', 'info');
      }
    } catch (err: any) {
      showToast(err.message || 'Scanning failed', 'error');
    } finally {
      setIsScanning(false);
    }
  };

  const handleAddSelected = async () => {
    if (selectedIPs.size === 0) return;
    
    const devicesToAdd = results
      .filter(r => selectedIPs.has(r.ip_address))
      .map(r => ({
         ip_address: r.ip_address,
         hostname: r.hostname || r.ip_address,
         protocol: r.protocol,
         vendor: r.vendor,
         username: formData.username,
         password: formData.password
      }));

    try {
      await addDiscoveredDevices({ devices: devicesToAdd });
      queryClient.invalidateQueries({ queryKey: deviceKeys.lists() });
      onSuccess(devicesToAdd.length);
    } catch(err: any) {
      showToast(err.message || 'Failed to add devices', 'error');
    }
  };

  const toggleSelection = (ip: string) => {
    const newSet = new Set(selectedIPs);
    if (newSet.has(ip)) newSet.delete(ip);
    else newSet.add(ip);
    setSelectedIPs(newSet);
  };

  return (
    <div style={s.overlay} role="dialog" aria-modal="true">
      <div style={s.modal}>
        <div style={s.header}>
          <h2 style={s.title}>🔍 Auto Discover BMCs</h2>
          <button onClick={onClose} style={s.closeBtn}>✕</button>
        </div>

        <div style={s.body}>
          <form onSubmit={handleSubmit(handleScan)} style={s.form}>
            <div style={s.row2}>
              <div style={s.fieldWrapper}>
                <label style={s.label}>IP Range (CIDR) *</label>
                <input 
                  style={inp(!!errors.cidr)} 
                  {...register('cidr', { required: 'CIDR is required', pattern: { value: /^(?:\d{1,3}\.){3}\d{1,3}(?:\/\d{1,2})?$/, message: 'Invalid IP or CIDR' } })} 
                  placeholder="e.g. 192.168.1.0/24 or 10.0.0.50" 
                />
                {errors.cidr && <span style={s.errorText}>{errors.cidr.message}</span>}
              </div>
            </div>

            <p style={s.hintText}>Optional: Provide credentials to auto-detect Vendor/Model and classify protocols.</p>

            <div style={s.row2}>
              <div style={s.fieldWrapper}>
                <label style={s.label}>Generic Username</label>
                <input style={inp(false)} {...register('username')} placeholder="e.g. root" />
              </div>
              <div style={s.fieldWrapper}>
                <label style={s.label}>Generic Password</label>
                <div style={s.pwWrap}>
                  <input 
                    type={showPassword ? 'text' : 'password'} 
                    style={{...inp(false), flex: 1}} 
                    {...register('password')} 
                  />
                  <button type="button" style={s.eyeBtn} onClick={() => setShowPassword(!showPassword)}>
                    {showPassword ? '🙈' : '👁'}
                  </button>
                </div>
              </div>
            </div>

            <div style={s.scanBtnWrapper}>
              <button type="submit" style={s.scanBtn} disabled={isScanning}>
                {isScanning ? 'Scanning Network...' : 'Start Scan'}
              </button>
            </div>
          </form>

          {isScanning && (
            <div style={s.loadingBox}>
              <div style={s.spinner} />
              <p>Scanning IPs and probing BMC ports (443, 623, 161, 22)...</p>
            </div>
          )}

          {!isScanning && results.length > 0 && (
            <div style={s.resultsArea}>
              <h3 style={s.resultsTitle}>Discovered Devices ({results.length})</h3>
              <div style={s.tableWrapper}>
                <table style={s.table}>
                  <thead>
                    <tr>
                      <th style={{...s.th, width: '40px'}}>
                        <input 
                          type="checkbox" 
                          checked={selectedIPs.size === results.filter(r => r.status !== 'error').length && results.length > 0}
                          onChange={(e) => {
                            if (e.target.checked) setInstalledIPsToValid(results);
                            else setSelectedIPs(new Set());
                          }}
                        />
                      </th>
                      <th style={s.th}>IP Address</th>
                      <th style={s.th}>Protocol</th>
                      <th style={s.th}>Vendor</th>
                      <th style={s.th}>Status</th>
                    </tr>
                  </thead>
                  <tbody>
                    {results.map((r, i) => (
                      <tr key={i} style={s.tr}>
                        <td style={s.td}>
                          <input 
                            type="checkbox" 
                            disabled={r.status === 'error'}
                            checked={selectedIPs.has(r.ip_address)} 
                            onChange={() => toggleSelection(r.ip_address)} 
                          />
                        </td>
                        <td style={s.td}>
                          <strong>{r.ip_address}</strong>
                          {r.hostname && r.hostname !== r.ip_address && <div style={{fontSize: '0.75rem', color: '#64748b'}}>{r.hostname}</div>}
                        </td>
                        <td style={s.td}>
                          <span style={s.badge}>{r.protocol.toUpperCase()}</span>
                        </td>
                        <td style={s.td}>
                          {r.vendor || 'Unknown'} {r.model ? `(${r.model})` : ''}
                        </td>
                        <td style={s.td}>
                          {r.status === 'discovered' && <span style={{color: '#4ade80'}}>Ready to Add</span>}
                          {r.status === 'needs_credentials' && <span style={{color: '#fbbf24'}}>Needs Credentials</span>}
                          {r.status === 'error' && <span style={{color: '#f87171'}} title={r.error_message}>Access Denied / Error</span>}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          )}
        </div>

        <div style={s.footer}>
          <button onClick={onClose} style={s.cancelBtn}>Cancel</button>
          <button 
            onClick={handleAddSelected} 
            disabled={selectedIPs.size === 0 || isScanning} 
            style={{...s.submitBtn, ...(selectedIPs.size === 0 || isScanning ? { opacity: 0.5, cursor: 'not-allowed' } : {})}}
          >
            Add Selected Devices ({selectedIPs.size})
          </button>
        </div>
      </div>
    </div>
  );

  function setInstalledIPsToValid(res: DiscoveryResult[]) {
    const valid = res.filter(r => r.status !== 'error').map(r => r.ip_address);
    setSelectedIPs(new Set(valid));
  }
};

const inp = (err: boolean): React.CSSProperties => ({
  width: '100%', padding: '0.6rem 0.75rem', fontSize: '0.875rem', borderRadius: '6px',
  border: `1px solid ${err ? '#ef4444' : '#334155'}`, backgroundColor: '#1e293b', color: '#f1f5f9',
  boxSizing: 'border-box'
});

const s: Record<string, React.CSSProperties> = {
  overlay: { position: 'fixed', inset: 0, backgroundColor: 'rgba(0,0,0,0.7)', display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 1000, padding: '1rem' },
  modal: { backgroundColor: '#0f172a', border: '1px solid #1e293b', borderRadius: '12px', boxShadow: '0 25px 60px rgba(0,0,0,0.5)', width: '100%', maxWidth: '800px', maxHeight: '90vh', display: 'flex', flexDirection: 'column' },
  header: { display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '1.25rem 1.5rem', borderBottom: '1px solid #1e293b' },
  title: { margin: 0, fontSize: '1.25rem', fontWeight: 600, color: '#f1f5f9' },
  closeBtn: { background: 'none', border: 'none', color: '#64748b', fontSize: '1.25rem', cursor: 'pointer', padding: '0.25rem' },
  body: { padding: '1.5rem', overflowY: 'auto' },
  form: { display: 'flex', flexDirection: 'column', gap: '1rem' },
  row2: { display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '1rem' },
  fieldWrapper: { display: 'flex', flexDirection: 'column', gap: '0.4rem' },
  label: { fontSize: '0.875rem', fontWeight: 500, color: '#cbd5e1' },
  pwWrap: { display: 'flex', alignItems: 'center', backgroundColor: '#1e293b', border: '1px solid #334155', borderRadius: '6px', paddingRight: '0.5rem' },
  eyeBtn: { background: 'none', border: 'none', color: '#94a3b8', cursor: 'pointer', display: 'flex', alignItems: 'center' },
  hintText: { fontSize: '0.8rem', color: '#94a3b8', margin: '0 0 0.5rem 0' },
  errorText: { fontSize: '0.75rem', color: '#ef4444', marginTop: '0.2rem' },
  scanBtnWrapper: { display: 'flex', justifyContent: 'flex-start', marginTop: '1rem' },
  scanBtn: { backgroundColor: '#2563eb', color: '#fff', border: 'none', borderRadius: '6px', padding: '0.75rem 1.5rem', fontSize: '0.875rem', fontWeight: 600, cursor: 'pointer' },
  loadingBox: { marginTop: '2rem', padding: '2rem', textAlign: 'center', color: '#94a3b8', backgroundColor: '#1e293b', borderRadius: '8px' },
  spinner: { width: '24px', height: '24px', border: '3px solid #334155', borderTopColor: '#3b82f6', borderRadius: '50%', animation: 'spin 1s linear infinite', margin: '0 auto 1rem' },
  resultsArea: { marginTop: '2rem' },
  resultsTitle: { fontSize: '1.1rem', color: '#e2e8f0', margin: '0 0 1rem 0' },
  tableWrapper: { border: '1px solid #1e293b', borderRadius: '8px', overflow: 'hidden' },
  table: { width: '100%', borderCollapse: 'collapse', fontSize: '0.875rem', textAlign: 'left' },
  th: { backgroundColor: '#1e293b', padding: '0.75rem' },
  tr: { borderBottom: '1px solid #1e293b' },
  td: { padding: '0.75rem', color: '#e2e8f0' },
  badge: { backgroundColor: '#334155', padding: '0.2rem 0.5rem', borderRadius: '4px', fontSize: '0.75rem' },
  footer: { display: 'flex', justifyContent: 'flex-end', gap: '0.75rem', padding: '1.25rem 1.5rem', borderTop: '1px solid #1e293b', backgroundColor: '#0f172a', borderBottomLeftRadius: '12px', borderBottomRightRadius: '12px' },
  cancelBtn: { backgroundColor: 'transparent', border: '1px solid #334155', borderRadius: '6px', color: '#cbd5e1', cursor: 'pointer', padding: '0.5rem 1rem', fontSize: '0.875rem', fontWeight: 500 },
  submitBtn: { backgroundColor: '#10b981', border: 'none', borderRadius: '6px', color: '#fff', cursor: 'pointer', padding: '0.5rem 1rem', fontSize: '0.875rem', fontWeight: 600 },
};

export default AutoDiscoveryModal;
