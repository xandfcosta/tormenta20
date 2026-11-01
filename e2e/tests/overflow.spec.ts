import { expect, type Page, test } from '@playwright/test'

/**
 * NADA transborda o cartão. O caso real é a barra sem espaço — a magia que lista
 * `(abalado/atordoado/apavorado/…)` em 100 caracteres —, que o navegador não
 * trata como oportunidade de quebra e pinta POR CIMA dos cartões vizinhos.
 *
 * E2E porque a pergunta é de LEIAUTE REAL: quebra de linha depende da fonte, da
 * largura da coluna e do algoritmo do navegador. Em jsdom todo elemento mede
 * zero e este guarda passaria verde sempre.
 *
 * AMOSTRAGEM sobre as abas: a aba que entrar amanhã já nasce medida.
 */
test.use({ storageState: '.auth/user.json' })

/**
 * As cenas a visitar saem do TRILHO, lidas da página — não de uma lista aqui:
 * uma lista escrita à mão não sabe da parada que nasceu ontem, e a cena nova
 * ficaria sem medição em silêncio.
 */
async function cenasDosCatalogos(page: Page): Promise<string[]> {
  await page.goto('/mestre/condicoes')
  const addresses = await page
    .locator('nav[aria-label="Ferramentas do mestre"] a')
    .evaluateAll((links) =>
      links.map((a) => (a as HTMLAnchorElement).getAttribute('href') ?? ''),
    )
  // As FERRAMENTAS ficam de fora: encontros e improviso não desenham cartão de
  // acervo, e cobrar transbordo delas mediria outra coisa.
  return addresses.filter((e) => !e.endsWith('/encontros') && !e.endsWith('/improviso'))
}

