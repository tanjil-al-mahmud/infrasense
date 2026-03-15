import React, { useState, useMemo } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useAuth } from '../contexts/AuthContext';
import { useToast } from '../contexts/ToastContext';
import {
  listUsers, createUser, updateUser, deleteUser, changePassword,
  UserRecord, CreateUserData, UpdateUserData,
} from '../services/userApi';

// ── Types ────────────────────────────────────────────────────────────────────

type Role = 'admin' | 'operator' | 'viewer';

interface UserFormValues {
  full_name: string;
  username: string;
  email: string;
  role: Role;
  password: string;
  confirmPassword: string;
  enabled: boolean;
}

interface ChangePasswordFormValues {
  newPassword: string;
  confirmPassword: string;
}

// ── Helpers ──────────────────────────────────────────────────────────────────

const ROLE_LABELS: Record<Role, string> = {
  admin: 'Administrator',
  operator: 'Operator',
  viewer: 'Read Only',
};

const ROLE_COLORS: Record<Role, { bg: string; text: string }> = {
  admin:    { bg: '#450a0a', text: '#dc2626' },
  operator: { bg: '#1e3a5f', text: '#3b82f6' },
  viewer:   { bg: '#052e16', text: '#22c55e' },
};

function validateUserForm(values: UserFormValues, isEdit: boolean): Partial<Record<keyof UserFormValues, string>> {
  const errs: Partial<Record<keyof UserFormValues, string>> = {};
  if (!values.full_name.trim()) errs.full_name = 'Full name is required';
  if (!isEdit) {
    if (!values.username.trim()) errs.username = 'Username is required';
    else if (!/^[a-zA-Z0-9_]{3,50}$/.test(values.username))
      errs.username = 'Username must be 3-50 alphanumeric characters or underscores';
  }
  if (values.email && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(values.email))
    errs.email = 'Enter a valid email address';
  if (!isEdit && !values.password) errs.password = 'Password is required for new users';
  if (values.password && values.password.length < 8) errs.password = 'Password must be at least 8 characters';
  if (values.password && values.confirmPassword !== values.password) errs.confirmPassword = 'Passwords do not match';
  return errs;
}

function validateChangePasswordForm(values: ChangePasswordFormValues): Partial<Record<keyof ChangePasswordFormValues, string>> {
  const errs: Partial<Record<keyof ChangePasswordFormValues, string>> = {};
  if (!values.newPassword) errs.newPassword = 'New password is required';
  else if (values.newPassword.length < 8) errs.newPassword = 'Password must be at least 8 characters';
  if (values.confirmPassword !== values.newPassword) errs.confirmPassword = 'Passwords do not match';
  return errs;
}

// ── Change Password Modal ────────────────────────────────────────────────────

interface ChangePasswordModalProps {
  user: UserRecord;
  onClose: () => void;
}

const ChangePasswordModal: React.FC<ChangePasswordModalProps> = ({ user, onClose }) => {
  const { showToast } = useToast();
  const [values, setValues] = useState<ChangePasswordFormValues>({ newPassword: '', confirmPassword: '' });
  const [errors, setErrors] = useState<Partial<Record<keyof ChangePasswordFormValues, string>>>({});
  const [saving, setSaving] = useState(false);

  const handleSave = async () => {
    const errs = validateChangePasswordForm(values);
    setErrors(errs);
    if (Object.keys(errs).length > 0) return;
    setSaving(true);
    try {
      await changePassword(user.id, { new_password: values.newPassword });
      showToast('Password changed successfully', 'success');
      onClose();
    } catch (err) {
      showToast(err instanceof Error ? err.message : 'Failed to change password', 'error');
    } finally {
      setSaving(false);
    }
  };

  return (
    <div style={styles.overlay} role="dialog" aria-modal="true" aria-label="Change Password">
      <div style={styles.modal}>
        <h2 style={styles.modalTitle}>Change Password — {user.username}</h2>
        <div style={styles.fieldGroup}>
          <label style={styles.label}>New Password</label>
          <input
            type="password"
            value={values.newPassword}
            onChange={(e) => setValues((v) => ({ ...v, newPassword: e.target.value }))}
            style={{ ...styles.input, ...(errors.newPassword ? styles.inputError : {}) }}
          />
          {errors.newPassword && <span style={styles.fieldError}>{errors.newPassword}</span>}
        </div>
        <div style={styles.fieldGroup}>
          <label style={styles.label}>Confirm Password</label>
          <input
            type="password"
            value={values.confirmPassword}
            onChange={(e) => setValues((v) => ({ ...v, confirmPassword: e.target.value }))}
            style={{ ...styles.input, ...(errors.confirmPassword ? styles.inputError : {}) }}
          />
          {errors.confirmPassword && <span style={styles.fieldError}>{errors.confirmPassword}</span>}
        </div>
        <div style={styles.modalActions}>
          <button onClick={onClose} style={styles.cancelBtn} disabled={saving}>Cancel</button>
          <button onClick={handleSave} style={{ ...styles.saveBtn, ...(saving ? { opacity: 0.6, cursor: 'not-allowed' } : {}) }} disabled={saving}>
            {saving ? 'Saving...' : 'Save'}
          </button>
        </div>
      </div>
    </div>
  );
};

