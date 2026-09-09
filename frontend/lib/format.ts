/**
 * Formato de moneda COP — centralizado.
 * Siempre sin decimales, separador de miles con punto (es-CO).
 * Ejemplo: $1.500  $45.000  $1.245.000
 */
export const formatCOP = (n: number): string =>
  '$' + Math.round(n).toLocaleString('es-CO', { maximumFractionDigits: 0 });

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
