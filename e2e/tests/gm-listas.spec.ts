import { expect, test } from '@playwright/test'

/**
 * As listas virtualizadas do MESTRE — bestiário e catálogos.
 *
 * `VirtualList` mede as linhas para saber quais existem, e em jsdom todo
 * elemento mede zero: a lista renderiza NENHUMA linha e um teste de unidade
 * passa verde sobre a tela vazia. Foi assim que a ALE-84 entrou em produção com
 * a suíte inteira verde. Só um browser prova que a linha pintou — e que o
 * filtro TROCA o conjunto pintado, que é o outro modo de falha da
 * virtualização: ficar com o que pintou primeiro.
 *
 * Só leitura: filtra e navega, nunca escreve.
 */
test.describe('Listas virtualizadas do mestre', () => {
  test('o bestiário da sessão pinta linhas e o filtro troca o conjunto', async ({ page }) => {
    await page.goto('/campaigns/1/sessions/4')
    await page.getByRole('tab', { name: 'Bestiário' }).click()

    const busca = page.getByRole('searchbox', { name: 'Buscar criatura' })
    await expect(busca).toBeVisible()
    // A linha só existe se a lista mediu e pintou.
    const antes = await page.getByRole('button', { name: /ND / }).count()
    expect(antes).toBeGreaterThan(0)

    await busca.fill('ogro')
    // Mais de um ogro no bestiário (o comum e o ancião) — `first()` de propósito.
    await expect(page.getByRole('button', { name: /^Ogro/ }).first()).toBeVisible()
    expect(await page.getByRole('button', { name: /ND / }).count()).toBeLessThan(antes)
  })

  test('o catálogo da sessão pinta linhas e a busca as troca', async ({ page }) => {
    await page.goto('/campaigns/1/sessions/4')
    await page.getByRole('tab', { name: 'Catálogos' }).click()

    const busca = page.getByRole('searchbox', { name: 'Buscar nos catálogos' })
    await expect(busca).toBeVisible()

    await busca.fill('espada')
    await expect(page.getByText(/Espada/).first()).toBeVisible()

    // O outro modo de falha: a lista guardar o que pintou primeiro. Depois de
    // uma busca sem resultado, nada de "espada" pode sobreviver na tela.
    await busca.fill('zzzzzz')
    await expect(page.getByText(/Espada longa/)).toHaveCount(0)
  })

  test('a ferramenta Bestiário pinta a lista e abre a criatura escolhida', async ({ page }) => {
    await page.goto('/gm/bestiario')

    const busca = page.getByRole('searchbox', { name: 'Buscar criatura' })
    await expect(busca).toBeVisible()
    await busca.fill('ogro')

    const linha = page.getByRole('button', { name: /^Ogro/ }).first()
    await expect(linha).toBeVisible()
    await linha.click()

    await expect(page.getByRole('region', { name: 'Criatura escolhida' })).toContainText('Ogro')
  })
})
