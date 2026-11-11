'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { api } from '@/lib/api';
import {
  BACKUP_STATUS_PATH,
  describeBackupStatus,
  shouldShowBackupBanner,
  type BackupStatus,
} from '@/lib/backupSettings';

// Admin dashboard warning: shown only when the last backup failed or fell back.
export default function BackupBanner() {
  const [status, setStatus] = useState<BackupStatus | null>(null);

  useEffect(() => {
    api
      .get<BackupStatus>(BACKUP_STATUS_PATH)
      .then(setStatus)
      .catch(() => setStatus(null));
  }, []);

  if (!status || !shouldShowBackupBanner(status)) return null;

  return (
    <div
      role="alert"
      style={{
        padding: '12px 16px',
        borderRadius: '8px',
        fontSize: '13px',
        border: '1px solid var(--border)',
        background: 'var(--surface-2)',
        color: 'var(--text-primary)',
        overflowWrap: 'anywhere',
      }}
    >
      <strong>Copia de seguridad con problemas.</strong> {describeBackupStatus(status)}{' '}
      <Link href="/settings" style={{ textDecoration: 'underline' }}>
        Revisar en Configuración
      </Link>
    </div>
  );
}
