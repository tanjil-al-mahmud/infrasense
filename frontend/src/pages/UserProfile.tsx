import React, { useState, useEffect } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useAuth } from '../contexts/AuthContext';
import { useToast } from '../contexts/ToastContext';
import { getMe, updateMe, changeOwnPassword } from '../services/userApi';

const ROLE_LABELS: Record<string, string> = {
  admin: 'Administrator',
  operator: 'Operator',
  viewer: 'Read Only',
};

const UserProfile: React.FC = () => {
  const { refreshUser } = useAuth();
  const { showToast } = useToast();
  const queryClient = useQueryClient();

  const { data: me, isLoading } = useQuery({ queryKey: ['me'], queryFn: getMe });

  // Profile edit state
  const [fullName, setFullName] = useState('');
  const [email, setEmail] = useState('');
  const [profileErrors, setProfileErrors] = useState<{ full_name?: string; email?: string }>({});

  // Password change state
  const [currentPassword, setCurrentPassword] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [pwErrors, setPwErrors] = useState<{ current?: string; new?: string; confirm?: string }>({});

  useEffect(() => {
    if (me) {
      setFullName(me.full_name ?? '');
      setEmail(me.email ?? '');
    }
  }, [me]);

  const profileMutation = useMutation({
    mutationFn: () => updateMe({ full_name: fullName, email: email || undefined }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['me'] });
      refreshUser();
      showToast('Profile updated successfully', 'success');
    },
    onError: (err) => showToast(err instanceof Error ? err.message : 'Failed to update profile', 'error'),
  });

  const passwordMutation = useMutation({
    mutationFn: () => changeOwnPassword({ current_password: currentPassword, new_password: newPassword }),
    onSuccess: () => {
      setCurrentPassword('');
      setNewPassword('');
      setConfirmPassword('');
      showToast('Password changed successfully', 'success');
    },
    onError: (err) => showToast(err instanceof Error ? err.message : 'Failed to change password', 'error'),
  });

  const handleProfileSave = () => {
    const errs: typeof profileErrors = {};
    if (!fullName.trim()) errs.full_name = 'Full name is required';
    if (email && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) errs.email = 'Enter a valid email address';
    setProfileErrors(errs);
    if (Object.keys(errs).length === 0) profileMutation.mutate();
  };

  const handlePasswordSave = () => {
    const errs: typeof pwErrors = {};
    if (!currentPassword) errs.current = 'Current password is required';
    if (!newPassword) errs.new = 'New password is required';
    else if (newPassword.length < 8) errs.new = 'Password must be at least 8 characters';
    if (confirmPassword !== newPassword) errs.confirm = 'Passwords do not match';
    setPwErrors(errs);
    if (Object.keys(errs).length === 0) passwordMutation.mutate();
  };

  if (isLoading) {
    return (
      <div style={{ padding: '1.5rem', color: '#64748b' }}>Loading profile...</div>
    );
  }

  return (
    <div style={styles.page}>
      <h1 style={styles.pageTitle}>My Profile</h1>

      {/* Info card */}
      <div style={styles.card}>
        <h2 style={styles.sectionTitle}>Account Information</h2>
        <div style={styles.infoGrid}>
          <div style={styles.infoItem}><span style={styles.infoLabel}>Username</span><span style={styles.infoValue}>{me?.username}</span></div>
          <div style={styles.infoItem}><span style={styles.infoLabel}>Role</span><span style={styles.infoValue}>{ROLE_LABELS[me?.role ?? ''] ?? me?.role}</span></div>
          <div style={styles.infoItem}><span style={styles.infoLabel}>Status</span><span style={{ ...styles.infoValue, color: me?.enabled ? '#22c55e' : '#64748b' }}>{me?.enabled ? 'Active' : 'Disabled'}</span></div>
          <div style={styles.infoItem}><span style={styles.infoLabel}>Last Login</span><span style={styles.infoValue}>{me?.last_login_at ? new Date(me.last_login_at).toLocaleString() : '—'}</span></div>
        </div>
      </div>

      {/* Edit profile */}
      <div style={styles.card}>
        <h2 style={styles.sectionTitle}>Edit Profile</h2>
        <div style={styles.fieldGroup}>
          <label style={styles.label}>Full Name</label>
          <input value={fullName} onChange={(e) => setFullName(e.target.value)} style={{ ...styles.input, ...(profileErrors.full_name ? styles.inputError : {}) }} />
          {profileErrors.full_name && <span style={styles.fieldError}>{profileErrors.full_name}</span>}
        </div>
        <div style={styles.fieldGroup}>
          <label style={styles.label}>Email</label>
          <input type="email" value={email} onChange={(e) => setEmail(e.target.value)} style={{ ...styles.input, ...(profileErrors.email ? styles.inputError : {}) }} />
          {profileErrors.email && <span style={styles.fieldError}>{profileErrors.email}</span>}
        </div>
        <button
          onClick={handleProfileSave}
          style={{ ...styles.saveBtn, ...(profileMutation.isPending ? { opacity: 0.6, cursor: 'not-allowed' } : {}) }}
          disabled={profileMutation.isPending}
        >
          {profileMutation.isPending ? 'Saving...' : 'Save Changes'}
        </button>
      </div>

      {/* Change password */}
      <div style={styles.card}>
        <h2 style={styles.sectionTitle}>Change Password</h2>
        <div style={styles.fieldGroup}>
          <label style={styles.label}>Current Password</label>
          <input type="password" value={currentPassword} onChange={(e) => setCurrentPassword(e.target.value)} style={{ ...styles.input, ...(pwErrors.current ? styles.inputError : {}) }} />
          {pwErrors.current && <span style={styles.fieldError}>{pwErrors.current}</span>}
        </div>
        <div style={styles.fieldGroup}>
          <label style={styles.label}>New Password</label>
          <input type="password" value={newPassword} onChange={(e) => setNewPassword(e.target.value)} style={{ ...styles.input, ...(pwErrors.new ? styles.inputError : {}) }} />
          {pwErrors.new && <span style={styles.fieldError}>{pwErrors.new}</span>}
        </div>
        <div style={styles.fieldGroup}>
          <label style={styles.label}>Confirm New Password</label>
          <input type="password" value={confirmPassword} onChange={(e) => setConfirmPassword(e.target.value)} style={{ ...styles.input, ...(pwErrors.confirm ? styles.inputError : {}) }} />
          {pwErrors.confirm && <span style={styles.fieldError}>{pwErrors.confirm}</span>}
        </div>
        <button
          onClick={handlePasswordSave}
          style={{ ...styles.saveBtn, ...(passwordMutation.isPending ? { opacity: 0.6, cursor: 'not-allowed' } : {}) }}
          disabled={passwordMutation.isPending}
        >
          {passwordMutation.isPending ? 'Saving...' : 'Change Password'}
        </button>
      </div>
    </div>
  );
};

