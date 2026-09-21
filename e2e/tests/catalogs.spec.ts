import { expect, test } from '@playwright/test'
import { expectNothingIsClippedSideways } from './support/geometry'
import {
  expectNoHorizontalOverflow,
  expectOnlyTheScrollerScrolls,
  VIEWPORTS,
} from './support/viewports'

test.describe('Os catálogos', () => {
  test.use({ storageState: '.auth/user.json' })

  // Cada catálogo é uma CENA e a aba vem do CAMINHO: o `?aba=` da consulta não
  // tem leitor nenhum, e um endereço que o carregasse mediria o acervo do
  // caminho fingindo medir o outro (ALE-332).
  const CONDITIONS = '/mestre/condicoes'
  const POWERS = '/mestre/poderes'
  const SCROLLER = '[role="region"][aria-label="Resultados dos catálogos"]'

  /**
   * O único guarda desta cena que precisa mesmo de browser.
   *
   * A cena manda as centenas de entradas de uma vez — sem virtualização, por
   * decisão do dono. O que sustenta essa decisão é a lista rolar DENTRO da
   * caixa: deixada crescer até a altura do conteúdo, ela vaza milhares de
   * pixels para fora do cartão (ALE-149).
   *
   * Este caso mediu a coisa errada duas vezes, e as duas estão consertadas
   * aqui (ALE-332). Pedia `/mestre/condicoes?aba=poderes` — consulta que
   * handler nenhum lê, porque a aba vem do CAMINHO —, então media o catálogo
   * mais magro do livro dizendo no comentário que media o mais gordo. E
   * afirmava que o DOCUMENTO não crescia, o que a casca `h-dvh
   * overflow-hidden` já garante: com o rolador sabotado a lista foi a 25.187px
   * e a página continuou com 900. O instrumento está no `viewports.ts`, com a
   * medição.
   */
  test('com o acervo inteiro na tela, a lista rola DENTRO da caixa em todo formato', async ({
    page,
  }) => {
    await page.goto(POWERS)
    // O DENOMINADOR desta cena: Poderes é o maior acervo do livro, e é dele que
    // o defeito precisa — com uma fração das entradas a corrente aguenta e o
    // caso fica verde sem ter medido nada.
    const drawn = await page.locator('.collection-in-columns > *').count()
    expect(
      drawn,
      `a cena desenhou ${drawn} verbetes, e Poderes tem centenas — este caso não está medindo o acervo que ele diz medir`,
    ).toBeGreaterThan(500)

    await expectOnlyTheScrollerScrolls(page, SCROLLER, VIEWPORTS)
    await expectNoHorizontalOverflow(page, VIEWPORTS)

    // E o eixo horizontal PELA CENA, porque o do documento é inerte aqui: a
    // casca é `overflow-hidden`, então o cartão que passava 44px da caixa a
    // 390px era recortado sem o documento crescer um pixel (ALE-337).
    for (const viewport of VIEWPORTS) {
      await page.setViewportSize({ width: viewport.width, height: viewport.height })
      await expectNothingIsClippedSideways(page, '#catalogs')
    }
  })

  /**
   * O TETO DE TRÊS COLUNAS é medida de leitura: acima de três a linha fica
   * curta demais para descrição de regra, e abaixo de 22rem a prosa vira fita.
   * Um `auto-fill minmax(22rem, 1fr)` puro mede QUATRO colunas a 1920.
   *
   * E2E porque a única testemunha é o `gridTemplateColumns` COMPUTADO — a
   * classe é a mesma em toda largura, então asserção de classe não veria nada.
   */
  test('a grade nunca passa de três colunas, e nunca some', async ({ page }) => {
    for (const [width, height, want] of [
      [1920, 1080, 3],
      [1440, 900, 3],
      [1024, 768, 2],
      [390, 844, 1],
    ] as const) {
      await page.setViewportSize({ width: width, height: height })
      await page.goto(CONDITIONS)
      const columns = await page.evaluate(() => {
        const grade = document.querySelector('.collection-in-columns')
        if (!grade) return 0
        return getComputedStyle(grade).gridTemplateColumns.split(' ').length
      })
      expect(columns, `${width}px devia dar ${want} coluna(s)`).toBe(want)
    }
  })

  /**
   * A busca varre TODOS os catálogos, não só o que está aberto: filtrar só a
   * aba ativa faz "bola de fogo" digitado em Condições dizer "nada encontrado"
   * com a magia existindo (ALE-22).
   */
  test('buscar varre todos os catálogos, não só o que está aberto', async ({ page }) => {
    await page.setViewportSize({ width: 1440, height: 900 })
    await page.goto(CONDITIONS)
    await expect(page.getByRole('navigation', { name: 'Ferramentas do mestre' })).toBeVisible()

    await page.getByRole('searchbox', { name: 'Buscar nos catálogos' }).fill('fogo')

    await expect(
      page.getByRole('region', { name: 'Magias' }),
      'buscar de dentro das condições não alcançou as magias (ALE-22)',
    ).toBeVisible()
  })
})
