'use client';

import { useEffect, useRef } from 'react';

interface ConfirmDialogProps {
  open: boolean;
  title: string;
  message: string;
  confirmLabel?: string;
  cancelLabel?: string;
  danger?: boolean;
  onConfirm: () => void;
  onCancel: () => void;
}

export default function ConfirmDialog({
  open,
  title,
  message,
  confirmLabel = 'Confirmar',
  cancelLabel = 'Cancelar',
  danger = false,
  onConfirm,
  onCancel,
}: ConfirmDialogProps) {
  const cancelRef = useRef<HTMLButtonElement>(null);

  useEffect(() => {
    if (open) cancelRef.current?.focus();
  }, [open]);

  useEffect(() => {
    if (!open) return;
    const handler = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onCancel();
    };
    window.addEventListener('keydown', handler);
    return () => window.removeEventListener('keydown', handler);
  }, [open, onCancel]);

  if (!open) return null;

  return (
    <div
      style={{
        position: 'fixed',
        inset: 0,
        background: 'rgba(0,0,0,0.6)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        zIndex: 9998,
        animation: 'fadeIn 150ms ease-out',
      }}
      onClick={onCancel}
    >
      <div
        style={{
          background: 'var(--surface-1)',
          border: '1px solid var(--border-emphasis)',
          borderRadius: '12px',
          padding: '24px',
          width: '100%',
          maxWidth: '400px',
          boxShadow: '0 8px 32px rgba(0,0,0,0.4)',
          animation: 'modalIn 200ms ease-out',
        }}
        onClick={(e) => e.stopPropagation()}
      >
        <h3
          style={{
            margin: '0 0 8px',
            fontSize: '15px',
            fontWeight: 600,
            color: 'var(--text-primary)',
            letterSpacing: '-0.3px',
          }}
        >
          {title}
        </h3>
        <p
          style={{
            margin: '0 0 24px',
            fontSize: '13px',
            color: 'var(--text-secondary)',
            lineHeight: '1.5',
          }}
        >
          {message}
        </p>
        <div style={{ display: 'flex', gap: '8px', justifyContent: 'flex-end' }}>
          <button ref={cancelRef} className="btn btn-sm" onClick={onCancel}>
            {cancelLabel}
          </button>
          <button
            className={danger ? 'btn btn-sm btn-danger' : 'btn btn-sm btn-accent'}
            onClick={onConfirm}
          >
            {confirmLabel}
          </button>
        </div>
      </div>
    </div>
  );
}
