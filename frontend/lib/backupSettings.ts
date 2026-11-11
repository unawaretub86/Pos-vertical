// Backup folder setting and last-run report (US-089). Pure helpers, unit tested
// without React. Timestamps arrive as RFC 3339 with offset and are shown in the
// shop's zone whatever the browser zone is.

export interface BackupStatus {
  configured_dir: string;
  default_dir: string;
  effective_dir: string;
  last_attempt_at: string | null;
  last_success_at: string | null;
  last_ok: boolean;
  fell_back: boolean;
  last_error: string;
  last_file: string;
}

export const BACKUP_STATUS_PATH = '/backup/status';
export const BACKUP_CONFIG_PATH = '/backup/config';

const SHOP_TIME_ZONE = 'America/Bogota';

const dateTimeFormatter = new Intl.DateTimeFormat('en-CA', {
  timeZone: SHOP_TIME_ZONE,
  year: 'numeric',
  month: '2-digit',
  day: '2-digit',
  hour: '2-digit',
  minute: '2-digit',
  hourCycle: 'h23',
});

/** "dd/mm/yyyy HH:mm" in America/Bogota. */
function formatBackupTime(iso: string): string {
  const parts = dateTimeFormatter.formatToParts(new Date(iso));
  const get = (type: string) => parts.find(p => p.type === type)?.value ?? '';
  return `${get('day')}/${get('month')}/${get('year')} ${get('hour')}:${get('minute')}`;
}

/** True for `D:\x`, `C:/x` and `\\server\share`; trims first. Mirrors the server check. */
export function isAbsoluteWindowsPath(path: string): boolean {
  const p = path.trim();
  return /^[A-Za-z]:[\\/]/.test(p) || /^\\\\[^\\/]+[\\/][^\\/]+/.test(p);
}

export function describeBackupStatus(s: BackupStatus): string {
  if (!s.last_attempt_at) return 'Aún no se ha hecho ninguna copia.';

  if (!s.last_ok) {
    const lastGood = s.last_success_at
      ? ` Última copia buena: ${formatBackupTime(s.last_success_at)}.`
      : ' Todavía no hay ninguna copia buena.';
    return `La última copia falló (${formatBackupTime(s.last_attempt_at)}): ${s.last_error}.${lastGood}`;
  }

  const when = formatBackupTime(s.last_success_at ?? s.last_attempt_at);
  if (s.fell_back) {
    // The server prefixes the reason with the failing folder; show it once.
    const reason = s.last_error.startsWith(`${s.configured_dir}: `)
      ? s.last_error.slice(s.configured_dir.length + 2)
      : s.last_error;
    return `No se pudo usar la carpeta ${s.configured_dir}: ${reason}. La copia del ${when} se guardó en ${s.default_dir}.`;
  }
  return `Última copia: ${when} en ${s.effective_dir}`;
}

/** The admin dashboard shows a banner only when the last backup failed or fell back. */
export function shouldShowBackupBanner(s: BackupStatus): boolean {
  return s.last_attempt_at !== null && (s.fell_back || !s.last_ok);
}
