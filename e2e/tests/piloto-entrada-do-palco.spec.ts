import { expect, test } from '@playwright/test'

/**
 * A ENTRADA DO PALCO na cena de personagens (ALE-235).
 *
 * E2E, e só e2e: linha do tempo de animação é coisa que só o navegador tem —
 * em jsdom não há `animationstart`, nem duração, nem atraso. O que o servidor
 * escreve (as classes e o gesto que diz o sentido) está preso em Go, que é mais
 * barato; o que sobra para cá é a única pergunta que o Go não responde — **a
 * animação TOCA quando o cursor anda?**
 *
 * Por que ela precisava de issue própria: na SPA quem animava era `animate-in`,
 * que dispara no MOUNT, e o `<Show keyed>` reconstruía o nó a cada troca
 * justamente para isso (ALE-97). Aqui a cena inteira é desenhada e o cursor só
 * alterna `data-show` — nada nunca monta. O que substitui o mount é a CLASSE
 * entrando num nó que não a tinha.
 *
 * O CLIQUE É REAL de ponta a ponta, e isso não é preciosismo: `element.click()`
 * por JS **não move o foco**, então ele dispara só o `click` e não o `focusin`.
 * Foi assim que o defeito da direção passou despercebido na primeira medição —
 * com os dois eventos, a segunda passagem do gesto recalculava o sentido com o
 * índice já atualizado e o palco entrava sempre "adiante".
 */
test.use({ storageState: '.auth/user.json' })

/** Arma a escuta ANTES do gesto e devolve o que tocou. */
async function animacoesDoGesto(page: import('@playwright/test').Page, gesto: () => Promise<void>) {
  await page.evaluate(() => {
    ;(window as unknown as { __anim: string[] }).__anim = []
    document.addEventListener(
      'animationstart',
      (e) => (window as unknown as { __anim: string[] }).__anim.push((e as AnimationEvent).animationName),
      true,
    )
  })
  await gesto()
  // A animação dura 220ms e a placa começa 80ms depois dela.
  await page.waitForTimeout(500)
  return page.evaluate(() => (window as unknown as { __anim: string[] }).__anim)
}

test('o palco entra pelo lado para onde o cursor foi', async ({ page }) => {
  await page.setViewportSize({ width: 1400, height: 900 })
  await page.goto('/personagens')
  await page.waitForLoadState('networkidle')

  const quadros = page.locator('[role="option"]')
  // O CONTROLE: com menos de três quadros não há como andar para frente E para
  // trás, e "a direção não mudou" seria verdade sobre uma cena que não tem para
  // onde andar — a mensagem apontaria o lugar errado.
  expect(await quadros.count()).toBeGreaterThanOrEqual(3)

  const adiante = await animacoesDoGesto(page, () => quadros.nth(2).click())
  expect(adiante, 'andar para frente no trilho toca a entrada pela direita').toContain(
    'palcoEntraAdiante',
  )
  // A PLACA sobe junto, e é ela que carrega o atraso: sem ela o palco inteiro
  // desliza como um bloco só.
  expect(adiante, 'a placa não subiu').toContain('placaSobe')

  const atras = await animacoesDoGesto(page, () => quadros.nth(0).click())
  expect(atras, 'voltar no trilho tem de entrar pelo outro lado').toContain('palcoEntraAtras')
  expect(atras, 'voltar tocou a entrada de ir adiante').not.toContain('palcoEntraAdiante')
})

test('o palco que entra é o que o cursor escolheu, e a placa espera o retrato', async ({
  page,
}) => {
  await page.setViewportSize({ width: 1400, height: 900 })
  await page.goto('/personagens')
  await page.waitForLoadState('networkidle')

  await page.locator('[role="option"]').nth(1).click()

  // A animação vive no palco VISÍVEL, e o `both` a deixa no estado final — um
  // palco preso no primeiro quadro (transparente, deslocado) seria a falha que
  // este caso existe para pegar.
  const medida = await page.evaluate(() => {
    const palcos = [...document.querySelectorAll('[data-show^="$cursor =="]')]
    const ativo = palcos.find((p) => p.getBoundingClientRect().height > 0)
    if (!ativo) return null
    const retrato = ativo.querySelector('.palco-retrato')
    const placa = ativo.querySelector('.palco-placa')
    if (!retrato || !placa) return null
    const cs = (el: Element) => {
      const s = getComputedStyle(el)
      return { nome: s.animationName, atraso: s.animationDelay, opacidade: s.opacity }
    }
    return { retrato: cs(retrato), placa: cs(placa) }
  })

  expect(medida, 'o palco ativo não tem as duas partes que animam').not.toBeNull()
  expect(medida?.retrato.nome).toMatch(/^palcoEntra/)
  expect(medida?.placa.nome).toBe('placaSobe')
  // Os 80ms são metade do efeito: o retrato chega primeiro e o nome pousa em
  // cima dele.
  expect(medida?.placa.atraso).toBe('0.08s')
  // E nada fica preso transparente depois que a animação termina.
  await expect
    .poll(
      async () =>
        page.evaluate(() => {
          const palcos = [...document.querySelectorAll('[data-show^="$cursor =="]')]
          const ativo = palcos.find((p) => p.getBoundingClientRect().height > 0)
          const placa = ativo?.querySelector('.palco-placa')
          return placa ? getComputedStyle(placa).opacity : '0'
        }),
      { timeout: 3000 },
    )
    .toBe('1')
})
