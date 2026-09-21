import { expect, test } from '@playwright/test'
import { expectDentroDaJanela } from './support/geometry'
import { VIEWPORTS, expectNoHorizontalOverflow } from './support/viewports'

const CAMPAIGN = '/campanhas/1' // Snapshot Test ALE-33 (seed)

test.describe('Detalhe da campanha', () => {
  // 'troca de aba mostra o roster de membros' saiu daqui de propósito: clique
  // numa aba e nomes na tela não pedem browser nenhum. A garantia está partida
  // em duas, cada metade na camada que a sustenta:
  //   - a SEÇÃO É ENDEREÇO (link, URL, e o botão voltar) tem e2e próprio no
  //     bloco `A crônica`, no fim deste arquivo — histórico é do navegador e
  //     jsdom não o tem;
  //   - o ROSTER desenhado é `TestTheGmComesFirstInTheCast`, em
  //     `api/campaigns_one_test.go`, que é a camada mais barata que o sustenta.
})

/**
 * Criar e excluir uma campanha: o único caso daqui que ESCREVE de verdade, e
 * por isso ele apaga o que criou — a seed é compartilhada com todos os specs, e
 * uma execução que deixa campanha para trás envenena a próxima.
 *
 * Entrar por convite NÃO mora aqui de propósito: quem responde é o
 * `api/campaigns_join_test.go`, mais a `invite.spec.ts` para o que só o browser
 * vê. A página `/campanhas/entrar` continua sendo carregada nos seis formatos
 * pelo bloco responsivo abaixo.
 */
test.describe('Abrir e fechar uma campanha', () => {
  test('criar leva direto para a nova campanha, e excluir traz de volta', async ({
    page,
  }) => {
    const name = `E2E Descartável ${Date.now()}`
    await page.goto('/campanhas/nova')

    await page.getByLabel('Nome').fill(name)
    await page.getByLabel('Descrição').fill('Criada e excluída pelo E2E.')
    await page.getByRole('button', { name: 'Abrir campanha' }).click()

    await expect(page).toHaveURL(/\/campanhas\/\d+/)
    await expect(page.getByRole('heading', { name, level: 1 })).toBeVisible()

    // A limpeza é pela tela, que de quebra exercita o caminho de exclusão.
    await page.goto(`${new URL(page.url()).pathname}?tab=config`)
    await page.getByRole('button', { name: /Excluir campanha/ }).click()
    await page.getByRole('dialog').getByRole('button', { name: 'Excluir' }).click()
    // Volta para a LISTA, e o que se afirma é a LISTA e não a URL dela: prender
    // o teste ao endereço o faria quebrar num renome de rota que não muda nada
    // para quem usa.
    await expect(page.getByRole('listbox', { name: 'Campanhas' })).toBeVisible()
    await expect(page.getByRole('option', { name: new RegExp(name) })).toHaveCount(0)
  })
})

