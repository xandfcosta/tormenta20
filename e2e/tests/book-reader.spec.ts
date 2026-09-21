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
  // `count()` ANTES de `getAttribute`, e a ordem é o conserto: `getAttribute`
  // ESPERA o elemento aparecer, então numa bancada sem `LIVRO_PDF` — onde o
  // botão não existe e nunca vai existir — ele consome os 30s e estoura por
  // timeout, deixando o `test.skip` abaixo INALCANÇÁVEL. E o modo de falhar
  // mente sobre a causa: parece leitor quebrado, é bancada sem livro.
  const button = page.locator('a[href*="/livro/ler"]').first()

  if ((await button.count()) === 0) {
    // Sem `LIVRO_PDF` configurado não há botão, e isso é estado LEGÍTIMO. O que
    // não pode existir é meio caminho: link sem livro ou livro sem link.
    expect(await page.locator('a[href*="/livro"]').count()).toBe(0)
    test.skip(true, 'esta bancada não serve o livro (LIVRO_PDF vazio)')
    return
  }

  const address = await button.getAttribute('href')

  expect(address).toMatch(/\?p=\d+&t=Lobo/)
  await page.goto(address)

  // O CONTROLE de que o pdf.js de fato desenhou: um canvas com área. Sem ele,
  // "achei a marca" poderia ser verdade sobre uma página em branco.
  // Esperar o `data-pronto` e não só a visibilidade: um `<canvas>` sem desenhar
  // mede 300×150 (o default do elemento) e passa por "visível" — o instrumento
  // chega antes do render e reprova uma página que está certa.
  const reader = page.locator('#reader[data-pronto]')
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
  if (response?.status() === 404) {
    test.skip(true, 'esta bancada não serve o livro (LIVRO_PDF vazio)')
    return
  }

  await expect(page.locator('#reader[data-pronto]')).toBeAttached({ timeout: 30_000 })
  const caption = page.locator('[data-pagina-atual]')
  await expect(caption).toHaveText(/p290 de \d+/)

  await page.locator('[data-acao="proxima"]').click()
  await expect(caption).toHaveText(/p291 de \d+/)

  await page.locator('[data-acao="anterior"]').click()
  await expect(caption).toHaveText(/p290 de \d+/)
})

test('o livro abre POR CIMA da cena e o fechar devolve a memória', async ({ page }) => {
  await page.setViewportSize({ width: 1400, height: 900 })
  await page.goto('/mestre/condicoes')

  const button = page.locator('a[href*="/livro/ler"]').first()
  if ((await button.count()) === 0) {
    test.skip(true, 'esta bancada não serve o livro (LIVRO_PDF vazio)')
    return
  }

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
    await expect(inside.locator('#reader[data-pronto]')).toBeAttached({ timeout: 30_000 })
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
