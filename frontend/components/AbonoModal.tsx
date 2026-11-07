'use client';

// Shared abono (partial payment toward a fiado balance) modal, used by the
// Customers list, the customer detail and the dashboard (US-084).

import { useState, useEffect, useRef } from 'react';
import { api } from '@/lib/api';
import { useToast } from '@/components/Toast';
import { formatCOP } from '@/lib/format';
import {
  ABONO_METHOD_OPTIONS,
  DEFAULT_ABONO_METHOD,
  showsNoRegisterNotice,
  quickAbonoAmounts,
  abonoFieldValue,
  abonoAmountToSend,
  abonoPreview,
  type AbonoMethod,
} from '@/lib/abono';

export interface AbonoPayment {
  id: number;
  customer_id: number;
  amount: number;
  method: AbonoMethod;
  register_id: number | null;
  note: string;
  created_at: string;
}

interface AbonoModalProps {
  customerId: number;
  customerName: string;
  balance: number;
  onClose: () => void;
  onSaved: (payment: AbonoPayment) => void;
}

const IconX = () => (
  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
    <line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/>
  </svg>
);

const labelStyle: React.CSSProperties = {
  fontSize: '12px', fontWeight: 600, color: 'var(--text-secondary)',
  textTransform: 'uppercase', letterSpacing: '0.4px', display: 'block', marginBottom: '6px',
};

