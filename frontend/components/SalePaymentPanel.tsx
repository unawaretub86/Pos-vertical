import { useState, useEffect } from 'react';
import { api } from '@/lib/api';
import { formatCOP, formatInputCOP, parseInputCOP } from '@/lib/format';
import { paymentsBalanced as linesBalanced, sumPayments, calcChange, exceedsCreditLimit } from '@/lib/saleCalc';
import { useToast } from '@/components/Toast';

export type PaymentMethod = 'efectivo' | 'transferencia' | 'tarjeta' | 'credito';

export interface Customer {
  id: number;
  name: string;
  phone: string;
  notes: string;
  credit_limit: number | null;
}

export interface CheckoutData {
  payMethod: PaymentMethod;
  paymentLines: { method: PaymentMethod; amount: number }[];
  cashReceivedNum: number;
  change: number;
  isMixedMode: boolean;
}

interface CustomerPickerProps {
  selected: Customer | null;
  onSelect: (c: Customer | null) => void;
}

// We reuse the CustomerPicker, so we need it here or exported from POSPage.
// I'll put it in SalePaymentPanel to keep it self-contained.
function CustomerPicker({ selected, onSelect }: CustomerPickerProps) {
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState('');
  const [results, setResults] = useState<Customer[]>([]);
  const [loading, setLoading] = useState(false);
  const [debouncedQuery, setDebouncedQuery] = useState('');

  useEffect(() => {
    const t = setTimeout(() => setDebouncedQuery(query), 250);
    return () => clearTimeout(t);
  }, [query]);

  useEffect(() => {
    if (!open) return;
    queueMicrotask(() => setLoading(true));
    api.get<{ customers: Customer[] | null }>(`/customers${debouncedQuery ? `?q=${encodeURIComponent(debouncedQuery)}` : ''}`)
      .then(res => setResults(res.customers ?? []))
      .catch(() => setResults([]))
      .finally(() => setLoading(false));
  }, [open, debouncedQuery]);

  return (
    <div style={{ position: 'relative' }}>
      <button
        type="button"
        onClick={() => setOpen(o => !o)}
        style={{
          width: '100%', display: 'flex', alignItems: 'center', justifyContent: 'space-between',
          padding: '8px 10px', border: `1px solid ${selected ? 'var(--accent)' : 'var(--border)'}`,
          borderRadius: '6px', background: selected ? 'var(--accent-dim)' : 'var(--surface-1)',
          color: selected ? 'var(--accent)' : 'var(--text-muted)', cursor: 'pointer', textAlign: 'left', gap: '8px',
          fontSize: '13px', fontWeight: selected ? 600 : 400,
        }}
      >
        <span style={{ display: 'flex', alignItems: 'center', gap: '6px', flex: 1, minWidth: 0 }}>
          <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" style={{ flexShrink: 0 }}>
            <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/>
          </svg>
          <span style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
            {selected ? selected.name : 'Seleccionar cliente...'}
          </span>
        </span>
        {selected ? (
          <span onClick={(e) => { e.stopPropagation(); onSelect(null); }} title="Quitar cliente" style={{ flexShrink: 0, fontSize: '16px', lineHeight: 1, opacity: 0.7 }}>×</span>
        ) : (
          <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round" style={{ flexShrink: 0 }}>
            <polyline points="6 9 12 15 18 9"/>
          </svg>
        )}
      </button>

      {open && (
        <div style={{
          position: 'absolute', bottom: 'calc(100% + 4px)', left: 0, right: 0,
          background: 'var(--surface-2)', border: '1px solid var(--border-emphasis)',
          borderRadius: '8px', zIndex: 100, boxShadow: '0 8px 24px rgba(0,0,0,0.3)', overflow: 'hidden',
        }}>
          <div style={{ padding: '8px' }}>
            <input
              autoFocus
              value={query}
              onChange={e => setQuery(e.target.value)}
              placeholder="Buscar cliente..."
              style={{ width: '100%', boxSizing: 'border-box', fontSize: '13px' }}
            />
          </div>
          <div style={{ maxHeight: '180px', overflowY: 'auto' }}>
            {loading ? (
              <div style={{ padding: '10px 12px', fontSize: '12px', color: 'var(--text-muted)' }}>Buscando...</div>
            ) : results.length === 0 ? (
              <div style={{ padding: '10px 12px', fontSize: '12px', color: 'var(--text-muted)' }}>
                {query ? 'Sin resultados' : 'No hay clientes registrados'}
              </div>
            ) : (
              results.map(c => (
                <button
                  key={c.id}
                  type="button"
                  onClick={() => { onSelect(c); setOpen(false); setQuery(''); }}
                  style={{
                    width: '100%', textAlign: 'left', padding: '8px 12px',
                    background: 'transparent', border: 'none', cursor: 'pointer',
                    display: 'flex', flexDirection: 'column', gap: '1px', borderBottom: '1px solid var(--border)',
                  }}
                  onMouseEnter={e => (e.currentTarget.style.background = 'var(--surface-3)')}
                  onMouseLeave={e => (e.currentTarget.style.background = 'transparent')}
                >
                  <span style={{ fontSize: '13px', fontWeight: 600, color: 'var(--text-primary)' }}>{c.name}</span>
                  {c.phone && <span style={{ fontSize: '11px', color: 'var(--text-muted)' }}>{c.phone}</span>}
                </button>
              ))
            )}
          </div>
        </div>
      )}
    </div>
  );
}

