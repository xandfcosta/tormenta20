import { expect, test } from '@playwright/test'
import { ensurePowersFixture } from './support/character'
import {
  expectColunasMonotonicas,
  expectEnchePai,
  expectNadaEscapa,
  expectSemFaixaMorta,
} from './support/geometry'

/**
 * As listas virtualizadas do MESTRE — bestiário e catálogos.
 *
 * `VirtualList` mede as linhas para saber quais existem, e em jsdom todo
 * elemento mede zero: a lista renderiza NENHUMA linha e um teste de unidade
 * passa verde sobre a tela vazia. Foi assim que a ALE-84 entrou em produção com
 * a suíte inteira verde. Só um browser prova que a linha pintou — e que o
 * filtro TROCA o conjunto pintado, que é o outro modo de falha da
 * virtualização: ficar com o que pintou primeiro.
 *
 * Só leitura: filtra e navega, nunca escreve.
 */
test.describe('Listas virtualizadas do mestre', () => {
  test('o bestiário da sessão pinta linhas e o filtro troca o conjunto', async ({ page }) => {
    await page.goto('/campaigns/1/sessions/4')
    // O bestiário é GAVETA desde a ALE-198: a cena do mestre não tem mais abas,
    // e quem chama as consultas é o trilho da direita.
    await page.getByRole('navigation', { name: 'Consultas do mestre' })
      .getByRole('button', { name: 'Bestiário' })
      .click()

    const busca = page.getByRole('searchbox', { name: 'Buscar criatura' })
    await expect(busca).toBeVisible()
    // A linha só existe se a lista mediu e pintou — e a busca aparece ANTES
    // disso. Contar no instante em que ela surge é uma corrida: o teste falhou
    // duas vezes na suíte cheia com zero linhas, e o artefato mostrou a cena
    // montada com a lista ainda vazia. Esperar a primeira linha não afrouxa a
    // asserção, porque o que se promete é justamente que ela pinte.
    const linhas = page.getByRole('button', { name: /ND / })
    await expect(linhas.first()).toBeVisible()
    const antes = await linhas.count()
    expect(antes).toBeGreaterThan(0)

    await busca.fill('ogro')
    // Mais de um ogro no bestiário (o comum e o ancião) — `first()` de propósito.
    await expect(page.getByRole('button', { name: /^Ogro/ }).first()).toBeVisible()
    expect(await linhas.count()).toBeLessThan(antes)
  })

  test('o catálogo da sessão pinta linhas e a busca as troca', async ({ page }) => {
    await page.goto('/campaigns/1/sessions/4')
    await page.getByRole('navigation', { name: 'Consultas do mestre' })
      .getByRole('button', { name: 'Catálogos' })
      .click()

    const busca = page.getByRole('searchbox', { name: 'Buscar nos catálogos' })
    await expect(busca).toBeVisible()

    await busca.fill('espada')
    await expect(page.getByText(/Espada/).first()).toBeVisible()

    // O outro modo de falha: a lista guardar o que pintou primeiro. Depois de
    // uma busca sem resultado, nada de "espada" pode sobreviver na tela.
    await busca.fill('zzzzzz')
    await expect(page.getByText(/Espada longa/)).toHaveCount(0)

    // E o terceiro, que só o browser vê: a lista TERMINAR fora do cartão.
    // O painel de aba é um bloco, então o `flex-1` do filho não limitava altura
    // nenhuma e a lista descia até a borda da janela, 12px além do cartão —
    // sem a página rolar, porque a cena inteira é `overflow-hidden`, e por isso
    // "a cena do mestre não rola" ficava verde por cima (ALE-149).
    await busca.fill('')
    await expect(page.getByText('Abalado')).toBeVisible()
    const vazamento = await page.evaluate(() => {
      const rolante = [...document.querySelectorAll('*')].find((el) => {
        const estilo = getComputedStyle(el)
        return (
          /(auto|scroll)/.test(estilo.overflowY) &&
          el.scrollHeight > el.clientHeight + 4 &&
          (el.textContent ?? '').includes('Abalado')
        )
      })
      if (!rolante) throw new Error('não achei a lista de condições rolando')
      const cartao = document.querySelector('[role="tablist"]')?.parentElement
      if (!cartao) throw new Error('não achei o cartão do workspace')
      return Math.round(rolante.getBoundingClientRect().bottom - cartao.getBoundingClientRect().bottom)
    })
    expect(vazamento, `a lista passou ${vazamento}px do fundo do cartão`).toBeLessThanOrEqual(0)
  })

  test('a ferramenta Bestiário pinta a lista e abre a criatura escolhida', async ({ page }) => {
    await page.goto('/gm/bestiario')

    const busca = page.getByRole('searchbox', { name: 'Buscar criatura' })
    await expect(busca).toBeVisible()
    await busca.fill('ogro')

    const linha = page.getByRole('button', { name: /^Ogro/ }).first()
    await expect(linha).toBeVisible()
    await linha.click()

    await expect(page.getByRole('region', { name: 'Criatura escolhida' })).toContainText('Ogro')
  })


  /**
   * A sexta e última lista virtualizada da auditoria: o pool de poderes gerais,
   * dentro da ficha. Ele só pinta com o card da classe ABERTO, e o card abre
   * sozinho quando há escolha pendente — por isso o teste CRIA um Guerreiro de
   * 6º nível sem poder escolhido (três vagas em aberto) em vez de usar um
   * personagem da seed, onde as vagas já estão gastas e a lista fica fechada.
   *
   * O herói é criado pela API na primeira rodada e REUSADO nas seguintes: o app
   * não apaga personagem, então criar um por rodada entulharia o elenco.
   */
  test('o pool de poderes gerais da ficha pinta e filtra', async ({ page, request }) => {
    const id = await ensurePowersFixture(request)
    await page.goto(`/characters/${id}`)
    await page.getByRole('tab', { name: /^Poderes/ }).click()
    // O pool mora dentro do card da classe, e o card mora dentro do grupo
    // "Classe" — os dois recolhidos, nada pinta e não há o que medir.
    await page.getByRole('button', { name: /^Classe/ }).first().click()

    // `textbox`, não `searchbox`: este campo não declara `type="search"` como os
    // irmãos dele (bestiário e loja) — o papel segue o elemento, não o rótulo.
    const busca = page.getByRole('textbox', { name: 'Buscar poder geral' })
    await expect(busca).toBeVisible()
    // A linha do pool é um `div` com caixa de marcar e o nome em texto — não um
    // botão por linha, como nas outras listas.
    await expect(page.getByText('Ataque Poderoso', { exact: true })).toBeVisible()

    await busca.fill('esquiva')
    await expect(page.getByText(/Esquiva/).first()).toBeVisible()
    // O modo de falha da virtualização: guardar o que pintou primeiro.
    await expect(page.getByText('Ataque Poderoso', { exact: true })).toHaveCount(0)
  })

  /**
   * Dentro da GAVETA a lista continua virtualizada (ALE-138).
   *
   * O terceiro modo de falha da virtualização, e o mais silencioso: a lista não
   * quebra, ela só deixa de virtualizar. O corpo do `SidePanel` era um BLOCO
   * com `overflow-y-auto`, então o `flex-1 min-h-0` da lista não tinha o que
   * limitar e o contêiner de rolagem dela crescia até a altura do próprio
   * conteúdo. Com `clientHeight === scrollHeight` o virtualizador conclui, com
   * razão, que TUDO está visível — e pinta tudo. Medido antes do conserto na
   * aba Poderes: 566 de 566 linhas no DOM, 3397 nós, contra 15 depois.
   *
   * Nada parecia quebrado porque a gaveta rolava: quem rolava era o corpo do
   * painel, e a lista de dentro nunca tinha o que rolar. Por isso o guarda
   * afirma o MECANISMO (a caixa de rolagem é menor que o conteúdo) junto com a
   * consequência (poucas linhas pintadas) — só a segunda passaria verde se
   * alguém trocasse a virtualização por uma fatia fixa.
   *
   * Por que e2e: em jsdom todo elemento mede zero, `clientHeight` e
   * `scrollHeight` dão zero os dois, e NENHUMA linha renderiza. O teste de
   * unidade passa verde tanto sobre a lista virtualizada quanto sobre as 566
   * linhas — ele não consegue distinguir os dois casos.
   */
  test('a gaveta de Catálogos virtualiza a aba Poderes em vez de pintar as 566', async ({
    page,
  }) => {
    await page.goto('/campaigns/1/sessions/4')
    await page.getByRole('navigation', { name: 'Consultas do mestre' })
      .getByRole('button', { name: 'Catálogos' })
      .click()

    await expect(page.getByRole('searchbox', { name: 'Buscar nos catálogos' })).toBeVisible()
    await page.getByRole('tab', { name: 'Poderes' }).click()

    const linhas = page.locator('[data-index]')
    await expect(linhas.first()).toBeVisible()

    const medida = await page.evaluate(() => {
      const primeira = document.querySelector('[data-index]')
      const rolante = primeira?.closest<HTMLElement>('.overflow-y-auto')
      if (!rolante) throw new Error('não achei o contêiner de rolagem da lista')
      const entradas = document.body.textContent?.match(/(\d+) entradas/)?.[1]
      return {
        entradas: Number(entradas ?? 0),
        pintadas: document.querySelectorAll('[data-index]').length,
        caixa: rolante.clientHeight,
        conteudo: rolante.scrollHeight,
      }
    })

    // Sem isto a asserção não prova nada: uma lista que COUBE pintaria tudo com
    // razão. A aba Poderes tem centenas de entradas justamente por isso.
    expect(medida.entradas, 'a aba Poderes veio vazia — não há o que virtualizar').toBeGreaterThan(
      200,
    )
    expect(
      medida.caixa,
      `a caixa de rolagem tem ${medida.caixa}px para ${medida.conteudo}px de conteúdo — ` +
        'nada limita a altura dela e o virtualizador acha que tudo está visível',
    ).toBeLessThan(medida.conteudo)
    expect(
      medida.pintadas,
      `${medida.pintadas} de ${medida.entradas} linhas no DOM — a lista parou de virtualizar`,
    ).toBeLessThan(medida.entradas / 4)
  })
})

