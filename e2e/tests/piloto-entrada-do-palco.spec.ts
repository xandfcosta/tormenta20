import { expect, test } from '@playwright/test'

/**
 * A ENTRADA DO PALCO nas DUAS cenas de seleção (ALE-235, ALE-297).
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
 *
 * # POR QUE DUAS CENAS, E POR QUE NÃO A MESMA COISA DUAS VEZES
 *
 * A ALE-297 tirou o livro de couro das campanhas e pôs o mesmo palco lá. O
 * mecanismo é UM só e ele é prendido UMA vez — o caso da direção e o do atraso
 * medem o CSS, e o CSS é o mesmo para as duas. O que cada cena tem de provar
 * separado é a LIGAÇÃO: que ela escreve a classe no nó certo e que o cursor de
 * verdade a faz tocar. Por isso o primeiro caso varre as duas e o segundo, que
 * mede os 80ms e a opacidade final, roda numa só.
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

// As duas cenas que têm palco. Cena nova entra aqui e nasce medida — é a mesma
// enumeração do guarda de Go, e ela é remendo pelo mesmo motivo: nada IMPEDE uma
// terceira cena de nascer fora da lista.
const SCENES_WITH_A_STAGE = [
  { nome: 'personagens', url: '/personagens' },
  { nome: 'campanhas', url: '/campanhas' },
]

for (const cena of SCENES_WITH_A_STAGE) {
  test(`o palco de ${cena.nome} entra pelo lado para onde o cursor foi`, async ({ page }) => {
    await page.setViewportSize({ width: 1400, height: 900 })
    await page.goto(cena.url)
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
}

/**
 * O PALCO NÃO DANÇA AO ANDAR NO TRILHO (ALE-99, e de novo na ALE-297).
 *
 * A capa tem de pousar no MESMO y em toda posição do cursor. Quando ela não
 * pousa, andar no trilho faz a cena inteira saltar debaixo do ponteiro — e o
 * salto é pequeno o bastante para ninguém chamar de defeito e grande o bastante
 * para cansar.
 *
 * ESTE GUARDA NASCEU VERMELHO, na bancada e contra a seed de verdade: a capa
 * pousava em y=126 nas quatro campanhas de sinopse com duas linhas e em y=136
 * nas três de uma linha. A causa era a sinopse ser texto do mestre, de altura
 * livre, num palco que é coluna centralizada — cada linha a mais empurra tudo
 * em volta. O conserto foi a caixa de duas linhas fixas (`synopsisBox`).
 *
 * Ele varre as DUAS cenas porque a forma é a mesma e o risco também: qualquer
 * campo de altura livre entre a capa e as ações reabre o defeito.
 *
 * E2E porque a pergunta é sobre LEIAUTE REAL — quantas linhas um texto ocupa
 * numa largura, e onde isso põe os irmãos. Em jsdom todo elemento mede zero e a
 * resposta seria "não dança" para qualquer código.
 */
for (const cena of SCENES_WITH_A_STAGE) {
  test(`a capa de ${cena.nome} pousa no mesmo y em toda posição do trilho`, async ({ page }) => {
    await page.setViewportSize({ width: 1400, height: 900 })
    await page.goto(cena.url)
    await page.waitForLoadState('networkidle')

    const opcoes = page.locator('[role="option"]')
    const quantas = await opcoes.count()
    // O CONTROLE: menos de três posições e "não dançou" seria verdade sobre uma
    // cena que quase não tem por onde andar.
    expect(quantas, 'o trilho é curto demais para o guarda dizer alguma coisa').toBeGreaterThanOrEqual(3)

    const topos: number[] = []
    for (let i = 0; i < quantas; i++) {
      await opcoes.nth(i).click()
      // A entrada dura 220ms + 80ms de atraso da placa; medir antes pegaria o
      // palco no meio do `translateX` e o número seria da animação, não do
      // leiaute.
      await page.waitForTimeout(350)
      topos.push(
        await page.evaluate(() => {
          const palcos = [...document.querySelectorAll('[data-show^="$cursor =="]')]
          const ativo = palcos.find((p) => p.getBoundingClientRect().height > 0)
          if (!ativo) return -1
          return Math.round(ativo.querySelector('.palco-retrato')!.getBoundingClientRect().top)
        }),
      )
    }

    expect(
      [...new Set(topos)],
      `a capa pousou em alturas diferentes ao percorrer o trilho: ${topos.join(', ')}`,
    ).toHaveLength(1)
  })
}

/**
 * QUEM PEDE MENOS MOVIMENTO NÃO RECEBE NENHUM.
 *
 * A regra é declarativa (`@media (prefers-reduced-motion: reduce)` sobre as
 * mesmas classes), então ela vale para as duas cenas de uma vez e é medida numa.
 *
 * O CONTROLE é a metade que importa: o mesmo gesto, no mesmo endereço, SEM a
 * preferência, tem de tocar. Sem ele, "nada animou" é indistinguível de "o
 * clique não chegou" — e as duas passam verde.
 */
test('sob movimento reduzido o palco troca sem animar', async ({ browser }) => {
  const medir = async (reducedMotion: 'reduce' | 'no-preference') => {
    const ctx = await browser.newContext({ storageState: '.auth/user.json', reducedMotion })
    const page = await ctx.newPage()
    try {
      await page.setViewportSize({ width: 1400, height: 900 })
      await page.goto('/personagens')
      await page.waitForLoadState('networkidle')
      const quadros = page.locator('[role="option"]')
      return await animacoesDoGesto(page, () => quadros.nth(2).click())
    } finally {
      // Limpeza com `catch`: ela não pode falar mais alto que o defeito (ALE-245).
      await ctx.close().catch(() => {})
    }
  }

  expect(await medir('no-preference'), 'o CONTROLE não animou — o gesto não chegou').toContain(
    'palcoEntraAdiante',
  )
  expect(await medir('reduce'), 'quem pediu menos movimento recebeu a entrada mesmo assim').toEqual(
    [],
  )
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
