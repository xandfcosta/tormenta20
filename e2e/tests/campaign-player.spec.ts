import { type Page, expect, test } from '@playwright/test'

/**
 * The campaign scene seen by a PLAYER, not the GM (ALE-24). Every other spec
 * runs as the seeded GM through the shared storageState; this one opts out and
 * signs in as the player who is a member of campaign 1 but owns nothing there.
 *
 * The real gate is the server's — this is the UX half: offering a member a
 * button that can only 403 is worse than not offering it.
 */
test.use({ storageState: '.auth/player.json' })

const CAMPAIGN = '/campaigns/1' // do mestre; o jogador é só membro

/**
 * Opens a section and waits for the tome to actually be on screen.
 *
 * Every check here is about what must NOT be there, and an assertion of absence
 * passes on a blank page — so each one has to start from a scene proven to have
 * rendered. `networkidle` because in dev the first cold visit to a route makes
 * Vite re-optimize deps and force a reload, which strands the default wait on
 * the discarded document.
 */
async function openSection(page: Page, tab: string): Promise<void> {
  await page.goto(`${CAMPAIGN}?tab=${tab}`, { waitUntil: 'networkidle' })
  await expect(page.getByRole('heading', { name: /Snapshot Test ALE-33/i })).toBeVisible()
}

test.describe('Campanha vista pelo jogador', () => {
  test('o rail não oferece a seção do mestre', async ({ page }) => {
    await openSection(page, 'visao')

    await expect(page.getByRole('tab', { name: 'Visão geral' })).toBeVisible()
    await expect(page.getByRole('tab', { name: 'Membros' })).toBeVisible()
    await expect(page.getByRole('tab', { name: 'Config' })).toHaveCount(0)
  })

  test('nenhuma ação de dono aparece nas seções', async ({ page }) => {
    await openSection(page, 'visao')

    // Percorre o rail clicando, como o jogador faria — e não com três loads,
    // que é o que a cena existe para evitar.
    for (const secao of ['Visão geral', 'Sessões', 'Membros']) {
      await page.getByRole('tab', { name: secao }).click()
      await expect(page.getByRole('tab', { name: secao })).toHaveAttribute('data-selected', '')

      await expect(page.getByRole('button', { name: /Convite/ })).toHaveCount(0)
      await expect(page.getByRole('button', { name: /^Sessão \d+$/ })).toHaveCount(0)
      await expect(page.getByRole('button', { name: 'Editar' })).toHaveCount(0)
      await expect(page.getByRole('button', { name: /Excluir campanha/ })).toHaveCount(0)
      await expect(page.getByLabel(/^Remover /)).toHaveCount(0)
    }
  })

  // `?tab=config` digitado à mão não pode revelar a seção — e os bumpers
  // (PgUp/PgDn) também só andam pelas seções que o rail do jogador tem.
  test('pedir a seção do mestre pela URL cai na visão geral', async ({ page }) => {
    await openSection(page, 'config')

    // Caiu numa seção de verdade, não numa tela vazia.
    await expect(page.getByRole('tab', { name: 'Visão geral' })).toHaveAttribute('data-selected', '')
    await expect(page.getByText('Zona de perigo')).toHaveCount(0)
    await expect(page.getByRole('button', { name: /Excluir campanha/ })).toHaveCount(0)
  })

  // O jogador PERDE ações de escrita, não a mesa: ele continua lendo a crônica
  // e entrando na sessão ao vivo.
  test('o jogador ainda lê a crônica e entra na sessão ao vivo', async ({ page }) => {
    await openSection(page, 'sessoes')

    await expect(page.getByText('JOGANDO')).toBeVisible()
    await expect(page.getByRole('button', { name: /Continuar a sessão/ })).toBeVisible()
    await expect(page.getByText('Sessão 5')).toBeVisible()
  })
})
