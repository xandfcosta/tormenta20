import { expect, test } from '@playwright/test'
import { expectDentroDaJanela } from './support/geometry'
import { VIEWPORTS, expectNoHorizontalOverflow } from './support/viewports'

const CAMPAIGN = '/campanhas/1' // Snapshot Test ALE-33 (seed)

test.describe('Detalhe da campanha', () => {
  // 'troca de aba mostra o roster de membros' saiu na ALE-187: clique numa aba
  // e nomes na tela, sem medida que precise de browser.
  //
  // O DESTINO que aquela issue escreveu era `campaign-detail-page.test.tsx`, e
  // ele NÃO existe mais nesta branch: a migração apagou a página da SPA que ele
  // montava, porque a crônica virou cena do servidor (ALE-255). O ponteiro
  // quebrou no merge que trouxe a poda, e ficaria mandando procurar num arquivo
  // ausente — que é a forma mais cara de perder um guarda, porque parece que ele
  // existe.
  //
  // Aqui a garantia está partida em duas, cada metade na camada que a sustenta:
  //   - a SEÇÃO É ENDEREÇO (link, URL, e o botão voltar) tem e2e próprio no
  //     bloco `A crônica`, no fim deste arquivo — histórico é do navegador e
  //     jsdom não o tem;
  //   - o ROSTER desenhado é `TestTheGmComesFirstInTheCast`, em
  //     `api/campaigns_one_test.go`, que é a camada mais barata que o sustenta.
})

/**
 * Criar e excluir uma campanha (ALE-79, ALE-80): o único caso daqui que ESCREVE
 * de verdade, e por isso ele apaga o que criou — a seed é compartilhada com
 * todos os specs e uma execução que deixa campanha para trás envenena a
 * próxima.
 *
 * O bloco "Entrar por convite" saiu na ALE-144, e DOIS dos três destinos que ela
 * escreveu não existem mais nesta branch: a carta de convite virou cena do
 * servidor, e com ela foram embora o `entities/campaign/join-target.test.ts` e o
 * `features/campaign-join/hero-picker.test.tsx`. Quem responde agora é o
 * `api/campaigns_join_test.go` — a resolução do alvo em
 * `TestWithoutAnInviteSomeoneElsesTableIsRefusedWithTheNextStep` e o convite morto em
 * `TestADeadInviteBecomesASentenceAndNotABrokenPage` —, mais a
 * `invite.spec.ts` para o que só o browser vê. A prévia do convite continua em
 * (`entities/queries.test.ts`), esse sobreviveu, e a
 * página `/campaigns/join` continua sendo carregada nos seis formatos pelo
 * bloco responsivo abaixo.
 */
test.describe('Abrir e fechar uma campanha', () => {
  test('criar leva direto para a nova campanha, e excluir traz de volta', async ({
    page,
  }) => {
    const name = `E2E Descartável ${Date.now()}`
    await page.goto('/campaigns/new')

    await page.getByLabel('Nome').fill(name)
    await page.getByLabel('Descrição').fill('Criada e excluída pelo E2E.')
    await page.getByRole('button', { name: 'Abrir campanha' }).click()

    // Landed on the new chronicle's own page — a do SERVIDOR desde a ALE-255.
    await expect(page).toHaveURL(/\/campanhas\/\d+/)
    await expect(page.getByRole('heading', { name, level: 1 })).toBeVisible()

    // Clean up through the UI, which also exercises the ALE-79 delete path.
    await page.goto(`${new URL(page.url()).pathname}?tab=config`)
    await page.getByRole('button', { name: /Excluir campanha/ }).click()
    await page.getByRole('dialog').getByRole('button', { name: 'Excluir' }).click()
    // Volta para a LISTA, e o que se afirma é a lista — não a URL dela. Desde a
    // ALE-234 a cena é do servidor e `/campaigns` encaminha para
    // `/campanhas`; prender o teste ao endereço faria ele quebrar de novo
    // quando o prefixo `/` cair, sem que nada de verdade tivesse mudado.
    await expect(page.getByRole('listbox', { name: 'Campanhas' })).toBeVisible()
    await expect(page.getByRole('option', { name: new RegExp(name) })).toHaveCount(0)
  })
})

