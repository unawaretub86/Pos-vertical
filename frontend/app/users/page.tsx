'use client';

import { useState, useEffect, useCallback } from 'react';
import AppLayout from '@/components/AppLayout';
import Topbar from '@/components/Topbar';
import { api } from '@/lib/api';
import { useToast } from '@/components/Toast';

// ─── Types ───────────────────────────────────────────────────────────────────

interface User {
  id: number;
  username: string;
  role: string;
  active: boolean;
  created_at: string;
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString('es-CO', {
    day: '2-digit',
    month: 'short',
    year: 'numeric',
  });
}

// ─── Icons ───────────────────────────────────────────────────────────────────

const IconPlus = () => (
  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
    <line x1="12" y1="5" x2="12" y2="19" /><line x1="5" y1="12" x2="19" y2="12" />
  </svg>
);
const IconEdit = () => (
  <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
    <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7" />
    <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z" />
  </svg>
);
const IconToggleOff = () => (
  <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
    <rect x="1" y="5" width="22" height="14" rx="7" ry="7" /><circle cx="8" cy="12" r="3" fill="currentColor" />
  </svg>
);
const IconToggleOn = () => (
  <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
    <rect x="1" y="5" width="22" height="14" rx="7" ry="7" /><circle cx="16" cy="12" r="3" fill="currentColor" />
  </svg>
);
const IconX = () => (
  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
    <line x1="18" y1="6" x2="6" y2="18" /><line x1="6" y1="6" x2="18" y2="18" />
  </svg>
);
const IconUsers = () => (
  <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
    <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2" /><circle cx="9" cy="7" r="4" />
    <path d="M23 21v-2a4 4 0 0 0-3-3.87" /><path d="M16 3.13a4 4 0 0 1 0 7.75" />
  </svg>
);

// ─── User Modal ───────────────────────────────────────────────────────────────

interface UserModalProps {
  initial?: User | null;
  onClose: () => void;
  onSaved: () => void;
}

