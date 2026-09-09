'use client';

import { useState, useEffect, useCallback, useRef } from 'react';
import { useIsMobile } from '@/hooks/useIsMobile';
import AppLayout from '@/components/AppLayout';
import Topbar from '@/components/Topbar';
import { api } from '@/lib/api';
import { useToast } from '@/components/Toast';
import { formatCOP, formatInputCOP, parseInputCOP } from '@/lib/format';

// ─── Types ───────────────────────────────────────────────────────────────────

interface Category {
  id: number;
  name: string;
  description: string;
}

interface Product {
  id: number;
  name: string;
  code: string;
  barcode: string;
  category_id: number;
  category_name?: string;
  type: 'product' | 'service';
  sale_price: number;
  wholesale_price: number | null;
  cost: number;
  unit: string;
  stock: number;
  min_stock: number;
  tax_rate: number;
  active: boolean;
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

function stockClass(stock: number, minStock: number) {
  if (stock <= 0) return 'stock-out';
  if (stock <= minStock) return 'stock-low';
  return 'stock-ok';
}

function stockLabel(stock: number, minStock: number) {
  if (stock <= 0) return 'Sin stock';
  if (stock <= minStock) return 'Bajo';
  return 'OK';
}

// ─── Icons ───────────────────────────────────────────────────────────────────

const IconPlus = () => (
  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
    <line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/>
  </svg>
);
const IconSearch = () => (
  <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
    <circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/>
  </svg>
);
const IconEdit = () => (
  <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
    <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/>
    <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/>
  </svg>
);
const IconTrash = () => (
  <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
    <polyline points="3 6 5 6 21 6"/><path d="M19 6l-1 14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2L5 6"/>
    <path d="M10 11v6"/><path d="M14 11v6"/><path d="M9 6V4h6v2"/>
  </svg>
);
const IconX = () => (
  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
    <line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/>
  </svg>
);
const IconBoxIn = () => (
  <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
    <path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/>
    <polyline points="12 22 12 12"/><polyline points="7 9.5 12 12 17 9.5"/>
  </svg>
);

// ─── Modal ────────────────────────────────────────────────────────────────────

function Modal({ title, onClose, children }: { title: string; onClose: () => void; children: React.ReactNode }) {
  return (
    <div style={{
      position: 'fixed', inset: 0, zIndex: 1000,
      background: 'rgba(0,0,0,0.6)',
      display: 'flex', alignItems: 'center', justifyContent: 'center',
    }} onClick={e => e.target === e.currentTarget && onClose()}>
      <div style={{
        background: 'var(--surface-1)',
        border: '1px solid var(--border)',
        borderRadius: '12px',
        padding: '24px',
        width: '480px',
        maxWidth: '94vw',
        boxShadow: '0 8px 32px rgba(0,0,0,0.4)',
        display: 'flex',
        flexDirection: 'column',
        gap: '20px',
      }}>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <div style={{ fontSize: '15px', fontWeight: 600, color: 'var(--text-primary)' }}>{title}</div>
          <button className="btn btn-icon" onClick={onClose} style={{ color: 'var(--text-muted)' }}><IconX /></button>
        </div>
        {children}
      </div>
    </div>
  );
}

// ─── Category Modal ───────────────────────────────────────────────────────────

function CategoryModal({
  cat,
  onClose,
  onSaved,
}: { cat: Category | null; onClose: () => void; onSaved: () => void }) {
  const { toast } = useToast();
  const [name, setName] = useState(cat?.name ?? '');
  const [description, setDescription] = useState(cat?.description ?? '');
  const [loading, setLoading] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!name.trim()) return;
    setLoading(true);
    try {
      if (cat) {
        await api.put(`/categories/${cat.id}`, { name: name.trim(), description: description.trim() });
        toast('Categoría actualizada', 'success');
      } else {
        await api.post('/categories', { name: name.trim(), description: description.trim() });
        toast('Categoría creada', 'success');
      }
      onSaved();
      onClose();
    } catch (err: unknown) {
      toast((err as Error).message || 'Error al guardar', 'error');
    } finally {
      setLoading(false);
    }
  }

  return (
    <Modal title={cat ? 'Editar categoría' : 'Nueva categoría'} onClose={onClose}>
      <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: '14px' }}>
        <div style={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
          <label style={{ fontSize: '12px', fontWeight: 600, color: 'var(--text-secondary)', textTransform: 'uppercase', letterSpacing: '0.4px' }}>Nombre *</label>
          <input className="input" value={name} onChange={e => setName(e.target.value)} placeholder="Ej: Bebidas, Snacks..." autoFocus required />
        </div>
        <div style={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
          <label style={{ fontSize: '12px', fontWeight: 600, color: 'var(--text-secondary)', textTransform: 'uppercase', letterSpacing: '0.4px' }}>Descripción</label>
          <input className="input" value={description} onChange={e => setDescription(e.target.value)} placeholder="Opcional" />
        </div>
        <div style={{ display: 'flex', gap: '8px', justifyContent: 'flex-end', marginTop: '4px' }}>
          <button type="button" className="btn" onClick={onClose}>Cancelar</button>
          <button type="submit" className="btn btn-accent" disabled={loading || !name.trim()}>
            {loading ? 'Guardando...' : cat ? 'Guardar cambios' : 'Crear categoría'}
          </button>
        </div>
      </form>
    </Modal>
  );
}

