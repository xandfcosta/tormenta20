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

/**
 * ALE-95. Reading a pending `useQuery().data` suspends, the nearest boundary is
 * the router's own per-match `Suspense`, and Solid resolves that by detaching
 * and re-inserting the WHOLE match subtree — which restarts every CSS animation
 * under it, so the entire scene re-plays its enter animation on each arrow key.
 *
 * Nothing remounts (same nodes, same classes), so no DOM or a11y assertion can
 * see it: the only trace is `scene-in`'s `startTime` moving. Hence this test
 * watches the animation itself. jsdom has no animation timeline at all, which
 * is why this lives in e2e and not in vitest.
 *
 * It must arrow FORWARD into characters not yet visited — going back is a cache
 * hit, which never suspends and would pass over a broken app.
 */
test.describe('Roster — a cena não reanima ao trocar de personagem', () => {
  test.use({ viewport: DESK })

  const sceneInStart = (page: import('@playwright/test').Page) =>
    page.evaluate(() => {
      const scene = document.querySelector('[data-slot="scene-content"]')
      if (!scene) throw new Error('cena não encontrada: [data-slot="scene-content"]')
      const enter = scene.getAnimations().find((a) => 'animationName' in a && a.animationName === 'scene-in')
      return enter ? Math.round(enter.startTime as number) : null
    })

  test('a animação de entrada da cena não reinicia a cada troca', async ({ page }) => {
    await page.goto('/characters')
    const spotlight = page.getByRole('heading', { level: 2 })
    await expect(spotlight).not.toHaveText('')

    const atEntry = await sceneInStart(page)
    expect(atEntry, 'a cena deve ter tocado sua animação de entrada ao montar').not.toBeNull()

    for (let step = 0; step < 3; step++) {
      const before = await textOf(spotlight, `spotlight antes do passo ${step}`)
      await page.keyboard.press('ArrowRight')
      await expect(spotlight).not.toHaveText(before)
      // Wide enough to cover the sheet fetch that used to trigger the suspend.
      await page.waitForTimeout(500)
      expect(await sceneInStart(page), `a cena reanimou no passo ${step}`).toBe(atEntry)
    }
  })
})
