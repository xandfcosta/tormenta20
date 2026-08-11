import { type Locator, type Page, expect, test } from '@playwright/test'
import { openSheetFromRoster } from './support/roster'

// A hero no other spec asserts on, so the vitals edit can't disturb them.
const HERO = 'Necromante Nv12 Magias'

/**
 * Waits for a vitals edit to actually reach the API. The two directions take
 * different routes — reduzir goes through the damage pipeline (temp HP pool),
 * curar patches the vitals directly — so match either.
 */
function vitalsWrite(page: Page) {
  return page.waitForResponse(
    (res) =>
      /\/api\/characters\/\d+\/(damage|vitals)$/.test(res.url()) &&
      res.request().method() !== 'GET' &&
      res.ok(),
  )
}

async function currentHp(vida: Locator): Promise<number> {
  const value = await vida.getAttribute('aria-valuenow')
  if (value === null) throw new Error('barra de Vida sem aria-valuenow (esperado um inteiro)')
  return Number(value)
}

/**
 * Login → Hub → abrir personagem → editar bloco da ficha (vitals).
 *
 * Reads the current PV instead of hardcoding it and puts it back at the end:
 * the edit is a real server mutation, so the test must leave the seed as it
 * found it (F.I.R.S.T — repeatable).
 */
test('Hub → herói → editar Vida no bloco de vitals (persiste no servidor)', async ({ page }) => {
  await page.goto('/')
  await page.getByText('Meus Heróis').click()
  await expect(page).toHaveURL(/\/characters$/)
  await openSheetFromRoster(page, HERO)
  await expect(page).toHaveURL(/\/characters\/\d+$/)

  const vida = page.getByRole('progressbar', { name: 'Vida' })
  const before = await currentHp(vida)

  const decremented = vitalsWrite(page)
  await page.getByRole('button', { name: /^Reduzir Vida/ }).click()
  await expect(vida).toHaveAttribute('aria-valuenow', String(before - 1))
  await decremented

  // Reload proves the write reached the API — not just the optimistic cache.
  await page.reload()
  await expect(vida).toHaveAttribute('aria-valuenow', String(before - 1))

  const restored = vitalsWrite(page)
  await page.getByRole('button', { name: /^Aumentar Vida/ }).click()
  await expect(vida).toHaveAttribute('aria-valuenow', String(before))
  await restored
})