const styles: Record<string, React.CSSProperties> = {
  page: { padding: '1.5rem', maxWidth: '720px', margin: '0 auto' },
  pageTitle: { fontSize: '1.5rem', fontWeight: 700, color: '#f1f5f9', margin: '0 0 1.5rem' },
  card: { backgroundColor: '#0f172a', border: '1px solid #1e293b', borderRadius: '10px', padding: '1.5rem', marginBottom: '1.25rem' },
  sectionTitle: { fontSize: '1rem', fontWeight: 600, color: '#e2e8f0', margin: '0 0 1rem' },
  infoGrid: { display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(200px, 1fr))', gap: '0.75rem' },
  infoItem: { display: 'flex', flexDirection: 'column', gap: '0.2rem' },
  infoLabel: { fontSize: '0.75rem', color: '#64748b', fontWeight: 600, textTransform: 'uppercase', letterSpacing: '0.05em' },
  infoValue: { fontSize: '0.9rem', color: '#e2e8f0', fontWeight: 500 },
  fieldGroup: { display: 'flex', flexDirection: 'column', gap: '0.25rem', marginBottom: '0.875rem' },
  label: { fontSize: '0.8125rem', fontWeight: 500, color: '#94a3b8' },
  input: { border: '1px solid #334155', borderRadius: '6px', fontSize: '0.875rem', padding: '0.5rem 0.75rem', background: '#1e293b', color: '#e2e8f0', outline: 'none', width: '100%', boxSizing: 'border-box' },
  inputError: { borderColor: '#dc2626' },
  fieldError: { color: '#f87171', fontSize: '0.75rem' },
  saveBtn: { backgroundColor: '#2563eb', border: 'none', borderRadius: '6px', color: '#fff', cursor: 'pointer', fontSize: '0.875rem', fontWeight: 600, padding: '0.5rem 1.25rem' },
};

export default UserProfile;
