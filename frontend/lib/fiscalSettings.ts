// US-090: DIAN e-invoicing section of the Settings page.

// Backend route of the fiscal config (internal/handler/handler.go).
export const FISCAL_CONFIG_PATH = '/fiscal/config';

// The section renders only when the saved config has DIAN turned on, so an
// install without e-invoicing never sees it. A config that could not be
// loaded (null) counts as off.
export function showFiscalSection(cfg: { enabled?: boolean } | null): boolean {
  return cfg?.enabled === true;
}
