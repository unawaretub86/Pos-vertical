'use client';

import { useState, useEffect, useCallback, useRef } from 'react';
import { useRouter } from 'next/navigation';
import { useIsMobile } from '@/hooks/useIsMobile';
import AppLayout from '@/components/AppLayout';
import Topbar from '@/components/Topbar';
import { api } from '@/lib/api';
import { useToast } from '@/components/Toast';
import { formatCOP, formatInputCOP, parseInputCOP } from '@/lib/format';
import LoadingSpinner from '@/components/LoadingSpinner';
import EmptyState from '@/components/EmptyState';

// ─── Types ───────────────────────────────────────────────────────────────────

interface Supplier {
  id: number;
  name: string;
}

interface Product {
  id: number;
  name: string;
  unit: string;
  cost: number;
  stock: number;
}

interface PurchaseItem {
  product_id: number;
  product_name: string;
  unit: string;
  quantity: number;
  unit_cost: number;
}

interface Purchase {
  id: number;
  supplier_id: number;
  supplier_name: string;
  reference: string;
  total: number;
  created_at: string;
}

// ─── Icons ───────────────────────────────────────────────────────────────────

const IconTrash = () => (
  <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
    <polyline points="3 6 5 6 21 6"/><path d="M19 6l-1 14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2L5 6"/>
    <path d="M10 11v6"/><path d="M14 11v6"/><path d="M9 6V4h6v2"/>
  </svg>
);
const IconBox = () => (
  <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
    <path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/>
  </svg>
);

// ─── Product picker (debounced search + dropdown) ──────────────────────────────

function ProductPicker({ onPick }: { onPick: (p: Product) => void }) {
  const [query, setQuery] = useState('');
  const [results, setResults] = useState<Product[]>([]);
  const [open, setOpen] = useState(false);
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  function handleChange(val: string) {
    setQuery(val);
    if (debounceRef.current) clearTimeout(debounceRef.current);
    if (!val.trim()) { setResults([]); setOpen(false); return; }
    debounceRef.current = setTimeout(async () => {
      try {
        const data = await api.get<Product[]>(`/products?search=${encodeURIComponent(val)}`);
        setResults(data ?? []);
        setOpen(true);
      } catch {
        setResults([]);
      }
    }, 250);
  }

  return (
    <div style={{ position: 'relative' }}>
      <input
        value={query}
        onChange={e => handleChange(e.target.value)}
        onFocus={() => results.length > 0 && setOpen(true)}
        onBlur={() => setTimeout(() => setOpen(false), 150)}
        placeholder="Buscar producto por nombre o código..."
        style={{ width: '100%', boxSizing: 'border-box' }}
      />
      {open && results.length > 0 && (
        <div style={{
          position: 'absolute', top: '100%', left: 0, right: 0, zIndex: 20,
          background: 'var(--surface-1)', border: '1px solid var(--border)', borderRadius: 8,
          marginTop: 4, maxHeight: 240, overflowY: 'auto', boxShadow: '0 8px 24px rgba(0,0,0,0.25)',
        }}>
          {results.map(p => (
            <button
              key={p.id}
              type="button"
              onMouseDown={() => { onPick(p); setQuery(''); setResults([]); setOpen(false); }}
              style={{
                display: 'flex', justifyContent: 'space-between', width: '100%', textAlign: 'left',
                padding: '8px 12px', background: 'none', border: 'none', cursor: 'pointer',
                fontSize: 13, color: 'var(--text-primary)', borderBottom: '1px solid var(--border)',
              }}
            >
              <span>{p.name}</span>
              <span style={{ color: 'var(--text-muted)', fontSize: 12 }}>Costo actual: {formatCOP(p.cost)}</span>
            </button>
          ))}
        </div>
      )}
    </div>
  );
}

// ─── Purchase Form ──────────────────────────────────────────────────────────────