/**
 * Um teste por cena, com os seis formatos DENTRO dele — e não um teste por par.
 * O leiaute responde ao `setViewportSize` na hora (media query é só de largura,
 * regra da casa), então pagar uma carga de página por formato não comprava
 * nada. Ver `support/viewports.ts` para o que isto prova e o que não prova.
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
   * O `expectNoHorizontalOverflow` acima passa VERDE sobre um botão pintado
   * fora da janela, e não por descuido: o `overflow-x-hidden` da cena zera o
   * `scrollWidth` da raiz. É preciso medir contra a JANELA, e é o que esta
   * asserção faz. (A causa típica é `min-width: auto` em item de grid, que
   * dimensiona a trilha pelo MIN-CONTENT.)
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
   * Apertar o respiro por `max-height` casa DUAS situações diferentes: o
   * celular deitado, que é quem a consulta quer atender, e o celular EM PÉ com
   * o teclado virtual aberto (390x844 vira ~390x494). Estas folhas hospedam
   * campo de texto, então o respiro encolheria debaixo do dedo no meio da
   * digitação. É por isso que a regra da casa manda chavear por LARGURA.
   *
   * O teste afirma as DUAS metades de propósito: só a primeira passaria verde
   * com a tampa simplesmente APAGADA, que é o conserto errado. Medido no
   * deitado, com a tampa aparecem 89% do botão e sem ela apenas 31% — o limiar
   * de 0,8 fica com folga dos dois lados desse vão. `ratio: 1` seria exigir
   * zero pixel de sobra, que é asserção que depende da fonte instalada.
   *
   * Só e2e: em jsdom não há viewport e nenhuma media query resolve.
   */
  test('o respiro da folha não muda quando o teclado abre, e o deitado continua cabendo', async ({
    page,
  }) => {
    await page.goto('/campanhas/nova')
    await expect(page.getByRole('heading', { name: /Abrir nova campanha/i })).toBeVisible()

    // A testemunha é o RESPIRO da folha, e ele se lê no valor computado e não
    // na posição de um filho. Tentei os dois caminhos posicionais e os dois
    // medem outra coisa: no vertical entram as margens automáticas que centram
    // o formulário (173px numa janela alta contra 34 numa baixa, com o mesmo
    // respiro nas duas), e no horizontal entra 1px de arredondamento de barra
    // de rolagem — ruído da ordem do sinal, que era de 24px para 16.
    const breathingRoom = () =>
      page
        .locator('[data-tome-root]')
        .evaluate((el) => `${getComputedStyle(el).paddingLeft}/${getComputedStyle(el).rowGap}`)

    await page.setViewportSize({ width: 390, height: 844 })
    const noKeyboard = await breathingRoom()

    // O teclado virtual do celular não muda a largura, só a altura.
    await page.setViewportSize({ width: 390, height: 494 })
    const withKeyboard = await breathingRoom()

    expect(
      withKeyboard,
      'o respiro da folha encolheu quando o teclado abriu — ela está chaveando por ALTURA',
    ).toBe(noKeyboard)

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
    const options = page.getByRole('option')
    await expect(options.first()).toBeVisible()

    const first = (await options.first().textContent())?.trim() ?? ''
    const second = (await options.nth(1).textContent())?.trim() ?? ''
    expect(first, 'a seed precisa de duas campanhas para este guarda').not.toBe(second)

    let requests = 0
    page.on('request', () => {
      requests++
    })

    await options.first().focus()
    // Seta DIREITA e não abaixo: a listagem virou uma tira deitada no rodapé,
    // igual à de personagens, e o driver lê `data-nav-layout="row"`.
    await page.keyboard.press('ArrowRight')

    await expect(options.nth(1)).toHaveAttribute('aria-selected', 'true')
    await expect(options.first()).toHaveAttribute('aria-selected', 'false')
    expect(requests, 'andar no trilho foi à rede — o cursor deixou de ser sinal').toBe(0)
  })

  /**
   * A busca é do SERVIDOR, e quem prende o RESULTADO dela é o
   * `api/search_test.go` — inclusive o caso que mais surpreende, "tauron"
   * casando "Segredos de Wynlla" por subsequência na sinopse.
   *
   * O que sobra aqui é o que só o navegador vê: que a busca de fato FILTRA a
   * lista renderizada.
   */
  test('a busca filtra a lista que o servidor desenhou', async ({ page }) => {
    await page.goto('/campanhas')
    // A VAGA conta como opção no trilho e nunca é filtrada, então ela sai do
    // número: contar o trilho inteiro faria o guarda medir "3 achados" onde a
    // busca achou 2.
    const campaigns = page.getByRole('option').filter({ hasNotText: 'Folha em branco' })
    expect(await campaigns.count(), 'a seed precisa de mais de duas campanhas').toBeGreaterThan(2)

    await page.getByRole('searchbox', { name: 'Buscar campanha' }).fill('tauron')
    await expect(campaigns).toHaveCount(3)
    await expect(page.getByRole('option', { name: /A Queda de Tauron/ })).toBeVisible()

    await page.getByRole('searchbox', { name: 'Buscar campanha' }).fill('zzzzzz')
    await expect(page.getByText(/Nenhuma campanha combina/)).toBeVisible()
  })
})

test.describe('A folha em branco', () => {
  test.use({ storageState: '.auth/user.json' })

  // Tela nova se valida nos seis formatos, e aqui importa mais que de costume:
  // a folha hospeda campos de texto, e num telefone deitado o botão de enviar
  // é o primeiro a cair para fora da tela.
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
   * aqui é que o navegador PARA a digitação no limite, em vez de deixar
   * escrever 3000 caracteres e recusar no fim.
   */
  test('a recusa devolve o texto, e o limite avisa enquanto se escreve', async ({ page }) => {
    await page.goto('/campanhas/nova')
    const description = page.getByLabel('Descrição')

    // O `maxlength` nativo é o aviso durante a digitação.
    await description.fill('x'.repeat(2500))
    expect((await description.inputValue()).length, 'o navegador deixou passar do teto').toBe(2000)

    // Nome de puros espaços: o `required` não pega, o servidor pega.
    await page.getByLabel('Nome').fill('   ')
    await description.fill('A caravana parte de Valkaria ao amanhecer.')
    await page.getByRole('button', { name: 'Abrir campanha' }).click()

    await expect(page.getByText(/O nome é obrigatório/)).toBeVisible()
    await expect(description, 'a descrição sumiu na recusa').toHaveValue(
      'A caravana parte de Valkaria ao amanhecer.',
    )
  })

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
   * A ABA É ENDEREÇO, e é essa a decisão que a cena inteira apoia: o `?tab=`
   * chega com o pedido, e o que se afirma é a consequência disso para quem usa
   * — o link é colável e o histórico funciona.
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
   * campanha 1 para o próximo teste.
   */
  test('alternar a regra opcional troca o estado sem recarregar a página', async ({ page }) => {
    await page.goto('/campanhas/1?tab=config')
    const key = page.getByRole('switch', { name: 'Limites de carga' })
    const before = await key.getAttribute('aria-checked')

    // A navegação NÃO pode acontecer: se acontecesse, este marcador sumiria.
    await page.evaluate(() => {
      ;(window as unknown as { __mesmaPagina: boolean }).__mesmaPagina = true
    })

    await key.click()
    await expect(key, 'o interruptor não trocou de estado').not.toHaveAttribute(
      'aria-checked',
      before ?? '',
    )
    expect(
      await page.evaluate(() => (window as unknown as { __mesmaPagina?: boolean }).__mesmaPagina),
      'a página recarregou — o remendo virou navegação',
    ).toBe(true)

    await key.click()
    await expect(key, 'o teste não devolveu a regra ao estado original').toHaveAttribute(
      'aria-checked',
      before ?? '',
    )
  })

})
