import path from 'node:path'
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import { tanstackRouter } from '@tanstack/router-plugin/vite'

/**
 * Manual chunk boundaries. Rolldown otherwise concatenates every
 * vendor lib plus t20-data catalogs into a monolithic index chunk
 * shipped on every route load (500KB+). Splitting keeps first paint
 * cheap and lets browser-cached vendor chunks survive across
 * unrelated app deploys.
 */
function chunkFor(id: string): string | undefined {
  if (!id.includes('node_modules')) {
    // App-side split: the big data catalogs get their own chunks so
    // routes that don't touch them (login, session tracker) don't pull
    // 200KB+ of spell/monster tables.
    if (id.includes('t20-data/src/spell-catalog')) return 't20-spells'
    if (id.includes('t20-data/src/bestiary')) return 't20-bestiary'
    if (id.includes('t20-data')) {
      // Group the remaining 120+ t20-data modules by concern so no
      // single chunk becomes another 500KB monolith.
      if (
        id.includes('t20-data/src/wondrous-items') ||
        id.includes('t20-data/src/specific-magic-items') ||
        id.includes('t20-data/src/consumable-magic-items') ||
        id.includes('t20-data/src/magic-items') ||
        id.includes('t20-data/src/loot') ||
        id.includes('t20-data/src/reward-tables') ||
        id.includes('t20-data/src/magic-potion-tables') ||
        id.includes('t20-data/src/armor-enchants') ||
        id.includes('t20-data/src/weapon-enchants')
      ) {
        return 't20-items'
      }
      if (id.includes('skill-usages') || id.includes('skill-index')) {
        return 't20-skills'
      }
      if (id.includes('power-mechanics') || id.includes('parceiro') || id.includes('divine-power')) {
        return 't20-classes'
      }
      if (
        id.includes('t20-data/src/racas') ||
        id.includes('t20-data/src/origens') ||
        id.includes('t20-data/src/character-sheet') ||
        id.includes('habilidades-gerais') ||
        id.includes('deusiades') ||
        id.includes('deuses') ||
        id.includes('t20-data/src/classes')
      ) {
        return 't20-sheet-refs'
      }
      if (id.includes('t20-data/src/abilities/')) {
        return 't20-abilities'
      }
      if (
        id.includes('conditions') ||
        id.includes('combat') ||
        id.includes('maneuvers') ||
        id.includes('test-resolution') ||
        id.includes('spell-') ||
        id.includes('t20-data/src/spells')
      ) {
        return 't20-rules'
      }
      return 't20-data-core'
    }
    return undefined
  }
  // Vendor chunks. Order matters — first prefix that matches wins.
  if (id.includes('@tanstack/react-router')) return 'vendor-router'
  if (id.includes('@tanstack/react-query')) return 'vendor-query'
  if (
    id.includes('@tanstack/react-form') ||
    id.includes('@tanstack/react-pacer')
  ) {
    return 'vendor-tanstack'
  }
  if (id.includes('radix-ui') || id.includes('@radix-ui')) return 'vendor-radix'
  if (id.includes('socket.io-client') || id.includes('engine.io')) {
    return 'vendor-socketio'
  }
  if (id.includes('lucide-react')) return 'vendor-icons'
  if (
    id.includes('zod') ||
    id.includes('class-variance-authority') ||
    id.includes('clsx') ||
    id.includes('tailwind-merge')
  ) {
    return 'vendor-utils'
  }
  if (id.includes('react-dom') || id.includes('scheduler')) {
    return 'vendor-react-dom'
  }
  if (id.match(/[\\/]node_modules[\\/]react[\\/]/)) return 'vendor-react'
  return 'vendor-misc'
}

export default defineConfig({
  plugins: [
    tanstackRouter({ target: 'react', autoCodeSplitting: true }),
    react(),
    tailwindcss(),
  ],
  resolve: {
    alias: { '@': path.resolve(__dirname, './src') },
  },
  build: {
    rolldownOptions: {
      output: {
        advancedChunks: {
          groups: [
            {
              name: (id: string) => chunkFor(id) ?? '',
              test: (id: string) => chunkFor(id) !== undefined,
            },
          ],
        },
      },
    },
  },
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://localhost:3000',
        changeOrigin: true,
        rewrite: (p) => p.replace(/^\/api/, ''),
      },
      // The realtime client connects to window.location.origin (dev:
      // the Vite port). socket.io lives on the Nest server at :3000 —
      // proxy its path with `ws: true` so the WebSocket upgrade + polling
      // fallback both reach the backend. Without this the socket targets
      // the Vite port (no socket.io handler) and never connects.
      '/socket.io': {
        target: 'http://localhost:3000',
        changeOrigin: true,
        ws: true,
      },
    },
  },
})
