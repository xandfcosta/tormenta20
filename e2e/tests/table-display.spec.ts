import { type BrowserContext, expect, type Page, test } from '@playwright/test'
import { disposableTable, openTheBoard } from './support/table'

/**
 * A TELA DA MESA (ALE-608) segue o enquadramento do mestre e para quando ele
 * congela. O navegador é a única testemunha: o recorte nasce da MEDIDA do
 * tabuleiro do mestre e é encaixado pela medida do display — duas caixas que
 * só existem com leiaute —, e viaja entre dois documentos.
 */

test.use({ storageState: '.auth/user.json' })

/** O lado do quadrado que a tela está desenhando agora. */
async function squareOf(page: Page): Promise<string> {
  return page
    .locator('#board-scene')
    .evaluate((el) => getComputedStyle(el).getPropertyValue('--quadrado').trim())
}

test('a tela da mesa segue o recorte do mestre e para quando ele congela', async ({ page, context }) => {
  await page.setViewportSize({ width: 1440, height: 900 })
  // UMA MESA SÓ DESTE CASO: abrir tabuleiro na sessão da semente mudaria o
  // que os outros specs encontram nela.
  const { mesa, apagar } = await disposableTable(page)
  try {
    await openTheBoard(page, mesa)
    await followsAndFreezes(page, context, mesa)
  } finally {
    await apagar()
  }
})

async function followsAndFreezes(page: Page, context: BrowserContext, mesa: string): Promise<void> {
  const display = await context.newPage()
  await display.setViewportSize({ width: 1920, height: 1080 })
  await display.goto(`${mesa}/tela`)
  await expect(display.locator('#board-scene'), 'a tela da mesa não desenhou o mapa').toBeVisible()
  // Ela não tem controle nenhum: só o mapa.
  await expect(display.getByRole('button', { name: 'Aproximar o mapa' })).toHaveCount(0)

  // O CONTROLE: com a tela aberta, o mestre vê o interruptor ao vivo.
  const live = page.getByText('Tela da mesa ao vivo')
  await expect(live, 'o mestre não viu que há uma tela da mesa aberta').toBeVisible()
  const zoomIn = page.getByRole('button', { name: 'Aproximar o mapa' })
  const zoomOut = page.getByRole('button', { name: 'Afastar o mapa' })

  const before = await squareOf(display)
  await zoomIn.click()
  await expect.poll(() => squareOf(display), { message: 'a tela não seguiu o zoom do mestre' }).not.toBe(before)

  await page.getByRole('button', { name: 'Congelar' }).click()
  await expect(page.getByText('Tela da mesa congelada')).toBeVisible()
  const frozen = await squareOf(display)
  await zoomOut.click()
  await zoomOut.click()
  // Esperar mais que o batimento do fluxo e o adiamento do recorte: a
  // ausência de mudança só prova algo depois do tempo em que ela viria.
  await display.waitForTimeout(1500)
  expect(await squareOf(display), 'a tela congelada seguiu o zoom do mestre').toBe(frozen)

  await page.getByRole('button', { name: 'Soltar' }).click()
  await expect.poll(() => squareOf(display), { message: 'solta, a tela não voltou ao recorte do mestre' }).not.toBe(frozen)
  await expect(live).toBeVisible()

  await display.close()
}
