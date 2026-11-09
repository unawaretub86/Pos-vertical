'use client';

import { useEffect, useRef, useState, type CSSProperties } from 'react';
import { api } from '@/lib/api';
import {
  closedRegisterNotice,
  saleLoadErrorMessage,
  voidSale,
  voidSummary,
  type VoidPayment,
} from '@/lib/saleVoid';

interface SaleDetail {
  id: number;
  total: number;
  register_id: number | null;
  payments?: VoidPayment[];
}

interface VoidSaleDialogProps {
  saleId: number;
  customerName: string | null;
  // Id of the register open now; null when none is open.
  openRegisterId: number | null;
  onCancel: () => void;
  onVoided: (saleId: number) => void;
  // Receives the raw `Error.message` thrown by `api`.
  onFailed: (saleId: number, message: string) => void;
}

// Confirmation for an admin void (US-095). Mount it with `key={saleId}` so
// each sale starts from the loading state. Nothing is sent to the void
// endpoint until the admin taps `Anular venta`.
export default function VoidSaleDialog({
  saleId,
  customerName,
  openRegisterId,
  onCancel,
  onVoided,
  onFailed,
}: VoidSaleDialogProps) {
  const [sale, setSale] = useState<SaleDetail | null>(null);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  // Guards a second tap before the `busy` re-render lands.
  const sentRef = useRef(false);
  const cancelRef = useRef<HTMLButtonElement>(null);

  useEffect(() => {
    let cancelled = false;
    api
      .get<SaleDetail>(`/sales/${saleId}`)
      .then(s => {
        if (!cancelled) setSale(s);
      })
      .catch((err: unknown) => {
        if (!cancelled) setLoadError(saleLoadErrorMessage(err instanceof Error ? err.message : ''));
      });
    return () => {
      cancelled = true;
    };
  }, [saleId]);

  useEffect(() => {
    cancelRef.current?.focus();
  }, []);

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && !sentRef.current) onCancel();
    };
    window.addEventListener('keydown', handler);
    return () => window.removeEventListener('keydown', handler);
  }, [onCancel]);

  function cancel() {
    if (!sentRef.current) onCancel();
  }

  function confirm() {
    if (!sale || sentRef.current) return;
    sentRef.current = true;
    setBusy(true);
    voidSale(saleId)
      .then(() => onVoided(saleId))
      .catch((err: unknown) => onFailed(saleId, err instanceof Error ? err.message : ''));
  }

  const payments = sale?.payments ?? [];
  const summary = sale ? voidSummary({ id: sale.id, total: sale.total, payments }, customerName) : null;
  const notice = sale ? closedRegisterNotice(sale.register_id, openRegisterId, payments) : [];
  const textStyle: CSSProperties = {
    margin: 0,
    fontSize: '13px',
    color: 'var(--text-secondary)',
    lineHeight: '1.5',
    overflowWrap: 'anywhere',
  };

  return (
    <div
      style={{
        position: 'fixed',
        inset: 0,
        // Dims the page with the theme's own canvas color (no hard-coded color).
        background: 'color-mix(in srgb, var(--canvas) 75%, transparent)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        zIndex: 9998,
        animation: 'fadeIn 150ms ease-out',
      }}
      onClick={cancel}
    >
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby="void-sale-title"
        style={{
          background: 'var(--surface-1)',
          border: '1px solid var(--border-emphasis)',
          borderRadius: '12px',
          padding: '20px',
          width: 'min(400px, calc(100% - 32px))',
          maxHeight: 'calc(100% - 32px)',
          overflowY: 'auto',
          boxSizing: 'border-box',
          animation: 'modalIn 200ms ease-out',
        }}
        onClick={e => e.stopPropagation()}
      >
        <h3
          id="void-sale-title"
          style={{
            margin: '0 0 12px',
            fontSize: '15px',
            fontWeight: 600,
            color: 'var(--text-primary)',
            letterSpacing: '-0.3px',
            overflowWrap: 'anywhere',
          }}
        >
          {summary ? summary.title : `¿Anular la venta #${saleId}?`}
        </h3>

        <div style={{ display: 'flex', flexDirection: 'column', gap: '6px', marginBottom: '20px' }}>
          {loadError !== null ? (
            <p style={{ ...textStyle, color: 'var(--danger)' }}>{loadError}</p>
          ) : !summary ? (
            <p style={textStyle}>Cargando venta...</p>
          ) : (
            <>
              <p style={{ ...textStyle, color: 'var(--text-primary)', fontWeight: 600 }}>{summary.total}</p>
              {summary.lines.map((line, i) => (
                <p key={i} style={textStyle}>{line}</p>
              ))}
              {summary.customer && <p style={textStyle}>{summary.customer}</p>}
              <p style={textStyle}>El inventario de los productos se devuelve. Esta acción no se puede deshacer.</p>
              {notice.map((line, i) => (
                <p key={`n${i}`} style={{ ...textStyle, color: 'var(--danger)' }}>{line}</p>
              ))}
            </>
          )}
        </div>

        <div style={{ display: 'flex', gap: '8px', justifyContent: 'flex-end', flexWrap: 'wrap' }}>
          <button ref={cancelRef} className="btn btn-sm" onClick={cancel} disabled={busy}>
            Cancelar
          </button>
          <button className="btn btn-sm btn-danger" onClick={confirm} disabled={busy || !sale}>
            {busy ? 'Anulando...' : 'Anular venta'}
          </button>
        </div>
      </div>
    </div>
  );
}
