'use client';

import { useState, useEffect, useCallback, useRef } from 'react';
import { useRouter } from 'next/navigation';
import { useIsMobile } from '@/hooks/useIsMobile';
import AppLayout from '@/components/AppLayout';
import Topbar from '@/components/Topbar';
import { api } from '@/lib/api';
import { useToast } from '@/components/Toast';
import ConfirmDialog from '@/components/ConfirmDialog';
import LoadingSpinner from '@/components/LoadingSpinner';
import EmptyState from '@/components/EmptyState';

// ─── Types ───────────────────────────────────────────────────────────────────

interface Supplier {
  id: number;
  name: string;
  phone: string;
  notes: string;
  created_at: string;
}

// ─── Icons ───────────────────────────────────────────────────────────────────

const IconPlus = () => (
  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
    <line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/>
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
const IconTruck = () => (
  <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
    <rect x="1" y="3" width="15" height="13"/><polygon points="16 8 20 8 23 11 23 16 16 16 16 8"/>
    <circle cx="5.5" cy="18.5" r="2.5"/><circle cx="18.5" cy="18.5" r="2.5"/>
  </svg>
);

// ─── Supplier Modal ───────────────────────────────────────────────────────────

function SupplierModal({
  initial,
  onClose,
  onSaved,
}: {
  initial?: Supplier | null;
  onClose: () => void;
  onSaved: () => void;
}) {
  const { toast } = useToast();
  const [name, setName] = useState(initial?.name ?? '');
  const [phone, setPhone] = useState(initial?.phone ?? '');
  const [notes, setNotes] = useState(initial?.notes ?? '');
  const [loading, setLoading] = useState(false);
  const nameRef = useRef<HTMLInputElement>(null);

  useEffect(() => { nameRef.current?.focus(); }, []);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!name.trim()) return;
    setLoading(true);
    try {
      if (initial) {
        await api.put(`/suppliers/${initial.id}`, { name, phone, notes });
        toast('Proveedor actualizado', 'success');
      } else {
        await api.post('/suppliers', { name, phone, notes });
        toast('Proveedor creado', 'success');
      }
      onSaved();
      onClose();
    } catch (err: unknown) {
      toast((err as Error).message, 'error');
    } finally {
      setLoading(false);
    }
  }

  return (
    <div
      style={{ position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.6)', display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 50, padding: 12 }}
      onClick={onClose}
    >
      <div
        style={{ background: 'var(--surface-1)', border: '1px solid var(--border-emphasis)', borderRadius: 12, padding: 24, width: 'min(400px, 100%)' }}
        onClick={e => e.stopPropagation()}
      >
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 20 }}>
          <h2 style={{ fontSize: 15, fontWeight: 600, color: 'var(--text-primary)', margin: 0 }}>
            {initial ? 'Editar proveedor' : 'Nuevo proveedor'}
          </h2>
          <button onClick={onClose} style={{ background: 'none', border: 'none', cursor: 'pointer', color: 'var(--text-muted)', padding: 2 }}>
            <IconX />
          </button>
        </div>

        <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
          <div>
            <label style={{ fontSize: 12, fontWeight: 600, color: 'var(--text-secondary)', textTransform: 'uppercase', letterSpacing: '0.4px', display: 'block', marginBottom: 6 }}>
              Nombre *
            </label>
            <input ref={nameRef} value={name} onChange={e => setName(e.target.value)} placeholder="Ej: Lácteos El Establo" required style={{ width: '100%', boxSizing: 'border-box' }} />
          </div>
          <div>
            <label style={{ fontSize: 12, fontWeight: 600, color: 'var(--text-secondary)', textTransform: 'uppercase', letterSpacing: '0.4px', display: 'block', marginBottom: 6 }}>
              Teléfono
            </label>
            <input value={phone} onChange={e => setPhone(e.target.value)} placeholder="Ej: 3001234567" style={{ width: '100%', boxSizing: 'border-box' }} />
          </div>
          <div>
            <label style={{ fontSize: 12, fontWeight: 600, color: 'var(--text-secondary)', textTransform: 'uppercase', letterSpacing: '0.4px', display: 'block', marginBottom: 6 }}>
              Notas
            </label>
            <textarea value={notes} onChange={e => setNotes(e.target.value)} rows={3} maxLength={500}
              style={{ width: '100%', boxSizing: 'border-box', resize: 'vertical', fontFamily: 'inherit', fontSize: 13, padding: '8px 12px', borderRadius: 6, border: '1px solid var(--border)', background: 'var(--surface-1)', color: 'var(--text-primary)' }} />
          </div>
          <div style={{ display: 'flex', gap: 8, justifyContent: 'flex-end', marginTop: 4 }}>
            <button type="button" onClick={onClose} className="btn btn-sm">Cancelar</button>
            <button type="submit" disabled={loading || !name.trim()} className="btn btn-sm btn-accent">
              {loading ? 'Guardando...' : (initial ? 'Guardar cambios' : 'Crear proveedor')}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

