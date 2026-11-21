import { expect, type Page, test } from '@playwright/test'
import { tidyUp } from './support/tidy'

/**
 * O `−` E O `+` DO ATRIBUTO PARTEM DO QUE ESTÁ NO CAMPO, e não do valor que a
 * página desenhou. Quem escreve 5 e toca `+` quer 6: com o valor do desenho, o
 * `+` gravava o antigo mais um, e o 5 escrito se perdia.
 *
 * Só um navegador mede isto: o campo escrito sem sair dele, o toque no botão
 * que NÃO tira o foco do campo, e o pedido único que sai dele.
 */
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

/** A resposta do POST que grava a Força em `value`, e não qualquer uma. */
const strengthPostOf = (value: number) => (r: { url(): string; request(): { method(): string } }) =>
  strengthPost(r) && r.url().includes(`/atributos/strength/${value}?`)

/**
 * O TOQUE NO `+` É O ÚNICO PEDIDO. O campo grava quando perde o foco, e o `+` partia do que estava escrito: se o
 * toque tirasse o foco, saíam dois pedidos juntos — o do campo (7) e o do botão (8) —, e o servidor gravava os dois
 * na ordem em que terminassem. O rastro da CI pegou o 8 terminando antes do 7, e a Força ficando em 7. Por isso o
 * caso CONTA os pedidos, e não só o valor final: dois pedidos é o defeito, mesmo quando a sorte acerta a ordem.
 *
 * Com o mouse e com o toque, porque o caminho do foco é outro em cada um.
 */
for (const pointer of ['mouse', 'touch'] as const) {
  test(`escrever um valor e tocar + grava o valor mais um, num pedido só (${pointer})`, async ({ browser }) => {
    const context = await browser.newContext({
      storageState: '.auth/user.json',
      viewport: { width: 390, height: 844 },
      hasTouch: pointer === 'touch',
    })
    const page = await context.newPage()
    const id = await tanqueId(page)
    let field = await strengthField(page, id)
    const before = await field.inputValue()
    const written = Number(before) + 3
    const posted: string[] = []
    page.on('request', (r) => {
      if (r.method() === 'POST' && r.url().includes('/atributos/strength/')) posted.push(r.url().match(/\/atributos\/strength\/(-?\d+)/)?.[1] ?? '?')
    })

    try {
      await field.fill(String(written))
      const plus = page.getByRole('button', { name: 'Aumentar Força' })
      const answered = page.waitForResponse(strengthPostOf(written + 1))
      if (pointer === 'touch') await plus.tap()
      else await plus.click()
      await answered
      expect(posted, `escrito ${written} e tocado o +, saíram os pedidos ${posted.join(', ')}; want só o ${written + 1}`).toEqual([
        String(written + 1),
      ])

      // O CAMPO, AINDA FOCADO, MOSTRA O VALOR NOVO: é o que ele grava quando o foco sair dele. Se o redesenho
      // poupasse o campo focado, a saída gravaria o valor escrito por cima do que o `+` gravou.
      await expect(field, 'o campo focado ainda mostra o valor escrito depois do +').toHaveValue(String(written + 1))

      field = await strengthField(page, id)
      await expect(field, `escrito ${written} e tocado o +, a Força livre devia ser ${written + 1}`).toHaveValue(
        String(written + 1),
      )
    } finally {
      await tidyUp(`a Força do ${TANQUE} fora do valor da semente`, async () => {
        field = await strengthField(page, id)
        if ((await field.inputValue()) === before) return
        const restored = page.waitForResponse(strengthPost)
        await field.fill(before)
        await field.press('Enter')
        await restored
      })
      await context.close()
    }
  })
}
