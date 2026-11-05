import { expect, test } from '@playwright/test'

// O mestre monta a regra no editor guiado, liga e apaga. Os selects dependem uns dos outros, e
// é por isso que este caminho é medido no navegador.
test('o mestre monta, liga e apaga uma regra da mesa', async ({ page }) => {
  await page.goto('/campanhas/1?tab=regras')
  await page.getByRole('button', { name: 'Nova regra' }).click()
  const editor = page.locator('dialog[open]')
  await expect(editor.getByLabel('Nome')).toBeVisible()

  await editor.getByLabel('Nome').fill('Regra do teste')
  await editor.getByRole('button', { name: 'Condição' }).click()
  const kind = editor.getByLabel('Tipo da condição 1')
  await expect(kind).toBeVisible()
  await kind.selectOption({ label: 'Raça' })
  const value = editor.getByLabel('Qual, na condição 1')
  await expect(value.locator('option', { hasText: 'Minotauro' })).toHaveCount(1)
  await value.selectOption({ label: 'Minotauro' })
  await expect(editor.getByText('da raça Minotauro')).toBeVisible()

  await editor.getByLabel('Em quê').selectOption({ label: 'Defesa' })
  await editor.getByLabel('Quanto').fill('2')
  await editor.getByLabel('Quanto').blur()
  await expect(editor.getByText('da raça Minotauro: +2 na Defesa')).toBeVisible()
  await editor.getByRole('button', { name: 'Gravar' }).click()

  const row = page.locator('.ds-table-rule', { hasText: 'Regra do teste' })
  await expect(row).toContainText('+2 na Defesa')
  const toggle = row.getByRole('switch')
  await expect(toggle).not.toBeChecked()
  await toggle.click()
  await expect(page.locator('.ds-table-rule', { hasText: 'Regra do teste' }).getByRole('switch')).toBeChecked()

  await page.reload()
  await expect(page.locator('.ds-table-rule', { hasText: 'Regra do teste' }).getByRole('switch')).toBeChecked()

  await page.locator('.ds-table-rule', { hasText: 'Regra do teste' }).getByRole('button', { name: 'Editar' }).click()
  await expect(page.locator('dialog[open]').getByLabel('Nome')).toHaveValue('Regra do teste')
  // Excluir pede confirmação: a regra ligada está na ficha dos jogadores.
  await page.locator('dialog[open]').getByRole('button', { name: 'Excluir' }).click()
  const confirm = page.getByRole('dialog', { name: 'Excluir esta regra?' })
  await confirm.getByRole('button', { name: 'Cancelar' }).click()
  await expect(confirm).toBeHidden()
  await expect(page.locator('dialog[open]').getByLabel('Nome')).toHaveValue('Regra do teste')
  await page.locator('dialog[open]').getByRole('button', { name: 'Excluir' }).click()
  await confirm.getByRole('button', { name: 'Excluir' }).click()
  await expect(page.locator('.ds-table-rule', { hasText: 'Regra do teste' })).toHaveCount(0)
})

// A resposta de um rascunho pode chegar depois de a pessoa ter seguido escrevendo. Se ela trouxer de
// volta o que a pessoa digitou, o texto volta atrás sem aviso — e a bancada rápida nunca vê.
test('a resposta atrasada não desfaz o que se digitou depois', async ({ page }) => {
  await page.goto('/campanhas/1?tab=regras')
  await page.getByRole('button', { name: 'Nova regra' }).click()
  const editor = page.locator('dialog[open]')
  await expect(editor.getByLabel('Nome')).toBeVisible()

  let release: () => void = () => {}
  const held = new Promise<void>((resolve) => {
    release = resolve
  })
  let holding = true
  await page.route('**/regras-da-mesa/rascunho*', async (route) => {
    if (holding) {
      holding = false
      const reply = await route.fetch()
      await held
      await route.fulfill({ response: reply })
      return
    }
    await route.continue()
  })

  const answered = page.waitForResponse('**/regras-da-mesa/rascunho*')
  await editor.getByLabel('Em quê').selectOption({ label: 'Defesa' })
  await editor.getByLabel('Nome').fill('Escrita depois')
  await editor.getByLabel('Quanto').fill('7')
  release()
  await answered

  await expect(editor.getByLabel('Nome')).toHaveValue('Escrita depois')
  await expect(editor.getByLabel('Quanto')).toHaveValue('7')
  await editor.getByLabel('Quanto').blur()
  await expect(editor.getByText('+7 na Defesa')).toBeVisible()
})

