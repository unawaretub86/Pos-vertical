'use client';

import { useState, useEffect, useCallback } from 'react';
import AppLayout from '@/components/AppLayout';
import Topbar from '@/components/Topbar';
import { api } from '@/lib/api';
import { useToast } from '@/components/Toast';
import { formatCOP, formatInputCOP, parseInputCOP } from '@/lib/format';
import { COP_DENOMINATIONS, sumDenominations, type DenominationCounts } from '@/lib/denominations';
import { getUser } from '@/lib/auth';
import CloseTicketPreview, { type CloseTicketData } from '@/components/CloseTicketPreview';
import type { TicketBusiness } from '@/components/TicketPreview';

// ─── Types ────────────────────────────────────────────────────────────────────

interface Register {
  id: number;
  user_id: number;
  opening_amount: number;
  expected_close: number | null;
  actual_close: number | null;
  difference: number | null;
  status: 'open' | 'closed';
  opened_at: string;
  closed_at: string | null;
}

interface Movement {
  id: number;
  register_id: number;
  type: 'opening' | 'sale' | 'cash_in' | 'cash_out';
  amount: number;
  description: string;
  created_at: string;
}

interface RegisterSummary {
  register: Register;
  cash_sales: number;
  cash_in: number;
  cash_out: number;
  expected: number;
  movements: Movement[];
  by_payment_method: { method: string; total: number; count: number }[];
  open_orders_count: number;
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

const formatTime = (iso: string) => {
  const d = new Date(iso);
  return d.toLocaleTimeString('es-CO', { hour: '2-digit', minute: '2-digit' });
};

const formatDateTime = (iso: string) => {
  const d = new Date(iso);
  return d.toLocaleString('es-CO', {
    day: '2-digit', month: '2-digit', year: 'numeric',
    hour: '2-digit', minute: '2-digit',
  });
};

const hoursSince = (iso: string) => {
  const diff = Date.now() - new Date(iso).getTime();
  return Math.floor(diff / 3_600_000);
};

// ─── Icons ───────────────────────────────────────────────────────────────────

const IconCheck = () => (
  <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
    <polyline points="20 6 9 17 4 12"/>
  </svg>
);

const IconX = () => (
  <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
    <line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/>
  </svg>
);

const IconPlus = () => (
  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
    <line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/>
  </svg>
);

const IconMinus = () => (
  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
    <line x1="5" y1="12" x2="19" y2="12"/>
  </svg>
);

const IconCash = () => (
  <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
    <rect x="2" y="6" width="20" height="12" rx="2"/>
    <circle cx="12" cy="12" r="2"/>
    <path d="M6 12h.01M18 12h.01"/>
  </svg>
);

const IconLock = () => (
  <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
    <rect x="3" y="11" width="18" height="11" rx="2" ry="2"/>
    <path d="M7 11V7a5 5 0 0 1 10 0v4"/>
  </svg>
);

// ─── Modal de Apertura ────────────────────────────────────────────────────────

function OpenModal({ onClose, onOpen }: { onClose: () => void; onOpen: (amount: number) => Promise<void> }) {
  const [amount, setAmount] = useState('');
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const parsed = parseInputCOP(amount);
    if (parsed < 0) return;
    setLoading(true);
    try {
      await onOpen(parsed);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div style={{
      position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.6)',
      display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 1000,
    }}>
      <div style={{
        background: 'var(--surface-1)', border: '1px solid var(--border)',
        borderRadius: 12, padding: 28, width: 380,
        boxShadow: '0 8px 32px rgba(0,0,0,0.4)',
      }}>
        <h3 style={{ margin: '0 0 4px', fontSize: 15, fontWeight: 600 }}>Abrir Caja</h3>
        <p style={{ margin: '0 0 20px', fontSize: 13, color: 'var(--text-secondary)' }}>
          Ingresa el monto base con el que inicias el turno.
        </p>
        <form onSubmit={handleSubmit}>
          <label style={{ display: 'block', fontSize: 12, fontWeight: 600, color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '0.5px', marginBottom: 6 }}>
            Monto de apertura (COP)
          </label>
          <input
            className="input"
            style={{ width: '100%', marginBottom: 20, boxSizing: 'border-box', fontVariantNumeric: 'tabular-nums' }}
            type="text"
            inputMode="numeric"
            value={amount}
            onChange={e => setAmount(formatInputCOP(e.target.value))}
            placeholder="$0"
            autoFocus
          />
          <div style={{ display: 'flex', gap: 8 }}>
            <button type="button" className="btn btn-full" onClick={onClose} disabled={loading}>
              Cancelar
            </button>
            <button type="submit" className="btn-accent btn btn-full" disabled={loading}>
              {loading ? 'Abriendo...' : 'Abrir Caja'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

// ─── Modal de Movimiento ──────────────────────────────────────────────────────

function MovementModal({
  onClose,
  onSave,
}: {
  onClose: () => void;
  onSave: (type: 'cash_in' | 'cash_out', amount: number, description: string) => Promise<void>;
}) {
  const [mvType, setMvType] = useState<'cash_in' | 'cash_out'>('cash_in');
  const [amount, setAmount] = useState('');
  const [description, setDescription] = useState('');
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const parsed = parseInputCOP(amount);
    if (parsed <= 0) return;
    setLoading(true);
    try {
      await onSave(mvType, parsed, description);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div style={{
      position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.6)',
      display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 1000,
    }}>
      <div style={{
        background: 'var(--surface-1)', border: '1px solid var(--border)',
        borderRadius: 12, padding: 28, width: 380,
        boxShadow: '0 8px 32px rgba(0,0,0,0.4)',
      }}>
        <h3 style={{ margin: '0 0 16px', fontSize: 15, fontWeight: 600 }}>Registrar Movimiento</h3>

        {/* Tipo */}
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 8, marginBottom: 16 }}>
          {(['cash_in', 'cash_out'] as const).map(t => (
            <button
              key={t}
              type="button"
              onClick={() => setMvType(t)}
              style={{
                padding: '8px 0', borderRadius: 6, fontSize: 13, fontWeight: 500, cursor: 'pointer',
                border: mvType === t
                  ? `1px solid ${t === 'cash_in' ? 'var(--success)' : 'var(--danger)'}`
                  : '1px solid var(--border)',
                background: mvType === t
                  ? (t === 'cash_in' ? 'var(--success-dim)' : 'var(--danger-dim)')
                  : 'var(--surface-2)',
                color: mvType === t
                  ? (t === 'cash_in' ? 'var(--success)' : 'var(--danger)')
                  : 'var(--text-secondary)',
                transition: '150ms ease',
              }}
            >
              {t === 'cash_in' ? '↑ Entrada' : '↓ Salida'}
            </button>
          ))}
        </div>

        <form onSubmit={handleSubmit}>
          <label style={{ display: 'block', fontSize: 12, fontWeight: 600, color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '0.5px', marginBottom: 6 }}>
            Monto (COP)
          </label>
          <input
            className="input"
            style={{ width: '100%', marginBottom: 12, boxSizing: 'border-box', fontVariantNumeric: 'tabular-nums' }}
            type="text"
            inputMode="numeric"
            value={amount}
            onChange={e => setAmount(formatInputCOP(e.target.value))}
            placeholder="$0"
            autoFocus
          />
          <label style={{ display: 'block', fontSize: 12, fontWeight: 600, color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '0.5px', marginBottom: 6 }}>
            Descripción (opcional)
          </label>
          <input
            className="input"
            style={{ width: '100%', marginBottom: 20, boxSizing: 'border-box' }}
            type="text"
            value={description}
            onChange={e => setDescription(e.target.value)}
            placeholder="Ej: Pago proveedor, cambio..."
          />
          <div style={{ display: 'flex', gap: 8 }}>
            <button type="button" className="btn btn-full" onClick={onClose} disabled={loading}>
              Cancelar
            </button>
            <button type="submit" className="btn-accent btn btn-full" disabled={loading}>
              {loading ? 'Guardando...' : 'Registrar'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

// ─── Modal de Cierre ──────────────────────────────────────────────────────────

function DenominationGrid({ counts, onChange }: { counts: DenominationCounts; onChange: (counts: DenominationCounts) => void }) {
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 6, marginBottom: 12 }}>
      {COP_DENOMINATIONS.map(value => (
        <div key={value} style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
          <span style={{ flex: 1, fontSize: 13, fontVariantNumeric: 'tabular-nums', color: 'var(--text-secondary)' }}>
            {formatCOP(value)}
          </span>
          <input
            className="input"
            style={{ width: 72, boxSizing: 'border-box', fontVariantNumeric: 'tabular-nums', textAlign: 'right' }}
            type="text"
            inputMode="numeric"
            value={counts[value] ? String(counts[value]) : ''}
            onChange={e => {
              const n = parseInt(e.target.value.replace(/\D/g, ''), 10);
              onChange({ ...counts, [value]: Number.isNaN(n) ? 0 : n });
            }}
            placeholder="0"
          />
          <span style={{ width: 96, textAlign: 'right', fontSize: 12, color: 'var(--text-muted)', fontVariantNumeric: 'tabular-nums' }}>
            {formatCOP(value * (counts[value] || 0))}
          </span>
        </div>
      ))}
    </div>
  );
}

