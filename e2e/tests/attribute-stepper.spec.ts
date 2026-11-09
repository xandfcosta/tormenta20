import { expect, type Page, test } from '@playwright/test'

/**
 * O `−` E O `+` DO ATRIBUTO PARTEM DO QUE ESTÁ NO CAMPO, e não do valor que a
 * página desenhou. Quem escreve 5 e toca `+` quer 6: com o valor do desenho, o
 * `+` gravava o antigo mais um, e o 5 escrito se perdia.
 *
 * Só um navegador mede isto: o campo escrito sem sair dele, o toque no botão
 * que tira o foco, e os dois pedidos que saem em sequência.
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

/** Abre o diálogo da Força e devolve o campo do valor livre. */
async function strengthField(page: Page, id: string) {
  await page.goto(`/personagens/${id}?tab=combat`)
  await page.getByRole('button', { name: /^Força .*: ver a soma$/ }).first().click()
  const field = page.getByRole('textbox', { name: 'Valor de Força' })
  await expect(field).toBeVisible()
  return field
}

const strengthPost = (r: { url(): string; request(): { method(): string } }) =>
  r.url().includes('/atributos/strength/') && r.request().method() === 'POST'

test('escrever 5 e tocar + grava 6, e não o valor antigo mais um', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 })
  const id = await tanqueId(page)
  let field = await strengthField(page, id)
  const before = await field.inputValue()
  const written = Number(before) + 3

  // O CAMPO É ESCRITO SEM SAIR DELE: o toque no `+` é o que tira o foco, e é
  // aí que os dois pedidos saem — o do campo e o do botão.
  await field.fill(String(written))
  const both = Promise.all([page.waitForResponse(strengthPost), page.waitForResponse(strengthPost)])
  await page.getByRole('button', { name: 'Aumentar Força' }).click()
  await both

  field = await strengthField(page, id)
  await expect(field, `escrito ${written} e tocado o +, a Força livre devia ser ${written + 1}`).toHaveValue(
    String(written + 1),
  )

  // Devolve a Força, para a semente seguir como estava.
  const restored = page.waitForResponse(strengthPost)
  await field.fill(before)
  await field.press('Enter')
  await restored
})
