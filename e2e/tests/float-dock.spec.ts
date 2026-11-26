import { expect, type Page, test } from '@playwright/test'

/**
 * OS CARTÕES DO MESTRE NASCEM EMBAIXO, À DIREITA, E RECOLHEM (ALE-723; ver
 * GLOSSARY, **cartão flutuante**).
 *
 * E2E porque o lugar onde o cartão nasce é calculado no navegador
 * (`lib/float-card.ts`, contra a caixa real das faixas e da linha do turno), e
 * o recolhido tem de sobreviver ao REMENDO do Datastar — que reescreve os
 * atributos do cartão a cada mudança — e ao recarregar. O cartão do caso é o
 * Marcar do teste estendido, aberto pela API na mesa da campanha 7.
 */

const MESA = '/campanhas/7/sessoes/10'

async function command(page: Page, path: string, data: Record<string, unknown> = {}): Promise<string> {
  const reply = await page.request.post(MESA + path, { headers: { 'datastar-request': 'true' }, data })
  expect(reply.ok(), `${path} respondeu ${reply.status()}`).toBeTruthy()
  const found = /"command_error":"((?:[^"\\]|\\.)*)"/.exec(await reply.text())
  return found ? JSON.parse(`"${found[1]}"`) : ''
}

async function opensTheTest(page: Page): Promise<void> {
  await command(page, '/teste-estendido/encerrar')
  expect(await command(page, '/teste-estendido/abrir', { ext_name: 'Atravessar o Pântano', ext_required: 5, ext_each: false })).toBe('')
}

test('o cartão Marcar nasce embaixo, à direita, longe da faixa, e o recolhido sobrevive ao remendo e ao recarregar', async ({ page }) => {
  await page.setViewportSize({ width: 1440, height: 900 })
  await opensTheTest(page)
  try {
    await page.goto(MESA)
    const card = page.locator('#table-extended-marks')
    const band = page.locator('#table-extended')
    await expect(card.getByRole('button', { name: 'Marcar um sucesso', exact: true })).toBeVisible()
    expect(await card.evaluate((el) => getComputedStyle(el).position)).toBe('fixed')

    // NASCE NA PILHA: encostado na borda direita, acima da linha do turno, e sem
    // pousar sobre a faixa do topo — o defeito que abriu a issue.
    const box = await card.boundingBox()
    const strip = await band.boundingBox()
    const floor = await page.locator('[data-float-floor]').boundingBox()
    if (!box || !strip || !floor) throw new Error('falta caixa ao cartão, à faixa ou à linha do turno')
    expect(Math.round(1440 - (box.x + box.width))).toBe(12)
    expect(box.y + box.height).toBeLessThanOrEqual(floor.y - 11)
    expect(box.y, 'o cartão pousou sobre a faixa do teste').toBeGreaterThan(strip.y + strip.height)

    // RECOLHER deixa a cabeça, com o placar.
    await card.getByRole('button', { name: 'Recolher ou abrir o Marcar' }).click()
    await expect(card.getByRole('button', { name: 'Marcar um sucesso', exact: true })).toBeHidden()
    await expect(card.locator('.ds-float-count')).toHaveText('0/5')

    // O REMENDO QUE MUDA O PRÓPRIO CARTÃO: o placar trocar é o controle de que ele
    // chegou — sem isso, "continuou recolhido" pode ser "nada foi remendado".
    expect(await command(page, '/teste-estendido/marcar/sucesso')).toBe('')
    await expect(card.locator('.ds-float-count')).toHaveText('1/5')
    await expect(card.getByRole('button', { name: 'Marcar um sucesso', exact: true })).toBeHidden()
    // Recolhido, ele DESCE: a pilha encosta no chão pelo tamanho novo.
    const folded = await card.boundingBox()
    expect(Math.round((folded?.y ?? 0) + (folded?.height ?? 0))).toBe(Math.round(floor.y - 12))

    await page.reload()
    await expect(card.locator('.ds-float-count')).toHaveText('1/5')
    await expect(card.getByRole('button', { name: 'Marcar um sucesso', exact: true })).toBeHidden()

    await card.getByRole('button', { name: 'Recolher ou abrir o Marcar' }).click()
    await expect(card.getByRole('button', { name: 'Marcar um sucesso', exact: true })).toBeVisible()

    // NO CELULAR ele fica no fluxo, com os botões à vista e sem alça nem seta.
    await page.setViewportSize({ width: 390, height: 844 })
    expect(await card.evaluate((el) => getComputedStyle(el).position)).not.toBe('fixed')
    await expect(card.getByRole('button', { name: 'Marcar um sucesso', exact: true })).toBeVisible()
    await expect(card.getByRole('button', { name: 'Recolher ou abrir o Marcar' })).toBeHidden()
    await expect(card.locator('[data-float-grip]')).toBeHidden()
  } finally {
    await command(page, '/teste-estendido/encerrar').catch(() => '')
  }
})
