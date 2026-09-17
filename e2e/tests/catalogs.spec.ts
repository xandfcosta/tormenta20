import { expect, test } from '@playwright/test'
import { expectNoHorizontalOverflow, VIEWPORTS } from './support/viewports'

test.describe('Os catálogos', () => {
  test.use({ storageState: '.auth/user.json' })

  const CATALOGOS = '/mestre/condicoes'

  /**
   * O único guarda desta cena que precisa mesmo de browser.
   *
   * A cena manda as centenas de entradas de uma vez — sem virtualização, por
   * decisão do dono. O que sustenta essa decisão é a lista rolar DENTRO da
   * caixa, e "a cena não rola" fica VERDE por cima do defeito: a lista cresce
   * até a altura do conteúdo e vaza para fora do cartão sem a página rolar.
   *
   * Por isso a asserção é a INVERSA: o DOCUMENTO não pode ser mais alto que a
   * janela. Um `min-h-0` faltando em qualquer elo da corrente dá uma página de
   * dezenas de milhares de pixels — e a aba visitada é a de Poderes, que é a
   * maior.
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
   * O TETO DE TRÊS COLUNAS é medida de leitura: acima de três a linha fica
   * curta demais para descrição de regra, e abaixo de 22rem a prosa vira fita.
   * Um `auto-fill minmax(22rem, 1fr)` puro mede QUATRO colunas a 1920.
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
   * A busca varre TODOS os catálogos, não só o que está aberto: filtrar só a
   * aba ativa faz "bola de fogo" digitado em Condições dizer "nada encontrado"
   * com a magia existindo (ALE-22).
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