/**
 * A lista do bestiário PREENCHE a coluna no tablet em pé (ALE-175).
 *
 * A lista tinha uma tampa de `45vh` cujo comentário dizia proteger "o resto da
 * ferramenta" de 80 linhas. Mas abaixo de `lg` o resto da ferramenta é o painel
 * de detalhe, que ali é `hidden`: a tampa protegia conteúdo que não existe
 * naquele formato. O preço eram 243px mortos em 768×1024 — um quarto da tela —
 * com a lista mostrando 459px de 5216 de conteúdo.
 *
 * O que se afirma é a FAIXA MORTA e não a altura da lista: altura é
 * consequência do formato, e prendê-la seria prender o número errado. O que a
 * cena promete é não deixar banda vazia embaixo do último elemento da coluna.
 *
 * Por que e2e: é caixa contra caixa em altura real. Em jsdom todo elemento mede
 * zero e a faixa morta dá zero em qualquer arranjo.
 */
test('no tablet em pé, a lista do bestiário não deixa faixa morta', async ({ page }) => {
  await page.setViewportSize({ width: 768, height: 1024 })
  await page.goto('/gm/bestiario')
  await expect(page.getByRole('button', { name: /ND / }).first()).toBeVisible()

  // Sem transbordo a asserção não prova nada: seria uma lista que coube.
  const transbordou = await page.evaluate(
    () =>
      [...document.querySelectorAll('*')].find(
        (n) => n.scrollHeight > n.clientHeight + 8 && n.clientHeight > 100,
      ) !== undefined,
  )
  expect(transbordou, 'a lista não transbordou — o teste não mediu nada').toBe(true)

  await expectSemFaixaMorta(page, '[aria-labelledby=mesa-bestiario]')
})

