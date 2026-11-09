'use client';

import { useState, useEffect, useCallback, useRef, useSyncExternalStore } from 'react';
import { useParams, useRouter } from 'next/navigation';
import AppLayout from '@/components/AppLayout';
import Topbar from '@/components/Topbar';
import { api } from '@/lib/api';
import { useToast } from '@/components/Toast';
import { formatCOP } from '@/lib/format';
import { useBusinessConfig } from '@/lib/business-config-context';
import { buildCollectionWhatsAppUrl } from '@/lib/whatsapp';
import { customerBalanceLabel, type AbonoMethod } from '@/lib/abono';
import AbonoModal from '@/components/AbonoModal';
import { resolveRouteParam, parseCustomerId } from '@/lib/routeParam';
import { getUser } from '@/lib/auth';
import {
  parseAmountInput,
  isValidDebtDate,
  todayISODate,
  formatDebtDate,
  MAX_OPENING_BALANCE,
  type OpeningBalance,
} from '@/lib/openingBalance';

// ─── Types ────────────────────────────────────────────────────────────────────

interface Customer {
  id: number;
  name: string;
  phone: string;
  address: string;
  notes: string;
  credit_limit: number | null;
  created_at: string;
}

interface CustomerPayment {
  id: number;
  customer_id: number;
  amount: number;
  method: AbonoMethod;
  register_id: number | null;
  note: string;
  created_at: string;
}

interface CreditSale {
  id: number;
  total: number;
  subtotal: number;
  discount: number;
  tax: number;
  status: string;
  created_at: string;
}

