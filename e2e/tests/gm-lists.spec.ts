import { expect, test } from '@playwright/test'
import {
  expectColunasMonotonicas,
  expectNadaEscapa,
  expectSemFaixaMorta,
} from './support/geometry'

/**
 * As listas do MESTRE — bestiário e catálogos, nas cenas.
 *
 * Por que e2e: LEIAUTE REAL. Coluna que some ao alargar a janela e faixa morta
 * no tablet não existem em HTML nenhum.
 *
 * Só leitura: filtra e navega, nunca escreve.
 */
test.describe('As listas do mestre', () => {

  // NÃO existe aqui um caso "a ferramenta Bestiário pinta a lista e abre a
  // criatura escolhida": sem virtualização ele mede HTML do servidor, que é a
  // camada de baixo. Quem cobre: `TestTheBestiaryOpensWithTheWholeBook` e
  // `TestTheSearchIsAnAddress`.

test('no tablet em pé, a lista do bestiário não deixa faixa morta', async ({ page }) => {
  await page.setViewportSize({ width: 768, height: 1024 })
  await page.goto('/mestre/bestiario')
  await expect(page.getByRole('link', { name: /ND / }).first()).toBeVisible()

  // Sem transbordo a asserção não prova nada: seria uma lista que coube.
  const overflowed = await page.evaluate(
    () =>
      [...document.querySelectorAll('*')].find(
        (n) => n.scrollHeight > n.clientHeight + 8 && n.clientHeight > 100,
      ) !== undefined,
  )
  expect(overflowed, 'a lista não transbordou — o teste não mediu nada').toBe(true)

  // Tolerância de 12 e não 8, e a razão é do INSTRUMENTO: a primitiva mede o
  // último elemento com TEXTO, e cada linha tem `p-2`, então o texto da última
  // fica ~11px acima do fim do contêiner. Não cega o guarda — a banda morta que
  // ele caça mede centenas de pixels, e recolocar uma tampa de altura na lista
  // o deixa VERMELHO com a tolerância em 12.
  await expectSemFaixaMorta(page, '[aria-labelledby=table-bestiary-panel]', 12)
})

/**
 * A varredura é de LARGURA com altura fixa de propósito. A decisão real é "cabe
 * painel lateral?", e ela tem duas dimensões: o mesmo contêiner de 812px cabe
 * num tablet deitado (768px de altura) e não cabe num celular deitado (390px).
 * Comparar caixas de alturas diferentes acusaria como defeito a exceção que É o
 * conserto.
 *
 * Por que e2e: media/container query só resolve em browser de verdade. Em
 * jsdom nenhuma consulta casa e a grade responde sempre a mesma coisa.
 */
test('alargar a janela nunca tira uma coluna do bestiário', async ({ page }) => {
  await page.goto('/mestre/bestiario')
  await expect(page.getByRole('link', { name: /ND / }).first()).toBeVisible()

  await expectColunasMonotonicas(
    page,
    '[aria-labelledby=table-bestiary-panel] div.grid',
    [1920, 1440, 1200, 1100, 1040, 1024, 1000, 950, 900, 860, 844, 830, 812, 800, 768, 390],
  )
})

/**
 * A ferramenta divide a tela com a trilha do mestre, então largura de JANELA
 * mente por centenas de pixels sobre quanto espaço o painel tem — por isso o
 * mesmo guarda do bestiário.
 */
test('alargar a janela nunca tira uma coluna do catálogo', async ({ page }) => {
  await page.goto('/mestre/condicoes')
  await expect(page.locator('.collection-in-columns').first()).toBeVisible()

  await expectColunasMonotonicas(
    page,
    '.collection-in-columns',
    [1920, 1440, 1200, 1100, 1024, 1000, 900, 844, 768, 600, 390],
  )

  // A grade é nativa e preenche sozinha, então fileira com vazio à direita não
  // pode existir. O risco é o oposto, e é o que se afirma: a grade declarando
  // MAIS colunas do que cabem para ler.
  await page.setViewportSize({ width: 1920, height: 1080 })
  const cols = await page.evaluate(() => {
    const grade = document.querySelector('.collection-in-columns')
    if (!grade) return null
    return getComputedStyle(grade).gridTemplateColumns.split(' ').filter(Boolean).length
  })
  expect(cols, 'nenhuma grade pintou em 1920').not.toBeNull()
  expect(cols, 'o teto de três colunas é medida de leitura (ALE-170)').toBeLessThanOrEqual(3)
})

/**
 * `expectNadaEscapa` NÃO reproduz um defeito de hoje: junto de `flex-1` o
 * `min-w-0` é obrigatório, porque um item flex não encolhe abaixo do conteúdo e
 * o rótulo mais longo empurra a última parada para FORA do trilho. Ela protege
 * o próximo rótulo mais longo.
 *
 * Por que e2e: é caixa contra caixa. Em jsdom todo elemento mede zero e
 * `expectNadaEscapa` passaria verde sobre qualquer arranjo.
 */
test('o trilho do mestre segura todas as paradas em qualquer largura', async ({ page }) => {
  // Uma navegação só, redimensionando depois: recarregar por largura paga o
  // portão dos catálogos (18 buscas antes da primeira tela) a cada volta, e foi
  // assim que a versão anterior deste teste estourou o timeout sem que nada
  // estivesse errado.
  await page.goto('/mestre/condicoes')
  const rail = 'nav[aria-label="Ferramentas do mestre"]'
  await expect(page.getByRole('link', { name: 'Condições' })).toBeVisible()

  // A CONTAGEM de paradas não mora aqui: ela é `TestTheRailHasOneStopPerCatalog`
  // no Go, por amostragem sobre `abasDoAcervo`. Escrita aqui como número, ela
  // ficava vermelha por parada nova sem proteger nada. Aqui fica só a geometria,
  // que é o que o navegador testemunha.
  let reference = 0

  for (const width of [1920, 1024, 768, 390]) {
    await page.setViewportSize({ width: width, height: 900 })
    await expect(page.getByRole('link', { name: 'Condições' })).toBeVisible()

    // Nenhuma parada escapa do trilho.
    await expectNadaEscapa(page, rail)

    // E TODAS continuam alcançáveis: uma parada que some da tela é uma
    // ferramenta que deixou de existir para quem está naquela largura.
    const reachable = await page.locator(`${rail} a`).count()
    if (width === 1920) reference = reachable
    expect(reachable, `a ${width}px o trilho perdeu paradas`).toBe(reference)
  }
})
})

