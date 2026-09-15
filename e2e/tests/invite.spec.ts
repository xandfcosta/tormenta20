import { expect, test } from '@playwright/test'
import { expectDentroDaJanela } from './support/geometry'
import { expectNoHorizontalOverflow, VIEWPORTS } from './support/viewports'

test.describe('A carta de convite', () => {
  test.use({ storageState: '.auth/user.json' })

  test('a carta cabe nos seis formatos', async ({ page }) => {
    await page.goto('/campanhas/entrar')
    await expect(page.getByRole('group', { name: /Qual herói/ })).toBeVisible()

    await expectNoHorizontalOverflow(page, VIEWPORTS)
    for (const viewport of VIEWPORTS) {
      await page.setViewportSize({ width: viewport.width, height: viewport.height })
      await expectDentroDaJanela(page)
    }
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
    const aviso = await radios.first().evaluate((el: HTMLInputElement) => el.validationMessage)
    expect(aviso, 'o navegador barrou em silêncio').not.toBe('')
  })

  // Convite morto: a carta diz, e NÃO oferece o botão. Um botão que não pode
  // funcionar é uma porta pintada na parede — quem explica é a carta.
  test('convite morto vira frase, e a carta não oferece o botão', async ({ page }) => {
    await page.goto('/campanhas/entrar?token=nao-existe-mesmo')
    await expect(page.getByText(/Convite inválido ou expirado/)).toBeVisible()
    await expect(page.getByRole('button', { name: 'Entrar na mesa' })).toHaveCount(0)
  })

})