/**
 * One test per scene, six viewports inside each — not one test per pair. The
 * layout answers `setViewportSize` live (media queries are width-only, a house
 * rule), so paying a full page load per viewport bought nothing: this block was
 * 18 tests and 134s. See `support/viewports.ts` for what it does and does not
 * prove.
 */
const SCENES = [
  { name: 'detalhe', path: `${CAMPAIGN}?tab=membros`, heading: /Snapshot Test ALE-33/i },
  { name: 'nova', path: '/campanhas/nova', heading: /Abrir nova campanha/i },
  { name: 'convite', path: '/campanhas/entrar', heading: /Entrar na mesa/i },
]

test.describe('Campanha — responsivo (sem overflow horizontal)', () => {
  for (const scene of SCENES) {
    test(`${scene.name}: sem scroll horizontal nos seis formatos`, async ({ page }) => {
      await page.goto(scene.path)
      await expect(page.getByRole('heading', { name: scene.heading })).toBeVisible()

      await expectNoHorizontalOverflow(page, VIEWPORTS)
    })
  }

  /**
   * O defeito da ALE-160: a 390px os painéis da visão geral eram pintados 169px
   * fora do pai, e o botão "Convite" ia parar em x 392–487 numa tela de 390 —
   * fora da janela e sem eixo que rolasse até ele. A causa é `min-width: auto`
   * em item de grid, que dimensiona a trilha pelo MIN-CONTENT: 457px numa caixa
   * de 288.
   *
   * O `expectNoHorizontalOverflow` acima passava VERDE sobre isso, e não por
   * descuido: o `overflow-x-hidden` da cena zera o `scrollWidth` da raiz. É
   * preciso medir contra a JANELA, e é o que a asserção nova faz.
   */
  test('nada clicável fica fora da janela na campanha, em nenhum formato', async ({ page }) => {
    await page.goto(`${CAMPAIGN}?tab=visao`)
    await expect(page.getByRole('heading', { name: /Snapshot Test ALE-33/i })).toBeVisible()

    for (const viewport of VIEWPORTS) {
      await page.setViewportSize({ width: viewport.width, height: viewport.height })
      await expectDentroDaJanela(page)
    }
  })

  /**
   * ALE-176. A folha do grimório apertava o respiro por `max-height: 520px`, e
   * essa consulta casa DUAS situações diferentes: o celular deitado, que é
   * quem ela queria atender, e o celular EM PÉ com o teclado virtual aberto
   * (390x844 vira ~390x494). Estas folhas hospedam campo de texto — "nova
   * campanha" e o convite —, então o respiro encolhia debaixo do dedo no meio
   * da digitação. É por isso que a regra da casa manda chavear por LARGURA.
   *
   * O teste afirma as DUAS metades de propósito. Só a primeira passaria verde
   * com a tampa simplesmente APAGADA, que é o conserto errado: medido no
   * deitado, com a tampa aparecem 89% do botão "Abrir campanha" e sem ela
   * apenas 31% (y=379,6..412,7 numa janela de 390). O limiar de 0,8 fica com
   * folga dos dois lados desse vão, e não é número mágico: está aqui porque a
   * tampa não faz o botão CABER — ela o traz de quase escondido para quase
   * inteiro, e prender `ratio: 1` seria exigir zero pixel de sobra, que é o
   * tipo de asserção que a ALE-184 mostrou depender da fonte instalada.
   *
   * Só e2e: em jsdom não há viewport e nenhuma media query resolve.
   */
  test('o respiro da folha não muda quando o teclado abre, e o deitado continua cabendo', async ({
    page,
  }) => {
    await page.goto('/campaigns/new')
    await expect(page.getByRole('heading', { name: /Abrir nova campanha/i })).toBeVisible()

    // A testemunha é o RESPIRO da folha, e ele se lê no valor computado e não
    // na posição de um filho. Tentei os dois caminhos posicionais e os dois
    // medem outra coisa: no vertical entram as margens automáticas que centram
    // o formulário (173px numa janela alta contra 34 numa baixa, com o mesmo
    // respiro nas duas), e no horizontal entra 1px de arredondamento de barra
    // de rolagem — ruído da ordem do sinal, que era de 24px para 16.
    const respiro = () =>
      page
        .locator('[data-tome-root]')
        .evaluate((el) => `${getComputedStyle(el).paddingLeft}/${getComputedStyle(el).rowGap}`)

    await page.setViewportSize({ width: 390, height: 844 })
    const semTeclado = await respiro()

    // O teclado virtual do celular não muda a largura, só a altura.
    await page.setViewportSize({ width: 390, height: 494 })
    const comTeclado = await respiro()

    expect(
      comTeclado,
      'o respiro da folha encolheu quando o teclado abriu — ela está chaveando por ALTURA',
    ).toBe(semTeclado)

    await page.setViewportSize({ width: 844, height: 390 })
    await expect(
      page.getByRole('button', { name: /Abrir campanha/i }),
      'no celular deitado o botão que fecha a tarefa saiu da tela',
    ).toBeInViewport({ ratio: 0.8 })
  })
})

