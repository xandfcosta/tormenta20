import { defineConfig, devices } from '@playwright/test'

/**
 * E2E suite (ALE-68). Lives OUTSIDE `frontend/` on purpose: it drives the running
 * app by URL, so it's **framework-agnostic** and survives the React→Solid
 * migration (ALE-63) unchanged — unlike the React-coupled unit tests.
 *
 * O ALVO PADRÃO É O BUILD, servido pelo Go — a forma de produção (ALE-256).
 *
 * Era o servidor de desenvolvimento, e isso custava duas coisas. A primeira é
 * tempo: cada carregamento puxava ~2.500 módulos ES soltos contra 33 chamadas de
 * API, e a suíte levava 7,7 min contra ~2,9 min pelo build.
 *
 * A segunda é a que importa: **os guardas mediam um CSS que não vai para
 * produção.** O `o aviso da mesa é pintado na cor da mesa` passava em dev e
 * falhava contra o build porque o minificador reescreve `oklch(0.27 0.016 300)`
 * como `oklch(27% .016 300)` — mesma cor, texto diferente. Esta suíte tem
 * dezenas de guardas de cor, contraste, tipografia e leiaute, e todos estavam do
 * lado errado do aviso que o `vite.config.ts` já carregava desde a ALE-76:
 * medir em DEV produziu o fantasma "o React bloqueia 64–74 ms", que não
 * sobreviveu a uma corrida de produção.
 *
 * `E2E_DEV=1` volta ao Vite para iterar num spec, e é ESCAPE explícito de
 * propósito: enquanto o atalho era o padrão, ninguém sabia que estava medindo
 * outra coisa.
 *
 * `vite preview` não serve de alvo — ele não é a forma de produção. O binário
 * único do Go serve SPA e API na MESMA ORIGEM, que é o que roda na mesa do dono,
 * e desde a ALE-253 isso dispensa `CORS_ORIGIN`: sem socket, o tempo real é uma
 * rota como as outras.
 */
const CONTRA_O_DEV = process.env.E2E_DEV === '1'
// O CI sobe o servidor ELE MESMO, porque precisa aplicar a seed depois de o
// goose migrar e antes de o primeiro teste rodar — ordem que o `webServer` do
// Playwright não expõe. Então ele diz aqui que já há alguém de pé, em vez de a
// gente adivinhar por `process.env.CI`: adivinhar acertaria hoje e erraria no
// dia em que alguém rodasse a suíte localmente com um servidor próprio.
const SERVIDOR_JA_DE_PE = process.env.E2E_SERVIDOR_EXTERNO === '1'
// Porta própria para o alvo do build: reaproveitar a :3001 pegaria um servidor
// de dev SEM `STATIC_DIR`, que responde a API e não serve SPA nenhuma — a suíte
// morreria em 404 com cara de defeito do app.
const PORTA_DO_BUILD = process.env.E2E_PORT ?? '3010'
const BASE_URL =
  process.env.E2E_BASE_URL ??
  (CONTRA_O_DEV ? 'http://localhost:5173' : `http://localhost:${PORTA_DO_BUILD}`)

export default defineConfig({
  testDir: './tests',
  timeout: 30_000,
  expect: { timeout: 7_000 },
  fullyParallel: true,
  // Um worker por padrão na máquina de dev, dois no CI.
  //
  // UM WORKER EM TODA PARTE, inclusive no CI (ALE-238).
  //
  // Histórico: o padrão do Playwright (metade dos núcleos → 4 aqui) saturava e
  // dava timeout PURO, com pico de 32,3s contra o teto de 30s; capar em 2 saiu
  // MAIS rápido (1,9 min contra 3,0), que é o sintoma clássico de
  // sobrescrição (ALE-93). Só que a máquina do dono ficou mais cheia desde
  // então — dev server, API, o browser dele, o browser da automação — e a
  // suíte passou a TRAVAR o laptop.
  //
  // O CI ficou em 2 e isso custou caro: três corridas vermelhas em sete num dia
  // só, sempre em asserções de LEIAUTE, sempre com vítimas DIFERENTES
  // (`session.spec:614`, `session.spec:300`, `piloto-datastar:969`) — e todas
  // passando no rerun. Medido aqui, nesta máquina de 8 núcleos:
  //
  //   1 worker  → 3,7 min, 183/183 verde (duas corridas)
  //   2 workers → 4,5 min, 1 vermelha
  //   2 workers → 8,3 min, 2 vermelhas
  //
  // **Dois workers é mais LENTO que um**, e é esse número que decide: o mesmo
  // sintoma de sobrescrição do ALE-93, com a mesma conclusão. No CI é pior por
  // aritmética — `ubuntu-latest` tem 2 vCPUs, então são 2 workers por 2 núcleos
  // MAIS o Chromium e o servidor Go, proporcionalmente pior que os 4-em-8 que
  // já haviam sido medidos como saturação.
  //
  // Vítima que varia a cada corrida é contenção de RECURSO; estado
  // compartilhado escolheria sempre a mesma. E o custo do vermelho não é o
  // rerun: é que um vermelho frequente e conhecido ensina a ignorar vermelho.
  //
  // O PREÇO, medido no CI depois da troca e não estimado: o job de e2e foi de
  // 3m33s–4m49s (dois workers) para 5m36s (um). Aqui, com 8 núcleos, um worker
  // era MAIS rápido; lá, com 2 vCPUs, ele custa ~1 min. Eu tinha escrito que o
  // custo era nenhum, e o número me corrigiu — fica assim porque um minuto por
  // corrida é barato contra 43% de vermelho que não é regressão.
  //
  // `E2E_WORKERS=2` continua disponível para quem quiser medir de novo.
  workers: Number(process.env.E2E_WORKERS ?? 1),
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
  // `undefined` e não um objeto: com servidor externo não há nada a subir, e
  // qualquer tentativa esbarraria na porta ocupada.
  webServer: SERVIDOR_JA_DE_PE
    ? undefined
    : CONTRA_O_DEV
    ? {
        command: 'pnpm --filter frontend dev',
        url: BASE_URL,
        cwd: '..',
        reuseExistingServer: true,
        timeout: 120_000,
      }
    : {
        // Constrói e serve como produção. `reuseExistingServer: false` é
        // deliberado: um servidor já no ar nesta porta pode estar sem
        // `STATIC_DIR` ou com um `dist` velho, e a suíte mediria o binário
        // errado sem nada acusar. Falhar por porta ocupada é barulhento, que é
        // o que se quer.
        // `go run` e não o `start` do engine-go: aquele roda um binário
        // pré-construído com `APP_ENV=production`, que leria o
        // `.env.production` do dono da mesa — arquivo não versionado, com
        // `COOKIE_SECURE` e banco próprios. A bancada tem de ser o ambiente de
        // desenvolvimento servindo o BUILD, não produção pela metade.
        command:
          'pnpm --filter frontend build && cd engine-go && ' +
          `STATIC_DIR=../frontend/dist PORT=${PORTA_DO_BUILD} go run ./cmd/api`,
        url: BASE_URL,
        cwd: '..',
        reuseExistingServer: false,
        // O build da SPA leva ~3 min com o wasm; 120 s derrubaria a suíte antes
        // de o servidor existir.
        timeout: 420_000,
      },
})
