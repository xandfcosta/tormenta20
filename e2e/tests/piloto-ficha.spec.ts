import { expect, test } from '@playwright/test'
import { expectDentroDaJanela, expectNadaRolaDeLado } from './support/geometry'
import { expectNoHorizontalOverflow, VIEWPORTS } from './support/viewports'
import { medeOContraste } from './support/contraste'

/**
 * A FICHA em Datastar (ALE-272, fatia 1) — a casca, as abas e o crachá.
 *
 * O que o servidor escreve está preso em Go, que é mais barato: o endereço das
 * abas, a posse, o degrau de nível e a faixa dos vitais. O que sobra para cá é o
 * que só o navegador mede — LEIAUTE REAL nos seis formatos.
 *
 * E leiaute aqui não é enfeite: o crachá é uma fileira com retrato, identidade,
 * nível e oito botões de vital, e a barra de abas tem sete itens. As duas são
 * exatamente a forma que já transbordou nesta casa (ALE-162, ALE-178) — um
 * controle que sai da janela não é feio, é inalcançável.
 */
test.use({ storageState: '.auth/user.json' })

/** O conjurador da semente, o mesmo que o spec do Grimório usa. */
const CONJURADOR = 'Necromante Nv12 Magias'

/** O herói de mochila cheia: armadura vestida, machado e escudo nas mãos. */
const TANQUE = 'Tanque Placas Nv10'

/** O primeiro herói do elenco, pelo endereço da ficha nova. */
async function aFichaDoPrimeiro(page: import('@playwright/test').Page) {
  await page.goto('/piloto/personagens')
  const href = await page.locator('a[aria-label^="Abrir ficha de"]').first().getAttribute('href')
  const id = href?.match(/\d+/)?.[0]
  expect(id, 'não achei um herói no elenco: o resto do caso não mediria nada').toBeTruthy()
  await page.goto(`/piloto/personagens/${id}`)
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
 * A primeira versão deste arquivo media só a aba padrão, que na fatia 1 era um
 * aviso de duas linhas. Quando o painel de Proficiências chegou (fatia 2), o
 * guarda continuou verde sem nunca ter aberto o painel novo: é exatamente a
 * forma da ALE-237 e da ALE-252, onde a cobertura é função de onde o teste
 * NAVEGA e não de quantas asserções ele tem.
 *
 * Ele lê os `href` da barra em vez de trazer uma lista escrita, e essa é a
 * diferença entre AMOSTRAGEM e ENUMERAÇÃO: o painel da fatia 6 vai ser medido
 * sem ninguém lembrar de vir aqui. Uma lista escrita à mão nasce incompleta na
 * primeira vez que alguém esquece.
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
    // por ele em silêncio. Provei sabotando: o caso ficou VERDE com o bloco
    // largo no ar, e só o `expectNadaRolaDeLado` o viu. É a mesma lacuna que
    // a ALE-178 nomeou.
    await expectNadaRolaDeLado(page)

    // O CONTRASTE entra no MESMO caminhar, e não num caso à parte com uma lista
    // de abas: à parte ele seria enumeração, e a aba da fatia 6 nasceria sem
    // medição. A ficha atravessou as fatias 1 e 2 sem medição nenhuma de
    // contraste — não por decisão, mas porque o medidor era função privada de
    // outro spec (ver `support/contraste.ts`). O painel de Combate estreia a
    // paleta ARCANA na ficha, que é tinta clara sobre painel escuro e
    // exatamente a forma dos dois defeitos que este medidor já pegou.
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
  }
})

/**
 * O PAINEL RAMIFICA POR PERSONAGEM, e caminhar pelas abas não alcança isso.
 *
 * O caso acima abre as sete abas de UM herói, e o primeiro do elenco é um
 * guerreiro. A tripla mágica do Combate — Limite PM, CD Magia, Custo PM — só
 * existe para quem conjura por classe, e ela usa a paleta ARCANA, que é outra
 * tinta sobre o mesmo painel. Medida só no guerreiro, ela nunca foi medida.
 *
 * É a ALE-237 um nível abaixo: lá a cobertura era função de onde o teste
 * NAVEGA; aqui é função de QUEM ele abre. A saída continua sendo amostragem e
 * não lista — o herói é escolhido lendo o elenco, e o caso falha alto se o
 * elenco deixar de ter um conjurador, em vez de medir o vazio.
 */
/**
 * O id de um herói, pelo NOME e pela busca da cena. O elenco é um palco de um
 * herói por vez, ordenado por última alteração, então "o primeiro do elenco"
 * muda conforme o spec que rodou antes.
 *
 * Este era o `openSheetFromRoster` do `support/roster.ts`, que saiu na fatia 10
 * junto com os specs da ficha da SPA: com o elenco apontando para a ficha do
 * servidor, ele ficou sem nenhum outro chamador.
 */
async function oIdDoHeroi(page: import('@playwright/test').Page, nome: string) {
  await page.goto('/piloto/personagens')
  await page.getByRole('searchbox', { name: 'Buscar personagem' }).fill(nome)
  const abrir = page.getByRole('link', { name: `Abrir ficha de ${nome}` })
  await expect(abrir, `a semente não tem ${nome}`).toBeVisible()
  const href = await abrir.getAttribute('href')
  return href?.match(/\d+/)?.[0] as string
}

