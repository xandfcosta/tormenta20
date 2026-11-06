import { expect, test } from '@playwright/test'

// A CÓPIA DE CAMPANHA, do convite à ficha que se atualiza sozinha. E2e porque o caminho atravessa três
// telas, e porque a ficha solta só se atualiza por um fluxo ao vivo que nenhum teste de Go vê chegar.
//
// O rascunho é o "Batedor Nv2 Simples", que a semente não senta em mesa nenhuma, e a mesa é a "Segredos de
// Wynlla" (convite `seedtoken-03`), em que o jogador ainda não tem herói: a mesa aceita um por jogador. Numa segunda rodada sobre o mesmo banco a cópia já existe, e o caso
// segue dela: o convite recusaria um segundo herói do mesmo rascunho.
const DRAFT = 'Batedor Nv2 Simples'
const TABLE = 'Segredos de Wynlla'

test('o rascunho entra na mesa, a cópia aparece na lista e a ficha dela acompanha a mudança', async ({
  browser,
}) => {
  test.setTimeout(90_000)
  const first = await browser.newContext({ storageState: '.auth/player.json' })
  const second = await browser.newContext({ storageState: '.auth/player.json' })
  try {
    const page = await first.newPage()
    await page.goto('/personagens')
    const card = page.locator('.ds-cast-card', { hasText: DRAFT })
    if (!(await card.getByText('Nas mesas').count())) {
      await page.goto('/campanhas/entrar?token=seedtoken-03')
      await page.locator('label', { hasText: DRAFT }).click()
      await page.getByRole('button', { name: 'Entrar na mesa' }).click()
      await expect(page).not.toHaveURL(/entrar/)
      await page.goto('/personagens')
    }
    await expect(card).toContainText('Nas mesas')

    // O fluxo ao vivo abre com a página: o pedido dele é o sinal de que o ouvido está lá.
    const live = page.waitForRequest((request) => request.url().includes('/ao-vivo'))
    await card.getByRole('link', { name: `Ficha de ${DRAFT} na mesa ${TABLE}` }).click()
    await live
    await expect(page.locator('.ds-sheet-where')).toContainText(`Na mesa ${TABLE}`)
    const copyID = /\/personagens\/(\d+)/.exec(page.url())?.[1]
    expect(copyID, 'a linha da mesa não levou à ficha da cópia').toBeTruthy()

    const hp = async () => (await page.locator('#sheet-scene').innerText()).match(/(\d+)\s*\/\s*\d+/)?.[1] ?? ''
    await expect.poll(hp).toMatch(/\d+/)
    const before = Number(await hp())

    // O GESTO DA PRÓPRIA ABA não busca a ficha de novo: a resposta dele já a trouxe.
    const refetches: string[] = []
    page.on('request', (request) => {
      if (request.method() === 'GET' && request.url().includes(`/personagens/${copyID}?`)) {
        refetches.push(request.url())
      }
    })
    await page.getByRole('button', { name: 'Menos 1 PV' }).locator('visible=true').first().click()
    await expect.poll(hp).toBe(String(before - 1))
    await page.waitForTimeout(1500)
    expect(refetches, 'o gesto da própria aba repediu a ficha').toEqual([])

    const other = await second.newPage()
    await other.goto(`/personagens/${copyID}`)
    const reply = await other.request.post(`/personagens/${copyID}/vitais/pv/-1`, {
      headers: { 'datastar-request': 'true' },
      data: {},
    })
    expect(reply.ok(), `o dano pela outra aba respondeu ${reply.status()}`).toBeTruthy()

    await expect
      .poll(hp, { timeout: 8000, message: 'a ficha aberta não soube do dano dado pela outra aba' })
      .toBe(String(before - 2))
  } finally {
    await first.close().catch(() => {})
    await second.close().catch(() => {})
  }
})
