import path from 'node:path'
import tailwindcss from '@tailwindcss/vite'
import { tanstackRouter } from '@tanstack/router-plugin/vite'
import solid from 'vite-plugin-solid'
// vitest/config, not vite: it's the one whose config type knows `test`.
import { defineConfig } from 'vitest/config'

// Same backend the React app talks to: the Go server (cmd/api :3001). Both
// frontends can run at once — this one owns :5174 — so a ported scene can be
// A/B'd against its React original without stopping either.
const API_TARGET = process.env.API_TARGET ?? 'http://localhost:3001'

export default defineConfig({
  plugins: [
    // Must precede solid() so the generated route tree gets the Solid transform.
    tanstackRouter({ target: 'solid', autoCodeSplitting: true }),
    solid(),
    tailwindcss(),
  ],
  resolve: {
    alias: { '@': path.resolve(__dirname, './src') },
  },
  server: {
    port: 5174,
    proxy: {
      '/api': {
        target: API_TARGET,
        changeOrigin: true,
        rewrite: (p) => p.replace(/^\/api/, ''),
      },
      // The realtime client connects to window.location.origin; socket.io lives
      // on the same backend, so proxy its path with the WebSocket upgrade on.
      '/socket.io': { target: API_TARGET, changeOrigin: true, ws: true },
    },
  },
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['src/test-setup.ts'],
    // vite-plugin-solid needs Solid's client/test transform, not the SSR one —
    // without inlining, vitest loads the pre-built server bundle and every
    // render() throws (spike ALE-62).
    server: { deps: { inline: [/solid-js/, /@solidjs/, /@tanstack/] } },
  },
})
