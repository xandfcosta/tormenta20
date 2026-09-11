import { expect, type Page, test } from '@playwright/test'
import { openTheBoard, disposableTable } from './support/table'

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

async function withTheNotesOpen(page: Page) {
  const mesa = await disposableTable(page)
  await page.setViewportSize({ width: 1920, height: 1080 })
  await openTheBoard(page, mesa.mesa)
  await page.getByRole('button', { name: /Notas/ }).first().click()
  await expect(page.locator('#table-notes')).toBeVisible()
  return mesa
}

const trilhas = (page: Page) =>
  page.evaluate(
    () =>
      getComputedStyle(document.querySelector('.notes-layout')!).gridTemplateColumns.split(' ')
        .length,
  )

const width = (page: Page) =>
  page.evaluate(() => Math.round(document.getElementById('table-notes')!.getBoundingClientRect().width))

test('empilhado põe a prévia ABAIXO da caixa, mesmo onde as duas colunas cabem', async ({
  page,
}) => {
  const { apagar } = await withTheNotesOpen(page)
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
    await expect(page.locator('#table-notes-preview')).toBeVisible()
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
  const { apagar } = await withTheNotesOpen(page)
  try {
    const padrao = await width(page)
    // A 1920, 40% do palco. O número exato é do leiaute; o que importa é ele
    // ser o ponto de partida e o destino do `Home`.
    expect(padrao, 'a coluna nasceu sem largura: a medição não vale').toBeGreaterThan(400)

    const divisa = page.getByRole('separator', { name: 'Largura das notas' })
    await divisa.focus()

    // ESQUERDA cresce as notas: a divisa está à esquerda da coluna, e empurrá-la
    // para lá toma espaço do mapa.
    for (let i = 0; i < 4; i++) await divisa.press('ArrowLeft')
    const maior = await width(page)
    expect(maior, `a seta não alargou a coluna (${padrao} → ${maior})`).toBeGreaterThan(padrao)

    for (let i = 0; i < 8; i++) await divisa.press('ArrowRight')
    const menor = await width(page)
    expect(menor, `a seta não estreitou a coluna (${maior} → ${menor})`).toBeLessThan(maior)

    await divisa.press('Home')
    expect(await width(page), 'o Home não devolveu a largura padrão').toBe(padrao)
  } finally {
    await apagar()
  }
})

/**
 * A ESCOLHA GRUDA, como os modos grudam — é preferência de trabalho e não
 * estado da sessão.
 */
test('a largura sobrevive ao recarregar', async ({ page }) => {
  const { mesa, apagar } = await withTheNotesOpen(page)
  try {
    const divisa = page.getByRole('separator', { name: 'Largura das notas' })
    await divisa.focus()
    for (let i = 0; i < 4; i++) await divisa.press('ArrowLeft')
    const escolhida = await width(page)

    await page.goto(mesa)
    await page.getByRole('button', { name: /Notas/ }).first().click()
    await expect(page.locator('#table-notes')).toBeVisible()

    expect(await width(page), 'a largura escolhida não sobreviveu ao F5').toBe(escolhida)
  } finally {
    await apagar()
  }
})

/**
 * AS NOTAS FLUTUANDO SOBRE O MAPA (ALE-218), e o que ele prende é que o mapa
 * NÃO ENCOLHE.
 *
 * Essa é a diferença inteira entre as duas formas, e ela é geométrica: encostada
 * a coluna toma largura do tabuleiro; flutuando ela passa por cima. Um guarda
 * que só afirmasse "a coluna está posicionada" mediria CSS em vez do efeito.
 *
 * O cabeçalho do `notes.templ` registra que a coluna que EMPURRA é o desenho
 * certo para narrar olhando o tabuleiro — ela continua sendo o padrão, e este
 * caso começa provando isso.
 */
test('flutuar as notas não encolhe o mapa, e encostar volta a encolher', async ({ page }) => {
  const { apagar } = await withTheNotesOpen(page)
  try {
    const mapa = () =>
      page.evaluate(() => Math.round(document.querySelector('.board-scene')!.getBoundingClientRect().width))

    // A REFERÊNCIA é o mapa SEM notas: é o tamanho que flutuar tem de devolver.
    await page.getByRole('button', { name: 'Fechar as notas' }).click()
    await page.waitForTimeout(250)
    const semNotas = await mapa()
    await page.getByRole('button', { name: /Notas/ }).first().click()
    await expect(page.locator('#table-notes')).toBeVisible()
    await page.waitForTimeout(250)

    // O PADRÃO É ENCOSTADA, e o mapa paga por isso.
    const encostada = await mapa()
    const flutuar = page.getByRole('button', { name: 'Flutuar as notas sobre o mapa' })
    await expect(flutuar, 'o alternador nasceu ligado: o padrão deixou de ser a coluna').toHaveAttribute(
      'aria-pressed',
      'false',
    )

    await flutuar.click()
    await page.waitForTimeout(250)
    const flutuando = await mapa()
    // O ALVO É O MAPA FECHADO, e não "maior que antes": a primeira versão deste
    // guarda cobrava só crescimento, e passou verde sobre uma coluna que ficou
    // no fluxo — o mapa crescia 18px em vez de 728, porque só a DIVISA tinha
    // flutuado. "Cresceu um pouco" e "saiu do caminho" são coisas diferentes.
    expect(
      flutuando,
      `flutuar devolveu só ${flutuando - encostada}px ao mapa; fechada ela mede ${semNotas}`,
    ).toBeGreaterThan(semNotas - 20)

    // E as notas continuam à mostra POR CIMA: flutuar não é fechar.
    await expect(page.locator('#table-notes')).toBeVisible()

    // DO LADO CERTO, e este pedaço existe porque a primeira versão do guarda
    // não o tinha: o painel foi parar na ESQUERDA do mapa — a classe base traz
    // `inset-0`, que põe `left: 0`, e com os dois lados definidos o navegador
    // resolve pelo left. O mapa media certo e a tela estava errada.
    const [naDireita, meio] = await page.evaluate(() => {
      const n = document.getElementById('table-notes')!.getBoundingClientRect()
      return [Math.round(n.x + n.width), Math.round(innerWidth / 2)]
    })
    expect(naDireita, 'as notas flutuaram do lado errado do mapa').toBeGreaterThan(meio)

    await flutuar.click()
    await page.waitForTimeout(250)
    expect(await mapa(), 'encostar não devolveu o mapa ao tamanho de antes').toBe(encostada)
  } finally {
    await apagar()
  }
})
