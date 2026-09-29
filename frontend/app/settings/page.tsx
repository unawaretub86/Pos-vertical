'use client';

import { useState, useEffect, FormEvent, useRef } from 'react';
import { useRouter } from 'next/navigation';
import AppLayout from '@/components/AppLayout';
import Topbar from '@/components/Topbar';
import { useToast } from '@/components/Toast';
import { api } from '@/lib/api';
import { useBusinessConfig } from '@/lib/business-config-context';

// ─── Types ────────────────────────────────────────────────────────────────────

interface FiscalConfig {
  enabled: boolean;
  numbering_range_id: number;
  municipality_id: number;
}

interface BusinessConfig {
  name: string;
  address: string;
  nit: string;
  currency: string;
  ticket_header: string;
  ticket_footer: string;
  logo: string;
  default_opening_amount: number;
  allow_global_discount: boolean;
  owner_email: string;
  owner_email_app_password: string;
  restaurant_mode: boolean;
  lan_access_enabled: boolean;
  configured: boolean;
}

// ─── Config Row ───────────────────────────────────────────────────────────────

function ConfigRow({ label, sublabel, children }: { label: string; sublabel?: string; children: React.ReactNode }) {
  return (
    <div style={{
      padding: '12px 16px',
      background: 'var(--surface-1)',
      border: '1px solid var(--border)',
      borderRadius: '8px',
      display: 'flex',
      alignItems: 'center',
      gap: '16px',
    }}>
      <div style={{ flex: 1, minWidth: 0 }}>
        <div style={{ fontSize: '13px', fontWeight: 500, color: 'var(--text-primary)' }}>{label}</div>
        {sublabel && (
          <div style={{ fontSize: '11px', color: 'var(--text-muted)', marginTop: '2px' }}>{sublabel}</div>
        )}
      </div>
      <div style={{ flexShrink: 0 }}>{children}</div>
    </div>
  );
}

// ─── Tables section (restaurant mode) ─────────────────────────────────────────

interface DiningTable {
  id: number;
  name: string;
  zone: string;
  seats: number;
  sort_order: number;
}

