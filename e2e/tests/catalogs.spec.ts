import { expect, test } from '@playwright/test'
import { expectNoHorizontalOverflow, VIEWPORTS } from './support/viewports'

test.describe('Os catálogos', () => {
  test.use({ storageState: '.auth/user.json' })

  const CATALOGOS = '/mestre/condicoes'

  /**
   * A ALE-149 DO LADO DO SERVIDOR, e é o único guarda desta cena que precisa
   * mesmo de browser.
   *
   * A cena manda as 992 entradas de uma vez — sem virtualização, por decisão do
   * dono. O que sustenta essa decisão é a lista rolar DENTRO da caixa, e a
   * versão em React já errou exatamente isso: a lista crescia até a altura do
   * conteúdo e vazava 1854–2566px para fora do cartão sem a página rolar, então
   * a asserção "a cena não rola" ficava verde por cima do defeito.
   *
   * Aqui a asserção é a inversa e mede o que aquela não media: o DOCUMENTO não
   * pode ser mais alto que a janela. Com 566 poderes desenhados, um `min-h-0`
   * faltando em qualquer elo da corrente dá uma página de dezenas de milhares
   * de pixels — e é a aba de Poderes que precisa ser visitada, porque é a maior.
   */
  test('com o acervo inteiro na tela, a PÁGINA não cresce em nenhum formato', async ({
    page,
  }) => {
    for (const viewport of VIEWPORTS) {
      await page.setViewportSize({ width: viewport.width, height: viewport.height })
      await page.goto(`${CATALOGOS}?aba=poderes`)
      const altura = await page.evaluate(() => ({
        doc: document.documentElement.scrollHeight,
        janela: window.innerHeight,
      }))
      expect(
        altura.doc,
        `a página cresceu para ${altura.doc}px em ${viewport.width}×${viewport.height} — é a ALE-149`,
      ).toBeLessThanOrEqual(altura.janela + 2)
    }
    await expectNoHorizontalOverflow(page, VIEWPORTS)
  })

  /**
   * O TETO DE TRÊS COLUNAS é medida de leitura (ALE-170), e ele quase sumiu no
   * porte: a primeira versão usava `auto-fill minmax(22rem, 1fr)` puro e MEDIU
   * quatro colunas a 1920. Acima de três a linha fica curta demais para
   * descrição de regra; abaixo de 22rem a prosa vira fita.
   *
   * E2E porque a única testemunha é o `gridTemplateColumns` COMPUTADO — a
   * classe é a mesma em toda largura, então asserção de classe não veria nada.
   */
  test('a grade nunca passa de três colunas, e nunca some', async ({ page }) => {
    for (const [largura, altura, esperado] of [
      [1920, 1080, 3],
      [1440, 900, 3],
      [1024, 768, 2],
      [390, 844, 1],
    ] as const) {
      await page.setViewportSize({ width: largura, height: altura })
      await page.goto(`${CATALOGOS}?aba=condicoes`)
      const colunas = await page.evaluate(() => {
        const grade = document.querySelector('.collection-in-columns')
        if (!grade) return 0
        return getComputedStyle(grade).gridTemplateColumns.split(' ').length
      })
      expect(colunas, `${largura}px devia dar ${esperado} coluna(s)`).toBe(esperado)
    }
  })

  /**
   * A busca varre TODOS os catálogos, não só o que está aberto.
   *
   * É a ALE-22: a versão em React filtrava só a aba ativa, então "bola de fogo"
   * digitado na aba Condições dizia "nada encontrado" com a magia existindo.
   *
   * A segunda metade deste guarda — "e a fileira de abas sai de cena" — saiu
   * junto com a fileira: cada catálogo mora hoje no seu endereço e quem troca é
   * o trilho do mestre, que é permanente. Não sobrou fileira para sumir, e uma
   * asserção sobre elemento que não existe passa verde sobre nada.
   */
  test('buscar varre todos os catálogos, não só o que está aberto', async ({ page }) => {
    await page.setViewportSize({ width: 1440, height: 900 })
    await page.goto(`${CATALOGOS}?aba=condicoes`)
    await expect(page.getByRole('navigation', { name: 'Ferramentas do mestre' })).toBeVisible()

    await page.getByRole('searchbox', { name: 'Buscar nos catálogos' }).fill('fogo')

    await expect(
      page.getByRole('region', { name: 'Magias' }),
      'buscar de dentro das condições não alcançou as magias (ALE-22)',
    ).toBeVisible()
  })
})
