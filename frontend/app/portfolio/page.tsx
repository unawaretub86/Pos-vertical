'use client';

import { useState, useEffect, useCallback } from 'react';
import { useRouter } from 'next/navigation';
import { useIsMobile } from '@/hooks/useIsMobile';
import AppLayout from '@/components/AppLayout';
import Topbar from '@/components/Topbar';
import { api } from '@/lib/api';
import { formatCOP } from '@/lib/format';
import LoadingSpinner from '@/components/LoadingSpinner';
import EmptyState from '@/components/EmptyState';

// ─── Types ───────────────────────────────────────────────────────────────────

interface PortfolioEntry {
  id: number;
  name: string;
  phone: string;
  balance: number;
  oldest_debt_at: string | null;
}

type SortKey = 'balance' | 'oldest';

// ─── Helpers ─────────────────────────────────────────────────────────────────

function relativeDate(iso: string | null): string {
  if (!iso) return '—';
  const diff = Date.now() - new Date(iso).getTime();
  const days = Math.floor(diff / 86400000);
  if (days === 0) return 'hoy';
  if (days === 1) return 'ayer';
  if (days < 30) return `hace ${days} días`;
  if (days < 365) return `hace ${Math.floor(days / 30)} meses`;
  return `hace ${Math.floor(days / 365)} año(s)`;
}

// ─── Icons ───────────────────────────────────────────────────────────────────

const IconCreditCard = () => (
  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
    <rect x="1" y="4" width="22" height="16" rx="2" ry="2"/><line x1="1" y1="10" x2="23" y2="10"/>
  </svg>
);

const IconArrowDown = () => (
  <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
    <line x1="12" y1="5" x2="12" y2="19"/><polyline points="19 12 12 19 5 12"/>
  </svg>
);

const IconArrowUp = () => (
  <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
    <line x1="12" y1="19" x2="12" y2="5"/><polyline points="5 12 12 5 19 12"/>
  </svg>
);

const IconUser = () => (
  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
    <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/>
  </svg>
);

// ─── Component ────────────────────────────────────────────────────────────────

