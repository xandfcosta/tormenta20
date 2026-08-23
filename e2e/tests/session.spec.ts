import { expect, type Locator, type Page, test } from '@playwright/test'
import {
  expectDentroDaJanela,
  expectFormaColuna,
  expectNadaEscapa,
  expectNadaRolaDeLado,
} from './support/geometry'
import {
  abreAFicha,
  abreAFichaPelaGaveta,
  abreAFila,
  abreConsulta,
  acionaOCiclo,
  cenaViva,
  fechaAFicha,
  fechaAFila,
  garanteACena,
  labelsNaFila,
  labelsNaGaveta,
  primeiroDaFila,
  trilhoDaFila,
  trilhoDeConsultas,
} from './support/gm-scene'
import { expectPageDoesNotScroll, VIEWPORTS } from './support/viewports'

/**
 * O avanço de turno, achado pelo nome que ele ANUNCIA (ALE-184): o botão diz
 * "Próximo: Ogro" em combate, "Começar: Arwen" antes da primeira rodada e
 * "Ninguém na fila" quando não há para onde ir. Casar por prefixo é o que
 * mantém o teste falando do CONTROLE e não do combatente da vez.
 *
 * "Iniciar cena" fica DE FORA de propósito (ALE-210). Ele divide a vaga com o
 * avanço mas não é um deles: é o começo do ciclo, e contá-lo aqui faria o
 * guarda de "um e só um avanço" ficar verde numa tela que não tem avanço
 * nenhum — que é justamente o zero que o guarda chama de falha mais grave.
 */
function avancoDeTurno(page: Page) {
  return page.getByRole('button', { name: /^(Próximo|Começar|Ninguém na fila)/ })
}

const CAMPAIGN = 'Snapshot Test ALE-33' // the seed chronicle with a live session

/** Combatentes que o teste do recorte CRIA, para a fila passar do que cabe. */
const RECHEIO_DO_TRILHO = ['Recheio A', 'Recheio B', 'Recheio C', 'Recheio D', 'Recheio E']

/**
 * Hub → Campanhas → abrir campanha → entrar na sessão ao vivo.
 *
 * Read-only once inside: asserts the socket.io gateway connected, without
 * touching initiative/turns, so the seed survives the run untouched.
 */
/**
 * SERIAL: todos estes testes escrevem na sessão 4 da seed — abrem tabuleiro,
 * povoam a iniciativa, ferem, removem. Em paralelo eles se atropelam, e o
 * sintoma não parece corrida: é um número que não bate ou um clique que cai no
 * botão errado (ALE-124). A regra da casa já dizia isto; faltava aplicá-la.
 */
/**
 * Quais linhas da iniciativa têm animação de JS rodando agora (ALE-174).
 *
 * Filtra pelo TIPO e isso não é detalhe: `document.getAnimations()` devolve
 * também `CSSTransition` e `CSSAnimation`, e a linha da iniciativa tem
 * `transition-colors` própria mais a da barra de vitais. Sem o filtro o teste
 * passa VERDE com a animação desligada — foi o que aconteceu comigo, e só
 * apareceu porque sabotei o conserto para conferir. As animações desta issue
 * são WAAPI, que instancia `Animation` puro.
 */
async function linhasAnimadas(page: Page): Promise<string[]> {
  return page.evaluate(() =>
    document
      .getAnimations()
      .filter((a) => a.constructor.name === 'Animation')
      .map((a) => (a.effect as KeyframeEffect | null)?.target)
      .filter((alvo): alvo is Element => !!alvo)
      .map((alvo) => alvo.closest('[data-entry-id]')?.getAttribute('data-entry-id'))
      .filter((id): id is string => !!id),
  )
}

test.describe.configure({ mode: 'serial' })

