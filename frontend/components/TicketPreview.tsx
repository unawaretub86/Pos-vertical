'use client';

import { formatCOP } from '@/lib/format';

// ─── Types ────────────────────────────────────────────────────────────────────

export interface TicketBusiness {
  name: string;
  address: string;
  nit: string;
  ticket_header: string;
  ticket_footer: string;
}

export interface TicketItem {
  product_name: string;
  quantity: number;
  unit_price: number;
  discount: number;
  subtotal: number;
  unit: string;
}

export interface TicketData {
  sale_id: number;
  created_at: string;
  payment_method: string;
  cash_tendered?: number;
  change_amount?: number;
  reference?: string;
  status: string;
  subtotal: number;
  discount: number;
  tax: number;
  total: number;
  items: TicketItem[];
  customer?: { id: number; name: string; phone: string };
  business: TicketBusiness;
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

const formatQty = (qty: number, unit: string) => {
  const unitLabels: Record<string, string> = { unit: 'und', kg: 'kg', lb: 'lb', liter: 'lt' };
  const label = unitLabels[unit] ?? unit;
  return `${qty % 1 === 0 ? qty.toFixed(0) : qty.toFixed(2)} ${label}`;
};

const METHOD_LABELS: Record<string, string> = {
  cash: 'Efectivo',
  card: 'Tarjeta',
  transfer: 'Transferencia',
  credit: 'Crédito',
};

// ─── Component ────────────────────────────────────────────────────────────────

interface Props {
  ticket: TicketData;
  onPrint?: () => void;
  onClose?: () => void;
}

export default function TicketPreview({ ticket, onPrint, onClose }: Props) {
  const isCredit = ticket.payment_method === 'credit';
  const isVoided = ticket.status === 'voided';

  const date = new Date(ticket.created_at);
  const dateStr = date.toLocaleDateString('es-CO', { day: '2-digit', month: 'short', year: 'numeric' });
  const timeStr = date.toLocaleTimeString('es-CO', { hour: '2-digit', minute: '2-digit', hour12: true });

  function handlePrint() {
    if (onPrint) onPrint();
    window.print();
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '16px', alignItems: 'center' }}>
      {/* Ticket paper */}
      <div
        className="ticket-print-area"
        style={{
          width: '216px',  /* ~58mm a 96dpi */
          background: '#fff',
          color: '#111',
          fontFamily: "'Courier New', Courier, monospace",
          fontSize: '11px',
          padding: '12px 10px',
          borderRadius: '4px',
          border: '1px solid #ddd',
          boxShadow: '0 2px 8px rgba(0,0,0,0.12)',
        }}
      >
        {/* ANULADO banner */}
        {isVoided && (
          <div style={{
            textAlign: 'center',
            background: '#fee2e2',
            color: '#b91c1c',
            fontWeight: 700,
            fontSize: '11px',
            padding: '6px',
            marginBottom: '10px',
            border: '1px solid #fca5a5',
            letterSpacing: '2px',
          }}>
            *** ANULADO ***
          </div>
        )}

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
            <div style={{ fontSize: '11px', color: '#444' }}>
              NIT: {ticket.business.nit}
            </div>
          )}
          {ticket.business.ticket_header && (
            <div style={{ fontSize: '11px', marginTop: '4px', color: '#555', fontStyle: 'italic', whiteSpace: 'pre-wrap' }}>
              {ticket.business.ticket_header}
            </div>
          )}
        </div>

        <div style={{ borderTop: '1px dashed #aaa', margin: '8px 0' }} />

        {/* Sale info */}
        <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: '11px', marginBottom: '2px' }}>
          <span>Venta #{ticket.sale_id}</span>
          <span>{dateStr}</span>
        </div>
        <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: '11px', marginBottom: '6px' }}>
          <span>Método: {METHOD_LABELS[ticket.payment_method] ?? ticket.payment_method}</span>
          <span>{timeStr}</span>
        </div>

        {/* CRÉDITO banner */}
        {isCredit && (
          <div style={{
            textAlign: 'center',
            border: '2px solid #111',
            padding: '4px 6px',
            marginBottom: '6px',
            fontWeight: 700,
            fontSize: '12px',
            letterSpacing: '2px',
          }}>
            *** CRÉDITO ***
          </div>
        )}

        {/* Customer for credit */}
        {ticket.customer && (
          <div style={{ fontSize: '11px', marginBottom: '6px' }}>
            <span style={{ fontWeight: 600 }}>Cliente: </span>{ticket.customer.name}
            {ticket.customer.phone && ` · ${ticket.customer.phone}`}
          </div>
        )}

        <div style={{ borderTop: '1px dashed #aaa', margin: '8px 0' }} />

        {/* Items */}
        <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '11px' }}>
          <thead>
            <tr>
              <th style={{ textAlign: 'left', paddingBottom: '4px', fontWeight: 600 }}>Producto</th>
              <th style={{ textAlign: 'right', paddingBottom: '4px', fontWeight: 600 }}>Subtotal</th>
            </tr>
          </thead>
          <tbody>
            {ticket.items.map((item, i) => (
              <tr key={i}>
                <td style={{ paddingBottom: '3px', paddingRight: '8px' }}>
                  <div>{item.product_name}</div>
                  <div style={{ color: '#555', fontSize: '10px' }}>
                    {formatQty(item.quantity, item.unit)} × {formatCOP(item.unit_price)}
                    {item.discount > 0 && ` − ${formatCOP(item.discount)}`}
                  </div>
                </td>
                <td style={{ textAlign: 'right', verticalAlign: 'top', paddingBottom: '3px', fontVariantNumeric: 'tabular-nums' }}>
                  {formatCOP(item.subtotal)}
                </td>
              </tr>
            ))}
          </tbody>
        </table>

        <div style={{ borderTop: '1px dashed #aaa', margin: '8px 0' }} />

        {/* Totals */}
        <div style={{ fontSize: '11px' }}>
          {ticket.discount > 0 && (
            <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '2px' }}>
              <span>Descuento</span>
              <span style={{ fontVariantNumeric: 'tabular-nums' }}>−{formatCOP(ticket.discount)}</span>
            </div>
          )}
          {ticket.tax > 0 && (
            <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '2px' }}>
              <span>IVA</span>
              <span style={{ fontVariantNumeric: 'tabular-nums' }}>{formatCOP(ticket.tax)}</span>
            </div>
          )}
        </div>
        <div style={{ display: 'flex', justifyContent: 'space-between', fontWeight: 700, fontSize: '12px', marginTop: '4px' }}>
          <span>TOTAL</span>
          <span style={{ fontVariantNumeric: 'tabular-nums' }}>{formatCOP(ticket.total)}</span>
        </div>

        {/* Cash change */}
        {ticket.payment_method === 'cash' && ticket.cash_tendered != null && (
          <div style={{ fontSize: '11px', marginTop: '6px' }}>
            <div style={{ display: 'flex', justifyContent: 'space-between' }}>
              <span>Recibido</span>
              <span style={{ fontVariantNumeric: 'tabular-nums' }}>{formatCOP(ticket.cash_tendered)}</span>
            </div>
            <div style={{ display: 'flex', justifyContent: 'space-between', fontWeight: 600 }}>
              <span>Cambio</span>
              <span style={{ fontVariantNumeric: 'tabular-nums' }}>{formatCOP(ticket.change_amount ?? 0)}</span>
            </div>
          </div>
        )}

        {/* Reference */}
        {ticket.reference && (
          <div style={{ fontSize: '11px', marginTop: '6px', color: '#555' }}>
            Ref: {ticket.reference}
          </div>
        )}

        {/* Footer */}
        {ticket.business.ticket_footer && (
          <>
            <div style={{ borderTop: '1px dashed #aaa', margin: '8px 0' }} />
            <div style={{ textAlign: 'center', fontSize: '11px', color: '#555', fontStyle: 'italic', whiteSpace: 'pre-wrap' }}>
              {ticket.business.ticket_footer}
            </div>
          </>
        )}

        <div style={{ borderTop: '1px dashed #aaa', margin: '8px 0' }} />
        <div style={{ textAlign: 'center', fontSize: '10px', color: '#888' }}>
          {dateStr} {timeStr}
        </div>
      </div>

      {/* Action buttons (hidden when printing) */}
      <div className="ticket-no-print" style={{ display: 'flex', gap: '8px' }}>
        <button
          onClick={handlePrint}
          style={{
            padding: '8px 20px',
            borderRadius: '6px',
            border: 'none',
            background: 'var(--accent)',
            color: '#000',
            fontWeight: 600,
            fontSize: '13px',
            cursor: 'pointer',
          }}
        >
          Imprimir
        </button>
        {onClose && (
          <button
            onClick={onClose}
            style={{
              padding: '8px 20px',
              borderRadius: '6px',
              border: '1px solid var(--border)',
              background: 'var(--surface-2)',
              color: 'var(--text-primary)',
              fontWeight: 500,
              fontSize: '13px',
              cursor: 'pointer',
            }}
          >
            Cerrar
          </button>
        )}
      </div>
    </div>
  );
}
