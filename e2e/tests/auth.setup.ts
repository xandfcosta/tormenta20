import { expect, test as setup } from '@playwright/test'

// Seeded GM account from the Go backend (engine-go/t20-go.db).
const EMAIL = process.env.E2E_EMAIL ?? 'mestre@t20.local'
const PASSWORD = process.env.E2E_PASSWORD ?? 'mestre123456'
const AUTH_FILE = '.auth/user.json'

/**
 * Logs in through the real UI and persists the session (localStorage token +
 * cookies) so every other spec starts on an authenticated context — the
 * framework-agnostic way to handle auth (survives the Solid migration).
 */
setup('authenticate', async ({ page }) => {
  await page.goto('/login')
  await page.getByLabel('E-mail').fill(EMAIL)
  await page.getByLabel('Senha').fill(PASSWORD)
  await page.getByRole('button', { name: 'Entrar' }).click()

  // Landed on the Hub (the game's main menu).
  await expect(page.getByText('Meus Heróis')).toBeVisible()
  await page.context().storageState({ path: AUTH_FILE })
})