export default function PortfolioPage() {
  const router = useRouter();
  const isMobile = useIsMobile();
  const [entries, setEntries] = useState<PortfolioEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [sortKey, setSortKey] = useState<SortKey>('balance');
  const [sortAsc, setSortAsc] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const data = await api.get<PortfolioEntry[]>('/reports/portfolio');
      setEntries(data ?? []);
    } catch {
      setEntries([]);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { load(); }, [load]);

  const sorted = [...entries].sort((a, b) => {
    let diff = 0;
    if (sortKey === 'balance') {
      diff = b.balance - a.balance;
    } else {
      const ta = a.oldest_debt_at ? new Date(a.oldest_debt_at).getTime() : 0;
      const tb = b.oldest_debt_at ? new Date(b.oldest_debt_at).getTime() : 0;
      diff = ta - tb; // older first by default
    }
    return sortAsc ? -diff : diff;
  });

  const totalDebt = entries.reduce((s, e) => s + e.balance, 0);

  function toggleSort(key: SortKey) {
    if (sortKey === key) setSortAsc(v => !v);
    else { setSortKey(key); setSortAsc(false); }
  }

  const SortIcon = ({ k }: { k: SortKey }) => {
    if (sortKey !== k) return <span style={{ opacity: 0.3 }}><IconArrowDown /></span>;
    return sortAsc ? <IconArrowUp /> : <IconArrowDown />;
  };

  return (
    <AppLayout>
      <div style={{ display: 'flex', flexDirection: 'column', height: '100%', overflow: 'hidden' }}>
        <Topbar
          title="Cartera"
          subtitle={loading ? 'Cargando…' : `${formatCOP(totalDebt)} por cobrar · ${entries.length} cliente${entries.length !== 1 ? 's' : ''}`}
        />

        <div style={{ flex: 1, overflow: 'auto', padding: isMobile ? '12px 8px' : '24px' }}>
          {loading ? (
            <LoadingSpinner message="Cargando cartera…" />
          ) : entries.length === 0 ? (
            <EmptyState
              icon={<IconCreditCard />}
              title="No hay deudas pendientes"
              description="Todos los clientes están al día."
            />
          ) : (
            <div className="table-wrap">
              <table style={{ width: '100%', borderCollapse: 'collapse' }}>
                <thead>
                  <tr>
                    <th style={{ padding: '10px 16px', textAlign: 'left', fontSize: '11px', fontWeight: 600, textTransform: 'uppercase', letterSpacing: '0.5px', color: 'var(--text-muted)', borderBottom: '1px solid var(--border)' }}>
                      Cliente
                    </th>
                    {!isMobile && (
                      <th style={{ padding: '10px 16px', textAlign: 'left', fontSize: '11px', fontWeight: 600, textTransform: 'uppercase', letterSpacing: '0.5px', color: 'var(--text-muted)', borderBottom: '1px solid var(--border)' }}>
                        Teléfono
                      </th>
                    )}
                    {!isMobile && (
                      <th
                        onClick={() => toggleSort('oldest')}
                        style={{ padding: '10px 16px', textAlign: 'left', fontSize: '11px', fontWeight: 600, textTransform: 'uppercase', letterSpacing: '0.5px', color: sortKey === 'oldest' ? 'var(--accent)' : 'var(--text-muted)', borderBottom: '1px solid var(--border)', cursor: 'pointer', userSelect: 'none', whiteSpace: 'nowrap' }}
                      >
                        <span style={{ display: 'inline-flex', alignItems: 'center', gap: '4px' }}>
                          Deuda más antigua <SortIcon k="oldest" />
                        </span>
                      </th>
                    )}
                    <th
                      onClick={() => toggleSort('balance')}
                      style={{ padding: '10px 16px', textAlign: 'right', fontSize: '11px', fontWeight: 600, textTransform: 'uppercase', letterSpacing: '0.5px', color: sortKey === 'balance' ? 'var(--accent)' : 'var(--text-muted)', borderBottom: '1px solid var(--border)', cursor: 'pointer', userSelect: 'none', whiteSpace: 'nowrap' }}
                    >
                      <span style={{ display: 'inline-flex', alignItems: 'center', gap: '4px' }}>
                        Saldo <SortIcon k="balance" />
                      </span>
                    </th>
                    <th style={{ padding: '10px 16px', textAlign: 'right', fontSize: '11px', fontWeight: 600, textTransform: 'uppercase', letterSpacing: '0.5px', color: 'var(--text-muted)', borderBottom: '1px solid var(--border)' }}>
                      Acciones
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {sorted.map((entry, i) => (
                    <tr
                      key={entry.id}
                      style={{ borderBottom: i < sorted.length - 1 ? '1px solid var(--border)' : 'none', cursor: 'pointer' }}
                      onClick={() => router.push(`/customers/${entry.id}`)}
                      onMouseEnter={e => (e.currentTarget as HTMLTableRowElement).style.background = 'var(--surface-2)'}
                      onMouseLeave={e => (e.currentTarget as HTMLTableRowElement).style.background = 'transparent'}
                    >
                      <td style={{ padding: '12px 16px' }}>
                        <div style={{ display: 'flex', flexDirection: 'column', gap: '4px' }}>
                          <span style={{ fontSize: '13px', fontWeight: 600, color: 'var(--text-primary)' }}>
                            {entry.name}
                          </span>
                          {/* Debt bar */}
                          <div style={{ width: '120px', height: '4px', background: 'var(--surface-3)', borderRadius: '4px', overflow: 'hidden' }}>
                            <div style={{
                              height: '100%',
                              background: 'var(--danger)',
                              borderRadius: '4px',
                              width: `${Math.min(100, (entry.balance / Math.max(...entries.map(e => e.balance))) * 100)}%`,
                              transition: '0.3s ease',
                            }} />
                          </div>
                        </div>
                      </td>
                      {!isMobile && (
                        <td style={{ padding: '12px 16px', fontSize: '13px', color: 'var(--text-secondary)' }}>
                          {entry.phone || '—'}
                        </td>
                      )}
                      {!isMobile && (
                        <td style={{ padding: '12px 16px', fontSize: '12px', color: 'var(--text-muted)' }}>
                          {relativeDate(entry.oldest_debt_at)}
                        </td>
                      )}
                      <td style={{ padding: '12px 16px', textAlign: 'right' }}>
                        <span style={{ fontSize: '14px', fontWeight: 600, color: 'var(--danger)', fontVariantNumeric: 'tabular-nums' }}>
                          {formatCOP(entry.balance)}
                        </span>
                      </td>
                      <td style={{ padding: '12px 16px', textAlign: 'right' }}>
                        <div style={{ display: 'flex', gap: '6px', justifyContent: 'flex-end' }} onClick={e => e.stopPropagation()}>
                          <button
                            className="btn btn-sm"
                            onClick={() => router.push(`/customers/${entry.id}`)}
                          >
                            Ver
                          </button>
                          <button
                            className="btn btn-sm btn-accent"
                            onClick={() => router.push(`/customers/${entry.id}`)}
                          >
                            Abonar
                          </button>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>

              {/* Total row */}
              <div style={{
                padding: '12px 16px',
                borderTop: '1px solid var(--border)',
                display: 'flex',
                justifyContent: 'flex-end',
                alignItems: 'center',
                gap: '8px',
              }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: '6px', color: 'var(--text-muted)' }}>
                  <IconUser />
                  <span style={{ fontSize: '12px' }}>{entries.length} cliente{entries.length !== 1 ? 's' : ''}</span>
                </div>
                <span style={{ color: 'var(--border)', margin: '0 4px' }}>·</span>
                <span style={{ fontSize: '13px', color: 'var(--text-secondary)' }}>Total por cobrar:</span>
                <span style={{ fontSize: '16px', fontWeight: 700, color: 'var(--danger)', fontVariantNumeric: 'tabular-nums', letterSpacing: '-0.5px' }}>
                  {formatCOP(totalDebt)}
                </span>
              </div>
            </div>
          )}
        </div>
      </div>
    </AppLayout>
  );
}
