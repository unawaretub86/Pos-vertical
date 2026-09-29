'use client';

import { useEffect, useState, ReactNode } from 'react';
import { useRouter, usePathname } from 'next/navigation';
import { isAuthenticated, mustChangePassword } from '@/lib/auth';
import { api } from '@/lib/api';
import Sidebar from './Sidebar';
import { BusinessConfigProvider } from '@/lib/business-config-context';
import AssistantWidget from './AssistantWidget';

interface AppLayoutProps {
  children: ReactNode;
}

export default function AppLayout({ children }: AppLayoutProps) {
  const router = useRouter();
  const pathname = usePathname();
  const [ready, setReady] = useState(false);
  const [isMobile, setIsMobile] = useState(false);
  const [drawerOpen, setDrawerOpen] = useState(false);

  useEffect(() => {
    const mq = window.matchMedia('(max-width: 767px)');
    setIsMobile(mq.matches);
    const handler = (e: MediaQueryListEvent) => setIsMobile(e.matches);
    mq.addEventListener('change', handler);
    return () => mq.removeEventListener('change', handler);
  }, []);

  // Close drawer on navigation
  useEffect(() => {
    setDrawerOpen(false);
  }, [pathname]);

  useEffect(() => {
    if (!isAuthenticated()) {
      router.replace('/login');
      return;
    }
    if (mustChangePassword()) {
      router.replace('/change-password');
      return;
    }
    api.get<{ configured: boolean }>('/setup').then(cfg => {
      if (!cfg.configured) {
        router.replace('/setup');
      } else {
        setReady(true);
      }
    }).catch(() => setReady(true));
  }, [router]);

  if (!ready) return null;

  return (
    <BusinessConfigProvider>
      <div style={{ display: 'flex', height: '100vh', overflow: 'hidden', background: 'var(--canvas)' }}>
        {/* Desktop sidebar */}
        {!isMobile && <Sidebar isMobile={false} isOpen={false} onClose={() => {}} />}

        {/* Mobile drawer overlay */}
        {isMobile && drawerOpen && (
          <div
            onClick={() => setDrawerOpen(false)}
            style={{
              position: 'fixed',
              inset: 0,
              background: 'rgba(0,0,0,0.45)',
              zIndex: 99,
            }}
          />
        )}

        {/* Mobile drawer */}
        {isMobile && (
          <div style={{
            position: 'fixed',
            top: 0,
            left: 0,
            height: '100vh',
            zIndex: 100,
            transform: drawerOpen ? 'translateX(0)' : 'translateX(-100%)',
            transition: 'transform 250ms ease',
          }}>
            <Sidebar isMobile={true} isOpen={drawerOpen} onClose={() => setDrawerOpen(false)} />
          </div>
        )}

        <div style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden', minWidth: 0 }}>
          {/* Mobile top bar with hamburger */}
          {isMobile && (
            <div style={{
              height: '48px',
              display: 'flex',
              alignItems: 'center',
              padding: '0 12px',
              borderBottom: '1px solid var(--border)',
              background: 'var(--canvas)',
              flexShrink: 0,
              gap: '12px',
            }}>
              <button
                onClick={() => setDrawerOpen(true)}
                aria-label="Abrir menú"
                style={{
                  width: '36px',
                  height: '36px',
                  border: '1px solid var(--border)',
                  borderRadius: '8px',
                  background: 'var(--surface-2)',
                  cursor: 'pointer',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  color: 'var(--text-primary)',
                  flexShrink: 0,
                }}
              >
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                  <line x1="3" y1="6" x2="21" y2="6"/><line x1="3" y1="12" x2="21" y2="12"/><line x1="3" y1="18" x2="21" y2="18"/>
                </svg>
              </button>
              <span style={{ fontSize: '15px', fontWeight: 600, color: 'var(--text-primary)' }}>
                POS Vertical
              </span>
            </div>
          )}
          {children}
        </div>
      </div>
      <AssistantWidget />
    </BusinessConfigProvider>
  );
}
