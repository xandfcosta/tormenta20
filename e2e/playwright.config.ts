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
  // `retries: 0` local é DELIBERADO e fica: retentativa esconde intermitência,
  // que é justamente o que se caça numa máquina de dev (ALE-244).
  retries: process.env.CI ? 1 : 0,
  reporter: [['list'], ['html', { open: 'never' }]],
  use: {
    baseURL: BASE_URL,
    // `retain-on-failure` e NÃO `on-first-retry`, que era o que estava aqui.
    // As duas linhas se cancelavam: sem retry não há primeira tentativa
    // repetida, então fora do CI o trace nunca era gravado — não raramente,
    // NUNCA. E `e2e/test-results` sequer existia em disco (ALE-244).
    //
    // O preço disso foi alto e é o motivo deste comentário. A ALE-238 nasceu de
    // UMA assinatura de erro transcrita à mão de scrollback de terminal, num
    // formato que o Playwright não emite, sem segunda amostra em lugar nenhum.
    // Duas sessões construíram e derrubaram duas explicações elaboradas em cima
    // dela, e onze corridas cheias produziram zero artefato aproveitável. A
    // DÉCIMA SEGUNDA, a primeira com esta linha, capturou a falha e mostrou
    // numa tacada que não era estado herdado e que uma condição aplicada tinha
    // sumido depois de aparecer.
    //
    // O `retries: 0` local FICA, e é deliberado — não é a metade esquecida
    // deste conserto: retentativa ESCONDE flake, que é justamente o que se está
    // caçando aqui. O que estava errado era só o gatilho do trace.
    //
    // Guarda o trace de toda tentativa que FALHA e descarta as que passam: o
    // custo em disco é proporcional ao que quebrou, não ao tamanho da suíte.
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
