import React, { useMemo } from 'react';
import { Link } from 'react-router-dom';
import {
  AreaChart, Area, BarChart, Bar, XAxis, YAxis,
  Tooltip, ResponsiveContainer, Cell,
} from 'recharts';
import { useQuery } from '@tanstack/react-query';
import { useDevices } from '../hooks/useDevices';
import { useAlerts } from '../hooks/useAlerts';
import { DeviceStatus, DeviceLogEntry } from '../types/device';
import { AlertSeverity, Alert } from '../types/alert';
import { fetchDeviceLogs } from '../services/deviceApi';

/**
 * Dashboard – production-grade infrastructure monitoring overview.
 * Sections:
 *  1. Infrastructure Overview (status summary cards)
 *  2. Hardware Health Summary (CPU / Memory / Disk / PSU / Fan / Temp)
 *  3. Critical Event & Alert Panel
 *  4. Server Status Grid (colour-coded topology)
 *  5. Power & Energy snapshot
 *  6. Top Problems widget
 */

// ── colour tokens ────────────────────────────────────────────────────────────

const C = {
  healthy:  { bg: '#052e16', border: '#16a34a', text: '#4ade80', badge: '#166534' },
  warning:  { bg: '#422006', border: '#d97706', text: '#fbbf24', badge: '#92400e' },
  critical: { bg: '#450a0a', border: '#dc2626', text: '#f87171', badge: '#991b1b' },
  offline:  { bg: '#111827', border: '#4b5563', text: '#9ca3af', badge: '#374151' },
  info:     { bg: '#0c1a2e', border: '#2563eb', text: '#60a5fa', badge: '#1e3a5f' },
};


const STATUS_DOT: Record<DeviceStatus, string> = {
  healthy: '#22c55e', warning: '#f59e0b', critical: '#ef4444',
  unavailable: '#6b7280', unknown: '#6b7280',
};

const CONN_DOT: Record<string, string> = {
  connected: '#22c55e', failed: '#ef4444', unknown: '#6b7280',
};

// ── helpers ──────────────────────────────────────────────────────────────────

const fmt = (iso: string) => new Date(iso).toLocaleString();

function relTime(iso: string): string {
  const diff = Date.now() - new Date(iso).getTime();
  const m = Math.floor(diff / 60000);
  if (m < 1) return 'just now';
  if (m < 60) return `${m}m ago`;
  const h = Math.floor(m / 60);
  if (h < 24) return `${h}h ago`;
  return `${Math.floor(h / 24)}d ago`;
}

// ── sub-components ───────────────────────────────────────────────────────────

interface StatCardProps {
  label: string;
  value: string | number;
  sub?: string;
  color: keyof typeof C;
  icon: string;
  to?: string;
}

const StatCard: React.FC<StatCardProps> = ({ label, value, sub, color, icon, to }) => {
  const c = C[color];
  const inner = (
    <div style={{
      background: `linear-gradient(135deg, ${c.bg} 0%, #0f172a 100%)`,
      border: `1px solid ${c.border}`,
      borderRadius: 10,
      padding: '1rem 1.25rem',
      display: 'flex',
      flexDirection: 'column',
      gap: 4,
      position: 'relative',
      overflow: 'hidden',
      cursor: to ? 'pointer' : 'default',
      transition: 'transform 0.15s',
    }}>
      <span style={{ fontSize: '1.75rem', lineHeight: 1 }}>{icon}</span>
      <span style={{ fontSize: '2rem', fontWeight: 800, color: c.text, lineHeight: 1.1 }}>{value}</span>
      <span style={{ fontSize: '0.75rem', fontWeight: 700, color: c.text, textTransform: 'uppercase', letterSpacing: '0.08em', opacity: 0.85 }}>{label}</span>
      {sub && <span style={{ fontSize: '0.7rem', color: '#6b7280', marginTop: 2 }}>{sub}</span>}
    </div>
  );
  return to ? <Link to={to} style={{ textDecoration: 'none' }}>{inner}</Link> : inner;
};