function TablesSection() {
  const [tables, setTables] = useState<DiningTable[]>([]);
  const [loaded, setLoaded] = useState(false);
  const [error, setError] = useState('');
  const [newName, setNewName] = useState('');
  const [newZone, setNewZone] = useState('');
  const [editId, setEditId] = useState<number | null>(null);
  const [edit, setEdit] = useState({ name: '', zone: '', sort_order: 0 });

  async function load() {
    try {
      setTables(await api.get<DiningTable[]>('/tables'));
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Error al cargar mesas');
    } finally {
      setLoaded(true);
    }
  }

  useEffect(() => {
    api.get<DiningTable[]>('/tables')
      .then(setTables)
      .catch((err: unknown) => setError(err instanceof Error ? err.message : 'Error al cargar mesas'))
      .finally(() => setLoaded(true));
  }, []);

  async function run(action: () => Promise<unknown>) {
    setError('');
    try {
      await action();
      await load();
      return true;
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Error');
      return false;
    }
  }

  async function handleCreate(e: FormEvent) {
    e.preventDefault();
    const ok = await run(() => api.post('/tables', { name: newName, zone: newZone }));
    if (ok) { setNewName(''); setNewZone(''); }
  }

  function startEdit(t: DiningTable) {
    setEditId(t.id);
    setEdit({ name: t.name, zone: t.zone, sort_order: t.sort_order });
  }

  async function saveEdit(t: DiningTable) {
    const ok = await run(() => api.put(`/tables/${t.id}`, { ...edit, seats: t.seats }));
    if (ok) setEditId(null);
  }

  return (
    <div style={{ marginTop: '32px' }}>
      <div style={{ fontSize: '11px', fontWeight: 600, textTransform: 'uppercase', letterSpacing: '0.5px', color: 'var(--text-muted)', marginBottom: '10px' }}>
        Mesas
      </div>

      {error && (
        <div role="alert" style={{ padding: '8px 12px', marginBottom: '8px', borderRadius: '6px', fontSize: '13px', border: '1px solid var(--border)', background: 'var(--surface-2)', color: 'var(--text-primary)' }}>
          {error}
        </div>
      )}

      <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
        {loaded && tables.length === 0 && (
          <div style={{ padding: '12px 16px', fontSize: '13px', color: 'var(--text-muted)', background: 'var(--surface-1)', border: '1px solid var(--border)', borderRadius: '8px' }}>
            Aún no hay mesas. Crea la primera con el botón de abajo.
          </div>
        )}

        {tables.map(t => (
          <div key={t.id} style={{ padding: '10px 12px', background: 'var(--surface-1)', border: '1px solid var(--border)', borderRadius: '8px', display: 'flex', flexWrap: 'wrap', alignItems: 'center', gap: '8px' }}>
            {editId === t.id ? (
              <>
                <input className="input" aria-label="Nombre" value={edit.name} onChange={e => setEdit(v => ({ ...v, name: e.target.value }))} style={{ flex: '1 1 120px', minWidth: 0 }} />
                <input className="input" aria-label="Zona" placeholder="Zona" value={edit.zone} onChange={e => setEdit(v => ({ ...v, zone: e.target.value }))} style={{ flex: '1 1 100px', minWidth: 0 }} />
                <input className="input" aria-label="Orden" type="number" value={edit.sort_order} onChange={e => setEdit(v => ({ ...v, sort_order: parseInt(e.target.value) || 0 }))} style={{ width: '70px' }} />
                <button type="button" className="btn btn-sm btn-accent" onClick={() => saveEdit(t)}>Guardar</button>
                <button type="button" className="btn btn-sm" onClick={() => setEditId(null)}>Cancelar</button>
              </>
            ) : (
              <>
                <div style={{ flex: '1 1 120px', minWidth: 0 }}>
                  <div style={{ fontSize: '13px', fontWeight: 500, color: 'var(--text-primary)', overflowWrap: 'anywhere' }}>{t.name}</div>
                  <div style={{ fontSize: '11px', color: 'var(--text-muted)' }}>
                    {t.zone || 'Sin zona'} · Orden {t.sort_order}
                  </div>
                </div>
                <button type="button" className="btn btn-sm" onClick={() => startEdit(t)}>Editar</button>
                <button type="button" className="btn btn-sm" onClick={() => run(() => api.delete(`/tables/${t.id}`))}>Desactivar</button>
              </>
            )}
          </div>
        ))}

        <form onSubmit={handleCreate} style={{ display: 'flex', flexWrap: 'wrap', gap: '8px' }}>
          <input className="input" aria-label="Nombre de la mesa" placeholder="Nombre (Ej: Mesa 1)" value={newName} onChange={e => setNewName(e.target.value)} style={{ flex: '1 1 140px', minWidth: 0 }} />
          <input className="input" aria-label="Zona de la mesa" placeholder="Zona (opcional)" value={newZone} onChange={e => setNewZone(e.target.value)} style={{ flex: '1 1 120px', minWidth: 0 }} />
          <button type="submit" className="btn btn-sm btn-accent">Crear mesa</button>
        </form>
      </div>
    </div>
  );
}

// ─── Page ─────────────────────────────────────────────────────────────────────

