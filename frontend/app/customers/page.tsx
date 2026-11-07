'use client';

import { useState, useEffect, useCallback, useRef } from 'react';
import { useRouter } from 'next/navigation';
import { useIsMobile } from '@/hooks/useIsMobile';
import AppLayout from '@/components/AppLayout';
import Topbar from '@/components/Topbar';
import { api } from '@/lib/api';
import { useToast } from '@/components/Toast';
import { formatCOP } from '@/lib/format';
import { useBusinessConfig } from '@/lib/business-config-context';
import { buildCollectionWhatsAppUrl } from '@/lib/whatsapp';
import ConfirmDialog from '@/components/ConfirmDialog';
import LoadingSpinner from '@/components/LoadingSpinner';
import EmptyState from '@/components/EmptyState';
import AbonoModal from '@/components/AbonoModal';
import { customerBalanceLabel } from '@/lib/abono';

// ─── Types ───────────────────────────────────────────────────────────────────

interface Customer {
  id: number;
  name: string;
  phone: string;
  address: string;
  notes: string;
  credit_limit: number | null;
  created_at: string;
}

interface CustomerWithBalance extends Customer {
  balance: number;
  oldest_debt_at: string | null;
}

interface CustomersResponse {
  customers: CustomerWithBalance[] | null;
  total_portfolio: number;
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

function daysOverdue(iso: string | null): number {
  if (!iso) return 0;
  return Math.floor((Date.now() - new Date(iso).getTime()) / 86400000);
}

function agingColor(days: number): string {
  if (days > 14) return 'var(--danger)';
  if (days >= 7) return '#ca8a04';
  return 'var(--text-muted)';
}

function agingBg(days: number): string {
  if (days > 14) return 'rgba(239,68,68,0.1)';
  if (days >= 7) return 'rgba(202,138,4,0.1)';
  return 'var(--surface-3)';
}

// ─── Icons ───────────────────────────────────────────────────────────────────

const IconPlus = () => (
  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
    <line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/>
  </svg>
);
const IconSearch = () => (
  <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
    <circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/>
  </svg>
);
const IconEdit = () => (
  <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
    <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/>
    <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/>
  </svg>
);
const IconTrash = () => (
  <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
    <polyline points="3 6 5 6 21 6"/><path d="M19 6l-1 14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2L5 6"/>
    <path d="M10 11v6"/><path d="M14 11v6"/><path d="M9 6V4h6v2"/>
  </svg>
);
const IconX = () => (
  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
    <line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/>
  </svg>
);
const IconWhatsApp = () => (
  <svg width="13" height="13" viewBox="0 0 24 24" fill="currentColor">
    <path d="M17.472 14.382c-.297-.149-1.758-.867-2.03-.967-.273-.099-.471-.148-.67.15-.197.297-.767.966-.94 1.164-.173.199-.347.223-.644.075-.297-.15-1.255-.463-2.39-1.475-.883-.788-1.48-1.761-1.653-2.059-.173-.297-.018-.458.13-.606.134-.133.298-.347.446-.52.149-.174.198-.298.298-.497.099-.198.05-.371-.025-.52-.075-.149-.669-1.612-.916-2.207-.242-.579-.487-.5-.669-.51-.173-.008-.371-.01-.57-.01-.198 0-.52.074-.792.372-.272.297-1.04 1.016-1.04 2.479 0 1.462 1.065 2.875 1.213 3.074.149.198 2.096 3.2 5.077 4.487.709.306 1.262.489 1.694.625.712.227 1.36.195 1.871.118.571-.085 1.758-.719 2.006-1.413.248-.694.248-1.289.173-1.413-.074-.124-.272-.198-.57-.347m-5.421 7.403h-.004a9.87 9.87 0 01-5.031-1.378l-.361-.214-3.741.982.998-3.648-.235-.374a9.86 9.86 0 01-1.51-5.26c.001-5.45 4.436-9.884 9.888-9.884 2.64 0 5.122 1.03 6.988 2.898a9.825 9.825 0 012.893 6.994c-.003 5.45-4.437 9.884-9.885 9.884m8.413-18.297A11.815 11.815 0 0012.05 0C5.495 0 .16 5.335.157 11.892c0 2.096.547 4.142 1.588 5.945L.057 24l6.305-1.654a11.882 11.882 0 005.683 1.448h.005c6.554 0 11.89-5.335 11.893-11.893a11.821 11.821 0 00-3.48-8.413z"/>
  </svg>
);
const IconArrowDown = () => (
  <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
    <line x1="12" y1="5" x2="12" y2="19"/><polyline points="19 12 12 19 5 12"/>
  </svg>
);
const IconArrowUp = () => (
  <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
    <line x1="12" y1="19" x2="12" y2="5"/><polyline points="5 12 12 5 19 12"/>
  </svg>
);
const IconUsers = () => (
  <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
    <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/>
    <path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/>
  </svg>
);

// ─── Customer Modal ───────────────────────────────────────────────────────────

interface CustomerModalProps {
  initial?: Customer | null;
  onClose: () => void;
  onSaved: () => void;
}

function CustomerModal({ initial, onClose, onSaved }: CustomerModalProps) {
  const { toast } = useToast();
  const [name, setName] = useState(initial?.name ?? '');
  const [phone, setPhone] = useState(initial?.phone ?? '');
  const [address, setAddress] = useState(initial?.address ?? '');
  const [notes, setNotes] = useState(initial?.notes ?? '');
  const [creditLimitInput, setCreditLimitInput] = useState(
    initial?.credit_limit != null ? initial.credit_limit.toLocaleString('es-CO') : ''
  );
  const [loading, setLoading] = useState(false);
  const nameRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    nameRef.current?.focus();
  }, []);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!name.trim()) return;
    setLoading(true);
    const digits = creditLimitInput.replace(/\./g, '').replace(/,/g, '');
    const credit_limit = digits === '' ? null : parseInt(digits, 10) || null;
    try {
      if (initial) {
        await api.put(`/customers/${initial.id}`, { name, phone, address, notes, credit_limit });
        toast('Cliente actualizado', 'success');
      } else {
        await api.post('/customers', { name, phone, address, notes, credit_limit });
        toast('Cliente creado', 'success');
      }
      onSaved();
      onClose();
    } catch (err: unknown) {
      toast((err as Error).message, 'error');
    } finally {
      setLoading(false);
    }
  }

  function handleCreditLimitChange(val: string) {
    const digits = val.replace(/[^0-9]/g, '');
    const num = parseInt(digits || '0', 10);
    if (num === 0) { setCreditLimitInput(''); return; }
    setCreditLimitInput(num.toLocaleString('es-CO'));
  }

  const overlayStyle: React.CSSProperties = {
    position: 'fixed', inset: 0,
    background: 'rgba(0,0,0,0.6)',
    display: 'flex', alignItems: 'center', justifyContent: 'center',
    zIndex: 50,
  };
  const modalStyle: React.CSSProperties = {
    background: 'var(--surface-1)',
    border: '1px solid var(--border-emphasis)',
    borderRadius: '12px',
    padding: '24px',
    width: '400px',
    maxWidth: '90vw',
  };

  return (
    <div style={overlayStyle} onClick={onClose}>
      <div style={modalStyle} onClick={e => e.stopPropagation()}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '20px' }}>
          <h2 style={{ fontSize: '15px', fontWeight: 600, color: 'var(--text-primary)', margin: 0 }}>
            {initial ? 'Editar cliente' : 'Nuevo cliente'}
          </h2>
          <button onClick={onClose} style={{ background: 'none', border: 'none', cursor: 'pointer', color: 'var(--text-muted)', padding: '2px' }}>
            <IconX />
          </button>
        </div>

        <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: '14px' }}>
          <div>
            <label style={{ fontSize: '12px', fontWeight: 600, color: 'var(--text-secondary)', textTransform: 'uppercase', letterSpacing: '0.4px', display: 'block', marginBottom: '6px' }}>
              Nombre *
            </label>
            <input
              ref={nameRef}
              value={name}
              onChange={e => setName(e.target.value)}
              placeholder="Nombre completo"
              required
              style={{ width: '100%', boxSizing: 'border-box' }}
            />
          </div>

          <div>
            <label style={{ fontSize: '12px', fontWeight: 600, color: 'var(--text-secondary)', textTransform: 'uppercase', letterSpacing: '0.4px', display: 'block', marginBottom: '6px' }}>
              Teléfono
            </label>
            <input
              value={phone}
              onChange={e => setPhone(e.target.value)}
              placeholder="Ej: 3001234567"
              style={{ width: '100%', boxSizing: 'border-box' }}
            />
          </div>

          <div>
            <label style={{ fontSize: '12px', fontWeight: 600, color: 'var(--text-secondary)', textTransform: 'uppercase', letterSpacing: '0.4px', display: 'block', marginBottom: '6px' }}>
              Dirección
            </label>
            <input
              value={address}
              onChange={e => setAddress(e.target.value)}
              placeholder="Dirección (opcional)"
              style={{ width: '100%', boxSizing: 'border-box' }}
            />
          </div>

          <div>
            <label style={{ fontSize: '12px', fontWeight: 600, color: 'var(--text-secondary)', textTransform: 'uppercase', letterSpacing: '0.4px', display: 'block', marginBottom: '6px' }}>
              Tope de crédito (COP, opcional)
            </label>
            <div style={{ position: 'relative' }}>
              <span style={{ position: 'absolute', left: '12px', top: '50%', transform: 'translateY(-50%)', color: 'var(--text-muted)', fontSize: '14px', fontWeight: 600 }}>$</span>
              <input
                value={creditLimitInput}
                onChange={e => handleCreditLimitChange(e.target.value)}
                placeholder="Sin límite"
                inputMode="numeric"
                style={{ paddingLeft: '26px', width: '100%', boxSizing: 'border-box' }}
              />
            </div>
            {creditLimitInput && (
              <div style={{ fontSize: '11px', color: 'var(--text-muted)', marginTop: '4px' }}>
                Avisa si la deuda supera este monto. La venta no se bloquea.
              </div>
            )}
          </div>

          <div>
            <label style={{ fontSize: '12px', fontWeight: 600, color: 'var(--text-secondary)', textTransform: 'uppercase', letterSpacing: '0.4px', display: 'block', marginBottom: '6px' }}>
              Notas internas
            </label>
            <textarea
              value={notes}
              onChange={e => setNotes(e.target.value)}
              placeholder='Ej: "no fiar más de $150K", "paga los viernes"'
              maxLength={500}
              rows={3}
              style={{ width: '100%', boxSizing: 'border-box', resize: 'vertical', fontFamily: 'inherit', fontSize: '13px', padding: '8px 12px', borderRadius: '6px', border: '1px solid var(--border)', background: 'var(--surface-1)', color: 'var(--text-primary)' }}
            />
            <div style={{ fontSize: '11px', color: 'var(--text-muted)', textAlign: 'right', marginTop: '2px' }}>
              {notes.length}/500
            </div>
          </div>

          <div style={{ display: 'flex', gap: '8px', justifyContent: 'flex-end', marginTop: '4px' }}>
            <button type="button" onClick={onClose} className="btn btn-sm">
              Cancelar
            </button>
            <button type="submit" disabled={loading || !name.trim()} className="btn btn-sm btn-accent">
              {loading ? 'Guardando...' : (initial ? 'Guardar cambios' : 'Crear cliente')}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