// O mestre liga a regra e a "Minha ficha" aberta na mesa acompanha, sem o jogador recarregar. E2e
// porque é fluxo entre dois clientes: o aviso passa pelo stream da mesa e pelo sinal da versão.
test('ligar e excluir a regra chega na ficha aberta do jogador', async ({ browser }) => {
  test.setTimeout(90_000)
  const gm = await browser.newContext({ storageState: '.auth/user.json' })
  const player = await browser.newContext({ storageState: '.auth/player.json' })
  try {
    const playerScreen = await player.newPage()
    await playerScreen.goto('/campanhas/1/sessoes/5')
    await expect(playerScreen.getByRole('navigation', { name: 'Superfícies da sessão' })).toBeVisible()
    await playerScreen.getByRole('button', { name: 'Ficha', exact: true }).click()
    const sheet = playerScreen.getByRole('navigation', { name: 'Seções da ficha' })
    await sheet.getByRole('link', { name: 'Combate' }).click()
    const sheetHP = async () => {
      const text = await playerScreen.locator('#sheet-scene').innerText()
      return text.match(/\d+\s*\/\s*(\d+)/)?.[1] ?? ''
    }
    await expect.poll(sheetHP).toMatch(/\d+/)
    const before = Number(await sheetHP())

    const gmScreen = await gm.newPage()
    await gmScreen.goto('/campanhas/1?tab=regras')
    await gmScreen.getByRole('button', { name: 'Nova regra' }).click()
    const editor = gmScreen.locator('dialog[open]')
    await editor.getByLabel('Nome').fill('Vigor da mesa')
    await editor.getByLabel('Em quê').selectOption({ label: 'PV' })
    await editor.getByLabel('Quanto').fill('5')
    await editor.getByLabel('Quanto').blur()
    await expect(editor.getByText('+5 em PV')).toBeVisible()
    await editor.getByRole('button', { name: 'Gravar' }).click()
    const row = gmScreen.locator('.ds-table-rule', { hasText: 'Vigor da mesa' })
    await row.getByRole('switch').click()
    await expect(gmScreen.locator('.ds-table-rule', { hasText: 'Vigor da mesa' }).getByRole('switch')).toBeChecked()

    await expect
      .poll(sheetHP, { timeout: 8000, message: 'a ficha aberta não soube da regra ligada' })
      .toBe(String(before + 5))

    await gmScreen.locator('.ds-table-rule', { hasText: 'Vigor da mesa' }).getByRole('button', { name: 'Editar' }).click()
    await expect(gmScreen.locator('dialog[open]').getByLabel('Nome')).toHaveValue('Vigor da mesa')
    await gmScreen.locator('dialog[open]').getByRole('button', { name: 'Excluir' }).click()
    await gmScreen.getByRole('dialog', { name: 'Excluir esta regra?' }).getByRole('button', { name: 'Excluir' }).click()
    await expect(gmScreen.locator('.ds-table-rule', { hasText: 'Vigor da mesa' })).toHaveCount(0)

    await expect
      .poll(sheetHP, { timeout: 8000, message: 'a ficha aberta não soube da regra excluída' })
      .toBe(String(before))
  } finally {
    await gm.close().catch(() => {})
    await player.close().catch(() => {})
  }
})
