import { expect, test } from '@playwright/test'

/**
 * O LEITOR DO LIVRO: o verbete aberto na página certa e DESTACADO (ALE-264).
 *
 * E2E porque não há outra testemunha. O que se afirma aqui é que o pdf.js
 * carregou num WORKER, pediu faixas de um PDF de 89 MB, desenhou uma página num
 * CANVAS e posicionou as marcas sobre o texto — nada disso existe em jsdom, onde
 * canvas mede zero e worker não roda.
 *
 * Ele nasceu porque o visualizador do navegador não serve: medido, o Chrome
 * IGNORA `#search=` (o dono conferiu na tela) e transfere o arquivo inteiro para
 * abrir uma página.
 */
test.use({ storageState: '.auth/user.json' })

test('o botão do bestiário abre o livro na página do verbete, com o nome marcado', async ({
  page,
}) => {
  await page.setViewportSize({ width: 1400, height: 900 })
  await page.goto('/piloto/mestre/bestiario?criatura=lobo')

  // O endereço vem da CENA e não é escrito aqui: se a página do Lobo mudar no
  // catálogo, este guarda continua medindo o que a tela oferece — e não um
  // número que eu teria de lembrar de atualizar.
  // `count()` ANTES de `getAttribute`, e a ordem é o conserto: `getAttribute`
  // ESPERA o elemento aparecer, então numa bancada sem `LIVRO_PDF` — onde o
  // botão não existe e nunca vai existir — ele consumia os 30s do teste e
  // estourava por timeout. O `test.skip` logo abaixo era código INALCANÇÁVEL,
  // e o modo de falhar mentia sobre a causa: parecia leitor quebrado, era
  // bancada sem livro. `count()` resolve na hora, com zero.
  const botao = page.locator('a[href*="/piloto/livro/ler"]').first()

  if ((await botao.count()) === 0) {
    // Sem `LIVRO_PDF` configurado não há botão, e isso é estado LEGÍTIMO. O que
    // não pode existir é meio caminho: link sem livro ou livro sem link.
    expect(await page.locator('a[href*="/piloto/livro"]').count()).toBe(0)
    test.skip(true, 'esta bancada não serve o livro (LIVRO_PDF vazio)')
    return
  }

  const endereco = await botao.getAttribute('href')

  expect(endereco).toMatch(/\?p=\d+&t=Lobo/)
  await page.goto(endereco)

  // O CONTROLE de que o pdf.js de fato desenhou: um canvas com área. Sem ele,
  // "achei a marca" poderia ser verdade sobre uma página em branco.
  // Esperar o `data-pronto` e não só a visibilidade: um `<canvas>` sem
  // desenhar mede 300×150 (o default do elemento) e passa por "visível". A
  // primeira versão deste guarda mediu exatamente isso e reprovou uma página
  // que estava certa — o instrumento chegou antes do render.
  const leitor = page.locator('#leitor[data-pronto]')
  await expect(leitor).toBeAttached({ timeout: 30_000 })

  const tela = page.locator('#leitor canvas')
  const caixa = await tela.boundingBox()
  expect(caixa?.width ?? 0).toBeGreaterThan(200)
  expect(caixa?.height ?? 0).toBeGreaterThan(200)

  // E as marcas: elas só existem se o texto da página foi lido e o termo casou.
  const marcas = page.locator('.leitor-marca')
  await expect.poll(async () => await marcas.count(), { timeout: 15_000 }).toBeGreaterThan(0)

  // A marca cai DENTRO da página desenhada. Uma marca fora do canvas seria a
  // transformação de coordenadas errada — o destaque existiria no DOM e não
  // estaria sobre palavra nenhuma.
  const marca = await marcas.first().boundingBox()
  expect(marca).not.toBeNull()
  expect(marca!.x).toBeGreaterThanOrEqual(caixa!.x - 1)
  expect(marca!.x + marca!.width).toBeLessThanOrEqual(caixa!.x + caixa!.width + 1)
})

test('as setas andam pelo livro e a barra diz a página impressa', async ({ page }) => {
  await page.setViewportSize({ width: 1400, height: 900 })
  const resposta = await page.goto('/piloto/livro/ler?p=290&t=Lobo')
  if (resposta?.status() === 404) {
    test.skip(true, 'esta bancada não serve o livro (LIVRO_PDF vazio)')
    return
  }

  await expect(page.locator('#leitor[data-pronto]')).toBeAttached({ timeout: 30_000 })
  const rotulo = page.locator('[data-pagina-atual]')
  await expect(rotulo).toHaveText(/p290 de \d+/)

  await page.locator('[data-acao="proxima"]').click()
  await expect(rotulo).toHaveText(/p291 de \d+/)

  await page.locator('[data-acao="anterior"]').click()
  await expect(rotulo).toHaveText(/p290 de \d+/)
})

test('o livro abre POR CIMA da cena e o fechar devolve a memória', async ({ page }) => {
  await page.setViewportSize({ width: 1400, height: 900 })
  await page.goto('/piloto/mestre/condicoes')

  const botao = page.locator('a[href*="/piloto/livro/ler"]').first()
  if ((await botao.count()) === 0) {
    test.skip(true, 'esta bancada não serve o livro (LIVRO_PDF vazio)')
    return
  }

  const dialogo = page.locator('#livro-em-dialogo')
  // O CONTROLE: a moldura existe e nasce VAZIA. É isso que faz uma cena que
  // nunca abre o livro não pagar um byte de pdf.js.
  await expect(dialogo).toBeAttached()
  expect(await dialogo.locator('iframe').getAttribute('src')).toBeNull()

  // DUAS aberturas seguidas, e não uma: a segunda é a que prova que fechar não
  // deixou o leitor num estado que impede a próxima. Sondando por CDP eu vi a
  // segunda falhar e quase consertei um defeito que não existia — o que estava
  // quebrado era a página em que eu tinha mexido à mão.
  for (const indice of [0, 3]) {
    await page.locator('a[href*="/piloto/livro/ler"]').nth(indice).click()
    expect(await dialogo.evaluate((d: HTMLDialogElement) => d.open)).toBe(true)
    // A CENA CONTINUA ATRÁS — é a diferença entre isto e a aba nova: a fila da
    // iniciativa e os filtros do acervo ficam onde estavam.
    await expect(page.locator('[data-slot="scene-content"]')).toBeVisible()

    const dentro = dialogo.frameLocator('iframe')
    await expect(dentro.locator('#leitor[data-pronto]')).toBeAttached({ timeout: 30_000 })
    await expect(dentro.locator('.leitor-marca').first()).toBeAttached()

    await dialogo.locator('button[aria-label="Fechar o livro"]').click()
    expect(await dialogo.evaluate((d: HTMLDialogElement) => d.open)).toBe(false)
    // E o documento do livro é DESCARTADO. Sem isto o worker do pdf.js e o
    // bitmap da página ficariam vivos até a navegação seguinte — a conta de
    // memória que o iframe existe para poder zerar.
    await expect
      .poll(async () => await dialogo.locator('iframe').evaluate((f: HTMLIFrameElement) => f.src))
      .toBe('about:blank')
  }
})
