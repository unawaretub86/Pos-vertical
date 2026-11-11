'use client';

import { useEffect, useState } from 'react';
import { api } from '@/lib/api';
import { useToast } from '@/components/Toast';
import {
  BACKUP_CONFIG_PATH,
  BACKUP_STATUS_PATH,
  describeBackupStatus,
  isAbsoluteWindowsPath,
  type BackupStatus,
} from '@/lib/backupSettings';

// Settings section "Copias de seguridad" (US-089). It loads the admin-only
// status endpoint, so for any other user it renders nothing.
export default function BackupSection() {
  const { toast } = useToast();
  const [status, setStatus] = useState<BackupStatus | null>(null);
  const [dir, setDir] = useState('');
  const [error, setError] = useState('');
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    api
      .get<BackupStatus>(BACKUP_STATUS_PATH)
      .then(s => {
        setStatus(s);
        setDir(s.configured_dir);
      })
      .catch(() => setStatus(null));
  }, []);

  if (!status) return null;

  const warn = status.last_attempt_at !== null && (status.fell_back || !status.last_ok);

  async function handleSave() {
    const value = dir.trim();
    if (value !== '' && !isAbsoluteWindowsPath(value)) {
      setError('Escriba la ruta completa de la carpeta, por ejemplo D:\\Respaldos');
      return;
    }
    setError('');
    setSaving(true);
    try {
      const s = await api.put<BackupStatus>(BACKUP_CONFIG_PATH, { dir: value });
      setStatus(s);
      setDir(s.configured_dir);
      toast('Carpeta de copias guardada', 'success');
    } catch (e) {
      setError(e instanceof Error ? e.message : 'No se pudo guardar la carpeta');
    } finally {
      setSaving(false);
    }
  }

  return (
    <div>
      <div style={{ fontSize: '11px', fontWeight: 600, textTransform: 'uppercase', letterSpacing: '0.5px', color: 'var(--text-muted)', marginBottom: '10px' }}>
        Copias de seguridad
      </div>
      <div style={{
        padding: '12px 16px',
        background: 'var(--surface-1)',
        border: '1px solid var(--border)',
        borderRadius: '8px',
        display: 'flex',
        flexDirection: 'column',
        gap: '10px',
      }}>
        <div>
          <div style={{ fontSize: '13px', fontWeight: 500, color: 'var(--text-primary)' }}>Carpeta de las copias</div>
          <div style={{ fontSize: '11px', color: 'var(--text-muted)', marginTop: '2px' }}>
            Se guarda una copia por día y se conservan las últimas 30. Déjela vacía para guardarlas junto al programa.
          </div>
        </div>
        <div style={{ display: 'flex', flexWrap: 'wrap', gap: '8px' }}>
          <input
            className="input"
            type="text"
            aria-label="Carpeta de las copias"
            placeholder="D:\Respaldos"
            value={dir}
            onChange={e => setDir(e.target.value)}
            style={{ flex: '1 1 200px', minWidth: 0 }}
          />
          <button type="button" className="btn btn-sm" disabled={saving} onClick={handleSave}>
            {saving ? 'Guardando...' : 'Guardar'}
          </button>
        </div>
        {error && (
          <div role="alert" style={{ fontSize: '12px', color: 'var(--text-primary)' }}>{error}</div>
        )}
        <div
          role={warn ? 'alert' : undefined}
          style={{
            fontSize: '12px',
            color: warn ? 'var(--text-primary)' : 'var(--text-secondary)',
            background: warn ? 'var(--surface-2)' : 'transparent',
            border: warn ? '1px solid var(--border)' : 'none',
            borderRadius: '6px',
            padding: warn ? '8px 10px' : 0,
            overflowWrap: 'anywhere',
          }}
        >
          {warn && <strong>Atención: </strong>}
          {describeBackupStatus(status)}
        </div>
        <div style={{ fontSize: '11px', color: 'var(--text-muted)', overflowWrap: 'anywhere' }}>
          Carpeta en uso: {status.effective_dir || status.default_dir}
        </div>
      </div>
    </div>
  );
}
