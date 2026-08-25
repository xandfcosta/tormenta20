import { expect, type Page, test } from '@playwright/test'

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

/**
 * As cenas a visitar saem do TRILHO, lidas da página — não de uma lista aqui.
 *
 * A lista escrita à mão tinha oito entradas e o trilho ganhou duas (escolas de
 * magia e perícias) sem que ela soubesse: as duas cenas novas nasceriam sem
 * medição, em silêncio, que é a marca desta família (ALE-252). Lendo o trilho, a
 * décima primeira já entra medida.
 */
async function cenasDosCatalogos(page: Page): Promise<string[]> {
  await page.goto('/piloto/mestre/condicoes')
  const enderecos = await page
    .locator('nav[aria-label="Ferramentas do mestre"] a')
    .evaluateAll((links) =>
      links.map((a) => (a as HTMLAnchorElement).getAttribute('href') ?? ''),
    )
  // As FERRAMENTAS ficam de fora: encontros e improviso não desenham cartão de
  // acervo, e cobrar transbordo delas mediria outra coisa.
  return enderecos.filter((e) => !e.endsWith('/encontros') && !e.endsWith('/improviso'))
}

test('nenhum cartão do acervo transborda a coluna, em nenhuma cena', async ({ page }) => {
  await page.setViewportSize({ width: 1400, height: 900 })
  const cenas = await cenasDosCatalogos(page)
  // O CONTROLE: o trilho tem catálogos de verdade. Uma lista vazia faria o
  // laço abaixo não medir nada e passar verde.
  expect(cenas.length, 'o trilho não ofereceu catálogo nenhum').toBeGreaterThan(8)

  let medidas = 0
  for (const cena of cenas) {
    await page.goto(cena)

    const medida = await page.evaluate(() => {
      const cartoes = [...document.querySelectorAll('.acervo-em-colunas > div')]
      return {
        temGrade: !!document.querySelector('.acervo-em-colunas'),
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

    // O BESTIÁRIO está entre os catálogos do trilho e NÃO usa a grade de
    // cartões: ele tem lista e painel, com os filtros próprios dele. Cena sem
    // grade é pulada, e a conta abaixo é o que impede isso de virar um jeito de
    // não medir nada.
    if (!medida.temGrade) continue
    medidas++

    // O CONTROLE: a cena desenhou cartões. Sem ele, "nada transbordou" seria
    // verdade sobre uma tela vazia.
    expect(medida.quantos, `${cena} não desenhou cartão nenhum`).toBeGreaterThan(0)
    expect(medida.estouram, `cartões de ${cena} pintam por cima do vizinho`).toEqual([])
  }

  // Nove ou mais cenas MEDIDAS de verdade: sem isto, um seletor que parasse de
  // casar transformaria o guarda inteiro num laço que não afirma nada.
  expect(medidas, 'quase nenhuma cena foi medida').toBeGreaterThan(8)
})

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
