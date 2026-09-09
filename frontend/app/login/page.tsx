'use client';

import { useState, useEffect, FormEvent } from 'react';
import { useRouter } from 'next/navigation';
import { setAuth, isAuthenticated } from '@/lib/auth';
import { api, getSetupStatus } from '@/lib/api';
import { Logo } from '@/components/Logo';

interface LoginResponse {
  token: string;
  force_password_change: boolean;
  user: { id: number; username: string; role: 'admin' | 'cashier' };
}

export default function LoginPage() {
  const router = useRouter();
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (isAuthenticated()) {
      router.replace('/');
      return;
    }
    getSetupStatus()
      .then(status => {
        if (!status.completed) router.replace('/onboarding');
      })
      .catch(() => {
        // If the status check fails, fall through to the normal login form.
      });
  }, [router]);

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError('');
    setLoading(true);
    try {
      const data = await api.post<LoginResponse>('/auth/login', { username, password });
      setAuth(data.token, data.user, data.force_password_change);
      router.replace(data.force_password_change ? '/change-password' : '/');
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Error al ingresar');
    } finally {
      setLoading(false);
    }
  }

  return (
    <div style={{
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      minHeight: '100vh',
      background: 'var(--canvas)',
      padding: '24px',
    }}>
      <div style={{ width: '100%', maxWidth: '360px' }}>
        {/* Logo */}
        <div style={{ marginBottom: '32px' }}>
          <Logo />
        </div>

        {/* Card */}
        <div style={{
          background: 'var(--surface-1)',
          border: '1px solid var(--border)',
          borderRadius: '12px',
          padding: '32px',
        }}>
          <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
            {/* Username */}
            <div>
              <label style={{ display: 'block', fontSize: '13px', fontWeight: 500, color: 'var(--text-secondary)', marginBottom: '6px' }}>
                Usuario
              </label>
              <div className="input-wrap">
                <span className="input-icon">
                  {/* User icon */}
                  <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                    <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/>
                    <circle cx="12" cy="7" r="4"/>
                  </svg>
                </span>
                <input
                  className="input"
                  type="text"
                  placeholder="Ingresa tu usuario"
                  value={username}
                  onChange={e => setUsername(e.target.value)}
                  autoComplete="username"
                  autoFocus
                  required
                />
              </div>
            </div>

            {/* Password */}
            <div>
              <label style={{ display: 'block', fontSize: '13px', fontWeight: 500, color: 'var(--text-secondary)', marginBottom: '6px' }}>
                Contraseña
              </label>
              <div className="input-wrap" style={{ position: 'relative' }}>
                <span className="input-icon">
                  {/* Lock icon */}
                  <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                    <rect x="3" y="11" width="18" height="11" rx="2" ry="2"/>
                    <path d="M7 11V7a5 5 0 0 1 10 0v4"/>
                  </svg>
                </span>
                <input
                  className="input"
                  type={showPassword ? 'text' : 'password'}
                  placeholder="Ingresa tu contraseña"
                  value={password}
                  onChange={e => setPassword(e.target.value)}
                  autoComplete="current-password"
                  style={{ paddingRight: '40px' }}
                  required
                />
                <button
                  type="button"
                  onClick={() => setShowPassword(v => !v)}
                  style={{
                    position: 'absolute',
                    right: '11px',
                    top: '50%',
                    transform: 'translateY(-50%)',
                    background: 'none',
                    border: 'none',
                    cursor: 'pointer',
                    color: 'var(--text-muted)',
                    display: 'flex',
                    alignItems: 'center',
                    padding: 0,
                  }}
                >
                  {showPassword ? (
                    // EyeOff
                    <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                      <path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94"/>
                      <path d="M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19"/>
                      <line x1="1" y1="1" x2="23" y2="23"/>
                    </svg>
                  ) : (
                    // Eye
                    <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                      <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/>
                      <circle cx="12" cy="12" r="3"/>
                    </svg>
                  )}
                </button>
              </div>
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
              disabled={loading}
              style={{ marginTop: '4px', padding: '10px 14px', fontSize: '14px' }}
            >
              {loading ? 'Ingresando...' : 'Ingresar'}
            </button>
          </form>
        </div>
      </div>
    </div>
  );
}
