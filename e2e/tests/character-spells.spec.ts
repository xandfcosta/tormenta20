import { expect, test } from '@playwright/test'
import { openSheetFromRoster } from './support/roster'

const CASTER = 'Necromante Nv12 Magias'

/**
 * The Grimório block (ALE-88).
 *
 * It carries TWO virtualized lists — the learned spellbook and the whole-catalog
 * learn dialog — and jsdom measures every element as 0, so a unit test sees an
 * empty list and passes green either way. Only a real browser proves a row
 * renders and that the filters narrow to the right one.
 *
 * Read-only: it opens dialogs and filters, never writes, so the seed is left as
 * found (F.I.R.S.T — repeatable).
 *
 * O diálogo de conjurar saiu na ALE-144: a soma dos aprimoramentos é ARITMÉTICA
 * e mora em `features/character-sheet/spell-augments.test.ts`, que cobre oito
 * casos onde aqui cabia um.
 */
test.describe('Grimório', () => {
  test('lista as magias aprendidas com custo e CD', async ({ page }) => {
    await page.goto('/piloto/personagens')
    await openSheetFromRoster(page, CASTER)
    await page.getByRole('tab', { name: 'Magias' }).click()

    await expect(page.getByRole('heading', { name: 'Grimório' })).toBeVisible()
    // The rows only exist if the virtualized list actually measured and painted.
    await expect(page.getByText('Bola de Fogo')).toBeVisible()
    await expect(page.getByRole('button', { name: 'Conjurar Bola de Fogo' })).toBeVisible()
  })

  test('o catálogo filtra por nome e por círculo', async ({ page }) => {
    await page.goto('/piloto/personagens')
    await openSheetFromRoster(page, CASTER)
    await page.getByRole('tab', { name: 'Magias' }).click()

    await page.getByRole('button', { name: 'Aprender' }).click()
    // Scoped to the dialog: the grimoire behind it lists the same spells, and
    // the modal marks that copy aria-hidden.
    const dialog = page.getByRole('dialog')
    const search = dialog.getByRole('searchbox', { name: 'Buscar magia' })
    await expect(search).toBeVisible()

    await search.fill('bola')
    await expect(dialog.getByText('Bola de Fogo')).toBeVisible()
    // A row from another part of the catalog must NOT survive the filter — the
    // failure mode here is a virtualized list keeping whatever painted first.
    await expect(dialog.getByText('Teletransporte')).toHaveCount(0)

    await search.fill('')
    await dialog.getByRole('combobox', { name: 'Círculo' }).selectOption('5')
    await expect(dialog.getByText('Bola de Fogo')).toHaveCount(0)
  })

})
