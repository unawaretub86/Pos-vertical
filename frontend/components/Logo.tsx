interface LogoProps {
  size?: number;
  subtitle?: string;
}

export function Logo({ size = 48, subtitle = 'Punto de Venta · Colombia' }: LogoProps) {
  const iconSize = Math.round(size * 0.625);

  return (
    <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: '12px' }}>
      <div style={{
        width: `${size}px`,
        height: `${size}px`,
        background: 'var(--accent-dim)',
        borderRadius: `${Math.round(size * 0.29)}px`,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
      }}>
        <svg width={iconSize} height={iconSize} viewBox="0 0 24 24" fill="none">
          <path
            d="M12.59 2.59A2 2 0 0 0 11.17 2H4a2 2 0 0 0-2 2v7.17a2 2 0 0 0 .59 1.41l8.83 8.83a2 2 0 0 0 2.83 0l7.17-7.17a2 2 0 0 0 0-2.83l-8.83-8.82Z"
            stroke="var(--accent)"
            strokeWidth="1.8"
            strokeLinejoin="round"
          />
          <circle cx="7.5" cy="7.5" r="1.5" fill="var(--accent)" />
          <path
            d="m9 15 3 3 5-5"
            stroke="var(--accent)"
            strokeWidth="1.8"
            strokeLinecap="round"
            strokeLinejoin="round"
          />
        </svg>
      </div>
      <div style={{ textAlign: 'center' }}>
        <div style={{ fontSize: '18px', fontWeight: 700, color: 'var(--text-primary)', letterSpacing: '-0.3px' }}>
          POS <span style={{ color: 'var(--accent)' }}>Chévere</span>
        </div>
        {subtitle && (
          <div style={{ fontSize: '11px', color: 'var(--text-secondary)', marginTop: '2px', letterSpacing: '0.04em' }}>
            {subtitle.toUpperCase()}
          </div>
        )}
      </div>
    </div>
  );
}
