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
  await expectSemFaixaMorta(page, '#bestiary', 12)
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
    '#bestiary',
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
  await expect(page.locator('.ds-catalog-grid').first()).toBeVisible()

  await expectColunasMonotonicas(
    page,
    '.ds-catalog-grid',
    [1920, 1440, 1200, 1100, 1024, 1000, 900, 844, 768, 600, 390],
  )

  // A grade é nativa e preenche sozinha, então fileira com vazio à direita não
  // pode existir. O risco é o oposto, e é o que se afirma: a grade declarando
  // MAIS colunas do que cabem para ler.
  await page.setViewportSize({ width: 1920, height: 1080 })
  const cols = await page.evaluate(() => {
    const grade = document.querySelector('.ds-catalog-grid')
    if (!grade) return null
    return getComputedStyle(grade).gridTemplateColumns.split(' ').filter(Boolean).length
  })
  expect(cols, 'nenhuma grade pintou em 1920').not.toBeNull()
  expect(cols, 'o teto de três colunas é medida de leitura (ALE-170)').toBeLessThanOrEqual(3)
})

/**
 * O menu lateral aparece a partir de 1024px e segura todas as telas, por mais longo
 * que seja o rótulo. Por que e2e: é caixa contra caixa. Abaixo dessa largura quem leva
 * às telas é a barra de baixo, e os destinos dela estão em
 * `TestTheMasterFrameReachesEveryTool`, que é a camada mais barata para isso.
 */
test('o menu do mestre segura todas as paradas nas larguras em que aparece', async ({ page }) => {
  await page.goto('/mestre/improviso')
  const menu = '.ds-side-menu'
  let reference = 0

  for (const width of [1920, 1280, 1024]) {
    await page.setViewportSize({ width: width, height: 900 })
    await expect(page.getByRole('link', { name: 'Condições' })).toBeVisible()
    await expectNadaEscapa(page, menu)

    const reachable = await page.locator(`${menu} a`).count()
    if (width === 1920) reference = reachable
    expect(reachable, `a ${width}px o menu perdeu paradas`).toBe(reference)
  }
  expect(reference, 'o menu veio vazio: este caso não mediria nada').toBeGreaterThan(10)
})
})

/**
 * OS FILTROS DOS CATÁLOGOS: à vista no desktop, e num diálogo abaixo dele.
 *
 * O CASO MEDE OS DOIS LADOS, e o segundo não é redundância — é o CONTROLE. Filtros
 * que se escondem em toda largura passariam com folga numa asserção só do celular,
 * e o custo seria eles sumirem da tela larga, que é onde mais servem.
 *
 * Só um browser testemunha: a troca é de CSS por largura, e o mesmo HTML sai nas
 * duas. O visível vem de `toBeVisible`, porque os dois desenhos estão no documento.
 *
 * Ele CAMINHA pelo menu em vez de trazer uma lista de paradas: a parada que entrar
 * amanhã já nasce medida.
 */
test('abaixo do desktop os filtros abrem num diálogo, e no desktop ficam à vista', async ({ page }) => {
  await page.goto('/mestre/bestiario')
  const stops = await page
    .getByRole('navigation', { name: 'Ferramentas do mestre' })
    .getByRole('link')
    .evaluateAll((links) => links.map((l) => (l as HTMLAnchorElement).href))
  expect(stops.length, 'o menu veio vazio: este caso não mediria nada').toBeGreaterThan(10)

  let withFilters = 0
  let noFilters = 0
  for (const waypoint of stops) {
    await page.setViewportSize({ width: 390, height: 844 })
    await page.goto(waypoint)
    const opener = page.locator('[commandfor="catalog-filters"]')
    if ((await opener.count()) === 0) {
      noFilters++
      continue
    }
    withFilters++
    const dialog = page.locator('#catalog-filters')
    const inline = page.locator('.ds-catalog-filters-inline')

    for (const form of [
      { width: 390, height: 844 },
      { width: 844, height: 390 },
    ]) {
      await page.setViewportSize(form)
      const where = `${waypoint} a ${form.width}×${form.height}`
      await expect(opener, `sem o botão, os filtros de ${where} ficam inalcançáveis`).toBeVisible()
      await expect(inline, `os filtros de ${where} ainda ocupam a cena`).toBeHidden()
      await opener.click()
      await expect(dialog.getByRole('button').nth(1), `o diálogo de ${where} não mostrou os filtros`).toBeVisible()
      await dialog.getByRole('button', { name: 'Fechar' }).click()
      await expect(dialog).toBeHidden()
    }

    await page.setViewportSize({ width: 1280, height: 720 })
    await expect(inline, `no desktop os filtros de ${waypoint} sumiram`).toBeVisible()
    await expect(opener, `no desktop ${waypoint} ainda mostra o botão`).toBeHidden()
  }

  // O DENOMINADOR: sem ele, um seletor que parou de casar dá o mesmo verde que
  // uma tela em ordem. Medido: sete paradas filtram.
  expect(withFilters, 'nenhuma parada tinha filtros: o seletor `commandfor` parou de casar').toBeGreaterThanOrEqual(7)
  expect(withFilters + noFilters).toBe(stops.length)
})

