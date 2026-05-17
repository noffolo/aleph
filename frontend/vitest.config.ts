import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'
import path from 'path'

export default defineConfig({
  plugins: [react()],
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./src/test-setup.ts'],
    pool: 'forks',
    timeout: 10000,
    coverage: {
      reporter: ['text', 'lcov', 'html'],
      enabled: true,
      exclude: ['api/proto/**', '**/*.pb.*', '**/_grpc.*'],
      thresholds: {
        statements: 48,
        branches: 40,
        functions: 60,
        lines: 48,
      },
    },
    include: [
      'src/**/*.{test,spec}.{ts,tsx}',
      'src/**/__tests__/**/*.{test,spec}.{ts,tsx}',
    ],
  },
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
})
