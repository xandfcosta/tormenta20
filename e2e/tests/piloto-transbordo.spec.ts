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
  await page.goto('/mestre/condicoes')
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
  await page.goto('/mestre/condicoes')

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
  expect(page.url()).toContain('/mestre/condicoes')
  await expect(page.getByText('-2 em testes de perícia.')).toBeVisible()

  await page.keyboard.press('Escape')
  expect(await caixa.evaluate((d: HTMLDialogElement) => d.open)).toBe(false)
})

test('a ficha do monstro usa a largura: duas colunas quando cabe, empilhada quando não', async ({
  page,
}) => {
  // O defeito, visto pelo dono na tela: a ficha parava numa coluna de 56rem com
  // meia tela vazia ao lado, e as habilidades especiais ficavam abaixo da dobra.
  //
  // E2E porque a decisão é de `@container`: a pergunta é quanto o BLOCO recebe,
  // e o bloco é desenhado em três larguras diferentes (painel do mestre, diálogo
  // da Mesa, ficha em diálogo do telefone). Em jsdom todo elemento mede zero e a
  // consulta de contêiner nunca dispara.
  await page.setViewportSize({ width: 1500, height: 900 })
  await page.goto('/mestre/bestiario?criatura=dragao-adulto')

  // O bloco é desenhado DUAS vezes na cena — no painel e na ficha em diálogo do
  // telefone, que o CSS esconde nesta largura. Medir o do painel é medir o que
  // está na tela; o outro tem largura zero, e medir caixa escondida é medir
  // nada com cara de medição.
  const colunas = page.locator('.mesa-painel .bloco-do-verbete-colunas')
  await expect(colunas).toBeVisible()

  const largo = await colunas.evaluate((el) => ({
    colunas: getComputedStyle(el).gridTemplateColumns.split(' ').length,
    // Os dois filhos lado a lado: mesma linha significa mesmo topo.
    mesmoTopo:
      el.children[0].getBoundingClientRect().top === el.children[1].getBoundingClientRect().top,
    sobra: el.parentElement!.getBoundingClientRect().width - el.getBoundingClientRect().width,
  }))
  expect(largo.colunas, 'a ficha não abriu em duas colunas com 1500px').toBe(2)
  expect(largo.mesmoTopo, 'as duas colunas não estão lado a lado').toBe(true)
  // E ela USA a largura: o bloco não pode parar muito antes do painel.
  expect(largo.sobra, 'a ficha deixou meia tela vazia ao lado').toBeLessThan(40)

  // Num painel ESTREITO volta a empilhar — a mesma árvore, sem segundo desenho:
  // duas árvores para o mesmo bloco se desencontram.
  //
  // 1000px e não 420: a esta largura o painel ainda existe (o palco passa dos
  // 50rem que a `.mesa-duas-colunas` pede) mas dá ao bloco menos que 46rem. A
  // 420 o painel some inteiro, e a asserção mediria um elemento escondido.
  await page.setViewportSize({ width: 1000, height: 900 })
  await expect(colunas).toBeVisible()
  const estreito = await colunas.evaluate((el) => ({
    colunas: getComputedStyle(el).gridTemplateColumns.split(' ').length,
    largura: el.getBoundingClientRect().width,
  }))
  expect(estreito.largura, 'o painel não ficou estreito o bastante para medir').toBeLessThan(46 * 16)
  expect(estreito.colunas, 'a ficha continuou em duas colunas num painel estreito').toBe(1)
})

