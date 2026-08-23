import { expect, test } from '@playwright/test'

/**
 * O ÚNICO e2e do Hub, e o que ele guarda hoje é a PARTIDA: chegar em `/` sem
 * dizer mais nada e cair autenticado no Hub, que desde a ALE-231 é uma página
 * do servidor alcançada por desvio no mux — nenhuma outra camada atravessa esse
 * caminho inteiro.
 *
 * O clique no menu ficou junto porque ele é uma navegação de VERDADE entre duas
 * páginas do servidor. O `piloto-datastar.spec.ts` afirma os destinos do menu
 * pelas setas; aqui se afirma que o clique chega lá e que a cena desenhou.
 *
 * (A referência antiga a `pages/home/hub.test.tsx` saiu: aquele arquivo foi
 * apagado quando o Hub virou servidor, e a menção sobreviveu ao arquivo.)
 */
test('o Hub sobe autenticado e o menu leva ao elenco', async ({ page }) => {
  await page.goto('/')
  await expect(page.getByRole('heading', { name: /Tormenta 20/i })).toBeVisible()
  await expect(page.getByText('Ferramentas do Mestre')).toBeVisible()

  await page.getByText('Meus Heróis').click()
  await expect(page).toHaveURL(/\/piloto\/personagens$/)
  await expect(page.getByRole('listbox', { name: 'Personagens' })).toBeVisible()
})