function PayBtn({ label, active, onClick }: { label: string; active: boolean; onClick: () => void }) {
  return (
    <button
      onClick={onClick}
      style={{
        padding: '8px 4px', border: `1px solid ${active ? 'var(--accent)' : 'var(--border)'}`,
        borderRadius: '6px', background: active ? 'rgba(245,166,35,0.12)' : 'var(--surface-1)',
        color: active ? 'var(--accent)' : 'var(--text-secondary)',
        fontSize: '12px', fontWeight: active ? 600 : 400, cursor: 'pointer', transition: 'all 120ms ease',
      }}
    >
      {label}
    </button>
  );
}

interface SalePaymentPanelProps {
  total: number;
  cartTotal: number;
  selectedCustomer: Customer | null;
  onCustomerChange: (c: Customer | null) => void;
  onCheckout: (data: CheckoutData) => void;
  loading: boolean;
  disabled: boolean;
}

export default function SalePaymentPanel({
  total,
  cartTotal,
  selectedCustomer,
  onCustomerChange,
  onCheckout,
  loading,
  disabled
}: SalePaymentPanelProps) {
  const { toast } = useToast();
  
  const [payMethod, setPayMethod] = useState<PaymentMethod>('efectivo');
  const [cashReceived, setCashReceived] = useState<string>('');
  const [extraPayments, setExtraPayments] = useState<{ method: PaymentMethod; amountInput: string }[]>([]);
  const [firstAmountInput, setFirstAmountInput] = useState<string>('');
  const [customerBalance, setCustomerBalance] = useState<number | null>(null);

  useEffect(() => {
    if (!selectedCustomer) { queueMicrotask(() => setCustomerBalance(null)); return; }
    api.get<{ balance: number }>(`/customers/${selectedCustomer.id}/balance`)
      .then(r => setCustomerBalance(r.balance))
      .catch(() => setCustomerBalance(null));
  }, [selectedCustomer]);

  // Derived state
  const cashReceivedNum = parseInputCOP(cashReceived);
  const change = payMethod === 'efectivo' ? calcChange(cashReceivedNum, total) : 0;
  
  const isMixedMode = extraPayments.length > 0;
  const firstLineAmount = isMixedMode ? parseInputCOP(firstAmountInput) : total;
  const paymentLines: { method: PaymentMethod; amount: number }[] = [
    { method: payMethod, amount: firstLineAmount },
    ...extraPayments.map(e => ({ method: e.method, amount: parseInputCOP(e.amountInput) })),
  ];
  const paymentsCovered = sumPayments(paymentLines);
  const remaining = total - paymentsCovered;
  const paymentsBalanced = linesBalanced(paymentLines, total);
  const anyLineIsCredit = paymentLines.some(l => l.method === 'credito');

  function addPaymentLine() {
    const used = new Set(paymentLines.map(l => l.method));
    const next = (['efectivo', 'transferencia', 'tarjeta', 'credito'] as PaymentMethod[]).find(m => !used.has(m)) ?? 'transferencia';
    setExtraPayments(prev => [...prev, { method: next, amountInput: '' }]);
  }

  function removePaymentLine(idx: number) {
    setExtraPayments(prev => prev.filter((_, i) => i !== idx));
  }

  function handleCobrar() {
    if (disabled) return;
    if (total <= 0) {
      toast('El total debe ser mayor a cero', 'error');
      return;
    }
    if (anyLineIsCredit && !selectedCustomer) {
      toast('Selecciona un cliente para ventas a crédito', 'error');
      return;
    }
    if (!paymentsBalanced) {
      toast('La suma de los pagos debe ser igual al total', 'error');
      return;
    }
    
    onCheckout({
      payMethod,
      paymentLines,
      cashReceivedNum,
      change,
      isMixedMode
    });
  }

  const canCobrar = !disabled && !loading && paymentsBalanced && !(anyLineIsCredit && !selectedCustomer);

  return (
    <>
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '6px' }}>
        {(['efectivo', 'transferencia', 'tarjeta', 'credito'] as PaymentMethod[]).map(m => (
          <PayBtn
            key={m}
            label={m.charAt(0).toUpperCase() + m.slice(1)}
            active={payMethod === m}
            onClick={() => {
              setPayMethod(m);
              const stillNeedsCredit = m === 'credito' || extraPayments.some(e => e.method === 'credito');
              if (!stillNeedsCredit) onCustomerChange(null);
            }}
          />
        ))}
      </div>

      <div style={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
        {isMixedMode && (
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', fontSize: '13px', color: 'var(--text-secondary)' }}>
            <span>{payMethod.charAt(0).toUpperCase() + payMethod.slice(1)}</span>
            <input
              type="text"
              inputMode="numeric"
              value={firstAmountInput}
              placeholder="0"
              onChange={e => setFirstAmountInput(formatInputCOP(e.target.value))}
              style={{
                width: '110px', textAlign: 'right', padding: '3px 8px', border: '1px solid var(--border)',
                borderRadius: '4px', background: 'var(--surface-1)', color: 'var(--text-primary)',
                fontSize: '13px', fontVariantNumeric: 'tabular-nums',
              }}
            />
          </div>
        )}
        {extraPayments.map((line, idx) => (
          <div key={idx} style={{ display: 'flex', gap: '6px', alignItems: 'center' }}>
            <select
              value={line.method}
              onChange={e => {
                const method = e.target.value as PaymentMethod;
                setExtraPayments(prev => prev.map((l, i) => (i === idx ? { ...l, method } : l)));
              }}
              style={{
                flex: 1, minWidth: 0, padding: '5px 6px', border: '1px solid var(--border)',
                borderRadius: '4px', background: 'var(--surface-1)', color: 'var(--text-primary)', fontSize: '12px',
              }}
            >
              {(['efectivo', 'transferencia', 'tarjeta', 'credito'] as PaymentMethod[]).map(m => (
                <option key={m} value={m}>{m.charAt(0).toUpperCase() + m.slice(1)}</option>
              ))}
            </select>
            <input
              type="text"
              inputMode="numeric"
              value={line.amountInput}
              placeholder="0"
              onChange={e => {
                const amountInput = formatInputCOP(e.target.value);
                setExtraPayments(prev => prev.map((l, i) => (i === idx ? { ...l, amountInput } : l)));
              }}
              style={{
                width: '100px', textAlign: 'right', padding: '5px 8px', border: '1px solid var(--border)',
                borderRadius: '4px', background: 'var(--surface-1)', color: 'var(--text-primary)',
                fontSize: '12px', fontVariantNumeric: 'tabular-nums',
              }}
            />
            <button
              onClick={() => removePaymentLine(idx)}
              style={{
                padding: '5px 9px', border: '1px solid var(--border)', borderRadius: '4px',
                background: 'var(--surface-1)', color: 'var(--danger)', fontSize: '13px', cursor: 'pointer',
              }}
            >
              ×
            </button>
          </div>
        ))}
        {paymentLines.length < 3 && (
          <button
            onClick={addPaymentLine}
            style={{
              alignSelf: 'flex-start', padding: '5px 8px', border: '1px dashed var(--border)',
              borderRadius: '4px', background: 'transparent', color: 'var(--accent)', fontSize: '12px', cursor: 'pointer',
            }}
          >
            + Agregar otro método de pago
          </button>
        )}
        {isMixedMode && (
          <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: '13px', fontWeight: 600 }}>
            <span style={{ color: 'var(--text-secondary)' }}>
              {remaining > 0.5 ? 'Falta cubrir' : remaining < -0.5 ? 'Excede por' : 'Cubierto'}
            </span>
            <span style={{ color: paymentsBalanced ? 'var(--success)' : 'var(--danger)', fontVariantNumeric: 'tabular-nums' }}>
              {formatCOP(Math.abs(remaining))}
            </span>
          </div>
        )}
      </div>

      {anyLineIsCredit && (
        <div>
          <div style={{ fontSize: '11px', fontWeight: 600, color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '0.4px', marginBottom: '6px' }}>
            Cliente *
          </div>
          <CustomerPicker selected={selectedCustomer} onSelect={onCustomerChange} />
          {!selectedCustomer && (
            <div style={{ fontSize: '11px', color: 'var(--danger)', marginTop: '4px' }}>
              Requerido para ventas a crédito
            </div>
          )}
          {selectedCustomer?.notes && (
            <div style={{
              marginTop: '6px', padding: '8px 10px', background: 'var(--accent-dim)',
              border: '1px solid var(--accent)', borderRadius: '6px', fontSize: '12px', color: 'var(--text-primary)',
            }}>
              <span style={{ fontWeight: 600, color: 'var(--accent)', marginRight: '4px' }}>📝</span>
              {selectedCustomer.notes}
            </div>
          )}
          {(() => {
            if (!selectedCustomer?.credit_limit || customerBalance === null) return null;
            if (exceedsCreditLimit(customerBalance, cartTotal, selectedCustomer.credit_limit)) {
              return (
                <div style={{
                  marginTop: '6px', padding: '8px 10px', background: 'var(--danger-dim)',
                  border: '1px solid var(--danger)', borderRadius: '6px', fontSize: '12px',
                  color: 'var(--danger)', fontWeight: 500,
                }}>
                  ⚠️ Tope de crédito: ${selectedCustomer.credit_limit.toLocaleString('es-CO')}. Deuda actual: ${customerBalance.toLocaleString('es-CO')}.
                </div>
              );
            }
            return null;
          })()}
        </div>
      )}

      {!isMixedMode && payMethod === 'efectivo' && (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', fontSize: '13px', color: 'var(--text-secondary)' }}>
            <span>Recibido</span>
            <input
              type="text"
              inputMode="numeric"
              value={cashReceived}
              placeholder={formatCOP(Math.ceil(total))}
              onChange={e => setCashReceived(formatInputCOP(e.target.value))}
              style={{
                width: '110px', textAlign: 'right', padding: '3px 8px', border: '1px solid var(--border)',
                borderRadius: '4px', background: 'var(--surface-1)', color: 'var(--text-primary)',
                fontSize: '13px', fontVariantNumeric: 'tabular-nums',
              }}
            />
          </div>
          {cashReceivedNum > 0 && (
            <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: '13px' }}>
              <span style={{ color: 'var(--text-secondary)' }}>Cambio</span>
              <span style={{ fontVariantNumeric: 'tabular-nums', fontWeight: 600, color: change >= 0 ? 'var(--success)' : 'var(--danger)' }}>
                {formatCOP(change)}
              </span>
            </div>
          )}
        </div>
      )}

      <button
        onClick={handleCobrar}
        disabled={!canCobrar}
        style={{
          width: '100%', padding: '14px', minHeight: '48px', borderRadius: '8px', border: 'none',
          background: canCobrar ? 'var(--accent)' : 'var(--surface-3)',
          color: canCobrar ? '#000' : 'var(--text-muted)',
          fontSize: '15px', fontWeight: 700, cursor: canCobrar ? 'pointer' : 'not-allowed',
          fontVariantNumeric: 'tabular-nums', transition: 'opacity 120ms ease',
          opacity: loading ? 0.7 : 1,
        }}
      >
        {loading ? 'Procesando...' : `Cobrar ${formatCOP(total)}`}
      </button>
    </>
  );
}
