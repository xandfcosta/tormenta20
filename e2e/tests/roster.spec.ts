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

  /**
   * ALE-98. The "+" used to be a mouse-only Link the arrows skipped: pressing →
   * on the last hero did nothing, and creation was unreachable without a
   * pointer. It is now a real cursor position — the stage shows an empty "?"
   * frame there and Enter opens the Forge.
   */
  test('a seta alcança o "+" e Enter leva para a criação', async ({ page }) => {
    await page.goto('/characters')
    await expect(page.getByRole('heading', { level: 2 })).not.toHaveText('')

    // Walk past the end of the roster; the cursor stops on the create slot.
    for (let i = 0; i < 20; i++) await page.keyboard.press('ArrowRight')

    await expect(page.getByRole('heading', { name: 'Novo personagem', level: 2 })).toBeVisible()
    await expect(page.getByRole('button', { name: /Criar novo personagem/ })).toBeVisible()

    await page.keyboard.press('Enter')
    await expect(page).toHaveURL(/\/characters\/new/)
  })

  test('← volta do "+" para o último herói do elenco', async ({ page }) => {
    await page.goto('/characters')
    await expect(page.getByRole('heading', { level: 2 })).not.toHaveText('')
    for (let i = 0; i < 20; i++) await page.keyboard.press('ArrowRight')
    await expect(page.getByRole('heading', { name: 'Novo personagem', level: 2 })).toBeVisible()

    // The peek on the left names the hero we came from — pressing ← returns to him.
    const voltandoPara = await page
      .getByRole('button', { name: /^Anterior:/ })
      .getAttribute('aria-label')
    await page.keyboard.press('ArrowLeft')

    await expect(page.getByRole('heading', { name: 'Novo personagem', level: 2 })).toBeHidden()
    expect(voltandoPara).toContain(await textOf(page.getByRole('heading', { level: 2 }), 'herói'))
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

  /**
   * ALE-97, the mirror of the test above: the scene must stay put, but the
   * STAGE must still animate. Those two came apart once — fixing the scene
   * flash left the roster switching characters in total silence, because the
   * stage's `animate-in` only ever replayed as a side effect of the bug.
   * Asserted together so neither fix can quietly undo the other.
   */
  test('o palco anima na troca, e só ele', async ({ page }) => {
    await page.goto('/characters')
    const spotlight = page.getByRole('heading', { level: 2 })
    const antes = await textOf(spotlight, 'spotlight inicial')

    await page.evaluate(() => {
      const fired: string[] = []
      ;(window as unknown as { __fired: string[] }).__fired = fired
      document.addEventListener(
        'animationstart',
        (e) => fired.push(`${e.animationName}|${(e.target as HTMLElement).className}`),
        true,
      )
    })

    await page.keyboard.press('ArrowRight')
    // Judge the animations only after the swap actually happened — a fixed
    // sleep here made the test flaky when the arrow landed before the roster
    // finished settling, and blamed the animation for a missed keypress.
    await expect(spotlight).not.toHaveText(antes)
    await page.waitForTimeout(300)

    const fired = await page.evaluate(() => (window as unknown as { __fired: string[] }).__fired)
    expect(fired.some((f) => f.includes('zoom-in-95')), 'o retrato não animou').toBe(true)
    expect(fired.some((f) => f.startsWith('scene-in')), 'a cena inteira reanimou').toBe(false)
  })

  /**
   * The peek label used to be `opacity-0` + `group-hover:opacity-100`, so the
   * name only ever showed under a mouse — invisible on touch and under keyboard
   * navigation, where two initials don't say who's next. jsdom applies no CSS,
   * so only a real browser can hold this: assert the computed opacity WITHOUT
   * hovering anything.
   */
  test('os dois peeks mostram o nome sem precisar de hover', async ({ page }) => {
    await page.goto('/characters')
    // Wait for the roster: an arrow pressed over an empty list is a no-op.
    await expect(page.getByRole('heading', { level: 2 })).not.toHaveText('')
    // Step in once so there is a peek on BOTH sides of the stage.
    await page.keyboard.press('ArrowRight')
    await expect(page.getByRole('button', { name: /^Anterior:/ })).toBeVisible()

    for (const side of [/^Anterior:/, /^Próximo:/]) {
      const peek = page.getByRole('button', { name: side })
      const label = peek.locator('span').last()
      await expect(label).not.toHaveText('')
      await expect(label).toHaveCSS('opacity', '1')
    }
  })
})

/**
 * The create slot (ALE-98) is a new stage, and the house rule is that any new
 * screen is validated at all six form factors. Reaching it needs the keyboard,
 * which only answers at ≥xl, so below that the slot is reached by URL-free
 * means: the filmstrip "+" is a plain Link everywhere, and what's checked here
 * is that the slot's stage itself never forces the page sideways.
 */
const VIEWPORTS = [
  { name: 'desktop', width: 1920, height: 1080 },
  { name: 'laptop', width: 1440, height: 900 },
  { name: 'tablet-landscape', width: 1024, height: 768 },
  { name: 'tablet-portrait', width: 768, height: 1024 },
  { name: 'mobile-landscape', width: 844, height: 390 },
  { name: 'mobile-portrait', width: 390, height: 844 },
]

test.describe('Roster — responsivo (sem overflow horizontal)', () => {
  for (const vp of VIEWPORTS) {
    test(`elenco: sem scroll horizontal @ ${vp.name} (${vp.width}×${vp.height})`, async ({
      page,
    }) => {
      await page.setViewportSize({ width: vp.width, height: vp.height })
      await page.goto('/characters')
      await expect(page.getByRole('heading', { level: 2 })).not.toHaveText('')

      const overflow = await page.evaluate(
        () => document.documentElement.scrollWidth - document.documentElement.clientWidth,
      )
      expect(overflow, 'a página não deve rolar horizontalmente').toBeLessThanOrEqual(1)
    })
  }

  // The slot only answers arrows on a desk setup (≥xl + pointer: fine).
  for (const vp of VIEWPORTS.filter((v) => v.width >= 1280)) {
    test(`slot de criação: sem scroll horizontal @ ${vp.name}`, async ({ page }) => {
      await page.setViewportSize({ width: vp.width, height: vp.height })
      await page.goto('/characters')
      await expect(page.getByRole('heading', { level: 2 })).not.toHaveText('')
      for (let i = 0; i < 20; i++) await page.keyboard.press('ArrowRight')
      await expect(page.getByRole('heading', { name: 'Novo personagem', level: 2 })).toBeVisible()

      const overflow = await page.evaluate(
        () => document.documentElement.scrollWidth - document.documentElement.clientWidth,
      )
      expect(overflow, 'a página não deve rolar horizontalmente').toBeLessThanOrEqual(1)
    })
  }
})
