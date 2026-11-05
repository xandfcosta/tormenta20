import { expect, test } from '@playwright/test'

// O mestre cria a melhoria da mesa e o jogador a coloca no item, na mesma lista das do livro. E2e
// porque é fluxo entre dois clientes, e o interruptor do diálogo mexe num sinal que vira a lista gravada.
test('o mestre cria a melhoria, o jogador a coloca no item e o mestre a exclui', async ({ browser }) => {
  test.setTimeout(90_000)
  const gm = await browser.newContext({ storageState: '.auth/user.json' })
  const player = await browser.newContext({ storageState: '.auth/player.json' })
  try {
    const gmScreen = await gm.newPage()
    await gmScreen.goto('/campanhas/1?tab=regras')
    await gmScreen.getByRole('button', { name: 'Nova melhoria' }).click()
    const editor = gmScreen.locator('dialog[open]')
    await expect(editor.getByLabel('Nome')).toBeVisible()
    await editor.getByLabel('Nome').fill('Insígnia de teste')
    await editor.getByRole('button', { name: 'Vestuário e esotérico' }).click()
    await expect(editor.getByText('num item de vestuário ou esotérico')).toBeVisible()
    await expect(editor.getByText('+1 em Luta')).toBeVisible()
    await editor.getByRole('button', { name: 'Gravar' }).click()
    const row = gmScreen.locator('.ds-table-rule', { hasText: 'Insígnia de teste' })
    await expect(row).toContainText('cabe em vestuário e esotérico')
    await expect(row).toContainText('em nenhum item')

    const playerScreen = await player.newPage()
    await playerScreen.goto('/personagens/13?tab=bag')
    await playerScreen.getByRole('button', { name: /Bandana/ }).first().click()
    await playerScreen.getByRole('button', { name: 'Melhorias de Bandana' }).click()
    const dialog = playerScreen.locator('dialog[open]', { hasText: 'Melhorias de Bandana' })
    await expect(dialog.getByText('da mesa')).toBeVisible()
    await dialog.getByRole('switch', { name: 'Insígnia de teste' }).click()
    await dialog.getByRole('button', { name: 'Aplicar' }).click()
    await expect(playerScreen.getByText('Insígnia de teste').first()).toBeVisible()

    await gmScreen.reload()
    await expect(gmScreen.locator('.ds-table-rule', { hasText: 'Insígnia de teste' })).toContainText('em 1 item')

    await gmScreen.locator('.ds-table-rule', { hasText: 'Insígnia de teste' }).getByRole('button', { name: 'Editar' }).click()
    await expect(gmScreen.locator('dialog[open]').getByLabel('Nome')).toHaveValue('Insígnia de teste')
    await gmScreen.locator('dialog[open]').getByRole('button', { name: 'Excluir' }).click()
    const confirm = gmScreen.getByRole('dialog', { name: 'Excluir esta melhoria?' })
    await expect(confirm).toContainText('em 1 item')
    await confirm.getByRole('button', { name: 'Excluir' }).click()
    await expect(gmScreen.locator('.ds-table-rule', { hasText: 'Insígnia de teste' })).toHaveCount(0)

    await playerScreen.reload()
    await expect(playerScreen.getByText('Insígnia de teste')).toHaveCount(0)
  } finally {
    await gm.close().catch(() => {})
    await player.close().catch(() => {})
  }
})
