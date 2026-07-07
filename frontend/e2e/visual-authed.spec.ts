import { expect, test, type Page } from '@playwright/test'

/**
 * Authenticated visual baseline. Depends on the `setup` project +
 * the backend seed (deterministic ids: character 1, campaign 1,
 * session 1).
 *
 * The character sheet route (`/characters/1`) is the biggest
 * regression surface — it renders ~150 data points across attrs,
 * skills, spells, inventory. Snapshotting it here is the whole point
 * of investing in seed infrastructure.
 */

const AUTHED_ROUTES = [
  { name: 'characters-index', path: '/characters' },
  { name: 'character-sheet', path: '/characters/1' },
  { name: 'campaigns-index', path: '/campaigns' },
  { name: 'campaign-detail', path: '/campaigns/1' },
  { name: 'session-tracker', path: '/campaigns/1/sessions/1' },
] as const

async function applyTheme(page: Page, theme: 'light' | 'dark') {
  await page.addInitScript((t) => {
    localStorage.setItem(
      't20-ui',
      JSON.stringify({ state: { theme: t }, version: 0 }),
    )
  }, theme)
}

async function waitForFonts(page: Page) {
  await page.evaluate(() => document.fonts.ready)
}

for (const route of AUTHED_ROUTES) {
  for (const theme of ['light', 'dark'] as const) {
    test(`${route.name} — ${theme}`, async ({ page }) => {
      await applyTheme(page, theme)
      await page.goto(route.path)
      /* Give React Query loaders a beat to hydrate — the character
       * sheet fires several parallel requests and we don't want to
       * snapshot mid-suspense. */
      await page.waitForLoadState('networkidle', { timeout: 20_000 })
      await waitForFonts(page)
      await expect(page).toHaveScreenshot(`${route.name}-${theme}.png`, {
        fullPage: true,
      })
    })
  }
}
