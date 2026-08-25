import { expect, test } from '@playwright/test'

/**
 * O BUSCADOR DO LIVRO: ⌃K abre uma caixa que procura nas 1.072 entradas do
 * livro de uma vez, em qualquer cena (ALE-264).
 *
 * E2E porque a garantia é de TECLADO, e teclado só o navegador de verdade mede.
 * Por CDP, `element.focus()` move o `document.activeElement` sem disparar o
 * evento `focus` quando a janela não tem foco do sistema — automação de aba mede
 * silêncio e parece medir ausência de comportamento. Foi assim que este mesmo
 * atalho pareceu quebrado durante a implementação: o ⌃K por CDP não chegava à
 * página, e um evento sintético provou que o ouvinte estava certo o tempo todo.
 *
 * O que ele NÃO precisa medir aqui: o ranqueamento e o corte, que são regra e
 * moram em `piloto_buscador_test.go` — a camada mais barata que os segura.
 */
test.use({ storageState: '.auth/user.json' })

const CAIXA = '#buscador'
const CAMPO = '#buscador-campo'
const ACHADO = '[data-resultado]'

// AMOSTRAGEM e não enumeração: o atalho mora na CASCA, então provar que ele
// funciona em duas cenas de formas diferentes — o Hub (casca título) e o
// bestiário (casca densa, e que já tem busca própria) — vale para as vinte.
const CENAS = [
  { nome: 'hub', url: '/piloto/' },
  { nome: 'bestiário', url: '/piloto/mestre/bestiario' },
]

for (const cena of CENAS) {
  test(`⌃K abre o buscador na cena de ${cena.nome}`, async ({ page }) => {
    await page.setViewportSize({ width: 1400, height: 900 })
    await page.goto(cena.url)

    // O CONTROLE: a caixa existe no documento e está FECHADA. Sem ele, "abriu"
    // seria verdade sobre uma caixa que nasce aberta, e a asserção não mediria
    // o atalho.
    await expect(page.locator(CAIXA)).toHaveCount(1)
    expect(await page.locator(CAIXA).evaluate((d: HTMLDialogElement) => d.open)).toBe(false)

    await page.keyboard.press('Control+k')

    expect(await page.locator(CAIXA).evaluate((d: HTMLDialogElement) => d.open)).toBe(true)
    // O foco tem de cair no CAMPO: uma caixa que abre e não recebe o que se
    // digita é pior que caixa nenhuma — a próxima tecla iria para a cena atrás.
    await expect(page.locator(CAMPO)).toBeFocused()
  })
}

test('a seta desce do campo para o primeiro achado e o Enter abre a cena', async ({ page }) => {
  await page.setViewportSize({ width: 1400, height: 900 })
  await page.goto('/piloto/')
  await page.keyboard.press('Control+k')
  await page.keyboard.type('abalado')

  const primeiro = page.locator(ACHADO).first()
  await expect(primeiro).toHaveText(/Abalado/)

  await page.keyboard.press('ArrowDown')
  await expect(primeiro).toBeFocused()

  await page.keyboard.press('Enter')
  // `waitForURL` e NUNCA `networkidle` depois de uma tecla que navega: o
  // segundo volta ANTES de a navegação começar, e a asserção leria a URL velha.
  await page.waitForURL(/\/piloto\/mestre\/condicoes\?/)
  await expect(page.locator('#buscador')).toHaveCount(1)
})

test('o Enter no campo abre o primeiro achado sem passar pelas setas', async ({ page }) => {
  await page.setViewportSize({ width: 1400, height: 900 })
  await page.goto('/piloto/')
  await page.keyboard.press('Control+k')
  await page.keyboard.type('lobo')

  await expect(page.locator(ACHADO).first()).toHaveText(/Lobo/)
  await page.keyboard.press('Enter')

  await page.waitForURL(/\/piloto\/mestre\/bestiario\?criatura=/)
})

test('o Esc fecha a caixa e devolve a cena', async ({ page }) => {
  await page.setViewportSize({ width: 1400, height: 900 })
  await page.goto('/piloto/mestre/bestiario')
  await page.keyboard.press('Control+k')
  await page.keyboard.type('abal')
  await expect(page.locator(ACHADO).first()).toBeVisible()

  await page.keyboard.press('Escape')

  expect(await page.locator(CAIXA).evaluate((d: HTMLDialogElement) => d.open)).toBe(false)
  // E a cena continua onde estava: o Esc do diálogo não pode subir para o
  // `data-voltar` da casca e levar a pessoa para o Hub.
  expect(page.url()).toContain('/piloto/mestre/bestiario')
})

test('o campo do buscador acende a linha, e não um retângulo colado na caixa', async ({ page }) => {
  await page.setViewportSize({ width: 1400, height: 900 })
  await page.goto('/piloto/mestre/condicoes')
  await page.keyboard.press('Control+k')
  await expect(page.locator(CAMPO)).toBeFocused()

  // O defeito, visto pelo dono na tela: a regra global da casa dá anel dourado a
  // qualquer `input` da cena e VENCE o `outline-none` do utilitário. Num campo
  // com borda isso fica certo; aqui o campo é a linha inteira do diálogo, e o
  // anel saía como um retângulo colado nas bordas.
  //
  // E2E porque a pergunta é de CASCATA: quem ganha entre duas folhas, e um
  // `:has()` que acende o pai. Só o navegador resolve isso.
  const medida = await page.evaluate(() => {
    const campo = document.getElementById('buscador-campo')!
    const linha = campo.closest('.buscador-linha')!
    return {
      anelDoCampo: getComputedStyle(campo).outlineStyle,
      bordaDaLinha: getComputedStyle(linha).borderBottomColor,
      bordaDeFora: getComputedStyle(document.getElementById('buscador')!).borderTopColor,
    }
  })
  expect(medida.anelDoCampo, 'o campo ainda desenha o anel do navegador').toBe('none')
  // O CONTROLE: a linha acende com uma cor DIFERENTE da borda da caixa. Sem ele,
  // "tem borda" seria verdade sobre a borda de sempre.
  expect(medida.bordaDaLinha).not.toBe(medida.bordaDeFora)
})