test.describe('Sessão ao vivo', () => {
  /**
   * A sessão 4 começa VAZIA, sempre — e isto é conserto, não zelo.
   *
   * Os testes daqui são serial e cada um limpa o que criou, mas limpeza que
   * falha não limpa: um teste que estoura no meio deixa combatentes na seed
   * COMPARTILHADA, e o próximo mede o lixo em vez do app. Isso já produziu, no
   * mesmo dia, um guarda anunciando o combatente de outro teste, um estouro de
   * layout que não reproduzia sozinho e três execuções de CI vermelhas cuja
   * causa não era o código.
   *
   * O reinício é o mesmo gesto que o mestre tem na tela — a iniciativa, a
   * rodada e o turno voltam a zero —, e deixa o arquivo começando de onde a
   * seed do CI começa. Provado: com a sessão suja de propósito (três restos e
   * rodada 7), sem isto o arquivo cai no teste do celular deitado; com isto,
   * 23/23.
   */
  test.beforeAll(async ({ browser }) => {
    const contexto = await browser.newContext({ storageState: '.auth/user.json' })
    const page = await contexto.newPage()
    try {
      await page.goto('/campaigns/1/sessions/4')
      await expect(page.getByRole('status', { name: 'Conectado' })).toBeVisible()
      // "Reiniciar" saiu do menu da sessão para o pé do trilho na ALE-210, e só
      // aparece quando há gente na fila: com a seed já limpa não existe o que
      // reiniciar, e clicar incondicionalmente esperaria para sempre por um
      // botão que a tela está CERTA em não desenhar. O trilho visível é o que
      // diz que a árvore assentou — `count()` sozinho é instantâneo.
      await expect(trilhoDaFila(page)).toBeVisible()
      const reiniciar = page.getByRole('button', { name: 'Reiniciar o combate' })
      if ((await reiniciar.count()) > 0) {
        await acionaOCiclo(page, 'Reiniciar o combate', 'Reiniciar o combate?')
      }
      // O broadcast do servidor é o que confirma: sem esperar por ele, o teste
      // seguinte pode abrir a cena antes de a limpeza ter chegado. A confirmação
      // é o TRILHO ficar sem itens — a lista com o "Sem combatentes ainda" mora
      // na gaveta desde a ALE-198, e abrir gaveta para confirmar uma limpeza
      // seria medir o overlay em vez do estado.
      await expect
        .poll(async () => (await labelsNaFila(page)).length, { timeout: 7000 })
        .toBe(0)
      // E a base fica EM CENA (ALE-210). O reiniciar desliga a cena junto, e
      // sem isto todo teste que avança um turno teria de ligá-la de novo —
      // vinte chamadas dizendo a mesma coisa. Quem fala DA cena a manipula
      // explicitamente; para o resto, "em cena e vazia" é o começo do arquivo.
      await garanteACena(page)
    } finally {
      await contexto.close()
    }
  })

  /**
   * A sessão 4 começa VAZIA, sempre — e isto é conserto, não zelo.
   *
   * Os testes daqui são serial e cada um limpa o que criou, mas limpeza que
   * falha não limpa: um teste que estoura no meio deixa combatentes na seed
   * COMPARTILHADA, e o próximo mede o lixo em vez do app. Isso já produziu, no
   * mesmo dia, um guarda anunciando o combatente de outro teste, um estouro de
   * layout que não reproduzia sozinho e três execuções de CI vermelhas cuja
   * causa não era o código.
   *
   * O reinício é o mesmo gesto que o mestre tem na tela — a iniciativa, a
   * rodada e o turno voltam a zero —, e deixa o arquivo começando de onde a
   * seed do CI começa. Sem isto, "passa aqui e falha lá" continua sendo função
   * de quem rodou o quê antes.
   */

  test('Campanhas → campanha → continuar a sessão (realtime conectado)', async ({ page }) => {
    await page.goto('/campaigns')
    // O estado "ao vivo" chega DEPOIS da lista (fan-out separado de sessões) e
    // troca os botões do livro. Esperar ele assentar antes de clicar — senão a
    // ação some debaixo do cursor e o clique cai no botão vizinho (ALE-78).
    await expect(page.getByRole('button', { name: /^Continuar a sessão/ })).toBeVisible()

    await page.getByRole('button', { name: /^Abrir campanha/ }).click()
    await expect(page.getByRole('heading', { name: CAMPAIGN, level: 1 })).toBeVisible()

    await page.getByRole('button', { name: 'Continuar a sessão' }).click()
    await expect(page).toHaveURL(/\/campaigns\/\d+\/sessions\/\d+$/)

    // A superfície permanente é o TABULEIRO (ALE-198): é ele que prova que a
    // cena montou. A iniciativa virou gaveta e não serve mais de âncora.
    await expect(page.getByRole('navigation', { name: 'Consultas do mestre' })).toBeVisible()
    // The connection chip flips to "Conectado" only after the socket handshake.
    await expect(cenaViva(page)).toBeVisible()
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
    await expect(cenaViva(page)).toBeVisible()
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
    // A lista mora na GAVETA desde a ALE-198, e a regra que este teste protege
    // não mudou de lugar com ela: as quebras da linha são do CONTÊINER e não da
    // viewport, e a gaveta é um contêiner mais estreito (26rem) do que a coluna
    // de antes — se havia margem para o defeito voltar, é aqui.
    const fila = await abreAFila(page)
    await fila.getByRole('button', { name: 'Combatente' }).click()
    await page.getByLabel('Nome').fill(alvo)
    await page.getByRole('button', { name: 'Adicionar', exact: true }).click()
    await expect(fila.getByRole('button', { name: `Ferir ${alvo}` })).toBeVisible()

    const tracker = fila.locator('section').first()
    const escaping = await tracker.evaluate((section) => {
      const limit = section.getBoundingClientRect().right
      return [...section.querySelectorAll('button')]
        .filter((button) => button.getBoundingClientRect().right > limit)
        .map((button) => button.getAttribute('aria-label') ?? button.textContent)
    })

    await fila.getByRole('button', { name: `Remover ${alvo}` }).click()
    await expect(fila.getByRole('button', { name: `Ferir ${alvo}` })).toBeHidden()

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

    await page.getByRole('button', { name: 'Recuperar · cena' }).click()

    await expect(page.getByText('Efeitos temporários de cena foram limpos.')).toBeVisible()

    // A aba viva é o que estava em jogo, e a sonda tem de provar RESPOSTA: um
    // clique que só o main thread pode atender, e a tela mudando por causa
    // dele. Numa aba travada isto nunca resolve.
    //
    // Antes a sonda era "o avanço de turno está habilitado", e ela quebrou no
    // CI com a ALE-184: o botão agora TRAVA quando não há ninguém na
    // iniciativa, e a seed do CI vem com a lista vazia. Uma sonda de vida que
    // depende do estado do combate mede a coisa errada — passou aqui e falhou
    // lá, com o app perfeitamente vivo nos dois.
    await page.getByRole('button', { name: 'Configurações da sessão' }).click()
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 })
    await page.keyboard.press('Escape')
  })

  /**
   * A premissa do app, dita pelo dono: "sensação de jogo, não ter scroll,
   * mostrar os dados todos na tela". A cena do mestre foi reconstruída como
   * shell por causa disso (ALE-122) e a verificação viveu só numa mensagem de
   * commit. Aqui ela roda de novo a cada push, nos seis formatos.
   */
  test('a cena do mestre cabe na tela: a página não rola em nenhum formato', async ({ page }) => {
    await page.goto('/campaigns/1/sessions/4')
    await expect(cenaViva(page)).toBeVisible()

    await expectPageDoesNotScroll(page, VIEWPORTS)
  })

  /**
   * A MATRIZ DE ESTADOS (ALE-144), e ela ataca um mecanismo específico: todo
   * teste de layout desta suíte media logo depois do `goto` — iniciativa vazia,
   * nenhum combatente aberto, nenhum tabuleiro. **Os defeitos moram na cena
   * CHEIA.** Foi assim que o teste vizinho ("a cena do mestre cabe na tela")
   * ficou verde enquanto a barra de abas da ficha ia parar em y=872 numa janela
   * de 860 (ALE-125). Não faltou browser; faltou ESTADO.
   *
   * Cada estado é medido de duas formas complementares: a página não rola
   * (global e negativa, herdada) e nada é pintado para fora do pai (relação, a
   * que enxerga o que a primeira absorve).
   *
   * Um `goto` só para os quatro estados, porque montar é cumulativo: abrir um
   * combatente não desfaz a iniciativa.
   */
  test('a cena cabe na tela em cada ESTADO, não só na vazia', async ({ page }) => {
    await page.setViewportSize({ width: 1920, height: 1080 })
    await page.goto('/campaigns/1/sessions/4')
    await expect(page.getByRole('status', { name: 'Conectado' })).toBeVisible()

    // Da GAVETA e não do trilho, porque o trilho virou JANELA (ALE-211) — mas
    // lido DENTRO do estado que já a abre, e não numa abertura só para isto:
    // esta matriz mede quatro estados em seis formatos e vive perto do teto de
    // 30s, então um par abre/fecha a mais é o que a empurra para fora.
    let antes: string[] = []
    const cena = () => page.locator('.scene-grimorio').first()

    /**
     * Os estados são CUMULATIVOS e cada um deixa a cena como a encontrou de
     * overlay: a medida é da geometria PERMANENTE, e um diálogo aberto por
     * cima mediria o overlay em vez dela. Os dois estados de overlay
     * (ficha, catálogos) medem com ele aberto de propósito — é ali que a cena
     * de trás pode estourar — e fecham em seguida.
     */
    const ESTADOS: { nome: string; montar: () => Promise<void> }[] = [
      { nome: 'vazia', montar: async () => {} },
      {
        nome: 'grupo na fila',
        montar: async () => {
          const fila = await abreAFila(page)
          antes = await labelsNaGaveta(fila)
          await fila.getByRole('button', { name: 'Adicionar grupo' }).click()
          await expect(page.locator('[role="progressbar"][aria-label^="PM "]').first()).toBeVisible()
          await fechaAFila(page)
        },
      },
      {
        nome: 'ficha de PC aberta',
        montar: async () => {
          // O primeiro da fila serve: o grupo acabou de entrar e são todos PCs.
          // O trilho é o caminho de um clique, e o nome inteiro está no rótulo
          // acessível dele — não é preciso caçar o texto dentro da linha.
          await abreAFicha(page, await primeiroDaFila(page))
          await expect(page.getByRole('tab', { name: 'Perícias' })).toBeVisible()
        },
      },
      {
        nome: 'catálogos abertos',
        montar: async () => {
          await fechaAFicha(page)
          await abreConsulta(page, 'Catálogos')
          await expect(page.getByText('Abalado')).toBeVisible()
        },
      },
    ]

    for (const estado of ESTADOS) {
      await estado.montar()
      for (const viewport of VIEWPORTS) {
        await page.setViewportSize({ width: viewport.width, height: viewport.height })
        await expectPageDoesNotScroll(page, [viewport])
        await expectNadaEscapa(page, '.scene-grimorio')
      }
      await page.setViewportSize({ width: 1920, height: 1080 })
      await expect(cena()).toBeVisible()
    }

    await page.keyboard.press('Escape')
    const paraLimpar = await abreAFila(page)
    for (const label of await novosDesde(paraLimpar, antes)) {
      await paraLimpar.getByRole('button', { name: `Remover ${label}` }).click()
    }
  })

  /**
   * O nome do combatente cabe INTEIRO no laptop (ALE-167).
   *
   * O nome é o que o mestre lê em voz alta na mesa, e reticências numa linha
   * só cortavam quatro dos combatentes a 1440 e a 1024 — "agora é o Paladino
   * Sagra…". O nome passou a quebrar em DUAS linhas (`line-clamp-2`) em vez de
   * virar reticências.
   *
   * A medição é `scrollWidth`/`scrollHeight` contra o cliente: é assim que se
   * pergunta ao DOM "sobrou texto escondido?", nos dois eixos — só a largura
   * deixaria passar um nome cortado na segunda linha.
   *
   * Por que e2e: depende da fonte real e da largura real da coluna. Em jsdom
   * todo elemento mede zero e nada nunca transborda.
   */
  test('o nome do combatente cabe inteiro no laptop', async ({ page }) => {
    await page.setViewportSize({ width: 1440, height: 900 })
    await page.goto('/campaigns/1/sessions/4')
    await expect(cenaViva(page)).toBeVisible()
    // A caixa que decide se o nome cabe é a GAVETA da fila (26rem), e não mais
    // uma coluna cuja largura variava com o tabuleiro aberto (ALE-198). O
    // encerra-se-houver continua: uma cena montada muda o que os testes
    // seguintes medem, e a ordem entre specs não é contrato.
    await encerraOTabuleiroSeHouver(page)
    const fila = await abreAFila(page)

    // 22 caracteres, o comprimento do pior caso da seed ("Guerreiro Veterano
    // Nv8"), com sufixo curto só para o teste achar e remover o que criou.
    // O tamanho é a metade do teste que importa: com 20 caracteres o nome
    // cabia por UM pixel e o guarda passava verde sobre o defeito — descobri
    // isso sabotando, e é o segundo teste desta leva que a sabotagem salvou.
    const nome = `Guerreiro Veterano ${Date.now() % 1000}`
    await fila.getByRole('button', { name: 'Combatente' }).click()
    await page.getByLabel('Nome').fill(nome)
    // COM PV, e isso é metade do teste: sem barras de vida a linha devolve os
    // ~176px delas ao nome, ele cabe, e o guarda passa verde sobre o defeito.
    // O caso real da mesa é um combatente COM vida.
    await page.locator('#combatant-hp').fill('30')
    await page.getByRole('button', { name: 'Adicionar', exact: true }).click()
    await expect(fila.getByRole('button', { name: `Remover ${nome}` })).toBeVisible()
    await fila.getByRole('button', { name: 'Fechar', exact: true }).click()

    try {
      const escondido = await page.evaluate((alvo) => {
        const el = [...document.querySelectorAll<HTMLElement>('button, span')].find(
          (n) => n.textContent?.trim() === alvo,
        )
        if (!el) return null
        return {
          largura: [Math.round(el.scrollWidth), Math.round(el.clientWidth)],
          altura: [Math.round(el.scrollHeight), Math.round(el.clientHeight)],
        }
      }, nome)

      expect(escondido, `não achei o nome "${nome}" na lista`).not.toBeNull()
      const [precisaW, mostraW] = escondido?.largura ?? [0, 0]
      const [precisaH, mostraH] = escondido?.altura ?? [0, 0]
      expect(precisaW, 'o nome está cortado na largura').toBeLessThanOrEqual(mostraW + 1)
      expect(precisaH, 'o nome está cortado na altura').toBeLessThanOrEqual(mostraH + 1)
    } finally {
      await fila.getByRole('button', { name: `Remover ${nome}` }).click()
    }
  })

  /**
   * No celular deitado a lista mostra DOIS combatentes, não um (ALE-164).
   *
   * O defeito era de ocupação: em 844×390 o cabeçalho do painel e a fileira de
   * "Adicionar grupo"/"+ Combatente" somavam 93px em duas faixas, a primeira
   * linha nascia em y=301 e sobravam 89px de lista — UM combatente de nove,
   * cortado ao meio, com 77% da tela virada moldura. Celular deitado é uma
   * forma muito natural de segurar o telefone na mesa.
   *
   * As ações subiram para o cabeçalho, que ficou com o lado direito vazio na
   * ALE-184: uma faixa de 65px no lugar de duas, primeira linha em y=261.
   *
   * A promessa SOBREVIVEU à mudança de casa (ALE-198): a lista virou a folha de
   * baixo do celular, com teto de 92dvh, e a pergunta continua sendo a mesma —
   * numa tela de 390px de altura, dá para ler dois combatentes? O que mudou é
   * que agora ela não divide a altura com a faixa do turno nem com o mapa.
   *
   * O que se afirma é o RESULTADO (a segunda linha começa dentro da tela), e
   * não a altura de nenhuma faixa: pixel de cromo é detalhe de implementação,
   * "dá para ver dois combatentes" é a promessa.
   *
   * Por que e2e: ocupação de tela contra a janela real. Em jsdom todo elemento
   * mede zero e qualquer arranjo "cabe".
   */
  test('no celular deitado, a lista mostra dois combatentes', async ({ page }) => {
    await page.setViewportSize({ width: 844, height: 390 })
    await page.goto('/campaigns/1/sessions/4')
    await expect(cenaViva(page)).toBeVisible()

    // Os combatentes são DESTE teste e saem no fim — a seed é compartilhada, e
    // contar com o que outro teste deixou é o acoplamento que já derrubou esta
    // suíte no CI mais de uma vez.
    const marca = Date.now()
    const nomes = [`Alvo A ${marca}`, `Alvo B ${marca}`]
    const fila = await abreAFila(page)
    await fila.getByRole('button', { name: 'Combatente' }).click()
    for (const nome of nomes) {
      await page.getByLabel('Nome').fill(nome)
      await page.getByRole('button', { name: 'Adicionar', exact: true }).click()
      await expect(fila.getByRole('button', { name: `Remover ${nome}` })).toBeVisible()
    }
    // O formulário fica aberto enquanto se adiciona vários (ALE-122); fechá-lo
    // é o que devolve a altura para a lista.
    await fila.getByRole('button', { name: 'Fechar', exact: true }).click()

    try {
      const linhas = fila.getByRole('button', { name: /^Mudar a iniciativa de/ })
      await expect(linhas.first()).toBeInViewport()
      // METADE da segunda linha, e não "ela começa dentro da tela": com a
      // fileira de ações de volta acima da lista, a segunda linha ainda NASCE
      // em y=374, e só 16 dos 65px dela aparecem — a asserção fraca passava
      // verde sobre o defeito, e eu só descobri isso ao sabotar. Ler o nome e
      // os PV do segundo combatente é a promessa; um filete não é.
      await expect(linhas.nth(1), 'só dá para ver um combatente').toBeInViewport({
        ratio: 0.5,
      })
    } finally {
      for (const nome of nomes) {
        await fila.getByRole('button', { name: `Remover ${nome}` }).click()
      }
    }
  })

  /**
   * O avanço trunca o nome LONGO em vez de pintar para fora (ALE-184).
   *
   * O botão anuncia quem entra, então o rótulo tem o tamanho do nome que o
   * mestre digitou — e um invólucro que não encolhe fica com a largura de
   * max-content: medido em 460px numa janela de 390. Esse era o `shrink-0` que
   * eu herdei do par de ícones, onde o conteúdo media 32px fixos.
   *
   * Por que e2e: `truncate` só encolhe se TODO ancestral flex puder encolher, e
   * o `min-width: auto` de um item flex é o min-content dele — que num
   * `white-space: nowrap` é o texto inteiro. Isso é cascata de layout real; em
   * jsdom todo elemento mede zero e a cadeia inteira passa verde.
   *
   * O CI achou isto antes de mim, e por um caminho torto: um nome de 32
   * caracteres cabia na minha máquina e estourava na do runner, porque a
   * quebra de linha da faixa depende da largura dos vizinhos, que depende da
   * métrica da fonte instalada. O nome longo daqui torna a falha determinística
   * nas duas.
   */
  test('o avanço trunca o nome longo em vez de estourar a faixa', async ({ page }) => {
    await page.setViewportSize({ width: 390, height: 844 })
    await page.goto('/campaigns/1/sessions/4')
    await expect(cenaViva(page)).toBeVisible()

    // O combatente é DESTE teste e sai no fim: a seed é compartilhada, e um
    // nome de 47 caracteres esquecido na lista muda o que todo teste de layout
    // depois dele mede.
    const marca = Date.now()
    const longo = `Zumbi Putrefato Ancião do Pântano ${marca}`
    const fila = await abreAFila(page)
    await fila.getByRole('button', { name: 'Combatente' }).click()
    await page.getByLabel('Nome').fill(longo)
    // Iniciativa ALTA de propósito: o rótulo do avanço nomeia o PRÓXIMO da
    // ordem, e com o valor padrão (0) qualquer resto deixado por outro teste
    // empata e pode vir antes — foi assim que este guarda falhou no CI,
    // anunciando o combatente de OUTRO teste. 99 põe o meu em primeiro sem
    // depender de quem mais está na lista.
    await page.locator('#combatant-initiative').fill('99')
    await page.getByRole('button', { name: 'Adicionar', exact: true }).click()
    await expect(fila.getByRole('button', { name: `Remover ${longo}` })).toBeVisible()
    // A faixa a medir é a de TRÁS: com a gaveta aberta por cima, o que se
    // mediria era o overlay.
    await fechaAFila(page)

    try {
      // O rótulo do avanço carrega o nome inteiro; o que não pode é a CAIXA
      // dele passar da faixa.
      await expect(avancoDeTurno(page)).toHaveAccessibleName(new RegExp(String(marca)))
      await expectNadaEscapa(page, '.scene-grimorio')
    } finally {
      const paraLimpar = await abreAFila(page)
      await paraLimpar.getByRole('button', { name: `Remover ${longo}` }).click()
    }
  })

  /**
   * EXATAMENTE um avanço de turno na tela, em todo formato (ALE-142).
   *
   * O avanço mora na FAIXA DO TURNO em toda largura desde a ALE-198 — a fila
   * virou gaveta, e o botão mais clicado da sessão não pode viver atrás de um
   * clique. Continuam sendo duas instâncias separadas por `lg:hidden` /
   * `hidden lg:flex`: abaixo do degrau só o avanço, acima dele o par com o
   * `‹`, porque os 44px do desfazer não cabem no orçamento de cromo da faixa a
   * 844×390 (ALE-146). O desenho inteiro depende de nunca aparecerem juntas —
   * duas seriam o defeito que a ALE-122 já tinha consertado uma vez.
   *
   * Zero também é falha, e é a mais grave: sem o avanço o mestre não joga.
   */
  test('há um e só um avanço de turno na tela, em todo formato', async ({ page }) => {
    await page.goto('/campaigns/1/sessions/4')
    await expect(cenaViva(page)).toBeVisible()
    // O avanço só existe DENTRO da cena (ALE-210): fora dela a vaga é do
    // "Iniciar cena", e este guarda mede o avanço, não a vaga.
    await garanteACena(page)

    for (const viewport of VIEWPORTS) {
      await page.setViewportSize({ width: viewport.width, height: viewport.height })

      // As DUAS regiões, e a de combate é a que importa: com a mesa aberta a
      // iniciativa nem existe na página, então contar só ali nunca veria duas.
      // Aprendido tentando sabotar este teste — a primeira versão passava verde
      // com o par duplicado, porque só olhava a mesa. Não há mais REGIÃO para
      // alternar (ALE-198): a cena tem uma superfície permanente só, e contar
      // na página inteira é agora a única contagem que existe.
      const avancos = avancoDeTurno(page)
      await expect(avancos, `${viewport.name}: avanços de turno na tela`).toHaveCount(1)
      await expect(avancos, `${viewport.name}: o avanço saiu da tela`).toBeInViewport()

      // O tamanho é o ponto da ALE-184: 34px num controle clicado uma vez por
      // combatente, por rodada, a noite inteira. 44 é o mínimo de toque, e só
      // o browser real mede altura.
      const caixa = await avancos.boundingBox()
      expect(caixa?.height ?? 0, `${viewport.name}: altura do avanço`).toBeGreaterThanOrEqual(44)
    }
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
    await expect(cenaViva(page)).toBeVisible()

    // O teste traz o PRÓPRIO grupo: a iniciativa da seed do CI está VAZIA, e
    // depender de um PC que só existe no banco de dev já quebrou o CI três vezes
    // nesta issue. "Adicionar grupo" é idempotente e traz os PCs da campanha.
    // O instantâneo sai da GAVETA e não do trilho: desde a ALE-211 o trilho é
    // uma janela, e um `antes` incompleto faz a limpeza do fim acusar como novo
    // quem só estava fora da vista.
    const fila = await abreAFila(page)
    const antes = await labelsNaGaveta(fila)
    await fila.getByRole('button', { name: 'Adicionar grupo' }).click()
    await expect(page.locator('[role="progressbar"][aria-label^="PM "]').first()).toBeVisible()
    await fechaAFila(page)

    // O primeiro da fila serve: o grupo acabou de entrar e são todos PCs, que
    // são os únicos com ficha atrás — o conteúdo mais alto que entra na região.
    const nomeDoPc = await primeiroDaFila(page)

    await abreAFicha(page, nomeDoPc)
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

    // `exact` porque o diálogo da ficha tem um título só para leitor de tela,
    // "Ficha de <nome>", e o casamento por nome do Playwright é SUBSTRING.
    const nomeNaFaixa = page.getByRole('heading', { name: nomeDoPc, exact: true })
    const fechar = page.getByRole('button', { name: 'Fechar o combatente' })

    for (const viewport of VIEWPORTS) {
      await page.setViewportSize({ width: viewport.width, height: viewport.height })
      // Sem região para alternar (ALE-198): a ficha é um diálogo, e ele está na
      // tela em toda largura.
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

      // O celular deitado continua FORA desta conta. A ALE-198 devolveu altura
      // — a ficha virou diálogo e não divide mais a região com a faixa do turno
      // nem com o seletor de região, que deixou de existir —, mas 390px de
      // altura menos a moldura do diálogo ainda não dão os 65% que o teto de
      // 35% exige da ficha. A garantia que vale aqui é a de ALCANCE, logo
      // acima, e essa roda nos seis.
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
      // Sai por dentro do popover, onde as três estão listadas juntas. O
      // popover é achado pelo que ele NÃO contém: desde a ALE-198 a ficha é ela
      // mesma um diálogo, e `getByRole('dialog')` sozinho casa os dois — o de
      // fora nunca fecha, e a espera estourava sobre uma limpeza que funcionou.
      // Filtrar pelo que ele CONTÉM não resolve: com duas condições ou menos a
      // própria faixa desenha os "Remover condição" soltos, dentro da ficha.
      // A barra de blocos é o que só a ficha tem.
      await gatilho.click()
      const painel = page
        .getByRole('dialog')
        .filter({ hasNot: page.getByRole('tab', { name: 'Perícias' }) })
      await painel.getByRole('button', { name: /^Remover condição/ }).first().click()
      await page.keyboard.press('Escape')
      await expect(painel).toHaveCount(0)
    }
    await expect(gatilho).toHaveCount(0)

    // Tira da fila só quem ESTE teste pôs.
    await fechaAFicha(page)
    const paraLimpar = await abreAFila(page)
    for (const label of await novosDesde(paraLimpar, antes)) {
      await paraLimpar.getByRole('button', { name: `Remover ${label}` }).click()
    }
  })

  /**
   * O modo "Lado a lado" das notas só existe onde ele CABE (ALE-139).
   *
   * Por que e2e: a decisão é a largura MEDIDA da região, via `ResizeObserver`.
   * Em jsdom não há observador nem leiaute, a largura é zero e a regra responde
   * "cabe" para sempre — o caminho estreito não existe em camada mais barata.
   *
   * Os dois formatos saem da MESMA página, e a ALE-198 INVERTEU a direção que
   * a ALE-139 tinha registrado aqui. Antes a região media 801px numa janela de
   * 1440 e 606px numa de 1920 — maximizar a janela ESTREITAVA a região, porque
   * a cena repartia o workspace por proporções condicionais. Agora a coluna das
   * notas é uma fração do palco (40%, entre 22rem e 44rem): ~509px a 1440, onde
   * o modo duplo NÃO cabe, e ~700px a 1920, onde cabe. Janela maior, região
   * maior.
   */
  test('o modo lado a lado só aparece onde a região comporta duas colunas', async ({
    page,
  }) => {
    await page.goto('/campaigns/1/sessions/4')
    await expect(cenaViva(page)).toBeVisible()

    await page.setViewportSize({ width: 1920, height: 1080 })
    await abreConsulta(page, 'Notas')
    const faixa = page.getByRole('group', { name: /Modo de visualização das notas/ })
    await expect(faixa.getByRole('button', { name: 'Lado a lado' })).toBeVisible()
    await expect(page.getByLabel('Notas da sessão')).toBeVisible()

    // Região estreita: o modo duplo some, e "Escrever" assume — a faixa tem de
    // dizer a verdade sobre o que está na tela, senão sobram dois botões e
    // nenhum marcado.
    await page.setViewportSize({ width: 1440, height: 900 })
    await page.waitForTimeout(600)
    await expect(faixa.getByRole('button', { name: 'Lado a lado' })).toHaveCount(0)
    await expect(faixa.getByRole('button', { name: 'Escrever' })).toHaveAttribute(
      'aria-pressed',
      'true',
    )
    await abreConsulta(page, 'Notas')
  })

  /**
   * Os descansos saem da fileira de turno no palco BAIXO, e continuam
   * alcançáveis (ALE-146).
   *
   * A 844×390 a faixa de turno enrolava em duas fileiras e media 90px — mais
   * que as duas barras de navegação somadas — num palco onde o conteúdo já
   * recebia 138px. Com os descansos fora da fileira ela desenrola para 62.
   *
   * Por que e2e: a decisão é a ALTURA MEDIDA do palco, via `ResizeObserver`.
   * Em jsdom não há observador nem leiaute, a altura é zero e a regra responde
   * "não é baixo" para sempre — nenhuma camada mais barata consegue ver isto.
   *
   * As DUAS metades são afirmadas de propósito. Só a primeira passaria verde
   * com o conserto errado, que é apagar os descansos do celular: eles têm de
   * SUMIR da fileira e APARECER no menu.
   */
  test('no palco baixo os descansos vão para o menu, e continuam alcançáveis', async ({
    page,
  }) => {
    await page.goto('/campaigns/1/sessions/4')
    await expect(cenaViva(page)).toBeVisible()

    const naFileira = page.getByRole('button', { name: /Recuperar · cena/ })

    // Palco alto: eles ficam onde a ALE-122 os pôs.
    await page.setViewportSize({ width: 768, height: 1024 })
    await expect(naFileira, 'no tablet em pé o descanso saiu da fileira').toBeVisible()

    // Palco baixo: saem da fileira...
    await page.setViewportSize({ width: 844, height: 390 })
    await expect(naFileira, 'no celular deitado o descanso continuou na fileira').toHaveCount(0)

    // ...e estão no menu da sessão, que é o que separa "mudou de casa" de
    // "sumiu".
    await page.getByRole('button', { name: 'Configurações da sessão' }).click()
    const painel = page.getByRole('dialog')
    await expect(
      painel.getByRole('button', { name: /Recuperar · cena/ }),
      'o descanso sumiu em vez de mudar de casa',
    ).toBeVisible()

    await page.keyboard.press('Escape')
    await expect(painel).toBeHidden()
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
    // ESPERA a caixa assentar antes de medir. A ficha vive num diálogo `fixed`
    // com altura em `vh`, e depois de um `setViewportSize` o
    // `chrome-headless-shell` leva um quadro (às vezes mais) para reresolver
    // isso — medindo na hora, a proporção sai contra a altura ANTERIOR e o
    // teste acusa a tela por um defeito do renderizador. Foi assim que este
    // guarda caiu três vezes na suíte cheia e passou isolado todas elas.
    //
    // O critério é genérico de propósito: duas leituras iguais em quadros
    // seguidos. Afirmar "a altura tem de ser 92vh" copiaria a regra do CSS
    // para dentro do teste, e aí ele passaria a proteger a cópia.
    await page.waitForFunction(
      () => {
        const janela = window as unknown as { __ultimaAltura?: number }
        const caixa = document.querySelector('[data-slot="dialog-content"]')
        if (!caixa) return false
        const agora = Math.round(caixa.getBoundingClientRect().height)
        const estavel = janela.__ultimaAltura === agora
        janela.__ultimaAltura = agora
        return estavel
      },
      undefined,
      { polling: 'raf', timeout: 5_000 },
    )

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
    await expect(cenaViva(page)).toBeVisible()

    // O instantâneo sai da GAVETA e não do trilho: desde a ALE-211 o trilho é
    // uma janela, e um `antes` incompleto faz a limpeza do fim acusar como novo
    // quem só estava fora da vista.
    const fila = await abreAFila(page)
    const antes = await labelsNaGaveta(fila)
    await fila.getByRole('button', { name: 'Adicionar grupo' }).click()
    await expect(page.locator('[role="progressbar"][aria-label^="PM "]').first()).toBeVisible()
    await fechaAFila(page)
    await abreAFicha(page, await primeiroDaFila(page))

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

      // A relação é filho contra PAI (`expectNadaEscapa`, ALE-144): a primeira
      // versão media contra a COLUNA e passava verde, porque o crachá vazava do
      // CARTÃO 6px, muito antes de chegar perto da borda da coluna.
      await expectNadaEscapa(page, 'section:has([role="progressbar"][aria-label="Vida"])')
    }

    // Sai como entrou.
    await fechaAFicha(page)
    const paraLimpar = await abreAFila(page)
    for (const label of await novosDesde(paraLimpar, antes)) {
      await paraLimpar.getByRole('button', { name: `Remover ${label}` }).click()
    }
  })

  /**
   * O trilho RECORTA a fila em vez de rolar, e mantém quem está na vez à vista
   * (ALE-211).
   *
   * A fila recolhida deixou de rolar: ela mostra uma janela centrada na vez,
   * com quem já agiu acima e quem ainda vai abaixo. Quantos vizinhos cabem é
   * conta medida por observador, e é aí que está o risco — com um número errado
   * o trilho MENTE sobre quem já jogou, ou transborda o bloco.
   *
   * Por que e2e: em jsdom todo elemento mede ZERO, então `cabemNaFila` responde
   * zero, a janela vira "mostra tudo" e a asserção passaria verde sobre um
   * trilho quebrado. É a mesma cegueira que a ALE-139 documentou. A regra em si
   * é unitária (`rail-geometry.test.ts`); o que só o navegador testemunha é que
   * a MEDIÇÃO chegou e recortou.
   */
  test('o trilho recorta a fila e nunca perde de vista quem está na vez', async ({ page }) => {
    // 1024 de largura para o trilho existir, e 768 de altura porque é o formato
    // em que ele mais aperta — quanto menor a altura, menor a janela.
    await page.setViewportSize({ width: 1024, height: 768 })
    await page.goto('/campaigns/1/sessions/4')
    await expect(cenaViva(page)).toBeVisible()
    await garanteACena(page)

    const fila = await abreAFila(page)
    const antes = await labelsNaGaveta(fila)
    await fila.getByRole('button', { name: 'Adicionar grupo' }).click()
    await expect(page.locator('[role="progressbar"][aria-label^="PM "]').first()).toBeVisible()
    // Mais gente do que cabe: os PCs da campanha sozinhos ainda cabem num
    // laptop, então o teste CRIA o excedente em vez de contar com a seed.
    await fila.getByRole('button', { name: 'Combatente' }).click()
    for (const nome of RECHEIO_DO_TRILHO) {
      await page.getByLabel('Nome').fill(nome)
      await page.getByRole('button', { name: 'Adicionar', exact: true }).click()
      await expect(fila.getByRole('button', { name: `Remover ${nome}` })).toBeVisible()
    }
    const naFila = (await labelsNaGaveta(fila)).length
    await fechaAFila(page)

    // O recorte: o trilho mostra MENOS do que a fila tem. Sem esta metade o
    // teste passaria verde num trilho que desenha todo mundo e transborda.
    await expect
      .poll(async () => (await labelsNaFila(page)).length, { timeout: 7000 })
      .toBeLessThan(naFila)
    expect(naFila, 'a fila precisa ser maior que a janela para o teste medir algo')
      .toBeGreaterThan(1)

    // E a outra metade, que é a razão de o bloco existir: passe o turno onde
    // passar, quem está na vez continua desenhado.
    for (let volta = 0; volta < 4; volta++) {
      await avancoDeTurno(page).first().click()
      const naVez = page.getByRole('button', { name: /— na vez$/ })
      await expect(naVez, `turno ${volta + 1}: a vez sumiu do trilho`).toHaveCount(1)
      await expect(naVez).toBeInViewport()
    }

    const paraLimpar = await abreAFila(page)
    for (const label of await novosDesde(paraLimpar, antes)) {
      await paraLimpar.getByRole('button', { name: `Remover ${label}` }).click()
    }
  })

  /**
   * As consultas do mestre CABEM na fileira a 390 (ALE-221).
   *
   * A fileira é `overflow-x-auto`, então nada quebra quando ela estoura — e é
   * justamente por isso que o defeito é invisível: medido, a sexta consulta
   * passava SETE pixels do trilho, e sete pixels não parecem roláveis. O último
   * ícone ficava cortado sem afordância nenhuma, e o último ícone é sempre o
   * mais novo. Eu mesmo introduzi isso ao acrescentar "Regras" e só peguei
   * medindo.
   *
   * A asserção é o CONTEÚDO contra a JANELA, e não uma contagem de ícones nem
   * uma largura fixa: ela sobrevive a mudar padding, gap ou ordem, e falha no
   * dia em que a sétima consulta chegar — que é exatamente quando alguém
   * precisa decidir o que fazer com ela.
   *
   * Só o browser mede isto. Em jsdom `scrollWidth` e `clientWidth` são ambos
   * zero, e a comparação passa verde sobre um trilho estourado.
   */
  test('as consultas do mestre cabem na fileira no telefone em pé', async ({ page }) => {
    await page.setViewportSize({ width: 390, height: 844 })
    await page.goto('/campaigns/1/sessions/4')
    await expect(cenaViva(page)).toBeVisible()

    const trilho = page.getByRole('navigation', { name: 'Consultas do mestre' })
    await expect(trilho).toBeVisible()
    // Espera a caixa assentar: depois de um `setViewportSize` o
    // `chrome-headless-shell` leva um quadro para reresolver o arranjo, e ler
    // antes disso mede o formato ANTERIOR (ALE-200).
    await expect
      .poll(
        async () =>
          trilho.evaluate((el) => `${el.scrollWidth}|${el.clientWidth}`),
        { polling: 'raf', timeout: 5000 },
      )
      .toBe(await trilho.evaluate((el) => `${el.scrollWidth}|${el.clientWidth}`))

    const { conteudo, janela } = await trilho.evaluate((el) => ({
      conteudo: el.scrollWidth,
      janela: el.clientWidth,
    }))
    expect(janela, 'o trilho não foi medido').toBeGreaterThan(0)
    expect(
      conteudo,
      `as consultas ocupam ${conteudo}px numa fileira de ${janela}px — a última fica cortada`,
    ).toBeLessThanOrEqual(janela)
  })

  /**
   * Cada verbo da linha da iniciativa ocupa a MESMA coluna em todas as linhas.
   *
   * O conjunto de botões muda por linha com razão — o olho só existe em linha
   * com vida, remover só para o mestre —, mas a POSIÇÃO não podia mudar junto:
   * a fileira encolhia e o `+` de uma linha caía onde estava o lápis de outra.
   * Medido antes: "Curar" aparecia em dois X (256 e 220), "Ferir" em 294 e 258
   * — 36px de deslocamento, que é a largura de um botão (ALE-141).
   *
   * Só o browser mede isto; em jsdom todo elemento fica em x=0 e a asserção
   * passaria verde sobre qualquer serrilhado.
   */
  test('os verbos da linha da iniciativa ficam na mesma coluna', async ({ page }) => {
    await page.setViewportSize({ width: 1920, height: 1080 })
    await page.goto('/campaigns/1/sessions/4')
    await expect(cenaViva(page)).toBeVisible()

    // Precisa de linhas HETEROGÊNEAS: os PCs da campanha entram com vida (e
    // portanto com olho), e a seed tem NPC sem vida, que é o caso sem olho.
    // O instantâneo sai da GAVETA e não do trilho: desde a ALE-211 o trilho é
    // uma janela, e um `antes` incompleto faz a limpeza do fim acusar como novo
    // quem só estava fora da vista.
    const fila = await abreAFila(page)
    const antes = await labelsNaGaveta(fila)
    await fila.getByRole('button', { name: 'Adicionar grupo' }).click()
    await expect(page.locator('[role="progressbar"][aria-label^="PM "]').first()).toBeVisible()

    for (const verbo of ['Curar', 'Ferir', 'Editar PV', 'Ocultar PV', 'Remover']) {
      await expectFormaColuna(page, `button[aria-label^="${verbo} "]`)
    }

    for (const label of await novosDesde(fila, antes)) {
      await fila.getByRole('button', { name: `Remover ${label}` }).click()
    }
  })

  /**
   * Com o tabuleiro ABERTO no telefone, todo controle continua alcançável
   * (ALE-178).
   *
   * O defeito: o cabeçalho do tabuleiro quebrava linha, mas a fileira interna
   * de controles não — a 390px "Trazer a iniciativa" saía cortado e o ✕ de
   * encerrar ficava FORA da tela. Quem abrisse um tabuleiro pelo telefone não
   * tinha como fechá-lo.
   *
   * Por que e2e: é a mesma família da ALE-160 — o alvo fora da janela sem eixo
   * que role até ele, que só a medição contra a viewport real acusa.
   */
  test('com o tabuleiro aberto no telefone, nada some fora da tela', async ({ page }) => {
    await page.setViewportSize({ width: 390, height: 844 })
    await page.goto('/campaigns/1/sessions/4')
    await expect(cenaViva(page)).toBeVisible()

    // As PEÇAS deste teste são criadas por ele: contar com a iniciativa da seed
    // era o acoplamento que o derrubou no CI, onde ela nasce VAZIA. A forma
    // mora na gaveta da fila (ALE-198), e o tabuleiro continua na tela por
    // baixo dela — não há mais região para alternar.
    const fila = await abreAFila(page)
    await fila.getByRole('button', { name: 'Combatente' }).click()
    for (const nome of PECAS_DO_TELEFONE) {
      await page.getByLabel('Nome').fill(nome)
      await page.getByRole('button', { name: 'Adicionar', exact: true }).click()
      await expect(fila.getByRole('button', { name: `Remover ${nome}` })).toBeVisible()
    }
    await fechaAFila(page)

    // Setup que se limpa sozinho: a seed é compartilhada e um tabuleiro
    // esquecido aberto por uma execução anterior faria este teste procurar um
    // botão que não existe — foi o que aconteceu quando a primeira versão
    // falhou no meio e deixou a cena montada.
    await encerraOTabuleiroSeHouver(page)
    await page.getByRole('button', { name: 'Abrir tabuleiro' }).click()
    await page.getByLabel('Lugar').fill('Cripta do teste')
    await page.getByRole('button', { name: 'Abrir', exact: true }).click()
    await expect(page.getByText('Cripta do teste')).toBeVisible()

    // POVOAR é parte do teste, não cenário: com o tabuleiro vazio o cabeçalho
    // diz "0 peças" e cabe; cheio, ele empurra os controles para fora. Medir a
    // cena vazia era exatamente a cegueira que a ALE-144 documentou, e aqui ela
    // custou uma versão deste teste que passava verde sobre o defeito.
    //
    // O que este teste precisa é de um cabeçalho CHEIO, não de um NÚMERO. Ele já
    // prendeu "9 peças" — o tamanho da iniciativa que o banco de
    // desenvolvimento acumulou — e caiu no CI, onde a seed nasce vazia e o
    // cabeçalho dizia "4 peças". Contar a lista antes de povoar também não
    // serve: outro worker mexe nela entre a contagem e o clique.
    // O clique esquerdo PERGUNTA quem vem (ALE-204): o padrão traz só os
    // jogadores, e aqui o teste quer o cabeçalho CHEIO — "Todos".
    await page.getByRole('button', { name: /Trazer a iniciativa/ }).click()
    await page.getByRole('dialog').getByRole('button', { name: 'Todos' }).click()
    await page.getByRole('dialog').getByRole('button', { name: /^Trazer \d/ }).click()
    await expect(page.getByText(/[1-9]\d* peças/)).toBeVisible()

    // Duas asserções, e cada uma diz uma coisa: nada fora da janela sem
    // caminho, E nenhum painel rolando de lado. O defeito desta issue passava
    // pela primeira com razão — o painel ROLAVA, então havia como chegar ao ✕
    // — e é a segunda que o pega, porque rolar de lado atrás do botão de
    // fechar é justamente o que a regra da casa proíbe.
    await expectDentroDaJanela(page)
    await expectNadaRolaDeLado(page, '.scene-grimorio')

    // Encerra pelo próprio botão que o defeito escondia — isto é asserção
    // disfarçada de limpeza: se o ✕ tivesse saído da tela, o clique não
    // aconteceria. E a seed é compartilhada: um tabuleiro esquecido aberto
    // sobrevive ao reinício desde que a persistência passou a funcionar
    // (ALE-124), e o teste seguinte pode ser o que EXIGE a mesa vazia.
    await page.getByRole('button', { name: 'Encerrar o tabuleiro' }).click()
    await page.getByRole('dialog').getByRole('button', { name: 'Encerrar' }).click()
    await expect(page.getByText('Nenhum tabuleiro aberto')).toBeVisible()
    // O DIÁLOGO precisa ter ido embora antes de mexer na cena atrás dele: o
    // Kobalte marca os irmãos do modal como `aria-hidden`, então enquanto ele
    // fecha o `getByRole` não acha NADA da página — e um `if (isVisible())` dá
    // falso em silêncio. Foi assim que esta limpeza parou de trocar de região
    // no CI: ela pulava o clique, procurava "Remover" na região errada e
    // estourava em 30s. A regra está no CLAUDE.md do front; o texto de
    // "Nenhum tabuleiro aberto" aparece ANTES de o diálogo terminar de sair.
    await expect(page.getByRole('dialog')).toBeHidden()

    // E leva embora os combatentes que criou, na gaveta onde eles moram.
    const paraLimpar = await abreAFila(page)
    for (const nome of PECAS_DO_TELEFONE) {
      await paraLimpar.getByRole('button', { name: `Remover ${nome}` }).click()
    }
  })

  /**
   * A ficha aberta DENTRO da cena não vaza quando a coluna aperta (ALE-188).
   *
   * Achado pelo CI, e por um caminho que vale registrar: enquanto o clique do
   * teste dos ESTADOS caía no botão do som, a ficha nunca abria lá e a tela
   * quebrada passava VERDE. Consertado o clique, o runner acusou cinco linhas
   * de perícia pintando para fora do pai.
   *
   * O defeito: item de grade não encolhe abaixo do conteúdo e o `<select>` de
   * atributo tem largura intrínseca — bastavam ~15px a menos na coluna para a
   * linha inteira transbordar. Quinze pixels é o que a barra de rolagem CLÁSSICA
   * ocupa (Linux e Windows a desenham dentro da caixa; a sobreposta do meu
   * desktop não ocupa nada), e é por isso que a tela passava aqui e vazava lá.
   *
   * Por isso este teste mede a 375 — os 390 do menor formato da casa MENOS a
   * barra — em vez de depender do tipo de barra da máquina. Medido antes do
   * conserto: 6px de sobra a 375, 21px a 360.
   */
  test('a ficha aberta na cena não vaza com a coluna apertada', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 844 })
    await page.goto('/campaigns/1/sessions/4')
    await expect(cenaViva(page)).toBeVisible()

    // A 375px o trilho da fila não existe (ele começa em 1024): quem lê a fila e
    // quem abre a ficha é a própria gaveta, do começo ao fim.
    const fila = await abreAFila(page)
    const antes = await labelsNaGaveta(fila)
    await fila.getByRole('button', { name: 'Adicionar grupo' }).click()
    await expect(page.locator('[role="progressbar"][aria-label^="PM "]').first()).toBeVisible()
    const depois = await labelsNaGaveta(fila)
    await abreAFichaPelaGaveta(page)
    await page.getByRole('tab', { name: 'Perícias' }).click()
    await expect(page.getByRole('heading', { name: 'Perícias' })).toBeVisible()

    // A ficha é DIÁLOGO agora, e ela é o pai que interessa: `.scene-grimorio` é
    // a cena de trás, e num telefone o diálogo cobre a tela inteira. Medir a
    // cena de trás com o overlay aberto mediria o pai errado.
    await expectNadaEscapa(page, 'section:has([role="progressbar"][aria-label="Vida"])')

    await fechaAFicha(page)
    const paraLimpar = await abreAFila(page)
    for (const label of depois.filter((nome) => !antes.includes(nome))) {
      await paraLimpar.getByRole('button', { name: `Remover ${label}` }).click()
    }
  })

  /**
   * Com o acervo cheio, "Abrir tabuleiro" continua alcançável (ALE-124).
   *
   * A cena vazia passou a mostrar os Lugares da campanha (fatia 5), e ela
   * centrava o conteúdo com `justify-center` DENTRO de uma caixa que rola.
   * Centrar conteúdo que transborda empurra o TOPO para fora da área rolável —
   * e o topo aqui é justamente o botão de abrir. O mestre com algumas cenas
   * guardadas perdia o botão de abrir a próxima.
   *
   * A ALE-198 tirou a LISTA daquela tela — o acervo virou um botão que abre o
   * diálogo —, e este guarda continua valendo pelo mesmo motivo pelo qual ele
   * nasceu: ele afirma que o botão de abrir RECEBE O CLIQUE numa janela baixa,
   * e é essa a promessa, não o desenho de quem está ao lado dele.
   *
   * É a mesma família do ✕ inalcançável da ALE-178, e foi assim que ele
   * apareceu: a spec dos dois clientes começou a pendurar em "Abrir tabuleiro",
   * e o trace mostrou o clique sendo INTERCEPTADO pelo painel da aba — o botão
   * estava clipado fora do alcance.
   *
   * A janela é BAIXA de propósito: é o que faz o conteúdo transbordar sem
   * depender de quantas cenas a campanha juntou.
   */
  test('com o acervo cheio, o botão de abrir tabuleiro continua alcançável', async ({ page }) => {
    // Este é o teste mais LONGO da suíte: três idas e voltas de abrir e
    // encerrar tabuleiro, cada uma um ciclo completo pelo socket, mais a
    // limpeza. Sozinho leva ~11s; na suíte cheia ele estourava os 30 do teto e
    // falhava com "Test timeout exceeded" — foi a instabilidade que apareceu
    // duas vezes e que eu tinha anotado para perseguir. O teto triplicado é o
    // conserto honesto: o teste não é lento por defeito, ele é longo por
    // desenho, e encurtá-lo custaria a premissa do "acervo cheio".
    test.slow()
    await page.setViewportSize({ width: 1280, height: 400 })
    await page.goto('/campaigns/1/sessions/4')
    await expect(cenaViva(page)).toBeVisible()
    await encerraOTabuleiroSeHouver(page)

    // O acervo do teste: abrir e encerrar ARQUIVA a cena (fatia 5), então três
    // idas e voltas deixam três lugares na lista.
    const guardados = [1, 2, 3].map((n) => `Cena de teste ${SUFIXO}-${n}`)
    try {
      for (const lugar of guardados) {
        await page.getByRole('button', { name: 'Abrir tabuleiro' }).click()
        await page.locator('#board-place').fill(lugar)
        await page.getByRole('dialog').getByRole('button', { name: 'Abrir' }).click()
        await expect(page.getByRole('grid', { name: new RegExp(lugar) })).toBeVisible()
        await encerraOTabuleiroSeHouver(page)
      }

      // A prova: com a lista na tela, o botão continua clicável — e o diálogo
      // abre. Um `toBeVisible` não bastaria: o botão ESTAVA visível para o
      // navegador, e o que falhava era o clique chegar nele.
      await page.getByRole('button', { name: 'Abrir tabuleiro' }).click()
      await expect(page.getByRole('dialog')).toBeVisible()
      await page.getByRole('dialog').getByRole('button', { name: 'Cancelar' }).click()
    } finally {
      await apagaOsLugares(page, guardados)
    }
  })

  /**
   * Apaga as cenas que o teste guardou, mesmo que ele tenha falhado no meio.
   *
   * Existe porque os lugares vivem atrás do SOCKET, e a varredura do `setup`
   * — que é HTTP — não os alcança: uma falha no meio deixava as cenas no banco
   * para sempre, e a lista só cresce, o que deixa esta cena mais lenta a cada
   * falha num teste que já leva 8s. Achei 3 restos de execuções anteriores.
   *
   * É à prova de falha de propósito, e cada peça foi provada sabotando: uma
   * falha costuma deixar um DIÁLOGO por cima, e sem fechá-lo o clique de
   * apagar não chega no botão; e o erro da limpeza não pode escapar, senão ele
   * substitui na saída o erro DE VERDADE que fez o teste falhar.
   */
  async function apagaOsLugares(page: Page, lugares: string[]): Promise<void> {
    try {
      // Fechar o diálogo é a PRIMEIRA coisa, e ESPERAR — nunca clicar. Duas
      // versões minhas morreram aqui. A primeira não fechava nada: com o
      // diálogo aberto o Kobalte marca os irmãos `aria-hidden` e `getByRole`
      // devolve ZERO, então a limpeza rodava, não achava botão e ia embora
      // calada (a armadilha está no guia do front). A segunda clicava em
      // "Cancelar" e travava os 30 segundos do teste, porque no caminho FELIZ
      // o diálogo já está fechando quando a limpeza começa — clicar num alvo
      // em movimento faz o Playwright esperar estabilidade que nunca vem.
      await page.keyboard.press('Escape')
      await page
        .getByRole('dialog')
        .waitFor({ state: 'hidden', timeout: 5000 })
        .catch(() => {})

      for (const lugar of lugares) {
        const apagar = page.getByRole('button', { name: `Apagar ${lugar}` })
        if (!(await apagar.count())) continue
        await apagar.click({ timeout: 5000 })
        await page.getByRole('dialog').getByRole('button', { name: 'Apagar' }).click()
        // Esperar o diálogo SUMIR antes da próxima volta, pelo mesmo motivo:
        // enquanto ele fecha, os irmãos seguem `aria-hidden` e a volta seguinte
        // não acha o botão dela — a limpeza apagava UMA e desistia calada.
        await expect(page.getByRole('dialog')).toBeHidden({ timeout: 5000 })
        await expect(apagar).toHaveCount(0, { timeout: 5000 })
      }
    } catch (erro) {
      console.warn(`[limpeza] não consegui apagar todas as cenas: ${erro}`)
    }
  }

  /**
   * Encerra o tabuleiro que estiver aberto. Chamado por quem PRECISA da cena
   * vazia e por quem abre uma cena, no fim — a seed é compartilhada e a ordem
   * dos testes não é contrato.
   */
  async function encerraOTabuleiroSeHouver(page: Page): Promise<void> {
    const encerrar = page.getByRole('button', { name: 'Encerrar o tabuleiro' })
    if (!(await encerrar.isVisible().catch(() => false))) return
    await encerrar.click()
    await page.getByRole('dialog').getByRole('button', { name: 'Encerrar' }).click()
    await expect(page.getByText('Nenhum tabuleiro aberto')).toBeVisible()
  }

  /** Quem entrou na fila depois do instantâneo — o que este teste tem de limpar. */
  /**
   * Quem entrou na fila desde o instantâneo — lido da GAVETA, que é a lista
   * inteira.
   *
   * Era lido do trilho até a ALE-211, e ali passou a ser uma JANELA: com mais
   * combatentes do que cabem, a limpeza enxergava cinco de nove e deixava
   * quatro na seed compartilhada. As duas pontas (o `antes` e o `agora`) têm de
   * sair da MESMA fonte, senão o diff acusa como novo quem só saiu da janela.
   */
  async function novosDesde(gaveta: Locator, antes: string[]): Promise<string[]> {
    const agora = await labelsNaGaveta(gaveta)
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
  /**
   * Arrastar move a VISTA e a roda dá zoom ANCORADO no ponteiro (ALE-140).
   *
   * Por que e2e, e aqui não há dúvida: o gesto é `setPointerCapture` mais
   * coordenada real mais layout real, e em jsdom não existe nenhum dos três —
   * tudo mede zero e a asserção passaria verde sobre um tabuleiro parado.
   *
   * As duas asserções que importam são de RELAÇÃO, não de valor: a janela ANDOU
   * para o lado certo, e o quadrado que estava sob o ponteiro CONTINUOU sob o
   * ponteiro depois do zoom. E, no meio, a que pega o defeito clássico do
   * arraste: com uma peça selecionada, puxar o mapa não pode POUSAR a peça.
   */
  /**
   * Os combatentes que estes testes criam para ter o que medir — com nome ÚNICO
   * por execução, como o `session-realtime.spec.ts` já fazia.
   *
   * O nome fixo parecia bastar porque o teste se limpa no fim; só que a limpeza
   * não roda quando ele FALHA no meio, e aí a execução seguinte encontra DUAS
   * peças com o mesmo nome — foi assim que este teste passou a falhar por causa
   * do lixo que ele mesmo tinha deixado.
   */
  const SUFIXO = Date.now()
  const PECAS_DO_TELEFONE = [`Peça de telefone A ${SUFIXO}`, `Peça de telefone B ${SUFIXO}`]
  const ALVO_DO_ARRASTE = `Alvo do arraste ${SUFIXO}`

  test('o tabuleiro anda com o arraste, e o zoom fica ancorado no ponteiro', async ({ page }) => {
    await page.setViewportSize({ width: 1920, height: 1080 })
    await page.goto('/campaigns/1/sessions/4')
    await expect(cenaViva(page)).toBeVisible()

    // Setup que se limpa sozinho: a seed é compartilhada e um tabuleiro
    // esquecido aberto por outra execução faria este teste procurar um botão
    // que não existe.
    await encerraOTabuleiroSeHouver(page)
    await page.getByRole('button', { name: 'Abrir tabuleiro' }).click()
    await page.getByLabel('Lugar').fill('Arena do arraste')
    await page.getByRole('button', { name: 'Abrir', exact: true }).click()
    // A peça deste teste é CRIADA por ele. Prender um nome da seed (era o
    // "Ogro") fazia o teste depender da iniciativa que o banco de
    // desenvolvimento acumulou: no CI, onde a seed nasce com a iniciativa
    // VAZIA, ele esperava 30s por uma peça que nunca existiu. Prender o número
    // já tinha sido corrigido antes; o nome tinha ficado.
    const fila = await abreAFila(page)
    await fila.getByRole('button', { name: 'Combatente' }).click()
    await page.getByLabel('Nome').fill(ALVO_DO_ARRASTE)
    await page.getByRole('button', { name: 'Adicionar', exact: true }).click()
    await expect(fila.getByRole('button', { name: `Remover ${ALVO_DO_ARRASTE}` })).toBeVisible()
    await fechaAFila(page)

    // "Todos" porque o alvo do arraste é um NPC, e o padrão do diálogo traz só
    // os jogadores (ALE-204).
    await page.getByRole('button', { name: /Trazer a iniciativa/ }).click()
    await page.getByRole('dialog').getByRole('button', { name: 'Todos' }).click()
    await page.getByRole('dialog').getByRole('button', { name: /^Trazer \d/ }).click()
    // Quantas peças não importa aqui, e prender o número tornaria este teste
    // refém do tamanho da iniciativa da seed — que muda quando outro teste
    // deixa resto para trás. O que importa é que HÁ peça para medir.
    await expect(page.getByText(/[1-9]\d* peças/)).toBeVisible()

    const plano = page.getByRole('grid', { name: /^Tabuleiro:/ })

    // Com uma peça SELECIONADA a superfície inteira vira casa clicável, e é
    // nesse estado que o arraste tem de continuar sendo arraste — e o TOQUE tem
    // de continuar sendo toque.
    const peca = plano.getByRole('button', { name: new RegExp(`^${ALVO_DO_ARRASTE}, coluna`) })
    await peca.click()
    const ondeEstava = await peca.getAttribute('aria-label')

    // A caixa é medida DEPOIS de selecionar: a barra de ações da peça nasce
    // embaixo e ENCOLHE o tabuleiro. Medindo antes, o arraste deste teste caía
    // sobre a barra e a janela não andava — o teste acusava a tela, e a tela
    // estava certa.
    await expect(page.getByRole('button', { name: `Esconder ${ALVO_DO_ARRASTE}` })).toBeVisible()
    const caixa = await plano.boundingBox()
    if (!caixa) throw new Error('o tabuleiro não tem caixa — a cena não montou')

    const colunaDe = async () => {
      const rotulo = (await plano.getAttribute('aria-label')) ?? ''
      return Number(rotulo.match(/coluna (-?\d+)/)?.[1])
    }
    const colunaAntes = await colunaDe()

    // Puxa o mapa para a DIREITA, longe da peça para o gesto não começar nela.
    const y = caixa.y + caixa.height - 30
    await page.mouse.move(caixa.x + 100, y)
    await page.mouse.down()
    await page.mouse.move(caixa.x + 300, y, { steps: 12 })
    await page.mouse.up()

    // Puxar para a direita mostra o que está à ESQUERDA: a coluna diminui. São
    // 200px sobre quadrados de 44px, então são uns 4 quadrados — a asserção é
    // de direção e ordem de grandeza, não do número exato, que depende do zoom.
    const colunaDepois = await colunaDe()
    expect(colunaDepois, 'a janela não andou com o arraste').toBeLessThan(colunaAntes - 2)

    /*
     * E o TOQUE continua pousando a peça. Esta é a asserção que pegou a pior
     * regressão desta issue: capturar o ponteiro logo no `pointerdown` faz o
     * browser reapontar o `click` para a superfície, e aí TODO clique numa casa
     * morre — o mestre perde a única forma de posicionar. A captura só pode
     * acontecer depois de o gesto virar arraste.
     *
     * O caminho oposto ("o arraste não pousa a peça") não está afirmado aqui de
     * propósito: eu escrevi essa asserção, não consegui vê-la VERMELHA nem
     * removendo a captura, e asserção que não falha não protege nada.
     */
    const casa = plano.getByRole('button', { name: /^Coluna/ }).first()
    const quadrado = await casa.boundingBox()
    const naTela = await peca.boundingBox()
    if (!quadrado || !naTela) throw new Error('sem casa ou sem peça para medir o toque')
    const vizinha = {
      x: Math.min(naTela.x + 2.5 * quadrado.width, caixa.x + caixa.width - quadrado.width),
      y: naTela.y + naTela.height / 2,
    }

    await page.mouse.move(vizinha.x, vizinha.y)
    await page.mouse.down()
    await page.mouse.move(vizinha.x + 10, vizinha.y, { steps: 4 })
    await page.mouse.up()

    // Sem clicar na peça de novo: ela CONTINUA selecionada (selecionar de novo
    // desselecionaria, que é como se larga a peça sem posicioná-la).
    await page.mouse.click(vizinha.x, vizinha.y)
    await expect(
      peca,
      'o toque simples deixou de pousar a peça — o mestre perdeu como posicionar',
    ).not.toHaveAttribute('aria-label', ondeEstava ?? '')

    // O zoom ancorado: o ponteiro no centro da peça, e a peça continua ali.
    const antesDoZoom = await peca.boundingBox()
    if (!antesDoZoom) throw new Error('a peça saiu da tela antes do zoom')
    const centro = {
      x: antesDoZoom.x + antesDoZoom.width / 2,
      y: antesDoZoom.y + antesDoZoom.height / 2,
    }
    await page.mouse.move(centro.x, centro.y)
    await page.mouse.wheel(0, -120)

    const depoisDoZoom = await peca.boundingBox()
    if (!depoisDoZoom) throw new Error('o zoom jogou a peça para fora da tela')
    expect(depoisDoZoom.width, 'a roda não deu zoom').toBeGreaterThan(antesDoZoom.width)
    // 4px de folga: o quadrado cresce em número inteiro de pixels, e a peça é
    // centrada dentro dele.
    expect(Math.abs(depoisDoZoom.x + depoisDoZoom.width / 2 - centro.x)).toBeLessThan(4)
    expect(Math.abs(depoisDoZoom.y + depoisDoZoom.height / 2 - centro.y)).toBeLessThan(4)

    // Limpa o que criou, e nesta ordem: a cena primeiro (o teste seguinte pode
    // ser o que EXIGE a mesa vazia), o combatente depois. A seed é
    // compartilhada, e um esquecido engorda a lista de todo mundo.
    await page.getByRole('button', { name: 'Encerrar o tabuleiro' }).click()
    await page.getByRole('dialog').getByRole('button', { name: 'Encerrar' }).click()
    await expect(page.getByText('Nenhum tabuleiro aberto')).toBeVisible()
    const paraLimpar = await abreAFila(page)
    await paraLimpar.getByRole('button', { name: `Remover ${ALVO_DO_ARRASTE}` }).click()
  })

  test('rolar a iniciativa não leva embora as ações', async ({ page }) => {
    const nomes = [1, 2, 3, 4, 5].map((n) => `Fileira de teste ${Date.now()}-${n}`)
    await page.setViewportSize({ width: 1280, height: 420 })
    await page.goto('/campaigns/1/sessions/4')
    await expect(cenaViva(page)).toBeVisible()

    const fila = await abreAFila(page)
    await fila.getByRole('button', { name: 'Combatente' }).click()
    for (const nome of nomes) {
      await page.getByLabel('Nome').fill(nome)
      await page.getByRole('button', { name: 'Adicionar', exact: true }).click()
      await expect(fila.getByRole('button', { name: `Remover ${nome}` })).toBeVisible()
    }

    const acao = fila.getByRole('button', { name: 'Adicionar grupo' })
    await expect(acao).toBeInViewport()

    // Rola a LISTA até o fim, não a página.
    const rolou = await page.evaluate(() => {
      // "Mudar a iniciativa de X" desde a ALE-134: o rótulo passou a dizer o
      // VERBO, porque o número parecia chip de leitura e ninguém descobria que
      // ele abre o diálogo.
      const linha = document.querySelector('button[aria-label^="Mudar a iniciativa de"]')
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
    await expect(fila.getByRole('heading', { name: 'Iniciativa' })).toBeInViewport()

    // O formulário fecha ANTES da limpeza, e isso não afrouxa nada: tudo o que
    // este teste promete já foi afirmado acima, com ele aberto. Numa janela de
    // 420px o formulário custa ~118px e deixa a LISTA com 16 — clicar no
    // "Remover" de uma linha dentro de uma janela de 16px é o clique que o
    // rodapé do avanço intercepta, e limpeza que falha deixa cinco combatentes
    // na seed compartilhada para todo mundo depois.
    await fila.getByRole('button', { name: 'Fechar', exact: true }).click()

    for (const nome of nomes) {
      await fila.getByRole('button', { name: `Remover ${nome}` }).click()
    }
    await expect(fila.getByRole('button', { name: `Remover ${nomes[0]}` })).toBeHidden()
  })

  /**
   * A armadilha mais reincidente do repositório (ALE-95/96/121/122, quatro vezes
   * só nesta issue): ler `.data` de uma query PENDENTE suspende, e o `Suspense`
   * que o router põe em todo route match DESANEXA a cena inteira — a tela pisca
   * ou fica em branco no meio do combate. `settledQuery` existe para isso e é
   * lido em oito módulos; a prova de que funciona vinha de contar nós à mão
   * depois de cada regressão, nunca de um teste.
   *
   * Aqui um MutationObserver testemunha o que o olho vê: a CENA não pode ser
   * removida do DOM enquanto o mestre percorre as consultas.
   *
   * O que se vigia mudou de dono com a ALE-198 e ficou mais forte: era a lista
   * de combate, que hoje é overlay e some por desenho a cada fechar; agora é o
   * TABULEIRO, a superfície permanente. Ele não pode piscar por nada — nem por
   * abrir uma consulta, nem por um `Suspense` desanexando o route match.
   *
   * Por que e2e: sem router não há Suspense, e em jsdom a leitura pendente
   * devolve `undefined` — o mesmo teste passaria verde sobre a tela que pisca.
   */
  test('percorrer as consultas não desanexa a cena', async ({ page }) => {
    await page.goto('/campaigns/1/sessions/4')
    await expect(cenaViva(page)).toBeVisible()

    await page.evaluate(() => {
      const janela = window as unknown as { __desanexos: string[] }
      janela.__desanexos = []
      const ehACena = (node: Node) =>
        node instanceof HTMLElement &&
        (node.classList.contains('scene-grimorio') || !!node.querySelector('.scene-grimorio'))
      new MutationObserver((records) => {
        for (const record of records) {
          for (const node of record.removedNodes) {
            // Guarda QUEM saiu, não só quantos: "1 desanexo" não diz o que
            // arrancar da cena, e o relatório do CI é tudo o que se tem.
            if (ehACena(node)) {
              const el = node as HTMLElement
              janela.__desanexos.push(`${el.tagName}.${el.className}`.slice(0, 120))
            }
          }
        }
      }).observe(document.body, { childList: true, subtree: true })
    })

    // O BESTIÁRIO fica de fora, e não por conveniência: montar a
    // `MonsterPickerList` desanexa a cena por um quadro, e isso é ANTERIOR a
    // esta issue — o mesmo acontece pelo "Adicionar criatura" do Montar
    // encontro, caminho que a ALE-198 não tocou. Medido: ~7ms depois do
    // clique, sem requisição na rede, com o catálogo já em cache. Está na
    // ALE-199; consertar lá é devolver "Bestiário" a esta lista.
    for (const consulta of ['Encontros', 'Catálogos', 'Notas'] as const) {
      await abreConsulta(page, consulta)
      await expect(
        trilhoDeConsultas(page).getByRole('button', { name: consulta, exact: true }),
      ).toHaveAttribute('aria-pressed', 'true')
      await abreConsulta(page, consulta)
    }

    const desanexos = await page.evaluate(
      () => (window as unknown as { __desanexos: string[] }).__desanexos,
    )
    expect(desanexos, 'a cena foi removida do DOM (suspend desanexou o route match)').toEqual([])
    await expect(page.locator('.scene-grimorio').first()).toBeVisible()
  })

  /**
   * O flash de dano toca SÓ na linha que sangrou (ALE-174).
   *
   * Numa lista de nove combatentes um número trocava sozinho pelo socket e
   * ninguém via quem tinha sangrado. O flash resolve isso, mas o jeito ERRADO
   * de fazê-lo é animação de entrada: o estado do socket chega como objeto novo
   * a cada sync e o `For` reconcilia por REFERÊNCIA — medido, carimbei os seis
   * botões desta lista e ZERO nós sobreviveram a um único sync. Uma animação de
   * mount replayaria em TODAS as linhas a cada ponto de vida que muda em
   * QUALQUER uma delas, que é o mecanismo da ALE-97.
   *
   * Por isso a asserção tem duas metades e a segunda é a que importa: uma
   * linha animou, e as OUTRAS não. Só a primeira passaria verde sobre o
   * conserto errado.
   *
   * Por que e2e: `getAnimations()` é a única testemunha, e ela não existe em
   * jsdom — lá `el.animate` nem é função, e o módulo devolve nada de
   * propósito para não derrubar a suíte.
   */
  test('ferir um combatente pisca a linha dele, e só a dele', async ({ page }) => {
    await page.setViewportSize({ width: 1440, height: 900 })
    await page.goto('/campaigns/1/sessions/4')
    await expect(cenaViva(page)).toBeVisible()

    // Os combatentes são DESTE teste e saem no fim: a seed é compartilhada, e
    // contar com o que outro teste deixou já derrubou esta suíte no CI. A lista
    // que anima é a da GAVETA (ALE-198), e ela fica aberta durante o teste —
    // é ela que se está observando.
    const marca = Date.now()
    const nomes = [`Ferido ${marca}`, `Intacto ${marca}`]
    const fila = await abreAFila(page)
    await fila.getByRole('button', { name: 'Combatente' }).click()
    for (const nome of nomes) {
      await page.getByLabel('Nome').fill(nome)
      await page.getByRole('spinbutton', { name: 'PV' }).fill('20')
      await page.getByRole('button', { name: 'Adicionar', exact: true }).click()
      await expect(fila.getByRole('button', { name: `Remover ${nome}` })).toBeVisible()
    }
    await fila.getByRole('button', { name: 'Fechar', exact: true }).click()

    try {
      const linhas = await page.evaluate(() =>
        [...document.querySelectorAll('[data-entry-id]')].map((n) => ({
          id: n.getAttribute('data-entry-id') ?? '',
          nome: n.textContent?.slice(0, 40) ?? '',
        })),
      )
      const ferido = linhas.find((l) => l.nome.includes(nomes[0] ?? ''))?.id
      expect(ferido, 'não achei a linha do combatente ferido').toBeTruthy()

      await page.getByRole('button', { name: `Ferir ${nomes[0]}` }).click()

      // Espera ATIVA em vez de amostrar na hora: o flash só começa depois que
      // o servidor devolve o novo PV, e uma amostra imediata pega a tela antes
      // de a animação existir. Guarda o instantâneo do momento em que ela
      // aparece, porque ela dura ~380ms e some sozinha.
      let amostra: string[] = []
      await expect
        .poll(async () => {
          amostra = await linhasAnimadas(page)
          return amostra.length
        }, { message: 'a linha que sangrou não piscou' })
        .toBeGreaterThan(0)

      expect(amostra, 'a linha que sangrou não piscou').toContain(ferido)
      expect(
        [...new Set(amostra)],
        'a animação vazou para outras linhas — está disparando por MONTAGEM, não por diferença de valor',
      ).toEqual([ferido])
    } finally {
      for (const nome of nomes) {
        await fila.getByRole('button', { name: `Remover ${nome}` }).click()
        await expect(fila.getByRole('button', { name: `Remover ${nome}` })).toBeHidden()
      }
    }
  })

  /**
   * Passar o turno faz a linha que ENTRA na vez pulsar (ALE-174, P2).
   *
   * O anel dourado apenas teleportava de uma linha para outra, e numa lista de
   * nove a mesa não via o holofote andar. O pulso é único e curto; o que ele
   * compra é a resposta a "de quem é a vez agora?" sem ninguém procurar.
   *
   * Vale a mesma regra do flash e por isso a asserção também é dupla: uma
   * linha pulsou, as outras não. Uma animação de entrada tocaria em todas,
   * porque a lista inteira remonta a cada sync.
   *
   * Por que e2e: `getAnimations()` é a única testemunha e não existe em jsdom.
   */
  test('passar o turno pulsa a linha que entra na vez, e só ela', async ({ page }) => {
    await page.setViewportSize({ width: 1440, height: 900 })
    await page.goto('/campaigns/1/sessions/4')
    await expect(cenaViva(page)).toBeVisible()

    const marca = Date.now()
    const nomes = [`Vez A ${marca}`, `Vez B ${marca}`]
    const fila = await abreAFila(page)
    await fila.getByRole('button', { name: 'Combatente' }).click()
    for (const [i, nome] of nomes.entries()) {
      await page.getByLabel('Nome').fill(nome)
      // Iniciativas altas e distintas: sem isso a ordem depende do que a seed
      // deixou, e o teste passaria a afirmar a sorte do empate.
      await page.getByRole('spinbutton', { name: 'Iniciativa' }).fill(String(99 - i))
      await page.getByRole('button', { name: 'Adicionar', exact: true }).click()
      await expect(fila.getByRole('button', { name: `Remover ${nome}` })).toBeVisible()
    }
    await fila.getByRole('button', { name: 'Fechar', exact: true }).click()

    try {
      // Duas passadas: a primeira só ancora o "turno anterior" que a fábrica
      // guarda; é a SEGUNDA que tem de animar.
      const avancar = page.getByRole('button', { name: /^(Começar|Próximo):/ }).first()
      await avancar.click()
      await expect(page.getByRole('button', { name: /^Próximo:/ }).first()).toBeVisible()
      await page.getByRole('button', { name: /^Próximo:/ }).first().click()

      // Espera ATIVA: o pulso só começa quando o novo turno volta do servidor.
      let amostra: string[] = []
      await expect
        .poll(async () => {
          amostra = await linhasAnimadas(page)
          return amostra.length
        }, { message: 'nenhuma linha pulsou quando a vez mudou' })
        .toBeGreaterThan(0)

      expect(
        [...new Set(amostra)].length,
        'o pulso vazou para outras linhas — está disparando por MONTAGEM',
      ).toBe(1)
    } finally {
      for (const nome of nomes) {
        await fila.getByRole('button', { name: `Remover ${nome}` }).click()
        await expect(fila.getByRole('button', { name: `Remover ${nome}` })).toBeHidden()
      }
    }
  })

  /**
   * O aviso é pintado na cor da MESA, não em branco (ALE-173).
   *
   * O `Toaster` é irmão do `Outlet`, então ele nasce fora de qualquer cena — e
   * enquanto a raiz do documento carregava a paleta clara do shadcn, ele
   * resolvia ALI: medido, `oklch(1 0 0)` de fundo com texto quase preto. Todo
   * "Sua vez!", toda iniciativa rolada e todo erro eram um cartão branco sobre
   * a mesa escura, e ninguém tinha reclamado porque o defeito parece decisão.
   *
   * Compara token com token lidos da cena, e não o oklch literal: prender a cor
   * seria prender uma decisão de paleta que pode mudar sem o aviso deixar de
   * pertencer à mesa.
   *
   * Por que e2e: o aviso só existe depois de disparado, e a cor dele depende de
   * qual escopo o nó resolve. Em jsdom nenhuma variável CSS resolve.
   */
  test('o aviso da mesa é pintado na cor da mesa', async ({ page }) => {
    await page.goto('/campaigns/1/sessions/4')
    await expect(page.getByRole('status', { name: 'Conectado' })).toBeVisible()

    await page.getByRole('button', { name: 'Recuperar · cena' }).click()
    await expect(page.getByText('Efeitos temporários de cena foram limpos.')).toBeVisible()

    // A testemunha é o token BRUTO da casa, não o `--popover`. Comparar com
    // `--popover` parece mais direto e não prova nada: desde que a raiz virou
    // grimório os dois são a MESMA variável, então apontar `--popover` para
    // branco moveria os dois juntos e o teste passaria verde sobre o defeito.
    // Descobri isso sabotando.
    const cores = await page.evaluate(() => {
      const aviso = document.querySelector('[data-sonner-toast]')
      const cena = document.querySelector('.scene-grimorio')
      if (!aviso || !cena) return null
      return {
        fundo: getComputedStyle(aviso).backgroundColor,
        painelDaCasa: getComputedStyle(cena).getPropertyValue('--grimorio-panel-raised').trim(),
      }
    })

    expect(cores, 'não achei o aviso na tela').not.toBeNull()
    expect(
      cores?.fundo,
      'o aviso não é da mesa — ele nasce fora da cena e voltou a resolver uma paleta clara',
    ).toBe(cores?.painelDaCasa)
  })

  test('Sair da sessão volta pra campanha', async ({ page }) => {
    await page.goto('/campaigns/1/sessions/4')
    await expect(cenaViva(page)).toBeVisible()

    // Um LINK, não um botão: sem `asChild` no Solid, um link com cara de botão
    // é um `<a>` vestindo as classes do botão (armadilha #6 do porte).
    await page.getByRole('link', { name: 'Sair da sessão' }).click()
    await expect(page).toHaveURL(/\/campaigns\/1$/)
    await expect(page.getByRole('heading', { name: CAMPAIGN, level: 1 })).toBeVisible()
  })
})
