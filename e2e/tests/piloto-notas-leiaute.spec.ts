import { expect, type Page, test } from '@playwright/test'
import { abreOTabuleiro, mesaDescartavel } from './support/mesa'

/**
 * AS FORMAS DAS NOTAS DA SESSÃO (ALE-218).
 *
 * A coluna tinha três modos e uma largura FIXA em 40% do palco (ALE-198). Ela
 * ganhou uma quarta forma — empilhado — e uma divisa que se arrasta.
 *
 * E2E porque as duas garantias são de LEIAUTE REAL: quantas trilhas a grade tem
 * depois de o navegador resolver uma consulta de contêiner, e quantos pixels a
 * coluna mede depois de um arrasto. Em jsdom nenhuma folha se aplica e as duas
 * respostas seriam zero.
 */
test.use({ storageState: '.auth/user.json' })

async function comAsNotasAbertas(page: Page) {
  const mesa = await mesaDescartavel(page)
  await page.setViewportSize({ width: 1920, height: 1080 })
  await abreOTabuleiro(page, mesa.mesa)
  await page.getByRole('button', { name: /Notas/ }).first().click()
  await expect(page.locator('#mesa-notas')).toBeVisible()
  return mesa
}

const trilhas = (page: Page) =>
  page.evaluate(
    () =>
      getComputedStyle(document.querySelector('.notas-arranjo')!).gridTemplateColumns.split(' ')
        .length,
  )

const largura = (page: Page) =>
  page.evaluate(() => Math.round(document.getElementById('mesa-notas')!.getBoundingClientRect().width))

test('empilhado põe a prévia ABAIXO da caixa, mesmo onde as duas colunas cabem', async ({
  page,
}) => {
  const { apagar } = await comAsNotasAbertas(page)
  try {
    // O CONTROLE: a 1920 a coluna mede 704px e o "lado a lado" REPARTE. Sem
    // isto, "empilhado tem uma trilha" seria verdade também numa largura em que
    // nada reparte — e o modo novo não estaria provando nada.
    await page.getByRole('button', { name: 'Lado a lado' }).click()
    expect(await trilhas(page), 'a 1920 o lado a lado não repartiu: o controle não vale').toBe(2)

    await page.getByRole('button', { name: 'Empilhado' }).click()
    expect(
      await trilhas(page),
      'empilhado continuou repartindo: ele é a MESMA coisa que lado a lado',
    ).toBe(1)

    // E as DUAS metades continuam à mostra — empilhado não é "Escrever" com
    // outro nome.
    await expect(page.getByRole('textbox', { name: 'Notas da sessão' })).toBeVisible()
    await expect(page.locator('#mesa-notas-previa')).toBeVisible()
  } finally {
    await apagar()
  }
})

/**
 * A DIVISA responde ao TECLADO, e este caso não é cortesia.
 *
 * A regra da casa diz que **gesto nunca é o único caminho**: uma divisa que só
 * responde a arrasto é uma preferência que quem não usa ponteiro não tem.
 *
 * O teclado é medido em vez do arrasto porque ele é o caminho que se perde em
 * silêncio — um arrasto quebrado alguém percebe na primeira tentativa.
 */
test('a divisa anda pelo teclado e o Home devolve a largura padrão', async ({ page }) => {
  const { apagar } = await comAsNotasAbertas(page)
  try {
    const padrao = await largura(page)
    // A 1920, 40% do palco. O número exato é do leiaute; o que importa é ele
    // ser o ponto de partida e o destino do `Home`.
    expect(padrao, 'a coluna nasceu sem largura: a medição não vale').toBeGreaterThan(400)

    const divisa = page.getByRole('separator', { name: 'Largura das notas' })
    await divisa.focus()

    // ESQUERDA cresce as notas: a divisa está à esquerda da coluna, e empurrá-la
    // para lá toma espaço do mapa.
    for (let i = 0; i < 4; i++) await divisa.press('ArrowLeft')
    const maior = await largura(page)
    expect(maior, `a seta não alargou a coluna (${padrao} → ${maior})`).toBeGreaterThan(padrao)

    for (let i = 0; i < 8; i++) await divisa.press('ArrowRight')
    const menor = await largura(page)
    expect(menor, `a seta não estreitou a coluna (${maior} → ${menor})`).toBeLessThan(maior)

    await divisa.press('Home')
    expect(await largura(page), 'o Home não devolveu a largura padrão').toBe(padrao)
  } finally {
    await apagar()
  }
})

/**
 * A ESCOLHA GRUDA, como os modos grudam — é preferência de trabalho e não
 * estado da sessão.
 */
test('a largura sobrevive ao recarregar', async ({ page }) => {
  const { mesa, apagar } = await comAsNotasAbertas(page)
  try {
    const divisa = page.getByRole('separator', { name: 'Largura das notas' })
    await divisa.focus()
    for (let i = 0; i < 4; i++) await divisa.press('ArrowLeft')
    const escolhida = await largura(page)

    await page.goto(mesa)
    await page.getByRole('button', { name: /Notas/ }).first().click()
    await expect(page.locator('#mesa-notas')).toBeVisible()

    expect(await largura(page), 'a largura escolhida não sobreviveu ao F5').toBe(escolhida)
  } finally {
    await apagar()
  }
})