function PurchaseForm({ suppliers, onSaved }: { suppliers: Supplier[]; onSaved: () => void }) {
  const { toast } = useToast();
  const isMobile = useIsMobile();
  const [supplierId, setSupplierId] = useState<number | ''>('');
  const [reference, setReference] = useState('');
  const [items, setItems] = useState<PurchaseItem[]>([]);
  const [loading, setLoading] = useState(false);

  const total = items.reduce((sum, i) => sum + i.quantity * i.unit_cost, 0);

  function addProduct(p: Product) {
    if (items.some(i => i.product_id === p.id)) {
      toast('Ese producto ya está en la compra', 'error');
      return;
    }
    setItems(prev => [...prev, { product_id: p.id, product_name: p.name, unit: p.unit, quantity: 1, unit_cost: p.cost }]);
  }

  function updateItem(productId: number, field: 'quantity' | 'unit_cost', value: number) {
    setItems(prev => prev.map(i => i.product_id === productId ? { ...i, [field]: value } : i));
  }

  function removeItem(productId: number) {
    setItems(prev => prev.filter(i => i.product_id !== productId));
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!supplierId || items.length === 0) return;
    setLoading(true);
    try {
      await api.post('/purchases', {
        supplier_id: supplierId,
        reference,
        items: items.map(i => ({ product_id: i.product_id, quantity: i.quantity, unit_cost: i.unit_cost })),
      });
      toast('Compra registrada — stock y costo actualizados', 'success');
      setSupplierId('');
      setReference('');
      setItems([]);
      onSaved();
    } catch (err: unknown) {
      toast((err as Error).message, 'error');
    } finally {
      setLoading(false);
    }
  }

  return (
    <form onSubmit={handleSubmit} style={{ background: 'var(--surface-1)', border: '1px solid var(--border)', borderRadius: 8, padding: 16, marginBottom: 20 }}>
      <div style={{ display: 'grid', gridTemplateColumns: isMobile ? '1fr' : '1fr 1fr', gap: 12, marginBottom: 12 }}>
        <div>
          <label style={{ fontSize: 12, fontWeight: 600, color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '0.5px', display: 'block', marginBottom: 6 }}>
            Proveedor *
          </label>
          <select
            className="input"
            value={supplierId}
            onChange={e => setSupplierId(e.target.value ? Number(e.target.value) : '')}
            style={{ width: '100%', boxSizing: 'border-box' }}
            required
          >
            <option value="">Selecciona un proveedor...</option>
            {suppliers.map(s => <option key={s.id} value={s.id}>{s.name}</option>)}
          </select>
        </div>
        <div>
          <label style={{ fontSize: 12, fontWeight: 600, color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '0.5px', display: 'block', marginBottom: 6 }}>
            Referencia / factura (opcional)
          </label>
          <input value={reference} onChange={e => setReference(e.target.value)} placeholder="Ej: FAC-00123" style={{ width: '100%', boxSizing: 'border-box' }} />
        </div>
      </div>

      <label style={{ fontSize: 12, fontWeight: 600, color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '0.5px', display: 'block', marginBottom: 6 }}>
        Agregar producto
      </label>
      <ProductPicker onPick={addProduct} />

      {items.length > 0 && (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 8, marginTop: 12, marginBottom: 12 }}>
          {items.map(item => (
            <div key={item.product_id} style={{
              display: 'flex', alignItems: 'center', gap: 8, flexWrap: isMobile ? 'wrap' : 'nowrap',
              background: 'var(--surface-2)', border: '1px solid var(--border)', borderRadius: 6, padding: '8px 10px',
            }}>
              <span style={{ flex: isMobile ? '1 1 100%' : 1, fontSize: 13, fontWeight: 500 }}>{item.product_name}</span>
              <input
                type="text" inputMode="decimal" value={item.quantity || ''}
                onChange={e => updateItem(item.product_id, 'quantity', parseFloat(e.target.value.replace(/[^0-9.]/g, '')) || 0)}
                placeholder="Cant."
                style={{ width: 70, boxSizing: 'border-box', textAlign: 'right' }}
              />
              <span style={{ fontSize: 11, color: 'var(--text-muted)', width: 28 }}>{item.unit}</span>
              <input
                type="text" inputMode="numeric" value={item.unit_cost ? formatInputCOP(String(item.unit_cost)) : ''}
                onChange={e => updateItem(item.product_id, 'unit_cost', parseInputCOP(e.target.value))}
                placeholder="Costo unit."
                style={{ width: 110, boxSizing: 'border-box', textAlign: 'right' }}
              />
              <span style={{ width: 100, textAlign: 'right', fontSize: 13, fontVariantNumeric: 'tabular-nums', fontWeight: 600 }}>
                {formatCOP(item.quantity * item.unit_cost)}
              </span>
              <button type="button" onClick={() => removeItem(item.product_id)} className="btn btn-sm" title="Quitar">
                <IconTrash />
              </button>
            </div>
          ))}
        </div>
      )}

      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', borderTop: '1px solid var(--border)', paddingTop: 12 }}>
        <div style={{ fontSize: 14, fontWeight: 700 }}>
          Total: <span style={{ color: 'var(--accent)', fontVariantNumeric: 'tabular-nums' }}>{formatCOP(total)}</span>
        </div>
        <button type="submit" className="btn btn-accent" disabled={loading || !supplierId || items.length === 0}>
          {loading ? 'Guardando...' : 'Registrar compra'}
        </button>
      </div>
    </form>
  );
}

// ─── Main Page ────────────────────────────────────────────────────────────────

