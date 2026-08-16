import { expect, test } from '@playwright/test'
import { VIEWPORTS, expectPageDoesNotScroll } from './support/viewports'

const CAMPAIGN = 'Snapshot Test ALE-33' // the seed chronicle with a live session

/**
 * Hub → Crônicas → abrir campanha → entrar na sessão ao vivo.
 *
 * Read-only once inside: asserts the socket.io gateway connected, without
 * touching initiative/turns, so the seed survives the run untouched.
 */
test.describe('Sessão ao vivo', () => {
  test('Crônicas → campanha → continuar a sessão (realtime conectado)', async ({ page }) => {
    await page.goto('/campaigns')
    // O estado "ao vivo" chega DEPOIS da lista (fan-out separado de sessões) e
    // troca os botões do livro. Esperar ele assentar antes de clicar — senão a
    // ação some debaixo do cursor e o clique cai no botão vizinho (ALE-78).
    await expect(page.getByRole('button', { name: /^Continuar a sessão/ })).toBeVisible()

    await page.getByRole('button', { name: /^Abrir crônica/ }).click()
    await expect(page.getByRole('heading', { name: CAMPAIGN, level: 1 })).toBeVisible()

    await page.getByRole('button', { name: 'Continuar a sessão' }).click()
    await expect(page).toHaveURL(/\/campaigns\/\d+\/sessions\/\d+$/)

    await expect(page.getByRole('heading', { name: 'Iniciativa' })).toBeVisible()
    // The connection chip flips to "Conectado" only after the socket handshake.
    await expect(page.getByRole('status', { name: 'Conectado' })).toBeVisible()
  })

  /**
   * A mesma família da ALE-96, um andar acima. O `SessionTrackerPage` lia
   * `session.data` e `campaign.data` para montar o TÍTULO, que é prop do
   * `MatchShell` — avaliado antes do `Show`. A leitura pendente suspende, o
   * `Suspense` que o solid-router põe em todo route match desanexa a cena
   * inteira, e o que o jogador vê ao clicar "Continuar sessão" é a tela EM
   * BRANCO. O Skeleton escrito para esse instante nunca podia pintar.
   *
   * Por que e2e: só um browser testemunha. Sem router não há Suspense, e em
   * jsdom a leitura pendente devolve `undefined` e o Skeleton aparece — verde
   * sobre a tela quebrada.
   *
   * A resposta da sessão fica SEGURA (não atrasada) para a asserção ser sobre
   * um estado, não sobre uma corrida.
   */
  test('a cena da sessão não fica em branco enquanto os dados carregam', async ({ page }) => {
    let release = (): void => {}
    const held = new Promise<void>((resolve) => {
      release = resolve
    })
    let requested = (): void => {}
    const inFlight = new Promise<void>((resolve) => {
      requested = resolve
    })
    await page.route('**/api/campaigns/1/sessions/4', async (route) => {
      requested()
      await held
      await route.continue()
    })

    await page.goto('/campaigns/1/sessions/4')
    await inFlight

    // O shell da partida continua na tela, e o lugar do conteúdo diz que está
    // carregando — o snapshot da falha original não tinha NADA disso.
    await expect(page.getByRole('link', { name: 'Sair da sessão' })).toBeVisible()
    await expect(page.getByRole('status', { name: 'Carregando a sessão' })).toBeVisible()

    release()
    await expect(page.getByRole('heading', { name: 'Iniciativa' })).toBeVisible()
  })

  /**
   * A coluna da iniciativa é 5/12 da tela no shell do mestre, mas TODAS as
   * quebras da linha eram por viewport (`sm:`): a 1024px o browser dava o
   * layout largo a uma coluna de 412px e os botões de PV iam parar 141px FORA
   * da área visível — inalcançáveis, e sem rolagem horizontal para chegar até
   * eles. A medida certa é a do CONTÊINER (ALE-122).
   *
   * Por que e2e: só um browser mede layout. Em jsdom todo elemento tem largura
   * zero, e a mesma asserção passa verde sobre a tela quebrada.
   */
  test('a 1024px os controles de PV ficam dentro da coluna da iniciativa', async ({ page }) => {
    const alvo = `Alvo de layout ${Date.now()}`
    await page.setViewportSize({ width: 1024, height: 768 })
    await page.goto('/campaigns/1/sessions/4')
    await expect(page.getByRole('status', { name: 'Conectado' })).toBeVisible()

    // O teste cria o próprio combatente: a iniciativa da seed do CI está VAZIA,
    // e esperar por uma linha que não existe fazia o teste falhar lá e passar
    // aqui. Ele também o remove no fim, para a seed sair como entrou.
    await page.getByLabel('Nome').fill(alvo)
    await page.getByRole('button', { name: 'Adicionar', exact: true }).click()
    await expect(page.getByRole('button', { name: `Ferir ${alvo}` })).toBeVisible()

    const tracker = page
      .getByRole('heading', { name: 'Iniciativa' })
      .locator('xpath=ancestor::section[1]')
    const escaping = await tracker.evaluate((section) => {
      const limit = section.getBoundingClientRect().right
      return [...section.querySelectorAll('button')]
        .filter((button) => button.getBoundingClientRect().right > limit)
        .map((button) => button.getAttribute('aria-label') ?? button.textContent)
    })

    await page.getByRole('button', { name: `Remover ${alvo}` }).click()
    await expect(page.getByRole('button', { name: `Ferir ${alvo}` })).toBeHidden()

    expect(escaping).toEqual([])
  })

  /**
   * O descanso de cena TRAVAVA a aba (ALE-122). O toast é uma árvore reativa de
   * terceiro que, ao montar, mede a própria altura e escreve o valor num sinal;
   * disparado de dentro de um `createEffect`, esse write caía no mesmo ciclo do
   * efeito e o ciclo se realimentava — medido na aba travada: o toast medido
   * 400 vezes com a mesma altura, `runUpdates` aninhado 78 níveis e subindo.
   *
   * Por que e2e: em jsdom todo elemento mede zero, então a medição que alimenta
   * o laço nunca acontece e a mesma asserção passa verde sobre a aba morta. O
   * teste exige o toast NA TELA (senão não haveria medição nenhuma) e depois
   * exige que a página ainda responda.
   */
  test('descanso de cena avisa a mesa sem travar a aba', async ({ page }) => {
    await page.goto('/campaigns/1/sessions/4')
    await expect(page.getByRole('status', { name: 'Conectado' })).toBeVisible()

    await page.getByRole('button', { name: 'Descanso de cena' }).click()

    await expect(page.getByText('Efeitos temporários de cena foram limpos.')).toBeVisible()
    // A aba viva é o que estava em jogo: numa aba travada isto nunca resolve.
    await expect(page.getByRole('button', { name: 'Próximo turno' })).toBeEnabled({ timeout: 5000 })
  })

  /**
   * A premissa do app, dita pelo dono: "sensação de jogo, não ter scroll,
   * mostrar os dados todos na tela". A cena do mestre foi reconstruída como
   * shell por causa disso (ALE-122) e a verificação viveu só numa mensagem de
   * commit. Aqui ela roda de novo a cada push, nos seis formatos.
   */
  test('a cena do mestre cabe na tela: a página não rola em nenhum formato', async ({ page }) => {
    await page.goto('/campaigns/1/sessions/4')
    await expect(page.getByRole('heading', { name: 'Iniciativa' })).toBeVisible()

    await expectPageDoesNotScroll(page, VIEWPORTS)
  })

  /**
   * A armadilha mais reincidente do repositório (ALE-95/96/121/122, quatro vezes
   * só nesta issue): ler `.data` de uma query PENDENTE suspende, e o `Suspense`
   * que o router põe em todo route match DESANEXA a cena inteira — a tela pisca
   * ou fica em branco no meio do combate. `settledQuery` existe para isso e é
   * lido em oito módulos; a prova de que funciona vinha de contar nós à mão
   * depois de cada regressão, nunca de um teste.
   *
   * Aqui um MutationObserver testemunha o que o olho vê: o rastreador não pode
   * ser REMOVIDO do DOM enquanto o mestre troca de aba e abre um combatente.
   *
   * Por que e2e: sem router não há Suspense, e em jsdom a leitura pendente
   * devolve `undefined` — o mesmo teste passaria verde sobre a tela que pisca.
   */
  test('trocar de aba e abrir um combatente não desanexa a cena', async ({ page }) => {
    await page.goto('/campaigns/1/sessions/4')
    await expect(page.getByRole('status', { name: 'Conectado' })).toBeVisible()

    // Vigia o RASTREADOR: trocar de aba desmonta o conteúdo da aba anterior, e
    // isso é normal; o que nunca pode acontecer é a lista de combate sair da
    // tela — é ela que o suspend arranca junto com a cena.
    await page.evaluate(() => {
      const janela = window as unknown as { __desanexos: number }
      janela.__desanexos = 0
      const ehRastreador = (node: Node) =>
        node instanceof HTMLElement &&
        [...node.querySelectorAll('h2')].some((h) => h.textContent?.includes('Iniciativa'))
      new MutationObserver((records) => {
        for (const record of records) {
          for (const node of record.removedNodes) {
            if (ehRastreador(node)) janela.__desanexos++
          }
        }
      }).observe(document.body, { childList: true, subtree: true })
    })

    for (const aba of ['Bestiário', 'Catálogos', 'Notas', 'Combatente']) {
      await page.getByRole('tab', { name: aba }).click()
      await expect(page.getByRole('tab', { name: aba })).toHaveAttribute('aria-selected', 'true')
    }

    const desanexos = await page.evaluate(
      () => (window as unknown as { __desanexos: number }).__desanexos,
    )
    expect(desanexos, 'o rastreador foi removido do DOM (suspend desanexou a cena)').toBe(0)
    await expect(page.getByRole('heading', { name: 'Iniciativa' })).toBeVisible()
  })

  test('Sair da sessão volta pra crônica', async ({ page }) => {
    await page.goto('/campaigns/1/sessions/4')
    await expect(page.getByRole('heading', { name: 'Iniciativa' })).toBeVisible()

    // Um LINK, não um botão: sem `asChild` no Solid, um link com cara de botão
    // é um `<a>` vestindo as classes do botão (armadilha #6 do porte).
    await page.getByRole('link', { name: 'Sair da sessão' }).click()
    await expect(page).toHaveURL(/\/campaigns\/1$/)
    await expect(page.getByRole('heading', { name: CAMPAIGN, level: 1 })).toBeVisible()
  })
})
