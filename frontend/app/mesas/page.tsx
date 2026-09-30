'use client';

import { useState, useEffect, useCallback } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import AppLayout from '@/components/AppLayout';
import Topbar from '@/components/Topbar';
import { api } from '@/lib/api';
import { useToast } from '@/components/Toast';
import { formatCOP } from '@/lib/format';
import { TABLE_STATE_COLOR, TABLE_STATE_LABEL, type TableState } from '@/lib/orders';

interface TableWithState {
  id: number;
  name: string;
  zone: string;
  seats: number;
  state: TableState;
  open_order_id: number | null;
  open_total: number;
  opened_minutes: number;
}

export default function MesasPage() {
  const router = useRouter();
  const { toast } = useToast();
  const [tables, setTables] = useState<TableWithState[]>([]);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    try {
      const data = await api.get<TableWithState[]>('/tables');
      setTables(data ?? []);
      return data ?? [];
    } catch (e) {
      toast(e instanceof Error ? e.message : 'Error al cargar las mesas', 'error');
      return [];
    } finally {
      setLoading(false);
    }
  }, [toast]);

  useEffect(() => {
    api.get<TableWithState[]>('/tables')
      .then(d => setTables(d ?? []))
      .catch(e => toast(e instanceof Error ? e.message : 'Error al cargar las mesas', 'error'))
      .finally(() => setLoading(false));
  }, [toast]);

  async function openTable(t: TableWithState) {
    if (t.open_order_id) {
      router.push(`/mesas/${t.open_order_id}`);
      return;
    }
    try {
      const order = await api.post<{ id: number }>('/orders', { table_id: t.id });
      router.push(`/mesas/${order.id}`);
    } catch (e) {
      // Another tablet may have opened it first: refresh and follow the existing order.
      const fresh = await load();
      const same = fresh.find(x => x.id === t.id);
      if (same?.open_order_id) {
        router.push(`/mesas/${same.open_order_id}`);
      } else {
        toast(e instanceof Error ? e.message : 'No se pudo abrir la mesa', 'error');
      }
    }
  }

  return (
    <AppLayout>
      <div style={{ display: 'flex', flexDirection: 'column', height: '100%', overflow: 'hidden' }}>
        <Topbar title="Mesas" subtitle="Toca una mesa para abrir o ver su pedido" />
        <div style={{ flex: 1, overflowY: 'auto', padding: '16px' }}>
          {loading ? (
            <div style={{ color: 'var(--text-muted)', fontSize: '14px' }}>Cargando...</div>
          ) : tables.length === 0 ? (
            <div style={{ color: 'var(--text-secondary)', fontSize: '14px', textAlign: 'center', padding: '40px 0' }}>
              No hay mesas.{' '}
              <Link href="/settings" style={{ color: 'var(--accent)' }}>Crea mesas en Ajustes</Link>
            </div>
          ) : (
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(min(100%, 160px), 1fr))', gap: '12px' }}>
              {tables.map(t => (
                <button
                  key={t.id}
                  onClick={() => openTable(t)}
                  style={{
                    background: 'var(--surface-1)',
                    border: `2px solid ${TABLE_STATE_COLOR[t.state]}`,
                    borderRadius: '8px', padding: '14px', textAlign: 'left',
                    cursor: 'pointer', display: 'flex', flexDirection: 'column', gap: '4px',
                  }}
                >
                  <span style={{ fontSize: '15px', fontWeight: 700, color: 'var(--text-primary)' }}>{t.name}</span>
                  <span style={{ fontSize: '12px', fontWeight: 600, color: TABLE_STATE_COLOR[t.state] }}>
                    {TABLE_STATE_LABEL[t.state]}
                  </span>
                  {t.state !== 'libre' && (
                    <>
                      <span style={{ fontSize: '16px', fontWeight: 700, color: 'var(--text-primary)', fontVariantNumeric: 'tabular-nums' }}>
                        {formatCOP(t.open_total)}
                      </span>
                      <span style={{ fontSize: '12px', color: 'var(--text-muted)' }}>{t.opened_minutes} min</span>
                    </>
                  )}
                </button>
              ))}
            </div>
          )}
        </div>
      </div>
    </AppLayout>
  );
}