export default function PurchasesPage() {
  const { toast } = useToast();
  const router = useRouter();
  const isMobile = useIsMobile();
  const [suppliers, setSuppliers] = useState<Supplier[]>([]);
  const [purchases, setPurchases] = useState<Purchase[]>([]);
  const [loading, setLoading] = useState(true);
  const [filterSupplier, setFilterSupplier] = useState<number | ''>('');
  const [filterFrom, setFilterFrom] = useState('');
  const [filterTo, setFilterTo] = useState('');

  const loadPurchases = useCallback(async () => {
    setLoading(true);
    try {
      const params = new URLSearchParams();
      if (filterSupplier) params.set('supplier_id', String(filterSupplier));
      if (filterFrom) params.set('from', new Date(filterFrom).toISOString());
      if (filterTo) params.set('to', new Date(filterTo + 'T23:59:59').toISOString());
      const data = await api.get<Purchase[]>(`/purchases?${params}`);
      setPurchases(data ?? []);
    } catch (err: unknown) {
      toast((err as Error).message, 'error');
    } finally {
      setLoading(false);
    }
  }, [filterSupplier, filterFrom, filterTo, toast]);

  useEffect(() => {
    api.get<Supplier[]>('/suppliers').then(data => setSuppliers(data ?? [])).catch(() => {});
  }, []);

  useEffect(() => { queueMicrotask(loadPurchases); }, [loadPurchases]);

  const hasFilters = filterSupplier !== '' || filterFrom !== '' || filterTo !== '';

  return (
    <AppLayout>
      <div style={{ display: 'flex', flexDirection: 'column', height: '100%', overflow: 'hidden' }}>
        <Topbar
          title="Compras a proveedores"
          subtitle={loading ? 'Cargando...' : `${purchases.length} compra(s)`}
          actions={
            <button className="btn btn-sm" onClick={() => router.push('/suppliers')}>Proveedores</button>
          }
        />

        <div style={{ flex: 1, overflowY: 'auto', padding: isMobile ? '12px' : '24px' }}>
          {suppliers.length === 0 && !loading ? (
            <EmptyState
              icon={<IconBox />}
              title="Aún no tienes proveedores"
              description="Crea un proveedor primero para poder registrar una compra."
            />
          ) : (
            <PurchaseForm suppliers={suppliers} onSaved={loadPurchases} />
          )}

          {/* Filters */}
          <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', marginBottom: 12, alignItems: 'center' }}>
            <select className="input" value={filterSupplier} onChange={e => setFilterSupplier(e.target.value ? Number(e.target.value) : '')}>
              <option value="">Todos los proveedores</option>
              {suppliers.map(s => <option key={s.id} value={s.id}>{s.name}</option>)}
            </select>
            <input type="date" className="input" value={filterFrom} onChange={e => setFilterFrom(e.target.value)} />
            <input type="date" className="input" value={filterTo} onChange={e => setFilterTo(e.target.value)} />
            {hasFilters && (
              <button className="btn btn-sm" onClick={() => { setFilterSupplier(''); setFilterFrom(''); setFilterTo(''); }}>
                Limpiar filtros
              </button>
            )}
          </div>

          {loading ? (
            <LoadingSpinner message="Cargando compras..." />
          ) : purchases.length === 0 ? (
            <EmptyState
              icon={<IconBox />}
              title={hasFilters ? 'No hay compras con esos filtros' : 'No hay compras registradas'}
            />
          ) : (
            <div className="table-wrap">
              <table style={{ width: '100%', borderCollapse: 'collapse' }}>
                <thead>
                  <tr>
                    <th style={{ textAlign: 'left' }}>Fecha</th>
                    <th style={{ textAlign: 'left' }}>Proveedor</th>
                    {!isMobile && <th style={{ textAlign: 'left' }}>Referencia</th>}
                    <th style={{ textAlign: 'right' }}>Total</th>
                  </tr>
                </thead>
                <tbody>
                  {purchases.map(p => (
                    <tr key={p.id}>
                      <td style={{ fontSize: 13, padding: '10px 8px' }}>
                        {new Date(p.created_at).toLocaleDateString('es-CO', { day: '2-digit', month: 'short', year: 'numeric' })}
                      </td>
                      <td style={{ fontSize: 13, fontWeight: 500 }}>{p.supplier_name}</td>
                      {!isMobile && <td style={{ fontSize: 12, color: 'var(--text-muted)' }}>{p.reference || '—'}</td>}
                      <td style={{ textAlign: 'right', fontSize: 13, fontWeight: 600, fontVariantNumeric: 'tabular-nums' }}>
                        {formatCOP(p.total)}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      </div>
    </AppLayout>
  );
}
