'use client';

import { useRef, useState, type CSSProperties } from 'react';
import { api } from '@/lib/api';
import { useToast } from '@/components/Toast';
import {
  IMPORT_FIELDS,
  TEMPLATE_FILE_NAME,
  buildMappingPayload,
  buildTemplateCSV,
  bytesToBase64,
  canCommit,
  moreErrorsText,
  readHeaderLine,
  type ImportReport,
  type ImportResult,
} from '@/lib/productImport';

interface ProductImportModalProps {
  onClose: () => void;
  // Called after a committed import so the page reloads products and categories.
  onImported: () => void;
}

const MISSING_COLUMNS = 'faltan columnas obligatorias';

function downloadTemplate() {
  const blob = new Blob([buildTemplateCSV()], { type: 'text/csv;charset=utf-8;' });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = TEMPLATE_FILE_NAME;
  a.click();
  URL.revokeObjectURL(url);
}

function emptyMapping(): Record<string, string> {
  return Object.fromEntries(IMPORT_FIELDS.map(f => [f.key, '']));
}

// Catalog import (US-094): pick a CSV, dry-run it, adjust the column mapping,
// then commit. Only rendered for admins (canImportProducts).
export default function ProductImportModal({ onClose, onImported }: ProductImportModalProps) {
  const { toast } = useToast();
  const [fileName, setFileName] = useState('');
  const [content, setContent] = useState('');
  const [headers, setHeaders] = useState<string[]>([]);
  const [mapping, setMapping] = useState<Record<string, string>>(emptyMapping);
  // Mapping sent in the last dry run (null = automatic); the commit reuses it.
  const [sentMapping, setSentMapping] = useState<Record<string, string> | null>(null);
  const [dirty, setDirty] = useState(false);
  const [report, setReport] = useState<ImportReport | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState<'checking' | 'importing' | null>(null);
  // Guards a second tap before the `busy` re-render lands.
  const sentRef = useRef(false);

  async function check(b64: string, name: string, explicit: Record<string, string> | null, bytes?: Uint8Array) {
    setBusy('checking');
    setError(null);
    setReport(null);
    try {
      const r = await api.post<ImportReport>('/products/import', {
        file_name: name, content_base64: b64, mapping: explicit, dry_run: true,
      });
      setReport(r);
      setHeaders(r.headers ?? []);
      setMapping({ ...emptyMapping(), ...(r.mapping ?? {}) });
      setSentMapping(explicit);
      setDirty(false);
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Error revisando el archivo';
      setError(msg);
      // Required columns not recognized: offer the manual mapping with the
      // header row read here, since the backend sent no report.
      if (bytes && msg.startsWith(MISSING_COLUMNS)) {
        setHeaders(readHeaderLine(bytes));
        setMapping(emptyMapping());
      }
    } finally {
      setBusy(null);
    }
  }

  async function handleFile(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    if (!file) return;
    const bytes = new Uint8Array(await file.arrayBuffer());
    const b64 = bytesToBase64(bytes);
    setFileName(file.name);
    setContent(b64);
    setHeaders([]);
    setMapping(emptyMapping());
    await check(b64, file.name, null, bytes);
  }

  async function handleCommit() {
    if (!report || sentRef.current) return;
    sentRef.current = true;
    setBusy('importing');
    setError(null);
    try {
      const res = await api.post<ImportResult>('/products/import', {
        file_name: fileName, content_base64: content, mapping: sentMapping, dry_run: false,
      });
      toast(`Se importaron ${res.products_created} productos`, 'success');
      onImported();
      onClose();
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Error importando productos');
    } finally {
      sentRef.current = false;
      setBusy(null);
    }
  }

  const shownErrors = report?.errors ?? [];
  const commitEnabled = !!report && canCommit(report) && !dirty && busy === null;

  return (
    <div style={overlayStyle} onClick={e => e.target === e.currentTarget && busy === null && onClose()}>
      <div role="dialog" aria-modal="true" aria-labelledby="product-import-title" style={panelStyle}>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: '8px' }}>
          <div id="product-import-title" style={{ fontSize: '15px', fontWeight: 600, color: 'var(--text-primary)' }}>Importar productos</div>
          <button type="button" className="btn btn-icon" onClick={onClose} disabled={busy !== null} aria-label="Cerrar" style={{ color: 'var(--text-muted)' }}>
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/>
            </svg>
          </button>
        </div>

        <div style={{ fontSize: '13px', color: 'var(--text-secondary)', lineHeight: 1.5 }}>
          Sube un archivo CSV. Desde Excel: Archivo › Guardar como › CSV UTF-8.{' '}
          <button type="button" onClick={downloadTemplate} style={linkStyle}>Descargar plantilla</button>
        </div>

        <input
          type="file"
          accept=".csv,.txt,text/csv"
          onChange={handleFile}
          disabled={busy !== null}
          className="input"
          style={{ maxWidth: '100%', minWidth: 0 }}
        />

        {busy === 'checking' && <div style={{ fontSize: '13px', color: 'var(--text-secondary)' }}>Revisando archivo...</div>}

        {error && (
          <div style={{ fontSize: '13px', color: 'var(--danger)', padding: '8px 12px', background: 'var(--danger-dim)', borderRadius: '6px', overflowWrap: 'anywhere' }}>
            {error}
          </div>
        )}

        {headers.length > 0 && (
          <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
            <div style={sectionTitleStyle}>Columnas</div>
            <div style={scrollStyle}>
              <table style={tableStyle}>
                <tbody>
                  {IMPORT_FIELDS.map(f => (
                    <tr key={f.key}>
                      <td style={{ ...tdStyle, whiteSpace: 'nowrap', color: 'var(--text-primary)' }}>
                        {f.label}{f.required ? ' *' : ''}
                      </td>
                      <td style={tdStyle}>
                        <select
                          className="input"
                          value={mapping[f.key] ?? ''}
                          onChange={e => { setMapping(m => ({ ...m, [f.key]: e.target.value })); setDirty(true); }}
                          disabled={busy !== null}
                          style={{ cursor: 'pointer', minWidth: '140px' }}
                        >
                          <option value="">(no importar)</option>
                          {headers.filter(h => h !== '').map((h, i) => <option key={`${h}-${i}`} value={h}>{h}</option>)}
                        </select>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            <div>
              <button
                type="button"
                className="btn btn-sm"
                disabled={busy !== null}
                onClick={() => check(content, fileName, buildMappingPayload(mapping))}
              >
                Revisar de nuevo
              </button>
            </div>
          </div>
        )}

        {report && !report.valid && (
          <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
            <div style={sectionTitleStyle}>Errores</div>
            <div style={{ ...scrollStyle, maxHeight: '240px', overflowY: 'auto' }}>
              <table style={tableStyle}>
                <thead>
                  <tr>
                    <th style={thStyle}>Fila</th>
                    <th style={thStyle}>Columna</th>
                    <th style={thStyle}>Error</th>
                  </tr>
                </thead>
                <tbody>
                  {shownErrors.map((er, i) => (
                    <tr key={`${er.row}-${er.field ?? ''}-${i}`}>
                      <td style={{ ...tdStyle, whiteSpace: 'nowrap' }} className="tnum">{er.row}</td>
                      <td style={{ ...tdStyle, whiteSpace: 'nowrap' }}>{er.column || '—'}</td>
                      <td style={{ ...tdStyle, color: 'var(--danger)', minWidth: '200px' }}>{er.message}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            {moreErrorsText(report.error_count, shownErrors.length) && (
              <div style={{ fontSize: '12px', color: 'var(--text-secondary)' }}>{moreErrorsText(report.error_count, shownErrors.length)}</div>
            )}
          </div>
        )}

        {report && report.valid && (
          <div style={{ fontSize: '13px', color: 'var(--text-primary)', display: 'flex', flexDirection: 'column', gap: '4px' }}>
            <div style={{ color: 'var(--success)', fontWeight: 600 }}>{report.valid_rows} productos listos para importar</div>
            {report.new_categories.length > 0 && (
              <div style={{ color: 'var(--text-secondary)' }}>Categorías nuevas: {report.new_categories.join(', ')}</div>
            )}
          </div>
        )}

        <div style={{ display: 'flex', gap: '8px', justifyContent: 'flex-end', flexWrap: 'wrap' }}>
          <button type="button" className="btn" onClick={onClose} disabled={busy !== null}>Cancelar</button>
          <button type="button" className="btn btn-accent" onClick={handleCommit} disabled={!commitEnabled}>
            {busy === 'importing' ? 'Importando...' : `Importar ${report?.valid ? report.valid_rows : 0} productos`}
          </button>
        </div>
      </div>
    </div>
  );
}

const overlayStyle: CSSProperties = {
  position: 'fixed', inset: 0, zIndex: 1000,
  background: 'color-mix(in srgb, var(--canvas) 75%, transparent)',
  display: 'flex', alignItems: 'center', justifyContent: 'center',
  padding: '8px',
};

const panelStyle: CSSProperties = {
  background: 'var(--surface-1)',
  border: '1px solid var(--border)',
  borderRadius: '12px',
  padding: '20px',
  width: '640px',
  maxWidth: '100%',
  maxHeight: '92vh',
  overflowY: 'auto',
  overflowX: 'hidden',
  boxSizing: 'border-box',
  display: 'flex',
  flexDirection: 'column',
  gap: '16px',
};

const linkStyle: CSSProperties = {
  background: 'none', border: 'none', padding: 0, cursor: 'pointer',
  color: 'var(--accent)', textDecoration: 'underline', font: 'inherit',
};

const sectionTitleStyle: CSSProperties = {
  fontSize: '12px', fontWeight: 600, color: 'var(--text-secondary)',
  textTransform: 'uppercase', letterSpacing: '0.4px',
};

const scrollStyle: CSSProperties = {
  overflowX: 'auto', maxWidth: '100%',
  border: '1px solid var(--border)', borderRadius: '8px',
};

const tableStyle: CSSProperties = { width: '100%', borderCollapse: 'collapse', fontSize: '13px' };

const thStyle: CSSProperties = {
  textAlign: 'left', padding: '8px 10px', fontSize: '11px', fontWeight: 600,
  color: 'var(--text-muted)', borderBottom: '1px solid var(--border)', whiteSpace: 'nowrap',
  background: 'var(--surface-2)',
};

const tdStyle: CSSProperties = {
  padding: '6px 10px', borderBottom: '1px solid var(--border)', color: 'var(--text-secondary)',
  verticalAlign: 'middle',
};
