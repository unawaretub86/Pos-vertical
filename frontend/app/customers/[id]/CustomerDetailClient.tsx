'use client';

import { useState, useEffect, useCallback, useRef } from 'react';
import { useParams, useRouter } from 'next/navigation';
import AppLayout from '@/components/AppLayout';
import Topbar from '@/components/Topbar';
import { api } from '@/lib/api';
import { useToast } from '@/components/Toast';
import { formatCOP } from '@/lib/format';
import { useBusinessConfig } from '@/lib/business-config-context';
import { ABONO_METHOD_OPTIONS, DEFAULT_ABONO_METHOD, showsNoRegisterNotice, type AbonoMethod } from '@/lib/abono';

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
const IconX = () => (
  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
    <line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/>
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

// ─── Payment Modal ────────────────────────────────────────────────────────────

interface PaymentModalProps {
  customerName: string;
  balance: number;
  onClose: () => void;
  onSaved: () => void;
  customerId: number;
}

function PaymentModal({ customerName, balance, onClose, onSaved, customerId }: PaymentModalProps) {
  const { toast } = useToast();
  const [amount, setAmount] = useState('');
  const [note, setNote] = useState('');
  const [method, setMethod] = useState<AbonoMethod>(DEFAULT_ABONO_METHOD);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    inputRef.current?.focus();
  }, []);

  const parsedAmount = parseFloat(amount.replace(/\./g, '').replace(',', '.')) || 0;
  const isValid = parsedAmount > 0 && parsedAmount <= balance;

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError('');
    if (!isValid) {
      setError(parsedAmount <= 0 ? 'Ingresa un monto válido' : `El monto no puede superar ${formatCOP(balance)}`);
      return;
    }
    setLoading(true);
    try {
      const saved = await api.post<CustomerPayment>(`/customers/${customerId}/payments`, { amount: parsedAmount, method, note });
      toast('Abono registrado exitosamente', 'success');
      if (showsNoRegisterNotice(saved.method, saved.register_id)) {
        toast('Este abono no quedará en ningún cierre de caja', 'info');
      }
      onSaved();
      onClose();
    } catch (err: unknown) {
      setError((err as Error).message);
    } finally {
      setLoading(false);
    }
  }

  function handleAmountChange(val: string) {
    // Only allow digits
    const digits = val.replace(/[^0-9]/g, '');
    const num = parseInt(digits || '0', 10);
    if (num === 0) { setAmount(''); return; }
    setAmount(num.toLocaleString('es-CO'));
    setError('');
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
    width: '420px',
    maxWidth: '90vw',
  };

  return (
    <div style={overlayStyle} onClick={onClose}>
      <div style={modalStyle} onClick={e => e.stopPropagation()}>
        {/* Header */}
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: '20px' }}>
          <div>
            <h2 style={{ fontSize: '15px', fontWeight: 600, color: 'var(--text-primary)', margin: '0 0 4px' }}>
              Registrar abono
            </h2>
            <p style={{ fontSize: '12px', color: 'var(--text-muted)', margin: 0 }}>
              {customerName}
            </p>
          </div>
          <button onClick={onClose} style={{ background: 'none', border: 'none', cursor: 'pointer', color: 'var(--text-muted)', padding: '2px' }}>
            <IconX />
          </button>
        </div>

        {/* Balance info */}
        <div style={{
          background: 'var(--danger-dim)',
          border: '1px solid var(--danger)',
          borderRadius: '8px',
          padding: '12px 14px',
          marginBottom: '20px',
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
        }}>
          <span style={{ fontSize: '12px', color: 'var(--text-secondary)' }}>Saldo pendiente</span>
          <span style={{ fontSize: '18px', fontWeight: 700, color: 'var(--danger)', fontVariantNumeric: 'tabular-nums' }}>
            {formatCOP(balance)}
          </span>
        </div>

        <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: '14px' }}>
          <div>
            <label style={{ fontSize: '12px', fontWeight: 600, color: 'var(--text-secondary)', textTransform: 'uppercase', letterSpacing: '0.4px', display: 'block', marginBottom: '6px' }}>
              Monto del abono *
            </label>
            <div style={{ position: 'relative' }}>
              <span style={{ position: 'absolute', left: '12px', top: '50%', transform: 'translateY(-50%)', color: 'var(--text-muted)', fontSize: '14px', fontWeight: 600 }}>$</span>
              <input
                ref={inputRef}
                value={amount}
                onChange={e => handleAmountChange(e.target.value)}
                placeholder="0"
                style={{ paddingLeft: '26px', width: '100%', boxSizing: 'border-box', fontSize: '16px', fontWeight: 600 }}
              />
            </div>
            {parsedAmount > 0 && parsedAmount <= balance && (
              <div style={{ fontSize: '11px', color: 'var(--success)', marginTop: '4px' }}>
                Saldo restante: {formatCOP(balance - parsedAmount)}
              </div>
            )}
          </div>

          <div>
            <label htmlFor="abono-method" style={{ fontSize: '12px', fontWeight: 600, color: 'var(--text-secondary)', textTransform: 'uppercase', letterSpacing: '0.4px', display: 'block', marginBottom: '6px' }}>
              Método de pago
            </label>
            <select
              id="abono-method"
              className="input"
              value={method}
              onChange={e => setMethod(e.target.value as AbonoMethod)}
              style={{ width: '100%', boxSizing: 'border-box', cursor: 'pointer' }}
            >
              {ABONO_METHOD_OPTIONS.map(o => (
                <option key={o.value} value={o.value}>{o.label}</option>
              ))}
            </select>
          </div>

          <div>
            <label style={{ fontSize: '12px', fontWeight: 600, color: 'var(--text-secondary)', textTransform: 'uppercase', letterSpacing: '0.4px', display: 'block', marginBottom: '6px' }}>
              Nota (opcional)
            </label>
            <input
              value={note}
              onChange={e => setNote(e.target.value)}
              placeholder="Ej: referencia de la transferencia"
              style={{ width: '100%', boxSizing: 'border-box' }}
            />
          </div>

          {error && (
            <div style={{ fontSize: '12px', color: 'var(--danger)', padding: '8px 12px', background: 'var(--danger-dim)', borderRadius: '6px' }}>
              {error}
            </div>
          )}

          <div style={{ display: 'flex', gap: '8px', justifyContent: 'flex-end', marginTop: '4px' }}>
            <button type="button" onClick={onClose} className="btn btn-sm">
              Cancelar
            </button>
            <button
              type="submit"
              disabled={loading || !isValid}
              className="btn btn-sm btn-accent"
              style={{ opacity: !isValid ? 0.5 : 1 }}
            >
              {loading ? 'Guardando...' : `Registrar ${parsedAmount > 0 ? formatCOP(parsedAmount) : ''}`}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

