import { expect, test } from '@playwright/test'
import { openSheetFromRoster } from './support/roster'

const HERO = 'Tanque Placas Nv10'

/**
 * The Mochila block (ALE-84).
 *
 * This spec exists because the catálogo dialog shipped BROKEN in the Solid port
 * and no unit test could see it: jsdom gives every element a zero-height rect,
 * so the virtualized list renders NO row there — while a real browser rendered
 * rows, measured them by an attribute that was not set yet, resolved index -1
 * and took the whole scene down. The second bug was the same list keeping the
 * rows it painted first, so searching "escudo" listed daggers.
 *
 * Read-only on purpose: it opens dialogs and filters, and never writes, so it
 * leaves the seed exactly as it found it (F.I.R.S.T — repeatable).
 */
test.describe('Mochila', () => {
  test('abre o catálogo e filtra pelo nome sem derrubar a cena', async ({ page }) => {
    await page.goto('/characters')
    await openSheetFromRoster(page, HERO)
    await expect(page).toHaveURL(/\/characters\/\d+$/)

    await page.getByRole('tab', { name: 'Mochila' }).click()
    await expect(page.getByRole('heading', { name: 'Mochila', exact: true })).toBeVisible()

    await page.getByRole('button', { name: 'Adicionar do catálogo' }).click()
    const search = page.getByPlaceholder(/Buscar pelo nome/)
    await expect(search).toBeVisible()

    // Before the fix this listed whatever the rows painted first (Adaga,
    // Espada curta, Foice) no matter what was typed.
    await search.fill('escudo')
    await expect(page.getByRole('button', { name: /^Escudo leve/ })).toBeVisible()
    await expect(page.getByRole('button', { name: /^Adaga/ })).toHaveCount(0)

    await page.getByRole('button', { name: 'Cancelar' }).click()
    await expect(search).toBeHidden()
  })

  test('a ficha do item abre pelo tile guardado', async ({ page }) => {
    await page.goto('/characters')
    await openSheetFromRoster(page, HERO)
    await page.getByRole('tab', { name: 'Mochila' }).click()

    await page.getByRole('button', { name: 'Abrir Bálsamo restaurador' }).click()
    const sheet = page.getByRole('dialog').filter({ hasText: 'Bálsamo restaurador' })
    await expect(sheet).toBeVisible()
    // A consumível offers "Usar" and no equip slot to move into.
    await expect(sheet.getByRole('button', { name: 'Usar' })).toBeVisible()
    await expect(sheet.getByRole('button', { name: /Empunhar/ })).toHaveCount(0)
  })
})