// ─── Main Page ────────────────────────────────────────────────────────────────

type SortKey = 'name' | 'balance' | 'oldest_debt';

function SortIcon({ k, sortKey, sortAsc }: { k: SortKey; sortKey: SortKey; sortAsc: boolean }) {
  if (sortKey !== k) return <span style={{ opacity: 0.3 }}><IconArrowDown /></span>;
  return sortAsc ? <IconArrowUp /> : <IconArrowDown />;
}

export default function CustomersPage() {
  const { toast } = useToast();
  const router = useRouter();
  const isMobile = useIsMobile();
  const [customers, setCustomers] = useState<CustomerWithBalance[]>([]);
  const [totalPortfolio, setTotalPortfolio] = useState(0);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');
  const [sortKey, setSortKey] = useState<SortKey>('name');
  const [sortAsc, setSortAsc] = useState(true);
  const [showModal, setShowModal] = useState(false);
  const [editing, setEditing] = useState<Customer | null>(null);
  const [confirmTarget, setConfirmTarget] = useState<CustomerWithBalance | null>(null);
  const [abonoTarget, setAbonoTarget] = useState<CustomerWithBalance | null>(null);
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Fetches the list without touching `loading`, so a refresh after an abono
  // updates rows in place instead of replacing the table by the spinner (US-084).
  const fetchCustomers = useCallback(async (q: string) => {
    try {
      const res = await api.get<CustomersResponse>(`/customers${q ? `?q=${encodeURIComponent(q)}` : ''}`);
      setCustomers(res.customers ?? []);
      setTotalPortfolio(res.total_portfolio ?? 0);
    } catch (err: unknown) {
      toast((err as Error).message, 'error');
    }
  }, [toast]);

  const load = useCallback(async (q: string) => {
    setLoading(true);
    try {
      await fetchCustomers(q);
    } finally {
      setLoading(false);
    }
  }, [fetchCustomers]);

  useEffect(() => {
    queueMicrotask(() => load(''));
  }, [load]);

  function handleSearchChange(val: string) {
    setSearch(val);
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => load(val), 250);
  }

  function toggleSort(key: SortKey) {
    if (sortKey === key) {
      setSortAsc(a => !a);
    } else {
      setSortKey(key);
      setSortAsc(key === 'name');
    }
  }

  const sorted = [...customers].sort((a, b) => {
    let cmp = 0;
    if (sortKey === 'name') cmp = a.name.localeCompare(b.name);
    else if (sortKey === 'balance') cmp = a.balance - b.balance;
    else if (sortKey === 'oldest_debt') {
      const ta = a.oldest_debt_at ? new Date(a.oldest_debt_at).getTime() : Infinity;
      const tb = b.oldest_debt_at ? new Date(b.oldest_debt_at).getTime() : Infinity;
      cmp = ta - tb;
    }
    return sortAsc ? cmp : -cmp;
  });

  const maxBalance = Math.max(...customers.map(c => c.balance), 1);

  async function handleDelete(c: CustomerWithBalance) {
    if (c.balance > 0) {
      toast('No se puede eliminar un cliente con saldo pendiente', 'error');
      return;
    }
    setConfirmTarget(c);
  }

  async function confirmDelete() {
    if (!confirmTarget) return;
    try {
      await api.delete(`/customers/${confirmTarget.id}`);
      toast('Cliente eliminado', 'success');
      load(search);
    } catch (err: unknown) {
      toast((err as Error).message, 'error');
    } finally {
      setConfirmTarget(null);
    }
  }

  return (
    <AppLayout>
      <div style={{ display: 'flex', flexDirection: 'column', height: '100%', overflow: 'hidden' }}>
        <Topbar
          title="Clientes · Cartera"
          subtitle={loading ? 'Cargando...' : `${formatCOP(totalPortfolio)} por cobrar en total`}
          actions={
            <div style={{ display: 'flex', gap: '8px' }}>
              <button className="btn btn-sm">Exportar</button>
              <button className="btn btn-sm btn-accent" onClick={() => { setEditing(null); setShowModal(true); }}>
                <IconPlus /> Nuevo cliente
              </button>
            </div>
          }
        />

        <div style={{ flex: 1, overflow: 'hidden', display: 'flex', flexDirection: 'column' }}>
          {/* Toolbar */}
          <div style={{
            display: 'flex', gap: '8px', padding: isMobile ? '12px 12px 10px' : '16px 24px 12px',
            borderBottom: '1px solid var(--border)',
            alignItems: 'center',
          }}>
            <div style={{ flex: 1, position: 'relative' }}>
              <span style={{ position: 'absolute', left: '10px', top: '50%', transform: 'translateY(-50%)', color: 'var(--text-muted)', pointerEvents: 'none' }}>
                <IconSearch />
              </span>
              <input
                value={search}
                onChange={e => handleSearchChange(e.target.value)}
                placeholder="Buscar por nombre o teléfono..."
                style={{ width: '100%', boxSizing: 'border-box', paddingLeft: '34px' }}
              />
            </div>
            <button
              className={`btn btn-sm${sortKey === 'balance' ? ' btn-accent' : ''}`}
              onClick={() => toggleSort('balance')}
              style={{ display: 'flex', alignItems: 'center', gap: '5px' }}
            >
              <SortIcon k="balance" sortKey={sortKey} sortAsc={sortAsc} /> Ordenar por deuda
            </button>
          </div>

          {/* Table */}
          <div style={{ flex: 1, overflowY: 'auto', padding: isMobile ? '0 8px 16px' : '0 24px 24px' }}>
            {loading ? (
              <LoadingSpinner message="Cargando clientes..." />
            ) : sorted.length === 0 ? (
              <>
                <EmptyState
                  icon={<IconUsers />}
                  title={search ? 'No se encontraron clientes con esa búsqueda' : 'No hay clientes registrados'}
                  description={!search ? 'Agrega tu primer cliente para comenzar.' : undefined}
                />
                {!search && (
                  <div style={{ display: 'flex', justifyContent: 'center' }}>
                    <button className="btn btn-sm btn-accent" onClick={() => { setEditing(null); setShowModal(true); }}>
                      <IconPlus /> Agregar primer cliente
                    </button>
                  </div>
                )}
              </>
            ) : (
              <div className="table-wrap" style={{ marginTop: '12px' }}>
                <table style={{ width: '100%', borderCollapse: 'collapse' }}>
                  <thead>
                    <tr>
                      <th
                        style={{ textAlign: 'left', cursor: 'pointer', userSelect: 'none' }}
                        onClick={() => toggleSort('name')}
                      >
                        <span style={{ display: 'inline-flex', alignItems: 'center', gap: '4px' }}>
                          Cliente <SortIcon k="name" sortKey={sortKey} sortAsc={sortAsc} />
                        </span>
                      </th>
                      {!isMobile && <th style={{ textAlign: 'left' }}>Teléfono</th>}
                      {!isMobile && (
                        <th
                          style={{ textAlign: 'left', cursor: 'pointer', userSelect: 'none' }}
                          onClick={() => toggleSort('oldest_debt')}
                        >
                          <span style={{ display: 'inline-flex', alignItems: 'center', gap: '4px' }}>
                            Deuda más antigua <SortIcon k="oldest_debt" sortKey={sortKey} sortAsc={sortAsc} />
                          </span>
                        </th>
                      )}
                      <th
                        style={{ textAlign: 'right', cursor: 'pointer', userSelect: 'none' }}
                        onClick={() => toggleSort('balance')}
                      >
                        <span style={{ display: 'inline-flex', alignItems: 'center', gap: '4px', justifyContent: 'flex-end' }}>
                          Saldo <SortIcon k="balance" sortKey={sortKey} sortAsc={sortAsc} />
                        </span>
                      </th>
                      <th style={{ textAlign: 'right' }}>Acciones</th>
                    </tr>
                  </thead>
                  <tbody>
                    {sorted.map(c => (
                      <CustomerRow
                        key={c.id}
                        customer={c}
                        maxBalance={maxBalance}
                        isMobile={isMobile}
                        onEdit={() => { setEditing(c); setShowModal(true); }}
                        onDelete={() => handleDelete(c)}
                        onOpen={() => router.push(`/customers/${c.id}`)}
                        onAbonar={() => setAbonoTarget(c)}
                      />
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        </div>
      </div>

      {showModal && (
        <CustomerModal
          initial={editing}
          onClose={() => setShowModal(false)}
          onSaved={() => load(search)}
        />
      )}

      {abonoTarget && (
        <AbonoModal
          customerId={abonoTarget.id}
          customerName={abonoTarget.name}
          balance={abonoTarget.balance}
          onClose={() => setAbonoTarget(null)}
          onSaved={() => fetchCustomers(search)}
        />
      )}

      <ConfirmDialog
        open={confirmTarget !== null}
        title="Eliminar cliente"
        message={`¿Eliminar a ${confirmTarget?.name}? Esta acción no se puede deshacer.`}
        confirmLabel="Eliminar"
        danger
        onConfirm={confirmDelete}
        onCancel={() => setConfirmTarget(null)}
      />
    </AppLayout>
  );
}

// ─── Customer Row ─────────────────────────────────────────────────────────────

interface CustomerRowProps {
  customer: CustomerWithBalance;
  maxBalance: number;
  isMobile: boolean;
  onEdit: () => void;
  onDelete: () => void;
  onOpen: () => void;
  onAbonar: () => void;
}

// Rendered inside AppLayout (via CustomersPage), so useBusinessConfig sees the provider.
function CustomerRow({ customer: c, maxBalance, isMobile, onEdit, onDelete, onOpen, onAbonar }: CustomerRowProps) {
  const { config: bizConfig } = useBusinessConfig();
  const hasDebt = c.balance > 0;
  const debtPct = hasDebt ? Math.min((c.balance / maxBalance) * 100, 100) : 0;
  const balanceLabel = customerBalanceLabel(c.balance);

  return (
    <tr className="customer-row" style={{ cursor: 'pointer' }} onClick={onOpen}>
      <td style={{ verticalAlign: 'top', paddingTop: '12px', paddingBottom: '12px' }}>
        <div style={{ fontWeight: 600, color: 'var(--text-primary)', fontSize: '13px', marginBottom: '2px' }}>
          {c.name}
        </div>
        {c.address && (
          <div style={{ fontSize: '11px', color: 'var(--text-muted)', marginBottom: '6px' }}>
            {c.address}
          </div>
        )}
        {/* Debt bar */}
        <div style={{
          width: '140px', height: '4px',
          background: 'var(--surface-3)',
          borderRadius: '4px',
          overflow: 'hidden',
        }}>
          <div
            className="debt-bar-fill"
            style={{
              width: `${debtPct}%`,
              height: '100%',
              background: hasDebt ? 'var(--danger)' : 'var(--success)',
              borderRadius: '4px',
              transition: 'width 0.3s ease',
            }}
          />
        </div>
      </td>
      {!isMobile && (
        <td style={{ fontSize: '13px', color: 'var(--text-secondary)', verticalAlign: 'top', paddingTop: '12px' }}>
          {c.phone || '—'}
        </td>
      )}
      {!isMobile && (
        <td style={{ verticalAlign: 'top', paddingTop: '12px' }}>
          {hasDebt ? (() => {
            const days = daysOverdue(c.oldest_debt_at);
            return (
              <span style={{
                display: 'inline-block',
                padding: '2px 8px',
                borderRadius: '12px',
                fontSize: '12px',
                fontWeight: 600,
                color: agingColor(days),
                background: agingBg(days),
              }}>
                {days === 0 ? 'hoy' : `${days}d`}
              </span>
            );
          })() : '—'}
        </td>
      )}
      <td style={{ textAlign: 'right', verticalAlign: 'top', paddingTop: '12px' }}>
        {hasDebt ? (
          <span
            className="danger-amount"
            style={{
              fontWeight: 600,
              color: balanceLabel.color,
              fontVariantNumeric: 'tabular-nums',
              fontFeatureSettings: '"tnum"',
            }}
          >
            {balanceLabel.text}
          </span>
        ) : (
          <span style={{ color: balanceLabel.color, fontSize: '12px', fontWeight: 500 }}>{balanceLabel.text}</span>
        )}
      </td>
      <td style={{ textAlign: 'right', verticalAlign: 'top', paddingTop: '10px' }}>
        <div style={{ display: 'flex', gap: '6px', justifyContent: 'flex-end' }}>
          {hasDebt && c.phone && (
            <a
              href={buildCollectionWhatsAppUrl(c.phone, c.name, c.balance, bizConfig.name)}
              target="_blank"
              rel="noopener noreferrer"
              className="btn btn-sm"
              title="Cobrar por WhatsApp"
              style={{ display: 'flex', alignItems: 'center', gap: '4px', color: '#16a34a' }}
              onClick={e => e.stopPropagation()}
            >
              <IconWhatsApp />
            </a>
          )}
          {hasDebt && (
            <button
              className="btn btn-sm btn-accent"
              title="Registrar abono"
              onClick={e => { e.stopPropagation(); onAbonar(); }}
            >
              Abonar
            </button>
          )}
          <button
            className="btn btn-sm"
            title="Editar cliente"
            onClick={e => { e.stopPropagation(); onEdit(); }}
            style={{ display: 'flex', alignItems: 'center', gap: '4px' }}
          >
            <IconEdit />{!isMobile && ' Editar'}
          </button>
          <button
            className="btn btn-sm"
            title={c.balance > 0 ? 'No se puede eliminar con deuda' : 'Eliminar cliente'}
            onClick={e => { e.stopPropagation(); onDelete(); }}
            disabled={c.balance > 0}
            style={{
              display: 'flex', alignItems: 'center', gap: '4px',
              opacity: c.balance > 0 ? 0.4 : 1,
            }}
          >
            <IconTrash />
          </button>
        </div>
      </td>
    </tr>
  );
}
