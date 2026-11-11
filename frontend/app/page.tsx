'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import Link from 'next/link';
import AppLayout from '@/components/AppLayout';
import Topbar from '@/components/Topbar';
import { api, getSetupStatus } from '@/lib/api';
import { getUser } from '@/lib/auth';
import { useIsMobile } from '@/hooks/useIsMobile';
import { formatCOP } from '@/lib/format';
import BackupBanner from '@/components/BackupBanner';
import AbonoModal from '@/components/AbonoModal';
import VoidSaleDialog from '@/components/VoidSaleDialog';
import { useToast } from '@/components/Toast';
import { canVoidSale, voidErrorMessage } from '@/lib/saleVoid';

// ─── Types ────────────────────────────────────────────────────────────────────

interface SalesSnap {
  total_revenue: number;
  total_count: number;
}

// The register id lives only in the nested `register` object (US-095 D-4).
interface Register {
  opening_amount: number;
  expected: number;
  cash_sales: number;
  cash_out: number;
  register: {
    id: number;
    status: string;
    opened_at: string;
    opening_amount: number;
  };
}

interface Portfolio {
  total: number;
  count: number;
}

interface RecentSale {
  id: number;
  created_at: string;
  customer_id: number | null;
  customer_name: string;
  item_count: number;
  total: number;
  payment_method: string;
  status: string;
}

interface DashboardData {
  today: SalesSnap;
  yesterday: SalesSnap;
  register: Register | null;
  portfolio: Portfolio;
  recent_sales: RecentSale[];
}

interface TopMarginItem {
  name: string;
  margin_pct: number;
  units_sold: number;
}

interface LowStockItem {
  product_id: number;
  name: string;
  stock: number;
  min_stock: number;
  avg_daily: number | null;
  days_remaining: number | null;
}

interface AgingReceivable {
  customer_id: number;
  name: string;
  balance: number;
  days_overdue: number;
}

interface TodayData {
  sold_today: number;
  credit_new_today: number;
  credit_collected_today: number;
  cash_estimated: number;
  top_product_name: string;
  top_product_qty: number;
  gross_profit: number;
  expenses_today: number;
  net_profit: number;
  unconfigured_cost_count: number;
  top_by_margin: TopMarginItem[];
  low_stock: LowStockItem[];
  aging_receivables: AgingReceivable[];
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

function greeting(): string {
  const h = new Date().getHours();
  if (h >= 5 && h < 12) return 'Buenos días';
  if (h >= 12 && h < 19) return 'Buenas tardes';
  return 'Buenas noches';
}

function greetingEmoji(): string {
  const h = new Date().getHours();
  if (h >= 5 && h < 12) return '☀️';
  if (h >= 12 && h < 19) return '🌤️';
  return '🌙';
}

function formatDate(): string {
  return new Date().toLocaleDateString('es-CO', {
    weekday: 'long',
    year: 'numeric',
    month: 'long',
    day: 'numeric',
  });
}

function formatTime(iso: string): string {
  return new Date(iso).toLocaleTimeString('es-CO', { hour: '2-digit', minute: '2-digit' });
}

function deltaPercent(today: number, yesterday: number): number | null {
  if (yesterday === 0) return null;
  return Math.round(((today - yesterday) / yesterday) * 100);
}

function agingColor(days: number): string {
  if (days > 14) return 'var(--danger)';
  if (days >= 7) return '#d97706';
  return 'var(--text-muted)';
}

function agingBg(days: number): string {
  if (days > 14) return 'var(--danger-dim)';
  if (days >= 7) return 'rgba(217,119,6,0.12)';
  return 'var(--surface-2)';
}

const METHOD_LABELS: Record<string, string> = {
  cash: 'Efectivo',
  transfer: 'Transferencia',
  card: 'Tarjeta',
  credit: 'Fiado',
};

const METHOD_BADGE: Record<string, string> = {
  cash: 'badge-efectivo',
  transfer: 'badge-transfer',
  card: 'badge-tarjeta',
  credit: 'badge-credito',
};

// ─── Sub-components ───────────────────────────────────────────────────────────

function StatCard({
  label,
  value,
  delta,
  sub,
  danger,
}: {
  label: string;
  value: string;
  delta?: number | null;
  sub?: string;
  danger?: boolean;
}) {
  return (
    <div className="stat-card" style={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
      <div className="stat-label">{label}</div>
      <div
        className="stat-value"
        style={{ color: danger ? 'var(--danger)' : 'var(--text-primary)' }}
      >
        {value}
      </div>
      {delta !== null && delta !== undefined && (
        <div
          className={`stat-delta ${delta >= 0 ? 'up' : 'down'}`}
          style={{ display: 'flex', alignItems: 'center', gap: '3px' }}
        >
          <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round">
            {delta >= 0
              ? <polyline points="18 15 12 9 6 15" />
              : <polyline points="6 9 12 15 18 9" />}
          </svg>
          {Math.abs(delta)}% vs ayer
        </div>
      )}
      {delta === null && sub && (
        <div style={{ fontSize: '11px', color: 'var(--text-muted)' }}>{sub}</div>
      )}
    </div>
  );
}

function CuadernoTile({
  label,
  value,
  sub,
  warn,
  good,
}: {
  label: string;
  value: string;
  sub?: string;
  warn?: boolean;
  good?: boolean;
}) {
  const color = warn ? 'var(--danger)' : good ? 'var(--success)' : 'var(--text-primary)';
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '4px' }}>
      <div style={{ fontSize: '11px', color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '0.04em' }}>{label}</div>
      <div style={{ fontSize: '20px', fontWeight: 700, color, lineHeight: 1.2 }}>{value}</div>
      {sub && <div style={{ fontSize: '11px', color: 'var(--text-muted)' }}>{sub}</div>}
    </div>
  );
}

