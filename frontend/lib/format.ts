/**
 * Centralized COP currency format.
 * Always without decimals, dot as the thousands separator (es-CO).
 * The sign goes before the symbol, and anything that rounds to zero is $0.
 * Example: $1.500  $45.000  -$8.000
 */
export const formatCOP = (n: number): string => {
  const r = Math.round(n);
  const abs = Math.abs(r).toLocaleString('es-CO', { maximumFractionDigits: 0 });
  return r < 0 ? '-$' + abs : '$' + abs;
};

/**
 * Formatea un string de entrada para mostrarlo como COP mientras el usuario escribe.
 * Elimina todo excepto dígitos y aplica formato es-CO.
 */
export const formatInputCOP = (raw: string): string => {
  const digits = raw.replace(/\D/g, '');
  if (!digits) return '';
  return parseInt(digits, 10).toLocaleString('es-CO', { maximumFractionDigits: 0 });
};

/**
 * Parsea un string formateado COP a number.
 * Elimina puntos y caracteres no numéricos.
 */
export const parseInputCOP = (formatted: string): number => {
  const digits = formatted.replace(/\D/g, '');
  return digits ? parseInt(digits, 10) : 0;
};
