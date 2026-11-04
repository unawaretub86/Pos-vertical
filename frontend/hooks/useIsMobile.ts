'use client';

import { useCallback, useSyncExternalStore } from 'react';
import { breakpointQuery } from '@/lib/breakpoint';

export function useIsMobile(breakpoint = 768): boolean {
  const query = breakpointQuery(breakpoint);

  const subscribe = useCallback((onChange: () => void) => {
    const mq = window.matchMedia(query);
    mq.addEventListener('change', onChange);
    return () => mq.removeEventListener('change', onChange);
  }, [query]);

  const getSnapshot = useCallback(() => window.matchMedia(query).matches, [query]);

  return useSyncExternalStore(subscribe, getSnapshot, () => false);
}
