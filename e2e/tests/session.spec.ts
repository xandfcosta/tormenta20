import { expect, test } from '@playwright/test'

const CAMPAIGN = 'Snapshot Test ALE-33' // the seed chronicle with a live session

/**
 * Hub → Crônicas → abrir campanha → entrar na sessão ao vivo.
 *
 * Read-only once inside: asserts the socket.io gateway connected, without
 * touching initiative/turns, so the seed survives the run untouched.
 */
test.describe('Sessão ao vivo', () => {
  test('Crônicas → campanha → continuar a sessão (realtime conectado)', async ({ page }) => {
    await page.goto('/campaigns')
    await page.getByRole('button', { name: /^Abrir crônica/ }).click()
    await expect(page.getByRole('heading', { name: CAMPAIGN, level: 1 })).toBeVisible()

    await page.getByRole('button', { name: 'Continuar a sessão' }).click()
    await expect(page).toHaveURL(/\/campaigns\/\d+\/sessions\/\d+$/)

    await expect(page.getByRole('heading', { name: 'Iniciativa' })).toBeVisible()
    // The connection chip flips to "Conectado" only after the socket handshake.
    await expect(page.getByRole('status', { name: 'Conectado' })).toBeVisible()
  })

  test('Sair da sessão volta pra crônica', async ({ page }) => {
    await page.goto('/campaigns/1/sessions/4')
    await expect(page.getByRole('heading', { name: 'Iniciativa' })).toBeVisible()

    await page.getByRole('button', { name: 'Sair da sessão' }).click()
    await expect(page).toHaveURL(/\/campaigns\/1$/)
    await expect(page.getByRole('heading', { name: CAMPAIGN, level: 1 })).toBeVisible()
  })
})