// ─── Product Modal ────────────────────────────────────────────────────────────

const UNITS = ['unit', 'kg', 'lb', 'liter'];
const UNIT_LABELS: Record<string, string> = { unit: 'Unidad', kg: 'Kilogramo', lb: 'Libra', liter: 'Litro' };

function ProductModal({
  product,
  categories,
  onClose,
  onSaved,
}: { product: Product | null; categories: Category[]; onClose: () => void; onSaved: () => void }) {
  const { toast } = useToast();
  const [loading, setLoading] = useState(false);
  const [form, setForm] = useState({
    name: product?.name ?? '',
    code: product?.code ?? '',
    barcode: product?.barcode ?? '',
    category_id: product?.category_id ?? (categories[0]?.id ?? 0),
    type: product?.type ?? 'product',
    sale_price: product?.sale_price ? formatInputCOP(String(Math.round(product.sale_price))) : '',
    wholesale_price: product?.wholesale_price ? formatInputCOP(String(Math.round(product.wholesale_price))) : '',
    cost: product?.cost ? formatInputCOP(String(Math.round(product.cost))) : '',
    unit: product?.unit ?? 'unit',
    stock: product?.stock ?? 0,
    min_stock: product?.min_stock ?? 5,
    tax_rate: product?.tax_rate ? product.tax_rate * 100 : 0, // stored as fraction (0.19); field is a % input
    active: product?.active ?? true,
  });

  function set(field: string, value: unknown) {
    setForm(prev => ({ ...prev, [field]: value }));
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!form.name.trim()) return;
    setLoading(true);
    try {
      const body = {
        ...form,
        name: form.name.trim(),
        code: form.code.trim(),
        barcode: form.barcode.trim() || null,
        category_id: Number(form.category_id) || null,
        sale_price: parseInputCOP(String(form.sale_price)),
        wholesale_price: form.wholesale_price ? parseInputCOP(String(form.wholesale_price)) : null,
        cost: parseInputCOP(String(form.cost)),
        stock: Number(form.stock),
        min_stock: Number(form.min_stock),
        tax_rate: Number(form.tax_rate) / 100, // % input back to fraction for storage
      };
      if (product) {
        await api.put(`/products/${product.id}`, body);
        toast('Producto actualizado', 'success');
      } else {
        await api.post('/products', body);
        toast('Producto creado', 'success');
      }
      onSaved();
      onClose();
    } catch (err: unknown) {
      toast((err as Error).message || 'Error al guardar', 'error');
    } finally {
      setLoading(false);
    }
  }

  const isService = form.type === 'service';

  return (
    <Modal title={product ? 'Editar producto' : 'Nuevo producto'} onClose={onClose}>
      <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: '14px', maxHeight: '70vh', overflowY: 'auto', paddingRight: '4px' }}>

        {/* Nombre + Código */}
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 140px', gap: '10px' }}>
          <div style={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
            <label style={labelStyle}>Nombre *</label>
            <input className="input" value={form.name} onChange={e => set('name', e.target.value)} placeholder="Nombre del producto" autoFocus required />
          </div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
            <label style={labelStyle}>Código</label>
            <input className="input" value={form.code} onChange={e => set('code', e.target.value)} placeholder="Ej: 001" />
          </div>
        </div>

        {/* Barcode */}
        <div style={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
          <label style={labelStyle}>Código de barras</label>
          <input
            className="input"
            value={form.barcode}
            onChange={e => set('barcode', e.target.value)}
            placeholder="Escanea o escribe el código"
          />
        </div>

        {/* Tipo + Categoría */}
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '10px' }}>
          <div style={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
            <label style={labelStyle}>Tipo</label>
            <select className="input" value={form.type} onChange={e => set('type', e.target.value)} style={{ cursor: 'pointer' }}>
              <option value="product">Producto</option>
              <option value="service">Servicio</option>
            </select>
          </div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
            <label style={labelStyle}>Categoría</label>
            <select className="input" value={form.category_id} onChange={e => set('category_id', e.target.value)} style={{ cursor: 'pointer' }}>
              <option value={0}>Sin categoría</option>
              {categories.map(c => <option key={c.id} value={c.id}>{c.name}</option>)}
            </select>
          </div>
        </div>

        {/* Precios */}
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: '10px' }}>
          <div style={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
            <label style={labelStyle}>Precio venta (COP)</label>
            <input className="input tnum" type="text" inputMode="numeric" value={form.sale_price} onChange={e => set('sale_price', formatInputCOP(e.target.value))} style={{ textAlign: 'right' }} placeholder="0" />
          </div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
            <label style={labelStyle}>Precio mayorista (opcional)</label>
            <input className="input tnum" type="text" inputMode="numeric" value={form.wholesale_price} onChange={e => set('wholesale_price', formatInputCOP(e.target.value))} style={{ textAlign: 'right' }} placeholder="Sin mayoreo" />
          </div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
            <label style={labelStyle}>Costo (COP)</label>
            <input className="input tnum" type="text" inputMode="numeric" value={form.cost} onChange={e => set('cost', formatInputCOP(e.target.value))} style={{ textAlign: 'right' }} placeholder="0" />
          </div>
        </div>

        {/* Stock (solo para productos) */}
        {!isService && (
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: '10px' }}>
            <div style={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
              <label style={labelStyle}>Stock actual</label>
              <input className="input tnum" type="number" min="0" value={form.stock} onChange={e => set('stock', e.target.value)} style={{ textAlign: 'right' }} />
            </div>
            <div style={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
              <label style={labelStyle}>Stock mínimo</label>
              <input className="input tnum" type="number" min="0" value={form.min_stock} onChange={e => set('min_stock', e.target.value)} style={{ textAlign: 'right' }} />
            </div>
            <div style={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
              <label style={labelStyle}>Unidad</label>
              <select className="input" value={form.unit} onChange={e => set('unit', e.target.value)} style={{ cursor: 'pointer' }}>
                {UNITS.map(u => <option key={u} value={u}>{UNIT_LABELS[u]}</option>)}
              </select>
            </div>
          </div>
        )}

        {/* IVA + Activo */}
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '10px' }}>
          <div style={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
            <label style={labelStyle}>IVA (%)</label>
            <input className="input tnum" type="number" min="0" max="100" value={form.tax_rate} onChange={e => set('tax_rate', e.target.value)} style={{ textAlign: 'right' }} />
          </div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
            <label style={labelStyle}>Estado</label>
            <select className="input" value={form.active ? 'true' : 'false'} onChange={e => set('active', e.target.value === 'true')} style={{ cursor: 'pointer' }}>
              <option value="true">Activo</option>
              <option value="false">Inactivo</option>
            </select>
          </div>
        </div>

        <div style={{ display: 'flex', gap: '8px', justifyContent: 'flex-end', marginTop: '4px' }}>
          <button type="button" className="btn" onClick={onClose}>Cancelar</button>
          <button type="submit" className="btn btn-accent" disabled={loading || !form.name.trim()}>
            {loading ? 'Guardando...' : product ? 'Guardar cambios' : 'Crear producto'}
          </button>
        </div>
      </form>
    </Modal>
  );
}

