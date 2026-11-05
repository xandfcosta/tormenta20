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
  await page.locator('dialog[open]').getByRole('button', { name: 'Excluir' }).click()
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
