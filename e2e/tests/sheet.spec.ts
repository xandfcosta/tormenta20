import { expect, test } from '@playwright/test'
import { medeOContraste } from './support/contrast'
import { expectOneFocusRing } from './support/focus'
import { expectDentroDaJanela, expectNadaRolaDeLado } from './support/geometry'
import { medeATipografia } from './support/typography'
import { expectNoHorizontalOverflow, VIEWPORTS } from './support/viewports'

/**
 * A FICHA — o que só o navegador mede: LEIAUTE REAL nos seis formatos.
 *
 * O que o servidor escreve está preso em Go, que é mais barato: endereço das
 * abas, posse, degrau de nível, faixa dos vitais. Sobra para cá a geometria: o
 * crachá é uma fileira com retrato, identidade, nível e oito botões de vital, e
 * a barra de abas tem sete itens — um controle que sai da janela não é feio, é
 * inalcançável.
 */
test.use({ storageState: '.auth/user.json' })

/** O conjurador da semente, o mesmo que o spec do Grimório usa. */
const CONJURADOR = 'Necromante Nv12 Magias'

/** O herói de mochila cheia: armadura vestida, machado e escudo nas mãos. */
const TANQUE = 'Tanque Placas Nv10'

/** O primeiro herói do elenco, pelo endereço da ficha nova. */
async function aFichaDoPrimeiro(page: import('@playwright/test').Page) {
  await page.goto('/personagens')
  const href = await page.locator('a[aria-label^="Abrir ficha de"]').first().getAttribute('href')
  const id = href?.match(/\d+/)?.[0]
  expect(id, 'não achei um herói no elenco: o resto do caso não mediria nada').toBeTruthy()
  await page.goto(`/personagens/${id}`)
  return id as string
}

test('a ficha cabe nos seis formatos, e nenhum botão do crachá sai da janela', async ({ page }) => {
  await aFichaDoPrimeiro(page)
  await expect(page.getByRole('navigation', { name: 'Seções da ficha' })).toBeVisible()

  await expectNoHorizontalOverflow(page, VIEWPORTS)

  // O CRACHÁ é o que mais aperta: oito passos de vital, o degrau de nível e o
  // retrato na mesma fileira. No telefone em pé é onde a conta estoura.
  await page.setViewportSize({ width: 390, height: 844 })
  await expectDentroDaJanela(page)
})

/**
 * O MESMO OLHAR, EM TODA ABA — e ele caminha pela barra em vez de ter uma lista.
 *
 * Ler os `href` da barra é AMOSTRAGEM; uma lista escrita à mão é ENUMERAÇÃO, e
 * nasce incompleta na primeira vez que alguém acrescenta um painel e esquece de
 * vir aqui. Cobertura é função de onde o teste NAVEGA, não de quantas asserções
 * ele tem.
 */
test('nenhum painel da ficha transborda o telefone', async ({ page }) => {
  await aFichaDoPrimeiro(page)
  const enderecos = await page
    .getByRole('navigation', { name: 'Seções da ficha' })
    .getByRole('link')
    .evaluateAll((links) => links.map((l) => (l as HTMLAnchorElement).href))
  expect(enderecos, 'a barra de abas veio vazia: este caso não mediria nada').toHaveLength(7)

  await page.setViewportSize({ width: 390, height: 844 })
  for (const endereco of enderecos) {
    await page.goto(endereco)
    await expect(page.getByRole('navigation', { name: 'Seções da ficha' })).toBeVisible()
    await expectDentroDaJanela(page)
    // AS DUAS, e a segunda não é redundância — foi medida.
    //
    // O `expectDentroDaJanela` ignora, DE PROPÓSITO, quem tem um eixo rolável
    // acima: pela definição dele, há como chegar lá. O painel da ficha rola na
    // vertical, e `overflow-y: auto` faz o navegador computar o `overflow-x`
    // como `auto` junto — então um bloco de 500px numa janela de 390 passa
    // por ele em silêncio. Provado por sabotagem: o caso ficou VERDE com o
    // bloco largo no ar, e só o `expectNadaRolaDeLado` o viu.
    await expectNadaRolaDeLado(page)

    // O CONTRASTE entra no MESMO caminhar, e não num caso à parte com uma lista
    // de abas: à parte ele seria enumeração, e a aba que nascer amanhã ficaria
    // sem medição.
    const contraste = await medeOContraste(page)
    // O DENOMINADOR: sem ele, uma lista de falhas vazia é indistinguível de "o
    // seletor não achou nada", e as duas se parecem no terminal. Trinta é bem
    // abaixo do que a aba mais pobre desenha (medido: a de Poderes, a mais
    // vazia das sete, passa de 40) e bem acima de zero.
    expect(
      contraste.medidos,
      `em ${endereco} o medidor olhou ${contraste.medidos} textos: o seletor da cena parou de casar`,
    ).toBeGreaterThan(30)
    expect(contraste.falhas, `texto abaixo do AA em ${endereco}`).toEqual([])

    // A TIPOGRAFIA entra no MESMO caminhar, pela mesma razão que o contraste.
    const tipografia = await medeATipografia(page)
    expect(
      tipografia.medidos,
      `em ${endereco} o medidor não achou NENHUM texto em Cinzel: ou a fonte não carregou, ou o filtro parou de casar — e o silêncio abaixo não seria evidência`,
    ).toBeGreaterThan(0)
    expect(tipografia.falhas, `Cinzel abaixo do piso de leitura em ${endereco}`).toEqual([])

    // E O ANEL DE FOCO entra no MESMO caminhar, pela terceira vez e pela mesma
    // razão: à parte ele seria enumeração.
    await expectOneFocusRing(page, `em ${endereco}`, 10)
  }
})