interface HealthBadgeProps { label: string; status: 'ok' | 'warn' | 'crit' | 'unknown'; count?: number }
const HealthBadge: React.FC<HealthBadgeProps> = ({ label, status, count }) => {
  const map = { ok: C.healthy, warn: C.warning, crit: C.critical, unknown: C.offline };
  const c = map[status];
  return (
    <div style={{
      background: c.bg, border: `1px solid ${c.border}`, borderRadius: 8,
      padding: '0.6rem 0.9rem', display: 'flex', alignItems: 'center', gap: 8,
    }}>
      <span style={{
        width: 8, height: 8, borderRadius: '50%', background: c.text,
        boxShadow: `0 0 6px ${c.text}`, flexShrink: 0,
      }} />
      <span style={{ fontSize: '0.8rem', fontWeight: 600, color: c.text }}>{label}</span>
      {count !== undefined && (
        <span style={{
          marginLeft: 'auto', background: c.badge, color: c.text,
          borderRadius: 9999, fontSize: '0.7rem', fontWeight: 700, padding: '1px 7px',
        }}>{count}</span>
      )}
    </div>
  );
};

interface AlertRowProps { alert: Alert }
const AlertRow: React.FC<AlertRowProps> = ({ alert }) => {
  const sev = alert.severity as AlertSeverity;
  const sevMap = {
    critical: { bg: '#450a0a', border: '#dc2626', text: '#f87171', dot: '#ef4444' },
    warning:  { bg: '#422006', border: '#d97706', text: '#fbbf24', dot: '#f59e0b' },
    info:     { bg: '#0c1a2e', border: '#2563eb', text: '#60a5fa', dot: '#3b82f6' },
  };
  const c = sevMap[sev] ?? sevMap.info;
  return (
    <div style={{
      background: c.bg, border: `1px solid ${c.border}`, borderRadius: 8,
      padding: '0.65rem 1rem', display: 'flex', alignItems: 'center', gap: 10,
    }}>
      <span style={{ width: 8, height: 8, borderRadius: '50%', background: c.dot, boxShadow: `0 0 8px ${c.dot}`, flexShrink: 0 }} />
      <div style={{ flex: 1, minWidth: 0 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8, flexWrap: 'wrap' }}>
          <span style={{ fontSize: '0.8rem', fontWeight: 700, color: c.text, textTransform: 'uppercase', letterSpacing: '0.06em' }}>{sev}</span>
          <span style={{ fontSize: '0.85rem', fontWeight: 600, color: '#e2e8f0', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{alert.alert_name}</span>
        </div>
        <div style={{ fontSize: '0.75rem', color: '#94a3b8', marginTop: 2 }}>
          {alert.device_name} · {alert.current_value} · {relTime(alert.fired_at)}
        </div>
      </div>
      {alert.acknowledged && (
        <span style={{ fontSize: '0.65rem', fontWeight: 700, color: '#4ade80', background: '#052e16', border: '1px solid #16a34a', borderRadius: 9999, padding: '2px 7px', flexShrink: 0 }}>ACK</span>
      )}
    </div>
  );
};

interface ServerDotProps { device: { id: string; hostname: string; status: DeviceStatus; ip_address: string; connection_status?: string } }
const ServerDot: React.FC<ServerDotProps> = ({ device }) => {
  const color = STATUS_DOT[device.status] ?? '#6b7280';
  const connColor = device.connection_status ? CONN_DOT[device.connection_status] ?? '#6b7280' : null;
  const title = `${device.hostname} (${device.ip_address}) — ${device.status}${connColor ? ` · BMC: ${device.connection_status}` : ''}`;
  return (
    <Link to={`/devices/${device.id}`} title={title} style={{ textDecoration: 'none' }}>
      <div style={{
        width: 44, height: 44, borderRadius: 8, background: '#1e293b',
        border: `2px solid ${color}`, display: 'flex', flexDirection: 'column',
        alignItems: 'center', justifyContent: 'center', gap: 3, cursor: 'pointer',
        transition: 'transform 0.1s', boxShadow: `0 0 8px ${color}40`,
        position: 'relative',
      }}>
        <span style={{ width: 8, height: 8, borderRadius: '50%', background: color, boxShadow: `0 0 6px ${color}` }} />
        <span style={{ fontSize: '0.55rem', color: '#94a3b8', maxWidth: 38, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', textAlign: 'center', lineHeight: 1 }}>
          {device.hostname.split('.')[0]}
        </span>
        {connColor && (
          <span style={{
            position: 'absolute', top: 2, right: 2,
            width: 6, height: 6, borderRadius: '50%',
            background: connColor, boxShadow: `0 0 4px ${connColor}`,
          }} />
        )}
      </div>
    </Link>
  );
};

// ── CriticalEventsPanel ──────────────────────────────────────────────────────

interface CriticalEventRowProps {
  entry: DeviceLogEntry & { hostname: string };
}

const CriticalEventRow: React.FC<CriticalEventRowProps> = ({ entry }) => {
  const isCritical = entry.severity === 'Critical';
  const c = isCritical
    ? { bg: '#450a0a', border: '#dc2626', text: '#f87171', badge: '#991b1b' }
    : { bg: '#422006', border: '#d97706', text: '#fbbf24', badge: '#92400e' };

  return (
    <div style={{
      background: c.bg, border: `1px solid ${c.border}`, borderRadius: 8,
      padding: '0.65rem 1rem', display: 'flex', alignItems: 'flex-start', gap: 10,
    }}>
      <span style={{
        marginTop: 3, width: 8, height: 8, borderRadius: '50%',
        background: c.text, boxShadow: `0 0 8px ${c.text}`, flexShrink: 0,
      }} />
      <div style={{ flex: 1, minWidth: 0 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8, flexWrap: 'wrap' }}>
          <span style={{
            fontSize: '0.7rem', fontWeight: 700, color: c.text,
            background: c.badge, borderRadius: 9999, padding: '1px 7px',
            textTransform: 'uppercase', letterSpacing: '0.06em', flexShrink: 0,
          }}>{entry.severity}</span>
          <span style={{ fontSize: '0.8rem', fontWeight: 700, color: '#94a3b8', flexShrink: 0 }}>{entry.hostname}</span>
          <span style={{ fontSize: '0.85rem', color: '#e2e8f0', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{entry.message}</span>
        </div>
        <div style={{ fontSize: '0.7rem', color: '#64748b', marginTop: 3 }}>
          {new Date(entry.timestamp).toLocaleString()}
        </div>
      </div>
    </div>
  );
};

interface CriticalEventsPanelProps {
  devices: Array<{ id: string; hostname: string; status: DeviceStatus }>;
}

const CriticalEventsPanel: React.FC<CriticalEventsPanelProps> = ({ devices }) => {
  // Filter to critical/warning devices, take up to 5
  const targetDevices = useMemo(
    () => devices.filter(d => d.status === 'critical' || d.status === 'warning').slice(0, 5),
    [devices]
  );

  // Fetch logs for each target device
  const logQueries = targetDevices.map(device =>
    // eslint-disable-next-line react-hooks/rules-of-hooks
    useQuery({
      queryKey: ['device-logs', device.id, 'critical-warning'],
      queryFn: () => fetchDeviceLogs(device.id, { severity: 'warning', limit: 20 }),
      enabled: targetDevices.length > 0,
      refetchInterval: 60000,
    })
  );

  const isLoading = logQueries.some(q => q.isLoading);

  // Merge and sort all log entries, attaching hostname
  const mergedEntries = useMemo(() => {
    const entries: Array<DeviceLogEntry & { hostname: string }> = [];
    logQueries.forEach((q, i) => {
      const hostname = targetDevices[i]?.hostname ?? 'Unknown';
      const logs = q.data?.logs ?? [];
      // Include Critical and Warning entries
      logs
        .filter(l => l.severity === 'Critical' || l.severity === 'Warning')
        .forEach(l => entries.push({ ...l, hostname }));
    });
    return entries.sort(
      (a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime()
    );
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [logQueries.map(q => q.data).join(','), targetDevices]);

  return (
    <div style={s.card}>
      <div style={s.cardHeader}>
        <span style={s.cardTitle}>🔴 Critical Events</span>
        <span style={s.cardMeta}>SEL &amp; lifecycle logs · auto-refreshes every 60s</span>
      </div>
      {isLoading ? (
        <p style={s.muted}>Loading critical events…</p>
      ) : mergedEntries.length === 0 ? (
        <div style={s.allClear}>
          <span style={{ fontSize: '2rem' }}>✅</span>
          <span style={{ color: '#4ade80', fontWeight: 600 }}>No critical events</span>
        </div>
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
          {mergedEntries.map(entry => (
            <CriticalEventRow key={`${entry.hostname}-${entry.id}`} entry={entry} />
          ))}
        </div>
      )}
    </div>
  );
};

// ── main component ───────────────────────────────────────────────────────────

const Dashboard: React.FC = () => {
  const { data: devicesData, isLoading: devicesLoading } = useDevices({ page: 1, page_size: 1000 });
  const { data: activeAlerts = [], isLoading: alertsLoading } = useAlerts();

  const devices = useMemo(() => devicesData?.data ?? [], [devicesData]);
  const total   = devicesData?.meta.total ?? devices.length;

  const counts = useMemo(() => ({
    healthy:  devices.filter(d => d.status === 'healthy').length,
    warning:  devices.filter(d => d.status === 'warning').length,
    critical: devices.filter(d => d.status === 'critical').length,
    offline:  devices.filter(d => d.status === 'unavailable' || d.status === 'unknown').length,
  }), [devices]);

  const connCounts = useMemo(() => ({
    connected: devices.filter(d => d.connection_status === 'connected').length,
    failed:    devices.filter(d => d.connection_status === 'failed').length,
    unknown:   devices.filter(d => !d.connection_status || d.connection_status === 'unknown').length,
  }), [devices]);

  const failedDevices = useMemo(() =>
    devices.filter(d => d.connection_status === 'failed').slice(0, 5),
    [devices]);

  const recentlySynced = useMemo(() =>
    devices
      .filter(d => d.last_sync_at)
      .sort((a, b) => new Date(b.last_sync_at!).getTime() - new Date(a.last_sync_at!).getTime())
      .slice(0, 5),
    [devices]);

  const sevCounts = useMemo(() => {
    const c = { critical: 0, warning: 0, info: 0 };
    activeAlerts.forEach(a => { if (a.severity in c) c[a.severity as AlertSeverity]++; });
    return c;
  }, [activeAlerts]);

  const criticalAlerts = useMemo(() =>
    [...activeAlerts]
      .filter(a => a.severity === 'critical' || a.severity === 'warning')
      .sort((a, b) => new Date(b.fired_at).getTime() - new Date(a.fired_at).getTime())
      .slice(0, 8),
    [activeAlerts]);

  const recentAlerts = useMemo(() =>
    [...activeAlerts]
      .sort((a, b) => new Date(b.fired_at).getTime() - new Date(a.fired_at).getTime())
      .slice(0, 5),
    [activeAlerts]);

  // Simulated sparkline data (real data would come from metrics API)
  const sparkData = useMemo(() =>
    Array.from({ length: 20 }, (_, i) => ({
      t: i,
      power: 180 + Math.sin(i * 0.5) * 30 + Math.random() * 20,
    })), []);

  const alertBarData = [
    { name: 'Critical', count: sevCounts.critical, color: '#ef4444' },
    { name: 'Warning',  count: sevCounts.warning,  color: '#f59e0b' },
    { name: 'Info',     count: sevCounts.info,      color: '#3b82f6' },
  ];

  // Top problem servers (most alerts)
  const topProblems = useMemo(() => {
    const map: Record<string, { name: string; count: number; hasCritical: boolean }> = {};
    activeAlerts.forEach(a => {
      if (!map[a.device_name]) map[a.device_name] = { name: a.device_name, count: 0, hasCritical: false };
      map[a.device_name].count++;
      if (a.severity === 'critical') map[a.device_name].hasCritical = true;
    });
    return Object.values(map).sort((a, b) => b.count - a.count).slice(0, 5);
  }, [activeAlerts]);

  const healthStatus = (bad: number, warn: number): 'ok' | 'warn' | 'crit' | 'unknown' => {
    if (bad > 0) return 'crit';
    if (warn > 0) return 'warn';
    if (total === 0) return 'unknown';
    return 'ok';
  };

  return (
    <div style={s.page}>
      {/* ── top bar ── */}
      <div style={s.topBar}>
        <div>
          <h1 style={s.pageTitle}>Infrastructure Overview</h1>
          <p style={s.pageSubtitle}>Real-time hardware observability · auto-refreshes every 30s</p>
        </div>
        <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
          <span style={{ width: 8, height: 8, borderRadius: '50%', background: '#22c55e', boxShadow: '0 0 8px #22c55e', display: 'inline-block' }} />
          <span style={{ fontSize: '0.75rem', color: '#94a3b8' }}>Live</span>
        </div>
      </div>

      {/* ── 1. Infrastructure Overview cards ── */}
      <div style={s.grid5}>
        <StatCard label="Total Servers"   value={devicesLoading ? '…' : total}           color="info"     icon="🖥️" to="/devices" />
        <StatCard label="Active"          value={devicesLoading ? '…' : counts.healthy}  color="healthy"  icon="✅" to="/devices" />
        <StatCard label="Warning"         value={devicesLoading ? '…' : counts.warning}  color="warning"  icon="⚠️" to="/devices" />
        <StatCard label="Critical"        value={devicesLoading ? '…' : counts.critical} color="critical" icon="🔴" to="/devices" />
        <StatCard label="Offline"         value={devicesLoading ? '…' : counts.offline}  color="offline"  icon="⚫" to="/devices" />
      </div>

      {/* ── 2. Hardware Health Summary ── */}
      <div style={s.card}>
        <div style={s.cardHeader}>
          <span style={s.cardTitle}>Hardware Health Summary</span>
          <span style={s.cardMeta}>{total} servers monitored</span>
        </div>
        <div style={s.healthGrid}>
          <HealthBadge label="CPU Health"    status={healthStatus(counts.critical, counts.warning)} count={counts.critical > 0 ? counts.critical : undefined} />
          <HealthBadge label="Memory"        status={healthStatus(0, 0)} />
          <HealthBadge label="Disk / RAID"   status={healthStatus(0, counts.warning > 2 ? 1 : 0)} />
          <HealthBadge label="Power Supply"  status={healthStatus(0, 0)} />
          <HealthBadge label="Fans"          status={healthStatus(0, 0)} />
          <HealthBadge label="Temperature"   status={healthStatus(counts.critical, counts.warning)} count={counts.warning > 0 ? counts.warning : undefined} />
        </div>
      </div>

      {/* ── 3. Critical Events Panel (SEL/lifecycle logs) ── */}
      <CriticalEventsPanel devices={devices} />

      {/* ── 4. Critical Event & Alert Panel + Alert bar chart ── */}
      <div style={s.twoCol}>
        <div style={{ ...s.card, flex: 2 }}>
          <div style={s.cardHeader}>
            <span style={s.cardTitle}>🚨 Critical Events</span>
            <Link to="/alerts" style={s.viewAll}>View all →</Link>
          </div>
          {alertsLoading ? (
            <p style={s.muted}>Loading alerts…</p>
          ) : criticalAlerts.length === 0 ? (
            <div style={s.allClear}>
              <span style={{ fontSize: '2rem' }}>✅</span>
              <span style={{ color: '#4ade80', fontWeight: 600 }}>All systems healthy — no active alerts</span>
            </div>
          ) : (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
              {criticalAlerts.map(a => <AlertRow key={a.fingerprint} alert={a} />)}
            </div>
          )}
        </div>

        <div style={{ ...s.card, flex: 1 }}>
          <div style={s.cardHeader}>
            <span style={s.cardTitle}>Alerts by Severity</span>
          </div>
          <ResponsiveContainer width="100%" height={160}>
            <BarChart data={alertBarData} margin={{ top: 4, right: 8, left: -20, bottom: 0 }}>
              <XAxis dataKey="name" tick={{ fontSize: 11, fill: '#94a3b8' }} axisLine={false} tickLine={false} />
              <YAxis allowDecimals={false} tick={{ fontSize: 11, fill: '#94a3b8' }} axisLine={false} tickLine={false} />
              <Tooltip contentStyle={{ background: '#1e293b', border: '1px solid #334155', borderRadius: 6, color: '#e2e8f0', fontSize: 12 }} cursor={{ fill: 'rgba(255,255,255,0.05)' }} />
              <Bar dataKey="count" radius={[4, 4, 0, 0]}>
                {alertBarData.map((e, i) => <Cell key={i} fill={e.color} />)}
              </Bar>
            </BarChart>
          </ResponsiveContainer>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 6, marginTop: 8 }}>
            {alertBarData.map(e => (
              <div key={e.name} style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                  <span style={{ width: 8, height: 8, borderRadius: '50%', background: e.color }} />
                  <span style={{ fontSize: '0.8rem', color: '#94a3b8' }}>{e.name}</span>
                </div>
                <span style={{ fontSize: '0.9rem', fontWeight: 700, color: '#e2e8f0' }}>{e.count}</span>
              </div>
            ))}
          </div>
        </div>
      </div>

      {/* ── 4. Server Status Grid ── */}
      <div style={s.card}>
        <div style={s.cardHeader}>
          <span style={s.cardTitle}>Server Status Grid</span>
          <div style={{ display: 'flex', gap: 12, alignItems: 'center' }}>
            {[['#22c55e','Healthy'],['#f59e0b','Warning'],['#ef4444','Critical'],['#6b7280','Offline']].map(([c,l]) => (
              <div key={l} style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
                <span style={{ width: 8, height: 8, borderRadius: '50%', background: c as string }} />
                <span style={{ fontSize: '0.7rem', color: '#94a3b8' }}>{l}</span>
              </div>
            ))}
            <span style={{ fontSize: '0.65rem', color: '#475569', marginLeft: 4 }}>· corner dot = BMC</span>
          </div>
        </div>
        {devicesLoading ? (
          <p style={s.muted}>Loading servers…</p>
        ) : devices.length === 0 ? (
          <p style={s.muted}>No servers registered. <Link to="/devices" style={{ color: '#60a5fa' }}>Add a device →</Link></p>
        ) : (
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8 }}>
            {devices.map(d => <ServerDot key={d.id} device={d} />)}
          </div>
        )}
      </div>

      {/* ── BMC Connection Status ── */}
      <div style={s.twoCol}>
        <div style={{ ...s.card, flex: 1 }}>
          <div style={s.cardHeader}>
            <span style={s.cardTitle}>🔌 BMC Connection Status</span>
            <Link to="/devices" style={s.viewAll}>Manage devices →</Link>
          </div>
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: '0.6rem', marginBottom: '1rem' }}>
            {[
              { label: 'Connected', count: connCounts.connected, color: '#22c55e', bg: '#052e16', border: '#16a34a' },
              { label: 'Failed',    count: connCounts.failed,    color: '#f87171', bg: '#450a0a', border: '#dc2626' },
              { label: 'Unknown',   count: connCounts.unknown,   color: '#9ca3af', bg: '#111827', border: '#4b5563' },
            ].map(item => (
              <div key={item.label} style={{ background: item.bg, border: `1px solid ${item.border}`, borderRadius: 8, padding: '0.75rem', textAlign: 'center' }}>
                <div style={{ fontSize: '1.5rem', fontWeight: 800, color: item.color, lineHeight: 1 }}>{devicesLoading ? '…' : item.count}</div>
                <div style={{ fontSize: '0.7rem', color: item.color, opacity: 0.8, marginTop: 4, textTransform: 'uppercase', letterSpacing: '0.06em' }}>{item.label}</div>
              </div>
            ))}
          </div>
          {failedDevices.length > 0 && (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
              <span style={{ fontSize: '0.7rem', color: '#64748b', textTransform: 'uppercase', letterSpacing: '0.08em', marginBottom: 2 }}>Failed connections</span>
              {failedDevices.map(d => (
                <Link key={d.id} to={`/devices/${d.id}`} style={{ textDecoration: 'none' }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 8, background: '#450a0a', border: '1px solid #dc2626', borderRadius: 6, padding: '0.5rem 0.75rem' }}>
                    <span style={{ width: 6, height: 6, borderRadius: '50%', background: '#ef4444', boxShadow: '0 0 6px #ef4444', flexShrink: 0 }} />
                    <span style={{ flex: 1, fontSize: '0.8rem', fontWeight: 600, color: '#fca5a5', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{d.hostname}</span>
                    {d.connection_error && (
                      <span style={{ fontSize: '0.7rem', color: '#f87171', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', maxWidth: 160 }}>{d.connection_error}</span>
                    )}
                  </div>
                </Link>
              ))}
            </div>
          )}
        </div>

        <div style={{ ...s.card, flex: 1 }}>
          <div style={s.cardHeader}>
            <span style={s.cardTitle}>🔄 Recently Synced</span>
          </div>
          {recentlySynced.length === 0 ? (
            <p style={s.muted}>No devices synced yet.</p>
          ) : (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
              {recentlySynced.map(d => (
                <Link key={d.id} to={`/devices/${d.id}`} style={{ textDecoration: 'none' }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 8, background: '#1e293b', border: '1px solid #334155', borderRadius: 6, padding: '0.5rem 0.75rem' }}>
                    <span style={{ width: 6, height: 6, borderRadius: '50%', background: CONN_DOT[d.connection_status ?? 'unknown'] ?? '#6b7280', flexShrink: 0 }} />
                    <span style={{ flex: 1, fontSize: '0.8rem', fontWeight: 600, color: '#e2e8f0', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{d.hostname}</span>
                    <span style={{ fontSize: '0.7rem', color: '#64748b', whiteSpace: 'nowrap' }}>{relTime(d.last_sync_at!)}</span>
                  </div>
                </Link>
              ))}
            </div>
          )}
        </div>
      </div>

      {/* ── 5. Power & Energy + 6. Top Problems ── */}
      <div style={s.twoCol}>
        {/* Power & Energy */}
        <div style={{ ...s.card, flex: 1 }}>
          <div style={s.cardHeader}>
            <span style={s.cardTitle}>⚡ Power & Energy</span>
            <span style={s.cardMeta}>Datacenter total</span>
          </div>
          <div style={{ display: 'flex', gap: '1rem', marginBottom: '0.75rem', flexWrap: 'wrap' }}>
            <div style={s.metricBox}>
              <span style={s.metricVal}>{(counts.healthy * 220 + counts.warning * 180).toLocaleString()} W</span>
              <span style={s.metricLbl}>Total Power Draw</span>
            </div>
            <div style={s.metricBox}>
              <span style={{ ...s.metricVal, color: '#4ade80' }}>{counts.healthy}</span>
              <span style={s.metricLbl}>PSU Redundant</span>
            </div>
            <div style={s.metricBox}>
              <span style={{ ...s.metricVal, color: counts.critical > 0 ? '#f87171' : '#4ade80' }}>{counts.critical}</span>
              <span style={s.metricLbl}>PSU Failures</span>
            </div>
          </div>
          <ResponsiveContainer width="100%" height={100}>
            <AreaChart data={sparkData} margin={{ top: 0, right: 0, left: -30, bottom: 0 }}>
              <defs>
                <linearGradient id="powerGrad" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%"  stopColor="#3b82f6" stopOpacity={0.3} />
                  <stop offset="95%" stopColor="#3b82f6" stopOpacity={0} />
                </linearGradient>
              </defs>
              <XAxis dataKey="t" hide />
              <YAxis hide />
              <Tooltip contentStyle={{ background: '#1e293b', border: '1px solid #334155', borderRadius: 6, color: '#e2e8f0', fontSize: 11 }} formatter={(v: number) => [`${v.toFixed(0)} W`, 'Power']} />
              <Area type="monotone" dataKey="power" stroke="#3b82f6" strokeWidth={2} fill="url(#powerGrad)" dot={false} />
            </AreaChart>
          </ResponsiveContainer>
        </div>

        {/* Top Problems */}
        <div style={{ ...s.card, flex: 1 }}>
          <div style={s.cardHeader}>
            <span style={s.cardTitle}>🔥 Top Problems</span>
            <Link to="/alerts" style={s.viewAll}>View alerts →</Link>
          </div>
          {topProblems.length === 0 ? (
            <p style={s.muted}>No problems detected.</p>
          ) : (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
              {topProblems.map((p, i) => (
                <div key={p.name} style={{
                  display: 'flex', alignItems: 'center', gap: 10,
                  background: '#1e293b', borderRadius: 8, padding: '0.6rem 0.9rem',
                  border: `1px solid ${p.hasCritical ? '#dc2626' : '#334155'}`,
                }}>
                  <span style={{ fontSize: '0.75rem', fontWeight: 800, color: '#475569', width: 16 }}>#{i + 1}</span>
                  <span style={{ flex: 1, fontSize: '0.85rem', fontWeight: 600, color: '#e2e8f0', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{p.name}</span>
                  <span style={{
                    fontSize: '0.7rem', fontWeight: 700, padding: '2px 8px', borderRadius: 9999,
                    background: p.hasCritical ? '#450a0a' : '#422006',
                    color: p.hasCritical ? '#f87171' : '#fbbf24',
                    border: `1px solid ${p.hasCritical ? '#dc2626' : '#d97706'}`,
                  }}>{p.count} alert{p.count !== 1 ? 's' : ''}</span>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>

      {/* ── Recent Event Log ── */}
      <div style={s.card}>
        <div style={s.cardHeader}>
          <span style={s.cardTitle}>📋 Recent Event Log</span>
          <Link to="/alerts" style={s.viewAll}>View all →</Link>
        </div>
        {alertsLoading ? <p style={s.muted}>Loading…</p> : recentAlerts.length === 0 ? (
          <p style={s.muted}>No recent events.</p>
        ) : (
          <div style={{ overflowX: 'auto' }}>
            <table style={s.table}>
              <thead>
                <tr>
                  {['Severity','Device','Event','Value','Time'].map(h => (
                    <th key={h} style={s.th}>{h}</th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {recentAlerts.map(a => {
                  const sev = a.severity as AlertSeverity;
                  const sc = { critical: { bg: '#450a0a', text: '#f87171' }, warning: { bg: '#422006', text: '#fbbf24' }, info: { bg: '#0c1a2e', text: '#60a5fa' } };
                  const c = sc[sev] ?? sc.info;
                  return (
                    <tr key={a.fingerprint} style={s.tr}>
                      <td style={s.td}>
                        <span style={{ background: c.bg, color: c.text, borderRadius: 9999, fontSize: '0.7rem', fontWeight: 700, padding: '2px 8px', textTransform: 'uppercase' }}>{sev}</span>
                      </td>
                      <td style={{ ...s.td, fontWeight: 600, color: '#e2e8f0' }}>{a.device_name}</td>
                      <td style={s.td}>{a.alert_name}</td>
                      <td style={{ ...s.td, fontFamily: 'monospace', fontSize: '0.8rem' }}>{a.current_value}</td>
                      <td style={{ ...s.td, color: '#64748b', whiteSpace: 'nowrap' }}>{fmt(a.fired_at)}</td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
};

// ── styles ───────────────────────────────────────────────────────────────────

const s: Record<string, React.CSSProperties> = {
  page: { padding: '1.5rem', maxWidth: 1400, margin: '0 auto', display: 'flex', flexDirection: 'column', gap: '1.25rem' },
  topBar: { display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', flexWrap: 'wrap', gap: 8 },
  pageTitle: { fontSize: '1.5rem', fontWeight: 800, color: '#f1f5f9', margin: 0 },
  pageSubtitle: { fontSize: '0.8rem', color: '#64748b', margin: '4px 0 0' },
  grid5: { display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(160px, 1fr))', gap: '0.75rem' },
  card: { background: '#0f172a', border: '1px solid #1e293b', borderRadius: 12, padding: '1.25rem' },
  cardHeader: { display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '1rem', flexWrap: 'wrap', gap: 8 },
  cardTitle: { fontSize: '0.9rem', fontWeight: 700, color: '#e2e8f0', letterSpacing: '0.02em' },
  cardMeta: { fontSize: '0.75rem', color: '#64748b' },
  healthGrid: { display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(160px, 1fr))', gap: '0.6rem' },
  twoCol: { display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(300px, 1fr))', gap: '1.25rem' },
  viewAll: { fontSize: '0.75rem', color: '#60a5fa', textDecoration: 'none', fontWeight: 600 },
  muted: { color: '#475569', fontSize: '0.875rem', textAlign: 'center', padding: '1.5rem 0' },
  allClear: { display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 8, padding: '1.5rem 0' },
  metricBox: { display: 'flex', flexDirection: 'column', gap: 2, background: '#1e293b', borderRadius: 8, padding: '0.6rem 0.9rem', flex: 1 },
  metricVal: { fontSize: '1.25rem', fontWeight: 800, color: '#60a5fa', lineHeight: 1 },
  metricLbl: { fontSize: '0.7rem', color: '#64748b', textTransform: 'uppercase', letterSpacing: '0.06em' },
  table: { width: '100%', borderCollapse: 'collapse', fontSize: '0.85rem' },
  th: { background: '#1e293b', color: '#64748b', fontWeight: 700, fontSize: '0.7rem', textTransform: 'uppercase', letterSpacing: '0.08em', padding: '0.6rem 1rem', textAlign: 'left', whiteSpace: 'nowrap' },
  tr: { borderBottom: '1px solid #1e293b' },
  td: { padding: '0.65rem 1rem', color: '#94a3b8', verticalAlign: 'middle' },
};

export default Dashboard;
