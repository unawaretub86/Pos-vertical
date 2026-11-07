'use client';

import { useState, useEffect, useCallback, useSyncExternalStore } from 'react';
import { useParams, useRouter } from 'next/navigation';
import AppLayout from '@/components/AppLayout';
import Topbar from '@/components/Topbar';
import ConfirmDialog from '@/components/ConfirmDialog';
import { api } from '@/lib/api';
import { useToast } from '@/components/Toast';
import { useBusinessConfig } from '@/lib/business-config-context';
import { shouldAutoPrint, handleTicketPrint } from '@/lib/printFlow';
import { formatCOP } from '@/lib/format';
import { ITEM_STATUS_LABEL, type Order, type OrderItem } from '@/lib/orders';
import { resolveRouteParam } from '@/lib/routeParam';
import TicketPreview, { type TicketData } from '@/components/TicketPreview';
import SalePaymentPanel, { type CheckoutData, type PaymentMethod, type Customer } from '@/components/SalePaymentPanel';

interface Product {
  id: number;
  name: string;
  sale_price: number;
}

// The browser path is only read on the client after hydration: the server
// snapshot is null, so the first render matches the exported HTML (US-086).
function subscribePath(onChange: () => void) {
  window.addEventListener('popstate', onChange);
  return () => window.removeEventListener('popstate', onChange);
}

