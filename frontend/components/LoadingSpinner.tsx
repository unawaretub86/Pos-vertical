'use client';

interface LoadingSpinnerProps {
  size?: number;
  message?: string;
}

export default function LoadingSpinner({ size = 20, message }: LoadingSpinnerProps) {
  return (
    <div
      style={{
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        padding: '48px 24px',
        gap: '12px',
      }}
    >
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke="var(--accent)"
        strokeWidth="2.5"
        strokeLinecap="round"
        style={{ animation: 'spin 0.8s linear infinite' }}
      >
        <path d="M12 2v4M12 18v4M4.93 4.93l2.83 2.83M16.24 16.24l2.83 2.83M2 12h4M18 12h4M4.93 19.07l2.83-2.83M16.24 7.76l2.83-2.83" />
      </svg>
      {message && (
        <p style={{ margin: 0, fontSize: '13px', color: 'var(--text-muted)' }}>{message}</p>
      )}
    </div>
  );
}

/** Skeleton para una fila de tabla */
export function SkeletonRow({ cols = 4 }: { cols?: number }) {
  return (
    <tr>
      {Array.from({ length: cols }).map((_, i) => (
        <td key={i} style={{ padding: '10px 16px' }}>
          <div
            style={{
              height: '14px',
              borderRadius: '4px',
              background: 'var(--surface-3)',
              width: i === 0 ? '60%' : i === cols - 1 ? '40%' : '80%',
              animation: 'skeleton-pulse 1.5s ease-in-out infinite',
            }}
          />
        </td>
      ))}
    </tr>
  );
}

/** Skeleton para stat cards del dashboard */
export function SkeletonCard({ height = 80 }: { height?: number }) {
  return (
    <div
      className="stat-card"
      style={{
        height,
        background: 'var(--surface-2)',
        animation: 'skeleton-pulse 1.5s ease-in-out infinite',
      }}
    />
  );
}
