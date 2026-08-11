import { type Locator, type Page, expect } from '@playwright/test'

/** Trimmed text of a locator, failing loudly instead of handing back `null`. */
export async function textOf(locator: Locator, what: string): Promise<string> {
  const text = await locator.textContent()
  if (text === null || text.trim() === '') {
    throw new Error(`${what}: esperado texto não-vazio, veio ${JSON.stringify(text)}`)
  }
  return text.trim()
}

/**
 * Opens a hero's sheet from the roster scene, by name.
 *
 * Goes through the scene's search box on purpose: the rail sorts by
 * last-updated, so ANY vitals edit reshuffles it — a spec that clicked "the
 * first card" would depend on which other spec ran before it.
 *
 * @example await openSheetFromRoster(page, 'Necromante Nv12 Magias')
 */
export async function openSheetFromRoster(page: Page, hero: string): Promise<void> {
  await page.getByRole('searchbox', { name: 'Buscar personagem' }).fill(hero)
  await expect(page.getByRole('option')).toHaveCount(1)
  await page.getByRole('button', { name: `Abrir ficha de ${hero}` }).click()
}