/**
 * Crescer a janela nunca tira uma coluna do bestiário (ALE-172).
 *
 * O gate das duas colunas olhava a JANELA (`lg:`), e a coluna de ferramentas
 * do `/gm` devolve largura à direita conforme a janela encolhe. O resultado
 * era invertido: numa janela de 1024 o palco recebia 800px e mostrava DUAS
 * colunas, e numa de 1000 recebia 968px e mostrava UMA. O mestre alargava a
 * janela e perdia o painel de detalhe.
 *
 * A varredura é de LARGURA com altura fixa de propósito. A decisão real ali é
 * "cabe painel lateral?", que tem duas dimensões: o mesmo contêiner de 812px
 * cabe num tablet deitado (768px de altura) e não cabe num celular deitado
 * (390px, onde a lista sobraria com 41px — menos que uma linha). Por isso o
 * conserto é `container-type: size` com as duas condições, e por isso comparar
 * caixas de alturas diferentes acusaria como defeito a exceção que É o
 * conserto.
 *
 * Por que e2e: media/container query só resolve em browser de verdade. Em
 * jsdom nenhuma consulta casa e a grade responde sempre a mesma coisa.
 */
test('alargar a janela nunca tira uma coluna do bestiário', async ({ page }) => {
  await page.goto('/gm/bestiario')
  await expect(page.getByRole('button', { name: /ND / }).first()).toBeVisible()

  await expectColunasMonotonicas(
    page,
    '[aria-labelledby=mesa-bestiario] div.grid',
    [1920, 1440, 1200, 1100, 1040, 1024, 1000, 950, 900, 860, 844, 830, 812, 800, 768, 390],
  )
})

