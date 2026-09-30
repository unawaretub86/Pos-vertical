export type ItemStatus = 'pending' | 'preparing' | 'ready' | 'delivered' | 'void';
export type TableState = 'libre' | 'ocupada' | 'por_cobrar';

export const ITEM_STATUS_LABEL: Record<ItemStatus, string> = {
  pending: 'Pendiente',
  preparing: 'Preparando',
  ready: 'Listo',
  delivered: 'Entregado',
  void: 'Anulado',
};

export const TABLE_STATE_LABEL: Record<TableState, string> = {
  libre: 'Libre',
  ocupada: 'Ocupada',
  por_cobrar: 'Por cobrar',
};

// Colors are CSS variables so the dark theme keeps working.
export const TABLE_STATE_COLOR: Record<TableState, string> = {
  libre: 'var(--text-muted)',
  ocupada: 'var(--accent)',
  por_cobrar: 'var(--text-primary)',
};

export interface OrderItem {
  id: number;
  product_id: number;
  product_name: string;
  quantity: number;
  unit_price: number;
  subtotal: number;
  note: string;
  status: ItemStatus;
}

export interface Order {
  id: number;
  type: string;
  table_id: number | null;
  status: 'open' | 'closed' | 'cancelled';
  total: number;
  items: OrderItem[];
}