const labelStyle: React.CSSProperties = {
  fontSize: '12px', fontWeight: 600, color: 'var(--text-secondary)',
  textTransform: 'uppercase', letterSpacing: '0.4px',
};

// ─── Confirm Dialog ───────────────────────────────────────────────────────────

function ConfirmModal({ message, onConfirm, onClose }: { message: string; onConfirm: () => void; onClose: () => void }) {
  return (
    <Modal title="Confirmar acción" onClose={onClose}>
      <p style={{ color: 'var(--text-secondary)', fontSize: '14px' }}>{message}</p>
      <div style={{ display: 'flex', gap: '8px', justifyContent: 'flex-end' }}>
        <button className="btn" onClick={onClose}>Cancelar</button>
        <button className="btn btn-danger" onClick={() => { onConfirm(); onClose(); }}>Eliminar</button>
      </div>
    </Modal>
  );
}

// ─── Stock Entry Modal ────────────────────────────────────────────────────────

function StockEntryModal({
  products,
  preselectedProduct,
  onClose,
  onSaved,
}: {
  products: Product[];
  preselectedProduct: Product | null;
  onClose: () => void;
  onSaved: () => void;
}) {
  const { toast } = useToast();
  const [productId, setProductId] = useState<number>(preselectedProduct?.id ?? 0);
  const [quantity, setQuantity] = useState('');
  const [note, setNote] = useState('');
  const [loading, setLoading] = useState(false);

  const stockProducts = products.filter(p => p.type === 'product');

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    const qty = parseFloat(quantity);
    if (!productId || !qty || qty <= 0 || qty > 99999) return;
    setLoading(true);
    try {
      await api.post('/stock-entries', { product_id: productId, quantity: qty, note: note.trim() });
      toast('Entrada registrada', 'success');
      onSaved();
      onClose();
    } catch (err: unknown) {
      toast((err as Error).message || 'Error al registrar entrada', 'error');
    } finally {
      setLoading(false);
    }
  }

  return (
    <Modal title="Entrada de inventario" onClose={onClose}>
      <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: '14px' }}>
        <div style={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
          <label style={labelStyle}>Producto *</label>
          <select
            className="input"
            value={productId}
            onChange={e => setProductId(Number(e.target.value))}
            required
            autoFocus={!preselectedProduct}
            style={{ cursor: 'pointer' }}
          >
            <option value={0}>Seleccionar producto...</option>
            {stockProducts.map(p => (
              <option key={p.id} value={p.id}>
                {p.name}{p.code ? ` (${p.code})` : ''} — Stock: {p.stock}
              </option>
            ))}
          </select>
        </div>
        <div style={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
          <label style={labelStyle}>Cantidad *</label>
          <input
            className="input tnum"
            type="number"
            min="0.01"
            max="99999"
            step="0.01"
            value={quantity}
            onChange={e => setQuantity(e.target.value)}
            placeholder="Ej: 10"
            style={{ textAlign: 'right' }}
            autoFocus={!!preselectedProduct}
            required
          />
        </div>
        <div style={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
          <label style={labelStyle}>Nota</label>
          <input
            className="input"
            value={note}
            onChange={e => setNote(e.target.value)}
            placeholder="Opcional — Ej: Compra proveedor"
          />
        </div>
        <div style={{ display: 'flex', gap: '8px', justifyContent: 'flex-end', marginTop: '4px' }}>
          <button type="button" className="btn" onClick={onClose}>Cancelar</button>
          <button
            type="submit"
            className="btn btn-accent"
            disabled={loading || !productId || !quantity || parseFloat(quantity) <= 0 || parseFloat(quantity) > 99999}
          >
            {loading ? 'Registrando...' : 'Registrar entrada'}
          </button>
        </div>
      </form>
    </Modal>
  );
}

// ─── Main Page ────────────────────────────────────────────────────────────────

type Tab = 'products' | 'categories';

export default function InventoryPage() {
  const { toast } = useToast();
  const isMobile = useIsMobile();

  // Data
  const [categories, setCategories] = useState<Category[]>([]);
  const [products, setProducts] = useState<Product[]>([]);
  const [allActive, setAllActive] = useState<Product[]>([]);
  const [loading, setLoading] = useState(true);

  // UI state
  const [tab, setTab] = useState<Tab>('products');
  const [search, setSearch] = useState('');
  const [filterCat, setFilterCat] = useState<number>(0);
  const [filterActive, setFilterActive] = useState<'active' | 'inactive' | 'all'>('active');

  // Modals
  const [catModal, setCatModal] = useState<{ open: boolean; cat: Category | null }>({ open: false, cat: null });
  const [prodModal, setProdModal] = useState<{ open: boolean; product: Product | null }>({ open: false, product: null });
  const [stockEntryModal, setStockEntryModal] = useState<{ open: boolean; product: Product | null }>({ open: false, product: null });
  const [confirm, setConfirm] = useState<{ message: string; onConfirm: () => void } | null>(null);

  // Debounce ref
  const searchTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const [debouncedSearch, setDebouncedSearch] = useState('');

  useEffect(() => {
    if (searchTimer.current) clearTimeout(searchTimer.current);
    searchTimer.current = setTimeout(() => setDebouncedSearch(search), 200);
    return () => { if (searchTimer.current) clearTimeout(searchTimer.current); };
  }, [search]);

  const loadAllActive = useCallback(async () => {
    try {
      const data = await api.get<Product[]>('/products?active=true');
      setAllActive(data ?? []);
    } catch { /* silent */ }
  }, []);

  const loadCategories = useCallback(async () => {
    try {
      const data = await api.get<Category[]>('/categories');
      setCategories(data ?? []);
    } catch {
      toast('Error al cargar categorías', 'error');
    }
  }, [toast]);

  const loadProducts = useCallback(async () => {
    setLoading(true);
    try {
      const params = new URLSearchParams();
      if (debouncedSearch) params.set('search', debouncedSearch);
      if (filterCat) params.set('category_id', String(filterCat));
      // Backend only supports active=true filter; 'inactive' and 'all' fetch everything and filter client-side
      if (filterActive === 'active') params.set('active', 'true');
      const data = await api.get<Product[]>(`/products?${params}`);
      const filtered = filterActive === 'inactive'
        ? (data ?? []).filter(p => !p.active)
        : (data ?? []);
      setProducts(filtered);
    } catch {
      toast('Error al cargar productos', 'error');
    } finally {
      setLoading(false);
    }
  }, [debouncedSearch, filterCat, filterActive, toast]);

  useEffect(() => { loadCategories(); }, [loadCategories]);
  useEffect(() => { loadProducts(); }, [loadProducts]);
  useEffect(() => { loadAllActive(); }, [loadAllActive]);

  // Stats
  const totalActive = allActive.length;
  const lowStock = allActive.filter(p => p.type === 'product' && p.stock > 0 && p.stock <= p.min_stock).length;
  const noStock = allActive.filter(p => p.type === 'product' && p.stock <= 0).length;

  // Delete handlers
  async function deleteCategory(id: number) {
    try {
      await api.delete(`/categories/${id}`);
      toast('Categoría eliminada', 'success');
      loadCategories();
    } catch (err: unknown) {
      toast((err as Error).message || 'No se puede eliminar', 'error');
    }
  }

  async function deleteProduct(id: number) {
    try {
      await api.delete(`/products/${id}`);
      toast('Producto eliminado', 'success');
      loadProducts();
      loadAllActive();
    } catch (err: unknown) {
      toast((err as Error).message || 'No se puede eliminar', 'error');
    }
  }

  const catMap = Object.fromEntries(categories.map(c => [c.id, c.name]));

  return (
    <AppLayout>
      <Topbar
        title="Inventario"
        subtitle={`${totalActive} productos activos${lowStock > 0 ? ` · ${lowStock} bajo mínimo` : ''}${noStock > 0 ? ` · ${noStock} sin stock` : ''}`}
        actions={
          tab === 'products'
            ? <div style={{ display: 'flex', gap: '6px' }}>
                <button className="btn btn-sm" onClick={() => setStockEntryModal({ open: true, product: null })}><IconBoxIn /> Entrada de inventario</button>
                <button className="btn btn-accent btn-sm" onClick={() => setProdModal({ open: true, product: null })}><IconPlus /> Nuevo producto</button>
              </div>
            : <button className="btn btn-accent btn-sm" onClick={() => setCatModal({ open: true, cat: null })}><IconPlus /> Nueva categoría</button>
        }
      />

      <div style={{ flex: 1, overflow: 'auto', padding: isMobile ? '12px 8px' : '20px 24px', display: 'flex', flexDirection: 'column', gap: '16px' }}>

        {/* Stats */}
        <div style={{ display: 'grid', gridTemplateColumns: isMobile ? 'repeat(3, 1fr)' : 'repeat(3, 1fr)', gap: isMobile ? '8px' : '12px' }}>
          <StatCard label="Total activos" value={String(totalActive)} />
          <StatCard label="Bajo stock mínimo" value={String(lowStock)} valueColor={lowStock > 0 ? 'var(--warning)' : undefined} />
          <StatCard label="Sin stock" value={String(noStock)} valueColor={noStock > 0 ? 'var(--danger)' : undefined} />
        </div>

        {/* Tabs */}
        <div style={{ display: 'flex', gap: '4px', borderBottom: '1px solid var(--border)', paddingBottom: '0' }}>
          {(['products', 'categories'] as Tab[]).map(t => (
            <button
              key={t}
              onClick={() => setTab(t)}
              style={{
                padding: '8px 16px',
                fontSize: '13px',
                fontWeight: tab === t ? 600 : 400,
                color: tab === t ? 'var(--accent)' : 'var(--text-secondary)',
                background: 'transparent',
                border: 'none',
                borderBottom: tab === t ? '2px solid var(--accent)' : '2px solid transparent',
                marginBottom: '-1px',
                cursor: 'pointer',
                transition: 'color 150ms ease',
              }}
            >
              {t === 'products' ? 'Productos' : 'Categorías'}
            </button>
          ))}
        </div>

        {/* Products Tab */}
        {tab === 'products' && (
          <>
            {/* Filters */}
            <div style={{ display: 'flex', gap: '8px', alignItems: 'center', flexWrap: 'wrap' }}>
              <div className="input-wrap" style={{ flex: '1', minWidth: '200px', maxWidth: '360px' }}>
                <span className="input-icon"><IconSearch /></span>
                <input
                  className="input"
                  placeholder="Buscar por nombre o código..."
                  value={search}
                  onChange={e => setSearch(e.target.value)}
                />
              </div>
              <select
                className="input"
                value={filterActive}
                onChange={e => setFilterActive(e.target.value as 'active' | 'inactive' | 'all')}
                style={{ width: 'auto', cursor: 'pointer', fontSize: '12px', padding: '5px 10px' }}
              >
                <option value="active">Solo activos</option>
                <option value="inactive">Solo inactivos</option>
                <option value="all">Todos</option>
              </select>
              <div style={{ display: 'flex', gap: '4px', flexWrap: 'wrap' }}>
                <button
                  className={`cat-tab${filterCat === 0 ? ' active' : ''}`}
                  onClick={() => setFilterCat(0)}
                >
                  Todos
                </button>
                {categories.map(c => (
                  <button
                    key={c.id}
                    className={`cat-tab${filterCat === c.id ? ' active' : ''}`}
                    onClick={() => setFilterCat(c.id)}
                  >
                    {c.name}
                  </button>
                ))}
              </div>
            </div>

            {/* Table */}
            <div className="table-wrap">
              <table style={{ width: '100%', borderCollapse: 'collapse' }}>
                <thead>
                  <tr style={{ borderBottom: '1px solid var(--border)' }}>
                    {(isMobile
                      ? ['Producto', 'Precio', 'Stock', 'Acciones']
                      : ['Producto', 'Categoría', 'Precio', 'Stock', 'Mín.', 'Estado', 'Activo', 'Acciones']
                    ).map(h => (
                      <th key={h} style={{
                        ...thStyle,
                        textAlign: ['Precio', 'Stock', 'Mín.'].includes(h) ? 'right' : (h === 'Acciones' ? 'right' : 'left'),
                      }}>{h}</th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {loading ? (
                    <tr><td colSpan={isMobile ? 4 : 8} style={emptyStyle}>Cargando...</td></tr>
                  ) : products.length === 0 ? (
                    <tr><td colSpan={isMobile ? 4 : 8} style={emptyStyle}>
                      <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: '8px', color: 'var(--text-muted)' }}>
                        <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5"><path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/></svg>
                        <span style={{ fontSize: '13px' }}>No hay productos</span>
                      </div>
                    </td></tr>
                  ) : products.map(p => (
                    <tr key={p.id} style={{ borderBottom: '1px solid var(--border)', background: p.type === 'product' && p.stock <= p.min_stock && p.active ? 'rgba(224,84,84,0.04)' : undefined }}>
                      <td style={tdStyle}>
                        <div style={{ fontWeight: 600, fontSize: '13px', color: 'var(--text-primary)' }}>{p.name}</div>
                        {p.code && <div style={{ fontSize: '11px', color: 'var(--text-muted)' }}>{p.code}</div>}
                        {p.type === 'service' && <div style={{ fontSize: '10px', color: 'var(--text-muted)', marginTop: '2px' }}>Servicio</div>}
                      </td>
                      {!isMobile && (
                        <td style={tdStyle}>
                          <span style={{ fontSize: '12px', color: 'var(--text-secondary)' }}>
                            {catMap[p.category_id] ?? '—'}
                          </span>
                        </td>
                      )}
                      <td style={{ ...tdStyle, textAlign: 'right' }}>
                        <span className="tnum" style={{ fontWeight: 600, fontSize: '13px' }}>{formatCOP(p.sale_price)}</span>
                      </td>
                      <td style={{ ...tdStyle, textAlign: 'right' }}>
                        {p.type === 'product'
                          ? <span className="tnum" style={{ fontSize: '13px' }}>{p.stock}</span>
                          : <span style={{ color: 'var(--text-muted)', fontSize: '12px' }}>—</span>}
                      </td>
                      {!isMobile && (
                        <td style={{ ...tdStyle, textAlign: 'right' }}>
                          {p.type === 'product'
                            ? <span className="tnum" style={{ fontSize: '12px', color: 'var(--text-muted)' }}>{p.min_stock}</span>
                            : <span style={{ color: 'var(--text-muted)', fontSize: '12px' }}>—</span>}
                        </td>
                      )}
                      {!isMobile && (
                        <td style={tdStyle}>
                          {p.type === 'product'
                            ? <span className={`stock-badge ${stockClass(p.stock, p.min_stock)}`}>{stockLabel(p.stock, p.min_stock)}</span>
                            : <span className="badge" style={{ background: 'var(--surface-3)', color: 'var(--text-muted)' }}>Servicio</span>}
                        </td>
                      )}
                      {!isMobile && (
                        <td style={tdStyle}>
                          <span className={`badge ${p.active ? 'badge-ok' : ''}`} style={!p.active ? { background: 'var(--surface-3)', color: 'var(--text-muted)' } : undefined}>
                            {p.active ? 'Activo' : 'Inactivo'}
                          </span>
                        </td>
                      )}
                      <td style={tdStyle}>
                        <div style={{ display: 'flex', gap: '4px', justifyContent: 'flex-end' }}>
                          {p.type === 'product' && (
                            <button
                              className="btn btn-sm"
                              title="Entrada de inventario"
                              onClick={() => setStockEntryModal({ open: true, product: p })}
                            ><IconBoxIn /></button>
                          )}
                          <button
                            className="btn btn-sm"
                            title="Editar"
                            onClick={() => setProdModal({ open: true, product: p })}
                          ><IconEdit /></button>
                          {!isMobile && (
                            <button
                              className="btn btn-sm btn-danger"
                              title="Eliminar"
                              onClick={() => setConfirm({
                                message: `¿Eliminar "${p.name}"? Esta acción no se puede deshacer.`,
                                onConfirm: () => deleteProduct(p.id),
                              })}
                            ><IconTrash /></button>
                          )}
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </>
        )}

        {/* Categories Tab */}
        {tab === 'categories' && (
          <div className="table-wrap">
            <table style={{ width: '100%', borderCollapse: 'collapse' }}>
              <thead>
                <tr style={{ borderBottom: '1px solid var(--border)' }}>
                  {['Nombre', 'Descripción', 'Acciones'].map(h => (
                    <th key={h} style={thStyle}>{h}</th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {categories.length === 0 ? (
                  <tr><td colSpan={3} style={emptyStyle}>
                    <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: '8px', color: 'var(--text-muted)' }}>
                      <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5"><rect x="3" y="3" width="7" height="7"/><rect x="14" y="3" width="7" height="7"/><rect x="14" y="14" width="7" height="7"/><rect x="3" y="14" width="7" height="7"/></svg>
                      <span style={{ fontSize: '13px' }}>No hay categorías</span>
                    </div>
                  </td></tr>
                ) : categories.map(c => (
                  <tr key={c.id} style={{ borderBottom: '1px solid var(--border)' }}>
                    <td style={tdStyle}>
                      <span style={{ fontWeight: 600, fontSize: '13px', color: 'var(--text-primary)' }}>{c.name}</span>
                    </td>
                    <td style={tdStyle}>
                      <span style={{ fontSize: '12px', color: 'var(--text-secondary)' }}>{c.description || '—'}</span>
                    </td>
                    <td style={tdStyle}>
                      <div style={{ display: 'flex', gap: '4px' }}>
                        <button
                          className="btn btn-sm"
                          title="Editar"
                          onClick={() => setCatModal({ open: true, cat: c })}
                        ><IconEdit /></button>
                        <button
                          className="btn btn-sm btn-danger"
                          title="Eliminar"
                          onClick={() => setConfirm({
                            message: `¿Eliminar la categoría "${c.name}"? Solo es posible si no tiene productos asignados.`,
                            onConfirm: () => deleteCategory(c.id),
                          })}
                        ><IconTrash /></button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Modals */}
      {catModal.open && (
        <CategoryModal
          cat={catModal.cat}
          onClose={() => setCatModal({ open: false, cat: null })}
          onSaved={loadCategories}
        />
      )}
      {prodModal.open && (
        <ProductModal
          product={prodModal.product}
          categories={categories}
          onClose={() => setProdModal({ open: false, product: null })}
          onSaved={() => { loadProducts(); loadAllActive(); }}
        />
      )}
      {stockEntryModal.open && (
        <StockEntryModal
          products={allActive}
          preselectedProduct={stockEntryModal.product}
          onClose={() => setStockEntryModal({ open: false, product: null })}
          onSaved={() => { loadProducts(); loadAllActive(); }}
        />
      )}
      {confirm && (
        <ConfirmModal
          message={confirm.message}
          onConfirm={confirm.onConfirm}
          onClose={() => setConfirm(null)}
        />
      )}
    </AppLayout>
  );
}

// ─── Small helpers ────────────────────────────────────────────────────────────

function StatCard({ label, value, valueColor }: { label: string; value: string; valueColor?: string }) {
  return (
    <div className="stat-card">
      <div className="stat-label">{label}</div>
      <div className="stat-value" style={valueColor ? { color: valueColor } : undefined}>{value}</div>
    </div>
  );
}

const thStyle: React.CSSProperties = {
  padding: '10px 14px',
  textAlign: 'left',
  fontSize: '11px',
  fontWeight: 600,
  textTransform: 'uppercase',
  letterSpacing: '0.4px',
  color: 'var(--text-muted)',
  whiteSpace: 'nowrap',
};

const tdStyle: React.CSSProperties = {
  padding: '10px 14px',
  verticalAlign: 'middle',
};

const emptyStyle: React.CSSProperties = {
  padding: '40px',
  textAlign: 'center',
  color: 'var(--text-muted)',
};
