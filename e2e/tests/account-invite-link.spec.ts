import { expect, test } from '@playwright/test'

test.describe('O link de convite de conta', () => {
  test.use({ storageState: '.auth/user.json' })

  // E2E porque quem escreve o endereço no campo é o `data-init`, no navegador: o
  // HTML que o servidor manda já vinha certo enquanto a tela mostrava o link velho.
  test('gerar outro link troca o endereço que a tela mostra', async ({ page }) => {
    await page.goto('/')
    const dialog = page.locator('dialog#hub-invite')
    await dialog.evaluate((el: HTMLDialogElement) => el.showModal())
    const generate = dialog.getByRole('button', { name: 'Gerar novo link' })
    const field = dialog.locator('#invite-url')

    await generate.click()
    await expect(field).toHaveValue(/convite=/)
    const first = await field.inputValue()

    await generate.click()
    await expect(field).not.toHaveValue(first)
    const path = await field.getAttribute('data-path')
    expect(path, 'o campo não diz qual link veio do servidor').toBeTruthy()
    await expect(field).toHaveValue(new URL(path ?? '', page.url()).href)
  })
})
