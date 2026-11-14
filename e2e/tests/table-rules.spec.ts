import { expect, type Page, test } from '@playwright/test'
import { tidyUp } from './support/tidy'

/** Apaga pela tela a regra deste nome, se ela ainda estiver na campanha 1. */
async function removeRule(page: Page, name: string): Promise<void> {
  await page.goto('/campanhas/1?tab=regras')
  const row = page.locator('.ds-table-rule', { hasText: name })
  if ((await row.count()) === 0) return
  await row.getByRole('button', { name: 'Editar' }).click()
  await page.locator('dialog[open]').getByRole('button', { name: 'Excluir' }).click()
  await page.getByRole('dialog', { name: 'Excluir esta regra?' }).getByRole('button', { name: 'Excluir' }).click()
  await expect(row).toHaveCount(0, { timeout: 5000 })
}


// EXCLUIR PEDE CONFIRMAÇÃO num `<dialog>` aberto por cima do editor, que também é `<dialog>`: o Cancelar
// tem de fechar só o de cima. É o empilhamento nativo de diálogos, e só o navegador o tem. O que o editor
// grava, desenha e reabre é dos `campaign_rules_*_test.go` em Go.
test('excluir a regra pede confirmação, e cancelar devolve ao editor', async ({ page }) => {
  const name = `Regra do e2e ${Date.now()}`
  const saved = await page.request.post('/campanhas/1/regras-da-mesa/gravar', {
    headers: { 'datastar-request': 'true' },
    data: { table_rule_name: name, table_rule_scope: 'mesa', table_rule_count: 0, table_rule_target: 'defense', table_rule_amount: 2 },
  })
  expect(saved.ok(), `gravar a regra do caso respondeu ${saved.status()}`).toBeTruthy()
  const row = page.locator('.ds-table-rule', { hasText: name })
  const editor = page.locator('dialog[open]')
  const confirm = page.getByRole('dialog', { name: 'Excluir esta regra?' })
  try {
    await page.goto('/campanhas/1?tab=regras')
    await row.getByRole('button', { name: 'Editar' }).click()
    await expect(editor.getByLabel('Nome')).toHaveValue(name)

    await editor.getByRole('button', { name: 'Excluir' }).click()
    await confirm.getByRole('button', { name: 'Cancelar' }).click()
    await expect(confirm).toBeHidden()
    await expect(editor.getByLabel('Nome'), 'cancelar a exclusão fechou o editor junto').toHaveValue(name)

    await editor.getByRole('button', { name: 'Excluir' }).click()
    await confirm.getByRole('button', { name: 'Excluir' }).click()
    await expect(row).toHaveCount(0)
  } finally {
    await tidyUp(`a regra "${name}" na campanha 1`, () => removeRule(page, name))
  }
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
    // LIGADA, a regra dá +5 PV a toda ficha da campanha 1: o próximo spec mediria a ficha errada.
    await tidyUp('a regra "Vigor da mesa" na campanha 1', async () => removeRule(await gm.newPage(), 'Vigor da mesa'))
    await gm.close().catch(() => {})
    await player.close().catch(() => {})
  }
})

// O outro lado da mesma mudança: o PV máximo na fila do mestre. Quem o relê é o stream da mesa dele.
test('ligar a regra chega na fila do mestre', async ({ browser }) => {
  test.setTimeout(90_000)
  const gm = await browser.newContext({ storageState: '.auth/user.json' })
  try {
    const table = await gm.newPage()
    await table.goto('/campanhas/1/sessoes/5')
    const start = table.getByRole('button', { name: 'Iniciar uma cena de Ação' })
    if (await start.count()) await start.click()
    await expect(start).toHaveCount(0)
    await table.getByRole('button', { name: 'Adicionar à fila' }).first().click()
    await table.locator('#gm-add').getByRole('button', { name: 'Grupo', exact: true }).click()
    await table.locator('#gm-add').getByRole('button', { name: 'Adicionar o grupo' }).click()
    const rows = table.locator('.ds-gm-row')
    await expect(rows.first()).toBeVisible()
    const queueRow = rows.filter({ hasText: 'Recruta' }).first()
    const queueHP = async () => (await queueRow.innerText()).match(/PV\s*\d+\s*\/\s*(\d+)/)?.[1] ?? ''
    await expect.poll(queueHP).toMatch(/\d+/)
    const before = Number(await queueHP())

    const rulesScreen = await gm.newPage()
    await rulesScreen.goto('/campanhas/1?tab=regras')
    await rulesScreen.getByRole('button', { name: 'Nova regra' }).click()
    const editor = rulesScreen.locator('dialog[open]')
    await editor.getByLabel('Nome').fill('Vigor na fila')
    await editor.getByLabel('Em quê').selectOption({ label: 'PV' })
    await editor.getByLabel('Quanto').fill('5')
    await editor.getByLabel('Quanto').blur()
    await expect(editor.getByText('+5 em PV')).toBeVisible()
    await editor.getByRole('button', { name: 'Gravar' }).click()
    await rulesScreen.locator('.ds-table-rule', { hasText: 'Vigor na fila' }).getByRole('switch').click()
    await expect(rulesScreen.locator('.ds-table-rule', { hasText: 'Vigor na fila' }).getByRole('switch')).toBeChecked()

    await expect.poll(queueHP, { timeout: 8000, message: 'a fila do mestre não soube da regra' }).toBe(String(before + 5))

    await rulesScreen.locator('.ds-table-rule', { hasText: 'Vigor na fila' }).getByRole('button', { name: 'Editar' }).click()
    await expect(rulesScreen.locator('dialog[open]').getByLabel('Nome')).toHaveValue('Vigor na fila')
    await rulesScreen.locator('dialog[open]').getByRole('button', { name: 'Excluir' }).click()
    await rulesScreen.getByRole('dialog', { name: 'Excluir esta regra?' }).getByRole('button', { name: 'Excluir' }).click()
    await expect(rulesScreen.locator('.ds-table-rule', { hasText: 'Vigor na fila' })).toHaveCount(0)
    await expect.poll(queueHP, { timeout: 8000, message: 'a fila do mestre não soube da regra excluída' }).toBe(String(before))
  } finally {
    await tidyUp('a regra "Vigor na fila" na campanha 1', async () => removeRule(await gm.newPage(), 'Vigor na fila'))
    // A sessão 5 é dividida com o spec dos dois clientes, então ela volta como estava.
    await tidyUp('a cena de ação aberta na sessão 5', async () => {
      const table = await gm.newPage()
      await table.goto('/campanhas/1/sessoes/5')
      const start = table.getByRole('button', { name: 'Iniciar uma cena de Ação' })
      if (await start.count()) return
      await table.getByRole('button', { name: 'Cena, descanso e sessão' }).click()
      await table.getByRole('button', { name: 'Encerrar a cena' }).click()
      await expect(start).toBeVisible({ timeout: 5000 })
    })
    await gm.close().catch(() => {})
  }
})