export default function OrderClient() {
  const params = useParams<{ orderId: string }>();
  const router = useRouter();
  const { toast } = useToast();
  const { config: businessConfig } = useBusinessConfig();
  const pathname = useSyncExternalStore(subscribePath, () => window.location.pathname, () => null);
  // null until the id is known; with no id the page sends no order request.
  const orderId = resolveRouteParam(params.orderId, pathname ?? '', 'mesas');

  const [order, setOrder] = useState<Order | null>(null);
  const [products, setProducts] = useState<Product[]>([]);
  const [search, setSearch] = useState('');
  const [tab, setTab] = useState<'catalog' | 'order'>('catalog');
  const [voidTarget, setVoidTarget] = useState<OrderItem | null>(null);
  const [cancelOpen, setCancelOpen] = useState(false);
  const [reason, setReason] = useState('');
  const [busy, setBusy] = useState(false);

  const [showCheckout, setShowCheckout] = useState(false);
  const [selectedCustomer, setSelectedCustomer] = useState<Customer | null>(null);
  const [ticketData, setTicketData] = useState<TicketData | null>(null);

  const loadOrder = useCallback(async () => {
    if (!orderId) return;
    try {
      setOrder(await api.get<Order>(`/orders/${orderId}`));
    } catch (e) {
      toast(e instanceof Error ? e.message : 'Error al cargar el pedido', 'error');
    }
  }, [orderId, toast]);

  useEffect(() => {
    if (!orderId) return;
    api.get<Order>(`/orders/${orderId}`)
      .then(setOrder)
      .catch(e => toast(e instanceof Error ? e.message : 'Error al cargar el pedido', 'error'));
  }, [orderId, toast]);

  useEffect(() => {
    const q = new URLSearchParams({ active: 'true' });
    if (search.trim()) q.set('search', search.trim());
    const timer = setTimeout(() => {
      api.get<Product[]>(`/products?${q}`)
        .then(d => setProducts(d ?? []))
        .catch(() => toast('Error al cargar productos', 'error'));
    }, 250);
    return () => clearTimeout(timer);
  }, [search, toast]);

  // Every mutation reconciles with the server: no optimistic lines.
  async function mutate(fn: () => Promise<unknown>, errMsg: string) {
    if (!orderId) return;
    setBusy(true);
    try {
      await fn();
      await loadOrder();
    } catch (e) {
      toast(e instanceof Error ? e.message : errMsg, 'error');
    } finally {
      setBusy(false);
    }
  }

  const addProduct = (p: Product) =>
    mutate(() => api.post(`/orders/${orderId}/items`, { product_id: p.id, quantity: 1 }), 'No se pudo agregar el ítem');

  const patchItem = (it: OrderItem, body: Record<string, unknown>) =>
    mutate(() => api.patch(`/orders/${orderId}/items/${it.id}`, body), 'No se pudo actualizar el ítem');

  async function confirmVoid() {
    const it = voidTarget;
    setVoidTarget(null);
    if (it) await mutate(() => api.delete(`/orders/${orderId}/items/${it.id}`), 'No se pudo anular el ítem');
  }

  async function confirmCancel() {
    if (!orderId) return;
    if (!reason.trim()) {
      toast('El motivo es obligatorio', 'error');
      return;
    }
    setBusy(true);
    try {
      await api.post(`/orders/${orderId}/cancel`, { reason: reason.trim() });
      router.push('/mesas');
    } catch (e) {
      toast(e instanceof Error ? e.message : 'No se pudo cancelar el pedido', 'error');
    } finally {
      setBusy(false);
    }
  }

  async function onCheckout(data: CheckoutData) {
    if (!orderId) return;
    setBusy(true);
    try {
      const pmMap: Record<PaymentMethod, string> = {
        efectivo: 'cash',
        transferencia: 'transfer',
        tarjeta: 'card',
        credito: 'credit',
      };
      
      const payments = data.paymentLines
          .filter(l => l.amount > 0)
          .map(l => ({ method: pmMap[l.method] ?? l.method, amount: l.amount }));

      const body: Record<string, unknown> = {
        payments,
        ...(selectedCustomer && { customer_id: selectedCustomer.id }),
      };

      if (!data.isMixedMode && data.payMethod === 'efectivo') {
        body.cash_tendered = data.cashReceivedNum || (order?.total ?? 0);
        body.change_amount = data.change;
      }

      const res = await api.post<{ sale_id: number }>(`/orders/${orderId}/checkout`, body);
      
      try {
        const ticket = await api.get<TicketData>(`/sales/${res.sale_id}/ticket`);
        setTicketData(ticket);
        // Direct printing: send the ticket once, the preview stays on screen.
        if (shouldAutoPrint(businessConfig.print_method)) {
          void handleTicketPrint(businessConfig.print_method, res.sale_id, {
            post: (path, body) => api.post(path, body),
            toast,
            browserPrint: () => {},
          });
        }
      } catch {
        toast('Venta completada', 'success');
        router.push('/mesas');
      }
    } catch (err: unknown) {
      toast(err instanceof Error ? err.message : 'Error al cobrar', 'error');
    } finally {
      setBusy(false);
    }
  }

  if (ticketData) {
    return (
      <TicketPreview
        ticket={ticketData}
        onClose={() => { setTicketData(null); router.push('/mesas'); }}
      />
    );
  }

  const isOpen = order?.status === 'open';
  const items = order?.items ?? [];
  const validItemsCount = items.filter(i => i.status !== 'void').length;

  const catalog = (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '10px', minHeight: 0, flex: 1 }}>
      <input
        value={search}
        onChange={e => setSearch(e.target.value)}
        placeholder="Buscar producto"
        style={{ padding: '10px', border: '1px solid var(--border)', borderRadius: '6px', background: 'var(--surface-1)', color: 'var(--text-primary)', fontSize: '14px' }}
      />
      <div style={{ overflowY: 'auto', display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(min(100%, 130px), 1fr))', gap: '8px', alignContent: 'start' }}>
        {products.map(p => (
          <button
            key={p.id}
            disabled={!isOpen || busy}
            onClick={() => addProduct(p)}
            style={{ background: 'var(--surface-1)', border: '1px solid var(--border)', borderRadius: '8px', padding: '10px', textAlign: 'left', cursor: 'pointer', opacity: isOpen ? 1 : 0.5 }}
          >
            <div style={{ fontSize: '13px', fontWeight: 600, color: 'var(--text-primary)' }}>{p.name}</div>
            <div style={{ fontSize: '14px', fontWeight: 700, color: 'var(--accent)' }}>{formatCOP(p.sale_price)}</div>
          </button>
        ))}
      </div>
    </div>
  );

  const orderPanel = (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '8px', minHeight: 0, flex: 1 }}>
      <div style={{ overflowY: 'auto', display: 'flex', flexDirection: 'column', gap: '8px', flex: 1 }}>
        {items.length === 0 && (
          <div style={{ color: 'var(--text-muted)', fontSize: '14px' }}>Aún no hay ítems. Toca un producto para agregarlo.</div>
        )}
        {items.map(it => {
          const isVoid = it.status === 'void';
          return (
            <div key={it.id} style={{ background: 'var(--surface-1)', border: '1px solid var(--border)', borderRadius: '8px', padding: '10px', display: 'flex', flexDirection: 'column', gap: '6px', opacity: isVoid ? 0.6 : 1 }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', gap: '8px' }}>
                <span style={{ fontSize: '14px', fontWeight: 600, color: 'var(--text-primary)', textDecoration: isVoid ? 'line-through' : 'none' }}>
                  {it.product_name}
                </span>
                <span style={{ fontSize: '14px', fontWeight: 700, color: 'var(--text-primary)', fontVariantNumeric: 'tabular-nums', textDecoration: isVoid ? 'line-through' : 'none' }}>
                  {formatCOP(it.subtotal)}
                </span>
              </div>
              <div style={{ fontSize: '12px', color: 'var(--text-muted)' }}>
                {ITEM_STATUS_LABEL[it.status]} · {formatCOP(it.unit_price)} c/u
              </div>
              {!isVoid && isOpen && (
                <>
                  <div style={{ display: 'flex', alignItems: 'center', gap: '8px', flexWrap: 'wrap' }}>
                    <button className="btn btn-sm" disabled={busy || it.quantity <= 1} onClick={() => patchItem(it, { quantity: it.quantity - 1 })}>-</button>
                    <span style={{ minWidth: '24px', textAlign: 'center', color: 'var(--text-primary)' }}>{it.quantity}</span>
                    <button className="btn btn-sm" disabled={busy} onClick={() => patchItem(it, { quantity: it.quantity + 1 })}>+</button>
                    <button className="btn btn-sm" disabled={busy} onClick={() => setVoidTarget(it)} style={{ marginLeft: 'auto' }}>Anular</button>
                  </div>
                  <input
                    key={`${it.id}-${it.note}`}
                    defaultValue={it.note}
                    placeholder="Nota (ej. sin azúcar)"
                    onBlur={e => { if (e.target.value !== it.note) patchItem(it, { note: e.target.value }); }}
                    style={{ padding: '6px 8px', border: '1px solid var(--border)', borderRadius: '6px', background: 'var(--canvas)', color: 'var(--text-primary)', fontSize: '13px' }}
                  />
                </>
              )}
              {(isVoid || !isOpen) && it.note && (
                <div style={{ fontSize: '12px', color: 'var(--text-secondary)' }}>{it.note}</div>
              )}
            </div>
          );
        })}
      </div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', borderTop: '1px solid var(--border)', paddingTop: '8px' }}>
        <span style={{ fontSize: '14px', color: 'var(--text-secondary)' }}>Total</span>
        <span style={{ fontSize: '20px', fontWeight: 700, color: 'var(--text-primary)', fontVariantNumeric: 'tabular-nums' }}>
          {formatCOP(order?.total ?? 0)}
        </span>
      </div>
      {isOpen && (
        <div style={{ display: 'flex', gap: '8px', marginTop: '10px' }}>
          <button className="btn btn-sm" disabled={busy} onClick={() => { setReason(''); setCancelOpen(true); }} style={{ flex: 1 }}>
            Cancelar
          </button>
          <button className="btn btn-sm" disabled={busy || validItemsCount === 0} onClick={() => setShowCheckout(true)} style={{ flex: 2, background: 'var(--accent)', color: 'white', border: 'none' }}>
            Cobrar
          </button>
        </div>
      )}
      {order && !isOpen && (
        <div style={{ fontSize: '13px', color: 'var(--text-muted)' }}>Este pedido ya no está abierto.</div>
      )}
    </div>
  );

  const tabBtn = (id: 'catalog' | 'order', label: string) => (
    <button
      key={id}
      onClick={() => setTab(id)}
      style={{
        flex: 1, padding: '12px', border: 'none', cursor: 'pointer', fontSize: '14px',
        background: tab === id ? 'var(--canvas)' : 'var(--surface-1)',
        color: tab === id ? 'var(--accent)' : 'var(--text-muted)',
        fontWeight: tab === id ? 700 : 400,
        borderBottom: tab === id ? '2px solid var(--accent)' : '2px solid transparent',
      }}
    >
      {label}
    </button>
  );

  return (
    <AppLayout>
      <div style={{ display: 'flex', flexDirection: 'column', height: '100%', overflow: 'hidden' }}>
        <Topbar
          title={order ? `Pedido #${order.id}` : 'Pedido'}
          actions={<button className="btn btn-sm" onClick={() => router.push('/mesas')}>Volver</button>}
        />
        <div className="mesa-tabs" style={{ display: 'flex', borderBottom: '1px solid var(--border)' }}>
          {tabBtn('catalog', 'Catálogo')}
          {tabBtn('order', `Pedido (${items.filter(i => i.status !== 'void').length})`)}
        </div>
        <div className="mesa-body" style={{ flex: 1, minHeight: 0, padding: '12px', display: 'flex', gap: '16px' }}>
          {showCheckout ? (
             <div style={{ flex: 1, display: 'flex', flexDirection: 'column', overflowY: 'auto' }}>
               <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '12px' }}>
                 <div style={{ fontSize: '18px', fontWeight: 600 }}>Cobrar Pedido</div>
                 <button className="btn btn-sm" onClick={() => setShowCheckout(false)}>Volver al pedido</button>
               </div>
               <SalePaymentPanel
                 total={order?.total ?? 0}
                 cartTotal={order?.total ?? 0}
                 selectedCustomer={selectedCustomer}
                 onCustomerChange={setSelectedCustomer}
                 onCheckout={onCheckout}
                 loading={busy}
                 disabled={busy || validItemsCount === 0}
               />
             </div>
          ) : (
            <>
              <div className={tab === 'catalog' ? 'mesa-pane' : 'mesa-pane mesa-hidden'}>{catalog}</div>
              <div className={tab === 'order' ? 'mesa-pane' : 'mesa-pane mesa-hidden'}>{orderPanel}</div>
            </>
          )}
        </div>
        <style>{`
          .mesa-pane { display: flex; flex: 1; min-height: 0; min-width: 0; }
          @media (max-width: 767px) { .mesa-hidden { display: none; } }
          @media (min-width: 768px) { .mesa-tabs { display: none !important; } }
        `}</style>
      </div>

      <ConfirmDialog
        open={voidTarget !== null}
        title="Anular ítem"
        message={voidTarget ? `¿Anular ${voidTarget.product_name}? Quedará tachado en el pedido.` : ''}
        confirmLabel="Anular"
        danger
        onConfirm={confirmVoid}
        onCancel={() => setVoidTarget(null)}
      />

      {cancelOpen && (
        <div
          style={{ position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.5)', display: 'flex', alignItems: 'center', justifyContent: 'center', padding: '16px', zIndex: 50 }}
          onClick={() => setCancelOpen(false)}
        >
          <div
            onClick={e => e.stopPropagation()}
            style={{ background: 'var(--surface-1)', border: '1px solid var(--border)', borderRadius: '10px', padding: '16px', width: '100%', maxWidth: '360px', display: 'flex', flexDirection: 'column', gap: '10px' }}
          >
            <div style={{ fontSize: '15px', fontWeight: 600, color: 'var(--text-primary)' }}>Cancelar pedido</div>
            <input
              autoFocus
              value={reason}
              onChange={e => setReason(e.target.value)}
              placeholder="Motivo (obligatorio)"
              style={{ padding: '8px', border: '1px solid var(--border)', borderRadius: '6px', background: 'var(--canvas)', color: 'var(--text-primary)', fontSize: '14px' }}
            />
            <div style={{ display: 'flex', gap: '8px', justifyContent: 'flex-end' }}>
              <button className="btn btn-sm" onClick={() => setCancelOpen(false)}>Volver</button>
              <button className="btn btn-sm" disabled={busy || !reason.trim()} onClick={confirmCancel}>Cancelar pedido</button>
            </div>
          </div>
        </div>
      )}
    </AppLayout>
  );
}