// ─── Page ─────────────────────────────────────────────────────────────────────

export default function DashboardPage() {
  const router = useRouter();
  const user = getUser();
  const [data, setData] = useState<DashboardData | null>(null);
  const [loading, setLoading] = useState(true);
  const [todayData, setTodayData] = useState<TodayData | null>(null);
  const [showVoided, setShowVoided] = useState(false);
  const [allSales, setAllSales] = useState<RecentSale[] | null>(null);
  const [loadingAll, setLoadingAll] = useState(false);
  const isMobile = useIsMobile(768);
  const [payTarget, setPayTarget] = useState<AgingReceivable | null>(null);
  const [voidTarget, setVoidTarget] = useState<RecentSale | null>(null);
  const { toast } = useToast();

  // Unauthenticated visits land here first (this is the app's root route) —
  // route to onboarding or login depending on first-run state instead of
  // rendering nothing.
  useEffect(() => {
    if (user) return;
    getSetupStatus()
      .then(status => router.replace(status.completed ? '/login' : '/onboarding'))
      .catch(() => router.replace('/login'));
  }, [user, router]);

  // Loads every dashboard figure; re-run after a void so the list and the
  // totals show the server's values (US-095 AC-6, AC-8).
  function loadDashboard() {
    api
      .get<DashboardData>('/dashboard')
      .then(setData)
      .catch(() => {})
      .finally(() => setLoading(false));
    api
      .get<TodayData>('/dashboard/today')
      .then(setTodayData)
      .catch(() => {});
  }

  useEffect(() => {
    if (!user) return;
    if (user.role !== 'admin') {
      router.replace('/pos');
      return;
    }
    loadDashboard();
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  function afterVoidAttempt() {
    setVoidTarget(null);
    // The cached `Ver anuladas` list is stale now; re-fetch on next opening.
    setAllSales(null);
    loadDashboard();
  }

  function handleVoided(saleId: number) {
    afterVoidAttempt();
    toast(`Venta #${saleId} anulada`, 'success');
  }

  function handleVoidFailed(_saleId: number, message: string) {
    afterVoidAttempt();
    toast(voidErrorMessage(message), 'error');
  }

  function toggleVoided() {
    if (!showVoided && allSales === null) {
      setLoadingAll(true);
      api
        .get<RecentSale[]>('/sales?include_voided=true&limit=50')
        .then(setAllSales)
        .catch(() => {})
        .finally(() => setLoadingAll(false));
    }
    setShowVoided(v => !v);
  }

  function handlePaymentSuccess(customerId: number, paidAmount: number) {
    setPayTarget(null);
    setTodayData(prev => {
      if (!prev) return prev;
      const updated = prev.aging_receivables
        .map(r => r.customer_id === customerId
          ? { ...r, balance: r.balance - paidAmount }
          : r
        )
        .filter(r => r.balance > 0);
      return { ...prev, aging_receivables: updated };
    });
  }

  const baseSales = showVoided ? (allSales ?? []) : (data?.recent_sales ?? []);
  const displayedSales = isMobile ? baseSales.slice(0, 3) : baseSales;
  const displayedLoading = showVoided ? loadingAll : loading;

  if (!user || user.role !== 'admin') return null;

  const todayDelta = data ? deltaPercent(data.today.total_revenue, data.yesterday.total_revenue) : null;
  const txDelta = data ? deltaPercent(data.today.total_count, data.yesterday.total_count) : null;
  const isOpen = !!data?.register?.register?.status;

  const showTopMargin = (todayData?.top_by_margin?.length ?? 0) >= 2;
  const showLowStock = (todayData?.low_stock?.length ?? 0) > 0;
  const showAging = (todayData?.aging_receivables?.length ?? 0) > 0;

  // Gross profit display: show "—" if all items had cost=0
  const allUnconfigured =
    todayData !== null &&
    todayData.gross_profit === 0 &&
    todayData.unconfigured_cost_count > 0 &&
    todayData.sold_today > 0;
  const grossProfitValue = allUnconfigured ? '—' : formatCOP(todayData?.gross_profit ?? 0);

  return (
    <AppLayout>
      <Topbar
        title={isMobile ? `${greeting()}, ${user.username}` : `${greeting()}, ${user.username} ${greetingEmoji()}`}
        subtitle={isMobile ? undefined : formatDate()}
        actions={
          <Link href="/pos" className="btn btn-accent btn-sm">
            {isMobile ? 'POS' : 'Abrir POS'}
          </Link>
        }
      />

      <div style={{ flex: 1, overflowY: 'auto', padding: '24px', display: 'flex', flexDirection: 'column', gap: '24px' }}>

        <BackupBanner />

        {/* ── Caja Banner ─────────────────────────────────────────────── */}
        <div
          className="caja-banner"
          style={{
            padding: '12px 16px',
            borderRadius: '8px',
            border: `1px solid ${isOpen ? 'var(--success)' : 'var(--danger)'}`,
            background: isOpen ? 'var(--success-dim)' : 'var(--danger-dim)',
            display: 'flex',
            alignItems: 'center',
            gap: '12px',
          }}
        >
          <div
            style={{
              width: '8px',
              height: '8px',
              borderRadius: '50%',
              background: isOpen ? 'var(--success)' : 'var(--danger)',
              flexShrink: 0,
            }}
          />
          <div style={{ flex: 1 }}>
            {isOpen ? (
              <span style={{ fontSize: '13px', fontWeight: 500, color: 'var(--success)' }}>
                Caja abierta — desde {formatTime(data!.register!.register.opened_at)}, monto base {formatCOP(data!.register!.register.opening_amount)}
              </span>
            ) : (
              <span style={{ fontSize: '13px', fontWeight: 500, color: 'var(--danger)' }}>
                Caja cerrada — abre la caja para comenzar a vender
              </span>
            )}
          </div>
          <Link href="/register" className="btn btn-sm" style={{
            color: isOpen ? 'var(--success)' : 'var(--danger)',
            borderColor: isOpen ? 'var(--success)' : 'var(--danger)',
            background: 'transparent',
          }}>
            Ver caja
          </Link>
        </div>

        {/* ── Stats Grid ──────────────────────────────────────────────── */}
        {loading ? (
          <div className="stats-grid" style={{ display: 'grid', gridTemplateColumns: isMobile ? 'repeat(2, 1fr)' : 'repeat(4, 1fr)', gap: '16px' }}>
            {[...Array(4)].map((_, i) => (
              <div key={i} className="stat-card" style={{ height: '80px', background: 'var(--surface-2)', borderRadius: '8px' }} />
            ))}
          </div>
        ) : (
          <div className="stats-grid" style={{ display: 'grid', gridTemplateColumns: isMobile ? 'repeat(2, 1fr)' : 'repeat(4, 1fr)', gap: '16px' }}>
            <StatCard
              label="Ventas hoy"
              value={formatCOP(data?.today.total_revenue ?? 0)}
              delta={todayDelta}
            />
            <StatCard
              label="Transacciones"
              value={String(data?.today.total_count ?? 0)}
              delta={txDelta}
            />
            <StatCard
              label="Efectivo en caja"
              value={data?.register ? formatCOP(data.register.expected) : '—'}
              delta={null}
              sub={data?.register ? 'Caja abierta' : 'Sin caja abierta'}
            />
            <StatCard
              label="Por cobrar (fiado)"
              value={formatCOP(data?.portfolio.total ?? 0)}
              delta={null}
              sub={`${data?.portfolio.count ?? 0} cliente${(data?.portfolio.count ?? 0) !== 1 ? 's' : ''}`}
              danger={(data?.portfolio.total ?? 0) > 0}
            />
          </div>
        )}

        {/* ── Cuaderno Digital ─────────────────────────────────────────── */}
        <div className="section-card" style={{ padding: '16px 20px' }}>
          <div className="section-title" style={{ marginBottom: '16px', fontSize: '14px', fontWeight: 600 }}>
            📒 Cuaderno del día
          </div>
          <div style={{ display: 'grid', gridTemplateColumns: isMobile ? 'repeat(2, 1fr)' : 'repeat(4, 1fr)', gap: '16px' }}>
            <CuadernoTile label="Vendido hoy" value={formatCOP(todayData?.sold_today ?? 0)} />
            <CuadernoTile label="Fiado nuevo hoy" value={formatCOP(todayData?.credit_new_today ?? 0)} warn={(todayData?.credit_new_today ?? 0) > 0} />
            <CuadernoTile label="Cobrado de fiado" value={formatCOP(todayData?.credit_collected_today ?? 0)} good={(todayData?.credit_collected_today ?? 0) > 0} />
            <CuadernoTile label="Efectivo en caja" value={formatCOP(todayData?.cash_estimated ?? 0)} />
            <CuadernoTile
              label="Más vendido hoy"
              value={todayData?.top_product_name ? todayData.top_product_name : '—'}
              sub={todayData?.top_product_name ? `${todayData.top_product_qty} uds` : undefined}
            />
            {/* Ganancia bruta */}
            <div style={{ display: 'flex', flexDirection: 'column', gap: '4px' }}>
              <div style={{ fontSize: '11px', color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '0.04em' }}>Ganancia bruta</div>
              <div style={{ fontSize: '20px', fontWeight: 700, color: 'var(--success)', lineHeight: 1.2 }}>
                {grossProfitValue}
              </div>
              {(todayData?.unconfigured_cost_count ?? 0) > 0 && (
                <div style={{ fontSize: '11px', color: '#d97706', display: 'flex', alignItems: 'center', gap: '4px' }}>
                  <span>⚠️</span>
                  <span>{todayData!.unconfigured_cost_count} prod. sin costo — ganancia incompleta</span>
                </div>
              )}
            </div>
            {/* Ganancia neta (US-049) */}
            <div style={{ display: 'flex', flexDirection: 'column', gap: '4px' }}>
              <div style={{ fontSize: '11px', color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '0.04em' }}>Ganancia neta</div>
              <div style={{
                fontSize: '20px', fontWeight: 700, lineHeight: 1.2,
                color: (todayData?.net_profit ?? 0) < 0 ? 'var(--danger)' : 'var(--success)',
              }}>
                {formatCOP(todayData?.net_profit ?? 0)}
              </div>
              {(todayData?.expenses_today ?? 0) > 0 && (
                <div style={{ fontSize: '11px', color: 'var(--text-muted)' }}>
                  −{formatCOP(todayData!.expenses_today)} en gastos
                </div>
              )}
            </div>
          </div>
        </div>

        {/* ── Top por margen (US-039) ────────────────────────────────── */}
        {showTopMargin && (
          <div className="section-card" style={{ padding: '16px 20px' }}>
            <div className="section-title" style={{ marginBottom: '12px', fontSize: '14px', fontWeight: 600 }}>
              📈 Top productos por margen hoy
            </div>
            <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
              {todayData!.top_by_margin.map((item, i) => (
                <div key={i} style={{ display: 'flex', alignItems: 'center', gap: '10px', fontSize: '13px' }}>
                  <span style={{ color: 'var(--text-muted)', width: '16px', textAlign: 'right', flexShrink: 0 }}>{i + 1}</span>
                  <span style={{ flex: 1, fontWeight: 500 }}>{item.name}</span>
                  <span style={{ color: 'var(--success)', fontWeight: 600 }}>{Math.round(item.margin_pct)}% margen</span>
                  <span style={{ color: 'var(--text-muted)', fontSize: '12px' }}>{item.units_sold} uds</span>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* ── Cartera pendiente (US-041) ────────────────────────────── */}
        {showAging && (
          <div className="section-card" style={{ padding: '16px 20px' }}>
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '12px' }}>
              <div className="section-title" style={{ fontSize: '14px', fontWeight: 600 }}>
                💰 Cartera pendiente
              </div>
              <Link href="/customers" className="btn btn-sm btn-ghost" style={{ fontSize: '12px' }}>
                Ver todos →
              </Link>
            </div>
            <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
              {todayData!.aging_receivables.map(r => (
                <div
                  key={r.customer_id}
                  style={{
                    display: 'flex', alignItems: 'center', gap: '10px',
                    padding: '8px 10px', borderRadius: '8px',
                    background: agingBg(r.days_overdue),
                  }}
                >
                  <span style={{ flex: 1, fontSize: '13px', fontWeight: 500 }}>{r.name}</span>
                  <span style={{ fontSize: '13px', fontWeight: 600 }}>{formatCOP(r.balance)}</span>
                  <span
                    style={{
                      fontSize: '11px', fontWeight: 600,
                      color: agingColor(r.days_overdue),
                      padding: '2px 6px', borderRadius: '4px',
                      background: 'rgba(0,0,0,0.08)',
                      minWidth: '52px', textAlign: 'center',
                    }}
                  >
                    {r.days_overdue} días
                  </span>
                  <button
                    className="btn btn-sm"
                    style={{ fontSize: '11px', padding: '4px 10px' }}
                    onClick={() => setPayTarget(r)}
                  >
                    Pagar
                  </button>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* ── Reponer pronto (US-042b) ──────────────────────────────── */}
        {showLowStock && (
          <div className="section-card" style={{ padding: '16px 20px' }}>
            <div className="section-title" style={{ marginBottom: '12px', fontSize: '14px', fontWeight: 600 }}>
              📦 Reponer pronto
            </div>
            <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
              {todayData!.low_stock.map(item => (
                <div key={item.product_id} style={{ display: 'flex', alignItems: 'center', gap: '8px', fontSize: '13px', flexWrap: 'wrap' }}>
                  <span style={{ fontWeight: 500 }}>{item.name}</span>
                  <span style={{ color: 'var(--text-muted)' }}>·</span>
                  <span style={{ color: 'var(--danger)', fontWeight: 600 }}>
                    {item.stock} unid
                  </span>
                  <span style={{ color: 'var(--text-muted)', fontSize: '12px' }}>
                    / mín {item.min_stock}
                  </span>
                  {item.days_remaining !== null && (
                    <>
                      <span style={{ color: 'var(--text-muted)' }}>·</span>
                      <span style={{ fontWeight: 700 }}>
                        {item.days_remaining} {item.days_remaining === 1 ? 'día restante' : 'días restantes'}
                      </span>
                    </>
                  )}
                </div>
              ))}
            </div>
          </div>
        )}

        {/* ── Ventas Recientes ─────────────────────────────────────────── */}
        <div className="section-card" style={{ padding: 0, overflow: 'hidden' }}>
          <div
            className="section-title"
            style={{
              padding: '12px 16px',
              display: 'flex',
              alignItems: isMobile ? 'flex-start' : 'center',
              flexDirection: isMobile ? 'column' : 'row',
              justifyContent: 'space-between',
              gap: isMobile ? '8px' : undefined,
              borderBottom: '1px solid var(--border)',
            }}
          >
            <span>Ventas recientes</span>
            <div style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>
              <button
                onClick={toggleVoided}
                className={`btn btn-sm ${showVoided ? 'btn-accent' : 'btn-ghost'}`}
              >
                {showVoided ? 'Completadas' : 'Ver anuladas'}
              </button>
              <Link href="/reports" className="btn btn-sm btn-ghost">
                Ver todas →
              </Link>
            </div>
          </div>
          <div className="table-wrap" style={{ background: 'transparent', border: 'none', borderRadius: 0 }}>
            <table>
              <thead>
                <tr>
                  <th>Hora</th>
                  <th>Cliente</th>
                  <th>Items</th>
                  <th style={{ textAlign: 'right' }}>Total</th>
                  <th>Método</th>
                  <th>Estado</th>
                </tr>
              </thead>
              <tbody>
                {displayedLoading ? (
                  <tr>
                    <td colSpan={6} style={{ textAlign: 'center', color: 'var(--text-muted)', padding: '24px' }}>
                      Cargando...
                    </td>
                  </tr>
                ) : displayedSales.length === 0 ? (
                  <tr>
                    <td colSpan={6} style={{ textAlign: 'center', color: 'var(--text-muted)', padding: '32px' }}>
                      <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: '8px' }}>
                        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" style={{ opacity: 0.4 }}>
                          <rect x="2" y="7" width="20" height="14" rx="2"/><path d="M16 21V5a2 2 0 0 0-2-2h-4a2 2 0 0 0-2 2v16"/>
                        </svg>
                        <span style={{ fontSize: '13px' }}>No hay ventas aún</span>
                      </div>
                    </td>
                  </tr>
                ) : (
                  displayedSales.map(sale => (
                    <tr key={sale.id} style={{ opacity: sale.status === 'voided' ? 0.6 : 1 }}>
                      <td style={{ fontSize: '12px', color: 'var(--text-muted)' }}>
                        {formatTime(sale.created_at)}
                      </td>
                      <td>
                        {sale.customer_name ? sale.customer_name : 'Consumidor'}
                      </td>
                      <td style={{ color: 'var(--text-muted)' }}>{sale.item_count}</td>
                      <td className="amount right">{formatCOP(sale.total)}</td>
                      <td>
                        <span className={`badge ${METHOD_BADGE[sale.payment_method] ?? ''}`}>
                          {METHOD_LABELS[sale.payment_method] ?? sale.payment_method}
                        </span>
                      </td>
                      <td>
                        <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                          {sale.status === 'voided' ? (
                            <span className="badge badge-error">Anulada</span>
                          ) : (
                            <span className="badge badge-ok">Completada</span>
                          )}
                          {!showVoided && canVoidSale(user.role, sale.status) && (
                            <button
                              type="button"
                              className="btn btn-sm btn-ghost"
                              style={{ color: 'var(--danger)' }}
                              onClick={() => setVoidTarget(sale)}
                            >
                              Anular
                            </button>
                          )}
                        </div>
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
        </div>

        {/* ── Acciones Rápidas ─────────────────────────────────────────── */}
        <div className="section-card">
          <div className="section-title" style={{ marginBottom: '12px' }}>Acciones rápidas</div>
          <div style={{ display: 'flex', gap: '10px', flexWrap: 'wrap' }}>
            <Link href="/pos" className="btn btn-accent">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <rect x="2" y="3" width="20" height="14" rx="2"/><line x1="8" y1="21" x2="16" y2="21"/><line x1="12" y1="17" x2="12" y2="21"/>
              </svg>
              Abrir POS
            </Link>
            <Link href="/customers" className="btn">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <rect x="1" y="4" width="22" height="16" rx="2" ry="2"/><line x1="1" y1="10" x2="23" y2="10"/>
              </svg>
              Ver Cartera
            </Link>
            <Link href="/register" className="btn">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <rect x="2" y="7" width="20" height="14" rx="2"/><path d="M16 21V5a2 2 0 0 0-2-2h-4a2 2 0 0 0-2 2v16"/>
              </svg>
              Ver Caja
            </Link>
          </div>
        </div>

      </div>

      {/* ── Payment Modal ──────────────────────────────────────────────── */}
      {payTarget && (
        <AbonoModal
          customerId={payTarget.customer_id}
          customerName={payTarget.name}
          balance={payTarget.balance}
          onClose={() => setPayTarget(null)}
          onSaved={payment => handlePaymentSuccess(payTarget.customer_id, payment.amount)}
        />
      )}

      {/* ── Void Sale Dialog ───────────────────────────────────────────── */}
      {voidTarget && (
        <VoidSaleDialog
          key={voidTarget.id}
          saleId={voidTarget.id}
          customerName={voidTarget.customer_name || null}
          openRegisterId={data?.register?.register?.id ?? null}
          onCancel={() => setVoidTarget(null)}
          onVoided={handleVoided}
          onFailed={handleVoidFailed}
        />
      )}
    </AppLayout>
  );
}
