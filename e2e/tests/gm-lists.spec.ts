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
  const transbordou = await page.evaluate(
    () =>
      [...document.querySelectorAll('*')].find(
        (n) => n.scrollHeight > n.clientHeight + 8 && n.clientHeight > 100,
      ) !== undefined,
  )
  expect(transbordou, 'a lista não transbordou — o teste não mediu nada').toBe(true)

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
  const colunas = await page.evaluate(() => {
    const grade = document.querySelector('.collection-in-columns')
    if (!grade) return null
    return getComputedStyle(grade).gridTemplateColumns.split(' ').filter(Boolean).length
  })
  expect(colunas, 'nenhuma grade pintou em 1920').not.toBeNull()
  expect(colunas, 'o teto de três colunas é medida de leitura (ALE-170)').toBeLessThanOrEqual(3)
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
  const trilho = 'nav[aria-label="Ferramentas do mestre"]'
  await expect(page.getByRole('link', { name: 'Condições' })).toBeVisible()

  // A CONTAGEM de paradas não mora aqui: ela é `TestTheRailHasOneStopPerCatalog`
  // no Go, por amostragem sobre `abasDoAcervo`. Escrita aqui como número, ela
  // ficava vermelha por parada nova sem proteger nada. Aqui fica só a geometria,
  // que é o que o navegador testemunha.
  let referencia = 0

  for (const largura of [1920, 1024, 768, 390]) {
    await page.setViewportSize({ width: largura, height: 900 })
    await expect(page.getByRole('link', { name: 'Condições' })).toBeVisible()

    // Nenhuma parada escapa do trilho.
    await expectNadaEscapa(page, trilho)

    // E TODAS continuam alcançáveis: uma parada que some da tela é uma
    // ferramenta que deixou de existir para quem está naquela largura.
    const alcancaveis = await page.locator(`${trilho} a`).count()
    if (largura === 1920) referencia = alcancaveis
    expect(alcancaveis, `a ${largura}px o trilho perdeu paradas`).toBe(referencia)
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
  const paradas = await page
    .getByRole('navigation', { name: 'Ferramentas do mestre' })
    .getByRole('link')
    .evaluateAll((links) => links.map((l) => (l as HTMLAnchorElement).href))
  expect(paradas.length, 'o trilho veio vazio: este caso não mediria nada').toBeGreaterThan(10)

  // A CAIXA da gaveta fechada é o resumo e mais nada: 44px de alvo de toque.
  const TETO_FECHADA = 44 + 12

  let comGaveta = 0
  let semGaveta = 0
  for (const parada of paradas) {
    // ── deitado: a gaveta existe e está recolhida ──────────────────────────
    await page.setViewportSize({ width: 844, height: 390 })
    await page.goto(parada)
    const gaveta = page.locator('.filters-in-drawer')
    if ((await gaveta.count()) === 0) {
      semGaveta++
      continue
    }
    comGaveta++

    const medida = await gaveta.first().evaluate((d: HTMLDetailsElement) => ({
      // A CAIXA, e nunca o filho: fechado, o miolo devolve retângulo mesmo.
      alta: Math.round(d.getBoundingClientRect().height),
      aberta: d.open,
      resumoAparece: d.querySelector('summary')!.checkVisibility(),
      filtroAparece: !!d.querySelector('button')?.checkVisibility(),
    }))
    expect(medida.aberta, `a gaveta de ${parada} nasce aberta no deitado`).toBe(false)
    expect(medida.resumoAparece, `sem o resumo, os filtros de ${parada} ficam inalcançáveis`).toBe(true)
    expect(medida.filtroAparece, `os filtros de ${parada} não recolheram deitado`).toBe(false)
    expect(
      medida.alta,
      `a gaveta de ${parada} recolhida ocupa ${medida.alta}px`,
    ).toBeLessThanOrEqual(TETO_FECHADA)

    // ── em pé e no laptop: a gaveta não existe como gaveta ─────────────────
    for (const forma of [
      { nome: 'no formato em pé', width: 390, height: 844 },
      { nome: 'no laptop', width: 1280, height: 720 },
    ]) {
      await page.setViewportSize({ width: forma.width, height: forma.height })
      const larga = await gaveta.first().evaluate((d: HTMLDetailsElement) => ({
        resumoAparece: d.querySelector('summary')!.checkVisibility(),
        filtroAparece: !!d.querySelector('button')?.checkVisibility(),
      }))
      expect(
        larga.filtroAparece,
        `em ${forma.nome} os filtros de ${parada} continuam escondidos: a regra de abertura caiu`,
      ).toBe(true)
      expect(
        larga.resumoAparece,
        `em ${forma.nome} a gaveta de ${parada} ainda mostra o resumo, e ali ela não deveria existir`,
      ).toBe(false)
    }
  }

  // O DENOMINADOR: sem ele, um seletor que parou de casar dá o mesmo verde que
  // uma tela em ordem. Medido: sete paradas filtram, seis não.
  expect(comGaveta, 'nenhuma parada tinha gaveta: o seletor `.filters-in-drawer` parou de casar').toBeGreaterThanOrEqual(7)
  expect(comGaveta + semGaveta).toBe(paradas.length)
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
  const lista = page.locator('#bestiary ul')
  await expect(lista, 'sem a lista não há medição').toBeVisible()

  const medida = await lista.evaluate((ul) => ({
    caixa: Math.round(ul.getBoundingClientRect().height),
    linha: Math.round(ul.firstElementChild!.getBoundingClientRect().height),
  }))
  expect(medida.linha, 'uma linha de zero passaria em qualquer teto').toBeGreaterThan(20)
  expect(
    medida.caixa,
    `a lista recebe ${medida.caixa}px e uma criatura mede ${medida.linha}px`,
  ).toBeGreaterThanOrEqual(medida.linha)
})
