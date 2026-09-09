'use client';

import { useState, useEffect, FormEvent, useRef } from 'react';
import { useRouter } from 'next/navigation';
import { isAuthenticated } from '@/lib/auth';
import { api } from '@/lib/api';

interface BusinessConfig {
  name: string;
  address: string;
  nit: string;
  currency: string;
  ticket_header: string;
  ticket_footer: string;
  logo: string;
  configured: boolean;
}

export default function SetupPage() {
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
  });
  const [logoPreview, setLogoPreview] = useState('');
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    if (!isAuthenticated()) {
      router.replace('/login');
      return;
    }
    api.get<BusinessConfig>('/setup').then(cfg => {
      if (cfg.configured) {
        setForm({
          name: cfg.name,
          address: cfg.address,
          nit: cfg.nit,
          currency: cfg.currency,
          ticket_header: cfg.ticket_header,
          ticket_footer: cfg.ticket_footer,
          logo: cfg.logo,
        });
        if (cfg.logo) setLogoPreview(cfg.logo);
      }
      setLoading(false);
    }).catch(() => setLoading(false));
  }, [router]);

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

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError('');
    setSaving(true);
    try {
      await api.put('/setup', form);
      router.replace('/');
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Error al guardar');
    } finally {
      setSaving(false);
    }
  }

  if (loading) return null;

  return (
    <div style={{
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      minHeight: '100vh',
      background: 'var(--canvas)',
      padding: '24px',
    }}>
      <div style={{ width: '100%', maxWidth: '480px' }}>
        {/* Header */}
        <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', marginBottom: '32px', gap: '12px' }}>
          <div style={{
            width: '48px', height: '48px',
            background: 'var(--accent)',
            borderRadius: '12px',
            display: 'flex', alignItems: 'center', justifyContent: 'center',
            fontWeight: 700, fontSize: '18px', color: '#000',
          }}>
            POS
          </div>
          <div style={{ textAlign: 'center' }}>
            <div style={{ fontSize: '16px', fontWeight: 600, color: 'var(--text-primary)' }}>
              Configura tu negocio
            </div>
            <div style={{ fontSize: '12px', color: 'var(--text-muted)', marginTop: '2px' }}>
              Esta información aparecerá en tus tickets y reportes
            </div>
          </div>
        </div>

        {/* Card */}
        <div style={{
          background: 'var(--surface-1)',
          border: '1px solid var(--border)',
          borderRadius: '12px',
          padding: '32px',
        }}>
          <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: '20px' }}>

            {/* Logo */}
            <div>
              <label style={{ display: 'block', fontSize: '13px', fontWeight: 500, color: 'var(--text-secondary)', marginBottom: '8px' }}>
                Logo del negocio (opcional)
              </label>
              <div style={{ display: 'flex', alignItems: 'center', gap: '16px' }}>
                <div style={{
                  width: '64px', height: '64px',
                  background: 'var(--surface-2)',
                  border: '1px solid var(--border)',
                  borderRadius: '8px',
                  display: 'flex', alignItems: 'center', justifyContent: 'center',
                  overflow: 'hidden', flexShrink: 0,
                }}>
                  {logoPreview ? (
                    <img src={logoPreview} alt="Logo" style={{ width: '100%', height: '100%', objectFit: 'contain' }} />
                  ) : (
                    <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="var(--text-muted)" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
                      <rect x="3" y="3" width="18" height="18" rx="2"/>
                      <circle cx="8.5" cy="8.5" r="1.5"/>
                      <path d="m21 15-5-5L5 21"/>
                    </svg>
                  )}
                </div>
                <div style={{ flex: 1 }}>
                  <button
                    type="button"
                    className="btn btn-sm"
                    onClick={() => fileRef.current?.click()}
                  >
                    {logoPreview ? 'Cambiar logo' : 'Subir logo'}
                  </button>
                  <div style={{ fontSize: '11px', color: 'var(--text-muted)', marginTop: '4px' }}>
                    PNG, JPG o SVG · Máx 1 MB
                  </div>
                  <input
                    ref={fileRef}
                    type="file"
                    accept="image/*"
                    style={{ display: 'none' }}
                    onChange={handleLogoChange}
                  />
                </div>
              </div>
            </div>

            {/* Nombre */}
            <div>
              <label style={{ display: 'block', fontSize: '13px', fontWeight: 500, color: 'var(--text-secondary)', marginBottom: '6px' }}>
                Nombre del negocio <span style={{ color: 'var(--danger)' }}>*</span>
              </label>
              <input
                className="input"
                type="text"
                placeholder="Ej: Tienda La Esquina"
                value={form.name}
                onChange={e => setForm(f => ({ ...f, name: e.target.value }))}
                required
                autoFocus
              />
            </div>

            {/* Dirección + NIT en grid */}
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '12px' }}>
              <div>
                <label style={{ display: 'block', fontSize: '13px', fontWeight: 500, color: 'var(--text-secondary)', marginBottom: '6px' }}>
                  Dirección
                </label>
                <input
                  className="input"
                  type="text"
                  placeholder="Calle 10 # 5-23"
                  value={form.address}
                  onChange={e => setForm(f => ({ ...f, address: e.target.value }))}
                />
              </div>
              <div>
                <label style={{ display: 'block', fontSize: '13px', fontWeight: 500, color: 'var(--text-secondary)', marginBottom: '6px' }}>
                  NIT / CC
                </label>
                <input
                  className="input"
                  type="text"
                  placeholder="900.123.456-7"
                  value={form.nit}
                  onChange={e => setForm(f => ({ ...f, nit: e.target.value }))}
                />
              </div>
            </div>

            {/* Ticket header */}
            <div>
              <label style={{ display: 'block', fontSize: '13px', fontWeight: 500, color: 'var(--text-secondary)', marginBottom: '6px' }}>
                Encabezado del ticket (opcional)
              </label>
              <input
                className="input"
                type="text"
                placeholder="Ej: ¡Gracias por su compra!"
                value={form.ticket_header}
                onChange={e => setForm(f => ({ ...f, ticket_header: e.target.value }))}
              />
            </div>

            {/* Ticket footer */}
            <div>
              <label style={{ display: 'block', fontSize: '13px', fontWeight: 500, color: 'var(--text-secondary)', marginBottom: '6px' }}>
                Pie del ticket (opcional)
              </label>
              <input
                className="input"
                type="text"
                placeholder="Ej: No se aceptan devoluciones."
                value={form.ticket_footer}
                onChange={e => setForm(f => ({ ...f, ticket_footer: e.target.value }))}
              />
            </div>

            {/* Error */}
            {error && (
              <div style={{ fontSize: '13px', color: 'var(--danger)', textAlign: 'center' }}>
                {error}
              </div>
            )}

            {/* Submit */}
            <button
              type="submit"
              className="btn btn-accent btn-full"
              disabled={saving}
              style={{ marginTop: '4px', padding: '10px 14px', fontSize: '14px' }}
            >
              {saving ? 'Guardando...' : 'Guardar y continuar →'}
            </button>
          </form>
        </div>
      </div>
    </div>
  );
}