/**
 * O id de um herói, pelo NOME e pela busca da cena. O elenco é ordenado por
 * última alteração, então "o primeiro do elenco" muda conforme o spec que rodou
 * antes — buscar pelo nome é o que torna os casos abaixo independentes.
 */
async function oIdDoHeroi(page: import('@playwright/test').Page, nome: string) {
  await page.goto('/personagens')
  await page.getByRole('searchbox', { name: 'Buscar personagem' }).fill(nome)
  const abrir = page.getByRole('link', { name: `Abrir ficha de ${nome}` })
  await expect(abrir, `a semente não tem ${nome}`).toBeVisible()
  const href = await abrir.getAttribute('href')
  return href?.match(/\d+/)?.[0] as string
}

const oIdDoConjurador = (page: import('@playwright/test').Page) => oIdDoHeroi(page, CONJURADOR)

/**
 * O PAINEL RAMIFICA PELO DADO, e caminhar pelas abas não alcança isso: o caso
 * acima abre as sete abas de um guerreiro. A tripla mágica do Combate — Limite
 * PM, CD Magia, Custo PM — só existe para quem conjura por classe, e ela usa a
 * paleta ARCANA, outra tinta sobre o mesmo painel.
 */
test('a paleta arcana do Combate é legível para quem conjura', async ({ page }) => {
  const id = await oIdDoConjurador(page)

  await page.setViewportSize({ width: 390, height: 844 })
  await page.goto(`/personagens/${id}?tab=combat`)

  // O CONTROLE de que a tripla está mesmo na tela: sem ela, o contraste abaixo
  // mede um Combate sem nada de arcano e passa dizendo o contrário.
  await expect(
    page.getByRole('button', { name: /^CD Magia/ }),
    'a ficha do conjurador não desenhou a tripla mágica: a paleta arcana não entrou na medição',
  ).toBeVisible()

  const contraste = await medeOContraste(page)
  expect(contraste.falhas, 'texto abaixo do AA no Combate de um conjurador').toEqual([])
})

/**
 * O GRIMÓRIO ABERTO — e os dois diálogos que só existem para quem conjura.
 *
 * Sem classe conjuradora o painel é uma frase, então o caminhar pelas abas do
 * guerreiro não vê nada disto. E os diálogos justificam o navegador por conta
 * própria: o de aprender leva as ~198 magias do Capítulo 4 numa caixa que rola
 * dentro de si, que o jsdom mede como zero.
 */