export default function AbonoModal({ customerId, customerName, balance, onClose, onSaved }: AbonoModalProps) {
  const { toast } = useToast();
  const [amount, setAmount] = useState('');
  const [note, setNote] = useState('');
  const [method, setMethod] = useState<AbonoMethod>(DEFAULT_ABONO_METHOD);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const inputRef = useRef<HTMLInputElement>(null);
  // Guards against a second submit before the disabled state re-renders.
  const submittingRef = useRef(false);

  useEffect(() => {
    inputRef.current?.focus();
  }, []);

  useEffect(() => {
    function onKeyDown(e: KeyboardEvent) {
      if (e.key === 'Escape') onClose();
    }
    window.addEventListener('keydown', onKeyDown);
    return () => window.removeEventListener('keydown', onKeyDown);
  }, [onClose]);

  const typed = parseInt(amount.replace(/\D/g, '') || '0', 10);
  const toSend = abonoAmountToSend(balance, typed);
  const preview = abonoPreview(balance, typed);
  const quick = quickAbonoAmounts(balance);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!preview.valid || submittingRef.current) return;
    submittingRef.current = true;
    setError('');
    setLoading(true);
    try {
      const saved = await api.post<AbonoPayment>(`/customers/${customerId}/payments`, { amount: toSend, method, note });
      toast('Abono registrado exitosamente', 'success');
      if (showsNoRegisterNotice(saved.method, saved.register_id)) {
        toast('Este abono no quedará en ningún cierre de caja', 'info');
      }
      onSaved(saved);
      onClose();
    } catch (err: unknown) {
      setError((err as Error).message);
    } finally {
      submittingRef.current = false;
      setLoading(false);
    }
  }

  function setFieldNumber(num: number) {
    setAmount(num > 0 ? num.toLocaleString('es-CO') : '');
    setError('');
  }

  function handleAmountChange(val: string) {
    // Only allow digits
    const digits = val.replace(/[^0-9]/g, '');
    setFieldNumber(parseInt(digits || '0', 10));
  }

  const overlayStyle: React.CSSProperties = {
    position: 'fixed', inset: 0,
    // Dims the page with the theme's own canvas color (no hard-coded color).
    background: 'color-mix(in srgb, var(--canvas) 75%, transparent)',
    display: 'flex', alignItems: 'center', justifyContent: 'center',
    padding: '16px',
    zIndex: 1000,
  };
  const modalStyle: React.CSSProperties = {
    background: 'var(--surface-1)',
    border: '1px solid var(--border-emphasis)',
    borderRadius: '12px',
    padding: '20px',
    width: '420px',
    maxWidth: '100%',
    maxHeight: '100%',
    overflowY: 'auto',
    boxSizing: 'border-box',
  };

  return (
    <div style={overlayStyle} onClick={onClose}>
      <div style={modalStyle} onClick={e => e.stopPropagation()} role="dialog" aria-modal="true">
        {/* Header */}
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: '16px', gap: '8px' }}>
          <div style={{ minWidth: 0 }}>
            <h2 style={{ fontSize: '15px', fontWeight: 600, color: 'var(--text-primary)', margin: '0 0 4px' }}>
              Registrar abono
            </h2>
            <p style={{ fontSize: '12px', color: 'var(--text-muted)', margin: 0, overflowWrap: 'anywhere' }}>
              {customerName}
            </p>
          </div>
          <button type="button" onClick={onClose} style={{ background: 'none', border: 'none', cursor: 'pointer', color: 'var(--text-muted)', padding: '2px' }}>
            <IconX />
          </button>
        </div>

        {/* Balance info */}
        <div style={{
          background: 'var(--danger-dim)',
          border: '1px solid var(--danger)',
          borderRadius: '8px',
          padding: '12px 14px',
          marginBottom: '16px',
          display: 'flex',
          flexWrap: 'wrap',
          justifyContent: 'space-between',
          alignItems: 'center',
          gap: '4px 8px',
        }}>
          <span style={{ fontSize: '12px', color: 'var(--text-secondary)' }}>Saldo pendiente</span>
          <span style={{ fontSize: '18px', fontWeight: 700, color: 'var(--danger)', fontVariantNumeric: 'tabular-nums' }}>
            {formatCOP(balance)}
          </span>
        </div>

        <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: '14px' }}>
          <div>
            <label htmlFor="abono-amount" style={labelStyle}>
              Monto del abono *
            </label>
            {quick.length > 0 && (
              <div style={{ display: 'flex', flexWrap: 'wrap', gap: '6px', marginBottom: '8px' }}>
                {quick.map(q => (
                  <button
                    key={q.label}
                    type="button"
                    className="btn btn-sm"
                    onClick={() => setFieldNumber(abonoFieldValue(q.amount))}
                  >
                    {q.label}
                  </button>
                ))}
              </div>
            )}
            <div style={{ position: 'relative' }}>
              <span style={{ position: 'absolute', left: '12px', top: '50%', transform: 'translateY(-50%)', color: 'var(--text-muted)', fontSize: '14px', fontWeight: 600 }}>$</span>
              <input
                id="abono-amount"
                ref={inputRef}
                inputMode="numeric"
                value={amount}
                onChange={e => handleAmountChange(e.target.value)}
                placeholder="0"
                style={{ paddingLeft: '26px', width: '100%', boxSizing: 'border-box', fontSize: '16px', fontWeight: 600 }}
              />
            </div>
            {preview.line && (
              <div style={{ fontSize: '11px', color: 'var(--success)', marginTop: '4px' }}>
                {preview.line}
              </div>
            )}
            {preview.error && (
              <div style={{ fontSize: '11px', color: 'var(--danger)', marginTop: '4px' }}>
                {preview.error}
              </div>
            )}
          </div>

          <div>
            <label htmlFor="abono-method" style={labelStyle}>
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
            <label htmlFor="abono-note" style={labelStyle}>
              Nota (opcional)
            </label>
            <input
              id="abono-note"
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

          <div style={{ display: 'flex', flexWrap: 'wrap', gap: '8px', justifyContent: 'flex-end', marginTop: '4px' }}>
            <button type="button" onClick={onClose} className="btn btn-sm">
              Cancelar
            </button>
            <button
              type="submit"
              disabled={loading || !preview.valid}
              className="btn btn-sm btn-accent"
              style={{ opacity: !preview.valid ? 0.5 : 1 }}
            >
              {loading ? 'Guardando...' : `Registrar ${toSend > 0 ? formatCOP(toSend) : ''}`}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
