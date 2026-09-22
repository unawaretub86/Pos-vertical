'use client';

import { formatCOP } from '@/lib/format';
import type { TicketBusiness } from './TicketPreview';

// ─── Types ────────────────────────────────────────────────────────────────────

export interface CloseTicketPaymentLine {
  method: string;
  total: number;
}

export interface CloseTicketData {
  business: TicketBusiness;
  cashier: string;
  opened_at: string;
  closed_at: string;
  opening_amount: number;
  by_payment_method: CloseTicketPaymentLine[];
  cash_in: number;
  cash_out: number;
  expected: number;
  actual_close: number;
  difference: number;
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

const METHOD_LABELS: Record<string, string> = {
  cash: 'Efectivo',
  card: 'Tarjeta',
  transfer: 'Transferencia',
  credit: 'Crédito',
  mixed: 'Mixto',
};

function formatDateTime(iso: string) {
  const d = new Date(iso);
  return d.toLocaleString('es-CO', {
    day: '2-digit', month: 'short', year: 'numeric',
    hour: '2-digit', minute: '2-digit',
  });
}

// ─── Component ────────────────────────────────────────────────────────────────

interface Props {
  ticket: CloseTicketData;
  onPrint?: () => void;
  onClose?: () => void;
}

export default function CloseTicketPreview({ ticket, onPrint, onClose }: Props) {
  const isNegative = ticket.difference < 0;

  function handlePrint() {
    if (onPrint) onPrint();
    window.print();
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '16px', alignItems: 'center' }}>
      <div
        className="ticket-print-area"
        style={{
          width: '302px', /* ~80mm a 96dpi */
          maxWidth: '100%',
          background: '#fff',
          color: '#111',
          fontFamily: "'Courier New', Courier, monospace",
          fontSize: '11px',
          padding: '14px 12px',
          borderRadius: '4px',
          border: '1px solid #ddd',
          boxShadow: '0 2px 8px rgba(0,0,0,0.12)',
        }}
      >
        {/* Business header */}
        <div style={{ textAlign: 'center', marginBottom: '10px' }}>
          {ticket.business.name && (
            <div style={{ fontWeight: 700, fontSize: '12px', letterSpacing: '0.5px' }}>
              {ticket.business.name.toUpperCase()}
            </div>
          )}
          {ticket.business.address && (
            <div style={{ fontSize: '11px', marginTop: '2px', color: '#444' }}>
              {ticket.business.address}
            </div>
          )}
          {ticket.business.nit && (
            <div style={{ fontSize: '11px', color: '#444' }}>NIT: {ticket.business.nit}</div>
          )}
          <div style={{ fontWeight: 700, fontSize: '12px', marginTop: '6px', letterSpacing: '1px' }}>
            CIERRE DE CAJA
          </div>
        </div>

        <div style={{ borderTop: '1px dashed #aaa', margin: '8px 0' }} />

        {/* Session info */}
        <div style={{ fontSize: '11px', marginBottom: '2px' }}>
          <span style={{ fontWeight: 600 }}>Cajero: </span>{ticket.cashier}
        </div>
        <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: '11px', marginBottom: '2px' }}>
          <span>Apertura</span>
          <span>{formatDateTime(ticket.opened_at)}</span>
        </div>
        <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: '11px' }}>
          <span>Cierre</span>
          <span>{formatDateTime(ticket.closed_at)}</span>
        </div>

        <div style={{ borderTop: '1px dashed #aaa', margin: '8px 0' }} />

        {/* Sales by payment method */}
        <div style={{ fontSize: '11px', fontWeight: 600, marginBottom: '4px' }}>Ventas por método</div>
        {ticket.by_payment_method.length === 0 ? (
          <div style={{ fontSize: '11px', color: '#555', marginBottom: '4px' }}>Sin ventas en el turno</div>
        ) : (
          ticket.by_payment_method.map(line => (
            <div key={line.method} style={{ display: 'flex', justifyContent: 'space-between', fontSize: '11px', marginBottom: '2px' }}>
              <span>{METHOD_LABELS[line.method] ?? line.method}</span>
              <span style={{ fontVariantNumeric: 'tabular-nums' }}>{formatCOP(line.total)}</span>
            </div>
          ))
        )}

        <div style={{ borderTop: '1px dashed #aaa', margin: '8px 0' }} />

        {/* Movements */}
        <div style={{ fontSize: '11px' }}>
          {[
            ['Base apertura', ticket.opening_amount],
            ['Entradas', ticket.cash_in],
            ['Salidas', -ticket.cash_out],
          ].map(([label, val]) => (
            <div key={label as string} style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '2px' }}>
              <span>{label as string}</span>
              <span style={{ fontVariantNumeric: 'tabular-nums' }}>{formatCOP(val as number)}</span>
            </div>
          ))}
        </div>

        <div style={{ borderTop: '1px dashed #aaa', margin: '8px 0' }} />

        {/* Totals */}
        <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: '11px', marginBottom: '2px' }}>
          <span>Esperado</span>
          <span style={{ fontVariantNumeric: 'tabular-nums' }}>{formatCOP(ticket.expected)}</span>
        </div>
        <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: '11px', marginBottom: '2px' }}>
          <span>Real contado</span>
          <span style={{ fontVariantNumeric: 'tabular-nums' }}>{formatCOP(ticket.actual_close)}</span>
        </div>
        <div style={{
          display: 'flex', justifyContent: 'space-between', fontWeight: 700, fontSize: '12px', marginTop: '4px',
          color: isNegative ? '#b91c1c' : '#111',
        }}>
          <span>Diferencia</span>
          <span style={{ fontVariantNumeric: 'tabular-nums' }}>
            {ticket.difference >= 0 ? '+' : ''}{formatCOP(ticket.difference)}
          </span>
        </div>

        <div style={{ borderTop: '1px dashed #aaa', margin: '8px 0' }} />
        <div style={{ textAlign: 'center', fontSize: '10px', color: '#888' }}>
          Firma: _____________________
        </div>
      </div>

      {/* Action buttons (hidden when printing) */}
      <div className="ticket-no-print" style={{ display: 'flex', gap: '8px', flexWrap: 'wrap', justifyContent: 'center' }}>
        <button
          onClick={handlePrint}
          style={{
            padding: '8px 20px', borderRadius: '6px', border: 'none',
            background: 'var(--accent)', color: '#000', fontWeight: 600,
            fontSize: '13px', cursor: 'pointer',
          }}
        >
          Imprimir
        </button>
        {onClose && (
          <button
            onClick={onClose}
            style={{
              padding: '8px 20px', borderRadius: '6px', border: '1px solid var(--border)',
              background: 'var(--surface-2)', color: 'var(--text-primary)',
              fontWeight: 500, fontSize: '13px', cursor: 'pointer',
            }}
          >
            Cerrar
          </button>
        )}
      </div>
    </div>
  );
}
