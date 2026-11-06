'use client';

import { useState, useEffect, useCallback, useRef } from 'react';
import AppLayout from '@/components/AppLayout';
import Topbar from '@/components/Topbar';
import { api } from '@/lib/api';
import { useToast } from '@/components/Toast';
import TicketPreview, { type TicketData } from '@/components/TicketPreview';
import ConfirmDialog from '@/components/ConfirmDialog';
import { formatCOP, formatInputCOP, parseInputCOP } from '@/lib/format';
import { getUser } from '@/lib/auth';
import { useIsMobile } from '@/hooks/useIsMobile';
import { useBusinessConfig } from '@/lib/business-config-context';
import { shouldAutoPrint, handleTicketPrint } from '@/lib/printFlow';
import { calcSubtotal, calcTax, calcTotal } from '@/lib/saleCalc';
import { readActiveCart, writeActiveCart } from '@/lib/activeCart';
import SalePaymentPanel, { type CheckoutData, type Customer } from '@/components/SalePaymentPanel';

// ─── Types ───────────────────────────────────────────────────────────────────

interface Category {
  id: number;
  name: string;
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
  tax_rate: number;
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

interface ParkedCart {
  id: string;
  name: string;
  createdAt: number;
  items: CartItem[];
  customer: Customer | null;
  globalDiscountInput: string;
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

const PARKED_CARTS_KEY = 'pos.parkedCarts';

function loadParkedCarts(): ParkedCart[] {
  if (typeof window === 'undefined') return [];
  try {
    const raw = window.localStorage.getItem(PARKED_CARTS_KEY);
    if (!raw) return [];
    const parsed = JSON.parse(raw);
    return Array.isArray(parsed) ? parsed : [];
  } catch {
    return [];
  }
}

function saveParkedCarts(carts: ParkedCart[]) {
  try {
    window.localStorage.setItem(PARKED_CARTS_KEY, JSON.stringify(carts));
  } catch {
    // storage unavailable — parked carts just won't persist across reloads
  }
}

function formatElapsed(ts: number): string {
  const mins = Math.max(0, Math.round((Date.now() - ts) / 60000));
  if (mins < 1) return 'hace un momento';
  if (mins === 1) return 'hace 1 min';
  if (mins < 60) return `hace ${mins} min`;
  const hours = Math.round(mins / 60);
  return hours === 1 ? 'hace 1 h' : `hace ${hours} h`;
}

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

const IconClock = () => (
  <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
    <circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/>
  </svg>
);

// ─── Stock badge ─────────────────────────────────────────────────────────────

function StockBadge({ product }: { product: Product }) {
  if (product.type === 'service') return null;
  const { stock, min_stock } = product;
  let bg = 'rgba(76,175,130,0.15)';
  let color = 'var(--success)';
  const label = String(stock);
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
        <span style={{ display: 'flex', alignItems: 'center', gap: '6px', flex: 1, minWidth: 0 }}>
          <span style={{ fontSize: '13px', fontWeight: 600, color: 'var(--text-primary)', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
            {item.product.name}
          </span>
          {!item.product.active && (
            <span style={{
              fontSize: '10px', fontWeight: 600, padding: '1px 6px', borderRadius: '4px',
              background: 'var(--danger-dim)', color: 'var(--danger)', flexShrink: 0,
            }}>
              Ya no disponible
            </span>
          )}
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

// ─── Main Page ────────────────────────────────────────────────────────────────

export default function POSPage() {
  const { toast } = useToast();
  // Restaurant mode uses table orders instead of parked carts
  const { config: businessConfig } = useBusinessConfig();
  const showParked = !businessConfig.restaurant_mode;
  const isAdmin = getUser()?.role === 'admin';

  // Mobile detection
  const isMobile = useIsMobile(768);
  const [mobileTab, setMobileTab] = useState<'products' | 'cart'>('products');

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

  // Checkout key for reset
  const [checkoutKey, setCheckoutKey] = useState(0);

  // Customer (for credit limit & parked carts)
  const [selectedCustomer, setSelectedCustomer] = useState<Customer | null>(null);

  // Sale loading
  const [cobrarLoading, setCobrarLoading] = useState(false);

  // Post-sale ticket modal
  const [ticketData, setTicketData] = useState<TicketData | null>(null);

  // Parked carts (venta en espera)
  const [parkedCarts, setParkedCarts] = useState<ParkedCart[]>([]);
  const [parkListOpen, setParkListOpen] = useState(false);
  const [parkNameInput, setParkNameInput] = useState<string | null>(null); // non-null while naming modal is open
  const [resumeCandidate, setResumeCandidate] = useState<ParkedCart | null>(null); // pending confirm when active cart isn't empty
  const [discardTarget, setDiscardTarget] = useState<ParkedCart | null>(null);

  // Live cart persistence (R-73): the cart is restored once, right after the
  // first products load (loadProducts), and saved on every change afterwards.
  const [cartHydrated, setCartHydrated] = useState(false);
  const cartHydrationStarted = useRef(false);
  useEffect(() => {
    if (!cartHydrated) return;
    writeActiveCart(window.localStorage, { items: cart, customer: selectedCustomer, globalDiscountInput });
  }, [cart, selectedCustomer, globalDiscountInput, cartHydrated]);

  useEffect(() => {
    queueMicrotask(() => setParkedCarts(loadParkedCarts()));
  }, []);

  // Debounce search
  useEffect(() => {
    if (searchTimer.current) clearTimeout(searchTimer.current);
    searchTimer.current = setTimeout(() => setDebouncedSearch(search), 200);
    return () => { if (searchTimer.current) clearTimeout(searchTimer.current); };
  }, [search]);

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
      if (!cartHydrationStarted.current) {
        // Restore the cart a session expiry left behind. Product data is refreshed;
        // a product that is gone or inactive is flagged inactive, as doResume does.
        cartHydrationStarted.current = true;
        const saved = readActiveCart<CartItem, Customer>(window.localStorage);
        if (saved) {
          setCart(saved.items.map(item => {
            const fresh = (data ?? []).find(p => p.id === item.product.id);
            return fresh ? { ...item, product: fresh } : { ...item, product: { ...item.product, active: false } };
          }));
          setSelectedCustomer(saved.customer);
          setGlobalDiscountInput(saved.globalDiscountInput);
          toast('Recuperamos la venta que tenías abierta', 'success');
        }
        setCartHydrated(true);
      }
    } catch {
      toast('Error al cargar productos', 'error');
    } finally {
      setLoading(false);
    }
  }, [debouncedSearch, selectedCat, toast]);

  useEffect(() => { queueMicrotask(loadProducts); }, [loadProducts]);

  // Cart operations
  function addToCart(product: Product) {
    let blocked = false;
    setCart(prev => {
      const idx = prev.findIndex(i => i.product.id === product.id);
      const nextQty = idx >= 0 ? prev[idx].quantity + 1 : 1;
      if (product.type === 'product' && nextQty > product.stock) {
        blocked = true;
        return prev;
      }
      if (idx >= 0) {
        const updated = [...prev];
        updated[idx] = { ...updated[idx], quantity: nextQty };
        return updated;
      }
      return [...prev, { product, quantity: 1, unitPrice: product.sale_price, priceInput: String(product.sale_price), priceMode: 'retail' as const, discount: 0, discountType: 'fixed', discountInput: '' }];
    });
    if (blocked) toast(`Solo hay ${product.stock} unidades disponibles de ${product.name}`, 'error');
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
    let blocked = false;
    setCart(prev => {
      const updated = [...prev];
      const item = updated[idx];
      const newQty = item.quantity + delta;
      if (newQty <= 0) {
        return updated.filter((_, i) => i !== idx);
      }
      if (delta > 0 && item.product.type === 'product' && newQty > item.product.stock) {
        blocked = true;
        return prev;
      }
      updated[idx] = { ...item, quantity: newQty };
      return updated;
    });
    if (blocked) {
      const item = cart[idx];
      toast(`Solo hay ${item.product.stock} unidades disponibles de ${item.product.name}`, 'error');
    }
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
    setSelectedCustomer(null);
    setCheckoutKey(k => k + 1);
  }

  // Parked carts (venta en espera) ────────────────────────────────────────────

  function openParkModal() {
    setParkNameInput('');
  }

  function confirmPark() {
    const name = (parkNameInput || '').trim() || `Venta ${parkedCarts.length + 1}`;
    const parked: ParkedCart = {
      id: `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
      name,
      createdAt: Date.now(),
      items: cart,
      customer: selectedCustomer,
      globalDiscountInput,
    };
    const updated = [...parkedCarts, parked];
    setParkedCarts(updated);
    saveParkedCarts(updated);
    clearCart();
    setParkNameInput(null);
    toast(`"${name}" dejado en espera`, 'success');
  }

  function requestResume(parked: ParkedCart) {
    if (cart.length > 0) {
      setResumeCandidate(parked);
    } else {
      doResume(parked);
    }
  }

  function doResume(parked: ParkedCart) {
    // Refresh product refs so deactivated/deleted products surface a warning instead of stale data.
    const restoredItems = parked.items.map(item => {
      const fresh = products.find(p => p.id === item.product.id);
      if (fresh) return { ...item, product: fresh };
      return { ...item, product: { ...item.product, active: false } };
    });
    setCart(restoredItems);
    setSelectedCustomer(parked.customer);
    setGlobalDiscountInput(parked.globalDiscountInput);
    const updated = parkedCarts.filter(p => p.id !== parked.id);
    setParkedCarts(updated);
    saveParkedCarts(updated);
    setResumeCandidate(null);
    setParkListOpen(false);
    if (isMobile) setMobileTab('cart');
  }

  function doDiscard(id: string) {
    const updated = parkedCarts.filter(p => p.id !== id);
    setParkedCarts(updated);
    saveParkedCarts(updated);
    setDiscardTarget(null);
  }

  const globalDiscount = allowGlobalDiscount ? parseInputCOP(globalDiscountInput) : 0;

  // Totals — calcSubtotal/calcTax/calcTotal live in lib/saleCalc.ts (unit
  // tested) and must match internal/service/sale.go's recalculation exactly,
  // or the server rejects the sale as "total inconsistente".
  const calcItems = cart.map(i => ({ unitPrice: i.unitPrice, quantity: i.quantity, discount: i.discount, taxRate: i.product.tax_rate ?? 0 }));
  const subtotal = calcSubtotal(calcItems);
  const tax = calcTax(calcItems);
  const total = calcTotal(calcItems, globalDiscount);

  // Cobrar
  async function handleCobrar(data: CheckoutData) {
    if (cart.length === 0) {
      toast('El carrito está vacío', 'error');
      return;
    }

    setCobrarLoading(true);
    try {
      const pmMap: Record<string, string> = { efectivo: 'cash', transferencia: 'transfer', tarjeta: 'card', credito: 'credit' };
      const body: Record<string, unknown> = {
        payment_method: pmMap[data.payMethod] ?? data.payMethod,
        payments: data.paymentLines
          .filter(l => l.amount > 0)
          .map(l => ({ method: pmMap[l.method] ?? l.method, amount: l.amount })),
        ...(selectedCustomer && { customer_id: selectedCustomer.id }),
        subtotal,
        discount: globalDiscount,
        tax,
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

      if (!data.isMixedMode && data.payMethod === 'efectivo') {
        body.cash_tendered = data.cashReceivedNum || total;
        body.change_amount = data.change;
      }

      const created = await api.post<{ id: number }>('/sales', body);
      clearCart();
      loadProducts(); // refresh stock
      // Fetch ticket data to show preview
      try {
        const ticket = await api.get<TicketData>(`/sales/${created.id}/ticket`);
        setTicketData(ticket);
        // Direct printing: send the ticket once, the preview stays on screen.
        if (shouldAutoPrint(businessConfig.print_method)) {
          void handleTicketPrint(businessConfig.print_method, created.id, {
            post: (path, body) => api.post(path, body),
            toast,
            browserPrint: () => {},
          });
        }
      } catch {
        toast('Venta completada', 'success');
      }
    } catch (err: unknown) {
      toast((err as Error).message || 'Error al procesar la venta', 'error');
    } finally {
      setCobrarLoading(false);
    }
  }

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
          alignItems: 'center',
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
          {showParked && parkedCarts.length > 0 && (
            <button
              onClick={() => setParkListOpen(true)}
              style={{
                display: 'flex', alignItems: 'center', gap: '4px',
                fontSize: '11px', fontWeight: 600, color: 'var(--accent)',
                background: 'var(--accent-dim)', border: '1px solid var(--accent)',
                borderRadius: '999px', padding: '4px 9px', cursor: 'pointer',
                margin: '0 8px', flexShrink: 0,
              }}
            >
              <IconClock />
              En espera ({parkedCarts.length})
            </button>
          )}
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
            <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
              {showParked && !isMobile && parkedCarts.length > 0 && (
                <button
                  onClick={() => setParkListOpen(true)}
                  style={{
                    display: 'flex', alignItems: 'center', gap: '5px',
                    fontSize: '12px', fontWeight: 600, color: 'var(--accent)',
                    background: 'var(--accent-dim)', border: '1px solid var(--accent)',
                    borderRadius: '999px', padding: '4px 10px', cursor: 'pointer',
                  }}
                >
                  <IconClock />
                  En espera ({parkedCarts.length})
                </button>
              )}
              {showParked && cart.length > 0 && (
                <button
                  onClick={openParkModal}
                  style={{
                    fontSize: '12px', color: 'var(--text-secondary)', background: 'transparent',
                    border: '1px solid var(--border)', cursor: 'pointer', padding: '3px 8px', borderRadius: '4px',
                  }}
                >
                  Dejar en espera
                </button>
              )}
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

            {/* IVA */}
            {tax > 0 && (
              <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: '13px', color: 'var(--text-secondary)' }}>
                <span>IVA</span>
                <span style={{ fontVariantNumeric: 'tabular-nums' }}>{formatCOP(tax)}</span>
              </div>
            )}

            {/* Total */}
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
              <span style={{ fontSize: '15px', fontWeight: 700, color: 'var(--text-primary)' }}>Total</span>
              <span style={{ fontSize: '22px', fontWeight: 700, color: 'var(--accent)', fontVariantNumeric: 'tabular-nums' }}>
                {formatCOP(total)}
              </span>
            </div>

            {/* Payment Panel */}
            <SalePaymentPanel
              key={checkoutKey}
              total={total}
              cartTotal={calcItems.reduce((s, i) => s + i.unitPrice * i.quantity - i.discount, 0)}
              selectedCustomer={selectedCustomer}
              onCustomerChange={setSelectedCustomer}
              onCheckout={handleCobrar}
              loading={cobrarLoading}
              disabled={cart.length === 0}
            />
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

      {/* Park name modal */}
      {parkNameInput !== null && (
        <div
          style={{
            position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.6)',
            display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 9998,
          }}
          onClick={() => setParkNameInput(null)}
        >
          <div
            style={{
              background: 'var(--surface-1)', border: '1px solid var(--border-emphasis)',
              borderRadius: '12px', padding: '24px', width: '100%', maxWidth: '360px',
              boxShadow: '0 8px 32px rgba(0,0,0,0.4)',
            }}
            onClick={e => e.stopPropagation()}
          >
            <h3 style={{ margin: '0 0 4px', fontSize: '15px', fontWeight: 600, color: 'var(--text-primary)' }}>
              Dejar carrito en espera
            </h3>
            <p style={{ margin: '0 0 14px', fontSize: '13px', color: 'var(--text-secondary)' }}>
              Ponle un nombre corto para identificarlo (ej. &quot;Pedro&quot;).
            </p>
            <input
              autoFocus
              value={parkNameInput}
              onChange={e => setParkNameInput(e.target.value)}
              onKeyDown={e => { if (e.key === 'Enter') confirmPark(); }}
              placeholder={`Venta ${parkedCarts.length + 1}`}
              style={{ width: '100%', boxSizing: 'border-box', fontSize: '13px', marginBottom: '18px' }}
            />
            <div style={{ display: 'flex', gap: '8px', justifyContent: 'flex-end' }}>
              <button className="btn btn-sm" onClick={() => setParkNameInput(null)}>Cancelar</button>
              <button className="btn btn-sm btn-accent" onClick={confirmPark}>Dejar en espera</button>
            </div>
          </div>
        </div>
      )}

      {/* Parked carts list (drawer/modal) */}
      {parkListOpen && (
        <div
          style={{
            position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.6)',
            display: 'flex', alignItems: isMobile ? 'flex-end' : 'center', justifyContent: 'center',
            zIndex: 9997,
          }}
          onClick={() => setParkListOpen(false)}
        >
          <div
            style={{
              background: 'var(--surface-1)', border: '1px solid var(--border-emphasis)',
              borderRadius: isMobile ? '12px 12px 0 0' : '12px',
              padding: '20px', width: isMobile ? '100%' : '420px',
              maxHeight: '80vh', overflowY: 'auto',
              boxShadow: '0 8px 32px rgba(0,0,0,0.4)',
            }}
            onClick={e => e.stopPropagation()}
          >
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '14px' }}>
              <h3 style={{ margin: 0, fontSize: '15px', fontWeight: 600, color: 'var(--text-primary)' }}>
                Ventas en espera
              </h3>
              <button
                onClick={() => setParkListOpen(false)}
                style={{ background: 'transparent', border: 'none', color: 'var(--text-muted)', fontSize: '18px', cursor: 'pointer', lineHeight: 1 }}
              >
                ×
              </button>
            </div>
            {parkedCarts.length === 0 ? (
              <p style={{ fontSize: '13px', color: 'var(--text-muted)', margin: 0 }}>No hay carritos en espera.</p>
            ) : (
              <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
                {parkedCarts.map(parked => {
                  const parkedTotal = parked.items.reduce((s, i) => s + i.unitPrice * i.quantity - i.discount, 0);
                  return (
                    <div
                      key={parked.id}
                      style={{
                        border: '1px solid var(--border)', borderRadius: '8px', padding: '10px 12px',
                        display: 'flex', flexDirection: 'column', gap: '6px', background: 'var(--surface-2)',
                      }}
                    >
                      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                        <span style={{ fontSize: '13px', fontWeight: 600, color: 'var(--text-primary)' }}>{parked.name}</span>
                        <span style={{ fontSize: '11px', color: 'var(--text-muted)' }}>{formatElapsed(parked.createdAt)}</span>
                      </div>
                      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', fontSize: '12px', color: 'var(--text-secondary)' }}>
                        <span>{parked.items.length} artículo{parked.items.length !== 1 ? 's' : ''}</span>
                        <span style={{ fontWeight: 600, fontVariantNumeric: 'tabular-nums', color: 'var(--accent)' }}>{formatCOP(parkedTotal)}</span>
                      </div>
                      <div style={{ display: 'flex', gap: '8px', justifyContent: 'flex-end' }}>
                        <button className="btn btn-sm btn-danger" onClick={() => setDiscardTarget(parked)}>Descartar</button>
                        <button className="btn btn-sm btn-accent" onClick={() => requestResume(parked)}>Retomar</button>
                      </div>
                    </div>
                  );
                })}
              </div>
            )}
          </div>
        </div>
      )}

      {/* Confirm: resume while active cart has items */}
      <ConfirmDialog
        open={resumeCandidate !== null}
        title="Reemplazar carrito activo"
        message={`El carrito actual tiene ${cart.length} artículo${cart.length !== 1 ? 's' : ''}. Al retomar "${resumeCandidate?.name}" se reemplazará. ¿Continuar? (el carrito actual no se guarda)`}
        confirmLabel="Retomar"
        danger
        onConfirm={() => resumeCandidate && doResume(resumeCandidate)}
        onCancel={() => setResumeCandidate(null)}
      />

      {/* Confirm: discard parked cart */}
      <ConfirmDialog
        open={discardTarget !== null}
        title="Descartar venta en espera"
        message={`Se eliminará "${discardTarget?.name}" y no se podrá recuperar.`}
        confirmLabel="Descartar"
        danger
        onConfirm={() => discardTarget && doDiscard(discardTarget.id)}
        onCancel={() => setDiscardTarget(null)}
      />
    </AppLayout>
  );
}
