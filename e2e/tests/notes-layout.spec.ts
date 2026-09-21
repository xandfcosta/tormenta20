import { expect, type Page, test } from '@playwright/test'
import { openTheBoard, disposableTable } from './support/table'

/**
 * AS FORMAS DAS NOTAS DA SESSÃO.
 *
 * E2E porque as garantias são de LEIAUTE REAL: quantas trilhas a grade tem
 * depois de o navegador resolver uma consulta de contêiner, e quantos pixels a
 * coluna mede depois de um arrasto. Em jsdom nenhuma folha se aplica e as duas
 * respostas seriam zero.
 */
test.use({ storageState: '.auth/user.json' })

async function withTheNotesOpen(page: Page) {
  const table = await disposableTable(page)
  await page.setViewportSize({ width: 1920, height: 1080 })
  await openTheBoard(page, table.mesa)
  await page.getByRole('button', { name: /Notas/ }).first().click()
  await expect(page.locator('#table-notes')).toBeVisible()
  return table
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
  const { apagar: remove } = await withTheNotesOpen(page)
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
    await remove()
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
  const { apagar: remove } = await withTheNotesOpen(page)
  try {
    const standard = await width(page)
    // A 1920, 40% do palco. O número exato é do leiaute; o que importa é ele
    // ser o ponto de partida e o destino do `Home`.
    expect(standard, 'a coluna nasceu sem largura: a medição não vale').toBeGreaterThan(400)

    const divider = page.getByRole('separator', { name: 'Largura das notas' })
    await divider.focus()

    // ESQUERDA cresce as notas: a divisa está à esquerda da coluna, e empurrá-la
    // para lá toma espaço do mapa.
    for (let i = 0; i < 4; i++) await divider.press('ArrowLeft')
    const max = await width(page)
    expect(max, `a seta não alargou a coluna (${standard} → ${max})`).toBeGreaterThan(standard)

    for (let i = 0; i < 8; i++) await divider.press('ArrowRight')
    const min = await width(page)
    expect(min, `a seta não estreitou a coluna (${max} → ${min})`).toBeLessThan(max)

    await divider.press('Home')
    expect(await width(page), 'o Home não devolveu a largura padrão').toBe(standard)
  } finally {
    await remove()
  }
})

/**
 * A ESCOLHA GRUDA, como os modos grudam — é preferência de trabalho e não
 * estado da sessão.
 */
test('a largura sobrevive ao recarregar', async ({ page }) => {
  const { mesa: tableState, apagar: erase } = await withTheNotesOpen(page)
  try {
    const divider = page.getByRole('separator', { name: 'Largura das notas' })
    await divider.focus()
    for (let i = 0; i < 4; i++) await divider.press('ArrowLeft')
    const chosen = await width(page)

    await page.goto(tableState)
    await page.getByRole('button', { name: /Notas/ }).first().click()
    await expect(page.locator('#table-notes')).toBeVisible()

    expect(await width(page), 'a largura escolhida não sobreviveu ao F5').toBe(chosen)
  } finally {
    await erase()
  }
})

/**
 * AS NOTAS FLUTUANDO SOBRE O MAPA, e o que se prende é que o mapa NÃO ENCOLHE.
 *
 * Essa é a diferença inteira entre as duas formas, e ela é geométrica: encostada
 * a coluna toma largura do tabuleiro; flutuando ela passa por cima. Um guarda
 * que só afirmasse "a coluna está posicionada" mediria CSS em vez do efeito.
 */
test('flutuar as notas não encolhe o mapa, e encostar volta a encolher', async ({ page }) => {
  const { apagar: erase } = await withTheNotesOpen(page)
  try {
    const board = () =>
      page.evaluate(() => Math.round(document.querySelector('.board-scene')!.getBoundingClientRect().width))

    // A REFERÊNCIA é o mapa SEM notas: é o tamanho que flutuar tem de devolver.
    await page.getByRole('button', { name: 'Fechar as notas' }).click()
    await page.waitForTimeout(250)
    const noNotes = await board()
    await page.getByRole('button', { name: /Notas/ }).first().click()
    await expect(page.locator('#table-notes')).toBeVisible()
    await page.waitForTimeout(250)

    // O PADRÃO É ENCOSTADA, e o mapa paga por isso.
    const flush = await board()
    const float = page.getByRole('button', { name: 'Flutuar as notas sobre o mapa' })
    await expect(float, 'o alternador nasceu ligado: o padrão deixou de ser a coluna').toHaveAttribute(
      'aria-pressed',
      'false',
    )

    await float.click()
    await page.waitForTimeout(250)
    const floating = await board()
    // O ALVO É O MAPA FECHADO, e não "maior que antes": cobrando só crescimento,
    // o guarda passa verde sobre uma coluna que ficou no fluxo e só teve a DIVISA
    // flutuando — 18px devolvidos em vez de 728. "Cresceu um pouco" e "saiu do
    // caminho" são coisas diferentes.
    expect(
      floating,
      `flutuar devolveu só ${floating - flush}px ao mapa; fechada ela mede ${noNotes}`,
    ).toBeGreaterThan(noNotes - 20)

    // E as notas continuam à mostra POR CIMA: flutuar não é fechar.
    await expect(page.locator('#table-notes')).toBeVisible()

    // DO LADO CERTO: a classe base traz `inset-0`, que põe `left: 0`, e com os
    // dois lados definidos o navegador resolve pelo left — o painel vai parar na
    // ESQUERDA do mapa. A medida do mapa acima continua certa e a tela está
    // errada, por isso a geometria é afirmada à parte.
    const [onRight, halfway] = await page.evaluate(() => {
      const n = document.getElementById('table-notes')!.getBoundingClientRect()
      return [Math.round(n.x + n.width), Math.round(innerWidth / 2)]
    })
    expect(onRight, 'as notas flutuaram do lado errado do mapa').toBeGreaterThan(halfway)

    await float.click()
    await page.waitForTimeout(250)
    expect(await board(), 'encostar não devolveu o mapa ao tamanho de antes').toBe(flush)
  } finally {
    await erase()
  }
})