// ── User Modal (Add / Edit) ──────────────────────────────────────────────────

interface UserModalProps {
  editUser?: UserRecord;
  currentUserId: string;
  onClose: () => void;
  onSaved: () => void;
}

const EMPTY_FORM: UserFormValues = {
  full_name: '', username: '', email: '', role: 'viewer', password: '', confirmPassword: '', enabled: true,
};

const UserModal: React.FC<UserModalProps> = ({ editUser, currentUserId, onClose, onSaved }) => {
  const { showToast } = useToast();
  const isEdit = !!editUser;
  const [values, setValues] = useState<UserFormValues>(
    editUser
      ? { full_name: editUser.full_name ?? '', username: editUser.username, email: editUser.email ?? '', role: editUser.role, password: '', confirmPassword: '', enabled: editUser.enabled }
      : EMPTY_FORM
  );
  const [errors, setErrors] = useState<Partial<Record<keyof UserFormValues, string>>>({});
  const [saving, setSaving] = useState(false);

  const set = (field: keyof UserFormValues, value: string | boolean) =>
    setValues((v) => ({ ...v, [field]: value }));

  const handleSave = async () => {
    const errs = validateUserForm(values, isEdit);
    setErrors(errs);
    if (Object.keys(errs).length > 0) return;
    setSaving(true);
    try {
      if (isEdit && editUser) {
        const data: UpdateUserData = { full_name: values.full_name, email: values.email || undefined, enabled: values.enabled };
        if (editUser.id !== currentUserId) data.role = values.role;
        await updateUser(editUser.id, data);
        if (values.password) await changePassword(editUser.id, { new_password: values.password });
        showToast('User updated successfully', 'success');
      } else {
        const data: CreateUserData = { username: values.username, full_name: values.full_name, email: values.email || undefined, role: values.role, password: values.password, enabled: values.enabled };
        await createUser(data);
        showToast('User created successfully', 'success');
      }
      onSaved();
    } catch (err) {
      showToast(err instanceof Error ? err.message : 'Failed to save user', 'error');
    } finally {
      setSaving(false);
    }
  };

  const isSelf = isEdit && editUser?.id === currentUserId;

  return (
    <div style={styles.overlay} role="dialog" aria-modal="true" aria-label={isEdit ? 'Edit User' : 'Add User'}>
      <div style={{ ...styles.modal, maxWidth: '520px' }}>
        <h2 style={styles.modalTitle}>{isEdit ? 'Edit User' : 'Add User'}</h2>
        <div style={styles.fieldGroup}>
          <label style={styles.label}>Full Name</label>
          <input value={values.full_name} onChange={(e) => set('full_name', e.target.value)} style={{ ...styles.input, ...(errors.full_name ? styles.inputError : {}) }} />
          {errors.full_name && <span style={styles.fieldError}>{errors.full_name}</span>}
        </div>
        <div style={styles.fieldGroup}>
          <label style={styles.label}>Username</label>
          <input value={values.username} onChange={(e) => set('username', e.target.value)} disabled={isEdit} style={{ ...styles.input, ...(errors.username ? styles.inputError : {}), ...(isEdit ? { opacity: 0.5 } : {}) }} />
          {errors.username && <span style={styles.fieldError}>{errors.username}</span>}
        </div>
        <div style={styles.fieldGroup}>
          <label style={styles.label}>Email</label>
          <input type="email" value={values.email} onChange={(e) => set('email', e.target.value)} style={{ ...styles.input, ...(errors.email ? styles.inputError : {}) }} />
          {errors.email && <span style={styles.fieldError}>{errors.email}</span>}
        </div>
        <div style={styles.fieldGroup}>
          <label style={styles.label}>Role</label>
          <select value={values.role} onChange={(e) => set('role', e.target.value)} disabled={isSelf} style={{ ...styles.input, ...(isSelf ? { opacity: 0.5 } : {}) }}>
            <option value="admin">Administrator</option>
            <option value="operator">Operator</option>
            <option value="viewer">Read Only</option>
          </select>
          {isSelf && <span style={{ fontSize: '0.75rem', color: '#64748b' }}>Cannot change your own role</span>}
        </div>
        <div style={styles.fieldGroup}>
          <label style={styles.label}>{isEdit ? 'New Password (leave blank to keep current)' : 'Password'}</label>
          <input type="password" value={values.password} onChange={(e) => set('password', e.target.value)} style={{ ...styles.input, ...(errors.password ? styles.inputError : {}) }} />
          {errors.password && <span style={styles.fieldError}>{errors.password}</span>}
        </div>
        <div style={styles.fieldGroup}>
          <label style={styles.label}>Confirm Password</label>
          <input type="password" value={values.confirmPassword} onChange={(e) => set('confirmPassword', e.target.value)} style={{ ...styles.input, ...(errors.confirmPassword ? styles.inputError : {}) }} />
          {errors.confirmPassword && <span style={styles.fieldError}>{errors.confirmPassword}</span>}
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', marginBottom: '1rem' }}>
          <input type="checkbox" id="enabled" checked={values.enabled} onChange={(e) => set('enabled', e.target.checked)} style={{ width: 16, height: 16, cursor: 'pointer' }} />
          <label htmlFor="enabled" style={{ ...styles.label, margin: 0, cursor: 'pointer' }}>Enabled</label>
        </div>
        <div style={styles.modalActions}>
          <button onClick={onClose} style={styles.cancelBtn} disabled={saving}>Cancel</button>
          <button onClick={handleSave} style={{ ...styles.saveBtn, ...(saving ? { opacity: 0.6, cursor: 'not-allowed' } : {}) }} disabled={saving}>
            {saving ? 'Saving...' : 'Save'}
          </button>
        </div>
      </div>
    </div>
  );
};

