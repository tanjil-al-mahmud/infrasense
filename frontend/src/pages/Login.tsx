import React, { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useForm } from 'react-hook-form';
import { useAuth } from '../contexts/AuthContext';

/**
 * Login Page Component
 *
 * Clean modern dark-theme login form.
 * - Auto-focuses username field
 * - Password show/hide toggle
 * - Remember me checkbox
 * - Loading spinner on submit
 * - Does not clear password on error
 *
 * Requirements: 16.1
 */

interface LoginFormValues {
  username: string;
  password: string;
  rememberMe: boolean;
}

const Login: React.FC = () => {
  const { login, isAuthenticated, loading } = useAuth();
  const navigate = useNavigate();
  const [showPassword, setShowPassword] = useState(false);

  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
    setError,
  } = useForm<LoginFormValues>({
    defaultValues: { username: '', password: '', rememberMe: false },
  });

  useEffect(() => {
    if (!loading && isAuthenticated) {
      navigate('/', { replace: true });
    }
  }, [isAuthenticated, loading, navigate]);

  const onSubmit = async (data: LoginFormValues) => {
    try {
      await login({ username: data.username, password: data.password });
      navigate('/', { replace: true });
    } catch {
      setError('root', { message: 'Invalid username or password' });
    }
  };

  if (loading) {
    return (
      <div style={{ minHeight: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center', background: '#020617' }}>
        <div style={styles.spinner} aria-hidden="true" />
      </div>
    );
  }

  return (
    <div style={styles.page}>
      <div style={styles.card}>
        {/* Logo / Title */}
        <div style={styles.header}>
          <div style={styles.logoCircle}>
            <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="#3b82f6" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
              <rect x="2" y="3" width="20" height="14" rx="2" />
              <path d="M8 21h8M12 17v4" />
              <path d="M6 8h.01M9 8h6" />
            </svg>
          </div>
          <h1 style={styles.title}>InfraSense</h1>
          <p style={styles.subtitle}>Infrastructure Hardware Monitoring</p>
        </div>

        <form onSubmit={handleSubmit(onSubmit)} noValidate style={styles.form}>
          {/* API error */}
          {errors.root && (
            <div style={styles.errorBanner} role="alert">
              {errors.root.message}
            </div>
          )}

          {/* Username */}
          <div style={styles.fieldGroup}>
            <label htmlFor="username" style={styles.label}>Username</label>
            <div style={styles.inputWrapper}>
              <span style={styles.inputIcon} aria-hidden="true">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                  <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2" />
                  <circle cx="12" cy="7" r="4" />
                </svg>
              </span>
              <input
                id="username"
                type="text"
                autoComplete="username"
                autoFocus
                disabled={isSubmitting}
                style={{ ...styles.input, ...(errors.username ? styles.inputError : {}) }}
                {...register('username', { required: 'Username is required' })}
              />
            </div>
            {errors.username && <p style={styles.fieldError} role="alert">{errors.username.message}</p>}
          </div>

          {/* Password */}
          <div style={styles.fieldGroup}>
            <label htmlFor="password" style={styles.label}>Password</label>
            <div style={styles.inputWrapper}>
              <span style={styles.inputIcon} aria-hidden="true">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                  <rect x="3" y="11" width="18" height="11" rx="2" ry="2" />
                  <path d="M7 11V7a5 5 0 0 1 10 0v4" />
                </svg>
              </span>
              <input
                id="password"
                type={showPassword ? 'text' : 'password'}
                autoComplete="current-password"
                disabled={isSubmitting}
                style={{ ...styles.input, paddingRight: '2.75rem', ...(errors.password ? styles.inputError : {}) }}
                {...register('password', { required: 'Password is required' })}
              />
              <button
                type="button"
                onClick={() => setShowPassword((v) => !v)}
                style={styles.eyeBtn}
                aria-label={showPassword ? 'Hide password' : 'Show password'}
                tabIndex={-1}
              >
                {showPassword ? (
                  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                    <path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94" />
                    <path d="M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19" />
                    <line x1="1" y1="1" x2="23" y2="23" />
                  </svg>
                ) : (
                  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                    <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z" />
                    <circle cx="12" cy="12" r="3" />
                  </svg>
                )}
              </button>
            </div>
            {errors.password && <p style={styles.fieldError} role="alert">{errors.password.message}</p>}
          </div>

          {/* Remember me */}
          <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
            <input type="checkbox" id="rememberMe" style={{ width: 15, height: 15, cursor: 'pointer', accentColor: '#2563eb' }} {...register('rememberMe')} />
            <label htmlFor="rememberMe" style={{ ...styles.label, cursor: 'pointer', margin: 0 }}>Remember me</label>
          </div>

          {/* Submit */}
          <button
            type="submit"
            disabled={isSubmitting}
            style={{ ...styles.button, ...(isSubmitting ? styles.buttonDisabled : {}) }}
          >
            {isSubmitting ? (
              <span style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '0.5rem' }}>
                <span style={styles.btnSpinner} aria-hidden="true" />
                Signing in...
              </span>
            ) : 'Sign In'}
          </button>
        </form>
      </div>
    </div>
  );
};

