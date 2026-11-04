export type Theme = 'dark' | 'light';

// Only 'light' is recognized; anything else (null, '', unknown strings) falls back to 'dark'.
export function readTheme(raw: string | null): Theme {
  return raw === 'light' ? 'light' : 'dark';
}
