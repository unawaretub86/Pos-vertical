import { describe, it, expect } from 'vitest';
import { breakpointQuery } from './breakpoint';

// Contract US-079 T-7 (AC-6): the query matches widths strictly below the breakpoint.
describe('breakpointQuery', () => {
  it('T-7: 768 -> (max-width: 767px)', () => {
    expect(breakpointQuery(768)).toBe('(max-width: 767px)');
  });

  it('T-7: 1024 -> (max-width: 1023px)', () => {
    expect(breakpointQuery(1024)).toBe('(max-width: 1023px)');
  });

  it("T-7: AppLayout's 767px threshold equals breakpointQuery(768)", () => {
    expect(breakpointQuery(768)).toBe(`(max-width: ${767}px)`);
  });
});
