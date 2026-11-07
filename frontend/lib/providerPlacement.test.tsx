// US-087 contract section 4, T-1..T-4 (AC-1, AC-10).
//
// These tests check only WHERE useBusinessConfig() is called: every call made
// while a page renders must be inside a BusinessConfigProvider. They run under
// vitest's node environment with react-dom/server; effects do not run under
// SSR, so no api mock is needed. Fixture per contract section 4:
//   - '@/lib/business-config-context' is replaced by a sentinel context
//     (createContext(false)); its BusinessConfigProvider provides true and its
//     useBusinessConfig records useContext(sentinel) in `log.calls`.
//   - '@/components/AppLayout' is a pass-through into that mocked provider.
//   - 'next/navigation' exposes useRouter, useParams and usePathname stubs.
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderToString } from 'react-dom/server';
import type { ReactNode } from 'react';

const log = vi.hoisted(() => ({
  calls: [] as boolean[],
  params: {} as Record<string, string>,
}));

vi.mock('@/lib/business-config-context', async () => {
  const React = await import('react');
  const Sentinel = React.createContext(false);
  const config = {
    name: '',
    logo: '',
    restaurant_mode: true,
    print_method: 'escpos_network',
  };
  return {
    BusinessConfigProvider: ({ children }: { children: ReactNode }) =>
      React.createElement(Sentinel.Provider, { value: true }, children),
    useBusinessConfig: () => {
      log.calls.push(React.useContext(Sentinel));
      return { config, loading: false, invalidate: () => {} };
    },
  };
});

vi.mock('@/components/AppLayout', async () => {
  const React = await import('react');
  const { BusinessConfigProvider } = await import('@/lib/business-config-context');
  return {
    default: ({ children }: { children: ReactNode }) =>
      React.createElement(BusinessConfigProvider, null, children),
  };
});

vi.mock('next/navigation', () => {
  const router = {
    push: () => {},
    replace: () => {},
    back: () => {},
    forward: () => {},
    refresh: () => {},
    prefetch: () => {},
  };
  return {
    useRouter: () => router,
    useParams: () => log.params,
    usePathname: () => '/',
  };
});

beforeEach(() => {
  log.calls = [];
  log.params = {};
});

describe('useBusinessConfig provider placement', () => {
  it('T-1 POSPage calls useBusinessConfig only inside the provider', async () => {
    const { default: POSPage } = await import('@/app/pos/page');
    renderToString(<POSPage />);
    expect(log.calls.length).toBeGreaterThan(0);
    expect(log.calls).toEqual(log.calls.map(() => true));
  });

  it('T-2 OrderClient calls useBusinessConfig only inside the provider', async () => {
    log.params = { orderId: '_' };
    const { default: OrderClient } = await import('@/app/mesas/[orderId]/OrderClient');
    renderToString(<OrderClient />);
    expect(log.calls.length).toBeGreaterThan(0);
    expect(log.calls).toEqual(log.calls.map(() => true));
  });

  it('T-3 BarraPage calls useBusinessConfig only inside the provider', async () => {
    const { default: BarraPage } = await import('@/app/barra/page');
    renderToString(<BarraPage />);
    expect(log.calls.length).toBeGreaterThan(0);
    expect(log.calls).toEqual(log.calls.map(() => true));
  });

  describe('T-4 pages that already read inside the provider keep doing so', () => {
    it('T-4 CustomerDetailClient default export', async () => {
      log.params = { id: '_' };
      const { default: CustomerDetailPage } = await import('@/app/customers/[id]/CustomerDetailClient');
      renderToString(<CustomerDetailPage />);
      expect(log.calls.length).toBeGreaterThan(0);
      expect(log.calls).toEqual(log.calls.map(() => true));
    });

    it('T-4 SettingsPage', async () => {
      const { default: SettingsPage } = await import('@/app/settings/page');
      renderToString(<SettingsPage />);
      expect(log.calls.length).toBeGreaterThan(0);
      expect(log.calls).toEqual(log.calls.map(() => true));
    });
  });
});
