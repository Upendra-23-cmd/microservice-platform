import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import path from 'path';

export default defineConfig({
  plugins: [react()],

  resolve: {
    alias: {
      '@':            path.resolve(__dirname, './src'),
      '@components':  path.resolve(__dirname, './src/components'),
      '@hooks':       path.resolve(__dirname, './src/hooks'),
      '@services':    path.resolve(__dirname, './src/services'),
      '@store':       path.resolve(__dirname, './src/store'),
      '@types':       path.resolve(__dirname, './src/types'),
      '@utils':       path.resolve(__dirname, './src/utils'),
    },
  },

  server: {
    port: 3000,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },

  build: {
    target: 'es2022',
    outDir: 'dist',
    sourcemap: false, // enable in staging; disable in prod for security
    rollupOptions: {
      output: {
        manualChunks: {
          vendor:  ['react', 'react-dom', 'react-router-dom'],
          query:   ['@tanstack/react-query'],
          charts:  ['recharts'],
          forms:   ['react-hook-form', 'zod', '@hookform/resolvers'],
          store:   ['zustand'],
        },
      },
    },
    chunkSizeWarningLimit: 600,
  },

  preview: {
    port: 3000,
  },
});
