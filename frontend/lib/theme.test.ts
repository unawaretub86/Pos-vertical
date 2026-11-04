import { describe, it, expect } from 'vitest';
import { readTheme } from './theme';

// Contract US-079 T-6 (AC-5): only 'light' yields light; everything else is dark.
describe('readTheme', () => {
  it('T-6: null (nothing stored) -> dark', () => {
    expect(readTheme(null)).toBe('dark');
  });

  it("T-6: 'light' -> light", () => {
    expect(readTheme('light')).toBe('light');
  });

  it("T-6: 'dark' -> dark", () => {
    expect(readTheme('dark')).toBe('dark');
  });

  it("T-6: unrecognized string 'garbage' -> dark", () => {
    expect(readTheme('garbage')).toBe('dark');
  });

  it('T-6: empty string -> dark', () => {
    expect(readTheme('')).toBe('dark');
  });
});
