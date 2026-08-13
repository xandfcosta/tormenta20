import { expect, test } from '@playwright/test'
import { openSheetFromRoster } from './support/roster'

const HERO = 'Tanque Placas Nv10'

/**
 * The Efeitos block (ALE-86).
 *
 * Two things here no unit test can answer:
 *
 *  1. A book condition must move the NUMBERS, not just add a badge — that was
 *     ALE-28. The assertion reads the HUD's Ataque tile before and after, which
 *     only lines up when the engine, the store and the HUD agree.
 *  2. "Encerrar cena" reaches `POST /characters/:id/end-scene`, a route that did
 *     not exist in the Go backend until this issue — the React front had been
 *     calling it into a 404. Running on BOTH fronts is the point: it pins the
 *     repair for the app being replaced as well as the one replacing it.
 *
 * Writes and undoes its own writes, so the seed is left as found (F.I.R.S.T).
 */
// Serial: both tests write to the SAME seed character. Run in parallel, ending
// a scene mid-way through the condition test flips the numbers under it.
test.describe.configure({ mode: 'serial' })

test.describe('Efeitos', () => {
  test('condição do livro muda o ataque e sai limpo', async ({ page }) => {
    await page.goto('/characters')
    await openSheetFromRoster(page, HERO)
    await page.getByRole('tab', { name: 'Efeitos' }).click()

    // The Solid HUD (ALE-90) names its combat tiles "<label> <valor>"; the React
    // HUD it replaces labels them another way and has no equivalent hook. The
    // numeric claim is about the block just built, so it runs where it can be
    // asserted rather than being weakened into "a chip appeared".
    const attack = (value: string) => page.getByRole('button', { name: `Atq CaC ${value}` })
    const anyAttack = page.getByRole('button', { name: /^Atq CaC / })
    const picker = page.getByRole('combobox', { name: 'Aplicar condição' })
    // Wait for the block to mount before counting: `count()` does not auto-wait,
    // and asking too early reports 0 on the front that DOES have the control.
    await expect(page.getByRole('heading', { name: 'Efeitos ativos' })).toBeVisible()
    // The React front names its condition picker after the placeholder text and
    // opens a command palette instead of a listbox. Rather than carry two sets
    // of locators for an app being deleted, this one runs where the control it
    // needs exists; the end-scene test below is the one that must pass on BOTH.
    test.skip(
      (await picker.count()) === 0 || (await anyAttack.count()) === 0,
      'front sem o picker/tile nomeados como no Solid — asserção é do bloco portado',
    )

    // Self-healing setup: a run that died mid-test left Caído applied, and the
    // picker hides conditions that are already on — the next run would then look
    // for an option that cannot be there (F.I.R.S.T — repeatable).
    const leftover = page.getByRole('button', { name: 'Remover condição Caído' })
    if (await leftover.count()) await leftover.click()

    // Read the CURRENT bonus rather than hardcoding the seed's +13.
    await expect(anyAttack).toBeVisible()
    const before = Number((await anyAttack.getAttribute('aria-label'))?.replace(/\D+/g, ''))

    await picker.click()
    await page.getByRole('option', { name: 'Caído' }).click()

    // Caído is −5 em Luta: a badge alone would leave the tile untouched (ALE-28).
    await expect(attack(`+${before - 5}`)).toBeVisible()

    await page.getByRole('button', { name: 'Remover condição Caído' }).click()
    await expect(attack(`+${before}`)).toBeVisible()
  })

  test('encerrar cena limpa os efeitos de cena', async ({ page }) => {
    await page.goto('/characters')
    await openSheetFromRoster(page, HERO)
    await page.getByRole('tab', { name: 'Efeitos' }).click()

    await page.getByRole('button', { name: 'Aplicar efeito' }).click()
    const search = page.getByPlaceholder(/Buscar magia/)
    await search.fill('escudo')
    await page.getByRole('button', { name: /^Escudo da Fé/ }).click()

    // Esperar o diálogo FECHAR antes de olhar a lista. Ele mostra o mesmo nome
    // da magia, então um `getByText` solto corria contra o fechamento e, numa
    // máquina lenta, casava dois elementos — o strict mode derrubava a suíte.
    // Escopar sozinho não resolveria: enquanto o diálogo está aberto ele marca
    // os irmãos `aria-hidden`, e aí a lista some da árvore de acessibilidade e
    // um locator por ROLE não acha nada.
    await expect(page.getByRole('dialog')).toBeHidden()
    const applied = page
      .getByRole('list', { name: 'Efeitos ativos' })
      .getByText('Escudo da Fé')
    await expect(applied).toBeVisible()

    // The trigger and the confirm share a label — scope to the dialog.
    await page.getByRole('button', { name: 'Encerrar cena' }).click()
    const confirm = page.getByRole('dialog').filter({ hasText: 'Encerrar cena?' })
    await confirm.getByRole('button', { name: 'Encerrar cena' }).click()

    // A 404 (the state before this issue) would leave the row on screen.
    await expect(applied).toBeHidden()
    await expect(page.getByText(/Nenhum consumível ativo/)).toBeVisible()
  })
})
