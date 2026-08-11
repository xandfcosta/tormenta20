import { type Page, expect, test as setup } from '@playwright/test'

// Seeded accounts from the Go backend (engine-go/t20-go.db). The GM owns
// campaign 1; the player is only a member of it — the pair the role-gating
// specs need (ALE-24).
const PASSWORD = process.env.E2E_PASSWORD ?? 'mestre123456'
const GM_EMAIL = process.env.E2E_EMAIL ?? 'mestre@t20.local'
const PLAYER_EMAIL = process.env.E2E_PLAYER_EMAIL ?? 'jogador@t20.local'

/**
 * Logs in through the real UI and persists the session (localStorage token +
 * cookies) so the specs start on an authenticated context — the
 * framework-agnostic way to handle auth (survives the Solid migration).
 *
 * Done ONCE per role here rather than in a beforeEach: signing in inside every
 * test adds a full page load per test, and against the dev server that raced
 * Vite's dependency re-optimization reload and made the suite flaky.
 */
async function signIn(page: Page, email: string, file: string): Promise<void> {
  await page.goto('/login')
  await page.getByLabel('E-mail').fill(email)
  await page.getByLabel('Senha').fill(PASSWORD)
  await page.getByRole('button', { name: 'Entrar' }).click()

  // Landed on the Hub (the game's main menu).
  await expect(page.getByText('Meus Heróis')).toBeVisible()
  await page.context().storageState({ path: file })
}

setup('authenticate', async ({ page }) => {
  await signIn(page, GM_EMAIL, '.auth/user.json')
})

setup('authenticate as player', async ({ page }) => {
  await signIn(page, PLAYER_EMAIL, '.auth/player.json')
})
