import { expect, test } from '@playwright/test'
import { expectNoHorizontalOverflow, VIEWPORTS } from './support/viewports'

test.describe('A carta de convite', () => {
  test.use({ storageState: '.auth/user.json' })

  test('a carta cabe nos seis formatos', async ({ page }) => {
    await page.goto('/campanhas/entrar')
    await expect(page.getByRole('group', { name: /qual herói/i })).toBeVisible()

    await expectNoHorizontalOverflow(page, VIEWPORTS)
  })

  /**
   * NENHUM herói vem escolhido, e quem recusa é o NAVEGADOR.
   *
   * Entrar numa mesa cria uma CÓPIA do herói lá dentro (ALE-33) e sair não é um
   * clique — um rádio já marcado faz quem estiver distraído sentar o herói
   * errado. A recusa nativa do `required` diz por que não enviou, o que o botão
   * desabilitado da SPA não dizia (queixa registrada na ALE-80).
   *
   * E2E porque validação de formulário nativa não existe em jsdom: o
   * `validationMessage` é do navegador, e o envio que NÃO acontece também.
   */
  test('sem escolher herói o navegador barra o envio e diz por quê', async ({ page }) => {
    await page.goto('/campanhas/entrar')
    const radios = page.locator('input[name="characterId"]')
    await expect(radios.first()).toBeAttached()
    expect(await page.locator('input[name="characterId"]:checked').count()).toBe(0)

    await page.getByLabel('Número da campanha').fill('1')
    await page.getByRole('button', { name: 'Entrar na mesa' }).click()

    await expect(page, 'o envio passou sem herói escolhido').toHaveURL(
      /\/campanhas\/entrar$/,
    )
    const notice = await radios.first().evaluate((el: HTMLInputElement) => el.validationMessage)
    expect(notice, 'o navegador barrou em silêncio').not.toBe('')
  })

})
