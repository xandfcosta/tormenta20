import { expect, test } from '@playwright/test'
import { disposableTable, openTheBoard, putATokenOnTheMap, putTheTokenOnTurn } from './support/table'

/**
 * OS CARTÕES SOBRE O TABULEIRO NO CELULAR, no padrão do cartão do ataque (ALE-653, mockup aprovado):
 * só a CABEÇA fica à vista, e o corpo abre num diálogo. O tabuleiro é o foco — a faixa do Estandarte
 * com um botão por aliado chegava a cobrir a tela inteira do mestre numa mesa grande.
 */

const ds = { headers: { 'datastar-request': 'true' } }

test('no celular, a faixa do Estandarte mostra só a cabeça, e os aliados abrem num diálogo', async ({ page }) => {
  const MESA = '/campanhas/1/sessoes/4'
  await page.setViewportSize({ width: 390, height: 844 })
  const start = await page.request.post(`${MESA}/cena/iniciar/acao`, { ...ds, data: {} })
  expect(start.ok(), `iniciar a cena respondeu ${start.status()}`).toBeTruthy()
  try {
    await page.goto(MESA, { waitUntil: 'domcontentloaded' })
    const banner = page.locator('#table-banner')
    await expect(banner, 'a cena começou e a faixa do Estandarte não apareceu').toBeVisible()
    await expect(banner.getByText('Estandarte').first()).toBeVisible()
    const ally = page.getByRole('button', { name: /^Dar o Estandarte a / }).first()
    await expect(ally, 'no celular os aliados ficam fora da vista até abrir a faixa').toBeHidden()
    const box = await banner.boundingBox()
    expect(box?.height ?? 0, 'a cabeça da faixa é uma linha, e não a lista de aliados').toBeLessThan(110)

    await page.getByRole('button', { name: 'Abrir o Estandarte' }).click()
    await expect(ally, 'tocar na cabeça não mostrou os aliados').toBeVisible()
    await expect(page.getByRole('button', { name: /^Pronto/ }).first()).toBeVisible()
  } finally {
    await page.request.post(`${MESA}/cena/encerrar`, { ...ds, data: {} }).catch(() => {})
  }
})

test('no celular, o cartão do movimento mostra a cabeça com o Confirmar, e o resto abre num diálogo', async ({ page }) => {
  test.setTimeout(90_000)
  const { mesa, apagar } = await disposableTable(page)
  try {
    await page.setViewportSize({ width: 1280, height: 800 })
    await openTheBoard(page, mesa)
    await putATokenOnTheMap(page)
    await putTheTokenOnTurn(page, mesa)
    const token = page.locator('.board-token-on-turn')
    const box = await token.boundingBox()
    if (!box) throw new Error('a peça não tem caixa')
    await page.mouse.move(box.x + box.width / 2, box.y + box.height / 2)
    await page.mouse.down()
    await page.mouse.move(box.x + box.width / 2 + 120, box.y + box.height / 2, { steps: 8 })
    const dropped = page.waitForEvent('requestfinished', { predicate: (r) => r.url().includes('/parada') })
    await page.mouse.up()
    await dropped

    await page.setViewportSize({ width: 390, height: 844 })
    await page.goto(mesa, { waitUntil: 'domcontentloaded' })
    await page.getByRole('navigation', { name: 'Superfícies do mestre' }).getByRole('button', { name: 'Tabuleiro' }).click()
    const card = page.locator('.ds-board-card', { hasText: 'Ogro do E2E' })
    await expect(card.getByRole('button', { name: 'Confirmar' }), 'a cabeça do movimento não traz o Confirmar à mão').toBeVisible()
    const cancel = page.getByRole('button', { name: 'Cancelar', exact: true })
    await expect(cancel, 'no celular o resto do cartão fica fora da vista até abrir').toBeHidden()

    await page.getByRole('button', { name: 'Abrir o movimento' }).click()
    await expect(cancel, 'tocar na cabeça não abriu o resto do movimento').toBeVisible()
  } finally {
    await apagar()
  }
})