/**
 * A GAVETA DE FILTROS do celular deitado.
 *
 * O CASO MEDE OS DOIS LADOS, e o segundo não é redundância — é o CONTROLE. Uma
 * gaveta que se fecha em toda forma passaria com folga numa asserção só de
 * altura, e o custo seria os filtros sumirem da tela larga, que é onde eles mais
 * servem. A folha da casa escreve DUAS regras de abertura (`display: revert` e
 * `::details-content`) porque os motores escondem o miolo de um `<details>`
 * fechado de dois jeitos diferentes e um deles pode não existir.
 *
 * Só um browser testemunha: a chave é `(max-width: 1023px) and (orientation:
 * landscape)`, e orientação não existe em jsdom. Pior: o miolo de um `<details>`
 * FECHADO ainda devolve `boundingClientRect` — o navegador o esconde por
 * `content-visibility` —, então a medição é da CAIXA e a visibilidade vem de
 * `checkVisibility`. Medir o filho foi o que mentiu na primeira sonda.
 *
 * Ele CAMINHA pelo trilho em vez de trazer uma lista de paradas: a parada que
 * entrar amanhã já nasce medida.
 */
test('deitado os filtros viram gaveta, e em toda outra forma eles ficam abertos', async ({ page }) => {
  await page.goto('/mestre/bestiario')
  const stops = await page
    .getByRole('navigation', { name: 'Ferramentas do mestre' })
    .getByRole('link')
    .evaluateAll((links) => links.map((l) => (l as HTMLAnchorElement).href))
  expect(stops.length, 'o trilho veio vazio: este caso não mediria nada').toBeGreaterThan(10)

  // A CAIXA da gaveta fechada é o resumo e mais nada: 44px de alvo de toque.
  const CLOSED_CEILING = 44 + 12

  let withDrawer = 0
  let noDrawer = 0
  for (const waypoint of stops) {
    // ── deitado: a gaveta existe e está recolhida ──────────────────────────
    await page.setViewportSize({ width: 844, height: 390 })
    await page.goto(waypoint)
    const drawer = page.locator('.filters-in-drawer')
    if ((await drawer.count()) === 0) {
      noDrawer++
      continue
    }
    withDrawer++

    const measurement = await drawer.first().evaluate((d: HTMLDetailsElement) => ({
      // A CAIXA, e nunca o filho: fechado, o miolo devolve retângulo mesmo.
      alta: Math.round(d.getBoundingClientRect().height),
      aberta: d.open,
      resumoAparece: d.querySelector('summary')!.checkVisibility(),
      filtroAparece: !!d.querySelector('button')?.checkVisibility(),
    }))
    expect(measurement.aberta, `a gaveta de ${waypoint} nasce aberta no deitado`).toBe(false)
    expect(measurement.resumoAparece, `sem o resumo, os filtros de ${waypoint} ficam inalcançáveis`).toBe(true)
    expect(measurement.filtroAparece, `os filtros de ${waypoint} não recolheram deitado`).toBe(false)
    expect(
      measurement.alta,
      `a gaveta de ${waypoint} recolhida ocupa ${measurement.alta}px`,
    ).toBeLessThanOrEqual(CLOSED_CEILING)

    // ── em pé e no laptop: a gaveta não existe como gaveta ─────────────────
    for (const form of [
      { nome: 'no formato em pé', width: 390, height: 844 },
      { nome: 'no laptop', width: 1280, height: 720 },
    ]) {
      await page.setViewportSize({ width: form.width, height: form.height })
      const wide = await drawer.first().evaluate((d: HTMLDetailsElement) => ({
        resumoAparece: d.querySelector('summary')!.checkVisibility(),
        filtroAparece: !!d.querySelector('button')?.checkVisibility(),
      }))
      expect(
        wide.filtroAparece,
        `em ${form.nome} os filtros de ${waypoint} continuam escondidos: a regra de abertura caiu`,
      ).toBe(true)
      expect(
        wide.resumoAparece,
        `em ${form.nome} a gaveta de ${waypoint} ainda mostra o resumo, e ali ela não deveria existir`,
      ).toBe(false)
    }
  }

  // O DENOMINADOR: sem ele, um seletor que parou de casar dá o mesmo verde que
  // uma tela em ordem. Medido: sete paradas filtram, seis não.
  expect(withDrawer, 'nenhuma parada tinha gaveta: o seletor `.filters-in-drawer` parou de casar').toBeGreaterThanOrEqual(7)
  expect(withDrawer + noDrawer).toBe(stops.length)
})

