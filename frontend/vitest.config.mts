import { fileURLToPath } from 'node:url';
import { defineConfig } from 'vitest/config';

// Mirrors the "@/*" path in tsconfig.json so tests can import page modules.
export default defineConfig({
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('.', import.meta.url)),
    },
  },
});
