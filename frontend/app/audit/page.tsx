'use client';

import { useState, useEffect, useCallback, useMemo } from 'react';
import { useRouter } from 'next/navigation';
import { getUser } from '@/lib/auth';
import { useIsMobile } from '@/hooks/useIsMobile';
import AppLayout from '@/components/AppLayout';
import Topbar from '@/components/Topbar';
import LoadingSpinner from '@/components/LoadingSpinner';
import EmptyState from '@/components/EmptyState';
import { api } from '@/lib/api';
import { formatCOP } from '@/lib/format';

// ─── Types ───────────────────────────────────────────────────────────────────

interface AuditEntry {
  id: number;
  user_id: number;
  username: string;
  action: string;
  entity: string;
  entity_id: number;
  details: string;
  created_at: string;
}

interface User {
  id: number;
  username: string;
  active: boolean;
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

const ACTION_LABELS: Record<string, string> = {
  void: 'Anulación',
  discount: 'Descuento',
  cash_out: 'Salida de caja',
  product_price_change: 'Cambio de precio',
  price_override: 'Precio modificado en venta',
};

function formatDateTime(iso: string): string {
  return new Date(iso).toLocaleString('es-CO', {
    day: '2-digit', month: 'short', year: 'numeric', hour: '2-digit', minute: '2-digit',
  });
}

function describeDetails(action: string, entity: string, entityID: number, details: string): string {
  let parsed: Record<string, unknown> = {};
  try { parsed = JSON.parse(details || '{}'); } catch { /* ignore malformed details */ }

  switch (action) {
    case 'void':
      return `Venta #${entityID} anulada${typeof parsed.total === 'number' ? ` · ${formatCOP(parsed.total)}` : ''}`;
    case 'discount':
      return `Descuento de ${formatCOP(Number(parsed.amount ?? 0))} en venta #${entityID}`;
    case 'cash_out':
      return `Salida de ${formatCOP(Number(parsed.amount ?? 0))}${parsed.description ? ` — ${parsed.description}` : ''}`;
    case 'product_price_change': {
      const field = parsed.field === 'cost' ? 'costo' : 'precio de venta';
      return `Producto #${entityID}: ${field} de ${formatCOP(Number(parsed.from ?? 0))} a ${formatCOP(Number(parsed.to ?? 0))}`;
    }
    case 'price_override':
      return `Precio de venta #${entityID} modificado a ${formatCOP(Number(parsed.override_price ?? 0))} (catálogo: ${formatCOP(Number(parsed.original_price ?? 0))})`;
    default:
      return `${entity} #${entityID}`;
  }
}

// ─── Icons ───────────────────────────────────────────────────────────────────

const IconShield = () => (
  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
    <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" />
  </svg>
);

// ─── Component ────────────────────────────────────────────────────────────────

export default function AuditPage() {
  const router = useRouter();
  const isMobile = useIsMobile();
  // getUser() re-parses localStorage into a new object on every call — memoize
  // it so it doesn't change identity on every render, which would otherwise
  // re-trigger the effects below on every load() completion (infinite loop).
  const user = useMemo(() => getUser(), []);

  const [entries, setEntries] = useState<AuditEntry[]>([]);
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(true);
  const [userID, setUserID] = useState('');
  const [action, setAction] = useState('');
  const [from, setFrom] = useState('');
  const [to, setTo] = useState('');

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const params = new URLSearchParams();
      if (userID) params.set('user_id', userID);
      if (action) params.set('action', action);
      if (from) params.set('from', new Date(`${from}T00:00:00`).toISOString());
      if (to) params.set('to', new Date(`${to}T23:59:59`).toISOString());
      const qs = params.toString();
      const data = await api.get<AuditEntry[]>(`/audit${qs ? `?${qs}` : ''}`);
      setEntries(data ?? []);
    } catch {
      setEntries([]);
    } finally {
      setLoading(false);
    }
  }, [userID, action, from, to]);

  useEffect(() => {
    if (!user) return;
    if (user.role !== 'admin') {
      router.replace('/pos');
      return;
    }
    api.get<User[]>('/users').then(setUsers).catch(() => {});
  }, [user, router]);

  useEffect(() => { if (user?.role === 'admin') queueMicrotask(load); }, [user, load]);

  if (!user || user.role !== 'admin') return null;

  const selectStyle: React.CSSProperties = {
    padding: '7px 10px',
    borderRadius: '8px',
    border: '1px solid var(--border)',
    background: 'var(--surface-1)',
    color: 'var(--text-primary)',
    fontSize: '13px',
  };

  return (
    <AppLayout>
      <div style={{ display: 'flex', flexDirection: 'column', height: '100%', overflow: 'hidden' }}>
        <Topbar
          title="Auditoría"
          subtitle={loading ? 'Cargando…' : `${entries.length} evento${entries.length !== 1 ? 's' : ''}`}
        />

        <div style={{ flex: 1, overflow: 'auto', padding: isMobile ? '12px 8px' : '24px' }}>
          {/* Filters */}
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: '8px', marginBottom: '16px' }}>
            <select value={userID} onChange={e => setUserID(e.target.value)} style={selectStyle}>
              <option value="">Todos los usuarios</option>
              {users.map(u => (
                <option key={u.id} value={u.id}>{u.username}{!u.active ? ' (inactivo)' : ''}</option>
              ))}
            </select>
            <select value={action} onChange={e => setAction(e.target.value)} style={selectStyle}>
              <option value="">Todas las acciones</option>
              <option value="void">Anulación</option>
              <option value="discount">Descuento</option>
              <option value="cash_out">Salida de caja</option>
              <option value="product_price_change">Cambio de precio</option>
              <option value="price_override">Precio en venta</option>
            </select>
            <input type="date" value={from} onChange={e => setFrom(e.target.value)} style={selectStyle} />
            <input type="date" value={to} onChange={e => setTo(e.target.value)} style={selectStyle} />
          </div>

          {loading ? (
            <LoadingSpinner message="Cargando auditoría…" />
          ) : entries.length === 0 ? (
            <EmptyState
              icon={<IconShield />}
              title="Sin eventos en este rango"
              description="No hay eventos de auditoría con los filtros aplicados."
            />
          ) : (
            <div className="table-wrap">
              <table style={{ width: '100%', borderCollapse: 'collapse' }}>
                <thead>
                  <tr>
                    <th style={thStyle}>Fecha/hora</th>
                    <th style={thStyle}>Usuario</th>
                    <th style={thStyle}>Acción</th>
                    <th style={thStyle}>Entidad</th>
                    <th style={thStyle}>Detalle</th>
                  </tr>
                </thead>
                <tbody>
                  {entries.map((e, i) => (
                    <tr key={e.id} style={{ borderBottom: i < entries.length - 1 ? '1px solid var(--border)' : 'none' }}>
                      <td style={{ ...tdStyle, whiteSpace: 'nowrap', color: 'var(--text-muted)' }}>{formatDateTime(e.created_at)}</td>
                      <td style={{ ...tdStyle, fontWeight: 600 }}>{e.username}</td>
                      <td style={tdStyle}>
                        <span className="badge">{ACTION_LABELS[e.action] ?? e.action}</span>
                      </td>
                      <td style={{ ...tdStyle, color: 'var(--text-muted)' }}>{e.entity} #{e.entity_id}</td>
                      <td style={{ ...tdStyle, color: 'var(--text-secondary)' }}>{describeDetails(e.action, e.entity, e.entity_id, e.details)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      </div>
    </AppLayout>
  );
}

const thStyle: React.CSSProperties = {
  padding: '10px 16px',
  textAlign: 'left',
  fontSize: '11px',
  fontWeight: 600,
  textTransform: 'uppercase',
  letterSpacing: '0.5px',
  color: 'var(--text-muted)',
  borderBottom: '1px solid var(--border)',
  whiteSpace: 'nowrap',
};

const tdStyle: React.CSSProperties = {
  padding: '12px 16px',
  fontSize: '13px',
};