function CloseModal({
  summary,
  onClose,
  onConfirm,
}: {
  summary: RegisterSummary;
  onClose: () => void;
  onConfirm: (actualClose: number) => Promise<void>;
}) {
  const [denomCounts, setDenomCounts] = useState<DenominationCounts>({});
  const [counted, setCounted] = useState('');
  const [showDenoms, setShowDenoms] = useState(true);
  const [loading, setLoading] = useState(false);
  const countedNum = parseInputCOP(counted);
  const difference = counted ? countedNum - summary.expected : 0;
  const hasFaltante = counted !== '' && difference < 0;

  const handleDenomChange = (next: DenominationCounts) => {
    setDenomCounts(next);
    setCounted(formatInputCOP(String(sumDenominations(next))));
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (hasFaltante || counted === '') return;
    setLoading(true);
    try {
      await onConfirm(countedNum);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div style={{
      position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.6)',
      display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 1000,
      padding: 12,
    }}>
      <div style={{
        background: 'var(--surface-1)', border: '1px solid var(--border)',
        borderRadius: 12, padding: 28, width: 'min(420px, 100%)',
        maxHeight: '92vh', overflowY: 'auto',
        boxShadow: '0 8px 32px rgba(0,0,0,0.4)',
      }}>
        <h3 style={{ margin: '0 0 4px', fontSize: 15, fontWeight: 600 }}>Cerrar Caja</h3>
        <p style={{ margin: '0 0 20px', fontSize: 13, color: 'var(--text-secondary)' }}>
          Cuenta las denominaciones para calcular el monto real automáticamente.
        </p>

        {summary.open_orders_count > 0 && (
          <div style={{
            background: 'rgba(255,165,0,0.1)', border: '1px solid var(--warning)',
            borderRadius: 8, padding: '12px 16px', marginBottom: 20, color: 'var(--warning-dark)',
            fontSize: 14, fontWeight: 500,
          }}>
            Hay {summary.open_orders_count} pedido{summary.open_orders_count === 1 ? '' : 's'} sin cobrar. Asegúrate de cerrarlo{summary.open_orders_count === 1 ? '' : 's'} antes de continuar o ciérralo{summary.open_orders_count === 1 ? '' : 's'} como pérdida si corresponde.
          </div>
        )}

        {/* Resumen esperado */}
        <div style={{
          background: 'var(--surface-2)', border: '1px solid var(--border)',
          borderRadius: 8, padding: '12px 16px', marginBottom: 20,
        }}>
          {[
            ['Base apertura', summary.register.opening_amount],
            ['Ventas efectivo', summary.cash_sales],
            ['Salidas manuales', -summary.cash_out],
          ].map(([label, val]) => (
            <div key={label as string} style={{ display: 'flex', justifyContent: 'space-between', fontSize: 13, marginBottom: 4 }}>
              <span style={{ color: 'var(--text-secondary)' }}>{label as string}</span>
              <span style={{ fontVariantNumeric: 'tabular-nums', fontWeight: 500 }}>
                {formatCOP(val as number)}
              </span>
            </div>
          ))}
          <div style={{ borderTop: '1px solid var(--border)', marginTop: 8, paddingTop: 8, display: 'flex', justifyContent: 'space-between', fontSize: 14, fontWeight: 700 }}>
            <span>Total esperado</span>
            <span style={{ color: 'var(--accent)', fontVariantNumeric: 'tabular-nums' }}>{formatCOP(summary.expected)}</span>
          </div>
        </div>

        <button
          type="button"
          onClick={() => setShowDenoms(s => !s)}
          style={{
            display: 'flex', alignItems: 'center', justifyContent: 'space-between', width: '100%',
            background: 'none', border: 'none', padding: 0, marginBottom: 8,
            fontSize: 12, fontWeight: 600, color: 'var(--text-muted)', textTransform: 'uppercase',
            letterSpacing: '0.5px', cursor: 'pointer',
          }}
        >
          <span>Conteo de denominaciones</span>
          <span>{showDenoms ? '−' : '+'}</span>
        </button>
        {showDenoms && <DenominationGrid counts={denomCounts} onChange={handleDenomChange} />}

        <form onSubmit={handleSubmit}>
          <label style={{ display: 'block', fontSize: 12, fontWeight: 600, color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '0.5px', marginBottom: 6 }}>
            Monto real contado
          </label>
          <input
            className="input"
            style={{ width: '100%', marginBottom: 12, boxSizing: 'border-box', fontVariantNumeric: 'tabular-nums' }}
            type="text"
            inputMode="numeric"
            value={counted}
            onChange={e => setCounted(formatInputCOP(e.target.value))}
            placeholder="$0"
          />

          {counted !== '' && (
            <div style={{
              display: 'flex', justifyContent: 'space-between', alignItems: 'center',
              padding: '10px 14px', borderRadius: 8, marginBottom: 12,
              background: difference >= 0 ? 'var(--success-dim)' : 'var(--danger-dim)',
              border: `1px solid ${difference >= 0 ? 'var(--success)' : 'var(--danger)'}`,
            }}>
              <span style={{ fontSize: 13, fontWeight: 500, color: difference >= 0 ? 'var(--success)' : 'var(--danger)' }}>
                {difference >= 0 ? 'Sobrante' : 'Faltante'}
              </span>
              <span style={{ fontSize: 15, fontWeight: 700, fontVariantNumeric: 'tabular-nums', color: difference >= 0 ? 'var(--success)' : 'var(--danger)' }}>
                {formatCOP(Math.abs(difference))}
              </span>
            </div>
          )}

          {hasFaltante && (
            <div style={{
              padding: '10px 14px', borderRadius: 8, marginBottom: 12,
              background: 'var(--danger-dim)', border: '1px solid var(--danger)',
              fontSize: 13, color: 'var(--danger)', fontWeight: 500,
            }}>
              No se puede cerrar la caja con faltante. Ajusta el monto contado o agrega efectivo antes de cerrar.
            </div>
          )}

          <div style={{ display: 'flex', gap: 8 }}>
            <button type="button" className="btn btn-full" onClick={onClose} disabled={loading}>
              Cancelar
            </button>
            <button type="submit" className="btn btn-full btn-danger" disabled={loading || hasFaltante || counted === ''}>
              {loading ? 'Cerrando...' : `Cerrar Caja · ${formatCOP(summary.expected)}`}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

// ─── History View ─────────────────────────────────────────────────────────────

interface RegisterListItem {
  id: number;
  opening_amount: number;
  expected_close: number | null;
  actual_close: number | null;
  difference: number | null;
  status: 'open' | 'closed';
  opened_at: string;
  closed_at: string | null;
}

function HistoryView() {
  const [registers, setRegisters] = useState<RegisterListItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [expanded, setExpanded] = useState<number | null>(null);
  const [movements, setMovements] = useState<Record<number, Movement[]>>({});
  const [loadingMv, setLoadingMv] = useState<number | null>(null);

  useEffect(() => {
    api.get<RegisterListItem[]>('/registers')
      .then(data => setRegisters(data ?? []))
      .finally(() => setLoading(false));
  }, []);

  const toggleExpand = async (id: number) => {
    if (expanded === id) { setExpanded(null); return; }
    setExpanded(id);
    if (!movements[id]) {
      setLoadingMv(id);
      try {
        const mvs = await api.get<Movement[]>(`/registers/${id}/movements`);
        setMovements(prev => ({ ...prev, [id]: mvs ?? [] }));
      } finally {
        setLoadingMv(null);
      }
    }
  };

  if (loading) return <div style={{ color: 'var(--text-muted)', fontSize: 14 }}>Cargando histórico...</div>;

  if (!registers.length) return (
    <div style={{ padding: 32, textAlign: 'center', color: 'var(--text-muted)', fontSize: 13 }}>
      No hay sesiones registradas
    </div>
  );

  return (
    <div style={{ background: 'var(--surface-1)', border: '1px solid var(--border)', borderRadius: 8, overflow: 'hidden' }}>
      <div style={{ padding: '12px 16px', borderBottom: '1px solid var(--border)' }}>
        <span style={{ fontSize: 13, fontWeight: 600 }}>Histórico de sesiones</span>
        <span style={{ fontSize: 12, color: 'var(--text-muted)', marginLeft: 8 }}>{registers.length} sesiones</span>
      </div>
      {registers.map(reg => {
        const isOpen = reg.status === 'open';
        const isExpanded = expanded === reg.id;
        const mvs = movements[reg.id];
        return (
          <div key={reg.id} style={{ borderBottom: '1px solid var(--border)' }}>
            <button
              onClick={() => toggleExpand(reg.id)}
              style={{
                width: '100%', display: 'flex', alignItems: 'center', gap: 12,
                padding: '12px 16px', background: 'none', border: 'none',
                cursor: 'pointer', textAlign: 'left',
              }}
            >
              <span style={{
                fontSize: 11, fontWeight: 600, padding: '2px 8px', borderRadius: 4,
                background: isOpen ? 'var(--success-dim)' : 'var(--surface-2)',
                color: isOpen ? 'var(--success)' : 'var(--text-muted)',
                border: `1px solid ${isOpen ? 'var(--success)' : 'var(--border)'}`,
                flexShrink: 0,
              }}>
                {isOpen ? 'ABIERTA' : 'CERRADA'}
              </span>
              <div style={{ flex: 1 }}>
                <div style={{ fontSize: 13, fontWeight: 500, color: 'var(--text-primary)' }}>
                  {formatDateTime(reg.opened_at)}
                  {reg.closed_at && <span style={{ color: 'var(--text-muted)', fontWeight: 400 }}> → {formatDateTime(reg.closed_at)}</span>}
                </div>
              </div>
              <div style={{ display: 'flex', gap: 24, flexShrink: 0 }}>
                <div style={{ textAlign: 'right' }}>
                  <div style={{ fontSize: 11, color: 'var(--text-muted)' }}>Base</div>
                  <div style={{ fontSize: 13, fontWeight: 600, fontVariantNumeric: 'tabular-nums' }}>{formatCOP(reg.opening_amount)}</div>
                </div>
                {reg.expected_close !== null && (
                  <div style={{ textAlign: 'right' }}>
                    <div style={{ fontSize: 11, color: 'var(--text-muted)' }}>Esperado</div>
                    <div style={{ fontSize: 13, fontWeight: 600, color: 'var(--accent)', fontVariantNumeric: 'tabular-nums' }}>{formatCOP(reg.expected_close)}</div>
                  </div>
                )}
                {reg.difference !== null && (
                  <div style={{ textAlign: 'right' }}>
                    <div style={{ fontSize: 11, color: 'var(--text-muted)' }}>Diferencia</div>
                    <div style={{ fontSize: 13, fontWeight: 600, fontVariantNumeric: 'tabular-nums', color: reg.difference >= 0 ? 'var(--success)' : 'var(--danger)' }}>
                      {reg.difference >= 0 ? '+' : ''}{formatCOP(reg.difference)}
                    </div>
                  </div>
                )}
              </div>
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"
                style={{ flexShrink: 0, color: 'var(--text-muted)', transform: isExpanded ? 'rotate(180deg)' : 'none', transition: '150ms' }}>
                <polyline points="6 9 12 15 18 9"/>
              </svg>
            </button>

            {isExpanded && (
              <div style={{ borderTop: '1px solid var(--border)', background: 'var(--surface-2)' }}>
                {loadingMv === reg.id ? (
                  <div style={{ padding: '12px 16px', fontSize: 13, color: 'var(--text-muted)' }}>Cargando movimientos...</div>
                ) : !mvs?.length ? (
                  <div style={{ padding: '12px 16px', fontSize: 13, color: 'var(--text-muted)' }}>Sin movimientos</div>
                ) : mvs.map(mv => {
                  const isCredit = mv.description?.startsWith('Crédito #');
                  const isIn = !isCredit && (mv.type === 'cash_in' || mv.type === 'opening' || mv.type === 'sale');
                  const dotColor = isCredit ? 'var(--accent)' : isIn ? 'var(--success)' : 'var(--danger)';
                  const amountColor = isCredit ? 'var(--accent)' : isIn ? 'var(--success)' : 'var(--danger)';
                  const amountPrefix = isCredit ? '' : isIn ? '+' : '-';
                  return (
                    <div key={mv.id} style={{ display: 'flex', alignItems: 'center', gap: 12, padding: '8px 16px 8px 32px', borderBottom: '1px solid var(--border)' }}>
                      <div style={{ width: 6, height: 6, borderRadius: '50%', flexShrink: 0, background: dotColor }} />
                      <div style={{ flex: 1 }}>
                        <div style={{ fontSize: 12, fontWeight: 500 }}>
                          {mv.description || (mv.type === 'opening' ? 'Apertura de caja' : mv.type === 'sale' ? 'Venta efectivo' : mv.type === 'cash_in' ? 'Entrada de efectivo' : 'Salida de efectivo')}
                        </div>
                        <div style={{ fontSize: 11, color: 'var(--text-muted)', marginTop: 1 }}>{formatDateTime(mv.created_at)}</div>
                      </div>
                      <div style={{ fontSize: 13, fontWeight: 600, fontVariantNumeric: 'tabular-nums', color: amountColor }}>
                        {amountPrefix}{formatCOP(mv.amount)}
                      </div>
                    </div>
                  );
                })}
              </div>
            )}
          </div>
        );
      })}
    </div>
  );
}

// ─── Page ─────────────────────────────────────────────────────────────────────

export default function RegisterPage() {
  const [summary, setSummary] = useState<RegisterSummary | null>(null);
  const [loading, setLoading] = useState(true);
  const [showOpen, setShowOpen] = useState(false);
  const [showMovement, setShowMovement] = useState(false);
  const [showClose, setShowClose] = useState(false);
  const [showHistory, setShowHistory] = useState(false);
  const [closeTicket, setCloseTicket] = useState<CloseTicketData | null>(null);
  const { toast } = useToast();

  const fetchOpen = useCallback(async () => {
    try {
      const data = await api.get<RegisterSummary | null>('/registers/open').catch(() => null);
      setSummary(data);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { fetchOpen(); }, [fetchOpen]);

  const handleOpen = async (amount: number) => {
    await api.post('/registers', { opening_amount: amount });
    toast('Caja abierta correctamente', 'success');
    setShowOpen(false);
    setLoading(true);
    await fetchOpen();
  };

  const handleMovement = async (type: 'cash_in' | 'cash_out', amount: number, description: string) => {
    if (!summary) return;
    await api.post(`/registers/${summary.register.id}/movements`, { type, amount, description });
    toast(type === 'cash_in' ? 'Entrada registrada' : 'Salida registrada', 'success');
    setShowMovement(false);
    await fetchOpen();
  };

  const handleClose = async (actualClose: number) => {
    if (!summary) return;
    const closed = await api.post<RegisterSummary>(`/registers/${summary.register.id}/close`, { actual_close: actualClose });
    toast('Caja cerrada correctamente', 'success');
    setShowClose(false);

    try {
      const business = await api.get<TicketBusiness>('/setup');
      setCloseTicket({
        business,
        cashier: getUser()?.username ?? '—',
        opened_at: closed.register.opened_at,
        closed_at: closed.register.closed_at ?? new Date().toISOString(),
        opening_amount: closed.register.opening_amount,
        by_payment_method: closed.by_payment_method ?? [],
        cash_in: closed.cash_in,
        cash_out: closed.cash_out,
        expected: closed.expected,
        actual_close: closed.register.actual_close ?? actualClose,
        difference: closed.register.difference ?? 0,
      });
    } catch {
      // Non-fatal: closing already succeeded, ticket is a convenience.
    }

    setLoading(true);
    await fetchOpen();
  };

  const isOpen = summary?.register?.status === 'open';

  const topbarActions = (
    <div style={{ display: 'flex', gap: 8 }}>
      {isOpen && !showHistory && (
        <>
          <button className="btn btn-sm" onClick={() => setShowMovement(true)}>
            <IconPlus /> Movimiento
          </button>
          <button className="btn btn-sm btn-danger" onClick={() => setShowClose(true)}>
            Cerrar Caja
          </button>
        </>
      )}
      {!isOpen && !loading && !showHistory && (
        <button className="btn-accent btn btn-sm" onClick={() => setShowOpen(true)}>
          Abrir Caja
        </button>
      )}
      <button className="btn btn-sm" onClick={() => setShowHistory(h => !h)}>
        {showHistory ? 'Ver sesión actual' : 'Ver histórico'}
      </button>
    </div>
  );

  return (
    <AppLayout>
      <div style={{ display: 'flex', flexDirection: 'column', height: '100%', overflow: 'hidden' }}>
        <Topbar
          title="Caja"
          subtitle={
            summary?.register
              ? `${isOpen ? 'Abierta' : 'Cerrada'} · ${formatDateTime(summary.register.opened_at)}`
              : 'Sin sesión activa'
          }
          actions={topbarActions}
        />

        <div style={{ flex: 1, overflow: 'auto', padding: 24 }}>
          {showHistory ? (
            <HistoryView />
          ) : loading ? (
            <div style={{ color: 'var(--text-muted)', fontSize: 14 }}>Cargando...</div>
          ) : (
            <>
              {/* Banner estado */}
              <div style={{
                display: 'flex', alignItems: 'center', gap: 10,
                background: isOpen ? 'var(--success-dim)' : 'var(--danger-dim)',
                border: `1px solid ${isOpen ? 'var(--success)' : 'var(--danger)'}`,
                borderRadius: 8, padding: '12px 16px', marginBottom: 20,
                color: isOpen ? 'var(--success)' : 'var(--danger)',
              }}>
                {isOpen ? <IconCheck /> : <IconX />}
                <span style={{ fontSize: 13, fontWeight: 500 }}>
                  {isOpen && summary
                    ? `Caja abierta · Inicio: ${formatTime(summary.register.opened_at)} · Turno de ${hoursSince(summary.register.opened_at)}h`
                    : 'Caja cerrada — abre una nueva sesión para empezar a vender'}
                </span>
              </div>

              {summary && (
                <>
                  {/* Summary grid */}
                  <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 12, marginBottom: 20 }}>
                    {[
                      { label: 'Base apertura', value: summary.register.opening_amount, color: 'var(--text-primary)' },
                      { label: 'Ventas efectivo', value: summary.cash_sales, color: 'var(--success)' },
                      { label: 'Total esperado', value: summary.expected, color: 'var(--accent)' },
                    ].map(({ label, value, color }) => (
                      <div key={label} style={{
                        background: 'var(--surface-1)', border: '1px solid var(--border)',
                        borderRadius: 8, padding: 16,
                      }}>
                        <div style={{ fontSize: 11, fontWeight: 600, color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '0.5px', marginBottom: 8 }}>
                          {label}
                        </div>
                        <div style={{ fontSize: 22, fontWeight: 700, color, fontVariantNumeric: 'tabular-nums', letterSpacing: '-0.5px' }}>
                          {formatCOP(value)}
                        </div>
                      </div>
                    ))}
                  </div>

                  {/* Cuadre (si cerrada) */}
                  {!isOpen && summary.register.actual_close !== null && (
                    <div style={{
                      background: 'var(--surface-1)', border: '1px solid var(--border)',
                      borderRadius: 8, padding: 16, marginBottom: 20,
                    }}>
                      <div style={{ fontSize: 13, fontWeight: 600, marginBottom: 12 }}>Resumen de cierre</div>
                      <div style={{ display: 'flex', gap: 24 }}>
                        {[
                          ['Monto real contado', summary.register.actual_close],
                          ['Total esperado', summary.expected],
                        ].map(([label, val]) => (
                          <div key={label as string}>
                            <div style={{ fontSize: 11, color: 'var(--text-muted)', marginBottom: 4 }}>{label as string}</div>
                            <div style={{ fontSize: 18, fontWeight: 700, fontVariantNumeric: 'tabular-nums' }}>{formatCOP(val as number)}</div>
                          </div>
                        ))}
                        <div>
                          <div style={{ fontSize: 11, color: 'var(--text-muted)', marginBottom: 4 }}>Diferencia</div>
                          <div style={{
                            fontSize: 18, fontWeight: 700, fontVariantNumeric: 'tabular-nums',
                            color: (summary.register.difference ?? 0) >= 0 ? 'var(--success)' : 'var(--danger)',
                          }}>
                            {(summary.register.difference ?? 0) >= 0 ? '+' : ''}{formatCOP(summary.register.difference ?? 0)}
                          </div>
                        </div>
                      </div>
                    </div>
                  )}

                  {/* Movimientos */}
                  <div style={{ background: 'var(--surface-1)', border: '1px solid var(--border)', borderRadius: 8, overflow: 'hidden' }}>
                    <div style={{ padding: '12px 16px', borderBottom: '1px solid var(--border)', display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                      <span style={{ fontSize: 13, fontWeight: 600 }}>Movimientos</span>
                      <span style={{ fontSize: 12, color: 'var(--text-muted)' }}>{summary.movements?.length ?? 0} registros</span>
                    </div>

                    {!summary.movements?.length ? (
                      <div style={{ padding: 32, textAlign: 'center', color: 'var(--text-muted)', fontSize: 13 }}>
                        No hay movimientos registrados
                      </div>
                    ) : (
                      summary.movements.map(mv => {
                        const isCredit = mv.description?.startsWith('Crédito #');
                        const isIn = !isCredit && (mv.type === 'cash_in' || mv.type === 'opening' || mv.type === 'sale');
                        const dotColor = isCredit ? 'var(--accent)' : isIn ? 'var(--success)' : 'var(--danger)';
                        const amountColor = isCredit ? 'var(--accent)' : isIn ? 'var(--success)' : 'var(--danger)';
                        const amountPrefix = isCredit ? '' : isIn ? '+' : '-';
                        return (
                          <div key={mv.id} style={{
                            display: 'flex', alignItems: 'center', gap: 12,
                            padding: '10px 16px', borderBottom: '1px solid var(--border)',
                          }}>
                            <div style={{
                              width: 8, height: 8, borderRadius: '50%', flexShrink: 0,
                              background: dotColor,
                            }} />
                            <div style={{ flex: 1 }}>
                              <div style={{ fontSize: 13, fontWeight: 500 }}>
                                {mv.description || (
                                  mv.type === 'opening' ? 'Apertura de caja' :
                                  mv.type === 'sale' ? 'Venta efectivo' :
                                  mv.type === 'cash_in' ? 'Entrada de efectivo' : 'Salida de efectivo'
                                )}
                              </div>
                              {isCredit && (
                                <div style={{ fontSize: 11, color: 'var(--accent)', marginTop: 1, fontWeight: 500 }}>
                                  Venta a crédito · no entra a caja
                                </div>
                              )}
                              <div style={{ fontSize: 11, color: 'var(--text-muted)', marginTop: 2 }}>
                                {formatDateTime(mv.created_at)}
                              </div>
                            </div>
                            <div style={{
                              fontSize: 14, fontWeight: 600, fontVariantNumeric: 'tabular-nums',
                              color: amountColor,
                            }}>
                              {amountPrefix}{formatCOP(mv.amount)}
                            </div>
                          </div>
                        );
                      })
                    )}
                  </div>
                </>
              )}

              {/* Estado vacío: sin caja */}
              {!summary && (
                <div style={{
                  display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center',
                  gap: 16, paddingTop: 80, color: 'var(--text-muted)',
                }}>
                  <div style={{ width: 48, height: 48, borderRadius: 12, background: 'var(--surface-3)', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                    <IconLock />
                  </div>
                  <div style={{ textAlign: 'center' }}>
                    <div style={{ fontSize: 15, fontWeight: 600, color: 'var(--text-primary)', marginBottom: 4 }}>Sin caja activa</div>
                    <div style={{ fontSize: 13 }}>Abre una nueva sesión para comenzar a registrar ventas.</div>
                  </div>
                  <button className="btn-accent btn" onClick={() => setShowOpen(true)}>
                    <IconCash /> Abrir Caja
                  </button>
                </div>
              )}
            </>
          )}
        </div>
      </div>

      {showOpen && <OpenModal onClose={() => setShowOpen(false)} onOpen={handleOpen} />}
      {showMovement && summary && (
        <MovementModal onClose={() => setShowMovement(false)} onSave={handleMovement} />
      )}
      {showClose && summary && (
        <CloseModal summary={summary} onClose={() => setShowClose(false)} onConfirm={handleClose} />
      )}
      {closeTicket && (
        <div style={{
          position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.6)',
          display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 1000,
          padding: 12,
        }}>
          <div style={{
            background: 'var(--surface-1)', border: '1px solid var(--border)',
            borderRadius: 12, padding: 24, maxHeight: '92vh', overflowY: 'auto',
            display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 16,
          }}>
            <div style={{ fontSize: 15, fontWeight: 600, color: 'var(--text-primary)', alignSelf: 'flex-start' }}>
              ✓ Caja cerrada
            </div>
            <CloseTicketPreview ticket={closeTicket} onClose={() => setCloseTicket(null)} />
          </div>
        </div>
      )}
    </AppLayout>
  );
}
