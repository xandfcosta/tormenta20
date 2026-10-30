import { expect, type Page, test } from '@playwright/test'

/**
 * A FICHA PARALELA RESPONDE AO TOQUE NA HORA (ALE-520) — e os dois mecanismos
 * só existem num navegador: o CICLO do pedido do Datastar (o evento
 * `datastar-fetch` com o elemento que disparou) e o HISTÓRICO da aba.
 *
 * O servidor já responde em ~30ms; o que se sentia era o botão sem retorno
 * nenhum até a resposta, e a troca de aba recarregando a página inteira.
 */
test.use({ storageState: '.auth/user.json' })

const TANQUE = 'Tanque Placas Nv10'

async function tanqueId(page: Page): Promise<string> {
  await page.goto('/personagens')
  const href = await page.getByRole('link', { name: `Abrir ficha de ${TANQUE}` }).first().getAttribute('href')
  const id = href?.match(/\d+/)?.[0]
  expect(id, `não achei o ${TANQUE} no elenco: o resto do caso não mediria nada`).toBeTruthy()
  return id as string
}

test('o botão diz que está trabalhando enquanto o gesto não volta', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 })
  const id = await tanqueId(page)
  await page.goto(`/personagens/${id}?tab=expertises&tempestade=1`)

  // A RESPOSTA ESPERA o teste soltar: é o que separa "marcou enquanto
  // esperava" de "o pedido já tinha voltado quando o teste olhou".
  let release: () => void = () => {}
  const held = new Promise<void>((r) => {
    release = r
  })
  await page.route('**/vitais/pv/**', async (route) => {
    await held
    await route.continue()
  })

  const less = page.locator('.ds-vitals').getByRole('button', { name: 'Menos 1 PV' })
  await less.click()
  await expect(less, 'o botão não marcou o pedido em curso').toHaveAttribute('aria-busy', 'true')

  release()
  await expect(page.locator('.ds-vitals').getByRole('button', { name: 'Menos 1 PV' })).not.toHaveAttribute(
    'aria-busy',
    'true',
  )

  // Devolve o PV, para a semente seguir como estava.
  await page.unroute('**/vitais/pv/**')
  await page.locator('.ds-vitals').getByRole('button', { name: 'Mais 1 PV' }).click()
})
