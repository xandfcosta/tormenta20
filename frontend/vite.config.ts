import path from 'node:path'
import tailwindcss from '@tailwindcss/vite'
import { tanstackRouter } from '@tanstack/router-plugin/vite'
import solid from 'vite-plugin-solid'
// vitest/config, not vite: it's the one whose config type knows `test`.
import { defineConfig } from 'vitest/config'

// The Go server (cmd/api :3001) is the backend. During the migration this app
// ran on :5174 beside the React one; since the cutover (ALE-76) it owns the
// canonical :5173, which is what the E2E suite and the docs point at.
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
    port: 5173,
    proxy: {
      '/api': {
        target: API_TARGET,
        changeOrigin: true,
        rewrite: (p) => p.replace(/^\/api/, ''),
      },
      // O piloto Datastar (ALE-219): uma página servida pelo GO, não por este
      // servidor. Sem proxy ela só abriria em :3001 — o cookie funcionaria
      // (cookie ignora porta), mas as fontes self-hosted de `public/` não, e o
      // link vindo da SPA sairia da origem. Apagar esta entrada é a outra
      // metade da saída do piloto.
      '/piloto': { target: API_TARGET, changeOrigin: true },
      // A raiz EXATA vai ao Go, que a desvia para o Hub (ALE-231). Chave em
      // regex porque prefixo `'/'` casaria com o app inteiro. Sem isto o
      // desvio só existiria em produção, e o e2e mediria outra coisa.
      '^/$': { target: API_TARGET, changeOrigin: true },
    },
  },
  /**
   * `vite preview` serves the BUILT `dist/` and has its own proxy config —
   * `server.proxy` above does not reach it. Without this block every `/api`
   * call 404s against the preview server itself, the screen loads empty, and
   * the production build cannot be inspected or measured at all.
   *
   * Not a production concern (there a static host/CDN serves `dist/` and the
   * deploy routes the API) — this exists so a local production build behaves
   * like one. Measuring in DEV is what produced the phantom "React blocks
   * 64–74ms" that did not survive a production run (ALE-76).
   */
  preview: {
    port: 5173,
    proxy: {
      '/api': {
        target: API_TARGET,
        changeOrigin: true,
        rewrite: (p) => p.replace(/^\/api/, ''),
      },
      // O piloto Datastar (ALE-219): uma página servida pelo GO, não por este
      // servidor. Sem proxy ela só abriria em :3001 — o cookie funcionaria
      // (cookie ignora porta), mas as fontes self-hosted de `public/` não, e o
      // link vindo da SPA sairia da origem. Apagar esta entrada é a outra
      // metade da saída do piloto.
      '/piloto': { target: API_TARGET, changeOrigin: true },
      // A raiz EXATA vai ao Go, que a desvia para o Hub (ALE-231). Chave em
      // regex porque prefixo `'/'` casaria com o app inteiro. Sem isto o
      // desvio só existiria em produção, e o e2e mediria outra coisa.
      '^/$': { target: API_TARGET, changeOrigin: true },
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