function UserModal({ initial, onClose, onSaved }: UserModalProps) {
  const { toast } = useToast();
  const isEdit = !!initial;

  const [username, setUsername] = useState(initial?.username ?? '');
  const [role, setRole] = useState(initial?.role ?? 'cashier');
  const [active, setActive] = useState(initial?.active ?? true);
  const [password, setPassword] = useState('');
  const [saving, setSaving] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setSaving(true);
    try {
      if (isEdit) {
        await api.put(`/users/${initial!.id}`, { username, role, active, password: password || undefined });
        toast('Usuario actualizado', 'success');
      } else {
        await api.post('/users', { username, password, role });
        toast('Usuario creado', 'success');
      }
      onSaved();
      onClose();
    } catch (err: unknown) {
      toast(err instanceof Error ? err.message : 'Error guardando usuario', 'error');
    } finally {
      setSaving(false);
    }
  }

  return (
    <div
      style={{
        position: 'fixed', inset: 0, zIndex: 1000,
        background: 'rgba(0,0,0,0.6)',
        display: 'flex', alignItems: 'center', justifyContent: 'center',
      }}
      onClick={onClose}
    >
      <div
        style={{
          background: 'var(--surface-1)', border: '1px solid var(--border-emphasis)',
          borderRadius: 12, padding: 24, width: 400, maxWidth: '92vw',
          boxShadow: '0 8px 32px rgba(0,0,0,0.4)',
        }}
        onClick={e => e.stopPropagation()}
      >
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 20 }}>
          <span style={{ fontSize: 15, fontWeight: 600, color: 'var(--text-primary)' }}>
            {isEdit ? 'Editar usuario' : 'Nuevo usuario'}
          </span>
          <button onClick={onClose} style={{ background: 'none', border: 'none', cursor: 'pointer', color: 'var(--text-muted)', padding: 4 }}>
            <IconX />
          </button>
        </div>

        <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
          <div>
            <label style={{ display: 'block', fontSize: 11, fontWeight: 600, textTransform: 'uppercase', letterSpacing: '0.4px', color: 'var(--text-muted)', marginBottom: 6 }}>
              Usuario
            </label>
            <input
              className="input"
              value={username}
              onChange={e => setUsername(e.target.value)}
              placeholder="nombre de usuario"
              required
              autoFocus
              style={{ width: '100%', boxSizing: 'border-box' }}
            />
          </div>

          <div>
            <label style={{ display: 'block', fontSize: 11, fontWeight: 600, textTransform: 'uppercase', letterSpacing: '0.4px', color: 'var(--text-muted)', marginBottom: 6 }}>
              Contraseña {isEdit && <span style={{ color: 'var(--text-muted)', fontWeight: 400, fontSize: 11, textTransform: 'none' }}>(dejar vacío para no cambiar)</span>}
            </label>
            <input
              className="input"
              type="password"
              value={password}
              onChange={e => setPassword(e.target.value)}
              placeholder={isEdit ? '••••••••' : 'mínimo 4 caracteres'}
              required={!isEdit}
              style={{ width: '100%', boxSizing: 'border-box' }}
            />
          </div>

          <div>
            <label style={{ display: 'block', fontSize: 11, fontWeight: 600, textTransform: 'uppercase', letterSpacing: '0.4px', color: 'var(--text-muted)', marginBottom: 6 }}>
              Rol
            </label>
            <select
              className="input"
              value={role}
              onChange={e => setRole(e.target.value)}
              style={{ width: '100%', boxSizing: 'border-box' }}
            >
              <option value="admin">Admin</option>
              <option value="cashier">Cajero</option>
            </select>
          </div>

          {isEdit && (
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '10px 12px', background: 'var(--surface-2)', borderRadius: 6, border: '1px solid var(--border)' }}>
              <div>
                <span style={{ fontSize: 13, fontWeight: 500, color: 'var(--text-primary)' }}>Estado</span>
                <div style={{ fontSize: 11, color: 'var(--text-muted)' }}>{active ? 'Activo' : 'Inactivo'}</div>
              </div>
              <button
                type="button"
                onClick={() => setActive(v => !v)}
                style={{
                  width: 36, height: 20, borderRadius: 20, border: 'none', cursor: 'pointer',
                  background: active ? 'var(--accent)' : 'var(--surface-3)',
                  position: 'relative', transition: 'background 150ms ease',
                  flexShrink: 0,
                }}
              >
                <span style={{
                  position: 'absolute', top: 3, left: active ? 18 : 3,
                  width: 14, height: 14, borderRadius: '50%', background: '#fff',
                  transition: 'left 150ms ease',
                }} />
              </button>
            </div>
          )}

          <div style={{ display: 'flex', gap: 8, marginTop: 8 }}>
            <button type="button" className="btn btn-full" onClick={onClose} disabled={saving}>
              Cancelar
            </button>
            <button type="submit" className="btn-accent btn-full" disabled={saving}>
              {saving ? 'Guardando...' : isEdit ? 'Guardar' : 'Crear usuario'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

// ─── Page ─────────────────────────────────────────────────────────────────────

export default function UsersPage() {
  const { toast } = useToast();
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(true);
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<User | null>(null);
  const [confirmDeactivate, setConfirmDeactivate] = useState<User | null>(null);
  const [confirmActivate, setConfirmActivate] = useState<User | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const users = await api.get<User[]>('/users');
      setUsers(users ?? []);
    } catch {
      toast('Error cargando usuarios', 'error');
    } finally {
      setLoading(false);
    }
  }, [toast]);

  useEffect(() => { queueMicrotask(load); }, [load]);

  async function handleDeactivate(user: User) {
    try {
      await api.delete(`/users/${user.id}`);
      toast('Usuario desactivado', 'success');
      load();
    } catch (err: unknown) {
      toast(err instanceof Error ? err.message : 'Error desactivando usuario', 'error');
    } finally {
      setConfirmDeactivate(null);
    }
  }

  async function handleActivate(user: User) {
    try {
      await api.put(`/users/${user.id}`, { username: user.username, role: user.role, active: true });
      toast('Usuario activado', 'success');
      load();
    } catch (err: unknown) {
      toast(err instanceof Error ? err.message : 'Error activando usuario', 'error');
    } finally {
      setConfirmActivate(null);
    }
  }

  return (
    <AppLayout>
      <Topbar
        title="Usuarios"
        subtitle="Gestión de cuentas y roles"
        actions={
          <button
            className="btn-accent"
            style={{ display: 'flex', alignItems: 'center', gap: 6 }}
            onClick={() => { setEditing(null); setModalOpen(true); }}
          >
            <IconPlus /> Nuevo usuario
          </button>
        }
      />

      <div style={{ padding: '24px' }}>
        {loading ? (
          <div style={{ padding: '40px 0', textAlign: 'center', color: 'var(--text-muted)', fontSize: 13 }}>
            Cargando...
          </div>
        ) : users.length === 0 ? (
          <div style={{ padding: '60px 0', textAlign: 'center' }}>
            <div style={{ color: 'var(--text-muted)', marginBottom: 12 }}><IconUsers /></div>
            <div style={{ fontSize: 13, color: 'var(--text-muted)' }}>No hay usuarios registrados</div>
          </div>
        ) : (
          <div className="table-wrap">
            <table style={{ width: '100%', borderCollapse: 'collapse' }}>
              <thead>
                <tr>
                  {['Usuario', 'Rol', 'Estado', 'Creado', 'Acciones'].map(h => (
                    <th key={h} style={{
                      padding: '8px 12px', textAlign: 'left',
                      fontSize: 11, fontWeight: 600, textTransform: 'uppercase',
                      letterSpacing: '0.5px', color: 'var(--text-muted)',
                      borderBottom: '1px solid var(--border)',
                    }}>{h}</th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {users.map(user => (
                  <tr
                    key={user.id}
                    style={{ borderBottom: '1px solid var(--border)' }}
                  >
                    <td style={{ padding: '10px 12px', fontSize: 13, fontWeight: 500, color: 'var(--text-primary)' }}>
                      {user.username}
                    </td>
                    <td style={{ padding: '10px 12px' }}>
                      <span
                        className={`badge ${user.role === 'admin' ? 'badge-credito' : 'badge-efectivo'}`}
                        style={{ fontSize: 11 }}
                      >
                        {user.role === 'admin' ? 'Admin' : 'Cajero'}
                      </span>
                    </td>
                    <td style={{ padding: '10px 12px' }}>
                      <span
                        className="badge"
                        style={{
                          fontSize: 11,
                          background: user.active ? 'var(--success-dim)' : 'var(--surface-3)',
                          color: user.active ? 'var(--success)' : 'var(--text-muted)',
                        }}
                      >
                        {user.active ? 'Activo' : 'Inactivo'}
                      </span>
                    </td>
                    <td style={{ padding: '10px 12px', fontSize: 12, color: 'var(--text-muted)' }}>
                      {formatDate(user.created_at)}
                    </td>
                    <td style={{ padding: '10px 12px' }}>
                      <div style={{ display: 'flex', gap: 6 }}>
                        <button
                          className="btn btn-sm btn-icon"
                          title="Editar"
                          onClick={() => { setEditing(user); setModalOpen(true); }}
                        >
                          <IconEdit />
                        </button>
                        {user.active ? (
                          <button
                            className="btn btn-sm btn-icon"
                            title="Desactivar"
                            style={{ color: 'var(--danger)' }}
                            onClick={() => setConfirmDeactivate(user)}
                          >
                            <IconToggleOff />
                          </button>
                        ) : (
                          <button
                            className="btn btn-sm btn-icon"
                            title="Activar"
                            style={{ color: 'var(--success)' }}
                            onClick={() => setConfirmActivate(user)}
                          >
                            <IconToggleOn />
                          </button>
                        )}
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Create / Edit modal */}
      {modalOpen && (
        <UserModal
          initial={editing}
          onClose={() => { setModalOpen(false); setEditing(null); }}
          onSaved={load}
        />
      )}

      {/* Activate confirmation */}
      {confirmActivate && (
        <div
          style={{
            position: 'fixed', inset: 0, zIndex: 1000,
            background: 'rgba(0,0,0,0.6)',
            display: 'flex', alignItems: 'center', justifyContent: 'center',
          }}
          onClick={() => setConfirmActivate(null)}
        >
          <div
            style={{
              background: 'var(--surface-1)', border: '1px solid var(--border-emphasis)',
              borderRadius: 12, padding: 24, width: 360, maxWidth: '92vw',
              boxShadow: '0 8px 32px rgba(0,0,0,0.4)',
            }}
            onClick={e => e.stopPropagation()}
          >
            <div style={{ fontSize: 15, fontWeight: 600, color: 'var(--text-primary)', marginBottom: 8 }}>
              Activar usuario
            </div>
            <p style={{ fontSize: 13, color: 'var(--text-secondary)', marginBottom: 20, lineHeight: 1.5 }}>
              ¿Activar a <strong>{confirmActivate.username}</strong>? El usuario podrá ingresar al sistema nuevamente.
            </p>
            <div style={{ display: 'flex', gap: 8 }}>
              <button className="btn btn-full" onClick={() => setConfirmActivate(null)}>
                Cancelar
              </button>
              <button
                className="btn-accent btn-full"
                onClick={() => handleActivate(confirmActivate)}
              >
                Activar
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Deactivate confirmation */}
      {confirmDeactivate && (
        <div
          style={{
            position: 'fixed', inset: 0, zIndex: 1000,
            background: 'rgba(0,0,0,0.6)',
            display: 'flex', alignItems: 'center', justifyContent: 'center',
          }}
          onClick={() => setConfirmDeactivate(null)}
        >
          <div
            style={{
              background: 'var(--surface-1)', border: '1px solid var(--border-emphasis)',
              borderRadius: 12, padding: 24, width: 360, maxWidth: '92vw',
              boxShadow: '0 8px 32px rgba(0,0,0,0.4)',
            }}
            onClick={e => e.stopPropagation()}
          >
            <div style={{ fontSize: 15, fontWeight: 600, color: 'var(--text-primary)', marginBottom: 8 }}>
              Desactivar usuario
            </div>
            <p style={{ fontSize: 13, color: 'var(--text-secondary)', marginBottom: 20, lineHeight: 1.5 }}>
              ¿Desactivar a <strong>{confirmDeactivate.username}</strong>? El usuario no podrá ingresar al sistema.
            </p>
            <div style={{ display: 'flex', gap: 8 }}>
              <button className="btn btn-full" onClick={() => setConfirmDeactivate(null)}>
                Cancelar
              </button>
              <button
                className="btn-danger btn btn-full"
                onClick={() => handleDeactivate(confirmDeactivate)}
              >
                Desactivar
              </button>
            </div>
          </div>
        </div>
      )}
    </AppLayout>
  );
}
