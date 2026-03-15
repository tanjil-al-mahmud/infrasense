import React, { useState, useRef, useEffect } from 'react';
import {
  BrowserRouter,
  Routes,
  Route,
  Navigate,
  NavLink,
  Outlet,
  useNavigate,
  Link,
} from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { AuthProvider, useAuth } from './contexts/AuthContext';
import { ToastProvider } from './contexts/ToastContext';
import { usePermissions } from './hooks/usePermissions';

// Pages
const Login = React.lazy(() => import('./pages/Login'));
const Dashboard = React.lazy(() => import('./pages/Dashboard'));
const Devices = React.lazy(() => import('./pages/Devices'));
const DeviceDetail = React.lazy(() => import('./pages/DeviceDetail'));
const Alerts = React.lazy(() => import('./pages/Alerts'));
const AlertRules = React.lazy(() => import('./pages/AlertRules'));
const UserManagement = React.lazy(() => import('./pages/UserManagement'));
const UserProfile = React.lazy(() => import('./pages/UserProfile'));

class ErrorBoundary extends React.Component<{ children: React.ReactNode }, { hasError: boolean }> {
  constructor(props: { children: React.ReactNode }) {
    super(props);
    this.state = { hasError: false };
  }
  static getDerivedStateFromError() {
    return { hasError: true };
  }
  render() {
    if (this.state.hasError) {
      return <div style={{ padding: 20, color: 'white' }}>Something went wrong. Please reload.</div>;
    }
    return this.props.children;
  }
}

const queryClient = new QueryClient({
  defaultOptions: { queries: { retry: 1, staleTime: 300_000 } },
});

// ── Protected Route ──────────────────────────────────────────────────────────

