import { defineConfig, devices } from '@playwright/test'

/**
 * E2E suite (ALE-68). Lives OUTSIDE `frontend/` on purpose: it drives the running
 * app by URL, so it's **framework-agnostic** and survives the React→Solid
 * migration (ALE-63) unchanged — unlike the React-coupled unit tests.
 *
 * Needs BOTH dev servers up: the frontend (Vite :5173) and the Go API (:3001)
 * that Vite proxies `/api` to. `webServer` reuses an already-running Vite; the
 * Go backend is assumed up (CI must start it too).
 */
const BASE_URL = process.env.E2E_BASE_URL ?? 'http://localhost:5173'

export default defineConfig({
  testDir: './tests',
  timeout: 30_000,
  expect: { timeout: 7_000 },
  fullyParallel: true,
  // Um worker por padrão na máquina de dev, dois no CI.
  //
  // Histórico: o padrão do Playwright (metade dos núcleos → 4 aqui) saturava e
  // dava timeout PURO, com pico de 32,3s contra o teto de 30s; capar em 2 saiu
  // MAIS rápido (1,9 min contra 3,0), que é o sintoma clássico de
  // sobrescrição (ALE-93). Só que a máquina do dono ficou mais cheia desde
  // então — dev server, API, o browser dele, o browser da automação — e a
  // suíte passou a TRAVAR o laptop. Um worker troca ~2 min de relógio por uma
  // máquina que continua usável enquanto a suíte roda, e quem quiser o
  // paralelismo de volta passa E2E_WORKERS=2.
  workers: Number(process.env.E2E_WORKERS ?? (process.env.CI ? 2 : 1)),
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  reporter: [['list'], ['html', { open: 'never' }]],
  use: {
    baseURL: BASE_URL,
    // `retain-on-failure` e NÃO `on-first-retry` (ALE-244). O segundo parece o
    // padrão sensato e aqui era o mesmo que desligar: fora do CI `retries` é
    // ZERO, então não há primeira retentativa e trace nunca era gravado —
    // `e2e/test-results` sequer existia em disco. Duas sessões passaram um dia
    // caçando um flake de 2/8 com o instrumento desligado, e a assinatura em
    // que as duas se apoiaram ("é sempre na terceira") acabou não sendo
    // verificável: não havia segunda amostra em lugar nenhum.
    //
    // O `retries: 0` local continua certo e é de propósito — retentativa
    // ESCONDE flake, que é o que se está caçando. O que estava errado era só o
    // gatilho do trace.
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
    launchOptions: {
      // O Chromium usa /dev/shm para memória compartilhada e, quando ele acaba,
      // o sistema começa a paginar — que é a hora em que a máquina inteira
      // engasga. Este argumento manda usar arquivo temporário comum.
      args: ['--disable-dev-shm-usage'],
    },
  },
  projects: [
    // Logs in once via the UI and saves the session (localStorage token) so the
    // other specs start authenticated.
    { name: 'setup', testMatch: /auth\.setup\.ts/ },
    {
      name: 'chromium',
      dependencies: ['setup'],
      use: { ...devices['Desktop Chrome'], storageState: '.auth/user.json' },
    },
  ],
  webServer: {
    command: 'pnpm --filter frontend dev',
    url: BASE_URL,
    cwd: '..',
    reuseExistingServer: true,
    timeout: 120_000,
  },
})
