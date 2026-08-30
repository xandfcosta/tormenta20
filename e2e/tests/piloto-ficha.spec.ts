import { expect, test } from '@playwright/test'
import { expectDentroDaJanela } from './support/geometry'
import { expectNoHorizontalOverflow, VIEWPORTS } from './support/viewports'

/**
 * A FICHA em Datastar (ALE-272, fatia 1) — a casca, as abas e o crachá.
 *
 * O que o servidor escreve está preso em Go, que é mais barato: o endereço das
 * abas, a posse, o degrau de nível e a faixa dos vitais. O que sobra para cá é o
 * que só o navegador mede — LEIAUTE REAL nos seis formatos.
 *
 * E leiaute aqui não é enfeite: o crachá é uma fileira com retrato, identidade,
 * nível e oito botões de vital, e a barra de abas tem sete itens. As duas são
 * exatamente a forma que já transbordou nesta casa (ALE-162, ALE-178) — um
 * controle que sai da janela não é feio, é inalcançável.
 */
test.use({ storageState: '.auth/user.json' })

/** O primeiro herói do elenco, pelo endereço da ficha nova. */
async function aFichaDoPrimeiro(page: import('@playwright/test').Page) {
  await page.goto('/piloto/personagens')
  const href = await page.locator('a[aria-label^="Abrir ficha de"]').first().getAttribute('href')
  const id = href?.match(/\d+/)?.[0]
  expect(id, 'não achei um herói no elenco: o resto do caso não mediria nada').toBeTruthy()
  await page.goto(`/piloto/personagens/${id}`)
  return id as string
}

test('a ficha cabe nos seis formatos, e nenhum botão do crachá sai da janela', async ({ page }) => {
  await aFichaDoPrimeiro(page)
  await expect(page.getByRole('navigation', { name: 'Seções da ficha' })).toBeVisible()

  await expectNoHorizontalOverflow(page, VIEWPORTS)

  // O CRACHÁ é o que mais aperta: oito passos de vital, o degrau de nível e o
  // retrato na mesma fileira. No telefone em pé é onde a conta estoura.
  await page.setViewportSize({ width: 390, height: 844 })
  await expectDentroDaJanela(page)
})

test('as sete abas são endereços, e a ativa se anuncia', async ({ page }) => {
  const id = await aFichaDoPrimeiro(page)

  const abas = page.getByRole('navigation', { name: 'Seções da ficha' }).getByRole('link')
  await expect(abas).toHaveCount(7)

  // A ABA É UM ENDEREÇO: recarregar tem de cair na mesma seção. É o contrato que
  // a SPA tinha e que um sinal de cliente não daria — e é por isso que elas são
  // links e não botões.
  await page.goto(`/piloto/personagens/${id}?tab=spells`)
  await expect(page.getByRole('link', { name: 'Magias', exact: true })).toHaveAttribute('aria-current', 'page')
  await page.reload()
  await expect(page.getByRole('link', { name: 'Magias', exact: true })).toHaveAttribute('aria-current', 'page')

  // E o endereço ANTIGO da Mochila continua chegando nela: `inventory` é
  // favorito de quando a aba se chamava assim.
  await page.goto(`/piloto/personagens/${id}?tab=inventory`)
  await expect(page.getByRole('link', { name: 'Mochila', exact: true })).toHaveAttribute('aria-current', 'page')
})