/**
 * O CRACHÁ remenda a cena por baixo do diálogo, e o diálogo continua aberto: quem
 * filtra costuma ligar mais de um. O número de trás é a testemunha de que o remendo
 * chegou, senão "continuou aberto" seria verdade sobre um clique que não fez nada.
 */
test('ligar um filtro no diálogo não fecha o diálogo', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 })
  await page.goto('/mestre/magias')
  await page.locator('[commandfor="catalog-filters"]').click()
  const dialog = page.locator('#catalog-filters')
  await dialog.getByRole('button', { name: '2º' }).click()

  await expect(page.locator('.ds-catalog-group > .ds-list-head')).toContainText(' de ')
  await expect(dialog).toBeVisible()
  await expect(dialog.getByRole('button', { name: '2º' })).toHaveAttribute('aria-pressed', 'true')
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
  const first = page.locator('#bestiary a[href*="criatura="]').first()
  await expect(first, 'sem a lista não há medição').toBeVisible()

  // O QUE SOBRA PARA A LISTA é a área que rola menos a barra de filtros presa no
  // alto dela: no celular quem rola é a área principal da moldura, entre o
  // cabeçalho e a barra de baixo, e não a lista sozinha.
  const measure = await first.evaluate((row) => {
    const main = row.closest('main') as HTMLElement
    const bar = main.querySelector('.ds-filter-bar') as HTMLElement
    return {
      caixa: Math.round(main.clientHeight - bar.getBoundingClientRect().height),
      linha: Math.round(row.getBoundingClientRect().height),
    }
  })
  expect(measure.linha, 'uma linha de zero passaria em qualquer teto').toBeGreaterThan(20)
  expect(
    measure.caixa,
    `a lista recebe ${measure.caixa}px e uma criatura mede ${measure.linha}px`,
  ).toBeGreaterThanOrEqual(measure.linha)
})

/**
 * OS FILTROS DO BESTIÁRIO no celular abrem num diálogo, e no desktop seguem numa
 * gaveta: a coluna da lista é estreita, e os filtros à vista empurrariam a lista.
 * O número de trás é a testemunha de que o remendo chegou, senão "continuou aberto"
 * seria verdade sobre um clique que não fez nada.
 */
test('no celular os filtros do Bestiário abrem num diálogo que fica aberto ao filtrar', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 })
  await page.goto('/mestre/bestiario')
  const opener = page.locator('#bestiary').getByRole('button', { name: 'Filtros' })
  const dialog = page.getByRole('dialog', { name: 'Filtros' })
  await expect(page.locator('#bestiary summary', { hasText: 'Filtros' })).toBeHidden()

  await opener.click()
  await dialog.getByRole('button', { name: 'Monstro' }).click()

  await expect(page.locator('#bestiary .ds-list-head').first()).not.toContainText('80 de 80')
  await expect(dialog).toBeVisible()
  await expect(dialog.getByRole('button', { name: 'Monstro' })).toHaveAttribute('aria-pressed', 'true')

  await page.setViewportSize({ width: 1280, height: 720 })
  await expect(opener).toBeHidden()
  await expect(page.locator('#bestiary summary', { hasText: 'Filtros' })).toBeVisible()
})
