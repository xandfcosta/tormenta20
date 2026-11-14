import { expect, test } from '@playwright/test'
import { expectNoHorizontalOverflow, VIEWPORTS } from './support/viewports'

test.describe('O improviso', () => {
  test.use({ storageState: '.auth/user.json' })

  const IMPROV = '/mestre/improviso'

  test('o improviso cabe nos seis formatos', async ({ page }) => {
    await page.goto(IMPROV)
    await expectNoHorizontalOverflow(page, VIEWPORTS)
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
    await page.goto(IMPROV)
    const earlier = await page.evaluate(() => history.length)

    for (let i = 0; i < 3; i++) {
      await page.getByRole('button', { name: 'Rolar d20' }).first().click()
      await expect(page.locator('#improv [aria-live="polite"]').first()).toBeVisible()
    }

    expect(await page.evaluate(() => history.length), 'as rolagens empilharam histórico').toBe(
      earlier,
    )
    await page.goBack()
    await expect(page, 'o Voltar desfez uma rolagem em vez de sair').toHaveURL(/\/$/)
  })

  /**
   * A SETA chega aos botões dos cartões e não mexe no campo de salas: num campo de número
   * ela mudaria o valor. E2E porque a escolha do vizinho é geometria real, e só existe com
   * o driver ligado.
   */
  test('a seta anda pelos botões e não mexe nas salas', async ({ page }) => {
    await page.setViewportSize({ width: 1400, height: 900 })
    await page.goto(IMPROV)
    await page.waitForLoadState('networkidle')
    const region = () =>
      page.evaluate(() => document.activeElement?.closest('[data-nav-region]')?.getAttribute('data-nav-region'))
    const rooms = await page.getByLabel('Salas').inputValue()

    await page.locator('.ds-side-menu-link[aria-current]').focus()
    await page.keyboard.press('ArrowRight')
    await expect.poll(region).toBe('tabelas')
    for (const key of ['ArrowDown', 'ArrowDown', 'ArrowRight', 'ArrowDown']) await page.keyboard.press(key)

    await expect(page.getByLabel('Salas'), 'a seta mexeu no número de salas').toHaveValue(rooms)
  })
})