test('o grimório e os diálogos de conjurar e aprender cabem no telefone', async ({ page }) => {
  const id = await oIdDoConjurador(page)
  await page.setViewportSize({ width: 390, height: 844 })
  await page.goto(`/personagens/${id}?tab=spells`)

  // O CONTROLE do painel: sem magia na tela o resto mede um grimório vazio.
  await expect(
    page.getByRole('button', { name: 'Conjurar Bola de Fogo' }),
    'o grimório do conjurador não desenhou as magias: nada abaixo mediria a fatia 6',
  ).toBeVisible()
  await expectNadaRolaDeLado(page)
  const noPainel = await medeOContraste(page)
  expect(noPainel.medidos, 'o medidor não achou texto no grimório').toBeGreaterThan(30)
  expect(noPainel.falhas, 'texto abaixo do AA no grimório').toEqual([])

  // O DENOMINADOR DE UM DIÁLOGO NÃO É COMPARATIVO: `medidos` dá o mesmo número
  // com o diálogo fechado e com ele aberto, porque o medidor descarta o nó que
  // ESCONDE A SI MESMO e não o que está debaixo de um ancestral escondido — e a
  // cena do Datastar esconde por `data-show` no pai. Quem prova que o diálogo
  // abriu é o `toBeVisible` de dentro dele; o que os dois blocos abaixo
  // acrescentam é LEIAUTE REAL com a caixa no ar.

  // O DIÁLOGO DE CONJURAR é onde mora o contador de pilha e o cadeado do
  // aprimoramento fora de alcance — o Necromante é nível 12, alcança o 3º
  // círculo, e a Invisibilidade tem aprimoramento de 4º.
  await page.getByRole('button', { name: 'Conjurar Invisibilidade' }).click()
  await expect(page.getByText(/exige o 4º círculo/)).toBeVisible()
  await expectNadaRolaDeLado(page)
  await page.keyboard.press('Escape')

  // O DIÁLOGO DE APRENDER leva o Capítulo 4 inteiro.
  await page.getByRole('button', { name: 'Aprender magia' }).click()
  const caixa = page.getByRole('dialog', { name: 'Aprender magia' })
  await expect(caixa).toBeVisible()
  await expectDentroDaJanela(page)
  await expectNadaRolaDeLado(page)
})

/**
 * A MOCHILA ABERTA, com a ficha de um item e o catálogo do Capítulo 3.
 *
 * O caminhar pelas sete abas cobre a tira e a grade, mas nenhum dos DIÁLOGOS —
 * equipar, usar, melhorias e as ~160 linhas do catálogo. Eles justificam o
 * navegador por conta própria: são caixas que rolam dentro de si numa tela de
 * 390px.
 */
test('a mochila abre a ficha do item e o catálogo sem estourar o telefone', async ({ page }) => {
  const id = await oIdDoHeroi(page, TANQUE)
  await page.setViewportSize({ width: 390, height: 844 })
  await page.goto(`/personagens/${id}?tab=bag`)

  // O CONTROLE do painel: sem a tira desenhada, o resto mede uma mochila vazia.
  await expect(
    page.getByRole('button', { name: 'Guardar Machado de batalha' }),
    'a mochila do tanque não desenhou os equipados: nada abaixo mediria a fatia 7',
  ).toBeVisible()
  await expectNadaRolaDeLado(page)
  const noPainel = await medeOContraste(page)
  expect(noPainel.medidos, 'o medidor não achou texto na mochila').toBeGreaterThan(30)
  expect(noPainel.falhas, 'texto abaixo do AA na mochila').toEqual([])

  // A FICHA DO ITEM, aberta pelo cartão da tira.
  await page.getByRole('button', { name: 'Abrir Machado de batalha' }).click()
  const ficha = page.getByRole('dialog', { name: 'Machado de batalha' })
  await expect(ficha).toBeVisible()
  await expect(ficha.getByRole('button', { name: 'Melhorias de Machado de batalha' })).toBeVisible()
  await expectDentroDaJanela(page)
  await expectNadaRolaDeLado(page)
  await page.keyboard.press('Escape')

  // O CATÁLOGO leva o Capítulo 3 inteiro numa caixa que rola dentro de si.
  await page.getByRole('button', { name: 'Adicionar do catálogo' }).click()
  await expect(page.getByRole('dialog', { name: 'Adicionar do catálogo' })).toBeVisible()
  await expectDentroDaJanela(page)
  await expectNadaRolaDeLado(page)
})

/**
 * A ABA PODERES ABERTA, com o diálogo de escolher.
 *
 * O caminhar pelas sete abas mede a lista; o que ele não alcança é o DIÁLOGO,
 * que leva os ~93 poderes eletivos de uma classe numa caixa que rola dentro de
 * si — e o contador de degraus da postura, que só existe para quem tem uma.
 */