const oIdDoConjurador = (page: import('@playwright/test').Page) => oIdDoHeroi(page, CONJURADOR)

test('a paleta arcana do Combate é legível para quem conjura', async ({ page }) => {
  const id = await oIdDoConjurador(page)

  await page.setViewportSize({ width: 390, height: 844 })
  await page.goto(`/piloto/personagens/${id}?tab=combat`)

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
 * O caminhar pelas sete abas mede a de Magias do PRIMEIRO herói do elenco, que
 * é um guerreiro: sem classe conjuradora o painel é uma frase. Nada do que a
 * fatia 6 desenhou — o ouro do grimório, o cadeado do aprimoramento fora de
 * alcance, o contador de pilha — passa por lá. É a mesma lição da ALE-237 no
 * nível de baixo: a cobertura é função de QUEM o teste abre.
 *
 * E os diálogos justificam o navegador por conta própria: o de aprender leva as
 * ~198 magias do Capítulo 4 numa caixa que rola dentro de si, que é a forma que
 * já transbordou nesta casa (ALE-178) e que o jsdom mede como zero.
 */
test('o grimório e os diálogos de conjurar e aprender cabem no telefone', async ({ page }) => {
  const id = await oIdDoConjurador(page)
  await page.setViewportSize({ width: 390, height: 844 })
  await page.goto(`/piloto/personagens/${id}?tab=spells`)

  // O CONTROLE do painel: sem magia na tela o resto mede um grimório vazio.
  await expect(
    page.getByRole('button', { name: 'Conjurar Bola de Fogo' }),
    'o grimório do conjurador não desenhou as magias: nada abaixo mediria a fatia 6',
  ).toBeVisible()
  await expectNadaRolaDeLado(page)
  const noPainel = await medeOContraste(page)
  expect(noPainel.medidos, 'o medidor não achou texto no grimório').toBeGreaterThan(30)
  expect(noPainel.falhas, 'texto abaixo do AA no grimório').toEqual([])

  // O DENOMINADOR DE UM DIÁLOGO NÃO É COMPARATIVO, e a primeira versão disto
  // errou: `medidos` deu 809 com o diálogo fechado e 809 com ele aberto. O
  // medidor descarta o nó que ESCONDE A SI MESMO, não o que está debaixo de um
  // ancestral escondido — e a cena do Datastar esconde por `data-show` no pai.
  // Quem prova que o diálogo abriu é o `toBeVisible` de dentro dele; o que os
  // dois blocos abaixo acrescentam é LEIAUTE REAL com a caixa no ar, que é o
  // que nenhuma medição de cor alcança.

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
 * O caminhar pelas sete abas mede a Mochila do primeiro herói do elenco, e isso
 * cobre a tira e a grade — mas nenhum dos DIÁLOGOS, que é onde a fatia 7 pôs o
 * equipar, o usar, as melhorias e as ~160 linhas do catálogo. Eles justificam o
 * navegador por conta própria: são caixas que rolam dentro de si numa tela de
 * 390px, que é a forma que já transbordou nesta casa (ALE-178).
 */
test('a mochila abre a ficha do item e o catálogo sem estourar o telefone', async ({ page }) => {
  const id = await oIdDoHeroi(page, TANQUE)
  await page.setViewportSize({ width: 390, height: 844 })
  await page.goto(`/piloto/personagens/${id}?tab=bag`)

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
 * O caminhar pelas sete abas mede a lista do primeiro herói do elenco. O que ele
 * não alcança é o DIÁLOGO — que leva os ~93 poderes eletivos de uma classe numa
 * caixa que rola dentro de si, e é a forma que já transbordou nesta casa
 * (ALE-178). E o contador de degraus da postura, que só existe para quem tem
 * uma.
 */
test('os poderes abrem o diálogo de escolher sem estourar o telefone', async ({ page }) => {
  const id = await oIdDoHeroi(page, TANQUE)
  await page.setViewportSize({ width: 390, height: 844 })
  await page.goto(`/piloto/personagens/${id}?tab=abilities`)

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

test('as sete abas são endereços, e a ativa se anuncia', async ({ page }) => {
  const id = await aFichaDoPrimeiro(page)

  const abas = page.getByRole('navigation', { name: 'Seções da ficha' }).getByRole('link')
  await expect(abas).toHaveCount(7)

  // A ABA É UM ENDEREÇO: recarregar tem de cair na mesma seção. É o contrato que
  // a SPA tinha e que um sinal de cliente não daria — e é por isso que elas são
  // links e não botões.
  await page.goto(`/piloto/personagens/${id}?tab=spells`)
  await expect(page.getByRole('link', { name: 'Magias', exact: true })).toHaveAttribute('aria-current', 'page')
  await page.reload()
  await expect(page.getByRole('link', { name: 'Magias', exact: true })).toHaveAttribute('aria-current', 'page')

  // E o endereço ANTIGO da Mochila continua chegando nela: `inventory` é
  // favorito de quando a aba se chamava assim.
  await page.goto(`/piloto/personagens/${id}?tab=inventory`)
  await expect(page.getByRole('link', { name: 'Mochila', exact: true })).toHaveAttribute('aria-current', 'page')
})
