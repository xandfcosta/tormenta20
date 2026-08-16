import { expect, test } from '@playwright/test'

/**
 * O ÚNICO e2e do Hub: prova que o app sobe autenticado e que a tabela de rotas
 * leva o menu para a cena certa — as duas coisas que nenhum teste de integração
 * vê (o `HubMenu` é coberto por `pages/home/hub.test.tsx`, que exercita cada
 * entrada, o rail de navegação e o rodapé; o que ele não tem é router de
 * verdade nem bundle de verdade).
 */
test('o Hub sobe autenticado e o menu leva ao elenco', async ({ page }) => {
  await page.goto('/')
  await expect(page.getByRole('heading', { name: /Tormenta 20/i })).toBeVisible()
  await expect(page.getByText('Ferramentas do Mestre')).toBeVisible()

  await page.getByText('Meus Heróis').click()
  await expect(page).toHaveURL(/\/characters/)
  await expect(page.getByText('Personagens')).toBeVisible()
})