// ─── Main Page ────────────────────────────────────────────────────────────────

function buildWhatsAppUrl(phone: string, customerName: string, balance: number, businessName: string): string {
  const digits = phone.replace(/\D/g, '');
  const fullPhone = digits.startsWith('57') ? digits : `57${digits}`;
  const amount = formatCOP(balance);
  const biz = businessName || 'el negocio';
  const text = `Hola ${customerName}, le recordamos que tiene un saldo pendiente de ${amount} en ${biz}. Gracias.`;
  return `https://wa.me/${fullPhone}?text=${encodeURIComponent(text)}`;
}

export default function CustomerDetailPage() {
  const params = useParams();
  const router = useRouter();
  const { toast } = useToast();
  const { config: bizConfig } = useBusinessConfig();
  const customerId = Number(params.id);

  const [detail, setDetail] = useState<CustomerDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [showPaymentModal, setShowPaymentModal] = useState(false);
  const [activeTab, setActiveTab] = useState<'sales' | 'payments'>('sales');

  const load = useCallback(async () => {
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
    queueMicrotask(load);
  }, [load]);

  if (loading) {
    return (
      <AppLayout>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '100%', color: 'var(--text-muted)', fontSize: '13px' }}>
          Cargando...
        </div>
      </AppLayout>
    );
  }

  if (!detail) {
    return (
      <AppLayout>
        <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', height: '100%', gap: '12px' }}>
          <div style={{ color: 'var(--text-muted)', fontSize: '14px' }}>Cliente no encontrado</div>
          <button className="btn btn-sm" onClick={() => router.push('/customers')}>
            Volver a clientes
          </button>
        </div>
      </AppLayout>
    );
  }

  const { customer, balance, payments, credit_sales } = detail;
  const salesList = credit_sales ?? [];
  const paymentsList = payments ?? [];

  return (
    <AppLayout>
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
                      const url = buildWhatsAppUrl(customer.phone, customer.name, balance, bizConfig.name);
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
                <div style={{ fontSize: '22px', fontWeight: 700, color: balance > 0 ? 'var(--danger)' : 'var(--success)', fontVariantNumeric: 'tabular-nums', letterSpacing: '-0.5px' }}>
                  {formatCOP(balance)}
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
        <PaymentModal
          customerId={customer.id}
          customerName={customer.name}
          balance={balance}
          onClose={() => setShowPaymentModal(false)}
          onSaved={load}
        />
      )}
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