export default function SettingsPage() {
  const { toast } = useToast();
  const { config: bizConfig, invalidate: invalidateConfig } = useBusinessConfig();
  const router = useRouter();
  const fileRef = useRef<HTMLInputElement>(null);

  const [form, setForm] = useState({
    name: '',
    address: '',
    nit: '',
    currency: 'COP',
    ticket_header: '',
    ticket_footer: '',
    logo: '',
    default_opening_amount: 0,
    allow_global_discount: false,
    owner_email: '',
    owner_email_app_password: '',
    restaurant_mode: false,
    lan_access_enabled: false,
  });
  const [fiscal, setFiscal] = useState<FiscalConfig>({
    enabled: false,
    numbering_range_id: 0,
    municipality_id: 149,
  });
  const [savingFiscal, setSavingFiscal] = useState(false);

  const [logoPreview, setLogoPreview] = useState('');
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    api.get<FiscalConfig>('/fiscal-config')
      .then(cfg => setFiscal({
        enabled: cfg.enabled ?? false,
        numbering_range_id: cfg.numbering_range_id ?? 0,
        municipality_id: cfg.municipality_id ?? 149,
      }))
      .catch(() => {/* fiscal config may not exist yet */});

    api.get<BusinessConfig>('/setup')
      .then(cfg => {
        setForm({
          name: cfg.name ?? '',
          address: cfg.address ?? '',
          nit: cfg.nit ?? '',
          currency: cfg.currency ?? 'COP',
          ticket_header: cfg.ticket_header ?? '',
          ticket_footer: cfg.ticket_footer ?? '',
          logo: cfg.logo ?? '',
          default_opening_amount: cfg.default_opening_amount ?? 0,
          allow_global_discount: cfg.allow_global_discount ?? false,
          owner_email: cfg.owner_email ?? '',
          owner_email_app_password: cfg.owner_email_app_password ?? '',
          restaurant_mode: cfg.restaurant_mode ?? false,
          lan_access_enabled: cfg.lan_access_enabled ?? false,
        });
        if (cfg.logo) setLogoPreview(cfg.logo);
      })
      .catch(() => toast('Error al cargar configuración', 'error'))
      .finally(() => setLoading(false));
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  function handleLogoChange(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    if (!file) return;
    const reader = new FileReader();
    reader.onload = ev => {
      const result = ev.target?.result as string;
      setLogoPreview(result);
      setForm(f => ({ ...f, logo: result }));
    };
    reader.readAsDataURL(file);
  }

  async function handleFiscalSave(e: FormEvent) {
    e.preventDefault();
    if (fiscal.enabled && fiscal.numbering_range_id === 0) {
      toast('ID Rango de Numeración es requerido cuando la facturación está habilitada', 'error');
      return;
    }
    setSavingFiscal(true);
    try {
      await api.put('/fiscal-config', fiscal);
      toast('Configuración DIAN guardada', 'success');
    } catch (err: unknown) {
      toast(err instanceof Error ? err.message : 'Error al guardar config DIAN', 'error');
    } finally {
      setSavingFiscal(false);
    }
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setSaving(true);
    try {
      await api.put('/setup', form);
      invalidateConfig();
      toast('Configuración guardada', 'success');
    } catch (err: unknown) {
      toast(err instanceof Error ? err.message : 'Error al guardar', 'error');
    } finally {
      setSaving(false);
    }
  }

  return (
    <AppLayout>
      <Topbar title="Configuración" subtitle="Negocio, tickets y preferencias" />

      <div style={{ padding: '24px', maxWidth: '680px' }}>
        {loading ? (
          <div style={{ color: 'var(--text-muted)', fontSize: '13px' }}>Cargando...</div>
        ) : (
          <>
          <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: '24px' }}>

            {/* ── Sección: Negocio ─────────────────────────── */}
            <div>
              <div style={{ fontSize: '11px', fontWeight: 600, textTransform: 'uppercase', letterSpacing: '0.5px', color: 'var(--text-muted)', marginBottom: '10px' }}>
                Información del negocio
              </div>
              <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>

                {/* Logo */}
                <ConfigRow label="Logo" sublabel="Se mostrará en la pantalla principal">
                  <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
                    <div style={{
                      width: '40px', height: '40px',
                      background: 'var(--surface-2)',
                      border: '1px solid var(--border)',
                      borderRadius: '6px',
                      display: 'flex', alignItems: 'center', justifyContent: 'center',
                      overflow: 'hidden', flexShrink: 0,
                    }}>
                      {logoPreview ? (
                        <img src={logoPreview} alt="Logo" style={{ width: '100%', height: '100%', objectFit: 'contain' }} />
                      ) : (
                        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="var(--text-muted)" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
                          <rect x="3" y="3" width="18" height="18" rx="2"/>
                          <circle cx="8.5" cy="8.5" r="1.5"/>
                          <path d="m21 15-5-5L5 21"/>
                        </svg>
                      )}
                    </div>
                    <button type="button" className="btn btn-sm" onClick={() => fileRef.current?.click()}>
                      {logoPreview ? 'Cambiar' : 'Subir'}
                    </button>
                    <input ref={fileRef} type="file" accept="image/*" style={{ display: 'none' }} onChange={handleLogoChange} />
                  </div>
                </ConfigRow>

                {/* Nombre */}
                <ConfigRow label="Nombre del negocio" sublabel="Aparece en tickets y reportes">
                  <input
                    className="input"
                    type="text"
                    placeholder="Ej: Tienda La Esquina"
                    value={form.name}
                    onChange={e => setForm(f => ({ ...f, name: e.target.value }))}
                    required
                    style={{ width: '220px' }}
                  />
                </ConfigRow>

                {/* Dirección */}
                <ConfigRow label="Dirección">
                  <input
                    className="input"
                    type="text"
                    placeholder="Calle 10 # 5-23"
                    value={form.address}
                    onChange={e => setForm(f => ({ ...f, address: e.target.value }))}
                    style={{ width: '220px' }}
                  />
                </ConfigRow>

                {/* NIT */}
                <ConfigRow label="NIT / Cédula">
                  <input
                    className="input"
                    type="text"
                    placeholder="900.123.456-7"
                    value={form.nit}
                    onChange={e => setForm(f => ({ ...f, nit: e.target.value }))}
                    style={{ width: '220px' }}
                  />
                </ConfigRow>
              </div>
            </div>

            {/* ── Sección: Tickets ─────────────────────────── */}
            <div>
              <div style={{ fontSize: '11px', fontWeight: 600, textTransform: 'uppercase', letterSpacing: '0.5px', color: 'var(--text-muted)', marginBottom: '10px' }}>
                Formato de tickets
              </div>
              <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>

                <ConfigRow label="Encabezado" sublabel="Línea extra bajo el nombre del negocio">
                  <input
                    className="input"
                    type="text"
                    placeholder="Ej: ¡Gracias por su compra!"
                    value={form.ticket_header}
                    onChange={e => setForm(f => ({ ...f, ticket_header: e.target.value }))}
                    style={{ width: '220px' }}
                  />
                </ConfigRow>

                <ConfigRow label="Pie de página" sublabel="Línea al final del ticket">
                  <input
                    className="input"
                    type="text"
                    placeholder="Ej: No se aceptan devoluciones."
                    value={form.ticket_footer}
                    onChange={e => setForm(f => ({ ...f, ticket_footer: e.target.value }))}
                    style={{ width: '220px' }}
                  />
                </ConfigRow>
              </div>
            </div>

            {/* ── Sección: Caja ────────────────────────────── */}
            <div>
              <div style={{ fontSize: '11px', fontWeight: 600, textTransform: 'uppercase', letterSpacing: '0.5px', color: 'var(--text-muted)', marginBottom: '10px' }}>
                Caja registradora
              </div>
              <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
                <ConfigRow label="Monto base por defecto" sublabel="Se pre-llena al abrir caja (opcional, 0 = sin valor por defecto)">
                  <input
                    className="input"
                    type="number"
                    min="0"
                    step="1000"
                    placeholder="0"
                    value={form.default_opening_amount || ''}
                    onChange={e => setForm(f => ({ ...f, default_opening_amount: parseFloat(e.target.value) || 0 }))}
                    style={{ width: '140px', textAlign: 'right' }}
                  />
                </ConfigRow>

                <ConfigRow label="Descuento global en ventas" sublabel="Permite aplicar un descuento sobre el total de la venta en el POS">
                  <label style={{ display: 'flex', alignItems: 'center', gap: '8px', cursor: 'pointer' }}>
                    <input
                      type="checkbox"
                      checked={form.allow_global_discount}
                      onChange={e => setForm(f => ({ ...f, allow_global_discount: e.target.checked }))}
                    />
                    <span style={{ fontSize: '13px', color: 'var(--text-secondary)' }}>
                      {form.allow_global_discount ? 'Habilitado' : 'Deshabilitado'}
                    </span>
                  </label>
                </ConfigRow>
              </div>
            </div>

            {/* ── Sección: Administración ──────────────────── */}
            <div>
              <div style={{ fontSize: '11px', fontWeight: 600, textTransform: 'uppercase', letterSpacing: '0.5px', color: 'var(--text-muted)', marginBottom: '10px' }}>
                Administración
              </div>
              <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
                <ConfigRow label="Usuarios" sublabel="Crear, editar o desactivar usuarios del sistema">
                  <button
                    type="button"
                    className="btn btn-sm"
                    onClick={() => router.push('/users')}
                  >
                    Gestionar usuarios →
                  </button>
                </ConfigRow>

                <ConfigRow label="Modo restaurante" sublabel="Activa Mesas y Barra en el menú y la gestión de mesas">
                  <label style={{ display: 'flex', alignItems: 'center', gap: '8px', cursor: 'pointer' }}>
                    <input
                      type="checkbox"
                      checked={form.restaurant_mode}
                      onChange={e => setForm(f => ({ ...f, restaurant_mode: e.target.checked }))}
                    />
                    <span style={{ fontSize: '13px', color: 'var(--text-secondary)' }}>
                      {form.restaurant_mode ? 'Activado' : 'Desactivado'}
                    </span>
                  </label>
                </ConfigRow>

                <ConfigRow label="Correo del dueño" sublabel="Recibe aquí el resumen de cierre de caja automáticamente, sin hacer nada">
                  <input
                    className="input"
                    type="email"
                    placeholder="dueño@gmail.com"
                    value={form.owner_email}
                    onChange={e => setForm(f => ({ ...f, owner_email: e.target.value }))}
                    style={{ width: '220px' }}
                  />
                </ConfigRow>

                <ConfigRow label="Contraseña de aplicación" sublabel="Generada en la cuenta de Gmail de arriba (Cuenta de Google → Seguridad → Contraseñas de aplicaciones), no tu contraseña normal">
                  <input
                    className="input"
                    type="password"
                    placeholder="•••• •••• •••• ••••"
                    autoComplete="new-password"
                    value={form.owner_email_app_password}
                    onChange={e => setForm(f => ({ ...f, owner_email_app_password: e.target.value }))}
                    style={{ width: '220px' }}
                  />
                </ConfigRow>
              </div>
            </div>

            {/* ── Guardar ──────────────────────────────────── */}
            <div>
              <button
                type="submit"
                className="btn btn-accent"
                disabled={saving}
                style={{ padding: '9px 24px', fontSize: '13px' }}
              >
                {saving ? 'Guardando...' : 'Guardar cambios'}
              </button>
            </div>
          </form>

          {bizConfig.restaurant_mode && <TablesSection />}

          {/* ── Sección: Facturación DIAN ─────────────────── */}
          <form onSubmit={handleFiscalSave} style={{ display: 'flex', flexDirection: 'column', gap: '24px', marginTop: '32px' }}>
            <div>
              <div style={{ fontSize: '11px', fontWeight: 600, textTransform: 'uppercase', letterSpacing: '0.5px', color: 'var(--text-muted)', marginBottom: '10px' }}>
                Facturación electrónica DIAN
              </div>
              <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>

                <ConfigRow label="Facturación habilitada" sublabel="Emitir documentos equivalentes electrónicos">
                  <label style={{ display: 'flex', alignItems: 'center', gap: '8px', cursor: 'pointer' }}>
                    <input
                      type="checkbox"
                      checked={fiscal.enabled}
                      onChange={e => setFiscal(f => ({ ...f, enabled: e.target.checked }))}
                    />
                    <span style={{ fontSize: '13px', color: 'var(--text-secondary)' }}>
                      {fiscal.enabled ? 'Habilitada' : 'Deshabilitada'}
                    </span>
                  </label>
                </ConfigRow>

                <ConfigRow
                  label="ID Rango de Numeración"
                  sublabel="Encuéntralo en tu cuenta Factus → Numeración"
                >
                  <input
                    className="input"
                    type="number"
                    min="0"
                    placeholder="Ej: 5"
                    value={fiscal.numbering_range_id || ''}
                    onChange={e => setFiscal(f => ({ ...f, numbering_range_id: parseInt(e.target.value) || 0 }))}
                    style={{ width: '140px', textAlign: 'right' }}
                  />
                </ConfigRow>

                <ConfigRow
                  label="Municipio DIAN (ID)"
                  sublabel="149 = Bogotá. Consulta la tabla de municipios DIAN."
                >
                  <input
                    className="input"
                    type="number"
                    min="1"
                    placeholder="149"
                    value={fiscal.municipality_id || ''}
                    onChange={e => setFiscal(f => ({ ...f, municipality_id: parseInt(e.target.value) || 149 }))}
                    style={{ width: '140px', textAlign: 'right' }}
                  />
                </ConfigRow>
              </div>
            </div>

            <div>
              <button
                type="submit"
                className="btn btn-accent"
                disabled={savingFiscal}
                style={{ padding: '9px 24px', fontSize: '13px' }}
              >
                {savingFiscal ? 'Guardando...' : 'Guardar config DIAN'}
              </button>
            </div>
          </form>
          </>
        )}
      </div>
    </AppLayout>
  );
}
