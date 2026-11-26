import { expect, test } from '@playwright/test'

/**
 * O EDITOR DO BLOCO DO NPC, no navegador, que é a única testemunha de duas coisas:
 *
 *  1. **As abas abrem.** A fileira (Números, Ataques, Perícias e posses) mora no corpo do editor, que rola, e uma
 *     caixa que rola encolhe até a borda quando o corpo aperta: ela chegou a ter 1 px, com o título "Identidade"
 *     por cima, e as abas de Ataques e Perícias não abriam. O handler vê as três no HTML.
 *  2. **O "Preencher pelo ND" escreve os ataques que a aba Ataques mostra** (Tabela 7-2, p323).
 */

test.use({ storageState: '.auth/user.json' })

const ELENCO = '/campanhas/1/sessoes/4/elenco'

for (const [width, height] of [
  [390, 844],
  [1440, 900],
]) {
  test(`a ${width}px as abas do editor abrem, e o Preencher pelo ND escreve os ataques que a aba mostra`, async ({ page }) => {
    await page.setViewportSize({ width, height })
    await page.goto(ELENCO)
    await page.getByRole('button', { name: 'Novo no elenco' }).first().click()
    const editor = page.locator('.ds-npc-editor')
    await expect(editor).toBeVisible()

    // A FILEIRA tem a altura das abas, e não a da borda.
    const tabs = editor.getByRole('tablist', { name: 'Partes do bloco' })
    const box = await tabs.boundingBox()
    expect(box?.height ?? 0, 'a fileira de abas encolheu até a borda').toBeGreaterThanOrEqual(40)

    await editor.locator('#npc-nd').fill('5')
    await editor.getByRole('button', { name: 'Preencher pelo ND' }).click()
    await expect(editor.locator('#npc-defesa')).toHaveValue('28')
    await expect(editor.locator('#npc-hp')).toHaveValue('150')

    // O CLIQUE é o de uma pessoa: sem `force`, para o título por cima reprovar em vez de ser atravessado.
    const attacks = editor.getByRole('tab', { name: /Ataques/ })
    await attacks.click()
    await expect(attacks).toHaveAttribute('aria-selected', 'true')
    const rows = editor.locator('#npc-attacks')
    await expect(rows, 'a aba Ataques não abriu').toBeVisible()
    await expect(rows.locator('input[type=number]')).toHaveCount(2)

    await editor.getByRole('button', { name: 'Cancelar' }).click()
  })
}