test('nenhum cartão do acervo transborda a coluna, em nenhuma cena', async ({ page }) => {
  await page.setViewportSize({ width: 1400, height: 900 })
  const scenes = await cenasDosCatalogos(page)
  // O CONTROLE: o trilho tem catálogos de verdade. Uma lista vazia faria o
  // laço abaixo não medir nada e passar verde.
  expect(scenes.length, 'o trilho não ofereceu catálogo nenhum').toBeGreaterThan(8)

  let measured = 0
  for (const scene of scenes) {
    await page.goto(scene)

    const measurement = await page.evaluate(() => {
      const cards = [...document.querySelectorAll('.ds-catalog-grid > article')]
      return {
        temGrade: !!document.querySelector('.ds-catalog-grid'),
        quantos: cards.length,
        estouram: cards
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
    if (!measurement.temGrade) continue
    measured++

    // O CONTROLE: a cena desenhou cartões. Sem ele, "nada transbordou" seria
    // verdade sobre uma tela vazia.
    expect(measurement.quantos, `${scene} não desenhou cartão nenhum`).toBeGreaterThan(0)
    expect(measurement.estouram, `cartões de ${scene} pintam por cima do vizinho`).toEqual([])
  }

  // Nove ou mais cenas MEDIDAS de verdade: sem isto, um seletor que parasse de
  // casar transformaria o guarda inteiro num laço que não afirma nada.
  expect(measured, 'quase nenhuma cena foi medida').toBeGreaterThan(8)
})

test('o elo mostra o conceito por cima, sem tirar a pessoa da regra que lia', async ({ page }) => {
  await page.setViewportSize({ width: 1400, height: 900 })
  await page.goto('/mestre/condicoes')

  // A condição Abalado termina em "Medo." — o tipo de efeito, que é outro
  // verbete. É o caso que o dono trouxe.
  // Pelo `title` e não pelo nome acessível: o texto do elo é "Medo." (com o
  // ponto do livro), e é ele que vira o nome — o `title` é a explicação.
  const link = page.locator('a[title="Ver Medo"]').first()
  await expect(link).toBeVisible()

  const crate = page.locator('#entry-in-dialog')
  expect(await crate.evaluate((d: HTMLDialogElement) => d.open)).toBe(false)

  await link.click()

  expect(await crate.evaluate((d: HTMLDialogElement) => d.open)).toBe(true)
  await expect(crate).toContainText('Medo capaz de prejudicar o alvo')
  // A CENA CONTINUA: o endereço não mudou e a condição que se estava lendo está
  // lá atrás. Era isso que a navegação para uma busca destruía.
  expect(page.url()).toContain('/mestre/condicoes')
  await expect(page.getByText('-2 em testes de perícia.')).toBeVisible()

  await page.keyboard.press('Escape')
  expect(await crate.evaluate((d: HTMLDialogElement) => d.open)).toBe(false)
})

test('a ficha do monstro usa a largura: duas colunas quando cabe, empilhada quando não', async ({
  page,
}) => {
  // O defeito, visto pelo dono na tela: a ficha parava numa coluna estreita com
  // meia tela vazia ao lado, e as habilidades especiais ficavam abaixo da dobra.
  //
  // E2E porque a decisão é de `@container`: a pergunta é quanto o BLOCO recebe,
  // e o mesmo bloco é desenhado no painel da Mesa do Mestre e na folha da mesa.
  // Em jsdom todo elemento mede zero e a consulta de contêiner nunca dispara.
  await page.setViewportSize({ width: 1500, height: 900 })
  await page.goto('/mestre/bestiario?criatura=dragao-adulto')

  // O bloco é desenhado DUAS vezes na cena — no painel e na folha do telefone,
  // que o CSS esconde nesta largura. Medir o do painel é medir o que está na
  // tela; o outro tem largura zero.
  const block = page.locator('[data-nav-region="ficha"] .ds-gm-creature')
  await expect(block).toBeVisible()

  const wide = await block.evaluate((el) => ({
    colunas: getComputedStyle(el).gridTemplateColumns.split(' ').length,
    mesmoTopo:
      el.children[0].getBoundingClientRect().top === el.children[1].getBoundingClientRect().top,
    sobra: el.parentElement!.getBoundingClientRect().width - el.getBoundingClientRect().width,
  }))
  expect(wide.colunas, 'a ficha não abriu em duas colunas com 1500px').toBe(2)
  expect(wide.mesmoTopo, 'as duas colunas não estão lado a lado').toBe(true)
  // E ela USA a largura: o bloco não pode parar muito antes do painel.
  expect(wide.sobra, 'a ficha deixou meia tela vazia ao lado').toBeLessThan(40)

  // Num painel ESTREITO volta a empilhar — a mesma árvore, sem segundo desenho.
  // 1100 e não menos: abaixo de 50rem de área principal o painel some inteiro,
  // e a asserção mediria um elemento escondido.
  await page.setViewportSize({ width: 1100, height: 900 })
  await expect(block).toBeVisible()
  const narrow = await block.evaluate((el) => ({
    colunas: getComputedStyle(el).gridTemplateColumns.split(' ').length,
    largura: el.getBoundingClientRect().width,
  }))
  expect(narrow.largura, 'o painel não ficou estreito o bastante para medir').toBeLessThan(46 * 16)
  expect(narrow.colunas, 'a ficha continuou em duas colunas num painel estreito').toBe(1)
})

/**
 * O CRACHÁ DA DEFESA cabe no cabeçalho da ficha a 390px, com o alvo CAÍDO.
 *
 * O Caído parte a Defesa em duas (p394), e o crachá mostra as duas: `22` vira
 * `17 CaC · 27 Dist`, várias vezes mais largo. Ele é uma coluna `auto` da grade
 * do cabeçalho, ao lado do nome do herói, que TRUNCA — então o risco não é o
 * crachá transbordar, é ele espremer o nome até sumir.
 *
 * E2E porque a pergunta é de LEIAUTE REAL: quanto o nome trunca depende da
 * fonte e do algoritmo do navegador, e em jsdom tudo mede zero.
 *
 * O caso APLICA a condição, e isso é metade do guarda: a seed não tem ninguém
 * caído, então o guarda de transbordo media o crachá curto e dizia "passou". A
 * cena estava na lista; o DADO não.
 */
test('o crachá da Defesa partida cabe no cabeçalho da ficha a 390px', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 })
  // A aba dos EFEITOS porque é lá que a condição se aplica; o crachá medido é o
  // do CABEÇALHO, que aparece em todas elas — e é justamente esse o argumento
  // dele.
  await page.goto('/personagens/1?tab=conditionals')

  // O CONTROLE: em pé, o crachá é curto — é assim que se sabe que a medição
  // depois é do caso novo e não do que já estava lá.
  const header = page.locator('header').filter({ has: page.getByRole('heading', { level: 1 }) })
  const badge = header.locator('span', { hasText: /^DEF$/ }).locator('..')
  await expect(badge).toBeVisible()
  const short = await badge.evaluate((el) => el.getBoundingClientRect().width)

  await page.getByRole('button', { name: 'Aplicar condição' }).click()
  await page.getByRole('button', { name: 'Aplicar condição: Caído' }).click()
  await expect(badge).toContainText('CaC')

  const measurement = await badge.evaluate((el) => {
    const box = el.getBoundingClientRect()
    const row = el.parentElement as HTMLElement
    const father = row.getBoundingClientRect()
    const label = row.querySelector('h1') as HTMLElement | null
    return {
      largura: box.width,
      foraPelaDireita: box.right - father.right,
      recorteDaLinha: row.scrollWidth - row.clientWidth,
      nomeVisivel: label ? label.getBoundingClientRect().width : -1,
    }
  })

  expect(measurement.largura, 'o crachá não cresceu: o caso não está medindo a Defesa partida')
    .toBeGreaterThan(short)
  expect(measurement.foraPelaDireita, 'o crachá saiu pela direita da linha').toBeLessThanOrEqual(1)
  expect(measurement.recorteDaLinha, 'a linha recortou o próprio conteúdo').toBeLessThanOrEqual(1)
  // O nome não pode ser espremido a nada: truncar é o desenho, sumir não é.
  expect(measurement.nomeVisivel, 'o crachá espremeu o nome do herói até sumir').toBeGreaterThan(24)
})
