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
 *
 * ELE SOBREVIVEU A UM APAGAMENTO NA `main`, e a nota fica para o próximo merge
 * não o apagar em silêncio: a ALE-187 podou sete e2e "que não precisavam de
 * browser", e este era um deles — quando ele testava o Hub da SPA. Aqui na base
 * ele testa OUTRA COISA: o desvio do mux levando `/` a uma página do SERVIDOR, e
 * uma navegação de verdade entre duas delas. Isso precisa de browser, e nenhuma
 * outra camada atravessa o caminho inteiro.
 *
 * Quando a migração terminar e a `main` receber tudo, vale reler: se o Hub da
 * SPA já não existir, a poda daquela issue perde o alvo e este arquivo é o que
 * resta.
 */
test('o Hub sobe autenticado e o menu leva ao elenco', async ({ page }) => {
  await page.goto('/')
  await expect(page.getByRole('heading', { name: /Tormenta 20/i })).toBeVisible()
  await expect(page.getByText('Ferramentas do Mestre')).toBeVisible()

  await page.getByText('Meus Heróis').click()
  await expect(page).toHaveURL(/\/personagens$/)
  await expect(page.getByRole('listbox', { name: 'Personagens' })).toBeVisible()
})