test.describe('A cena de campanhas', () => {
  test.use({ storageState: '.auth/user.json' })

  /**
   * O cursor segue o FOCO, e trocar de campanha não custa requisição.
   *
   * É a decisão que governa a cena: o servidor manda todos os livros e o
   * `data-show` escolhe um. Se alguém trocar isso por uma ida ao servidor por
   * passo, navegar por teclado vira uma conversa com a rede — e este guarda é
   * o que avisa, porque ele conta as requisições.
   *
   * E2E porque foco e geometria são do navegador: em jsdom todo elemento mede
   * zero e o driver não teria como escolher o vizinho.
   */
  test('as setas trocam de campanha sem pedir nada ao servidor', async ({ page }) => {
    await page.goto('/campanhas')
    const opcoes = page.getByRole('option')
    await expect(opcoes.first()).toBeVisible()

    const primeira = (await opcoes.first().textContent())?.trim() ?? ''
    const segunda = (await opcoes.nth(1).textContent())?.trim() ?? ''
    expect(primeira, 'a seed precisa de duas campanhas para este guarda').not.toBe(segunda)

    let pedidos = 0
    page.on('request', () => {
      pedidos++
    })

    await opcoes.first().focus()
    // Seta DIREITA e não abaixo: a listagem virou uma tira deitada no rodapé,
    // igual à de personagens, e o driver lê `data-nav-layout="row"`.
    await page.keyboard.press('ArrowRight')

    await expect(opcoes.nth(1)).toHaveAttribute('aria-selected', 'true')
    await expect(opcoes.first()).toHaveAttribute('aria-selected', 'false')
    expect(pedidos, 'andar no trilho foi à rede — o cursor deixou de ser sinal').toBe(0)
  })

  /**
   * A busca é do SERVIDOR, e o guarda que comparava as DUAS telas lado a lado
   * morreu com a virada (ALE-234): a tela da SPA não existe mais, então não há
   * segundo lado para comparar.
   *
   * A garantia não ficou órfã — ela desceu para onde é mais barata e mais
   * exata. Os sete casos de `api/search_test.go` foram conferidos um a um rodando o
   * `match-sorter` de verdade, incluindo o que mais surpreende: "tauron" casa
   * "Segredos de Wynlla" por subsequência na sinopse, e a biblioteca faz igual.
   * Comparar duas telas era a forma cara de afirmar isso enquanto as duas
   * existiam.
   *
   * O que sobra aqui é o que só o navegador vê: que a busca de fato FILTRA a
   * lista renderizada.
   */
  test('a busca filtra a lista que o servidor desenhou', async ({ page }) => {
    await page.goto('/campanhas')
    // A VAGA conta como opção no trilho (ALE-297) e ela nunca é filtrada — o
    // que se conta aqui são as CAMPANHAS, então ela sai do número. Contar o
    // trilho inteiro faria o guarda medir "3 achados" onde a busca achou 2.
    const campanhas = page.getByRole('option').filter({ hasNotText: 'Folha em branco' })
    expect(await campanhas.count(), 'a seed precisa de mais de duas campanhas').toBeGreaterThan(2)

    await page.getByRole('searchbox', { name: 'Buscar campanha' }).fill('tauron')
    await expect(campanhas).toHaveCount(3)
    await expect(page.getByRole('option', { name: /A Queda de Tauron/ })).toBeVisible()

    await page.getByRole('searchbox', { name: 'Buscar campanha' }).fill('zzzzzz')
    await expect(page.getByText(/Nenhuma campanha combina/)).toBeVisible()
  })
})

