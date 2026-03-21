import { defineConfig } from 'vite';
import preact from '@preact/preset-vite';

export default defineConfig({
  plugins: [preact()],
  build: {
    outDir: 'dist',
    emptyOutDir: true,
  },
  server: {
    port: 4501,
    proxy: {
      '/api': 'http://localhost:4599',
    },
  },
});
