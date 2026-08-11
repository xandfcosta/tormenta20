import { expect, test } from '@playwright/test'

test.describe('Hub → cenas', () => {
  test('o Hub abre autenticado com o menu principal', async ({ page }) => {
    await page.goto('/')
    await expect(page.getByRole('heading', { name: /Tormenta 20/i })).toBeVisible()
    await expect(page.getByText('Meus Heróis')).toBeVisible()
    await expect(page.getByText('Crônicas')).toBeVisible()
    await expect(page.getByText('Ferramentas do Mestre')).toBeVisible()
  })

  test('Hub → Personagens (roster)', async ({ page }) => {
    await page.goto('/')
    await page.getByText('Meus Heróis').click()
    await expect(page).toHaveURL(/\/characters/)
    await expect(page.getByText('Personagens')).toBeVisible()
  })

  test('Hub → Crônicas (lista de campanhas)', async ({ page }) => {
    await page.goto('/')
    await page.getByText('Crônicas').click()
    await expect(page).toHaveURL(/\/campaigns/)
  })
})