test('os poderes abrem o diálogo de escolher sem estourar o telefone', async ({ page }) => {
  const id = await oIdDoHeroi(page, TANQUE)
  await page.setViewportSize({ width: 390, height: 844 })
  await page.goto(`/personagens/${id}?tab=abilities`)

  // O CONTROLE da lista: sem poderes na tela o resto mede uma aba vazia.
  await expect(
    page.getByRole('button', { name: 'Escolher poderes', exact: true }),
    'a aba Poderes não desenhou o gesto de escolher: nada abaixo mediria a fatia 8',
  ).toBeVisible()
  await expectNadaRolaDeLado(page)
  const naLista = await medeOContraste(page)
  expect(naLista.medidos, 'o medidor não achou texto nos Poderes').toBeGreaterThan(30)
  expect(naLista.falhas, 'texto abaixo do AA nos Poderes').toEqual([])

  await page.getByRole('button', { name: 'Escolher poderes', exact: true }).click()
  const dialogo = page.getByRole('dialog', { name: 'Escolher poderes' })
  await expect(dialogo).toBeVisible()
  await dialogo.getByRole('button', { name: 'Classe', exact: true }).click()
  // O `:visible` não é preciosismo: as três abas são desenhadas de uma vez e
  // alternadas por `data-show`, então o primeiro `switch` do DOM é o da Origem,
  // que está escondido — e esperar por ele é esperar para sempre.
  await expect(dialogo.locator('[role="switch"]:visible').first()).toBeVisible()
  await expectDentroDaJanela(page)
  await expectNadaRolaDeLado(page)
})

// O endereço das abas, o alias `inventory`/`equipment`, o nome inválido caindo
// na primeira, as sete abas desenhando painel e o `aria-current` na aba pedida
// NÃO têm caso aqui de propósito: nenhum usa mecanismo que só um navegador
// tenha, e todos estão presos em Go (`TestTheSheetTabAddressSurvives`,
// `TestEverySheetTabDrawsSomething`).

/**
 * O CELULAR DEITADO, e o orçamento do crachá.
 *
 * O TETO É O CRACHÁ e não a lista, e isso é deliberado: ele é a única peça
 * COMPARTILHADA pelas sete abas, então uma regra presa aqui cobre as sete sem
 * enumerar nenhuma — a oitava aba já nasce medida. Prender "a lista mostra uma
 * linha" seria a mesma garantia sete vezes, e ainda ficaria impossível nas duas
 * abas cujo "primeiro filho" é um bloco de 242px e não uma linha.
 *
 * O TETO é ARITMÉTICA e não um número redondo: duas fileiras de alvo de toque
 * mais o respiro entre elas. Duas e não uma porque a fileira ENROLA de
 * propósito — os oito passos de vital são 44px de alvo mínimo cada, e
 * espremê-los numa linha só trocaria este defeito por um de alvo de toque.
 * Escrito como conta para dizer de qual soma o teto saiu; um número redondo some
 * no dia em que alguém precisa saber se pode mexer nele.
 *
 * Só um navegador testemunha: a chave é `max-lg:landscape:`, que é largura MAIS
 * orientação, e em jsdom não há orientação.
 */
test('deitado, o crachá do jogador não come metade da tela', async ({ page }) => {
  await aFichaDoPrimeiro(page)
  const enderecos = await page
    .getByRole('navigation', { name: 'Seções da ficha' })
    .getByRole('link')
    .evaluateAll((links) => links.map((l) => (l as HTMLAnchorElement).href))
  expect(enderecos, 'a barra de abas veio vazia: este caso não mediria nada').toHaveLength(7)

  // 44 é o alvo mínimo de toque; o respiro é o `py-1` (8), o `gap-y-1` (4) e a
  // borda de cima (1), com três de folga.
  const ALVO_DE_TOQUE = 44
  const TETO_DO_CRACHA = 2 * ALVO_DE_TOQUE + 8 + 4 + 1 + 3

  await page.setViewportSize({ width: 844, height: 390 })
  for (const endereco of enderecos) {
    await page.goto(endereco)
    const cracha = page.locator('#player-badge')
    // `toBeVisible` ANTES de medir: uma caixa escondida devolve zero sem
    // reclamar, e um zero passaria neste teto com folga.
    await expect(cracha, `o crachá sumiu em ${endereco}: sem ele não há medição`).toBeVisible()
    const caixa = await cracha.boundingBox()
    expect(
      caixa?.height,
      `o crachá come ${Math.round(caixa?.height ?? 0)}px dos 390 em ${endereco}`,
    ).toBeLessThanOrEqual(TETO_DO_CRACHA)

    // E o que o corte NÃO pode custar: mexer no PV é o gesto mais frequente da
    // noite, e o crachá existe para que ele não dependa de qual aba está
    // aberta. Um passo de vital fora da janela deitado seria trocar um defeito
    // por outro.
    await expectDentroDaJanela(page)
    await expectNadaRolaDeLado(page)
  }
})