// ─── Main Page ────────────────────────────────────────────────────────────────

export default function SuppliersPage() {
  const { toast } = useToast();
  const router = useRouter();
  const isMobile = useIsMobile();
  const [suppliers, setSuppliers] = useState<Supplier[]>([]);
  const [loading, setLoading] = useState(true);
  const [showModal, setShowModal] = useState(false);
  const [editing, setEditing] = useState<Supplier | null>(null);
  const [confirmTarget, setConfirmTarget] = useState<Supplier | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const res = await api.get<Supplier[]>('/suppliers');
      setSuppliers(res ?? []);
    } catch (err: unknown) {
      toast((err as Error).message, 'error');
    } finally {
      setLoading(false);
    }
  }, [toast]);

  useEffect(() => { load(); }, [load]);

  async function confirmDelete() {
    if (!confirmTarget) return;
    try {
      await api.delete(`/suppliers/${confirmTarget.id}`);
      toast('Proveedor eliminado', 'success');
      load();
    } catch (err: unknown) {
      toast((err as Error).message, 'error');
    } finally {
      setConfirmTarget(null);
    }
  }

  return (
    <AppLayout>
      <div style={{ display: 'flex', flexDirection: 'column', height: '100%', overflow: 'hidden' }}>
        <Topbar
          title="Proveedores"
          subtitle={loading ? 'Cargando...' : `${suppliers.length} proveedor(es)`}
          actions={
            <div style={{ display: 'flex', gap: 8 }}>
              <button className="btn btn-sm" onClick={() => router.push('/purchases')}>Ver compras</button>
              <button className="btn btn-sm btn-accent" onClick={() => { setEditing(null); setShowModal(true); }}>
                <IconPlus /> Nuevo proveedor
              </button>
            </div>
          }
        />

        <div style={{ flex: 1, overflowY: 'auto', padding: isMobile ? '12px' : '24px' }}>
          {loading ? (
            <LoadingSpinner message="Cargando proveedores..." />
          ) : suppliers.length === 0 ? (
            <>
              <EmptyState icon={<IconTruck />} title="No hay proveedores registrados" description="Agrega tu primer proveedor para comenzar a registrar compras." />
              <div style={{ display: 'flex', justifyContent: 'center' }}>
                <button className="btn btn-sm btn-accent" onClick={() => { setEditing(null); setShowModal(true); }}>
                  <IconPlus /> Agregar primer proveedor
                </button>
              </div>
            </>
          ) : (
            <div className="table-wrap">
              <table style={{ width: '100%', borderCollapse: 'collapse' }}>
                <thead>
                  <tr>
                    <th style={{ textAlign: 'left' }}>Nombre</th>
                    {!isMobile && <th style={{ textAlign: 'left' }}>Teléfono</th>}
                    {!isMobile && <th style={{ textAlign: 'left' }}>Notas</th>}
                    <th style={{ textAlign: 'right' }}>Acciones</th>
                  </tr>
                </thead>
                <tbody>
                  {suppliers.map(s => (
                    <tr key={s.id}>
                      <td style={{ fontWeight: 600, color: 'var(--text-primary)', fontSize: 13, padding: '10px 8px' }}>{s.name}</td>
                      {!isMobile && <td style={{ fontSize: 13, color: 'var(--text-secondary)' }}>{s.phone || '—'}</td>}
                      {!isMobile && <td style={{ fontSize: 12, color: 'var(--text-muted)' }}>{s.notes || '—'}</td>}
                      <td style={{ textAlign: 'right' }}>
                        <div style={{ display: 'flex', gap: 6, justifyContent: 'flex-end' }}>
                          <button className="btn btn-sm" title="Editar proveedor" onClick={() => { setEditing(s); setShowModal(true); }}>
                            <IconEdit />{!isMobile && ' Editar'}
                          </button>
                          <button className="btn btn-sm" title="Eliminar proveedor" onClick={() => setConfirmTarget(s)}>
                            <IconTrash />
                          </button>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      </div>

      {showModal && (
        <SupplierModal initial={editing} onClose={() => setShowModal(false)} onSaved={load} />
      )}

      <ConfirmDialog
        open={confirmTarget !== null}
        title="Eliminar proveedor"
        message={`¿Eliminar a ${confirmTarget?.name}? Si tiene compras registradas, no se podrá eliminar.`}
        confirmLabel="Eliminar"
        danger
        onConfirm={confirmDelete}
        onCancel={() => setConfirmTarget(null)}
      />
    </AppLayout>
  );
}