test.describe('A folha em branco', () => {
  test.use({ storageState: '.auth/user.json' })

  // Tela nova se valida nos seis formatos. Aqui importa mais que de costume: a
  // folha hospeda campos de texto, e o espaçamento dela encolhe com a
  // ORIENTAÇÃO justamente porque num telefone deitado o botão de enviar caía
  // para fora da tela (ALE-176).
  test('a folha cabe nos seis formatos', async ({ page }) => {
    await page.goto('/campanhas/nova')
    await expect(page.getByRole('button', { name: 'Abrir campanha' })).toBeVisible()

    await expectNoHorizontalOverflow(page, VIEWPORTS)
    for (const viewport of VIEWPORTS) {
      await page.setViewportSize({ width: viewport.width, height: viewport.height })
      await expectDentroDaJanela(page)
    }
  })

  /**
   * A RECUSA NÃO COME O TEXTO — e este é o caso que só o navegador vê inteiro,
   * porque envolve o `maxlength` NATIVO e o envio de verdade do formulário.
   *
   * O guarda em Go afirma o mesmo pelo lado do servidor. O que se acrescenta
   * aqui é que o navegador PARA a digitação no limite em vez de deixar escrever
   * 3000 caracteres e recusar no fim: ver a pessoa chegar ao fim enquanto
   * escreve é melhor que perder o texto ao enviar.
   */
  test('a recusa devolve o texto, e o limite avisa enquanto se escreve', async ({ page }) => {
    await page.goto('/campanhas/nova')
    const descricao = page.getByLabel('Descrição')

    // O `maxlength` nativo é o aviso durante a digitação.
    await descricao.fill('x'.repeat(2500))
    expect((await descricao.inputValue()).length, 'o navegador deixou passar do teto').toBe(2000)

    // Nome de puros espaços: o `required` não pega, o servidor pega.
    await page.getByLabel('Nome').fill('   ')
    await descricao.fill('A caravana parte de Valkaria ao amanhecer.')
    await page.getByRole('button', { name: 'Abrir campanha' }).click()

    await expect(page.getByText(/O nome é obrigatório/)).toBeVisible()
    await expect(descricao, 'a descrição sumiu na recusa').toHaveValue(
      'A caravana parte de Valkaria ao amanhecer.',
    )
  })

  // Aqui morava `o endereço antigo /campaigns/new encaminha para a folha nova`.
  //
  // Ele media um 303 do servidor — sem mecanismo que só um navegador tenha, que
  // é a única justificativa de e2e que o guia aceita. Quem varre a tabela
  // INTEIRA é o `TestEveryLegacyAddressLandsOnAScene`, e ele confere também a
  // preservação de parâmetro que este caso guardava:
  //     {"/campaigns/new", "/campanhas/nova"}
  // Uma regra, uma camada: apagar este caso não muda nada que o Go não acuse.
})

