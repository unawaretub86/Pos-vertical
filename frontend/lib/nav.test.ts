import { describe, it, expect } from 'vitest';
import { filterNavItems } from './nav';

// Contrato US-055 T-16. Item shape follows contract §5: adminOnly / restaurantOnly flags.
const items = [
  { href: '/pos', label: 'Punto de Venta' },
  { href: '/', label: 'Dashboard', adminOnly: true },
  { href: '/register', label: 'Caja' },
  { href: '/mesas', label: 'Mesas', restaurantOnly: true },
  { href: '/barra', label: 'Barra', restaurantOnly: true },
  { href: '/settings', label: 'Configuración', adminOnly: true },
];

const hrefs = (r: { href: string }[]) => r.map((i) => i.href);

describe('filterNavItems (T-16)', () => {
  it('restaurantMode=false hides /mesas and /barra for admin and cashier', () => {
    for (const isAdmin of [true, false]) {
      const h = hrefs(filterNavItems(items, { isAdmin, restaurantMode: false }));
      expect(h).not.toContain('/mesas');
      expect(h).not.toContain('/barra');
    }
  });

  it('restaurantMode=true, cashier sees mesas, barra, pos, register but not settings', () => {
    const h = hrefs(filterNavItems(items, { isAdmin: false, restaurantMode: true }));
    expect(h).toEqual(['/pos', '/register', '/mesas', '/barra']);
    expect(h).not.toContain('/settings');
  });

  it('restaurantMode=true, admin sees everything', () => {
    const h = hrefs(filterNavItems(items, { isAdmin: true, restaurantMode: true }));
    expect(h).toEqual(['/pos', '/', '/register', '/mesas', '/barra', '/settings']);
  });
});
