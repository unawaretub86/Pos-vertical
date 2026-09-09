'use client';

import { useState, useEffect, useCallback, useRef } from 'react';
import AppLayout from '@/components/AppLayout';
import Topbar from '@/components/Topbar';
import { api } from '@/lib/api';
import { useToast } from '@/components/Toast';
import TicketPreview, { type TicketData } from '@/components/TicketPreview';
import { formatCOP, formatInputCOP, parseInputCOP } from '@/lib/format';
import { getUser } from '@/lib/auth';

// ─── Types ───────────────────────────────────────────────────────────────────

interface Category {
  id: number;
  name: string;
}

interface Customer {
  id: number;
  name: string;
  phone: string;
  notes: string;
  credit_limit: number | null;
}

interface Product {
  id: number;
  name: string;
  code: string;
  barcode: string;
  category_id: number | null;
  type: 'product' | 'service';
  sale_price: number;
  wholesale_price: number | null;
  stock: number;
  min_stock: number;
  unit: string;
  active: boolean;
}

interface CartItem {
  product: Product;
  quantity: number;
  unitPrice: number;
  priceInput: string;
  priceMode: 'retail' | 'wholesale';
  discount: number;
  discountType: 'fixed' | 'pct';
  discountInput: string;
}

type PaymentMethod = 'efectivo' | 'transferencia' | 'tarjeta' | 'credito';

// ─── Helpers ─────────────────────────────────────────────────────────────────

// ─── Icons ───────────────────────────────────────────────────────────────────

const IconSearch = () => (
  <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
    <circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/>
  </svg>
);

const IconTrash = () => (
  <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
    <polyline points="3 6 5 6 21 6"/><path d="M19 6l-1 14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2L5 6"/>
    <path d="M10 11v6"/><path d="M14 11v6"/><path d="M9 6V4h6v2"/>
  </svg>
);

const IconCart = () => (
  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
    <circle cx="9" cy="21" r="1"/><circle cx="20" cy="21" r="1"/>
    <path d="M1 1h4l2.68 13.39a2 2 0 0 0 2 1.61h9.72a2 2 0 0 0 2-1.61L23 6H6"/>
  </svg>
);

// ─── Stock badge ─────────────────────────────────────────────────────────────

function StockBadge({ product }: { product: Product }) {
  if (product.type === 'service') return null;
  const { stock, min_stock } = product;
  let bg = 'rgba(76,175,130,0.15)';
  let color = 'var(--success)';
  let label = String(stock);
  if (stock <= 0) { bg = 'rgba(224,84,84,0.15)'; color = 'var(--danger)'; }
  else if (stock <= 3) { bg = 'rgba(224,84,84,0.15)'; color = 'var(--danger)'; }
  else if (stock <= min_stock) { bg = 'rgba(245,166,35,0.15)'; color = 'var(--accent)'; }

  return (
    <span style={{
      fontSize: '10px', padding: '2px 6px', borderRadius: '4px',
      background: bg, color, fontVariantNumeric: 'tabular-nums',
    }}>
      {label}
    </span>
  );
}

// ─── Product Card ─────────────────────────────────────────────────────────────

function ProductCard({ product, onAdd }: { product: Product; onAdd: (p: Product) => void }) {
  const [hovered, setHovered] = useState(false);
  const outOfStock = product.type === 'product' && product.stock <= 0;

  return (
    <button
      disabled={outOfStock}
      onClick={() => onAdd(product)}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      style={{
        background: hovered && !outOfStock ? 'var(--surface-2)' : 'var(--surface-1)',
        border: `1px solid ${hovered && !outOfStock ? 'var(--border-emphasis)' : 'var(--border)'}`,
        borderRadius: '8px',
        padding: '12px',
        cursor: outOfStock ? 'not-allowed' : 'pointer',
        textAlign: 'left',
        display: 'flex',
        flexDirection: 'column',
        gap: '6px',
        opacity: outOfStock ? 0.5 : 1,
        transition: 'border-color 120ms ease, background 120ms ease',
      }}
    >
      <div style={{
        fontSize: '13px', fontWeight: 600,
        color: 'var(--text-primary)',
        overflow: 'hidden', textOverflow: 'ellipsis',
        whiteSpace: 'nowrap',
      }}>
        {product.name}
      </div>
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <span style={{
          fontSize: '17px', fontWeight: 700,
          color: 'var(--accent)',
          fontVariantNumeric: 'tabular-nums',
        }}>
          {formatCOP(product.sale_price)}
        </span>
        <StockBadge product={product} />
      </div>
    </button>
  );
}

// ─── Cart Item Row ────────────────────────────────────────────────────────────

