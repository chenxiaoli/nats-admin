import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import path from 'node:path';

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: { '@': path.resolve(__dirname, 'src') },
  },
  server: {
    port: 5173,
    host: '0.0.0.0',
    allowedHosts: ['nats-admin-dev.sztitan.com'],
    proxy: {
      '/api': { target: 'http://localhost:8080', ws: true },
    },
  },
});
