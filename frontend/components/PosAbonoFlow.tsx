'use client';

// Abono (partial payment toward a fiado balance) from the POS top bar (US-105).
// The cashier picks one customer by name or phone (no balances in the list),
// the balance of that customer only is looked up, and the shared AbonoModal
// opens when something is owed. It never touches the cart or held sales.

import { useState, useEffect } from 'react';
import { api } from '@/lib/api';
import { customerBalanceLabel, posAbonoStep, type PosAbonoStep } from '@/lib/abono';
import AbonoModal from '@/components/AbonoModal';

interface PickedCustomer {
  id: number;
  name: string;
  phone: string;
}

type Stage =
  | { kind: 'search' }
  | { kind: 'loading'; customer: PickedCustomer }
  | { kind: 'result'; customer: PickedCustomer; balance: number | null; result: PosAbonoStep };

const overlayStyle: React.CSSProperties = {
  position: 'fixed', inset: 0,
  // Dims the page with the theme's own canvas color (no hard-coded color).
  background: 'color-mix(in srgb, var(--canvas) 75%, transparent)',
  display: 'flex', alignItems: 'center', justifyContent: 'center',
  padding: '16px',
  zIndex: 1000,
};

const panelStyle: React.CSSProperties = {
  background: 'var(--surface-1)',
  border: '1px solid var(--border-emphasis)',
  borderRadius: '12px',
  padding: '16px',
  width: '420px',
  maxWidth: '100%',
  maxHeight: '100%',
  overflowY: 'auto',
  boxSizing: 'border-box',
  display: 'flex', flexDirection: 'column', gap: '12px',
};

export default function PosAbonoFlow({ onClose }: { onClose: () => void }) {
  const [stage, setStage] = useState<Stage>({ kind: 'search' });
  const [query, setQuery] = useState('');
  const [debouncedQuery, setDebouncedQuery] = useState('');
  const [results, setResults] = useState<PickedCustomer[]>([]);
  const [searching, setSearching] = useState(false);

  const searchOpen = stage.kind === 'search';

  useEffect(() => {
    const t = setTimeout(() => setDebouncedQuery(query), 250);
    return () => clearTimeout(t);
  }, [query]);

  // Same name/phone search as the checkout picker; only names and phones are shown.
  useEffect(() => {
    if (!searchOpen) return;
    queueMicrotask(() => setSearching(true));
    api.get<{ customers: PickedCustomer[] | null }>(`/customers${debouncedQuery ? `?q=${encodeURIComponent(debouncedQuery)}` : ''}`)
      .then(res => setResults(res.customers ?? []))
      .catch(() => setResults([]))
      .finally(() => setSearching(false));
  }, [searchOpen, debouncedQuery]);

  // AbonoModal handles Escape itself; this covers the search and result steps.
  const modalOpen = stage.kind === 'result' && stage.result.step === 'modal';
  useEffect(() => {
    if (modalOpen) return;
    function onKeyDown(e: KeyboardEvent) {
      if (e.key === 'Escape') onClose();
    }
    window.addEventListener('keydown', onKeyDown);
    return () => window.removeEventListener('keydown', onKeyDown);
  }, [modalOpen, onClose]);

  async function pick(customer: PickedCustomer) {
    setStage({ kind: 'loading', customer });
    let balance: number | null = null;
    try {
      const res = await api.get<{ balance: number }>(`/customers/${customer.id}/balance`);
      balance = typeof res?.balance === 'number' ? res.balance : null;
    } catch {
      balance = null;
    }
    setStage({ kind: 'result', customer, balance, result: posAbonoStep(balance) });
  }

  if (stage.kind === 'result' && stage.result.step === 'modal' && stage.balance !== null) {
    return (
      <AbonoModal
        customerId={stage.customer.id}
        customerName={stage.customer.name}
        balance={stage.balance}
        onClose={onClose}
        onSaved={() => {}}
      />
    );
  }

  return (
    <div style={overlayStyle} onClick={onClose}>
      <div style={panelStyle} onClick={e => e.stopPropagation()} role="dialog" aria-modal="true">
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: '8px' }}>
          <h2 style={{ fontSize: '15px', fontWeight: 600, color: 'var(--text-primary)', margin: 0 }}>
            Registrar abono
          </h2>
          <button type="button" className="btn btn-sm" onClick={onClose} aria-label="Cerrar">
            ×
          </button>
        </div>

        {stage.kind === 'search' && (
          <>
            <input
              autoFocus
              value={query}
              onChange={e => setQuery(e.target.value)}
              placeholder="Buscar cliente..."
              style={{ width: '100%', boxSizing: 'border-box', fontSize: '13px' }}
            />
            <div style={{ maxHeight: '260px', overflowY: 'auto', border: '1px solid var(--border)', borderRadius: '8px' }}>
              {searching ? (
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
                    onClick={() => pick(c)}
                    style={{
                      width: '100%', textAlign: 'left', padding: '10px 12px',
                      background: 'transparent', border: 'none', cursor: 'pointer',
                      display: 'flex', flexDirection: 'column', gap: '1px', borderBottom: '1px solid var(--border)',
                    }}
                    onMouseEnter={e => (e.currentTarget.style.background = 'var(--surface-3)')}
                    onMouseLeave={e => (e.currentTarget.style.background = 'transparent')}
                  >
                    <span style={{ fontSize: '13px', fontWeight: 600, color: 'var(--text-primary)', overflowWrap: 'anywhere' }}>{c.name}</span>
                    {c.phone && <span style={{ fontSize: '11px', color: 'var(--text-muted)' }}>{c.phone}</span>}
                  </button>
                ))
              )}
            </div>
          </>
        )}

        {stage.kind === 'loading' && (
          <div style={{ fontSize: '13px', color: 'var(--text-muted)' }}>Cargando...</div>
        )}

        {stage.kind === 'result' && stage.result.step !== 'modal' && (
          <>
            <div style={{ fontSize: '14px', fontWeight: 600, color: 'var(--text-primary)', overflowWrap: 'anywhere' }}>
              {stage.customer.name}
            </div>
            <div style={{
              fontSize: '13px', fontWeight: 600,
              color: stage.result.step === 'error' || stage.balance === null
                ? 'var(--danger)'
                : customerBalanceLabel(stage.balance).color,
            }}>
              {stage.result.text}
            </div>
            <button type="button" className="btn btn-sm" onClick={() => setStage({ kind: 'search' })}>
              Buscar otro cliente
            </button>
          </>
        )}
      </div>
    </div>
  );
}
