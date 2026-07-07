import { expect, test, type Page } from '@playwright/test'

/**
 * Visual baseline for the pre-restyle state. Any pixel drift beyond
 * `maxDiffPixelRatio` (see playwright.config.ts) fails the run and
 * the reviewer must approve the new baseline explicitly.
 *
 * These are unauthenticated routes only — the character sheet /
 * session tracker snapshots need a seeded backend and land in a
 * follow-up.
 */

const PUBLIC_ROUTES = [
  { name: 'landing', path: '/' },
  { name: 'login', path: '/login' },
  { name: 'register', path: '/register' },
] as const

/**
 * Toggle html.dark by mutating localStorage before load — matches the
 * `useUiStore` persistence key so the app renders the requested theme
 * on first paint.
 */
async function applyTheme(page: Page, theme: 'light' | 'dark') {
  /* Key must match `THEME_STORAGE_KEY` in src/store/ui-store.ts —
   * zustand-persist reads on hydration, before first paint. */
  await page.addInitScript((t) => {
    localStorage.setItem(
      't20-ui',
      JSON.stringify({ state: { theme: t }, version: 0 }),
    )
  }, theme)
}

/** Wait for webfonts so we don't snapshot mid-swap. */
async function waitForFonts(page: Page) {
  await page.evaluate(() => document.fonts.ready)
}

for (const route of PUBLIC_ROUTES) {
  for (const theme of ['light', 'dark'] as const) {
    test(`${route.name} — ${theme}`, async ({ page }) => {
      await applyTheme(page, theme)
      await page.goto(route.path)
      await waitForFonts(page)
      await expect(page).toHaveScreenshot(`${route.name}-${theme}.png`, {
        fullPage: true,
      })
    })
  }
}
