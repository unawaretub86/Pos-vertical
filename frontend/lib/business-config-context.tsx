'use client';

import { createContext, useContext, useState, useEffect, useCallback, ReactNode } from 'react';
import { api } from './api';

interface BusinessConfig {
  name: string;
  logo: string;
  restaurant_mode: boolean;
  print_method: string;
}

interface BusinessConfigCtx {
  config: BusinessConfig;
  loading: boolean;
  invalidate: () => void;
}

const defaultConfig: BusinessConfig = { name: '', logo: '', restaurant_mode: false, print_method: 'browser' };

const BusinessConfigContext = createContext<BusinessConfigCtx>({
  config: defaultConfig,
  loading: true,
  invalidate: () => {},
});

export function BusinessConfigProvider({ children }: { children: ReactNode }) {
  const [config, setConfig] = useState<BusinessConfig>(defaultConfig);
  const [loading, setLoading] = useState(true);
  const [tick, setTick] = useState(0);

  useEffect(() => {
    api.get<BusinessConfig>('/setup')
      .then(cfg => {
        setConfig({ name: cfg.name ?? '', logo: cfg.logo ?? '', restaurant_mode: cfg.restaurant_mode ?? false, print_method: cfg.print_method ?? 'browser' });
        setLoading(false);
      })
      .catch(() => setLoading(false));
  }, [tick]);

  const invalidate = useCallback(() => setTick(t => t + 1), []);

  return (
    <BusinessConfigContext.Provider value={{ config, loading, invalidate }}>
      {children}
    </BusinessConfigContext.Provider>
  );
}

export function useBusinessConfig() {
  return useContext(BusinessConfigContext);
}
