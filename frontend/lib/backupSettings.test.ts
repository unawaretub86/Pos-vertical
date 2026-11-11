// US-089 contract section 4, F-1 and F-2 (AC-13, AC-16).
//
// Helper API assumed from the contract (names and shapes not given):
//   isAbsoluteWindowsPath(path: string): boolean            // trims first
//   interface BackupStatus { configured_dir, default_dir, effective_dir: string;
//     last_attempt_at: string | null; last_success_at: string | null; last_ok: boolean;
//     fell_back: boolean; last_error: string; last_file: string }
//   describeBackupStatus(s: BackupStatus): string           // Spanish sentence
//   shouldShowBackupBanner(s: BackupStatus): boolean
// Timestamps arrive as RFC 3339 with offset and are rendered dd/mm/yyyy HH:mm in America/Bogota.
import { describe, it, expect } from 'vitest';
import {
  isAbsoluteWindowsPath,
  describeBackupStatus,
  shouldShowBackupBanner,
  type BackupStatus,
} from './backupSettings';

describe('isAbsoluteWindowsPath', () => {
  it.each([
    ['D:\\Respaldos', true],
    ['C:/POS/copias', true],
    ['\\\\PC\\share', true],
    ['Respaldos', false],
    ['D:Respaldos', false],
    ['/tmp', false],
    ['  D:\\Respaldos  ', true],
  ])('%j -> %s', (path, want) => {
    expect(isAbsoluteWindowsPath(path)).toBe(want);
  });
});

const never: BackupStatus = {
  configured_dir: '',
  default_dir: 'C:\\POS',
  effective_dir: '',
  last_attempt_at: null,
  last_success_at: null,
  last_ok: false,
  fell_back: false,
  last_error: '',
  last_file: '',
};

const ok: BackupStatus = {
  configured_dir: 'D:\\Respaldos',
  default_dir: 'C:\\POS',
  effective_dir: 'D:\\Respaldos',
  last_attempt_at: '2026-10-31T21:05:00-05:00',
  last_success_at: '2026-10-31T21:05:00-05:00',
  last_ok: true,
  fell_back: false,
  last_error: '',
  last_file: 'D:\\Respaldos\\data.db.backup-20261031',
};

const fellBack: BackupStatus = {
  ...ok,
  effective_dir: 'C:\\POS',
  fell_back: true,
  last_error: 'La carpeta no existe',
  last_file: 'C:\\POS\\data.db.backup-20261031',
};

const failed: BackupStatus = {
  ...ok,
  last_ok: false,
  last_error: 'disco lleno',
};

describe('describeBackupStatus', () => {
  it('says no copy was made yet when there was no attempt', () => {
    expect(describeBackupStatus(never)).toBe('Aún no se ha hecho ninguna copia.');
  });

  it('reports the last good copy with date, time and folder', () => {
    expect(describeBackupStatus(ok)).toBe('Última copia: 31/10/2026 21:05 en D:\\Respaldos');
  });

  it('explains the fallback with the failing folder, the reason and the default folder', () => {
    const text = describeBackupStatus(fellBack);
    expect(text).toContain('No se pudo usar la carpeta D:\\Respaldos');
    expect(text).toContain('La carpeta no existe');
    expect(text).toContain('C:\\POS');
  });
});

describe('shouldShowBackupBanner', () => {
  it('is false when no backup has ever run', () => {
    expect(shouldShowBackupBanner(never)).toBe(false);
  });
  it('is false after a clean backup', () => {
    expect(shouldShowBackupBanner(ok)).toBe(false);
  });
  it('is true when the last backup fell back', () => {
    expect(shouldShowBackupBanner(fellBack)).toBe(true);
  });
  it('is true when the last backup failed', () => {
    expect(shouldShowBackupBanner(failed)).toBe(true);
  });
});