test.describe('A crônica', () => {
  test.use({ storageState: '.auth/user.json' })

  test('a crônica cabe nos seis formatos', async ({ page }) => {
    await page.goto('/campanhas/1')
    // UM `h1` só, e é o nome da campanha: a casca só desenha o dela quando há
    // título, e a crônica não passa um — ela tem o próprio.
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible()

    await expectNoHorizontalOverflow(page, VIEWPORTS)
    for (const viewport of VIEWPORTS) {
      await page.setViewportSize({ width: viewport.width, height: viewport.height })
      await expectDentroDaJanela(page)
    }
  })

  /**
   * A ABA É ENDEREÇO, e é essa a decisão que a cena inteira apoia.
   *
   * Na SPA o `?tab=` já era o estado, mas a versão em React precisava espelhá-lo
   * num `useState` com dois efeitos e um debounce de 250ms para a troca não
   * travar. Aqui o parâmetro chega com o pedido — e o que se afirma é a
   * consequência disso para quem usa: o link é colável e o histórico funciona.
   *
   * E2E porque histórico é do navegador: `goBack` não existe em jsdom.
   */
  test('a seção é endereço: ela sobrevive ao link colado e ao botão voltar', async ({ page }) => {
    await page.goto('/campanhas/1?tab=membros')
    await expect(page.locator('[aria-current="page"]')).toHaveText('Membros')

    await page.getByRole('link', { name: 'Sessões', exact: true }).click()
    await expect(page.locator('[aria-current="page"]')).toHaveText('Sessões')

    await page.goBack()
    await expect(page.locator('[aria-current="page"]'), 'o voltar não devolveu a seção').toHaveText(
      'Membros',
    )
  })

  /**
   * O INTERRUPTOR das regras é a única ação da crônica que NÃO navega, e por
   * isso é a única com Datastar: alternar um ajuste no meio de uma lista e
   * recarregar a página perderia a posição de quem está lendo.
   *
   * E2E porque o remendo é SSE trocando um pedaço do DOM — o guarda em Go
   * prova o estado do banco, e este prova que a tela acompanhou sem recarregar.
   *
   * Ele DEVOLVE o interruptor ao estado original no fim: a regra é do banco de
   * desenvolvimento, e deixá-la desligada mudaria a carga de todo personagem da
   * campanha 1 para o próximo teste — a família de problema da ALE-238.
   */
  test('alternar a regra opcional troca o estado sem recarregar a página', async ({ page }) => {
    await page.goto('/campanhas/1?tab=config')
    const chave = page.getByRole('switch', { name: 'Limites de carga' })
    const antes = await chave.getAttribute('aria-checked')

    // A navegação NÃO pode acontecer: se acontecesse, este marcador sumiria.
    await page.evaluate(() => {
      ;(window as unknown as { __mesmaPagina: boolean }).__mesmaPagina = true
    })

    await chave.click()
    await expect(chave, 'o interruptor não trocou de estado').not.toHaveAttribute(
      'aria-checked',
      antes ?? '',
    )
    expect(
      await page.evaluate(() => (window as unknown as { __mesmaPagina?: boolean }).__mesmaPagina),
      'a página recarregou — o remendo virou navegação',
    ).toBe(true)

    await chave.click()
    await expect(chave, 'o teste não devolveu a regra ao estado original').toHaveAttribute(
      'aria-checked',
      antes ?? '',
    )
  })

  // Aqui morava `o endereço antigo /campanhas/:id encaminha COM a seção`.
  //
  // Ele media um 303 do servidor — sem mecanismo que só um navegador tenha, que
  // é a única justificativa de e2e que o guia aceita. Quem varre a tabela
  // INTEIRA é o `TestEveryLegacyAddressLandsOnAScene`, e ele confere também a
  // preservação de parâmetro que este caso guardava:
  //     {"/campaigns/12?tab=config", "/campanhas/12?tab=config"}
  // Uma regra, uma camada: apagar este caso não muda nada que o Go não acuse.
})
