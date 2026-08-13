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
  // O padrão (metade dos núcleos → 4 aqui) satura uma máquina de dev que ainda
  // roda browser e outro dev server: medido em 2026-08-13, 2 falhas por timeout
  // PURO, pico de 32,3s contra o teto de 30s. Com 2 workers a MESMA máquina dá
  // 41/41 em 1,9 min contra 3,0 min — capar saiu mais rápido, que é o sintoma
  // de sobrescrição. O runner do CI é dedicado e o padrão já lhe dá 2 workers,
  // com o teste mais lento em 11,6s, então lá fica o padrão (ALE-93).
  workers: process.env.CI ? undefined : 2,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  reporter: [['list'], ['html', { open: 'never' }]],
  use: {
    baseURL: BASE_URL,
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
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
