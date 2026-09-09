'use client';

import type { ReactNode } from 'react';

interface EmptyStateProps {
  icon?: ReactNode;
  title: string;
  description?: string;
}

const DefaultIcon = () => (
  <svg
    width="32"
    height="32"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    strokeWidth="1.5"
    strokeLinecap="round"
    strokeLinejoin="round"
  >
    <rect x="2" y="3" width="20" height="14" rx="2" />
    <line x1="8" y1="21" x2="16" y2="21" />
    <line x1="12" y1="17" x2="12" y2="21" />
  </svg>
);

export default function EmptyState({ icon, title, description }: EmptyStateProps) {
  return (
    <div
      style={{
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        padding: '48px 24px',
        gap: '12px',
        color: 'var(--text-muted)',
      }}
    >
      <div style={{ opacity: 0.5 }}>{icon ?? <DefaultIcon />}</div>
      <p style={{ margin: 0, fontSize: '13px', fontWeight: 500, color: 'var(--text-secondary)' }}>
        {title}
      </p>
      {description && (
        <p style={{ margin: 0, fontSize: '12px', color: 'var(--text-muted)', textAlign: 'center' }}>
          {description}
        </p>
      )}
    </div>
  );
}
