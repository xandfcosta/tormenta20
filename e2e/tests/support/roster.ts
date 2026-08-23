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
 * Desde a ALE-239 a cena é do SERVIDOR (`/piloto/personagens`) e o retrato é um
 * LINK, não um botão — daí o `getByRole('link')`. A espera passou a ser pela
 * visibilidade dele em vez de `toHaveCount(1)` nas opções: a cena desenha todos
 * os palcos e esconde os que não estão no cursor, e o trilho tem uma opção a
 * mais que o elenco, que é a vaga de criar. Contar opções contaria essa vaga.
 *
 * @example await openSheetFromRoster(page, 'Necromante Nv12 Magias')
 */
export async function openSheetFromRoster(page: Page, hero: string): Promise<void> {
  await page.getByRole('searchbox', { name: 'Buscar personagem' }).fill(hero)
  const abrir = page.getByRole('link', { name: `Abrir ficha de ${hero}` })
  await expect(abrir).toBeVisible()
  await abrir.click()
}
