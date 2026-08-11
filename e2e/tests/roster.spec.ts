import { expect, test } from '@playwright/test'
import { textOf } from './support/roster'

// The scene-nav keyboard layer is gated to `≥xl + pointer: fine` (a desk setup,
// see frontend/CLAUDE.md), so the spotlight only answers arrows above 1280px.
// Pin the viewport instead of inheriting whatever the project default is.
const DESK = { width: 1440, height: 900 }

/**
 * Nothing here hardcodes a hero: the rail sorts by last-updated, so the seed
 * order is reshuffled by any spec that edits vitals. Expectations are derived
 * from the list as rendered — what's asserted is the *movement*, not the cast.
 */
test.describe('Roster — navegação por teclado', () => {
  test.use({ viewport: DESK })

  test('setas movem o spotlight pelo elenco e voltam', async ({ page }) => {
    await page.goto('/characters')
    const spotlight = page.getByRole('heading', { level: 2 })
    const first = await textOf(spotlight, 'spotlight inicial')
    const nextInRail = await textOf(page.getByRole('option').nth(1), 'segundo item do rail')

    await page.keyboard.press('ArrowRight')
    await expect(spotlight).not.toHaveText(first)
    // The rail mirrors the spotlight as the selected option (initials).
    await expect(page.getByRole('option', { selected: true })).toHaveAccessibleName(nextInRail)

    await page.keyboard.press('ArrowLeft')
    await expect(spotlight).toHaveText(first)
  })

  test('Enter abre a ficha do personagem em spotlight', async ({ page }) => {
    await page.goto('/characters')
    const spotlight = page.getByRole('heading', { level: 2 })
    const first = await textOf(spotlight, 'spotlight inicial')

    await page.keyboard.press('ArrowRight')
    await expect(spotlight).not.toHaveText(first)
    const moved = await textOf(spotlight, 'spotlight após ArrowRight')

    await page.keyboard.press('Enter')
    await expect(page).toHaveURL(/\/characters\/\d+$/)
    await expect(page.getByRole('heading', { name: moved, level: 1 }).first()).toBeVisible()
  })
})
