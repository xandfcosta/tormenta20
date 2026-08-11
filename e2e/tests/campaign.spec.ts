import { expect, test } from '@playwright/test'

const CAMPAIGN = '/campaigns/1' // Snapshot Test ALE-33 (seed)

test.describe('Detalhe da campanha', () => {
  test('troca de aba mostra o roster de membros', async ({ page }) => {
    await page.goto(`${CAMPAIGN}?tab=visao`)
    await expect(
      page.getByRole('heading', { name: /Snapshot Test ALE-33/i }),
    ).toBeVisible()

    await page.getByRole('tab', { name: 'Membros' }).click()
    await expect(page).toHaveURL(/tab=membros/)
    await expect(page.getByText('Tanque Placas Nv10')).toBeVisible()
    // Party roster from the rich seed (1 GM + 4 players).
    await expect(page.getByText('Recruta Nv1 Simples')).toBeVisible()
  })
})

/**
 * The GM's write block (ALE-79). Every case stops SHORT of a destructive write:
 * the seed is shared with the other specs, so editing is opened and cancelled,
 * and deleting only gets as far as the confirmation.
 */
test.describe('Escrita do mestre', () => {
  test('a aba Config traz o ledger e a zona de perigo', async ({ page }) => {
    await page.goto(`${CAMPAIGN}?tab=config`)

    await expect(page.getByText('Zona de perigo')).toBeVisible()
    await expect(
      page.getByRole('button', { name: /Excluir campanha/ }),
    ).toBeVisible()
  })

  test('editar abre o formulário com os valores atuais', async ({ page }) => {
    await page.goto(`${CAMPAIGN}?tab=config`)

    await page.getByRole('button', { name: 'Editar' }).click()

    await expect(page.getByLabel('Nome')).toHaveValue(/Snapshot Test ALE-33/)
    await expect(page.getByLabel('Descrição')).toBeVisible()
    await page.getByRole('button', { name: 'Cancelar' }).click()
    await expect(page.getByLabel('Nome')).toBeHidden()
  })

  test('excluir exige confirmação antes de qualquer coisa', async ({ page }) => {
    await page.goto(`${CAMPAIGN}?tab=config`)

    await page.getByRole('button', { name: /Excluir campanha/ }).click()

    const dialog = page.getByRole('dialog')
    await expect(dialog).toContainText('não pode ser desfeita')
    await dialog.getByRole('button', { name: 'Cancelar' }).click()
    await expect(page).toHaveURL(/tab=config/)
  })

  test('o convite gera um link de entrar na mesa', async ({ page }) => {
    await page.goto(`${CAMPAIGN}?tab=membros`)

    await page.getByRole('button', { name: /Convite/ }).click()
    await page.getByRole('button', { name: 'Gerar link' }).click()

    await expect(page.getByLabel('Link de convite')).toHaveValue(/\/join\/.+/)
  })
})

// The scene must FILL the width at every form factor — no horizontal body
// scroll. This is the deterministic version of the manual 6-resolution pass.
const VIEWPORTS = [
  { name: 'desktop', width: 1920, height: 1080 },
  { name: 'laptop', width: 1440, height: 900 },
  { name: 'tablet-landscape', width: 1024, height: 768 },
  { name: 'tablet-portrait', width: 768, height: 1024 },
  { name: 'mobile-landscape', width: 844, height: 390 },
  { name: 'mobile-portrait', width: 390, height: 844 },
]

test.describe('Campanha — responsivo (preenche a tela, sem overflow horizontal)', () => {
  for (const vp of VIEWPORTS) {
    test(`sem scroll horizontal @ ${vp.name} (${vp.width}×${vp.height})`, async ({
      page,
    }) => {
      await page.setViewportSize({ width: vp.width, height: vp.height })
      await page.goto(`${CAMPAIGN}?tab=membros`)
      await expect(
        page.getByRole('heading', { name: /Snapshot Test ALE-33/i }),
      ).toBeVisible()
      const overflow = await page.evaluate(
        () =>
          document.documentElement.scrollWidth -
          document.documentElement.clientWidth,
      )
      expect(overflow, 'a página não deve rolar horizontalmente').toBeLessThanOrEqual(1)
    })
  }
})
