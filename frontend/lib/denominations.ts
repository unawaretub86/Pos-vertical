// Pure denomination-counting math for the register close (arqueo) flow.
// Framework-free so it can be unit tested without React/DOM.

export const COP_DENOMINATIONS = [
  100000, 50000, 20000, 10000, 5000, 2000, 1000, 500, 200, 100, 50,
] as const;

export type DenominationCounts = Partial<Record<number, number>>;

/** Sum of (denomination value * count) across all COP denominations. */
export function sumDenominations(counts: DenominationCounts): number {
  return COP_DENOMINATIONS.reduce((sum, value) => sum + value * (counts[value] || 0), 0);
}
