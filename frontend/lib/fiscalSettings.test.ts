// US-090 contract section 4, T-8 (AC-9, AC-10).
import { describe, it, expect } from 'vitest';
import { FISCAL_CONFIG_PATH, showFiscalSection } from './fiscalSettings';

describe('fiscalSettings', () => {
  it('points at the backend route /fiscal/config', () => {
    expect(FISCAL_CONFIG_PATH).toBe('/fiscal/config');
  });

  it('hides the section when the config could not be loaded', () => {
    expect(showFiscalSection(null)).toBe(false);
  });

  it('hides the section when DIAN is off', () => {
    expect(showFiscalSection({ enabled: false })).toBe(false);
  });

  it('shows the section when DIAN is on', () => {
    expect(showFiscalSection({ enabled: true })).toBe(true);
  });
});
