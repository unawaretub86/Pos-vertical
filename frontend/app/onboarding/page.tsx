'use client';

import { useState, useEffect, FormEvent } from 'react';
import { useRouter } from 'next/navigation';
import { setAuth } from '@/lib/auth';
import { getSetupStatus, submitOnboarding } from '@/lib/api';
import { Logo } from '@/components/Logo';

const MIN_PASSWORD_LENGTH = 4;

export default function OnboardingPage() {
  const router = useRouter();
  const [checking, setChecking] = useState(true);
  const [step, setStep] = useState<1 | 2>(1);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const [fullName, setFullName] = useState('');
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');

  const [businessName, setBusinessName] = useState('');
  const [nit, setNit] = useState('');

  useEffect(() => {
    getSetupStatus()
      .then(status => {
        if (status.completed) {
          router.replace('/login');
        } else {
          setChecking(false);
        }
      })
      .catch(() => setChecking(false));
  }, [router]);

  function goToStep2(e: FormEvent) {
    e.preventDefault();
    setError('');
    if (fullName.trim() === '') {
      setError('El nombre completo es requerido');
      return;
    }
    if (username.trim() === '') {
      setError('El usuario es requerido');
      return;
    }
    if (password.length < MIN_PASSWORD_LENGTH) {
      setError(`La contraseña debe tener al menos ${MIN_PASSWORD_LENGTH} caracteres`);
      return;
    }
    if (password !== confirmPassword) {
      setError('Las contraseñas no coinciden');
      return;
    }
    setStep(2);
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError('');
    if (businessName.trim() === '') {
      setError('El nombre del negocio es requerido');
      return;
    }
    if (nit.trim() === '') {
      setError('El NIT es requerido');
      return;
    }

    setLoading(true);
    try {
      const data = await submitOnboarding({
        full_name: fullName.trim(),
        username: username.trim(),
        password,
        business_name: businessName.trim(),
        nit: nit.trim(),
      });
      setAuth(data.token, data.user, false);
      router.replace('/');
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Error al completar la configuración');
    } finally {
      setLoading(false);
    }
  }

  if (checking) return null;

  return (
    <div style={{
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      minHeight: '100vh',
      background: 'var(--canvas)',
      padding: '24px',
    }}>
      <div style={{ width: '100%', maxWidth: '380px' }}>
        <div style={{ marginBottom: '28px' }}>
          <Logo />
        </div>

        <div style={{
          background: 'var(--surface-1)',
          border: '1px solid var(--border)',
          borderRadius: '12px',
          padding: '32px',
        }}>
          {/* Stepper */}
          <div style={{ display: 'flex', alignItems: 'center', gap: '6px', marginBottom: '20px' }}>
            <StepDot n={1} active={step >= 1} />
            <div style={{ flex: 1, height: '2px', background: step >= 2 ? 'var(--accent)' : 'var(--border-emphasis)' }} />
            <StepDot n={2} active={step >= 2} />
          </div>
          <div style={{ fontSize: '12px', color: 'var(--text-secondary)', marginBottom: '4px' }}>
            Paso {step} de 2 · {step === 1 ? 'Tu cuenta' : 'Tu negocio'}
          </div>
          <div style={{ fontSize: '16px', fontWeight: 700, color: 'var(--text-primary)', marginBottom: '18px' }}>
            {step === 1 ? 'Crea tu usuario admin' : 'Cuéntanos de tu negocio'}
          </div>

          {step === 1 ? (
            <form onSubmit={goToStep2} style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
              <Field label="Nombre completo">
                <input
                  className="input"
                  type="text"
                  placeholder="Tu nombre completo"
                  value={fullName}
                  onChange={e => setFullName(e.target.value)}
                  autoFocus
                  required
                />
              </Field>
              <Field label="Usuario">
                <input
                  className="input"
                  type="text"
                  placeholder="Nombre de usuario"
                  value={username}
                  onChange={e => setUsername(e.target.value)}
                  autoComplete="username"
                  required
                />
              </Field>
              <Field label="Contraseña">
                <input
                  className="input"
                  type="password"
                  placeholder="Mínimo 4 caracteres"
                  value={password}
                  onChange={e => setPassword(e.target.value)}
                  autoComplete="new-password"
                  required
                />
              </Field>
              <Field label="Confirmar contraseña">
                <input
                  className="input"
                  type="password"
                  placeholder="Repite tu contraseña"
                  value={confirmPassword}
                  onChange={e => setConfirmPassword(e.target.value)}
                  autoComplete="new-password"
                  required
                />
              </Field>

              {error && <ErrorMessage message={error} />}

              <button type="submit" className="btn btn-accent btn-full" style={{ marginTop: '4px', padding: '10px 14px', fontSize: '14px' }}>
                Siguiente →
              </button>
            </form>
          ) : (
            <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
              <Field label="Nombre del negocio">
                <input
                  className="input"
                  type="text"
                  placeholder="Ej: Tienda Chévere"
                  value={businessName}
                  onChange={e => setBusinessName(e.target.value)}
                  autoFocus
                  required
                />
              </Field>
              <Field label="NIT">
                <input
                  className="input"
                  type="text"
                  placeholder="Ej: 900123456"
                  value={nit}
                  onChange={e => setNit(e.target.value)}
                  required
                />
              </Field>

              {error && <ErrorMessage message={error} />}

              <div style={{ display: 'flex', gap: '8px', marginTop: '4px' }}>
                <button
                  type="button"
                  className="btn btn-ghost"
                  onClick={() => { setError(''); setStep(1); }}
                  disabled={loading}
                  style={{ padding: '10px 14px', fontSize: '14px' }}
                >
                  ← Atrás
                </button>
                <button
                  type="submit"
                  className="btn btn-accent btn-full"
                  disabled={loading}
                  style={{ padding: '10px 14px', fontSize: '14px' }}
                >
                  {loading ? 'Configurando...' : 'Empezar a vender'}
                </button>
              </div>
            </form>
          )}
        </div>
      </div>
    </div>
  );
}

function StepDot({ n, active }: { n: number; active: boolean }) {
  return (
    <div style={{
      width: '22px',
      height: '22px',
      borderRadius: '50%',
      background: active ? 'var(--accent)' : 'var(--surface-3, var(--border))',
      color: active ? '#0D0D0D' : 'var(--text-muted)',
      fontSize: '11px',
      fontWeight: 700,
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      flexShrink: 0,
    }}>
      {n}
    </div>
  );
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div>
      <label style={{ display: 'block', fontSize: '13px', fontWeight: 500, color: 'var(--text-secondary)', marginBottom: '6px' }}>
        {label}
      </label>
      {children}
    </div>
  );
}

function ErrorMessage({ message }: { message: string }) {
  return (
    <div style={{ fontSize: '13px', color: 'var(--danger)', textAlign: 'center' }}>
      {message}
    </div>
  );
}