const ProtectedRoute: React.FC = () => {
  const { isAuthenticated, loading, user } = useAuth();

  if (loading) {
    return (
      <div style={{ minHeight: '100vh', display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', gap: '1rem', background: '#020617' }}>
        <div style={{ width: 32, height: 32, border: '4px solid #1e293b', borderTopColor: '#3b82f6', borderRadius: '50%', animation: 'spin 0.8s linear infinite' }} aria-hidden="true" />
        <p style={{ color: '#64748b', fontSize: '0.875rem' }}>Loading...</p>
      </div>
    );
  }

  if (!isAuthenticated) return <Navigate to="/login" replace />;

  return (
    <div style={{ display: 'flex', flexDirection: 'column', minHeight: '100vh', background: '#020617' }}>
      <NavBar />
      {user?.role === 'viewer' && (
        <div style={{ backgroundColor: '#1e293b', borderBottom: '1px solid #334155', padding: '0.4rem 1rem', textAlign: 'center', fontSize: '0.8125rem', color: '#94a3b8' }}>
          Read Only Mode — You have view-only access
        </div>
      )}
      <main style={{ flex: 1, overflowY: 'auto' }}>
        <Outlet />
      </main>
    </div>
  );
};

// ── Navigation Bar ───────────────────────────────────────────────────────────

const BASE_NAV_LINKS = [
  { to: '/', label: 'Dashboard', end: true },
  { to: '/devices', label: 'Devices', end: false },
  { to: '/alerts', label: 'Alerts', end: false },
  { to: '/alert-rules', label: 'Alert Rules', end: false },
] as const;

const EXTERNAL_NAV_LINKS = [
  { href: '/grafana', label: 'Grafana' },
] as const;

const NavBar: React.FC = () => {
  const { user, logout } = useAuth();
  const { canManageUsers } = usePermissions();
  const navigate = useNavigate();
  const [menuOpen, setMenuOpen] = useState(false);
  const [dropdownOpen, setDropdownOpen] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);

  const navLinks = canManageUsers
    ? [...BASE_NAV_LINKS, { to: '/users', label: 'Users', end: false } as const]
    : BASE_NAV_LINKS;

  const handleLogout = async () => {
    setDropdownOpen(false);
    setMenuOpen(false);
    await logout();
    navigate('/login', { replace: true });
  };

  // Close dropdown on outside click
  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(e.target as Node)) {
        setDropdownOpen(false);
      }
    };
    document.addEventListener('mousedown', handler);
    return () => document.removeEventListener('mousedown', handler);
  }, []);

  const avatarLetter = user?.username?.[0]?.toUpperCase() ?? '?';

  return (
    <nav style={{ background: '#0f172a', borderBottom: '1px solid #1e293b', position: 'sticky', top: 0, zIndex: 50 }} role="navigation" aria-label="Main navigation">
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', height: 56, padding: '0 1.5rem' }}>
        {/* Brand */}
        <span style={{ color: '#f1f5f9', fontWeight: 700, fontSize: '1.125rem', letterSpacing: '-0.01em', userSelect: 'none' }}>InfraSense</span>

        {/* Desktop nav */}
        <ul className="hidden md:flex" style={{ alignItems: 'center', gap: 4, flex: 1, marginLeft: '1.5rem', listStyle: 'none', padding: 0, margin: '0 0 0 1.5rem' }} role="list">
          {navLinks.map(({ to, label, end }) => (
            <li key={to}>
              <NavLink to={to} end={end} style={({ isActive }) => ({ padding: '0.375rem 0.75rem', borderRadius: 6, fontSize: '0.875rem', fontWeight: 500, textDecoration: 'none', color: isActive ? '#f1f5f9' : '#94a3b8', backgroundColor: isActive ? '#1e293b' : 'transparent', display: 'block' })}>
                {label}
              </NavLink>
            </li>
          ))}
          {EXTERNAL_NAV_LINKS.map(({ href, label }) => (
            <li key={href}>
              <a href={href} target="_blank" rel="noopener noreferrer" style={{ padding: '0.375rem 0.75rem', borderRadius: 6, fontSize: '0.875rem', fontWeight: 500, textDecoration: 'none', color: '#94a3b8', display: 'inline-flex', alignItems: 'center', gap: 4 }}>
                {label}
                <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                  <path d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
                </svg>
              </a>
            </li>
          ))}
        </ul>

        {/* Desktop user avatar dropdown */}
        {user && (
          <div ref={dropdownRef} className="hidden md:block" style={{ position: 'relative', flexShrink: 0 }}>
            <button
              onClick={() => setDropdownOpen((v) => !v)}
              style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', background: 'none', border: 'none', cursor: 'pointer', padding: '0.25rem 0.5rem', borderRadius: 6 }}
              aria-label="User menu"
              aria-expanded={dropdownOpen}
            >
              <div style={{ width: 32, height: 32, borderRadius: '50%', backgroundColor: '#2563eb', display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#fff', fontWeight: 700, fontSize: '0.875rem', flexShrink: 0 }}>
                {avatarLetter}
              </div>
              <span style={{ color: '#e2e8f0', fontSize: '0.875rem', fontWeight: 500 }}>{user.username}</span>
              <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="#64748b" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                <polyline points="6 9 12 15 18 9" />
              </svg>
            </button>
            {dropdownOpen && (
              <div style={{ position: 'absolute', right: 0, top: '100%', marginTop: 4, backgroundColor: '#0f172a', border: '1px solid #1e293b', borderRadius: 8, boxShadow: '0 10px 30px rgba(0,0,0,0.4)', minWidth: 180, zIndex: 100 }}>
                <Link to="/profile" onClick={() => setDropdownOpen(false)} style={{ display: 'block', padding: '0.625rem 1rem', fontSize: '0.875rem', color: '#e2e8f0', textDecoration: 'none' }}>My Profile</Link>
                <div style={{ height: 1, backgroundColor: '#1e293b', margin: '0.25rem 0' }} />
                <button onClick={handleLogout} style={{ display: 'block', width: '100%', textAlign: 'left', padding: '0.625rem 1rem', fontSize: '0.875rem', color: '#f87171', background: 'none', border: 'none', cursor: 'pointer' }}>
                  Logout
                </button>
              </div>
            )}
          </div>
        )}

        {/* Hamburger (mobile) */}
        <button
          className="md:hidden"
          onClick={() => setMenuOpen((v) => !v)}
          style={{ background: 'none', border: 'none', cursor: 'pointer', display: 'flex', flexDirection: 'column', gap: 5, padding: 4 }}
          aria-label={menuOpen ? 'Close menu' : 'Open menu'}
          aria-expanded={menuOpen}
        >
          <span style={{ display: 'block', width: 20, height: 2, backgroundColor: '#94a3b8', transition: 'transform 0.2s', transform: menuOpen ? 'translateY(7px) rotate(45deg)' : 'none' }} />
          <span style={{ display: 'block', width: 20, height: 2, backgroundColor: '#94a3b8', opacity: menuOpen ? 0 : 1, transition: 'opacity 0.2s' }} />
          <span style={{ display: 'block', width: 20, height: 2, backgroundColor: '#94a3b8', transition: 'transform 0.2s', transform: menuOpen ? 'translateY(-7px) rotate(-45deg)' : 'none' }} />
        </button>
      </div>

      {/* Mobile menu */}
      {menuOpen && (
        <div style={{ backgroundColor: '#0f172a', borderTop: '1px solid #1e293b', padding: '0.75rem 1rem 1rem' }}>
          <ul style={{ listStyle: 'none', padding: 0, margin: '0 0 0.75rem', display: 'flex', flexDirection: 'column', gap: 2 }}>
            {navLinks.map(({ to, label, end }) => (
              <li key={to}>
                <NavLink to={to} end={end} onClick={() => setMenuOpen(false)} style={({ isActive }) => ({ display: 'block', padding: '0.5rem 0.75rem', borderRadius: 6, fontSize: '0.875rem', fontWeight: 500, textDecoration: 'none', color: isActive ? '#f1f5f9' : '#94a3b8', backgroundColor: isActive ? '#1e293b' : 'transparent' })}>
                  {label}
                </NavLink>
              </li>
            ))}
          </ul>
          {user && (
            <div style={{ borderTop: '1px solid #1e293b', paddingTop: '0.75rem', display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
                <div style={{ width: 28, height: 28, borderRadius: '50%', backgroundColor: '#2563eb', display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#fff', fontWeight: 700, fontSize: '0.8rem' }}>{avatarLetter}</div>
                <div>
                  <div style={{ color: '#e2e8f0', fontSize: '0.875rem', fontWeight: 500 }}>{user.username}</div>
                  <div style={{ color: '#64748b', fontSize: '0.75rem', textTransform: 'capitalize' }}>{user.role}</div>
                </div>
              </div>
              <div style={{ display: 'flex', gap: '0.5rem' }}>
                <Link to="/profile" onClick={() => setMenuOpen(false)} style={{ fontSize: '0.8rem', color: '#94a3b8', textDecoration: 'none', padding: '0.375rem 0.625rem', border: '1px solid #334155', borderRadius: 4 }}>Profile</Link>
                <button onClick={handleLogout} style={{ fontSize: '0.8rem', color: '#f87171', background: 'none', border: '1px solid #7f1d1d', borderRadius: 4, cursor: 'pointer', padding: '0.375rem 0.625rem' }}>Logout</button>
              </div>
            </div>
          )}
        </div>
      )}
    </nav>
  );
};

// ── Routes ───────────────────────────────────────────────────────────────────

const AppRoutes: React.FC = () => (
  <ErrorBoundary>
    <React.Suspense fallback={<div style={{ padding: 20, color: 'white' }}>Loading page...</div>}>
      <Routes>
        <Route path="/login" element={<Login />} />
        <Route element={<ProtectedRoute />}>
          <Route path="/" element={<Dashboard />} />
          <Route path="/devices" element={<Devices />} />
          <Route path="/devices/:id" element={<DeviceDetail />} />
          <Route path="/alerts" element={<Alerts />} />
          <Route path="/alert-rules" element={<AlertRules />} />
          <Route path="/users" element={<UserManagement />} />
          <Route path="/profile" element={<UserProfile />} />
        </Route>
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </React.Suspense>
  </ErrorBoundary>
);

// ── Root App ─────────────────────────────────────────────────────────────────

const App: React.FC = () => (
  <QueryClientProvider client={queryClient}>
    <BrowserRouter>
      <AuthProvider>
        <ToastProvider>
          <AppRoutes />
        </ToastProvider>
      </AuthProvider>
    </BrowserRouter>
  </QueryClientProvider>
);

export default App;
