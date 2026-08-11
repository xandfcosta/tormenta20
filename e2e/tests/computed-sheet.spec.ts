import { expect, test } from '@playwright/test'
import { openSheetFromRoster } from './support/roster'

const HERO = 'Tanque Placas Nv10'

/**
 * The read-only computed sheet (`/characters/:id/sheet`).
 *
 * This spec exists because the screen shipped BROKEN and nothing caught it
 * (ALE-77): it read `data.computed` from an endpoint the Go backend answers
 * with a different payload, so the page threw and fell into the error
 * boundary. The unit tests all mock the HTTP client, so none of them could
 * see it — only a real browser against the real backend can.
 */
test.describe('Ficha computada (somente leitura)', () => {
  test('abre pelo roster e mostra os blocos derivados', async ({ page }) => {
    await page.goto('/characters')
    await openSheetFromRoster(page, HERO)
    await expect(page).toHaveURL(/\/characters\/\d+$/)

    await page.goto(`${page.url()}/sheet`)

    // The heading proves the page rendered rather than the error boundary.
    await expect(page.getByRole('heading', { name: HERO })).toBeVisible()

    // Anchored to the card titles, not loose text: the dev overlay renders its
    // own nodes containing these words, and a bare getByText matched both.
    const blocks = page.locator('[data-slot="card-title"]')
    for (const title of ['Atributos', 'Defesa', 'Resistências']) {
      await expect(blocks.filter({ hasText: title })).toBeVisible()
    }
  })

  test('não cai no error boundary', async ({ page }) => {
    const crashes: string[] = []
    page.on('pageerror', (error) => crashes.push(error.message))

    await page.goto('/characters/1/sheet')
    await expect(page.getByRole('heading', { name: HERO })).toBeVisible()

    expect(crashes, `erros de página: ${crashes.join(' | ')}`).toEqual([])
  })
})
