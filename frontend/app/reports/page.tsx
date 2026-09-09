'use client';

import { useState, useEffect, useCallback } from 'react';
import AppLayout from '@/components/AppLayout';
import Topbar from '@/components/Topbar';
import { api } from '@/lib/api';
import Link from 'next/link';
import { formatCOP } from '@/lib/format';

// ─── Types ────────────────────────────────────────────────────────────────────

interface PaymentMethodTotal {
  method: string;
  total: number;
  count: number;
}

interface SalesPeriod {
  period: string;
  total: number;
  count: number;
}

interface SalesSummary {
  total_revenue: number;
  total_cost: number;
  gross_margin: number;
  total_count: number;
  by_payment_method: PaymentMethodTotal[];
  grouped: SalesPeriod[];
}

interface TopProduct {
  product_id: number;
  name: string;
  quantity: number;
  revenue: number;
}

interface Register {
  id: number;
  user_id: number;
  opening_amount: number;
  expected_close: number | null;
  actual_close: number | null;
  difference: number | null;
  status: string;
  opened_at: string;
  closed_at: string | null;
}

interface PortfolioEntry {
  id: number;
  name: string;
  balance: number;
}

interface Expense {
  id: number;
  register_id: number | null;
  user_id: number;
  username: string;
  category: string;
  amount: number;
  note: string;
  created_at: string;
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

const formatDate = (iso: string) =>
  new Date(iso).toLocaleDateString('es-CO', { day: '2-digit', month: 'short', year: 'numeric' });

const METHOD_LABELS: Record<string, string> = {
  cash: 'Efectivo',
  card: 'Tarjeta',
  transfer: 'Transferencia',
  credit: 'Crédito',
};

const EXPENSE_CATEGORIES = [
  { value: 'servicios', label: 'Servicios' },
  { value: 'arriendo', label: 'Arriendo' },
  { value: 'nomina', label: 'Nómina' },
  { value: 'proveedor', label: 'Proveedor' },
  { value: 'otro', label: 'Otro' },
];

const CATEGORY_LABELS: Record<string, string> = Object.fromEntries(
  EXPENSE_CATEGORIES.map(c => [c.value, c.label])
);

type RangeKey = 'today' | 'week' | 'month' | 'custom';

function getRangeDates(range: RangeKey, customFrom: string, customTo: string): { from: Date; to: Date } {
  const now = new Date();
  const startOfDay = (d: Date) => new Date(d.getFullYear(), d.getMonth(), d.getDate(), 0, 0, 0);
  const endOfDay = (d: Date) => new Date(d.getFullYear(), d.getMonth(), d.getDate(), 23, 59, 59);

  switch (range) {
    case 'today':
      return { from: startOfDay(now), to: endOfDay(now) };
    case 'week': {
      const day = now.getDay(); // 0=Sun
      const diff = (day + 6) % 7; // Mon=0
      const mon = new Date(now);
      mon.setDate(now.getDate() - diff);
      return { from: startOfDay(mon), to: endOfDay(now) };
    }
    case 'month':
      return { from: new Date(now.getFullYear(), now.getMonth(), 1), to: endOfDay(now) };
    case 'custom': {
      const f = customFrom ? new Date(customFrom) : startOfDay(now);
      const t = customTo ? endOfDay(new Date(customTo)) : endOfDay(now);
      return { from: f, to: t };
    }
  }
}

function rangeGroupBy(range: RangeKey): string {
  if (range === 'today') return 'day';
  if (range === 'week') return 'day';
  if (range === 'month') return 'day';
  return 'day';
}

function rangeLabel(range: RangeKey, customFrom: string, customTo: string): string {
  if (range === 'today') return 'Hoy';
  if (range === 'week') return 'Esta semana';
  if (range === 'month') return 'Este mes';
  if (customFrom && customTo) return `${customFrom} — ${customTo}`;
  return 'Rango personalizado';
}

// ─── Icons ───────────────────────────────────────────────────────────────────

const IconTrend = () => (
  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
    <polyline points="23 6 13.5 15.5 8.5 10.5 1 18"/><polyline points="17 6 23 6 23 12"/>
  </svg>
);
const IconPackage = () => (
  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
    <path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/>
  </svg>
);
const IconCreditCard = () => (
  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
    <rect x="1" y="4" width="22" height="16" rx="2" ry="2"/><line x1="1" y1="10" x2="23" y2="10"/>
  </svg>
);
const IconCash = () => (
  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
    <rect x="2" y="7" width="20" height="14" rx="2"/><path d="M16 21V5a2 2 0 0 0-2-2h-4a2 2 0 0 0-2 2v16"/>
  </svg>
);
const IconArrow = () => (
  <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
    <line x1="5" y1="12" x2="19" y2="12"/><polyline points="12 5 19 12 12 19"/>
  </svg>
);

// ─── Bar Chart ────────────────────────────────────────────────────────────────

function BarChart({ data }: { data: SalesPeriod[] }) {
  if (!data || data.length === 0) {
    return (
      <div style={{ height: '120px', display: 'flex', alignItems: 'center', justifyContent: 'center', color: 'var(--text-muted)', fontSize: '12px' }}>
        Sin datos para el período
      </div>
    );
  }
  const max = Math.max(...data.map(d => d.total), 1);

  return (
    <div style={{ height: '120px', display: 'flex', alignItems: 'flex-end', gap: '3px', padding: '4px 0' }}>
      {data.map((d, i) => {
        const pct = (d.total / max) * 100;
        const isLast = i === data.length - 1;
        return (
          <div
            key={d.period}
            title={`${d.period}: ${formatCOP(d.total)} (${d.count} ventas)`}
            style={{
              flex: 1,
              minWidth: '4px',
              height: `${Math.max(pct, 4)}%`,
              background: isLast ? 'var(--accent)' : 'var(--accent-dim, rgba(209,140,25,0.3))',
              borderRadius: '3px 3px 0 0',
              borderTop: `2px solid ${isLast ? 'var(--accent)' : 'rgba(209,140,25,0.5)'}`,
              cursor: 'default',
              transition: 'opacity 150ms',
            }}
            onMouseEnter={e => (e.currentTarget.style.opacity = '0.8')}
            onMouseLeave={e => (e.currentTarget.style.opacity = '1')}
          />
        );
      })}
    </div>
  );
}

// ─── Report Card ──────────────────────────────────────────────────────────────

function ReportCard({ title, icon, children }: { title: string; icon: React.ReactNode; children: React.ReactNode }) {
  return (
    <div style={{
      background: 'var(--surface-1)',
      border: '1px solid var(--border)',
      borderRadius: '12px',
      padding: '20px',
      display: 'flex',
      flexDirection: 'column',
      gap: '16px',
    }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
        <span style={{ color: 'var(--accent)', display: 'flex' }}>{icon}</span>
        <span style={{ fontSize: '13px', fontWeight: 600, color: 'var(--text-primary)' }}>{title}</span>
      </div>
      {children}
    </div>
  );
}

// ─── CSV Export ───────────────────────────────────────────────────────────────

function exportSalesCSV(sales: SalesSummary | null) {
  if (!sales) return;

  const BOM = '\uFEFF';
  const headers = ['fecha', 'total_revenue', 'total_cost', 'gross_margin', 'total_count'];
  const rows = (sales.grouped ?? []).map(row =>
    [row.period, row.total, '', '', row.count].join(',')
  );
  // totals row
  rows.push(['TOTAL', sales.total_revenue, sales.total_cost, sales.gross_margin, sales.total_count].join(','));

  const csv = BOM + [headers.join(','), ...rows].join('\r\n');
  const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' });
  const url = URL.createObjectURL(blob);
  const today = new Date().toISOString().slice(0, 10);
  const a = document.createElement('a');
  a.href = url;
  a.download = `reporte-ventas-${today}.csv`;
  a.click();
  URL.revokeObjectURL(url);
}

const IconExpense = () => (
  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
    <circle cx="12" cy="12" r="10"/><line x1="8" y1="12" x2="16" y2="12"/>
  </svg>
);

const IconDownload = () => (
  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
    <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/>
  </svg>
);

// ─── Main Page ────────────────────────────────────────────────────────────────

export default function ReportsPage() {
  const [range, setRange] = useState<RangeKey>('month');
  const [customFrom, setCustomFrom] = useState('');
  const [customTo, setCustomTo] = useState('');

  const [sales, setSales] = useState<SalesSummary | null>(null);
  const [topProducts, setTopProducts] = useState<TopProduct[]>([]);
  const [registers, setRegisters] = useState<Register[]>([]);
  const [portfolio, setPortfolio] = useState<PortfolioEntry[]>([]);
  const [loading, setLoading] = useState(true);

  const [expenses, setExpenses] = useState<Expense[]>([]);
  const [expensesTotal, setExpensesTotal] = useState(0);
  const [expensesLoading, setExpensesLoading] = useState(true);
  const [expenseCategory, setExpenseCategory] = useState('');

  const load = useCallback(async () => {
    setLoading(true);
    const { from, to } = getRangeDates(range, customFrom, customTo);
    const fromISO = from.toISOString();
    const toISO = to.toISOString();
    const groupBy = rangeGroupBy(range);

    const [salesRes, topRes, regsRes, portRes] = await Promise.allSettled([
      api.get<SalesSummary>(`/reports/sales?from=${encodeURIComponent(fromISO)}&to=${encodeURIComponent(toISO)}&group_by=${groupBy}`),
      api.get<TopProduct[]>(`/reports/top-products?from=${encodeURIComponent(fromISO)}&to=${encodeURIComponent(toISO)}&limit=8`),
      api.get<Register[]>('/reports/registers'),
      api.get<PortfolioEntry[]>('/reports/portfolio'),
    ]);

    if (salesRes.status === 'fulfilled') setSales(salesRes.value);
    if (topRes.status === 'fulfilled') setTopProducts(topRes.value ?? []);
    if (regsRes.status === 'fulfilled') setRegisters(regsRes.value ?? []);
    if (portRes.status === 'fulfilled') setPortfolio(portRes.value ?? []);

    setLoading(false);
  }, [range, customFrom, customTo]);

  const loadExpenses = useCallback(async () => {
    setExpensesLoading(true);
    const { from, to } = getRangeDates(range, customFrom, customTo);
    const params = new URLSearchParams();
    params.set('from', from.toISOString());
    params.set('to', to.toISOString());
    if (expenseCategory) params.set('category', expenseCategory);
    try {
      const data = await api.get<{ expenses: Expense[]; total: number }>(`/expenses?${params.toString()}`);
      setExpenses(data?.expenses ?? []);
      setExpensesTotal(data?.total ?? 0);
    } catch {
      setExpenses([]);
      setExpensesTotal(0);
    } finally {
      setExpensesLoading(false);
    }
  }, [range, customFrom, customTo, expenseCategory]);

  useEffect(() => { load(); }, [load]);
  useEffect(() => { loadExpenses(); }, [loadExpenses]);

  const totalDebt = portfolio.reduce((s, e) => s + e.balance, 0);

  return (
    <AppLayout>
      <Topbar
        title="Reportes"
        subtitle={rangeLabel(range, customFrom, customTo)}
        actions={
          <div style={{ display: 'flex', gap: '6px', alignItems: 'center', flexWrap: 'wrap' }}>
            <button
              onClick={() => exportSalesCSV(sales)}
              disabled={loading || !sales}
              className="btn btn-sm"
              style={{ display: 'flex', alignItems: 'center', gap: '5px', color: 'var(--text-secondary)' }}
            >
              <IconDownload /> Exportar CSV
            </button>
            {(['today', 'week', 'month', 'custom'] as RangeKey[]).map(r => (
              <button
                key={r}
                onClick={() => setRange(r)}
                className={`btn btn-sm${range === r ? ' btn-accent' : ''}`}
                style={range !== r ? { color: 'var(--text-secondary)' } : undefined}
              >
                {{ today: 'Hoy', week: 'Semana', month: 'Mes', custom: 'Personalizado' }[r]}
              </button>
            ))}
            {range === 'custom' && (
              <>
                <input
                  type="date"
                  className="input"
                  value={customFrom}
                  onChange={e => setCustomFrom(e.target.value)}
                  style={{ width: '140px', fontSize: '12px', padding: '4px 8px' }}
                />
                <span style={{ color: 'var(--text-muted)', fontSize: '12px' }}>—</span>
                <input
                  type="date"
                  className="input"
                  value={customTo}
                  onChange={e => setCustomTo(e.target.value)}
                  style={{ width: '140px', fontSize: '12px', padding: '4px 8px' }}
                />
              </>
            )}
          </div>
        }
      />

      <div style={{
        flex: 1, overflow: 'auto', padding: '20px 24px',
        display: 'grid',
        gridTemplateColumns: '1fr 1fr',
        gap: '16px',
        alignContent: 'start',
      }}>

        {/* ── Card 1: Ventas del período ───────────────────────────────────── */}
        <ReportCard title="Ventas del período" icon={<IconTrend />}>
          {loading ? (
            <div style={{ color: 'var(--text-muted)', fontSize: '13px' }}>Cargando...</div>
          ) : (
            <>
              <BarChart data={sales?.grouped ?? []} />

              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: '10px' }}>
                <MetricItem label="Ingresos" value={formatCOP(sales?.total_revenue ?? 0)} />
                <MetricItem label="Costo" value={formatCOP(sales?.total_cost ?? 0)} />
                <MetricItem
                  label="Margen bruto"
                  value={formatCOP(sales?.gross_margin ?? 0)}
                  valueColor={(sales?.gross_margin ?? 0) >= 0 ? 'var(--success)' : 'var(--danger)'}
                />
              </div>

              <div style={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
                <div style={{ fontSize: '11px', fontWeight: 600, textTransform: 'uppercase', letterSpacing: '0.4px', color: 'var(--text-muted)' }}>
                  Por método de pago · {sales?.total_count ?? 0} ventas
                </div>
                {(sales?.by_payment_method ?? []).length === 0 ? (
                  <div style={{ fontSize: '12px', color: 'var(--text-muted)' }}>Sin ventas en este período</div>
                ) : (
                  sales?.by_payment_method.map(pm => (
                    <div key={pm.method} style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                      <span style={{ fontSize: '12px', color: 'var(--text-secondary)', minWidth: '90px' }}>
                        {METHOD_LABELS[pm.method] ?? pm.method}
                      </span>
                      <div style={{ flex: 1, height: '4px', background: 'var(--surface-3)', borderRadius: '2px', overflow: 'hidden' }}>
                        <div style={{
                          height: '100%',
                          width: `${sales.total_revenue > 0 ? (pm.total / sales.total_revenue) * 100 : 0}%`,
                          background: 'var(--accent)',
                          borderRadius: '2px',
                        }} />
                      </div>
                      <span className="tnum" style={{ fontSize: '12px', color: 'var(--text-primary)', fontWeight: 600, minWidth: '80px', textAlign: 'right' }}>
                        {formatCOP(pm.total)}
                      </span>
                      <span style={{ fontSize: '11px', color: 'var(--text-muted)', minWidth: '40px', textAlign: 'right' }}>
                        {pm.count}x
                      </span>
                    </div>
                  ))
                )}
              </div>
            </>
          )}
        </ReportCard>

        {/* ── Card 2: Top Productos ────────────────────────────────────────── */}
        <ReportCard title="Top Productos" icon={<IconPackage />}>
          {loading ? (
            <div style={{ color: 'var(--text-muted)', fontSize: '13px' }}>Cargando...</div>
          ) : topProducts.length === 0 ? (
            <div style={{ color: 'var(--text-muted)', fontSize: '13px' }}>Sin ventas en este período</div>
          ) : (
            <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
              {topProducts.map((p, i) => (
                <div key={p.product_id} style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
                  <span style={{
                    width: '20px', height: '20px', borderRadius: '50%', flexShrink: 0,
                    background: i === 0 ? 'var(--accent)' : 'var(--surface-3)',
                    color: i === 0 ? '#000' : 'var(--text-muted)',
                    fontSize: '10px', fontWeight: 700,
                    display: 'flex', alignItems: 'center', justifyContent: 'center',
                  }}>{i + 1}</span>
                  <span style={{ flex: 1, fontSize: '13px', color: 'var(--text-primary)', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                    {p.name}
                  </span>
                  <span style={{ fontSize: '11px', color: 'var(--text-muted)', minWidth: '40px', textAlign: 'right' }}>
                    {p.quantity % 1 === 0 ? p.quantity : p.quantity.toFixed(1)}x
                  </span>
                  <span className="tnum" style={{ fontSize: '13px', fontWeight: 600, color: 'var(--text-primary)', minWidth: '80px', textAlign: 'right' }}>
                    {formatCOP(p.revenue)}
                  </span>
                </div>
              ))}
            </div>
          )}
        </ReportCard>

        {/* ── Card 3: Cartera ──────────────────────────────────────────────── */}
        <ReportCard title="Cartera de crédito" icon={<IconCreditCard />}>
          {loading ? (
            <div style={{ color: 'var(--text-muted)', fontSize: '13px' }}>Cargando...</div>
          ) : (
            <>
              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '10px' }}>
                <MetricItem
                  label="Deuda total"
                  value={formatCOP(totalDebt)}
                  valueColor={totalDebt > 0 ? 'var(--warning)' : undefined}
                />
                <MetricItem label="Clientes con deuda" value={String(portfolio.length)} />
              </div>

              {portfolio.length === 0 ? (
                <div style={{ fontSize: '12px', color: 'var(--text-muted)' }}>No hay deuda pendiente</div>
              ) : (
                <div style={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
                  {portfolio.slice(0, 5).map(e => (
                    <div key={e.id} style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                      <span style={{ flex: 1, fontSize: '12px', color: 'var(--text-secondary)', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                        {e.name}
                      </span>
                      <span className="tnum" style={{ fontSize: '13px', fontWeight: 600, color: 'var(--warning)' }}>
                        {formatCOP(e.balance)}
                      </span>
                    </div>
                  ))}
                  {portfolio.length > 5 && (
                    <div style={{ fontSize: '11px', color: 'var(--text-muted)' }}>
                      +{portfolio.length - 5} más
                    </div>
                  )}
                </div>
              )}

              <Link href="/portfolio" style={{ display: 'flex', alignItems: 'center', gap: '4px', fontSize: '12px', color: 'var(--accent)', textDecoration: 'none', fontWeight: 600 }}>
                Ver cartera completa <IconArrow />
              </Link>
            </>
          )}
        </ReportCard>

        {/* ── Card: Gastos del negocio (US-049) ────────────────────────────── */}
        <div style={{ gridColumn: '1 / -1' }}>
          <ExpensesCard
            expenses={expenses}
            total={expensesTotal}
            loading={expensesLoading}
            category={expenseCategory}
            onCategoryChange={setExpenseCategory}
            onCreated={loadExpenses}
          />
        </div>

        {/* ── Card 4: Historial de Cajas ───────────────────────────────────── */}
        <ReportCard title="Historial de cajas" icon={<IconCash />}>
          {loading ? (
            <div style={{ color: 'var(--text-muted)', fontSize: '13px' }}>Cargando...</div>
          ) : registers.length === 0 ? (
            <div style={{ color: 'var(--text-muted)', fontSize: '13px' }}>No hay cajas registradas</div>
          ) : (
            <div style={{ display: 'flex', flexDirection: 'column', gap: '1px', overflow: 'hidden', borderRadius: '8px', border: '1px solid var(--border)' }}>
              {registers.slice(0, 8).map(reg => {
                const diff = reg.difference ?? 0;
                return (
                  <div key={reg.id} style={{
                    display: 'grid',
                    gridTemplateColumns: '1fr auto auto',
                    gap: '12px',
                    alignItems: 'center',
                    padding: '8px 12px',
                    background: 'var(--surface-2)',
                    borderBottom: '1px solid var(--border)',
                  }}>
                    <div>
                      <div style={{ fontSize: '12px', color: 'var(--text-primary)', fontWeight: 600 }}>
                        {formatDate(reg.opened_at)}
                      </div>
                      <div style={{ fontSize: '11px', color: 'var(--text-muted)' }}>
                        {reg.closed_at ? `Cerrada ${formatDate(reg.closed_at)}` : 'Abierta'}
                      </div>
                    </div>
                    <div style={{ textAlign: 'right' }}>
                      <div style={{ fontSize: '11px', color: 'var(--text-muted)' }}>Real</div>
                      <div className="tnum" style={{ fontSize: '12px', fontWeight: 600, color: 'var(--text-primary)' }}>
                        {reg.actual_close != null ? formatCOP(reg.actual_close) : '—'}
                      </div>
                    </div>
                    <div style={{ textAlign: 'right', minWidth: '60px' }}>
                      <div style={{ fontSize: '11px', color: 'var(--text-muted)' }}>Dif.</div>
                      <div className="tnum" style={{
                        fontSize: '12px', fontWeight: 600,
                        color: reg.difference == null ? 'var(--text-muted)' : diff >= 0 ? 'var(--success)' : 'var(--danger)',
                      }}>
                        {reg.difference == null ? '—' : (diff >= 0 ? '+' : '') + formatCOP(diff)}
                      </div>
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </ReportCard>

      </div>
    </AppLayout>
  );
}

// ─── Expenses Card (US-049) ────────────────────────────────────────────────────

function formatDateTime(iso: string): string {
  return new Date(iso).toLocaleString('es-CO', {
    day: '2-digit', month: 'short', hour: '2-digit', minute: '2-digit',
  });
}

function ExpensesCard({
  expenses,
  total,
  loading,
  category,
  onCategoryChange,
  onCreated,
}: {
  expenses: Expense[];
  total: number;
  loading: boolean;
  category: string;
  onCategoryChange: (v: string) => void;
  onCreated: () => void;
}) {
  const [formCategory, setFormCategory] = useState('servicios');
  const [amount, setAmount] = useState('');
  const [note, setNote] = useState('');
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');
  const [confirmation, setConfirmation] = useState('');

  async function handleSubmit() {
    const parsed = parseFloat(amount.replace(/\./g, '').replace(',', '.'));
    if (!parsed || parsed <= 0) {
      setError('Ingresa un monto válido');
      return;
    }
    setSaving(true);
    setError('');
    setConfirmation('');
    try {
      await api.post('/expenses', { category: formCategory, amount: parsed, note });
      setConfirmation(`Gasto registrado: ${formatCOP(parsed)} — ${CATEGORY_LABELS[formCategory]}`);
      setAmount('');
      setNote('');
      onCreated();
    } catch (err: unknown) {
      const e = err as { message?: string };
      setError(e.message ?? 'Error al registrar el gasto');
    } finally {
      setSaving(false);
    }
  }

  const inputStyle: React.CSSProperties = {
    padding: '7px 10px',
    borderRadius: '8px',
    border: '1px solid var(--border)',
    background: 'var(--surface-1)',
    color: 'var(--text-primary)',
    fontSize: '13px',
  };

  return (
    <ReportCard title="Gastos del negocio" icon={<IconExpense />}>
      {/* Formulario */}
      <div style={{ display: 'flex', flexWrap: 'wrap', gap: '8px', alignItems: 'flex-end' }}>
        <div style={{ display: 'flex', flexDirection: 'column', gap: '4px' }}>
          <label style={{ fontSize: '11px', color: 'var(--text-muted)' }}>Categoría</label>
          <select value={formCategory} onChange={e => setFormCategory(e.target.value)} style={{ ...inputStyle, minWidth: '130px' }}>
            {EXPENSE_CATEGORIES.map(c => (
              <option key={c.value} value={c.value}>{c.label}</option>
            ))}
          </select>
        </div>
        <div style={{ display: 'flex', flexDirection: 'column', gap: '4px' }}>
          <label style={{ fontSize: '11px', color: 'var(--text-muted)' }}>Monto *</label>
          <input
            type="number"
            className="input"
            placeholder="0"
            value={amount}
            onChange={e => { setAmount(e.target.value); setError(''); }}
            style={{ ...inputStyle, width: '120px' }}
          />
        </div>
        <div style={{ display: 'flex', flexDirection: 'column', gap: '4px', flex: 1, minWidth: '140px' }}>
          <label style={{ fontSize: '11px', color: 'var(--text-muted)' }}>Nota (opcional)</label>
          <input
            type="text"
            className="input"
            placeholder="Ej: recibo de luz"
            value={note}
            onChange={e => setNote(e.target.value)}
            style={inputStyle}
          />
        </div>
        <button onClick={handleSubmit} disabled={saving} className="btn btn-accent btn-sm">
          {saving ? 'Guardando...' : 'Registrar gasto'}
        </button>
      </div>

      {error && (
        <div style={{ fontSize: '12px', color: 'var(--danger)', padding: '8px', background: 'var(--danger-dim)', borderRadius: '6px' }}>
          {error}
        </div>
      )}
      {confirmation && !error && (
        <div style={{ fontSize: '12px', color: 'var(--success)', padding: '8px', background: 'var(--success-dim)', borderRadius: '6px' }}>
          {confirmation}
        </div>
      )}

      {/* Filtro + total */}
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: '8px', flexWrap: 'wrap' }}>
        <select value={category} onChange={e => onCategoryChange(e.target.value)} style={inputStyle}>
          <option value="">Todas las categorías</option>
          {EXPENSE_CATEGORIES.map(c => (
            <option key={c.value} value={c.value}>{c.label}</option>
          ))}
        </select>
        <div style={{ fontSize: '13px', color: 'var(--text-secondary)' }}>
          Total del período: <span className="tnum" style={{ fontWeight: 700, color: 'var(--text-primary)' }}>{formatCOP(total)}</span>
        </div>
      </div>

      {/* Listado */}
      {loading ? (
        <div style={{ color: 'var(--text-muted)', fontSize: '13px' }}>Cargando...</div>
      ) : expenses.length === 0 ? (
        <div style={{ color: 'var(--text-muted)', fontSize: '13px' }}>Sin gastos registrados en este período</div>
      ) : (
        <div className="table-wrap" style={{ overflowX: 'auto' }}>
          <table style={{ width: '100%', borderCollapse: 'collapse' }}>
            <thead>
              <tr>
                <th style={thStyleExpense}>Fecha</th>
                <th style={thStyleExpense}>Categoría</th>
                <th style={thStyleExpense}>Usuario</th>
                <th style={thStyleExpense}>Nota</th>
                <th style={{ ...thStyleExpense, textAlign: 'right' }}>Monto</th>
              </tr>
            </thead>
            <tbody>
              {expenses.map((e, i) => (
                <tr key={e.id} style={{ borderBottom: i < expenses.length - 1 ? '1px solid var(--border)' : 'none' }}>
                  <td style={{ ...tdStyleExpense, whiteSpace: 'nowrap', color: 'var(--text-muted)' }}>{formatDateTime(e.created_at)}</td>
                  <td style={tdStyleExpense}><span className="badge">{CATEGORY_LABELS[e.category] ?? e.category}</span></td>
                  <td style={{ ...tdStyleExpense, fontWeight: 600 }}>{e.username}</td>
                  <td style={{ ...tdStyleExpense, color: 'var(--text-secondary)' }}>{e.note || '—'}</td>
                  <td className="tnum" style={{ ...tdStyleExpense, textAlign: 'right', fontWeight: 600 }}>{formatCOP(e.amount)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </ReportCard>
  );
}

const thStyleExpense: React.CSSProperties = {
  padding: '8px 12px',
  textAlign: 'left',
  fontSize: '11px',
  fontWeight: 600,
  textTransform: 'uppercase',
  letterSpacing: '0.4px',
  color: 'var(--text-muted)',
  borderBottom: '1px solid var(--border)',
  whiteSpace: 'nowrap',
};

const tdStyleExpense: React.CSSProperties = {
  padding: '10px 12px',
  fontSize: '13px',
};

// ─── Small helpers ────────────────────────────────────────────────────────────

function MetricItem({ label, value, valueColor }: { label: string; value: string; valueColor?: string }) {
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '2px' }}>
      <div style={{ fontSize: '11px', color: 'var(--text-muted)', fontWeight: 600, textTransform: 'uppercase', letterSpacing: '0.4px' }}>{label}</div>
      <div className="tnum" style={{ fontSize: '16px', fontWeight: 700, color: valueColor ?? 'var(--text-primary)' }}>{value}</div>
    </div>
  );
}
