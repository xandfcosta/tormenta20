import { expect, test } from '@playwright/test'

/**
 * O LEITOR DO LIVRO: o verbete aberto na página certa e DESTACADO.
 *
 * E2E porque não há outra testemunha. O que se afirma aqui é que o pdf.js
 * carregou num WORKER, pediu faixas de um PDF de 89 MB, desenhou uma página num
 * CANVAS e posicionou as marcas sobre o texto — nada disso existe em jsdom, onde
 * canvas mede zero e worker não roda.
 *
 * O visualizador nativo do navegador não serve: o Chrome IGNORA `#search=` e
 * transfere o arquivo inteiro para abrir uma página.
 */
test.use({ storageState: '.auth/user.json' })

test('o botão do bestiário abre o livro na página do verbete, com o nome marcado', async ({
  page,
}) => {
  await page.setViewportSize({ width: 1400, height: 900 })
  await page.goto('/mestre/bestiario?criatura=lobo')

  // O endereço vem da CENA e não é escrito aqui: se a página do Lobo mudar no
  // catálogo, este guarda continua medindo o que a tela oferece — e não um
  // número que eu teria de lembrar de atualizar.
  const button = page.locator('a[href*="/livro/ler"]').first()
  await expect(button, 'a bancada não serve o livro: o `cmd/testbook` rodou?').toBeAttached()

  const address = await button.getAttribute('href')

  expect(address).toMatch(/\?p=\d+&t=Lobo/)
  await page.goto(address)

  // O CONTROLE de que o pdf.js de fato desenhou: um canvas com área. Sem ele,
  // "achei a marca" poderia ser verdade sobre uma página em branco.
  // Esperar o `data-ready` e não só a visibilidade: um `<canvas>` sem desenhar
  // mede 300×150 (o default do elemento) e passa por "visível" — o instrumento
  // chega antes do render e reprova uma página que está certa.
  const reader = page.locator('#reader[data-ready]')
  await expect(reader).toBeAttached({ timeout: 30_000 })

  const canvasEl = page.locator('#reader canvas')
  const box = await canvasEl.boundingBox()
  expect(box?.width ?? 0).toBeGreaterThan(200)
  expect(box?.height ?? 0).toBeGreaterThan(200)

  // E as marcas: elas só existem se o texto da página foi lido e o termo casou.
  const marks = page.locator('.reader-mark')
  await expect.poll(async () => await marks.count(), { timeout: 15_000 }).toBeGreaterThan(0)

  // A marca cai DENTRO da página desenhada. Uma marca fora do canvas seria a
  // transformação de coordenadas errada — o destaque existiria no DOM e não
  // estaria sobre palavra nenhuma.
  const marker = await marks.first().boundingBox()
  expect(marker).not.toBeNull()
  expect(marker!.x).toBeGreaterThanOrEqual(box!.x - 1)
  expect(marker!.x + marker!.width).toBeLessThanOrEqual(box!.x + box!.width + 1)
})

test('as setas andam pelo livro e a barra diz a página impressa', async ({ page }) => {
  await page.setViewportSize({ width: 1400, height: 900 })
  const response = await page.goto('/livro/ler?p=290&t=Lobo')
  expect(response?.status(), 'a bancada não serve o livro: o `cmd/testbook` rodou?').toBe(200)

  await expect(page.locator('#reader[data-ready]')).toBeAttached({ timeout: 30_000 })
  const caption = page.locator('[data-current-page]')
  await expect(caption).toHaveText(/p290 de \d+/)

  await page.locator('[data-action="next"]').click()
  await expect(caption).toHaveText(/p291 de \d+/)

  await page.locator('[data-action="previous"]').click()
  await expect(caption).toHaveText(/p290 de \d+/)
})

test('o livro abre POR CIMA da cena e o fechar devolve a memória', async ({ page }) => {
  await page.setViewportSize({ width: 1400, height: 900 })
  await page.goto('/mestre/condicoes')

  const button = page.locator('a[href*="/livro/ler"]').first()
  await expect(button, 'a bancada não serve o livro: o `cmd/testbook` rodou?').toBeAttached()

  const dialog = page.locator('#book-in-dialog')
  // O CONTROLE: a moldura existe e nasce VAZIA. É isso que faz uma cena que
  // nunca abre o livro não pagar um byte de pdf.js.
  await expect(dialog).toBeAttached()
  expect(await dialog.locator('iframe').getAttribute('src')).toBeNull()

  // DUAS aberturas seguidas, e não uma: a segunda é a que prova que fechar não
  // deixou o leitor num estado que impede a próxima.
  for (const index of [0, 3]) {
    await page.locator('a[href*="/livro/ler"]').nth(index).click()
    expect(await dialog.evaluate((d: HTMLDialogElement) => d.open)).toBe(true)
    // A CENA CONTINUA ATRÁS — é a diferença entre isto e a aba nova: a fila da
    // iniciativa e os filtros do acervo ficam onde estavam.
    await expect(page.locator('[data-slot="scene-content"]')).toBeVisible()

    const inside = dialog.frameLocator('iframe')
    await expect(inside.locator('#reader[data-ready]')).toBeAttached({ timeout: 30_000 })
    await expect(inside.locator('.reader-mark').first()).toBeAttached()

    await dialog.locator('button[aria-label="Fechar o livro"]').click()
    expect(await dialog.evaluate((d: HTMLDialogElement) => d.open)).toBe(false)
    // E o documento do livro é DESCARTADO. Sem isto o worker do pdf.js e o
    // bitmap da página ficariam vivos até a navegação seguinte — a conta de
    // memória que o iframe existe para poder zerar.
    await expect
      .poll(async () => await dialog.locator('iframe').evaluate((f: HTMLIFrameElement) => f.src))
      .toBe('about:blank')
  }
})