/**
 * A JANELA PRÓPRIA, e ela é EXCLUSIVA com a coluna.
 *
 * E2E porque a garantia atravessa DOIS documentos: uma janela anuncia por
 * `localStorage` que tomou as notas, e a outra fecha a coluna ao ouvir o evento
 * `storage` — que só existe entre janelas de verdade. Um teste de handler vê as
 * duas metades do HTML e nunca vê o pacto acontecendo, e é o pacto que impede
 * duas caixas de escreverem a mesma coluna do banco.
 *
 * O caminho mais óbvio — mover o nó vivo para uma janela de Document
 * Picture-in-Picture — foi medido e descartado: o Datastar não segue o nó
 * adotado por outro documento, e o painel chega lá MUDO, com a faixa ainda
 * dizendo "Salvo". Daí a cena com endereço próprio.
 */
test('destacar as notas abre uma janela e FECHA a coluna; fechá-la devolve as duas', async ({
  page,
}) => {
  const { apagar: remove } = await withTheNotesOpen(page)
  try {
    // O CONTROLE: a coluna está aberta AGORA, e é isso que a janela tem de
    // desfazer. Sem ele, "a coluna está fechada no fim" seria verdade também
    // se ela nunca tivesse aberto.
    await expect(page.locator('#table-notes')).toBeVisible()

    const [viewport] = await Promise.all([
      page.waitForEvent('popup'),
      page.getByRole('button', { name: 'Abrir as notas numa janela' }).click(),
    ])
    await viewport.waitForLoadState('domcontentloaded')

    expect(viewport.url(), 'a janela não abriu no endereço das notas').toContain('/notas')
    await expect(viewport.locator('#notes-window')).toBeVisible()
    await expect(viewport.getByRole('textbox', { name: 'Notas da sessão' })).toBeVisible()

    // A EXCLUSÃO acontecendo: a aba principal fechou a coluna porque a janela
    // anunciou, e não porque o clique pediu.
    await expect(page.locator('#table-notes')).toBeHidden()

    // E o botão do trilho passa a dizer que as notas estão em outro lugar — sem
    // isto, quem voltasse à aba principal veria um botão que "não faz nada".
    const rail = page.getByRole('button', { name: 'Notas da sessão' }).first()
    await expect(rail).toHaveAttribute('aria-pressed', 'true')

    // A JANELA SALVA sozinha, pelo mesmo autosave da coluna.
    //
    // A asserção é o POST e não a faixa dizer "Salvo": ela JÁ diz "Salvo" antes
    // de alguém digitar, porque `$notes` e `$notes_saved` nascem iguais. Um
    // mostrador cujo REPOUSO é igual ao sucesso não testemunha o sucesso.
    const saved = viewport.waitForResponse(
      (r) => r.url().includes('/notas') && r.request().method() === 'POST' && r.ok(),
    )
    await viewport.getByRole('textbox', { name: 'Notas da sessão' }).fill('# Escrito na janela')
    await saved

    await viewport.close()

    // DEVOLVIDA: a coluna volta a ser oferecida, e o texto que a janela gravou
    // está lá — que é a prova de que as duas falam com a mesma linha do banco.
    await expect(rail).toHaveAttribute('aria-pressed', 'false')
    await rail.click()
    await expect(page.locator('#table-notes')).toBeVisible()
    await page.reload()
    await page.getByRole('button', { name: /Notas/ }).first().click()
    await expect(page.getByRole('textbox', { name: 'Notas da sessão' })).toHaveValue(
      '# Escrito na janela',
    )
  } finally {
    await remove()
  }
})
