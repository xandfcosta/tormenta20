import { expect, test } from '@playwright/test'

/**
 * NADA transborda o cartão (ALE-264).
 *
 * O defeito foi visto pelo dono na tela: a magia "Sopro da Salvação" lista as
 * condições que ela remove como UM TOKEN sem espaço —
 * `(abalado/atordoado/apavorado/…)` com 100 caracteres —, e a barra não é
 * oportunidade de quebra de linha para o navegador. Medido: 1.343px de conteúdo
 * numa coluna de 540px, pintando POR CIMA dos cartões vizinhos.
 *
 * E2E porque a pergunta é de LEIAUTE REAL: quebra de linha depende da fonte, da
 * largura da coluna e do algoritmo do navegador. Em jsdom todo elemento mede
 * zero e este guarda passaria verde sempre.
 *
 * AMOSTRAGEM sobre as abas: a aba que entrar amanhã já nasce medida.
 */
test.use({ storageState: '.auth/user.json' })

const ABAS = ['condicoes', 'magias', 'poderes', 'itens', 'efeitos', 'racas', 'classes', 'deuses']

for (const aba of ABAS) {
  test(`nenhum cartão da aba ${aba} transborda a coluna`, async ({ page }) => {
    await page.setViewportSize({ width: 1400, height: 900 })
    await page.goto(`/piloto/mestre/catalogos?aba=${aba}`)

    const medida = await page.evaluate(() => {
      const cartoes = [...document.querySelectorAll('.acervo-em-colunas > div')]
      return {
        quantos: cartoes.length,
        estouram: cartoes
          .filter((c) => c.scrollWidth > c.clientWidth + 1)
          .slice(0, 3)
          .map((c) => ({
            texto: (c.textContent ?? '').trim().slice(0, 40),
            sobra: c.scrollWidth - c.clientWidth,
          })),
      }
    })

    // O CONTROLE: a aba desenhou cartões. Sem ele, "nada transbordou" seria
    // verdade sobre uma tela vazia.
    expect(medida.quantos, `a aba ${aba} não desenhou cartão nenhum`).toBeGreaterThan(0)
    expect(
      medida.estouram,
      `cartões da aba ${aba} pintam por cima do vizinho`,
    ).toEqual([])
  })
}