/**
 * As colunas do catálogo seguem o PAINEL, e alargar nunca tira uma (ALE-170).
 *
 * Mesma classe de defeito que a ALE-172 consertou no bestiário, e por isso o
 * mesmo guarda: a ferramenta divide a tela com a trilha do `/gm`, então largura
 * de janela mente por centenas de pixels sobre quanto espaço o painel tem.
 *
 * A segunda asserção é a que só o browser faz. Numa lista VIRTUALIZADA "três
 * colunas" não é grade de CSS — é o agrupamento dos dados antes de entregá-los.
 * A grade pode declarar três colunas com um cartão só em cada fileira, e a tela
 * fica com dois terços de vazio à direita enquanto o CSS jura que está certo.
 * Contar os cartões DENTRO da fileira é o que separa as duas metades.
 */
test('alargar a janela nunca tira uma coluna do catálogo', async ({ page }) => {
  await page.goto('/gm/catalogos')
  await expect(page.locator('[data-index]').first()).toBeVisible()

  await expectColunasMonotonicas(
    page,
    '[data-index] div.grid',
    [1920, 1440, 1200, 1100, 1024, 1000, 900, 844, 768, 600, 390],
  )

  await page.setViewportSize({ width: 1920, height: 1080 })
  await expect(page.locator('[data-index]').first()).toBeVisible()
  const fileira = await page.evaluate(() => {
    const grade = document.querySelector('[data-index] div.grid')
    if (!grade) return null
    return {
      declaradas: getComputedStyle(grade).gridTemplateColumns.split(' ').filter(Boolean).length,
      cartoes: grade.children.length,
    }
  })
  expect(fileira, 'nenhuma fileira pintou em 1920').not.toBeNull()
  expect(fileira?.cartoes, 'a grade declara colunas que a fileira não preenche').toBe(
    fileira?.declaradas,
  )
})

/**
 * As abas do catálogo DIVIDEM a faixa que recebem (ALE-138).
 *
 * `Condições | Magias | Poderes | Itens` ficavam encolhidas à esquerda com a
 * faixa inteira sobrando à direita. O gatilho do kit já nasce `flex-1`, mas o
 * `TabsList` nasce `inline-flex w-fit`: sem largura, o `flex-1` não tem o que
 * dividir. O print do dono mostrava a barra de cima ocupando tudo e a de dentro
 * não — mesma tela, dois comportamentos.
 *
 * A segunda asserção é a armadilha da ALE-122: junto de `flex-1` o `min-w-0` é
 * obrigatório, porque um item flex não encolhe abaixo do conteúdo e o rótulo
 * mais longo empurra a última aba para FORA da faixa. Ela NÃO reproduz um
 * defeito de hoje — medido a 390px, os quatro rótulos curtos cabem mesmo sem
 * `min-w-0`. Ela protege o próximo: uma quinta aba, ou "Condições" virando
 * "Condições e estados", e a faixa estoura sem ninguém ver.
 *
 * As duas larguras não são simetria: a régua é o CONTÊINER, e a mesma janela dá
 * a largura inteira na Mesa e ~384px na gaveta da sessão (ALE-138, ALE-172).
 *
 * Por que e2e: é caixa contra caixa. Em jsdom todo elemento mede zero e
 * `expectEnchePai` passaria verde sobre qualquer arranjo.
 */
test('as abas do catálogo enchem a faixa, e a última não sai dela', async ({ page }) => {
  // Uma navegação só, redimensionando depois — o mesmo padrão dos guardas de
  // coluna aqui do lado. Recarregar por largura paga o portão dos catálogos
  // (18 buscas antes da primeira tela) a cada volta, e foi assim que a versão
  // anterior deste teste estourou o timeout sem que nada estivesse errado.
  await page.goto('/gm/catalogos')
  await expect(page.getByRole('tab', { name: 'Condições' })).toBeVisible()

  for (const largura of [1920, 1024, 768, 390]) {
    await page.setViewportSize({ width: largura, height: 900 })
    await expect(page.getByRole('tab', { name: 'Condições' })).toBeVisible()

    // O par medido é a FAIXA contra a ferramenta, e não a aba contra a faixa:
    // com `w-fit` a lista encolhe até as abas, então as abas enchem a lista
    // perfeitamente e a asserção passa VERDE sobre o defeito. O espaço morto
    // está entre a lista e o painel, e é ali que ele se mede.
    await expectEnchePai(page, '[aria-labelledby=mesa-catalogos]', '[role=tablist]')
    // A outra metade, a da ALE-122: `flex-1` não encolhe abaixo do conteúdo, e
    // sem `min-w-0` o rótulo mais longo empurra a última aba para fora.
    await expectNadaEscapa(page, '[role=tablist]')
  }
})
