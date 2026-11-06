import { expect, test } from '@playwright/test'

// O MESTRE ABRE A FICHA DO JOGADOR pelo "Ficha" do grupo, numa aba nova, e pode mexer nela (decisão do
// dono). Antes a aba nova respondia 403. E2e porque o caminho é um link que abre outra aba a partir da
// página do Grupo, e o que se mede é a página que ela carrega.
test('o mestre abre a ficha do jogador pelo grupo da mesa', async ({ page }) => {
  await page.goto('/campanhas/1/sessoes/4/grupo')
  const popup = page.waitForEvent('popup')
  await page.getByRole('link', { name: /Abrir a ficha completa de Guerreiro Veterano Nv8/ }).first().click()
  const sheet = await popup
  await sheet.waitForLoadState('domcontentloaded')
  await expect(sheet.locator('.ds-sheet-name')).toHaveText('Guerreiro Veterano Nv8')
  await expect(sheet.locator('.ds-sheet-where')).toContainText('Na mesa')
})