interface CustomerDetail {
  customer: Customer;
  balance: number;
  payments: CustomerPayment[] | null;
  credit_sales: CreditSale[] | null;
  opening_balance: OpeningBalance | null;
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

function formatDate(iso: string): string {
  const d = new Date(iso);
  return d.toLocaleDateString('es-CO', { day: '2-digit', month: 'short', year: 'numeric' });
}

function formatDateTime(iso: string): string {
  const d = new Date(iso);
  return d.toLocaleDateString('es-CO', { day: '2-digit', month: 'short', year: 'numeric' }) +
    ' · ' + d.toLocaleTimeString('es-CO', { hour: '2-digit', minute: '2-digit' });
}

// ─── Icons ────────────────────────────────────────────────────────────────────

const IconArrowLeft = () => (
  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
    <line x1="19" y1="12" x2="5" y2="12"/><polyline points="12 19 5 12 12 5"/>
  </svg>
);
const IconCreditCard = () => (
  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
    <rect x="1" y="4" width="22" height="16" rx="2" ry="2"/><line x1="1" y1="10" x2="23" y2="10"/>
  </svg>
);
const IconReceipt = () => (
  <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
    <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/>
    <polyline points="14 2 14 8 20 8"/>
    <line x1="8" y1="13" x2="16" y2="13"/>
    <line x1="8" y1="17" x2="16" y2="17"/>
  </svg>
);
const IconPayment = () => (
  <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
    <circle cx="12" cy="12" r="10"/>
    <line x1="12" y1="8" x2="12" y2="12"/>
    <line x1="12" y1="16" x2="12.01" y2="16"/>
  </svg>
);
const IconWhatsApp = () => (
  <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor">
    <path d="M17.472 14.382c-.297-.149-1.758-.867-2.03-.967-.273-.099-.471-.148-.67.15-.197.297-.767.966-.94 1.164-.173.199-.347.223-.644.075-.297-.15-1.255-.463-2.39-1.475-.883-.788-1.48-1.761-1.653-2.059-.173-.297-.018-.458.13-.606.134-.133.298-.347.446-.52.149-.174.198-.298.298-.497.099-.198.05-.371-.025-.52-.075-.149-.669-1.612-.916-2.207-.242-.579-.487-.5-.669-.51-.173-.008-.371-.01-.57-.01-.198 0-.52.074-.792.372-.272.297-1.04 1.016-1.04 2.479 0 1.462 1.065 2.875 1.213 3.074.149.198 2.096 3.2 5.077 4.487.709.306 1.262.489 1.694.625.712.227 1.36.195 1.871.118.571-.085 1.758-.719 2.006-1.413.248-.694.248-1.289.173-1.413-.074-.124-.272-.198-.57-.347m-5.421 7.403h-.004a9.87 9.87 0 01-5.031-1.378l-.361-.214-3.741.982.998-3.648-.235-.374a9.86 9.86 0 01-1.51-5.26c.001-5.45 4.436-9.884 9.888-9.884 2.64 0 5.122 1.03 6.988 2.898a9.825 9.825 0 012.893 6.994c-.003 5.45-4.437 9.884-9.885 9.884m8.413-18.297A11.815 11.815 0 0012.05 0C5.495 0 .16 5.335.157 11.892c0 2.096.547 4.142 1.588 5.945L.057 24l6.305-1.654a11.882 11.882 0 005.683 1.448h.005c6.554 0 11.89-5.335 11.893-11.893a11.821 11.821 0 00-3.48-8.413z"/>
  </svg>
);

// ─── Main Page ────────────────────────────────────────────────────────────────

// The browser path is only read on the client after hydration: the server
// snapshot is null, so the first render matches the exported HTML (US-086).
function subscribePath(onChange: () => void) {
  window.addEventListener('popstate', onChange);
  return () => window.removeEventListener('popstate', onChange);
}

// Inner component: must render INSIDE AppLayout so useBusinessConfig sees the provider.
function CustomerDetailContent() {
  const params = useParams();
  const router = useRouter();
  const { toast } = useToast();
  const { config: bizConfig } = useBusinessConfig();
  const pathname = useSyncExternalStore(subscribePath, () => window.location.pathname, () => null);
  const customerId = parseCustomerId(resolveRouteParam(params.id, pathname ?? '', 'customers'));
  // No id from the param and the path is not known yet: keep showing Cargando.
  const resolvingId = customerId === null && pathname === null;

  const [detail, setDetail] = useState<CustomerDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [showPaymentModal, setShowPaymentModal] = useState(false);
  const [showOpeningModal, setShowOpeningModal] = useState(false);
  const [activeTab, setActiveTab] = useState<'sales' | 'payments'>('sales');

  const load = useCallback(async () => {
    if (customerId === null) return;
    setLoading(true);
    try {
      const data = await api.get<CustomerDetail>(`/customers/${customerId}/detail`);
      setDetail(data);
    } catch (err: unknown) {
      toast((err as Error).message, 'error');
    } finally {
      setLoading(false);
    }
  }, [customerId, toast]);

  useEffect(() => {
    if (customerId === null) return;
    queueMicrotask(load);
  }, [customerId, load]);

  if (resolvingId || (customerId !== null && loading)) {
    return (
      <>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '100%', color: 'var(--text-muted)', fontSize: '13px' }}>
          Cargando...
        </div>
      </>
    );
  }

  if (!detail) {
    return (
      <>
        <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', height: '100%', gap: '12px' }}>
          <div style={{ color: 'var(--text-muted)', fontSize: '14px' }}>Cliente no encontrado</div>
          <button className="btn btn-sm" onClick={() => router.push('/customers')}>
            Volver a clientes
          </button>
        </div>
      </>
    );
  }

  const { customer, balance, payments, credit_sales, opening_balance } = detail;
  // Only rendered after the detail loaded on the client, so localStorage is available.
  const isAdmin = getUser()?.role === 'admin';
  const salesList = credit_sales ?? [];
  const balanceLabel = customerBalanceLabel(balance);
  const paymentsList = payments ?? [];

  return (
    <>
      <div style={{ display: 'flex', flexDirection: 'column', height: '100%', overflow: 'hidden' }}>
        <Topbar
          title={customer.name}
          subtitle={customer.phone ? `${customer.phone}${customer.address ? ' · ' + customer.address : ''}` : (customer.address || 'Sin información adicional')}
          actions={
            <div style={{ display: 'flex', gap: '8px' }}>
              <button className="btn btn-sm" onClick={() => router.push('/customers')} style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
                <IconArrowLeft /> Volver
              </button>
              {balance > 0 && (
                <>
                  <button
                    className="btn btn-sm"
                    disabled={!customer.phone}
                    title={!customer.phone ? 'Agrega teléfono al cliente' : 'Cobrar por WhatsApp'}
                    onClick={() => {
                      const url = buildCollectionWhatsAppUrl(customer.phone, customer.name, balance, bizConfig.name);
                      window.open(url, '_blank');
                    }}
                    style={{ display: 'flex', alignItems: 'center', gap: '6px', color: '#25D366', opacity: !customer.phone ? 0.45 : 1 }}
                  >
                    <IconWhatsApp /> Cobrar
                  </button>
                  <button className="btn btn-sm btn-accent" onClick={() => setShowPaymentModal(true)} style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
                    <IconCreditCard /> Registrar abono
                  </button>
                </>
              )}
            </div>
          }
        />

        <div style={{ flex: 1, overflow: 'hidden', display: 'flex', flexDirection: 'column' }}>
          {/* Balance card */}
          <div style={{ padding: '20px 24px 0' }}>
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: '12px', marginBottom: '20px' }}>
              {/* Saldo pendiente */}
              <div style={{
                background: balance > 0 ? 'var(--danger-dim)' : 'var(--success-dim)',
                border: `1px solid ${balance > 0 ? 'var(--danger)' : 'var(--success)'}`,
                borderRadius: '8px', padding: '14px 16px',
              }}>
                <div style={{ fontSize: '11px', fontWeight: 600, color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '0.5px', marginBottom: '6px' }}>
                  Saldo pendiente
                </div>
                <div style={{ fontSize: '22px', fontWeight: 700, color: balanceLabel.color, fontVariantNumeric: 'tabular-nums', letterSpacing: '-0.5px' }}>
                  {balanceLabel.text}
                </div>
              </div>

              {/* Ventas a crédito */}
              <div style={{ background: 'var(--surface-1)', border: '1px solid var(--border)', borderRadius: '8px', padding: '14px 16px' }}>
                <div style={{ fontSize: '11px', fontWeight: 600, color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '0.5px', marginBottom: '6px' }}>
                  Ventas a crédito
                </div>
                <div style={{ fontSize: '22px', fontWeight: 700, color: 'var(--text-primary)', fontVariantNumeric: 'tabular-nums', letterSpacing: '-0.5px' }}>
                  {salesList.length}
                </div>
              </div>

              {/* Total pagado */}
              <div style={{ background: 'var(--surface-1)', border: '1px solid var(--border)', borderRadius: '8px', padding: '14px 16px' }}>
                <div style={{ fontSize: '11px', fontWeight: 600, color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '0.5px', marginBottom: '6px' }}>
                  Total pagado
                </div>
                <div style={{ fontSize: '22px', fontWeight: 700, color: 'var(--success)', fontVariantNumeric: 'tabular-nums', letterSpacing: '-0.5px' }}>
                  {formatCOP(paymentsList.reduce((sum, p) => sum + p.amount, 0))}
                </div>
              </div>
            </div>

            {/* Opening balance (deuda anterior, US-092): visible to every role, editable by admins */}
            {opening_balance ? (
              <div style={{
                background: 'var(--surface-1)',
                border: '1px solid var(--border)',
                borderRadius: '8px',
                padding: '10px 14px',
                marginBottom: '16px',
                display: 'flex',
                flexWrap: 'wrap',
                justifyContent: 'space-between',
                alignItems: 'center',
                gap: '8px',
              }}>
                <div style={{ minWidth: 0 }}>
                  <div style={{ fontSize: '11px', fontWeight: 600, color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '0.5px', marginBottom: '4px' }}>
                    Deuda anterior
                  </div>
                  <div style={{ fontSize: '13px', color: 'var(--text-primary)' }}>
                    <span style={{ fontWeight: 600, fontVariantNumeric: 'tabular-nums' }}>{formatCOP(opening_balance.amount)}</span>
                    {' · desde '}{formatDebtDate(opening_balance.debt_date)}
                  </div>
                  {opening_balance.note && (
                    <div style={{ fontSize: '12px', color: 'var(--text-secondary)', marginTop: '2px', overflowWrap: 'anywhere' }}>
                      {opening_balance.note}
                    </div>
                  )}
                </div>
                {isAdmin && (
                  <button className="btn btn-sm" onClick={() => setShowOpeningModal(true)}>
                    Corregir
                  </button>
                )}
              </div>
            ) : isAdmin && (
              <div style={{ marginBottom: '16px' }}>
                <button className="btn btn-sm" onClick={() => setShowOpeningModal(true)}>
                  Registrar deuda anterior
                </button>
              </div>
            )}

            {/* Notes + credit limit */}
            {(customer.notes || customer.credit_limit != null) && (
              <div style={{ display: 'flex', flexDirection: 'column', gap: '8px', marginBottom: '16px' }}>
                {customer.notes && (
                  <div style={{
                    background: 'var(--accent-dim)',
                    border: '1px solid var(--accent)',
                    borderRadius: '8px',
                    padding: '10px 14px',
                    fontSize: '13px',
                    color: 'var(--text-primary)',
                  }}>
                    <span style={{ fontWeight: 600, color: 'var(--accent)', marginRight: '6px' }}>📝 Nota:</span>
                    {customer.notes}
                  </div>
                )}
                {customer.credit_limit != null && (
                  <div style={{
                    background: 'var(--surface-1)',
                    border: '1px solid var(--border)',
                    borderRadius: '8px',
                    padding: '10px 14px',
                    fontSize: '13px',
                    color: 'var(--text-secondary)',
                    display: 'flex',
                    justifyContent: 'space-between',
                    alignItems: 'center',
                  }}>
                    <span>Tope de crédito</span>
                    <span style={{ fontWeight: 600, color: 'var(--text-primary)', fontVariantNumeric: 'tabular-nums' }}>
                      {formatCOP(customer.credit_limit)}
                    </span>
                  </div>
                )}
              </div>
            )}

            {/* Tabs */}
            <div style={{ display: 'flex', gap: '4px', borderBottom: '1px solid var(--border)', marginBottom: '0' }}>
              {(['sales', 'payments'] as const).map(tab => (
                <button
                  key={tab}
                  onClick={() => setActiveTab(tab)}
                  style={{
                    background: 'none', border: 'none', cursor: 'pointer',
                    padding: '8px 16px',
                    fontSize: '13px', fontWeight: activeTab === tab ? 600 : 400,
                    color: activeTab === tab ? 'var(--accent)' : 'var(--text-secondary)',
                    borderBottom: activeTab === tab ? '2px solid var(--accent)' : '2px solid transparent',
                    marginBottom: '-1px',
                    transition: 'color 150ms ease',
                  }}
                >
                  {tab === 'sales' ? `Ventas a crédito (${salesList.length})` : `Abonos (${paymentsList.length})`}
                </button>
              ))}
            </div>
          </div>

          {/* Tab content */}
          <div style={{ flex: 1, overflowY: 'auto', padding: '0 24px 24px' }}>
            {activeTab === 'sales' ? (
              salesList.length === 0 ? (
                <EmptyState icon={<IconReceipt />} text="No hay ventas a crédito registradas" />
              ) : (
                <div className="table-wrap" style={{ marginTop: '12px' }}>
                  <table style={{ width: '100%', borderCollapse: 'collapse' }}>
                    <thead>
                      <tr>
                        <th style={{ textAlign: 'left' }}>Fecha</th>
                        <th style={{ textAlign: 'right' }}>Subtotal</th>
                        <th style={{ textAlign: 'right' }}>Descuento</th>
                        <th style={{ textAlign: 'right' }}>Total</th>
                        <th style={{ textAlign: 'center' }}>Estado</th>
                      </tr>
                    </thead>
                    <tbody>
                      {salesList.map(sale => (
                        <tr key={sale.id}>
                          <td style={{ fontSize: '13px', color: 'var(--text-secondary)' }}>
                            <div>{formatDate(sale.created_at)}</div>
                            <div style={{ fontSize: '11px', color: 'var(--text-muted)' }}>
                              {new Date(sale.created_at).toLocaleTimeString('es-CO', { hour: '2-digit', minute: '2-digit' })}
                            </div>
                          </td>
                          <td style={{ textAlign: 'right', fontSize: '13px', color: 'var(--text-secondary)', fontVariantNumeric: 'tabular-nums' }}>
                            {formatCOP(sale.subtotal)}
                          </td>
                          <td style={{ textAlign: 'right', fontSize: '13px', color: sale.discount > 0 ? 'var(--danger)' : 'var(--text-muted)', fontVariantNumeric: 'tabular-nums' }}>
                            {sale.discount > 0 ? `-${formatCOP(sale.discount)}` : '—'}
                          </td>
                          <td style={{ textAlign: 'right', fontWeight: 600, fontSize: '13px', color: 'var(--text-primary)', fontVariantNumeric: 'tabular-nums' }}>
                            {formatCOP(sale.total)}
                          </td>
                          <td style={{ textAlign: 'center' }}>
                            <span style={{
                              fontSize: '11px', fontWeight: 600, padding: '2px 8px', borderRadius: '4px',
                              background: 'var(--accent-dim)', color: 'var(--accent)',
                            }}>
                              CRÉDITO
                            </span>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )
            ) : (
              paymentsList.length === 0 ? (
                <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', minHeight: '200px', gap: '12px' }}>
                  <div style={{ color: 'var(--text-muted)', opacity: 0.4 }}><IconPayment /></div>
                  <div style={{ color: 'var(--text-muted)', fontSize: '14px' }}>No hay abonos registrados</div>
                  {balance > 0 && (
                    <button className="btn btn-sm btn-accent" onClick={() => setShowPaymentModal(true)}>
                      Registrar primer abono
                    </button>
                  )}
                </div>
              ) : (
                <div style={{ marginTop: '12px' }}>
                  {paymentsList.map(p => (
                    <div key={p.id} style={{
                      display: 'flex', alignItems: 'center', justifyContent: 'space-between',
                      padding: '12px 14px',
                      borderBottom: '1px solid var(--border)',
                    }}>
                      <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                        <div style={{
                          width: '8px', height: '8px', borderRadius: '50%',
                          background: 'var(--success)', flexShrink: 0,
                        }} />
                        <div>
                          <div style={{ fontSize: '13px', fontWeight: 500, color: 'var(--text-primary)' }}>
                            {p.note || 'Abono'}
                          </div>
                          <div style={{ fontSize: '11px', color: 'var(--text-muted)', marginTop: '2px' }}>
                            {formatDateTime(p.created_at)}
                          </div>
                        </div>
                      </div>
                      <div style={{ fontSize: '15px', fontWeight: 600, color: 'var(--success)', fontVariantNumeric: 'tabular-nums' }}>
                        +{formatCOP(p.amount)}
                      </div>
                    </div>
                  ))}
                </div>
              )
            )}
          </div>
        </div>
      </div>

      {showPaymentModal && detail && (
        <AbonoModal
          customerId={customer.id}
          customerName={customer.name}
          balance={balance}
          onClose={() => setShowPaymentModal(false)}
          onSaved={load}
        />
      )}

      {showOpeningModal && (
        <OpeningBalanceModal
          customerId={customer.id}
          customerName={customer.name}
          existing={opening_balance}
          onClose={() => setShowOpeningModal(false)}
          onSaved={load}
        />
      )}
    </>
  );
}

// ─── Opening balance modal ────────────────────────────────────────────────────

const openingLabelStyle: React.CSSProperties = {
  fontSize: '12px', fontWeight: 600, color: 'var(--text-secondary)',
  textTransform: 'uppercase', letterSpacing: '0.4px', display: 'block', marginBottom: '6px',
};

// Records (POST) or corrects (PUT) the customer's opening balance. Server
// errors (400/409) stay inside the modal; it closes only on success.
function OpeningBalanceModal({ customerId, customerName, existing, onClose, onSaved }: {
  customerId: number;
  customerName: string;
  existing: OpeningBalance | null;
  onClose: () => void;
  onSaved: () => void;
}) {
  const { toast } = useToast();
  const today = todayISODate();
  const [amount, setAmount] = useState(existing ? existing.amount.toLocaleString('es-CO') : '');
  const [debtDate, setDebtDate] = useState(existing ? existing.debt_date : today);
  const [note, setNote] = useState(existing ? existing.note : '');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  // Guards against a second submit before the disabled state re-renders.
  const submittingRef = useRef(false);

  useEffect(() => {
    function onKeyDown(e: KeyboardEvent) {
      if (e.key === 'Escape') onClose();
    }
    window.addEventListener('keydown', onKeyDown);
    return () => window.removeEventListener('keydown', onKeyDown);
  }, [onClose]);

  const parsed = parseAmountInput(amount);

  function handleAmountChange(val: string) {
    const n = parseAmountInput(val);
    // Cap what can be typed a digit above the limit, so the server message still shows.
    setAmount(n === null ? '' : Math.min(n, MAX_OPENING_BALANCE * 10).toLocaleString('es-CO'));
    setError('');
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (parsed === null || submittingRef.current) return;
    if (!isValidDebtDate(debtDate, today)) {
      setError(debtDate > today ? 'la fecha de la deuda no puede ser futura' : 'fecha de la deuda inválida');
      return;
    }
    submittingRef.current = true;
    setError('');
    setLoading(true);
    try {
      const body = { amount: parsed, debt_date: debtDate, note };
      if (existing) {
        await api.put<OpeningBalance>(`/customers/${customerId}/opening-balance`, body);
        toast('Deuda anterior corregida', 'success');
      } else {
        await api.post<OpeningBalance>(`/customers/${customerId}/opening-balance`, body);
        toast('Deuda anterior registrada', 'success');
      }
      onSaved();
      onClose();
    } catch (err: unknown) {
      setError((err as Error).message);
    } finally {
      submittingRef.current = false;
      setLoading(false);
    }
  }

  return (
    <div
      style={{
        position: 'fixed', inset: 0,
        // Dims the page with the theme's own canvas color (no hard-coded color).
        background: 'color-mix(in srgb, var(--canvas) 75%, transparent)',
        display: 'flex', alignItems: 'center', justifyContent: 'center',
        padding: '16px', zIndex: 1000,
      }}
      onClick={onClose}
    >
      <div
        role="dialog"
        aria-modal="true"
        onClick={e => e.stopPropagation()}
        style={{
          background: 'var(--surface-1)', border: '1px solid var(--border-emphasis)', borderRadius: '12px',
          padding: '20px', width: '420px', maxWidth: '100%', maxHeight: '100%', overflowY: 'auto', boxSizing: 'border-box',
        }}
      >
        <div style={{ marginBottom: '16px', minWidth: 0 }}>
          <h2 style={{ fontSize: '15px', fontWeight: 600, color: 'var(--text-primary)', margin: '0 0 4px' }}>
            {existing ? 'Corregir deuda anterior' : 'Registrar deuda anterior'}
          </h2>
          <p style={{ fontSize: '12px', color: 'var(--text-muted)', margin: 0, overflowWrap: 'anywhere' }}>
            {customerName}
          </p>
        </div>

        <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: '14px' }}>
          <div>
            <label htmlFor="opening-amount" style={openingLabelStyle}>Monto *</label>
            <div style={{ position: 'relative' }}>
              <span style={{ position: 'absolute', left: '12px', top: '50%', transform: 'translateY(-50%)', color: 'var(--text-muted)', fontSize: '14px', fontWeight: 600 }}>$</span>
              <input
                id="opening-amount"
                inputMode="numeric"
                value={amount}
                onChange={e => handleAmountChange(e.target.value)}
                placeholder="0"
                style={{ paddingLeft: '26px', width: '100%', boxSizing: 'border-box', fontSize: '16px', fontWeight: 600 }}
              />
            </div>
          </div>

          <div>
            <label htmlFor="opening-date" style={openingLabelStyle}>Fecha de la deuda *</label>
            <input
              id="opening-date"
              type="date"
              value={debtDate}
              max={today}
              onChange={e => { setDebtDate(e.target.value); setError(''); }}
              style={{ width: '100%', boxSizing: 'border-box' }}
            />
          </div>

          <div>
            <label htmlFor="opening-note" style={openingLabelStyle}>Nota (opcional)</label>
            <input
              id="opening-note"
              value={note}
              onChange={e => setNote(e.target.value)}
              placeholder="Ej: cuaderno pág. 12"
              style={{ width: '100%', boxSizing: 'border-box' }}
            />
          </div>

          {error && (
            <div style={{ fontSize: '12px', color: 'var(--danger)', padding: '8px 12px', background: 'var(--danger-dim)', borderRadius: '6px' }}>
              {error}
            </div>
          )}

          <div style={{ display: 'flex', flexWrap: 'wrap', gap: '8px', justifyContent: 'flex-end', marginTop: '4px' }}>
            <button type="button" onClick={onClose} className="btn btn-sm">
              Cancelar
            </button>
            <button
              type="submit"
              disabled={loading || parsed === null}
              className="btn btn-sm btn-accent"
              style={{ opacity: parsed === null ? 0.5 : 1 }}
            >
              {loading ? 'Guardando...' : 'Guardar'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

export default function CustomerDetailPage() {
  return (
    <AppLayout>
      <CustomerDetailContent />
    </AppLayout>
  );
}

function EmptyState({ icon, text }: { icon: React.ReactNode; text: string }) {
  return (
    <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', minHeight: '200px', gap: '12px' }}>
      <div style={{ color: 'var(--text-muted)', opacity: 0.4 }}>{icon}</div>
      <div style={{ color: 'var(--text-muted)', fontSize: '14px' }}>{text}</div>
    </div>
  );
}

