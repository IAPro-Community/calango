import { fileURLToPath } from 'node:url';
import tailwindcss from '@tailwindcss/vite';
import react from '@vitejs/plugin-react';
import { defineConfig } from 'vite';

export default defineConfig({
  base: './',
  plugins: [react(), tailwindcss()],
  resolve: {
    // "@" is the UI kit in src/lib, kept apart from the renderer so it can become a package.
    alias: { '@': fileURLToPath(new URL('./src/lib', import.meta.url)) },
  },
  build: {
    outDir: 'dist/renderer',
    emptyOutDir: true,
    target: 'es2022',
  },
});
