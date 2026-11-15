import { type Page, expect, test } from '@playwright/test'
import { openTheBoard, putACombatantInTheTracker } from './support/table'

/**
 * AGIR NO TABULEIRO (ALE-660): tocar na peça de outro escolhe o alvo da vez e abre o painel das ações.
 *
 * Por que e2e: o gesto é de PONTEIRO numa peça do mapa, e o painel é a superfície Ações posta por cima do
 * tabuleiro pelo CSS (`ds-board-panel`). O Go confere os atributos no HTML; só o navegador diz se o toque chega
 * à peça, se o painel aparece onde a pessoa está, e se o ✕ o fecha sem desfazer o alvo.
 *
 * Numa mesa SÓ deste caso — a 11, do Clã do Machado Rubro, onde o jogador tem o Bárbaro Furioso e nenhuma
 * outra suíte escreve.
 */
const MESA = '/campanhas/8/sessoes/11'
const OGRO = 'Ogro do Painel'
const ds = { headers: { 'datastar-request': 'true' } }

async function post(page: Page, path: string, data: Record<string, unknown> = {}): Promise<void> {
  const reply = await page.request.post(MESA + path, { ...ds, data })
  expect(reply.ok(), `${path} respondeu ${reply.status()}`).toBeTruthy()
}

test('tocar na peça de outro escolhe o alvo da vez e abre o painel das ações', async ({ browser }) => {
  test.setTimeout(120_000)
  const gm = await browser.newContext({ storageState: '.auth/user.json', viewport: { width: 1280, height: 800 } })
  const player = await browser.newContext({ storageState: '.auth/player.json', viewport: { width: 1280, height: 800 } })
  const gmPage = await gm.newPage()
  const playerPage = await player.newPage()
  try {
    // O PALCO pelo mestre: o mapa, o ogro e o grupo na fila e no mapa, e a vez do bárbaro.
    await openTheBoard(gmPage, MESA)
    await putACombatantInTheTracker(gmPage, OGRO)
    await post(gmPage, '/iniciativa/por-no-mapa')
    await gmPage.goto(MESA, { waitUntil: 'domcontentloaded' })
    const ids = [...new Set([...(await gmPage.content()).matchAll(/commandfor="combatant-([^"]+)"/g)].map((m) => m[1]))]
    await post(gmPage, '/tabuleiro/pecas', { map_selection: ids.join(',') })
    await post(gmPage, '/cena/iniciar/acao')

    await playerPage.goto(MESA, { waitUntil: 'domcontentloaded' })
    for (let turn = 0; turn < 4 && !(await playerPage.locator('.ds-your-turn').isVisible()); turn++) {
      await post(gmPage, '/iniciativa/proxima-vez')
      await playerPage.reload({ waitUntil: 'domcontentloaded' })
    }
    await expect(playerPage.locator('.ds-your-turn'), 'a vez não chegou ao bárbaro').toBeVisible()

    // O GESTO, pela tela: a aba Tabuleiro, o enquadrar e o toque na peça do ogro.
    await playerPage.getByRole('navigation', { name: 'Superfícies da sessão' }).getByRole('button', { name: 'Tabuleiro' }).click()
    await playerPage.getByRole('group', { name: 'Enquadrar o mapa' }).getByRole('button').first().click()
    await playerPage.getByRole('button', { name: new RegExp(`^${OGRO} em `) }).click()

    const head = playerPage.locator('.ds-board-target-head')
    await expect(head, 'o toque na peça não abriu o painel').toBeVisible()
    await expect(head).toContainText(OGRO)
    await expect(head, 'o cabeçalho não diz a distância').toContainText(/a \d+,\dm · /)
    const strike = playerPage.locator('.ds-board-panel .ds-turn-action', { hasText: 'Atacar' }).first()
    await expect(strike, 'o painel não traz o golpe').toBeVisible()
    await expect(strike.getByRole('button', { name: 'Rolar' }), 'o golpe contra o alvo escolhido não está pronto').toBeEnabled()
    await expect(playerPage.locator('.ds-board-panel').getByText('Mover', { exact: true }), 'o painel trouxe o que não pede alvo').toBeHidden()

    // O ✕ SÓ FECHA (decisão do dono): o alvo continua escolhido.
    await playerPage.getByRole('button', { name: 'Fechar o painel' }).click()
    await expect(head, 'o ✕ não fechou o painel').toBeHidden()
    await expect(playerPage.locator('.ds-board-panel'), 'o ✕ não fechou o painel').toBeHidden()
    const chosen = await playerPage.locator('#table-turn-target select').inputValue()
    expect(chosen, 'o ✕ desfez o alvo escolhido').not.toBe('')
  } finally {
    await gm.close().catch(() => {})
    await player.close().catch(() => {})
  }
})