test('a cena de campanhas tem a mesma forma da de personagens: palco em cima, lista embaixo', async ({
  page,
}) => {
  // Decisão do dono: as duas telas respondem a mesma pergunta — escolha um da
  // lista e veja o palco —, e discordavam. A de campanhas punha a lista numa
  // COLUNA ao lado, e o livro ficava com metade da janela.
  //
  // E2E porque a afirmação é de GEOMETRIA: quem está acima de quem, e quanto o
  // livro recebe de largura. Em jsdom todo elemento mede zero.
  await page.setViewportSize({ width: 1500, height: 900 })

  const medidas: Record<string, { livro: number; tiraAbaixo: boolean; tiraDeitada: boolean }> = {}
  for (const cena of ['/campanhas', '/personagens']) {
    await page.goto(cena)
    medidas[cena] = await page.evaluate(() => {
      // A tira é a região que o driver de teclado dirige: `rail` nas campanhas
      // (o nome é contrato com o driver) e `filme` nos personagens.
      const tira = document.querySelector('[data-nav-region="rail"], [data-nav-region="filme"]')!
      const palco = tira.previousElementSibling!
      const t = tira.getBoundingClientRect()
      const p = palco.getBoundingClientRect()
      return { livro: p.width, tiraAbaixo: t.top >= p.bottom - 1, tiraDeitada: t.width > t.height }
    })
  }

  for (const [cena, m] of Object.entries(medidas)) {
    expect(m.tiraAbaixo, `${cena}: a lista não está abaixo do palco`).toBe(true)
    expect(m.tiraDeitada, `${cena}: a lista não está deitada`).toBe(true)
    // O palco toma a janela inteira: era isto que a coluna ao lado comia.
    expect(m.livro, `${cena}: o palco não usa a largura`).toBeGreaterThan(1400)
  }
})

/**
 * O CRACHÁ DA DEFESA cabe no rodapé da ficha a 390px, com o alvo CAÍDO (ALE-274).
 *
 * O Caído parte a Defesa em duas (p394), e desde a ALE-274 o crachá mostra as
 * duas: `10` vira `5 CaC · 15 Dist`, três vezes mais largo. Ele é `shrink-0` num
 * flex ao lado do nome do herói, que TRUNCA — então o risco não é o crachá
 * transbordar, é ele espremer o nome até sumir.
 *
 * E2E porque a pergunta é de LEIAUTE REAL: quanto o nome trunca depende da
 * fonte e do algoritmo do navegador, e em jsdom tudo mede zero.
 *
 * E VISITAR O CASO É METADE DO GUARDA. A suíte já rodava verde com esta mudança
 * dentro, porque a seed não tem ninguém caído — o guarda de transbordo media o
 * crachá curto e dizia "passou". É a família que o guia cataloga em "um guarda
 * só mede o que ele VISITA": a cena estava na lista, o DADO não.
 */
test('o crachá da Defesa partida cabe no rodapé da ficha a 390px', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 })
  // A aba dos EFEITOS porque é lá que a condição se aplica; o crachá medido é o
  // do RODAPÉ, que aparece em todas elas — e é justamente esse o argumento dele.
  await page.goto('/personagens/1?tab=conditionals')

  // O CONTROLE: em pé, o crachá é curto — é assim que se sabe que a medição
  // depois é do caso novo e não do que já estava lá.
  const cracha = page.locator('span', { hasText: /^DEF$/ }).locator('..')
  await expect(cracha).toBeVisible()
  const curto = await cracha.evaluate((el) => el.getBoundingClientRect().width)

  await page.getByRole('button', { name: 'Aplicar condição' }).click()
  await page.getByRole('button', { name: /^Caído/ }).click()
  await expect(cracha).toContainText('CaC')

  const medida = await cracha.evaluate((el) => {
    const caixa = el.getBoundingClientRect()
    const linha = el.parentElement as HTMLElement
    const pai = linha.getBoundingClientRect()
    const nome = linha.querySelector('p') as HTMLElement | null
    return {
      largura: caixa.width,
      foraPelaDireita: caixa.right - pai.right,
      recorteDaLinha: linha.scrollWidth - linha.clientWidth,
      nomeVisivel: nome ? nome.getBoundingClientRect().width : -1,
    }
  })

  expect(medida.largura, 'o crachá não cresceu: o caso não está medindo a Defesa partida')
    .toBeGreaterThan(curto)
  expect(medida.foraPelaDireita, 'o crachá saiu pela direita da linha').toBeLessThanOrEqual(1)
  expect(medida.recorteDaLinha, 'a linha recortou o próprio conteúdo').toBeLessThanOrEqual(1)
  // O nome não pode ser espremido a nada: truncar é o desenho, sumir não é.
  expect(medida.nomeVisivel, 'o crachá espremeu o nome do herói até sumir').toBeGreaterThan(24)
})