const styles: Record<string, React.CSSProperties> = {
  page: { minHeight: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center', backgroundColor: '#020617', padding: '1rem' },
  card: { backgroundColor: '#0f172a', border: '1px solid #1e293b', borderRadius: '12px', boxShadow: '0 25px 60px rgba(0,0,0,0.5)', padding: '2.5rem 2rem', width: '100%', maxWidth: '400px' },
  header: { textAlign: 'center', marginBottom: '2rem' },
  logoCircle: { width: 56, height: 56, borderRadius: '50%', backgroundColor: '#1e3a5f', display: 'flex', alignItems: 'center', justifyContent: 'center', margin: '0 auto 1rem' },
  title: { fontSize: '1.75rem', fontWeight: 700, color: '#f1f5f9', margin: '0 0 0.25rem' },
  subtitle: { fontSize: '0.875rem', color: '#64748b', margin: 0 },
  form: { display: 'flex', flexDirection: 'column', gap: '1rem' },
  errorBanner: { backgroundColor: '#450a0a', border: '1px solid #dc2626', borderRadius: '6px', color: '#f87171', fontSize: '0.875rem', padding: '0.75rem 1rem' },
  fieldGroup: { display: 'flex', flexDirection: 'column', gap: '0.3rem' },
  label: { fontSize: '0.8125rem', fontWeight: 500, color: '#94a3b8' },
  inputWrapper: { position: 'relative', display: 'flex', alignItems: 'center' },
  inputIcon: { position: 'absolute', left: '0.75rem', color: '#475569', pointerEvents: 'none', display: 'flex', alignItems: 'center' },
  input: { border: '1px solid #334155', borderRadius: '6px', fontSize: '0.875rem', padding: '0.6rem 0.75rem 0.6rem 2.5rem', background: '#1e293b', color: '#e2e8f0', outline: 'none', width: '100%', boxSizing: 'border-box', transition: 'border-color 0.15s' },
  inputError: { borderColor: '#dc2626' },
  eyeBtn: { position: 'absolute', right: '0.75rem', background: 'none', border: 'none', color: '#475569', cursor: 'pointer', display: 'flex', alignItems: 'center', padding: 0 },
  fieldError: { color: '#f87171', fontSize: '0.75rem', margin: 0 },
  button: { backgroundColor: '#2563eb', border: 'none', borderRadius: '6px', color: '#fff', cursor: 'pointer', fontSize: '0.9rem', fontWeight: 600, marginTop: '0.5rem', padding: '0.7rem 1rem', width: '100%', transition: 'background-color 0.15s' },
  buttonDisabled: { backgroundColor: '#1e3a5f', cursor: 'not-allowed', color: '#475569' },
  spinner: { width: 32, height: 32, border: '3px solid #1e293b', borderTopColor: '#3b82f6', borderRadius: '50%', animation: 'spin 0.8s linear infinite' },
  btnSpinner: { width: 14, height: 14, border: '2px solid rgba(255,255,255,0.3)', borderTopColor: '#fff', borderRadius: '50%', animation: 'spin 0.8s linear infinite', display: 'inline-block' },
};

export default Login;
