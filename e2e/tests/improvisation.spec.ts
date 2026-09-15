import { expect, test } from '@playwright/test'
import { expectDentroDaJanela } from './support/geometry'
import { expectNoHorizontalOverflow, VIEWPORTS } from './support/viewports'

test.describe('O improviso', () => {
  test.use({ storageState: '.auth/user.json' })

  const IMPROVISO = '/mestre/improviso'

  test('o improviso cabe nos seis formatos', async ({ page }) => {
    await page.goto(IMPROVISO)
    await expectNoHorizontalOverflow(page, VIEWPORTS)
    for (const viewport of VIEWPORTS) {
      await page.setViewportSize({ width: viewport.width, height: viewport.height })
      await expectDentroDaJanela(page)
    }
  })

  /**
   * ROLAR NÃO EMPILHA HISTÓRICO, e é a mesma decisão do rascunho do encontro
   * (ALE-259): o mestre rola várias vezes seguidas, e cada rolagem virando uma
   * entrada faria o botão Voltar desfazer dados em vez de sair da tela.
   *
   * E2E porque o histórico é do NAVEGADOR — `history.length` e `goBack` não
   * existem em jsdom.
   */
  test('rolar não empilha histórico, e o Voltar sai da tela', async ({ page }) => {
    await page.goto('/')
    await page.goto(IMPROVISO)
    const antes = await page.evaluate(() => history.length)

    for (let i = 0; i < 3; i++) {
      await page.getByRole('button', { name: 'Rolar d20' }).first().click()
      await expect(page.locator('#improv [aria-live="polite"]').first()).toBeVisible()
    }

    expect(await page.evaluate(() => history.length), 'as rolagens empilharam histórico').toBe(
      antes,
    )
    await page.goBack()
    await expect(page, 'o Voltar desfez uma rolagem em vez de sair').toHaveURL(/\/$/)
  })

  /**
   * O HISTÓRICO CURTO é o que a ferramenta tem de diferente de um botão que
   * mostra o último resultado: o mestre que rola duas vezes na mesma cena quer
   * comparar. Cinco é o fundo, e a sexta rolagem empurra a primeira para fora.
   */
  test('a tabela guarda as rolagens anteriores, e para em cinco', async ({ page }) => {
    await page.goto(IMPROVISO)
    // Pelo NOME da região, e não por contar filhos de `div`: acoplar ao formato
    // do DOM é o que o guia manda não escrever, e a primeira versão deste teste
    // fazia isso — e apontava para o cartão errado.
    const cartao = page.getByRole('region', { name: 'Ermos — Ruína' })

    for (let i = 0; i < 7; i++) {
      await cartao.getByRole('button', { name: 'Rolar d6' }).click()
      await expect(cartao.locator('[aria-live="polite"]')).toBeVisible()
    }
    // Uma manchete + quatro anteriores = os cinco que o fundo permite.
    await expect(cartao.locator('li'), 'o histórico passou do fundo de cinco').toHaveCount(4)
  })
})
