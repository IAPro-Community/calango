import { fileURLToPath } from 'node:url';
import react from '@vitejs/plugin-react';
import { defineConfig } from 'vitest/config';

// Renderer component tests (issue #118). jsdom stands in for the Electron window; the
// SDK client and the preload bridge are typed fakes (src/renderer/test).
export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: { '@': fileURLToPath(new URL('./src/lib', import.meta.url)) },
  },
  test: {
    environment: 'jsdom',
    include: ['src/renderer/**/*.test.{ts,tsx}', 'src/lib/**/*.test.{ts,tsx}'],
    setupFiles: ['src/renderer/test/setup.ts'],
    restoreMocks: true,
  },
});
