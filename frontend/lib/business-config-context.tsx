'use client';

import { createContext, useContext, useState, useEffect, useCallback, ReactNode } from 'react';
import { api } from './api';

interface BusinessConfig {
  name: string;
  logo: string;
  restaurant_mode: boolean;
}

interface BusinessConfigCtx {
  config: BusinessConfig;
  invalidate: () => void;
}

const defaultConfig: BusinessConfig = { name: '', logo: '', restaurant_mode: false };

const BusinessConfigContext = createContext<BusinessConfigCtx>({
  config: defaultConfig,
  invalidate: () => {},
});

export function BusinessConfigProvider({ children }: { children: ReactNode }) {
  const [config, setConfig] = useState<BusinessConfig>(defaultConfig);
  const [tick, setTick] = useState(0);

  useEffect(() => {
    api.get<BusinessConfig>('/setup')
      .then(cfg => setConfig({ name: cfg.name ?? '', logo: cfg.logo ?? '', restaurant_mode: cfg.restaurant_mode ?? false }))
      .catch(() => {});
  }, [tick]);

  const invalidate = useCallback(() => setTick(t => t + 1), []);

  return (
    <BusinessConfigContext.Provider value={{ config, invalidate }}>
      {children}
    </BusinessConfigContext.Provider>
  );
}

export function useBusinessConfig() {
  return useContext(BusinessConfigContext);
}
