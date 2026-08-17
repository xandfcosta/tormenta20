import { type Page, expect, test } from '@playwright/test'
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
    await page.getByRole('button', { name: 'Combatente' }).click()
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
   * A cena cabe na tela COM UM COMBATENTE ABERTO — que é o estado em que a mesa
   * de verdade fica (ALE-125).
   *
   * O teste vizinho ("a cena do mestre cabe na tela") mede a cena VAZIA e ficou
   * verde enquanto este defeito existia: abrir a ficha de um PC empurrava a
   * barra de abas dela para fora da área visível, e como todo contêiner da cena
   * tem `overflow-hidden` (posto ali justamente para a página não rolar), o
   * sintoma nunca chegava à raiz. Nenhuma asserção de "a página não rola" podia
   * vê-lo.
   *
   * Por isso aqui a asserção é de ALCANCE: a barra de abas da ficha tem de estar
   * na área visível em todo formato. É e2e porque é altura real — em jsdom todo
   * elemento mede zero e a mesma asserção passaria verde sobre a tela quebrada.
   *
   * O mesmo PC aberto paga por uma segunda asserção, esta de RELAÇÃO entre a
   * faixa do combatente e a ficha embaixo dela (ALE-145). Alcance e proporção
   * são coisas diferentes: a barra de abas pode estar na tela e a ficha ainda
   * assim receber uma nesga, que é o que o print do dono mostrava. Medido antes
   * do conserto, o que vinha ANTES da ficha comia 49%, 50%, 51%, 49% e 51% da
   * região nos cinco formatos abaixo; depois, 13%, 11%, 20%, 11% e 30%.
   */
  test('com a ficha de um PC aberta, a faixa é pequena e a barra de abas continua alcançável', async ({
    page,
  }) => {
    await page.goto('/campaigns/1/sessions/4')
    await expect(page.getByRole('status', { name: 'Conectado' })).toBeVisible()

    // O teste traz o PRÓPRIO grupo: a iniciativa da seed do CI está VAZIA, e
    // depender de um PC que só existe no banco de dev já quebrou o CI três vezes
    // nesta issue. "Adicionar grupo" é idempotente e traz os PCs da campanha.
    const antes = await labelsNaIniciativa(page)
    await page.getByRole('button', { name: 'Adicionar grupo' }).click()
    await expect(page.locator('[role="progressbar"][aria-label^="PM "]').first()).toBeVisible()

    // A barra de PM é o sinal de que a linha tem PERSONAGEM atrás dela — o
    // crachá "PC" mora fora do botão, e casar por texto pegaria "NPC" junto.
    const nomeDoPc = await page.evaluate(() => {
      const barra = document.querySelector('[role="progressbar"][aria-label^="PM "]')
      // Sobe até a LINHA — `closest('[class*=rounded]')` pararia no invólucro da
      // própria barra, que também é arredondado.
      let no: HTMLElement | null = barra as HTMLElement | null
      while (no && !no.querySelector('button[aria-pressed]')) no = no.parentElement
      return no?.querySelector('button[aria-pressed]')?.textContent?.trim() ?? ''
    })
    expect(nomeDoPc, 'não achei uma linha de personagem na iniciativa').not.toBe('')

    // Só um PC tem ficha atrás dele — é o conteúdo mais alto que entra na região.
    await page.locator('button[aria-pressed]', { hasText: nomeDoPc }).first().click()
    const abaDaFicha = page.getByRole('tab', { name: 'Perícias' })
    await expect(abaDaFicha).toBeVisible()

    // COM CONDIÇÕES ATIVAS, que é o estado em que a faixa quebrou (ALE-147):
    // elas são o único conteúdo dela que cresce sozinho durante o combate, e
    // com a fileira vazia nenhuma asserção via o defeito.
    for (const [i, condicao] of ['Abalado', 'Agarrado', 'Cego'].entries()) {
      const seletor = page.getByLabel('Aplicar condição')
      await seletor.click()
      await seletor.fill(condicao)
      await page.getByRole('option', { name: condicao, exact: true }).first().click()
      // Espera pelo GATILHO, não pelo chip: acima de duas condições a terceira
      // vive dentro do popover e o botão de remover dela nem está no DOM.
      const rotulo = i === 0 ? 'Ver a condição ativa' : `Ver as ${i + 1} condições ativas`
      await expect(page.getByRole('button', { name: rotulo })).toBeVisible()
    }
    await page.keyboard.press('Escape')

    const nomeNaFaixa = page.getByRole('heading', { name: nomeDoPc })
    const fechar = page.getByRole('button', { name: 'Fechar o combatente' })

    for (const viewport of VIEWPORTS) {
      await page.setViewportSize({ width: viewport.width, height: viewport.height })
      // Abaixo de 1024 a cena mostra uma região por vez: a ficha vive na mesa.
      const mesa = page.getByRole('button', { name: 'mesa', exact: true })
      if (await mesa.isVisible()) await mesa.click()
      await expect(abaDaFicha, `${viewport.name}: a barra de abas saiu da tela`).toBeInViewport()

      // As duas garantias que a ALE-147 quebrou. O nome é o que diz DE QUEM é a
      // ficha, e as condições o espremeram até "AI" (duas letras); fechar é a
      // saída da tela, e ele saiu dela. Ambas nos seis formatos.
      const largura = await nomeNaFaixa.evaluate((el) => el.getBoundingClientRect().width)
      expect(
        largura,
        `${viewport.name}: o nome do combatente ficou com ${Math.round(largura)}px`,
      ).toBeGreaterThanOrEqual(100)
      await expect(fechar, `${viewport.name}: fechar o combatente saiu da tela`).toBeInViewport()

      // O celular deitado fica FORA desta conta, e o motivo é medido, não
      // conveniência: dos 390px de altura, 179 são cromo da CENA (cabeçalho 49
      // + faixa de turno 50 + seletor de região 32 + barra de abas do workspace
      // 36), e a região do combatente inteira fica com 165px. Nenhuma faixa
      // utilizável cabe em 35% de 165 — o que sobra ali é defeito da cena, não
      // da faixa, e está registrado na ALE-146. A garantia que vale neste
      // formato é a de alcance, logo acima, e essa roda nos seis.
      if (viewport.name === 'mobile-landscape') continue

      const { regiao, antesDaFicha } = await medirRegiaoDoCombatente(page)
      expect(
        antesDaFicha / regiao,
        `${viewport.name}: a faixa comeu ${antesDaFicha}px dos ${regiao}px da região`,
      ).toBeLessThanOrEqual(0.35)
    }

    // Sai como entrou. As condições ficam GRAVADAS na ficha, então tirá-las é
    // obrigação deste teste — sem isto elas se acumulam entre execuções e a
    // próxima roda contra uma ficha que ninguém montou (F.I.R.S.T: repetível).
    await page.setViewportSize({ width: 1920, height: 1080 })
    const gatilho = page.getByRole('button', { name: /^Ver as? .*condi/ })
    for (let i = 0; i < 5 && (await gatilho.count()) > 0; i++) {
      // Sai por dentro do popover, onde as três estão listadas juntas.
      await gatilho.click()
      const painel = page.getByRole('dialog')
      await painel.getByRole('button', { name: /^Remover condição/ }).first().click()
      await page.keyboard.press('Escape')
      await expect(painel).toBeHidden()
    }
    await expect(gatilho).toHaveCount(0)

    // Tira da iniciativa só quem ESTE teste pôs.
    for (const label of await novosDesde(page, antes)) {
      await page.getByRole('button', { name: `Remover ${label}` }).click()
    }
  })

  /**
   * A altura da região do combatente e quanto dela é gasto ANTES de a ficha
   * começar. Ancorado na barra de PV (que só existe quando há combatente
   * aberto) e no painel de aba visível — nomes de classe mudam a cada restyle
   * e não prometem nada.
   */
  async function medirRegiaoDoCombatente(
    page: Page,
  ): Promise<{ regiao: number; antesDaFicha: number }> {
    return page.evaluate(() => {
      const vida = document.querySelector('[role="progressbar"][aria-label="Vida"]')
      if (!vida) throw new Error('nenhum combatente aberto: não há barra de Vida na tela')
      let secao: HTMLElement | null = vida as HTMLElement
      while (secao && secao.tagName !== 'SECTION') secao = secao.parentElement
      if (!secao) throw new Error('a barra de Vida não está dentro de uma <section>')
      const ficha = [...secao.querySelectorAll('[role="tabpanel"]')].find(
        (painel) => painel.getBoundingClientRect().height > 0,
      )
      if (!ficha) throw new Error('a ficha do combatente não tem painel de aba visível')
      const regiao = secao.getBoundingClientRect()
      return {
        regiao: Math.round(regiao.height),
        antesDaFicha: Math.round(ficha.getBoundingClientRect().top - regiao.top),
      }
    })
  }

  /**
   * Nenhum filho pode ser pintado para FORA do pai dentro da ficha do
   * combatente.
   *
   * Três defeitos do mesmo mecanismo apareceram no mesmo dia, todos reportados
   * pelo dono com print: os cartões de equipados da Mochila com 72px e o crachá
   * de bônus por cima do vizinho (ALE-148), o nome da perícia espremido a zero
   * (ALE-145) e a lista do Catálogos passando do cartão (ALE-149). A causa é
   * sempre a mesma: um bloco decide layout por breakpoint de JANELA (`sm:`,
   * `lg:`, `xl:`) enquanto vive numa coluna de 518px — numa janela de 1920 a
   * media casa e o bloco pega o layout largo dentro do espaço estreito.
   *
   * Por isso este teste roda só a 1920: é a largura em que janela e contêiner
   * mais discordam. Num viewport estreito as duas concordam e o defeito some.
   *
   * Só o browser mede isto — em jsdom todo elemento tem largura zero e a mesma
   * asserção passa verde sobre a tela quebrada.
   */
  test('a 1920px nenhum filho é pintado para fora do pai na ficha do combatente', async ({
    page,
  }) => {
    await page.setViewportSize({ width: 1920, height: 1080 })
    await page.goto('/campaigns/1/sessions/4')
    await expect(page.getByRole('status', { name: 'Conectado' })).toBeVisible()

    const antes = await labelsNaIniciativa(page)
    await page.getByRole('button', { name: 'Adicionar grupo' }).click()
    await expect(page.locator('[role="progressbar"][aria-label^="PM "]').first()).toBeVisible()
    const nomeDoPc = await page.evaluate(() => {
      const barra = document.querySelector('[role="progressbar"][aria-label^="PM "]')
      let no: HTMLElement | null = barra as HTMLElement | null
      while (no && !no.querySelector('button[aria-pressed]')) no = no.parentElement
      return no?.querySelector('button[aria-pressed]')?.textContent?.trim() ?? ''
    })
    await page.locator('button[aria-pressed]', { hasText: nomeDoPc }).first().click()

    // Os blocos com GRADE, que são os que quebram por medida errada. Cada um
    // espera por um conteúdo SEU — o clique na aba não garante que o bloco já
    // pintou, e medir antes disso mede a tela anterior.
    const BLOCOS = [
      { aba: 'Perícias', pinta: 'Fortitude' },
      { aba: 'Combate', pinta: 'Atq CaC' },
      { aba: 'Mochila', pinta: 'Mãos' },
    ]
    // Escopo na SEÇÃO do combatente: "Combate" também é o nome do seletor de
    // região da cena, e sem escopo o localizador casa os dois.
    const painel = page.locator('section', {
      has: page.getByRole('progressbar', { name: 'Vida' }),
    })
    for (const { aba: bloco, pinta } of BLOCOS) {
      await painel.getByRole('tab', { name: bloco }).click()
      await expect(page.getByText(pinta, { exact: true }).first()).toBeVisible()

      const escapando = await page.evaluate(() => {
        const vida = document.querySelector('[role="progressbar"][aria-label="Vida"]')
        let coluna: HTMLElement | null = vida as HTMLElement | null
        while (coluna && coluna.tagName !== 'SECTION') coluna = coluna.parentElement

        // A relação é FILHO contra PAI, não contra a coluna: o crachá de bônus
        // vazava 6px do cartão dele e era pintado sobre o vizinho, muito antes
        // de chegar perto da borda da coluna. Uma asserção contra a coluna
        // passava verde por cima disso — foi a primeira versão deste teste.
        return [...coluna!.querySelectorAll('*')]
          .filter((el) => {
            const pai = el.parentElement
            if (!pai) return false
            const estilo = getComputedStyle(el)
            // Absoluto sai do fluxo de propósito (o ✕ de desequipar), e um pai
            // que rola na horizontal contém o filho rolando.
            if (estilo.position === 'absolute' || estilo.position === 'fixed') return false
            if (getComputedStyle(pai).overflowX !== 'visible') return false
            const r = el.getBoundingClientRect()
            return r.width > 0 && r.right > pai.getBoundingClientRect().right + 1
          })
          .map((el) => (el.textContent ?? '').trim().slice(0, 30))
          .slice(0, 5)
      })
      expect(escapando, `${bloco}: filho pintado para fora do pai`).toEqual([])
    }

    // Sai como entrou.
    for (const label of await novosDesde(page, antes)) {
      await page.getByRole('button', { name: `Remover ${label}` }).click()
    }
  })

  /** Os rótulos que estão na iniciativa agora. */
  async function labelsNaIniciativa(page: Page): Promise<string[]> {
    return page.$$eval('button[aria-label^="Remover "]', (bs) =>
      bs.map((b) => (b.getAttribute('aria-label') ?? '').replace('Remover ', '')),
    )
  }

  /** Quem entrou na lista depois do instantâneo — o que este teste tem de limpar. */
  async function novosDesde(page: Page, antes: string[]): Promise<string[]> {
    const agora = await labelsNaIniciativa(page)
    return agora.filter((label) => !antes.includes(label))
  }

  /**
   * Só a LISTA rola; o cabeçalho e as ações ficam ancorados (ALE-131).
   *
   * O defeito: quem rolava era a coluna inteira, então descer a lista levava
   * embora "Adicionar grupo" e "+ Combatente" — numa mesa de dez combatentes,
   * adicionar o décimo primeiro exigia rolar de volta ao topo.
   *
   * Por que e2e: é rolagem e altura REAIS. Em jsdom todo elemento mede zero,
   * `scrollTop` nunca sai de zero e a mesma asserção passaria verde sobre a
   * tela quebrada. A janela é baixa de propósito, para a lista transbordar com
   * poucos combatentes — o teste traz os próprios, porque a iniciativa da seed
   * do CI está vazia.
   */
  test('rolar a iniciativa não leva embora as ações', async ({ page }) => {
    const nomes = [1, 2, 3, 4, 5].map((n) => `Fileira de teste ${Date.now()}-${n}`)
    await page.setViewportSize({ width: 1280, height: 420 })
    await page.goto('/campaigns/1/sessions/4')
    await expect(page.getByRole('status', { name: 'Conectado' })).toBeVisible()

    await page.getByRole('button', { name: 'Combatente' }).click()
    for (const nome of nomes) {
      await page.getByLabel('Nome').fill(nome)
      await page.getByRole('button', { name: 'Adicionar', exact: true }).click()
      await expect(page.getByRole('button', { name: `Remover ${nome}` })).toBeVisible()
    }

    const acao = page.getByRole('button', { name: 'Adicionar grupo' })
    await expect(acao).toBeInViewport()

    // Rola a LISTA até o fim, não a página.
    const rolou = await page.evaluate(() => {
      const linha = document.querySelector('button[aria-label^="Iniciativa de"]')
      const lista = linha?.closest('[class*="overflow-y-auto"]') as HTMLElement | null
      if (!lista) return { achou: false, transbordou: false, rolou: 0 }
      lista.scrollTop = lista.scrollHeight
      return {
        achou: true,
        transbordou: lista.scrollHeight > lista.clientHeight + 8,
        rolou: lista.scrollTop,
      }
    })

    // Sem transbordo o teste não prova nada: seria uma rolagem que não aconteceu.
    expect(rolou).toMatchObject({ achou: true, transbordou: true })
    expect(rolou.rolou).toBeGreaterThan(0)
    await expect(acao).toBeInViewport()
    await expect(page.getByRole('heading', { name: 'Iniciativa' })).toBeInViewport()

    for (const nome of nomes) {
      await page.getByRole('button', { name: `Remover ${nome}` }).click()
    }
    await expect(page.getByRole('button', { name: `Remover ${nomes[0]}` })).toBeHidden()
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
