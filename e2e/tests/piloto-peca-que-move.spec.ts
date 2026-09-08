import { expect, type Page, test } from '@playwright/test'
import { abreOTabuleiro, mesaDescartavel, poeUmaPecaNoMapa } from './support/mesa'

/**
 * A PEÇA QUE MOVEU DESLIZA, em vez de teleportar (ALE-174, P3).
 *
 * E2E porque a pergunta é sobre a LINHA DO TEMPO de uma animação disparada por
 * um remendo do servidor — três coisas que não existem fora do navegador. Em
 * jsdom não há `Element.animate`, não há duração e não há morph.
 *
 * # O que se mede é a POSIÇÃO PINTADA, e não a chamada
 *
 * A primeira versão destes guardas embrulhava o `Element.prototype.animate` e
 * afirmava que ele foi chamado com os quadros certos. Determinístico, e ainda
 * assim a pergunta errada: uma animação PEDIDA e uma animação VISTA são coisas
 * diferentes — uma regra de CSS com `transform` de maior peso engoliria a
 * segunda sem tocar na primeira, e o guarda seguiria verde sobre um teleporte.
 *
 * Então a sonda amostra `getBoundingClientRect().x` a cada quadro e conta
 * POSIÇÕES DISTINTAS. Um deslize passa por várias; um teleporte tem duas (antes
 * e depois) e um palco parado tem uma.
 *
 * **Capturar tela não serviria**: o `screenshot()` do Playwright desliga
 * animação por padrão e FINALIZA as finitas antes de fotografar. Medido nesta
 * issue — três quadros tirados a 30, 60 e 90ms saíram byte a byte idênticos, e
 * a leitura ingênua disso seria "a animação não existe".
 */
test.use({ storageState: '.auth/user.json' })

/**
 * Amostra onde a peça é PINTADA, quadro a quadro, durante `ms` milissegundos.
 *
 * ELA É ARMADA DEPOIS DO ARRASTO E ANTES DO CONFIRMAR, e a ordem é o conserto de
 * um guarda que passava verde sobre o defeito. Armada antes do gesto inteiro,
 * ela media o ARRASTO: o dedo atravessa quatro casas e pinta a peça em cada uma,
 * então "mais de duas posições" era verdade sem módulo nenhum carregado —
 * 630 → 674 → 718 → 762 → 806, que é o dedo e não a animação.
 *
 * Depois do arrasto não há corrida: a animação só começa quando o remendo do
 * servidor chega, e isso é depois do clique.
 */
async function posicoesPintadas(page: Page, gesto: () => Promise<void>, ms = 800): Promise<number[]> {
  await page.evaluate((limite) => {
    const w = window as unknown as { __x: number[] }
    w.__x = []
    const inicio = performance.now()
    const passo = () => {
      const peca = document.querySelector('.tabuleiro-peca')
      if (peca) w.__x.push(Math.round(peca.getBoundingClientRect().x))
      if (performance.now() - inicio < limite) requestAnimationFrame(passo)
    }
    requestAnimationFrame(passo)
  }, ms)
  await gesto()
  await page.waitForTimeout(ms + 100)
  const xs = await page.evaluate(() => (window as unknown as { __x: number[] }).__x)
  return [...new Set(xs)]
}

/** Arrasta a peça `casas` para a direita, deixando o movimento PROPOSTO. */
async function arrasta(page: Page, casas: number): Promise<void> {
  const peca = page.locator('.tabuleiro-peca').first()
  const caixa = await peca.boundingBox()
  if (!caixa) throw new Error('a peça não tem caixa: o arrasto não tem de onde partir')
  const x = caixa.x + caixa.width / 2
  const y = caixa.y + caixa.height / 2
  await page.mouse.move(x, y)
  await page.mouse.down()
  // PASSOS INTERMEDIÁRIOS, como o guarda da seta viva: a prévia é pedida a cada
  // casa atravessada, e um salto direto não constrói caminho nenhum.
  for (let i = 1; i <= casas; i++) await page.mouse.move(x + i * caixa.width, y)
  await page.mouse.up()
}

/** Arrasta, arma a sonda e confirma — nesta ordem, ver `posicoesPintadas`. */
async function deslizeAoConfirmar(page: Page, casas: number): Promise<number[]> {
  await arrasta(page, casas)
  return posicoesPintadas(page, () => page.getByRole('button', { name: 'Confirmar' }).click())
}

async function umTabuleiroComUmaPeca(page: Page) {
  const mesa = await mesaDescartavel(page)
  await abreOTabuleiro(page, mesa.mesa)
  await poeUmaPecaNoMapa(page)
  await page.getByLabel('Centralizar nas peças').click()
  return mesa
}

test('confirmar um movimento desliza a peça em vez de teleportá-la', async ({ page }) => {
  const { apagar } = await umTabuleiroComUmaPeca(page)
  try {
    const posicoes = await deslizeAoConfirmar(page, 4)

    // TRÊS é o piso do que se pode chamar de deslize: origem, destino e ao menos
    // um lugar no meio. Um teleporte dá exatamente duas.
    expect(
      posicoes.length,
      `a peça foi pintada em ${posicoes.length} posições (${posicoes.join(' → ')}): ` +
        'com duas ou menos ela teleportou, que é o defeito que a ALE-174 existe para consertar',
    ).toBeGreaterThan(2)
  } finally {
    await apagar()
  }
})

/**
 * O ZOOM NÃO É MOVIMENTO, e este caso é a razão de o disparo ser a mudança de
 * `--col`/`--lin` e não uma transição de `left`/`top`.
 *
 * A peça é posicionada por `calc(var(--col) * var(--quadrado))`, e o zoom muda o
 * `--quadrado`. Uma transição de CSS faria as nove peças escorregarem ao
 * aproximar — um movimento que ninguém fez, e que a mesa leria como alguém
 * tendo andado.
 *
 * A issue previa o perigo e nomeou a causa errada: ela dizia que era o PAN, que
 * na verdade é `transform` do contêiner e não toca na peça. Este guarda mede a
 * causa de verdade.
 */
test('aproximar o mapa não faz as peças deslizarem', async ({ page }) => {
  const { apagar } = await umTabuleiroComUmaPeca(page)
  try {
    // O CONTROLE vem primeiro, e é a metade que importa: um movimento de
    // verdade TEM de deslizar nesta mesma página. Sem ele, "o zoom não animou"
    // seria verdade também sobre uma sonda que não está amostrando nada.
    const movendo = await deslizeAoConfirmar(page, 3)
    expect(movendo.length, 'o CONTROLE não deslizou: a sonda não está medindo').toBeGreaterThan(2)

    const aproximando = await posicoesPintadas(page, () => page.getByLabel('Aproximar').click(), 600)
    expect(
      aproximando.length,
      `aproximar pintou a peça em ${aproximando.length} posições (${aproximando.join(' → ')}): ` +
        'ela está presa ao TAMANHO da casa e não à casa, e a mesa vê um movimento que ninguém fez',
    ).toBeLessThanOrEqual(2)
  } finally {
    await apagar()
  }
})
