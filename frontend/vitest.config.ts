import path from 'node:path'
import { defineConfig } from 'vitest/config'

/**
 * Standalone Vitest config to avoid pulling in the Vite app plugins
 * (router codegen, Tailwind) during unit-test runs. Tests target pure
 * helpers + zustand stores; a happy-dom env is enough for the few that
 * touch localStorage via zustand/persist.
 */
export default defineConfig({
  resolve: {
    alias: { '@': path.resolve(__dirname, './src') },
  },
  test: {
    environment: 'jsdom',
    include: ['src/**/*.test.ts'],
    setupFiles: ['src/test-setup.ts'],
  },
})
