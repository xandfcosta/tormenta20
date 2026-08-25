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
    await page.goto(`/piloto/mestre/${aba}`)

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

test('o elo mostra o conceito por cima, sem tirar a pessoa da regra que lia', async ({ page }) => {
  await page.setViewportSize({ width: 1400, height: 900 })
  await page.goto('/piloto/mestre/condicoes')

  // A condição Abalado termina em "Medo." — o tipo de efeito, que é outro
  // verbete. É o caso que o dono trouxe.
  // Pelo `title` e não pelo nome acessível: o texto do elo é "Medo." (com o
  // ponto do livro), e é ele que vira o nome — o `title` é a explicação.
  const elo = page.locator('a[title="Ver Medo"]').first()
  await expect(elo).toBeVisible()

  const caixa = page.locator('#verbete-em-dialogo')
  expect(await caixa.evaluate((d: HTMLDialogElement) => d.open)).toBe(false)

  await elo.click()

  expect(await caixa.evaluate((d: HTMLDialogElement) => d.open)).toBe(true)
  await expect(caixa).toContainText('Medo capaz de prejudicar o alvo')
  // A CENA CONTINUA: o endereço não mudou e a condição que se estava lendo está
  // lá atrás. Era isso que a navegação para uma busca destruía.
  expect(page.url()).toContain('/piloto/mestre/condicoes')
  await expect(page.getByText('-2 em testes de perícia.')).toBeVisible()

  await page.keyboard.press('Escape')
  expect(await caixa.evaluate((d: HTMLDialogElement) => d.open)).toBe(false)
})