// ── UserManagement Page ──────────────────────────────────────────────────────

const UserManagement: React.FC = () => {
  const { user: currentUser } = useAuth();
  const { showToast } = useToast();
  const queryClient = useQueryClient();

  const [search, setSearch] = useState('');
  const [roleFilter, setRoleFilter] = useState<Role | ''>('');
  const [showAddModal, setShowAddModal] = useState(false);
  const [editTarget, setEditTarget] = useState<UserRecord | null>(null);
  const [changePasswordTarget, setChangePasswordTarget] = useState<UserRecord | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<UserRecord | null>(null);

  const { data: users = [], isLoading, isError, error } = useQuery({
    queryKey: ['users'],
    queryFn: listUsers,
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => deleteUser(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['users'] });
      showToast('User deleted successfully', 'success');
      setDeleteTarget(null);
    },
    onError: (err) => {
      showToast(err instanceof Error ? err.message : 'Failed to delete user', 'error');
    },
  });

  const toggleEnabledMutation = useMutation({
    mutationFn: ({ id, enabled }: { id: string; enabled: boolean }) => updateUser(id, { enabled }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['users'] }),
    onError: (err) => showToast(err instanceof Error ? err.message : 'Failed to update user', 'error'),
  });

  const filtered = useMemo(() => {
    const q = search.toLowerCase();
    return users.filter((u) => {
      const matchSearch = !q || u.username.toLowerCase().includes(q) || (u.email ?? '').toLowerCase().includes(q);
      const matchRole = !roleFilter || u.role === roleFilter;
      return matchSearch && matchRole;
    });
  }, [users, search, roleFilter]);

  const SkeletonRows = () => (
    <>
      {Array.from({ length: 4 }).map((_, i) => (
        <tr key={i}>
          {Array.from({ length: 7 }).map((_, j) => (
            <td key={j} style={styles.td}>
              <div style={{ height: '14px', backgroundColor: '#1e293b', borderRadius: '4px', width: '70%' }} />
            </td>
          ))}
        </tr>
      ))}
    </>
  );

  return (
    <div style={styles.page}>
      <div style={styles.pageHeader}>
        <h1 style={styles.pageTitle}>User Management</h1>
        <button style={styles.addBtn} onClick={() => setShowAddModal(true)}>+ Add User</button>
      </div>

      <div style={styles.filterBar}>
        <input
          type="search"
          placeholder="Search by username or email..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          style={styles.searchInput}
          aria-label="Search users"
        />
        <select value={roleFilter} onChange={(e) => setRoleFilter(e.target.value as Role | '')} style={styles.select} aria-label="Filter by role">
          <option value="">All Roles</option>
          <option value="admin">Administrator</option>
          <option value="operator">Operator</option>
          <option value="viewer">Read Only</option>
        </select>
      </div>

      {isError && <div style={styles.errorBanner} role="alert">Failed to load users: {(error as Error).message}</div>}

      <div style={styles.tableWrapper}>
        <table style={styles.table}>
          <thead>
            <tr>
              <th style={styles.th}>Full Name</th>
              <th style={styles.th}>Username</th>
              <th style={styles.th}>Email</th>
              <th style={styles.th}>Role</th>
              <th style={styles.th}>Status</th>
              <th style={styles.th}>Last Login</th>
              <th style={styles.th}>Actions</th>
            </tr>
          </thead>
          <tbody>
            {isLoading ? <SkeletonRows /> : filtered.length === 0 ? (
              <tr><td colSpan={7} style={{ ...styles.td, textAlign: 'center', padding: '3rem', color: '#64748b' }}>No users found.</td></tr>
            ) : filtered.map((u) => {
              const rc = ROLE_COLORS[u.role];
              const isSelf = u.id === currentUser?.id;
              return (
                <tr key={u.id} style={{ borderBottom: '1px solid #1e293b' }}>
                  <td style={{ ...styles.td, fontWeight: 500, color: '#e2e8f0' }}>{u.full_name || '—'}</td>
                  <td style={styles.td}>{u.username}</td>
                  <td style={styles.td}>{u.email || '—'}</td>
                  <td style={styles.td}>
                    <span style={{ ...styles.badge, backgroundColor: rc.bg, color: rc.text }}>{ROLE_LABELS[u.role]}</span>
                  </td>
                  <td style={styles.td}>
                    <span style={{ ...styles.badge, backgroundColor: u.enabled ? '#052e16' : '#1e293b', color: u.enabled ? '#22c55e' : '#64748b' }}>
                      {u.enabled ? 'Active' : 'Disabled'}
                    </span>
                  </td>
                  <td style={styles.td}>{u.last_login_at ? new Date(u.last_login_at).toLocaleString() : '—'}</td>
                  <td style={styles.td}>
                    <div style={{ display: 'flex', gap: '0.35rem', flexWrap: 'wrap' }}>
                      <button style={styles.actionBtn} onClick={() => setEditTarget(u)}>Edit</button>
                      <button style={styles.actionBtn} onClick={() => setChangePasswordTarget(u)}>Password</button>
                      <button
                        style={styles.actionBtn}
                        onClick={() => toggleEnabledMutation.mutate({ id: u.id, enabled: !u.enabled })}
                        disabled={isSelf}
                      >
                        {u.enabled ? 'Disable' : 'Enable'}
                      </button>
                      <button
                        style={{ ...styles.actionBtn, ...styles.deleteActionBtn }}
                        onClick={() => setDeleteTarget(u)}
                        disabled={isSelf}
                      >
                        Delete
                      </button>
                    </div>
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>

      {(showAddModal || editTarget) && (
        <UserModal
          editUser={editTarget ?? undefined}
          currentUserId={currentUser?.id ?? ''}
          onClose={() => { setShowAddModal(false); setEditTarget(null); }}
          onSaved={() => { setShowAddModal(false); setEditTarget(null); queryClient.invalidateQueries({ queryKey: ['users'] }); }}
        />
      )}

      {changePasswordTarget && (
        <ChangePasswordModal user={changePasswordTarget} onClose={() => setChangePasswordTarget(null)} />
      )}

      {deleteTarget && (
        <div style={styles.overlay} role="dialog" aria-modal="true">
          <div style={styles.modal}>
            <h2 style={styles.modalTitle}>Delete User</h2>
            <p style={{ color: '#94a3b8', fontSize: '0.875rem', marginBottom: '1.5rem' }}>
              Are you sure you want to delete <strong style={{ color: '#e2e8f0' }}>{deleteTarget.username}</strong>? This cannot be undone.
            </p>
            <div style={styles.modalActions}>
              <button onClick={() => setDeleteTarget(null)} style={styles.cancelBtn} disabled={deleteMutation.isPending}>Cancel</button>
              <button
                onClick={() => deleteMutation.mutate(deleteTarget.id)}
                style={{ ...styles.deleteBtn, ...(deleteMutation.isPending ? { opacity: 0.6, cursor: 'not-allowed' } : {}) }}
                disabled={deleteMutation.isPending}
              >
                {deleteMutation.isPending ? 'Deleting...' : 'Delete'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

// ── Styles ───────────────────────────────────────────────────────────────────

const styles: Record<string, React.CSSProperties> = {
  page: { padding: '1.5rem', maxWidth: '1400px', margin: '0 auto' },
  pageHeader: { display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '1rem', flexWrap: 'wrap', gap: '0.75rem' },
  pageTitle: { fontSize: '1.5rem', fontWeight: 700, color: '#f1f5f9', margin: 0 },
  addBtn: { backgroundColor: '#2563eb', border: 'none', borderRadius: '6px', color: '#fff', cursor: 'pointer', fontSize: '0.875rem', fontWeight: 600, padding: '0.5rem 1.25rem' },
  filterBar: { display: 'flex', flexWrap: 'wrap', gap: '0.75rem', marginBottom: '1.25rem' },
  searchInput: { border: '1px solid #334155', borderRadius: '6px', fontSize: '0.875rem', padding: '0.5rem 0.75rem', minWidth: '220px', flex: '1 1 220px', background: '#1e293b', color: '#e2e8f0' },
  select: { border: '1px solid #334155', borderRadius: '6px', fontSize: '0.875rem', padding: '0.5rem 0.75rem', backgroundColor: '#1e293b', color: '#e2e8f0', minWidth: '150px' },
  errorBanner: { backgroundColor: '#450a0a', border: '1px solid #dc2626', borderRadius: '6px', color: '#f87171', fontSize: '0.875rem', padding: '0.75rem 1rem', marginBottom: '1rem' },
  tableWrapper: { overflowX: 'auto', borderRadius: '10px', border: '1px solid #1e293b' },
  table: { width: '100%', borderCollapse: 'collapse', fontSize: '0.875rem' },
  th: { backgroundColor: '#1e293b', borderBottom: '1px solid #334155', color: '#64748b', fontWeight: 700, fontSize: '0.7rem', textTransform: 'uppercase', letterSpacing: '0.08em', padding: '0.75rem 1rem', textAlign: 'left', whiteSpace: 'nowrap' },
  td: { padding: '0.75rem 1rem', color: '#94a3b8', verticalAlign: 'middle' },
  badge: { borderRadius: '9999px', display: 'inline-flex', alignItems: 'center', fontSize: '0.75rem', fontWeight: 600, padding: '0.2rem 0.6rem', whiteSpace: 'nowrap' },
  actionBtn: { backgroundColor: '#1e293b', border: '1px solid #334155', borderRadius: '4px', color: '#94a3b8', cursor: 'pointer', fontSize: '0.75rem', fontWeight: 500, padding: '0.25rem 0.6rem' },
  deleteActionBtn: { backgroundColor: '#450a0a', borderColor: '#dc2626', color: '#f87171' },
  overlay: { position: 'fixed', inset: 0, backgroundColor: 'rgba(0,0,0,0.7)', display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 1000, padding: '1rem' },
  modal: { backgroundColor: '#0f172a', border: '1px solid #1e293b', borderRadius: '12px', boxShadow: '0 25px 60px rgba(0,0,0,0.5)', maxWidth: '440px', width: '100%', padding: '1.5rem' },
  modalTitle: { fontSize: '1.125rem', fontWeight: 600, color: '#f1f5f9', margin: '0 0 1.25rem' },
  fieldGroup: { display: 'flex', flexDirection: 'column', gap: '0.25rem', marginBottom: '0.875rem' },
  label: { fontSize: '0.8125rem', fontWeight: 500, color: '#94a3b8' },
  input: { border: '1px solid #334155', borderRadius: '6px', fontSize: '0.875rem', padding: '0.5rem 0.75rem', background: '#1e293b', color: '#e2e8f0', outline: 'none', width: '100%', boxSizing: 'border-box' },
  inputError: { borderColor: '#dc2626' },
  fieldError: { color: '#f87171', fontSize: '0.75rem' },
  modalActions: { display: 'flex', gap: '0.75rem', justifyContent: 'flex-end', marginTop: '1rem' },
  cancelBtn: { backgroundColor: '#1e293b', border: '1px solid #334155', borderRadius: '6px', color: '#94a3b8', cursor: 'pointer', fontSize: '0.875rem', fontWeight: 500, padding: '0.5rem 1rem' },
  saveBtn: { backgroundColor: '#2563eb', border: 'none', borderRadius: '6px', color: '#fff', cursor: 'pointer', fontSize: '0.875rem', fontWeight: 600, padding: '0.5rem 1rem' },
  deleteBtn: { backgroundColor: '#dc2626', border: 'none', borderRadius: '6px', color: '#fff', cursor: 'pointer', fontSize: '0.875rem', fontWeight: 600, padding: '0.5rem 1rem' },
};

export default UserManagement;
