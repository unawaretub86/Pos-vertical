// Media query matching widths strictly below the given breakpoint.
export function breakpointQuery(breakpoint: number): string {
  return `(max-width: ${breakpoint - 1}px)`;
}