/**
 * O PISO DO ÚTIL, deitado: uma criatura INTEIRA. O caso prende o RESULTADO e não
 * o mecanismo — a gaveta é uma forma de chegar lá, e amanhã pode ser outra.
 *
 * Nos catálogos não há caso equivalente de propósito: um cartão de magia mede
 * 205px dos 390 da tela, então nem com cromo ZERO o piso seria alcançável ali.
 * O que não cabe é o CARTÃO, e isso é outra decisão.
 */
test('deitado, a lista do bestiário mostra uma criatura inteira', async ({ page }) => {
  await page.setViewportSize({ width: 844, height: 390 })
  await page.goto('/mestre/bestiario')
  const list = page.locator('#bestiary ul')
  await expect(list, 'sem a lista não há medição').toBeVisible()

  const measure = await list.evaluate((ul) => ({
    caixa: Math.round(ul.getBoundingClientRect().height),
    linha: Math.round(ul.firstElementChild!.getBoundingClientRect().height),
  }))
  expect(measure.linha, 'uma linha de zero passaria em qualquer teto').toBeGreaterThan(20)
  expect(
    measure.caixa,
    `a lista recebe ${measure.caixa}px e uma criatura mede ${measure.linha}px`,
  ).toBeGreaterThanOrEqual(measure.linha)
})
