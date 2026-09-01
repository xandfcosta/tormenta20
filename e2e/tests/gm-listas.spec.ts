import { expect, test } from '@playwright/test'
import {
  expectColunasMonotonicas,
  expectNadaEscapa,
  expectSemFaixaMorta,
} from './support/geometry'

/**
 * As listas do MESTRE — bestiário e catálogos, nas cenas do piloto.
 *
 * O arquivo nasceu medindo a `VirtualList` da SPA: ela mede as linhas para
 * saber quais existem, e em jsdom todo elemento mede zero — a lista renderiza
 * NENHUMA linha e um teste de unidade passa verde sobre a tela vazia. Foi assim
 * que a ALE-84 entrou em produção com a suíte inteira verde.
 *
 * Quatro casos foram embora com a SPA (ALE-272, fatia 10c): os que dirigiam a
 * sessão dela e a ficha antiga. O que ficou mede as cenas do servidor, e a
 * razão de serem e2e mudou de nome sem mudar de natureza — não é mais
 * virtualização, é LEIAUTE REAL: coluna que some ao alargar a janela e faixa
 * morta no tablet não existem em HTML nenhum.
 *
 * Só leitura: filtra e navega, nunca escreve.
 */
test.describe('As listas do mestre', () => {

  test('a ferramenta Bestiário pinta a lista e abre a criatura escolhida', async ({ page }) => {
    // A ferramenta virou cena do SERVIDOR na ALE-264, e o teste foi REAPONTADO
    // em vez de apagado: o que ele afirma — a lista pinta, a busca filtra, o
    // painel mostra a escolhida — continua sendo a promessa da tela, e o id
    // `mesa-bestiario` sobreviveu ao porte de propósito.
    //
    // A linha é LINK e não botão: abrir uma criatura passou a ser navegação,
    // com `?criatura=` no endereço.
    await page.goto('/piloto/mestre/bestiario')

    const busca = page.getByRole('searchbox', { name: 'Buscar criatura' })
    await expect(busca).toBeVisible()
    await busca.fill('ogro')

    const linha = page.getByRole('link', { name: /^Ogro/ }).first()
    await expect(linha).toBeVisible()
    await linha.click()

    await expect(page.getByRole('region', { name: 'Criatura escolhida' })).toContainText('Ogro')
  })


  /**
   * A sexta e última lista virtualizada da auditoria: o pool de poderes gerais,
   * dentro da ficha. Ele só pinta com o card da classe ABERTO, e o card abre
   * sozinho quando há escolha pendente — por isso o teste CRIA um Guerreiro de
   * 6º nível sem poder escolhido (três vagas em aberto) em vez de usar um
   * personagem da seed, onde as vagas já estão gastas e a lista fica fechada.
   *
   * O herói é criado pela API na primeira rodada e REUSADO nas seguintes: o app
   * não apaga personagem, então criar um por rodada entulharia o elenco.
   */
test('no tablet em pé, a lista do bestiário não deixa faixa morta', async ({ page }) => {
  await page.setViewportSize({ width: 768, height: 1024 })
  await page.goto('/piloto/mestre/bestiario')
  await expect(page.getByRole('link', { name: /ND / }).first()).toBeVisible()

  // Sem transbordo a asserção não prova nada: seria uma lista que coube.
  const transbordou = await page.evaluate(
    () =>
      [...document.querySelectorAll('*')].find(
        (n) => n.scrollHeight > n.clientHeight + 8 && n.clientHeight > 100,
      ) !== undefined,
  )
  expect(transbordou, 'a lista não transbordou — o teste não mediu nada').toBe(true)

  // A tolerância sobe de 8 para 12 px, e a razão é do INSTRUMENTO e não do
  // defeito: a primitiva mede o último elemento com TEXTO, e a lista da SPA era
  // virtualizada — linhas posicionadas exatamente, sem espaço próprio embaixo.
  // Sem virtualização (ALE-257) cada linha tem `p-2`, então o texto da última
  // fica legitimamente ~10px acima do fim do contêiner. Medido: 11px.
  //
  // Isto NÃO cega o guarda, e a diferença é de duas ordens de grandeza: o
  // defeito da ALE-175 eram 243px de banda morta. Provado — recolocar uma tampa
  // de altura na lista deixa este teste VERMELHO com a tolerância em 12.
  await expectSemFaixaMorta(page, '[aria-labelledby=mesa-bestiario]', 12)
})

/**
 * Crescer a janela nunca tira uma coluna do bestiário (ALE-172).
 *
 * O gate das duas colunas olhava a JANELA (`lg:`), e a coluna de ferramentas
 * do `/gm` devolve largura à direita conforme a janela encolhe. O resultado
 * era invertido: numa janela de 1024 o palco recebia 800px e mostrava DUAS
 * colunas, e numa de 1000 recebia 968px e mostrava UMA. O mestre alargava a
 * janela e perdia o painel de detalhe.
 *
 * A varredura é de LARGURA com altura fixa de propósito. A decisão real ali é
 * "cabe painel lateral?", que tem duas dimensões: o mesmo contêiner de 812px
 * cabe num tablet deitado (768px de altura) e não cabe num celular deitado
 * (390px, onde a lista sobraria com 41px — menos que uma linha). Por isso o
 * conserto é `container-type: size` com as duas condições, e por isso comparar
 * caixas de alturas diferentes acusaria como defeito a exceção que É o
 * conserto.
 *
 * Por que e2e: media/container query só resolve em browser de verdade. Em
 * jsdom nenhuma consulta casa e a grade responde sempre a mesma coisa.
 */
test('alargar a janela nunca tira uma coluna do bestiário', async ({ page }) => {
  await page.goto('/piloto/mestre/bestiario')
  await expect(page.getByRole('link', { name: /ND / }).first()).toBeVisible()

  await expectColunasMonotonicas(
    page,
    '[aria-labelledby=mesa-bestiario] div.grid',
    [1920, 1440, 1200, 1100, 1040, 1024, 1000, 950, 900, 860, 844, 830, 812, 800, 768, 390],
  )
})

/**
 * As colunas do catálogo seguem o PAINEL, e alargar nunca tira uma (ALE-170).
 *
 * Mesma classe de defeito que a ALE-172 consertou no bestiário, e por isso o
 * mesmo guarda: a ferramenta divide a tela com a trilha do `/gm`, então largura
 * de janela mente por centenas de pixels sobre quanto espaço o painel tem.
 *
 * A segunda asserção é a que só o browser faz. Numa lista VIRTUALIZADA "três
 * colunas" não é grade de CSS — é o agrupamento dos dados antes de entregá-los.
 * A grade pode declarar três colunas com um cartão só em cada fileira, e a tela
 * fica com dois terços de vazio à direita enquanto o CSS jura que está certo.
 * Contar os cartões DENTRO da fileira é o que separa as duas metades.
 */
test('alargar a janela nunca tira uma coluna do catálogo', async ({ page }) => {
  // A lista deixou de ser VIRTUALIZADA na ALE-258 — são 992 entradas e o
  // servidor manda todas —, então o alvo passa a ser a grade de verdade e não
  // a fileira que o virtualizador montava. A garantia é a mesma: alargar a
  // janela nunca pode tirar uma coluna.
  await page.goto('/piloto/mestre/condicoes')
  await expect(page.locator('.acervo-em-colunas').first()).toBeVisible()

  await expectColunasMonotonicas(
    page,
    '.acervo-em-colunas',
    [1920, 1440, 1200, 1100, 1024, 1000, 900, 844, 768, 600, 390],
  )

  // A segunda metade MUDOU DE PERGUNTA com a virada, e vale dizer por quê.
  //
  // Na lista virtualizada, "três colunas" não era grade de CSS — era o
  // agrupamento dos dados antes de entregá-los, e a grade podia declarar três
  // com um cartão só na fileira, deixando dois terços de vazio enquanto o CSS
  // jurava estar certo. Contar cartões DENTRO da fileira separava as duas
  // metades.
  //
  // Sem virtualização a grade é nativa e preenche sozinha, então essa
  // discrepância não pode existir. O que PODE existir é o oposto, e é o que se
  // afirma agora: a grade declarando MAIS colunas do que cabem — foi o defeito
  // medido na ALE-258, quatro colunas a 1920 onde o teto de leitura é três.
  await page.setViewportSize({ width: 1920, height: 1080 })
  const colunas = await page.evaluate(() => {
    const grade = document.querySelector('.acervo-em-colunas')
    if (!grade) return null
    return getComputedStyle(grade).gridTemplateColumns.split(' ').filter(Boolean).length
  })
  expect(colunas, 'nenhuma grade pintou em 1920').not.toBeNull()
  expect(colunas, 'o teto de três colunas é medida de leitura (ALE-170)').toBeLessThanOrEqual(3)
})

/**
 * As abas do catálogo DIVIDEM a faixa que recebem (ALE-138).
 *
 * `Condições | Magias | Poderes | Itens` ficavam encolhidas à esquerda com a
 * faixa inteira sobrando à direita. O gatilho do kit já nasce `flex-1`, mas o
 * `TabsList` nasce `inline-flex w-fit`: sem largura, o `flex-1` não tem o que
 * dividir. O print do dono mostrava a barra de cima ocupando tudo e a de dentro
 * não — mesma tela, dois comportamentos.
 *
 * A segunda asserção é a armadilha da ALE-122: junto de `flex-1` o `min-w-0` é
 * obrigatório, porque um item flex não encolhe abaixo do conteúdo e o rótulo
 * mais longo empurra a última parada para FORA do trilho. Ela NÃO reproduz um
 * defeito de hoje. Ela protege o próximo: um rótulo mais longo, uma parada a
 * mais, e o trilho estoura sem ninguém ver.
 *
 * As duas larguras não são simetria: a régua é o CONTÊINER, e a mesma janela dá
 * a largura inteira na Mesa e ~384px na gaveta da sessão (ALE-138, ALE-172).
 *
 * Por que e2e: é caixa contra caixa. Em jsdom todo elemento mede zero e
 * `expectNadaEscapa` passaria verde sobre qualquer arranjo.
 */
test('o trilho do mestre segura todas as paradas em qualquer largura', async ({ page }) => {
  // Uma navegação só, redimensionando depois — o mesmo padrão dos guardas de
  // coluna aqui do lado. Recarregar por largura paga o portão dos catálogos
  // (18 buscas antes da primeira tela) a cada volta, e foi assim que a versão
  // anterior deste teste estourou o timeout sem que nada estivesse errado.
  // A FILEIRA DE ABAS que este guarda media não existe mais: na ALE-264 cada
  // catálogo virou uma parada do TRILHO, e ter as duas coisas seria o mesmo
  // estado desenhado em dois lugares. A garantia não foi apagada — ela MUDOU DE
  // ENDEREÇO junto com o risco: eram quatro abas numa faixa, são treze paradas
  // num trilho que no telefone rola na horizontal.
  await page.goto('/piloto/mestre/condicoes')
  const trilho = 'nav[aria-label="Ferramentas do mestre"]'
  await expect(page.getByRole('link', { name: 'Condições' })).toBeVisible()

  // A CONTAGEM saiu daqui e virou `TestOTrilhoTemUmaParadaPorCatalogo` no Go,
  // por amostragem sobre `abasDoAcervo`. Ela estava escrita como o número onze,
  // e as duas paradas nascidas depois (escolas, perícias) só apareceram quando
  // ele ficou vermelho por um número velho — enumeração cobrando manutenção sem
  // proteger nada. Aqui fica o que só o navegador testemunha: a geometria.
  let referencia = 0

  for (const largura of [1920, 1024, 768, 390]) {
    await page.setViewportSize({ width: largura, height: 900 })
    await expect(page.getByRole('link', { name: 'Condições' })).toBeVisible()

    // Nenhuma parada escapa do trilho — a da ALE-122: `flex-1` não encolhe
    // abaixo do conteúdo, e sem `min-w-0` o rótulo mais longo empurra a última
    // para fora.
    await expectNadaEscapa(page, trilho)

    // E TODAS continuam alcançáveis: no laptop em coluna, no telefone
    // rolando. Uma parada que some da tela é uma ferramenta que deixou de
    // existir para quem está naquela largura — o defeito da ALE-178, que fez o
    // ✕ de encerrar ficar inalcançável a 390px.
    const alcancaveis = await page.locator(`${trilho} a`).count()
    if (largura === 1920) referencia = alcancaveis
    expect(alcancaveis, `a ${largura}px o trilho perdeu paradas`).toBe(referencia)
  }
})
})
