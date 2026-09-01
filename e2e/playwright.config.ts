import { defineConfig, devices } from '@playwright/test'

/**
 * A suíte de e2e (ALE-68). Ela vive FORA dos pacotes de propósito: dirige o app
 * RODANDO, por URL, e por isso atravessou o React→Solid (ALE-63) e o
 * Solid→Datastar (ALE-272) sem mudar de forma — ao contrário dos testes de
 * unidade, que morreram com o stack de cada época.
 *
 * O ALVO É O BINÁRIO DO GO, que é a forma de produção: um processo servindo as
 * cenas, a API e o fluxo ao vivo na mesma origem. Ele já foi o servidor de
 * desenvolvimento do Vite, e isso custava duas coisas — tempo (7,7 min contra
 * ~2,9) e, o que importa mais, **os guardas mediam um CSS que não ia para
 * produção**: o `o aviso da mesa é pintado na cor da mesa` passava em dev e
 * falhava contra o build, porque o minificador reescreve `oklch(0.27 0.016 300)`
 * como `oklch(27% .016 300)` — mesma cor, texto diferente (ALE-256).
 *
 * Com a SPA apagada não há mais dois alvos possíveis: o `E2E_DEV=1` que voltava
 * ao Vite saiu junto com ele.
 */
// O CI sobe o servidor ELE MESMO, porque precisa aplicar a seed depois de o
// goose migrar e antes de o primeiro teste rodar — ordem que o `webServer` do
// Playwright não expõe. Então ele diz aqui que já há alguém de pé, em vez de a
// gente adivinhar por `process.env.CI`: adivinhar acertaria hoje e erraria no
// dia em que alguém rodasse a suíte localmente com um servidor próprio.
const SERVIDOR_JA_DE_PE = process.env.E2E_SERVIDOR_EXTERNO === '1'
// Porta própria: reaproveitar a :3001 pegaria o servidor de desenvolvimento de
// quem estiver com o app aberto, e a suíte mediria o banco dele.
const PORTA_DO_BUILD = process.env.E2E_PORT ?? '3010'
// O BANCO DO E2E É PRÓPRIO, apagado e recriado a cada corrida (ALE-269).
//
// Antes a suíte caía no `DATABASE_URL` padrão — `data/t20-dev.db`, o MESMO
// arquivo que se usa ao conferir qualquer coisa no navegador. Um tabuleiro
// aberto à mão ou um combatente esquecido na fila viravam vermelho com cara de
// regressão do commit: o teste que quebra não tem relação com o que mudou, e a
// suíte estava verde antes. Aconteceu duas vezes na mesma issue.
//
// O `rm` mora no comando do servidor e não num passo à parte porque ele tem de
// acontecer ANTES de a API abrir o arquivo — é a abertura que migra. Os
// irmãos `-wal` e `-shm` vão junto: deixá-los faz o SQLite reconstruir o banco
// velho por cima do novo, e o "banco limpo" seria o sujo com outro nome.
const BANCO_DO_E2E = 'data/e2e.db'
const BASE_URL = process.env.E2E_BASE_URL ?? `http://localhost:${PORTA_DO_BUILD}`

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
    // A SEMENTE vem antes do login, e é a ordem inteira do arranjo: a API migra
    // o arquivo ao abrir, a seed é só INSERT, e o login precisa dos usuários que
    // ela cria. Só existe quando NÓS subimos o servidor — com servidor externo
    // quem semeia é o CI, e aplicar `seed.sql` duas vezes estoura nas chaves.
    ...(SERVIDOR_JA_DE_PE ? [] : [{ name: 'semente', testMatch: /semente\.setup\.ts/ }]),
    // Logs in once via the UI and saves the session (localStorage token) so the
    // other specs start authenticated.
    {
      name: 'setup',
      testMatch: /auth\.setup\.ts/,
      dependencies: SERVIDOR_JA_DE_PE ? [] : ['semente'],
    },
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
    : {
        // Serve como produção. `reuseExistingServer: false` é deliberado: um
        // servidor já no ar nesta porta pode ser de outra árvore ou de um
        // binário velho, e a suíte mediria o errado sem nada acusar. Falhar por
        // porta ocupada é barulhento, que é o que se quer.
        // `go run` e não o `start` do engine-go: aquele roda um binário
        // pré-construído com `APP_ENV=production`, que leria o
        // `.env.production` do dono da mesa — arquivo não versionado, com
        // `COOKIE_SECURE` e banco próprios. A bancada tem de ser o ambiente de
        // desenvolvimento, não produção pela metade.
        command:
          'cd engine-go && ' +
          `rm -f ${BANCO_DO_E2E} ${BANCO_DO_E2E}-wal ${BANCO_DO_E2E}-shm && ` +
          `PORT=${PORTA_DO_BUILD} DATABASE_URL=file:./${BANCO_DO_E2E} go run ./cmd/api`,
        url: BASE_URL,
        cwd: '..',
        reuseExistingServer: false,
        // O build da SPA leva ~3 min com o wasm; 120 s derrubaria a suíte antes
        // de o servidor existir.
        timeout: 420_000,
      },
})