function CartItemRow({
  item,
  isAdmin,
  onQtyChange,
  onDiscountChange,
  onPriceChange,
  onPriceModeChange,
  onRemove,
}: {
  item: CartItem;
  isAdmin: boolean;
  onQtyChange: (delta: number) => void;
  onDiscountChange: (discount: number, discountType: 'fixed' | 'pct', discountInput: string) => void;
  onPriceChange: (price: number, priceInput: string) => void;
  onPriceModeChange: (mode: 'retail' | 'wholesale') => void;
  onRemove: () => void;
}) {
  const [hovered, setHovered] = useState(false);
  const [trashHover, setTrashHover] = useState(false);
  const [showDiscount, setShowDiscount] = useState(item.discount > 0);
  const lineTotal = item.unitPrice * item.quantity - item.discount;
  const hasWholesale = (item.product.wholesale_price ?? 0) > 0;
  const expectedPrice = item.priceMode === 'wholesale' ? (item.product.wholesale_price ?? item.product.sale_price) : item.product.sale_price;
  const isPriceOverridden = item.unitPrice !== expectedPrice;

  function handleDiscountInputChange(raw: string) {
    const num = parseFloat(raw);
    if (raw === '' || raw === '-') {
      onDiscountChange(0, item.discountType, '');
      return;
    }
    if (isNaN(num) || num < 0) return;
    let discountAmount: number;
    if (item.discountType === 'pct') {
      const pct = Math.min(num, 100);
      discountAmount = (item.unitPrice * item.quantity * pct) / 100;
      onDiscountChange(discountAmount, 'pct', String(pct));
    } else {
      const max = item.unitPrice * item.quantity;
      discountAmount = Math.min(num, max);
      onDiscountChange(discountAmount, 'fixed', String(discountAmount));
    }
  }

  function toggleDiscountType() {
    const newType = item.discountType === 'fixed' ? 'pct' : 'fixed';
    onDiscountChange(0, newType, '');
  }

  return (
    <div
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      style={{
        padding: '8px 16px',
        background: hovered ? 'var(--surface-2)' : 'transparent',
        display: 'flex',
        flexDirection: 'column',
        gap: '4px',
        transition: 'background 100ms ease',
      }}
    >
      {/* Name + remove */}
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <span style={{ fontSize: '13px', fontWeight: 600, color: 'var(--text-primary)', flex: 1, minWidth: 0, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
          {item.product.name}
        </span>
        <button
          onClick={onRemove}
          onMouseEnter={() => setTrashHover(true)}
          onMouseLeave={() => setTrashHover(false)}
          style={{
            width: '20px', height: '20px',
            border: 'none', background: 'transparent', cursor: 'pointer',
            display: 'flex', alignItems: 'center', justifyContent: 'center',
            color: trashHover ? 'var(--danger)' : 'var(--text-muted)',
            borderRadius: '4px',
            flexShrink: 0,
            transition: 'color 100ms ease',
          }}
        >
          <IconTrash />
        </button>
      </div>

      {/* Wholesale toggle */}
      {hasWholesale && (
        <div style={{ display: 'flex', gap: '4px' }}>
          <button
            onClick={() => onPriceModeChange('retail')}
            style={{
              fontSize: '10px', padding: '1px 7px',
              border: '1px solid var(--border)', borderRadius: '4px',
              background: item.priceMode === 'retail' ? 'var(--accent)' : 'var(--surface-2)',
              color: item.priceMode === 'retail' ? '#000' : 'var(--text-muted)',
              cursor: 'pointer', fontWeight: item.priceMode === 'retail' ? 600 : 400,
            }}
          >
            Menudeo: {formatCOP(item.product.sale_price)}
          </button>
          <button
            onClick={() => onPriceModeChange('wholesale')}
            style={{
              fontSize: '10px', padding: '1px 7px',
              border: '1px solid var(--border)', borderRadius: '4px',
              background: item.priceMode === 'wholesale' ? 'var(--accent)' : 'var(--surface-2)',
              color: item.priceMode === 'wholesale' ? '#000' : 'var(--text-muted)',
              cursor: 'pointer', fontWeight: item.priceMode === 'wholesale' ? 600 : 400,
            }}
          >
            Mayoreo: {formatCOP(item.product.wholesale_price!)}
          </button>
        </div>
      )}

      {/* Controls + price */}
      <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
        {/* Qty controls */}
        <div style={{ display: 'flex', alignItems: 'center', gap: '4px' }}>
          <button
            onClick={() => onQtyChange(-1)}
            style={{
              width: '24px', height: '24px', border: '1px solid var(--border)',
              background: 'var(--surface-2)', borderRadius: '4px',
              cursor: 'pointer', fontSize: '14px', color: 'var(--text-primary)',
              display: 'flex', alignItems: 'center', justifyContent: 'center',
            }}
          >−</button>
          <input
            type="number"
            min="1"
            value={item.quantity}
            onChange={e => {
              const v = parseFloat(e.target.value);
              if (!isNaN(v) && v > 0) onQtyChange(v - item.quantity);
            }}
            style={{
              width: '44px', textAlign: 'center', padding: '2px 4px',
              border: '1px solid var(--border)', borderRadius: '4px',
              background: 'var(--surface-1)', color: 'var(--text-primary)',
              fontSize: '13px', fontVariantNumeric: 'tabular-nums',
            }}
          />
          <button
            onClick={() => onQtyChange(1)}
            style={{
              width: '24px', height: '24px', border: '1px solid var(--border)',
              background: 'var(--surface-2)', borderRadius: '4px',
              cursor: 'pointer', fontSize: '14px', color: 'var(--text-primary)',
              display: 'flex', alignItems: 'center', justifyContent: 'center',
            }}
          >+</button>
        </div>

        {/* Unit price */}
        {isAdmin ? (
          <input
            type="number"
            min="0"
            value={item.priceInput}
            onChange={e => {
              const raw = e.target.value;
              const num = parseFloat(raw);
              if (raw === '') { onPriceChange(item.product.sale_price, ''); return; }
              if (!isNaN(num) && num >= 0) onPriceChange(num, raw);
            }}
            onBlur={e => {
              const num = parseFloat(e.target.value);
              if (isNaN(num) || num < 0) onPriceChange(item.product.sale_price, String(item.product.sale_price));
            }}
            title="Modificar precio (solo admin)"
            style={{
              width: '80px', padding: '2px 4px', border: `1px solid ${isPriceOverridden ? 'var(--accent)' : 'var(--border)'}`,
              borderRadius: '4px', background: isPriceOverridden ? 'rgba(245,166,35,0.08)' : 'var(--surface-1)',
              color: isPriceOverridden ? 'var(--accent)' : 'var(--text-muted)',
              fontSize: '12px', fontVariantNumeric: 'tabular-nums', textAlign: 'right',
            }}
          />
        ) : (
          <span style={{ fontSize: '12px', color: 'var(--text-muted)', fontVariantNumeric: 'tabular-nums' }}>
            {formatCOP(item.unitPrice)}
          </span>
        )}

        {/* Line total */}
        <span style={{ marginLeft: 'auto', fontSize: '13px', fontWeight: 600, fontVariantNumeric: 'tabular-nums', textAlign: 'right' }}>
          {formatCOP(lineTotal)}
        </span>
      </div>

      {/* Discount row */}
      <div style={{ display: 'flex', alignItems: 'center', gap: '6px', fontSize: '12px' }}>
        <button
          onClick={() => { setShowDiscount(v => !v); if (showDiscount) onDiscountChange(0, item.discountType, ''); }}
          style={{
            fontSize: '11px', padding: '1px 6px', border: '1px solid var(--border)',
            borderRadius: '4px', background: showDiscount ? 'var(--accent)' : 'var(--surface-2)',
            color: showDiscount ? '#000' : 'var(--text-muted)', cursor: 'pointer',
          }}
        >
          Descuento
        </button>
        {showDiscount && (
          <>
            <button
              onClick={toggleDiscountType}
              style={{
                width: '28px', height: '22px', border: '1px solid var(--border)',
                borderRadius: '4px', background: 'var(--surface-2)', cursor: 'pointer',
                fontSize: '11px', color: 'var(--text-primary)', fontWeight: 600,
              }}
            >
              {item.discountType === 'pct' ? '%' : '$'}
            </button>
            <input
              type="number"
              min="0"
              max={item.discountType === 'pct' ? 100 : undefined}
              value={item.discountInput}
              onChange={e => handleDiscountInputChange(e.target.value)}
              placeholder={item.discountType === 'pct' ? '0' : '0'}
              style={{
                width: '60px', padding: '1px 4px', border: '1px solid var(--border)',
                borderRadius: '4px', background: 'var(--surface-1)', color: 'var(--text-primary)',
                fontSize: '12px', textAlign: 'right',
              }}
            />
            {item.discount > 0 && (
              <span style={{ color: 'var(--danger)', fontSize: '11px', fontVariantNumeric: 'tabular-nums' }}>
                −{formatCOP(item.discount)}
              </span>
            )}
          </>
        )}
      </div>
    </div>
  );
}

// ─── Customer Picker ──────────────────────────────────────────────────────────

function CustomerPicker({
  selected,
  onSelect,
}: {
  selected: Customer | null;
  onSelect: (c: Customer | null) => void;
}) {
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState('');
  const [results, setResults] = useState<Customer[]>([]);
  const [loading, setLoading] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const search = useCallback(async (q: string) => {
    setLoading(true);
    try {
      const res = await api.get<{ customers: Customer[] | null }>(
        `/customers${q ? `?q=${encodeURIComponent(q)}` : ''}`
      );
      setResults(res.customers ?? []);
    } catch {
      setResults([]);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    if (open) {
      search('');
      setTimeout(() => inputRef.current?.focus(), 50);
    }
  }, [open, search]);

  function handleQuery(val: string) {
    setQuery(val);
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => search(val), 250);
  }

  function pick(c: Customer) {
    onSelect(c);
    setOpen(false);
    setQuery('');
  }

  function clear(e: React.MouseEvent) {
    e.stopPropagation();
    onSelect(null);
  }

  return (
    <div style={{ position: 'relative' }}>
      {/* Trigger */}
      <button
        type="button"
        onClick={() => setOpen(o => !o)}
        style={{
          width: '100%',
          display: 'flex', alignItems: 'center', justifyContent: 'space-between',
          padding: '8px 10px',
          border: `1px solid ${selected ? 'var(--accent)' : 'var(--border)'}`,
          borderRadius: '6px',
          background: selected ? 'var(--accent-dim)' : 'var(--surface-1)',
          color: selected ? 'var(--accent)' : 'var(--text-muted)',
          cursor: 'pointer', textAlign: 'left', gap: '8px',
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
          <span onClick={clear} title="Quitar cliente" style={{ flexShrink: 0, fontSize: '16px', lineHeight: 1, opacity: 0.7 }}>×</span>
        ) : (
          <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round" style={{ flexShrink: 0 }}>
            <polyline points="6 9 12 15 18 9"/>
          </svg>
        )}
      </button>

      {/* Dropdown */}
      {open && (
        <div style={{
          position: 'absolute', bottom: 'calc(100% + 4px)', left: 0, right: 0,
          background: 'var(--surface-2)',
          border: '1px solid var(--border-emphasis)',
          borderRadius: '8px',
          zIndex: 100,
          boxShadow: '0 8px 24px rgba(0,0,0,0.3)',
          overflow: 'hidden',
        }}>
          <div style={{ padding: '8px' }}>
            <input
              ref={inputRef}
              value={query}
              onChange={e => handleQuery(e.target.value)}
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
                  onClick={() => pick(c)}
                  style={{
                    width: '100%', textAlign: 'left', padding: '8px 12px',
                    background: 'transparent', border: 'none', cursor: 'pointer',
                    display: 'flex', flexDirection: 'column', gap: '1px',
                    borderBottom: '1px solid var(--border)',
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

// ─── Payment Method Button ────────────────────────────────────────────────────

function PayBtn({ label, value, active, onClick }: { label: string; value: PaymentMethod; active: boolean; onClick: () => void }) {
  return (
    <button
      onClick={onClick}
      style={{
        padding: '8px 4px',
        border: `1px solid ${active ? 'var(--accent)' : 'var(--border)'}`,
        borderRadius: '6px',
        background: active ? 'rgba(245,166,35,0.12)' : 'var(--surface-1)',
        color: active ? 'var(--accent)' : 'var(--text-secondary)',
        fontSize: '12px',
        fontWeight: active ? 600 : 400,
        cursor: 'pointer',
        transition: 'all 120ms ease',
      }}
    >
      {label}
    </button>
  );
}

// ─── Main Page ────────────────────────────────────────────────────────────────

export default function POSPage() {
  const { toast } = useToast();
  const isAdmin = getUser()?.role === 'admin';

  // Mobile detection
  const [isMobile, setIsMobile] = useState(false);
  const [mobileTab, setMobileTab] = useState<'products' | 'cart'>('products');

  useEffect(() => {
    const mq = window.matchMedia('(max-width: 767px)');
    setIsMobile(mq.matches);
    const handler = (e: MediaQueryListEvent) => setIsMobile(e.matches);
    mq.addEventListener('change', handler);
    return () => mq.removeEventListener('change', handler);
  }, []);

  // Data
  const [categories, setCategories] = useState<Category[]>([]);
  const [products, setProducts] = useState<Product[]>([]);
  const [loading, setLoading] = useState(false);

  // Filters
  const [search, setSearch] = useState('');
  const [debouncedSearch, setDebouncedSearch] = useState('');
  const [selectedCat, setSelectedCat] = useState<number>(0);
  const searchTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Config
  const [allowGlobalDiscount, setAllowGlobalDiscount] = useState(false);

  // Cart
  const [cart, setCart] = useState<CartItem[]>([]);
  const [globalDiscountInput, setGlobalDiscountInput] = useState<string>('');

  // Payment
  const [payMethod, setPayMethod] = useState<PaymentMethod>('efectivo');
  const [cashReceived, setCashReceived] = useState<string>('');

  // Customer (required for credit)
  const [selectedCustomer, setSelectedCustomer] = useState<Customer | null>(null);
  const [customerBalance, setCustomerBalance] = useState<number | null>(null);

  // Sale loading
  const [cobrarLoading, setCobrarLoading] = useState(false);

  // Post-sale ticket modal
  const [ticketData, setTicketData] = useState<TicketData | null>(null);

  // Debounce search
  useEffect(() => {
    if (searchTimer.current) clearTimeout(searchTimer.current);
    searchTimer.current = setTimeout(() => setDebouncedSearch(search), 200);
    return () => { if (searchTimer.current) clearTimeout(searchTimer.current); };
  }, [search]);

  // Fetch customer balance when selected (for credit limit warning)
  useEffect(() => {
    if (!selectedCustomer) { setCustomerBalance(null); return; }
    api.get<{ balance: number }>(`/customers/${selectedCustomer.id}/balance`)
      .then(r => setCustomerBalance(r.balance))
      .catch(() => setCustomerBalance(null));
  }, [selectedCustomer]);

  // Load config
  useEffect(() => {
    api.get<{ allow_global_discount: boolean }>('/setup')
      .then(cfg => setAllowGlobalDiscount(cfg.allow_global_discount ?? false))
      .catch(() => {});
  }, []);

  // Load categories
  useEffect(() => {
    api.get<Category[]>('/categories')
      .then(data => setCategories(data ?? []))
      .catch(() => {});
  }, []);

  // Load products
  const loadProducts = useCallback(async () => {
    setLoading(true);
    try {
      const params = new URLSearchParams({ active: 'true' });
      if (debouncedSearch) params.set('search', debouncedSearch);
      if (selectedCat) params.set('category_id', String(selectedCat));
      const data = await api.get<Product[]>(`/products?${params}`);
      setProducts(data ?? []);
    } catch {
      toast('Error al cargar productos', 'error');
    } finally {
      setLoading(false);
    }
  }, [debouncedSearch, selectedCat, toast]);

  useEffect(() => { loadProducts(); }, [loadProducts]);

  // Cart operations
  function addToCart(product: Product) {
    setCart(prev => {
      const idx = prev.findIndex(i => i.product.id === product.id);
      if (idx >= 0) {
        const updated = [...prev];
        updated[idx] = { ...updated[idx], quantity: updated[idx].quantity + 1 };
        return updated;
      }
      return [...prev, { product, quantity: 1, unitPrice: product.sale_price, priceInput: String(product.sale_price), priceMode: 'retail' as const, discount: 0, discountType: 'fixed', discountInput: '' }];
    });
    // On mobile, briefly flash cart tab badge — stay on products so user can keep adding
  }

  // Barcode scanner: fires when Enter is pressed in the search field.
  // Tries barcode lookup first; if no match, keeps the text for normal name search.
  async function handleBarcodeEnter(e: React.KeyboardEvent<HTMLInputElement>) {
    if (e.key !== 'Enter') return;
    const query = search.trim();
    if (!query) return;
    e.preventDefault();
    try {
      const product = await api.get<Product>(`/products/barcode/${encodeURIComponent(query)}`);
      addToCart(product);
      setSearch('');
    } catch {
      // No barcode match — leave text in field so normal search runs
    }
  }

  function changeQty(idx: number, delta: number) {
    setCart(prev => {
      const updated = [...prev];
      const newQty = updated[idx].quantity + delta;
      if (newQty <= 0) {
        return updated.filter((_, i) => i !== idx);
      }
      updated[idx] = { ...updated[idx], quantity: newQty };
      return updated;
    });
  }

  function changePriceOverride(idx: number, price: number, priceInput: string) {
    setCart(prev => {
      const updated = [...prev];
      updated[idx] = { ...updated[idx], unitPrice: price, priceInput, discount: 0, discountInput: '' };
      return updated;
    });
  }

  function changePriceMode(idx: number, mode: 'retail' | 'wholesale') {
    setCart(prev => {
      const updated = [...prev];
      const item = updated[idx];
      const price = mode === 'wholesale' ? (item.product.wholesale_price ?? item.product.sale_price) : item.product.sale_price;
      updated[idx] = { ...item, priceMode: mode, unitPrice: price, priceInput: String(price), discount: 0, discountInput: '' };
      return updated;
    });
  }

  function changeDiscount(idx: number, discount: number, discountType: 'fixed' | 'pct', discountInput: string) {
    setCart(prev => {
      const updated = [...prev];
      updated[idx] = { ...updated[idx], discount, discountType, discountInput };
      return updated;
    });
  }

  function removeItem(idx: number) {
    setCart(prev => prev.filter((_, i) => i !== idx));
  }

  function clearCart() {
    setCart([]);
    setGlobalDiscountInput('');
    setCashReceived('');
    setSelectedCustomer(null);
  }

  const globalDiscount = allowGlobalDiscount ? parseInputCOP(globalDiscountInput) : 0;

  // Totals
  const subtotal = cart.reduce((sum, i) => sum + i.unitPrice * i.quantity - i.discount, 0);
  const total = subtotal - globalDiscount;
  const cashReceivedNum = parseInputCOP(cashReceived);
  const change = payMethod === 'efectivo' ? Math.max(0, cashReceivedNum - total) : 0;

  // Cobrar
  async function handleCobrar() {
    if (cart.length === 0) {
      toast('El carrito está vacío', 'error');
      return;
    }
    if (total <= 0) {
      toast('El total debe ser mayor a cero', 'error');
      return;
    }
    if (payMethod === 'credito' && !selectedCustomer) {
      toast('Selecciona un cliente para ventas a crédito', 'error');
      return;
    }

    setCobrarLoading(true);
    try {
      const pmMap: Record<string, string> = { efectivo: 'cash', transferencia: 'transfer', tarjeta: 'card', credito: 'credit' };
      const body: Record<string, unknown> = {
        payment_method: pmMap[payMethod] ?? payMethod,
        ...(selectedCustomer && { customer_id: selectedCustomer.id }),
        subtotal,
        discount: globalDiscount,
        tax: 0,
        total,
        status: 'completed',
        items: cart.map(i => ({
          product_id: i.product.id,
          quantity: i.quantity,
          unit_price: i.unitPrice,
          discount: i.discount,
          subtotal: i.unitPrice * i.quantity - i.discount,
        })),
      };

      if (payMethod === 'efectivo') {
        body.cash_tendered = cashReceivedNum || total;
        body.change_amount = change;
      }

      const created = await api.post<{ id: number }>('/sales', body);
      clearCart();
      loadProducts(); // refresh stock
      // Fetch ticket data to show preview
      try {
        const ticket = await api.get<TicketData>(`/sales/${created.id}/ticket`);
        setTicketData(ticket);
      } catch {
        toast('Venta completada', 'success');
      }
    } catch (err: unknown) {
      toast((err as Error).message || 'Error al procesar la venta', 'error');
    } finally {
      setCobrarLoading(false);
    }
  }

  const canCobrar = cart.length > 0 && !cobrarLoading && !(payMethod === 'credito' && !selectedCustomer);

  return (
    <AppLayout>
      <Topbar
        title="Punto de Venta"
        subtitle={cart.length > 0 ? `${cart.length} artículo${cart.length !== 1 ? 's' : ''} en carrito` : 'Selecciona productos para agregar al carrito'}
      />

      {/* Mobile tab bar */}
      {isMobile && (
        <div style={{
          display: 'flex',
          borderBottom: '1px solid var(--border)',
          background: 'var(--surface-1)',
        }}>
          <button
            onClick={() => setMobileTab('products')}
            style={{
              flex: 1, padding: '12px', border: 'none',
              background: mobileTab === 'products' ? 'var(--canvas)' : 'var(--surface-1)',
              color: mobileTab === 'products' ? 'var(--accent)' : 'var(--text-muted)',
              fontWeight: mobileTab === 'products' ? 700 : 400,
              fontSize: '14px', cursor: 'pointer',
              borderBottom: mobileTab === 'products' ? '2px solid var(--accent)' : '2px solid transparent',
            }}
          >
            Productos
          </button>
          <button
            onClick={() => setMobileTab('cart')}
            style={{
              flex: 1, padding: '12px', border: 'none',
              background: mobileTab === 'cart' ? 'var(--canvas)' : 'var(--surface-1)',
              color: mobileTab === 'cart' ? 'var(--accent)' : 'var(--text-muted)',
              fontWeight: mobileTab === 'cart' ? 700 : 400,
              fontSize: '14px', cursor: 'pointer',
              borderBottom: mobileTab === 'cart' ? '2px solid var(--accent)' : '2px solid transparent',
            }}
          >
            Carrito {cart.length > 0 && `(${cart.length})`}
          </button>
        </div>
      )}

      <div style={{ display: 'flex', height: isMobile ? 'calc(100% - 49px)' : '100%', overflow: 'hidden' }}>

        {/* ── Left Panel: Products ── */}
        <div className="pos-left" style={{
          flex: isMobile ? '0 0 100%' : '0 0 60%',
          display: isMobile && mobileTab !== 'products' ? 'none' : 'flex',
          flexDirection: 'column',
          borderRight: isMobile ? 'none' : '1px solid var(--border)',
          overflow: 'hidden',
        }}>
          {/* Search */}
          <div style={{ padding: '12px 16px', borderBottom: '1px solid var(--border)', display: 'flex', gap: '8px', alignItems: 'center' }}>
            <div style={{ position: 'relative', flex: 1 }}>
              <span style={{ position: 'absolute', left: '10px', top: '50%', transform: 'translateY(-50%)', color: 'var(--text-muted)', pointerEvents: 'none' }}>
                <IconSearch />
              </span>
              <input
                style={{
                  width: '100%', paddingLeft: '34px', paddingRight: '12px',
                  height: '36px', border: '1px solid var(--border)', borderRadius: '6px',
                  background: 'var(--surface-1)', color: 'var(--text-primary)',
                  fontSize: '13px', boxSizing: 'border-box',
                  outline: 'none',
                }}
                placeholder="Buscar producto o escanea código..."
                value={search}
                onChange={e => setSearch(e.target.value)}
                onKeyDown={handleBarcodeEnter}
              />
            </div>
          </div>

          {/* Category tabs */}
          <div style={{
            display: 'flex', gap: '4px', padding: '8px 16px',
            borderBottom: '1px solid var(--border)', flexWrap: 'wrap',
          }}>
            <button
              className={`cat-tab${selectedCat === 0 ? ' active' : ''}`}
              onClick={() => setSelectedCat(0)}
            >
              Todos
            </button>
            {categories.map(c => (
              <button
                key={c.id}
                className={`cat-tab${selectedCat === c.id ? ' active' : ''}`}
                onClick={() => setSelectedCat(c.id)}
              >
                {c.name}
              </button>
            ))}
          </div>

          {/* Product grid */}
          <div style={{ flex: 1, overflowY: 'auto', padding: '12px 16px' }}>
            {loading ? (
              <div style={{ color: 'var(--text-muted)', fontSize: '13px', padding: '24px', textAlign: 'center' }}>
                Cargando...
              </div>
            ) : products.length === 0 ? (
              <div style={{ color: 'var(--text-muted)', fontSize: '13px', padding: '40px', textAlign: 'center' }}>
                No hay productos
              </div>
            ) : (
              <div style={{ display: 'grid', gridTemplateColumns: isMobile ? 'repeat(2, 1fr)' : 'repeat(3, 1fr)', gap: '10px' }}>
                {products.map(p => (
                  <ProductCard key={p.id} product={p} onAdd={addToCart} />
                ))}
              </div>
            )}
          </div>
        </div>

        {/* ── Right Panel: Cart ── */}
        <div className="pos-right" style={{
          flex: isMobile ? '0 0 100%' : '0 0 40%',
          display: isMobile && mobileTab !== 'cart' ? 'none' : 'flex',
          flexDirection: 'column',
          overflow: 'hidden',
          background: 'var(--canvas)',
        }}>
          {/* Cart header */}
          <div style={{
            padding: '12px 16px',
            borderBottom: '1px solid var(--border)',
            display: 'flex', alignItems: 'center', justifyContent: 'space-between',
          }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: '8px', color: 'var(--text-primary)', fontWeight: 600, fontSize: '14px' }}>
              <IconCart />
              Carrito
            </div>
            {cart.length > 0 && (
              <button
                onClick={clearCart}
                style={{
                  fontSize: '12px', color: 'var(--text-muted)', background: 'transparent',
                  border: 'none', cursor: 'pointer', padding: '2px 6px', borderRadius: '4px',
                }}
              >
                Vaciar
              </button>
            )}
          </div>

          {/* Cart items */}
          <div style={{ flex: 1, overflowY: 'auto' }}>
            {cart.length === 0 ? (
              <div style={{
                display: 'flex', flexDirection: 'column', alignItems: 'center',
                justifyContent: 'center', height: '100%', gap: '12px',
                color: 'var(--text-muted)',
              }}>
                <IconCart />
                <span style={{ fontSize: '13px' }}>El carrito está vacío</span>
              </div>
            ) : (
              <div style={{ paddingTop: '4px' }}>
                {cart.map((item, idx) => (
                  <CartItemRow
                    key={item.product.id}
                    item={item}
                    isAdmin={isAdmin}
                    onQtyChange={delta => changeQty(idx, delta)}
                    onDiscountChange={(discount, discountType, discountInput) => changeDiscount(idx, discount, discountType, discountInput)}
                    onPriceChange={(price, priceInput) => changePriceOverride(idx, price, priceInput)}
                    onPriceModeChange={mode => changePriceMode(idx, mode)}
                    onRemove={() => removeItem(idx)}
                  />
                ))}
              </div>
            )}
          </div>

          {/* Totals + Payment */}
          <div style={{ borderTop: '1px solid var(--border)', padding: '12px 16px', display: 'flex', flexDirection: 'column', gap: '10px' }}>

            {/* Subtotal */}
            <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: '13px', color: 'var(--text-secondary)' }}>
              <span>Subtotal</span>
              <span style={{ fontVariantNumeric: 'tabular-nums' }}>{formatCOP(subtotal)}</span>
            </div>

            {/* Discount */}
            {allowGlobalDiscount && (
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', fontSize: '13px', color: 'var(--text-secondary)' }}>
                <span>Descuento</span>
                <input
                  type="text"
                  inputMode="numeric"
                  value={globalDiscountInput}
                  placeholder="0"
                  onChange={e => setGlobalDiscountInput(formatInputCOP(e.target.value))}
                  style={{
                    width: '100px', textAlign: 'right', padding: '3px 8px',
                    border: '1px solid var(--border)', borderRadius: '4px',
                    background: 'var(--surface-1)', color: 'var(--text-primary)',
                    fontSize: '13px', fontVariantNumeric: 'tabular-nums',
                  }}
                />
              </div>
            )}

            {/* Total */}
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
              <span style={{ fontSize: '15px', fontWeight: 700, color: 'var(--text-primary)' }}>Total</span>
              <span style={{ fontSize: '22px', fontWeight: 700, color: 'var(--accent)', fontVariantNumeric: 'tabular-nums' }}>
                {formatCOP(total)}
              </span>
            </div>

            {/* Payment methods */}
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '6px' }}>
              {(['efectivo', 'transferencia', 'tarjeta', 'credito'] as PaymentMethod[]).map(m => (
                <PayBtn
                  key={m}
                  value={m}
                  label={m.charAt(0).toUpperCase() + m.slice(1)}
                  active={payMethod === m}
                  onClick={() => { setPayMethod(m); if (m !== 'credito') setSelectedCustomer(null); }}
                />
              ))}
            </div>

            {/* Customer picker — only for credit */}
            {payMethod === 'credito' && (
              <div>
                <div style={{ fontSize: '11px', fontWeight: 600, color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '0.4px', marginBottom: '6px' }}>
                  Cliente *
                </div>
                <CustomerPicker selected={selectedCustomer} onSelect={setSelectedCustomer} />
                {!selectedCustomer && (
                  <div style={{ fontSize: '11px', color: 'var(--danger)', marginTop: '4px' }}>
                    Requerido para ventas a crédito
                  </div>
                )}
                {selectedCustomer?.notes && (
                  <div style={{
                    marginTop: '6px',
                    padding: '8px 10px',
                    background: 'var(--accent-dim)',
                    border: '1px solid var(--accent)',
                    borderRadius: '6px',
                    fontSize: '12px',
                    color: 'var(--text-primary)',
                  }}>
                    <span style={{ fontWeight: 600, color: 'var(--accent)', marginRight: '4px' }}>📝</span>
                    {selectedCustomer.notes}
                  </div>
                )}
                {(() => {
                  if (!selectedCustomer?.credit_limit || customerBalance === null) return null;
                  const cartTotal = cart.reduce((s, i) => s + i.unitPrice * i.quantity - i.discount, 0);
                  if (customerBalance + cartTotal > selectedCustomer.credit_limit) {
                    return (
                      <div style={{
                        marginTop: '6px',
                        padding: '8px 10px',
                        background: 'var(--danger-dim)',
                        border: '1px solid var(--danger)',
                        borderRadius: '6px',
                        fontSize: '12px',
                        color: 'var(--danger)',
                        fontWeight: 500,
                      }}>
                        ⚠️ Tope de crédito: ${selectedCustomer.credit_limit.toLocaleString('es-CO')}. Deuda actual: ${customerBalance.toLocaleString('es-CO')}.
                      </div>
                    );
                  }
                  return null;
                })()}
              </div>
            )}

            {/* Efectivo: recibido + cambio */}
            {payMethod === 'efectivo' && (
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
                      width: '110px', textAlign: 'right', padding: '3px 8px',
                      border: '1px solid var(--border)', borderRadius: '4px',
                      background: 'var(--surface-1)', color: 'var(--text-primary)',
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

            {/* Cobrar button */}
            <button
              onClick={handleCobrar}
              disabled={!canCobrar}
              style={{
                width: '100%',
                padding: '14px',
                minHeight: '48px',
                borderRadius: '8px',
                border: 'none',
                background: canCobrar ? 'var(--accent)' : 'var(--surface-3)',
                color: canCobrar ? '#000' : 'var(--text-muted)',
                fontSize: '15px',
                fontWeight: 700,
                cursor: canCobrar ? 'pointer' : 'not-allowed',
                fontVariantNumeric: 'tabular-nums',
                transition: 'opacity 120ms ease',
                opacity: cobrarLoading ? 0.7 : 1,
              }}
            >
              {cobrarLoading ? 'Procesando...' : `Cobrar ${formatCOP(total)}`}
            </button>
          </div>
        </div>
      </div>

      {/* Post-sale ticket modal */}
      {ticketData && (
        <div style={{
          position: 'fixed', inset: 0,
          background: 'rgba(0,0,0,0.6)',
          display: 'flex', alignItems: isMobile ? 'flex-end' : 'center', justifyContent: 'center',
          zIndex: 1000,
          padding: isMobile ? '0' : '24px',
        }}>
          <div style={{
            background: 'var(--surface-1)',
            border: '1px solid var(--border)',
            borderRadius: isMobile ? '12px 12px 0 0' : '12px',
            padding: isMobile ? '16px' : '24px',
            width: isMobile ? '100%' : 'auto',
            maxHeight: '90vh',
            overflowY: 'auto',
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            gap: '16px',
          }}>
            <div style={{ fontSize: '15px', fontWeight: 600, color: 'var(--text-primary)', alignSelf: 'flex-start' }}>
              ✓ Venta completada
            </div>
            <TicketPreview
              ticket={ticketData}
              onClose={() => setTicketData(null)}
            />
          </div>
        </div>
      )}
    </AppLayout>
  );
}
