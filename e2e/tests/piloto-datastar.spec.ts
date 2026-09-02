import { expect, test } from '@playwright/test'
import { VIEWPORTS, expectNoHorizontalOverflow } from './support/viewports'
import { expectDentroDaJanela } from './support/geometry'
import { medeOContraste, textoComContrasteBaixo } from './support/contraste'

/**
 * As DUAS telas do piloto Datastar (ALE-219): a Mesa do jogador e a
 * administração.
 *
 * Este é o único spec do piloto, e ele existe por UM motivo que nenhuma outra
 * camada cobre: contraste exige converter oklch para sRGB, e só o navegador faz
 * isso. Em jsdom o `getComputedStyle` devolve o oklch cru, e ler aqueles três
 * números como RGB dá razão inventada. É o mesmo motivo do guarda irmão em
 * `grimorio.spec.ts`, e a técnica de medição é a de lá.
 *
 * Ele nasceu de um defeito REAL: o crachá de condição saiu com a tinta de
 * penalidade sobre o próprio preenchimento e dava **2,36:1** contra os 4,5 do
 * AA — exatamente o erro que a ALE-200 consertou no botão destrutivo, cometido
 * de novo numa superfície nova. Nenhum teste do repositório o teria pego, e eu
 * só o vi porque medi.
 *
 * A página é do Go, não da SPA: o `baseURL` do Playwright é o Vite, e é o proxy
 * `/` do `vite.config.ts` que a alcança. Se o piloto for apagado, este
 * arquivo vai junto.
 */
test.use({ storageState: '.auth/player.json' })

test.describe('Mesa do jogador (piloto Datastar)', () => {
  test.use({ storageState: '.auth/player.json' })

  test('nenhum texto fica abaixo do mínimo de contraste do AA', async ({ page }) => {
    await page.goto('/mesa/1/4')
    // `exact`, e isto é conserto de um defeito LATENTE deste arquivo (ALE-234):
    // com a cena começada aparece um segundo `<h2>`, "Registrar iniciativa ·
    // <nome>", e o localizador casava os dois — strict mode violation. Ele
    // passava porque, quando o guarda foi escrito, nenhum spec antes dele
    // deixava cena ativa; passou a falhar de forma intermitente conforme a
    // ORDEM da suíte, que é o pior jeito de um teste falhar.
    await expect(page.getByRole('heading', { name: 'Iniciativa', exact: true })).toBeVisible()

    expect(await textoComContrasteBaixo(page), 'texto abaixo do AA na Mesa').toEqual([])
  })
})

/**
 * A MESMA cena, medida pelo OUTRO papel (ALE-276).
 *
 * O caso acima entra com `.auth/player.json`, e por isso mediu METADE da Mesa
 * durante toda a vida dele. Não é uma cena esquecida numa lista — é a segunda
 * forma de "um guarda só mede o que VISITA" que o CLAUDE.md descreve: a tela
 * RAMIFICA pelo dado, e percorrer a navegação não é cobertura. O diálogo de
 * configuração da sessão nasce dentro de um `if v.Mestre != nil`, então para o
 * jogador ele não existe no HTML, e o guarda passou por cima dele desde sempre.
 *
 * # O QUARTO texto da issue não mora aqui, e isso é para saber
 *
 * A ALE-276 lista quatro reprovados; este caso mede TRÊS. O que falta é o
 * `Encerrar` do diálogo "Encerrar o tabuleiro?", que só existe no HTML quando
 * há tabuleiro aberto — a mesma família de novo, um degrau abaixo: a cena
 * ramifica pelo dado, e a sessão 4 não tem tabuleiro. Ele foi consertado pelo
 * TOKEN e não por este caso, e não vale abrir um tabuleiro aqui só para
 * remedi-lo: ele usa exatamente o par de tinta que o "Excluir a sessão" já
 * prende três linhas abaixo, e uma regra se prende UMA vez. Quem o segura na
 * camada barata é o `TestEveryHouseTintExistsInTheStylesheet`, que varre a
 * FONTE e não depende de estado nenhum do banco.
 *
 * # O clique é CONTROLE, e não o que torna o texto mensurável
 *
 * O medidor já conta o diálogo FECHADO — ele olha o `display` do PRÓPRIO nó, e
 * um filho de um `<dialog>` fechado tem o `display` dele. Abrir não muda o
 * denominador (medido na ficha, ALE-272: 809 antes e 809 depois do clique). O
 * clique está aqui para provar que a superfície do mestre EXISTE nesta página:
 * sem ele, o dia em que o botão sumisse deixaria este caso VERDE medindo a Mesa
 * do jogador com outro login, que é exatamente o defeito que a issue veio
 * consertar.
 *
 * O denominador é a outra metade do controle, e é para isso que o medidor
 * devolve `medidos`: "nada reprovou" e "não medi nada" são a mesma cor no
 * terminal.
 */
test.describe('Mesa do mestre (piloto Datastar)', () => {
  test.use({ storageState: '.auth/user.json' })

  test('nenhum texto fica abaixo do mínimo de contraste do AA', async ({ page }) => {
    await page.goto('/mesa/1/4')
    // A âncora é a região que só o MESTRE tem, e não o `<h2> Iniciativa` do
    // caso do jogador: a Mesa do mestre é o trilho da ALE-211, onde a fila é um
    // `nav` recolhido com um botão no lugar do título. Ancorar no que os dois
    // papéis compartilham teria escondido de novo a metade que este caso veio
    // medir.
    await expect(page.getByRole('region', { name: 'Controles do mestre' })).toBeVisible()

    await page.getByRole('button', { name: 'Configurações da sessão' }).click()
    await expect(page.getByRole('heading', { name: /^Sessão \d/ })).toBeVisible()

    const { falhas, medidos } = await medeOContraste(page)
    expect(medidos, 'o medidor não achou texto na Mesa do mestre').toBeGreaterThan(100)
    expect(falhas, 'texto abaixo do AA na Mesa do mestre').toEqual([])
  })
})

test.describe('Administração (piloto Datastar)', () => {
  // A tela é do ADMIN, então o estado de login é o do mestre — o `requireAdmin`
  // responde 403 para o jogador, e é o servidor que decide, não a tela.
  test.use({ storageState: '.auth/user.json' })

  test('nenhum texto fica abaixo do mínimo de contraste do AA', async ({ page }) => {
    await page.goto('/admin')
    await expect(page.getByRole('heading', { name: 'Administração' })).toBeVisible()

    expect(await textoComContrasteBaixo(page), 'texto abaixo do AA na administração').toEqual([])
  })

  /**
   * A confirmação do DESTRUTIVO é um modal de verdade — sem Kobalte.
   *
   * Esta é a pergunta que a segunda superfície existe para responder, e o
   * guarda afirma as quatro propriedades que a biblioteca dava e que o
   * `<dialog>` nativo devolve: ele é `:modal` (o resto da página fica inerte),
   * o foco entra nele, ele tem nome acessível, e ao fechar o foco VOLTA para o
   * gatilho.
   *
   * E2E porque nada disto existe em jsdom: não há `showModal`, não há
   * `:modal`, e o foco é uma ficção.
   */
  // Aqui morava `o endereço antigo /admin encaminha para a cena nova`, que
  // media o desvio de `/admin` para `/piloto/admin`. As cenas subiram para a
  // raiz na ALE-280 e os dois endereços viraram um só: não há desvio para medir,
  // e o que sobrava era "a cena de administração abre" — sem mecanismo que só um
  // navegador tenha, que é a única justificativa de e2e que o guia aceita.
  //
  // Quem prende que a rota existe e é do administrador é o Go
  // (`piloto_admin_test.go`), e a decisão de o endereço VELHO responder 404 está
  // em `TestTheOldPilotPrefixIsGone`.

  /**
   * O LINK DE UMA PESSOA NÃO PODE APARECER SOB O NOME DE OUTRA (ALE-242).
   *
   * O token chega por remendo do servidor num `<div>` fixo (`#reset-link`), e o
   * diálogo é UM só reaproveitado por todas as linhas. Sem limpar ao abrir,
   * gerar o link da primeira conta, fechar, e abrir a caixa da segunda mostra o
   * link da PRIMEIRA sob o nome da SEGUNDA — e quem estiver com pressa entrega
   * a chave da conta errada.
   *
   * E2E porque a garantia é de ESTADO DE DOM ATRAVESSANDO duas aberturas de um
   * `<dialog>` nativo: em jsdom não há `showModal`, e o guarda em Go só
   * consegue afirmar que a limpeza está escrita no marcador, não que ela
   * acontece.
   *
   * Ele grava uma linha em `password_resets`, e isso é aceitável: nenhuma tela
   * a lista e nenhuma asserção a conta. Cunhar CONVITE seria diferente — aquilo
   * aparece num painel e a tela não sabe revogar —, e por isso aquela garantia
   * ficou em Go.
   */
  test('o link de redefinição não vaza para a caixa do jogador seguinte', async ({ page }) => {
    await page.goto('/admin')
    const gatilhos = page.getByRole('button', { name: /^Redefinir a senha de/ })
    await expect(gatilhos.first()).toBeVisible()

    await gatilhos.first().click()
    await page.getByRole('button', { name: 'Gerar link' }).click()
    const campo = page.locator('#reset-url')
    await expect(campo).toBeVisible()
    const primeiro = await campo.inputValue()
    expect(primeiro, 'o link nasceu sem token').toContain('token=')
    // A origem é a do NAVEGADOR, e não a do servidor: com o `r.Host` o link
    // nasceria apontando para a porta da API, que o proxy do Vite reescreve.
    expect(primeiro).toContain(new URL(page.url()).origin)

    await page.getByRole('button', { name: 'Fechar' }).click()
    await gatilhos.nth(1).click()

    await expect(page.locator('dialog#redefinir')).toBeVisible()
    await expect(campo, 'o link do primeiro jogador sobreviveu na caixa do segundo').toHaveCount(0)
  })

  test('o diálogo de apagar conta é modal, nomeado, e devolve o foco', async ({ page }) => {
    await page.goto('/admin')
    const gatilho = page.getByRole('button', { name: /^Apagar a conta de/ }).first()
    await gatilho.focus()
    await gatilho.press('Enter')

    const dialogo = page.locator('#confirmar')
    await expect(dialogo).toBeVisible()

    const estado = await page.evaluate(() => {
      const d = document.getElementById('confirmar') as HTMLDialogElement
      return {
        modal: d.matches(':modal'),
        focoDentro: d.contains(document.activeElement),
        nome: document.getElementById(d.getAttribute('aria-labelledby') ?? '')?.textContent?.trim() ?? '',
      }
    })
    expect(estado.modal, 'o fundo precisa ficar inerte').toBe(true)
    expect(estado.focoDentro, 'o foco precisa entrar no diálogo').toBe(true)
    expect(estado.nome, 'o diálogo precisa de nome acessível').toContain('Apagar a conta de')

    await page.keyboard.press('Escape')
    await expect(dialogo).toBeHidden()
    await expect(gatilho).toBeFocused()
  })
})

/**
 * NÃO existe aqui um guarda de "o morph preserva o foco", e a ausência é
 * deliberada e medida.
 *
 * Eu escrevi um: o risco parecia óbvio, porque o servidor substitui o `<main>`
 * inteiro a cada mudança e um jogador digitando o d20 no meio do turno perderia
 * o cursor. Ele passou verde, e depois passou verde SABOTADO duas vezes — com o
 * patch em `mode: replace` em vez de morph, e com o servidor renderizando o
 * `value` do campo. Nas duas o foco e o texto digitado sobreviveram.
 *
 * Então o guarda não protegia nada: era um e2e — o teste mais caro do
 * repositório — sem modo de falha demonstrável. O que ficou é o ACHADO, que
 * vale mais que ele: o morph do Datastar reaproveita o nó por identidade, e
 * essa classe de defeito, que eu apostava contra, não acontece.
 */
test.describe('A porta (piloto Datastar)', () => {
  // ANÔNIMA: são estas telas que criam a sessão, e com o estado de login o
  // servidor redireciona para dentro do app — o guarda mediria a tela errada.
  test.use({ storageState: { cookies: [], origins: [] } })

  /**
   * A porta é a tela-título, e a tela-título tem duas coisas que nenhuma outra
   * superfície do piloto tinha: o brilho do `scene-title-glow` sobre a pedra, e
   * o `text-muted-foreground` do rodapé sobre o fundo mais escuro da cena.
   *
   * Nenhuma delas dá para medir fora do navegador: converter oklch para sRGB é
   * trabalho dele, e em jsdom o `getComputedStyle` devolve o oklch cru.
   */
  test('nenhum texto fica abaixo do mínimo de contraste do AA', async ({ page }) => {
    for (const [caminho, marco] of [
      ['/entrar', 'Entrar'],
      ['/criar-conta?convite=nao-importa', 'Criar conta'],
      ['/redefinir-senha', 'Escolher nova senha'],
    ] as const) {
      await page.goto(caminho)
      await expect(page.getByRole('heading', { name: marco, level: 2 })).toBeVisible()
      expect(await textoComContrasteBaixo(page), `texto abaixo do AA em ${caminho}`).toEqual([])
    }
  })

  /**
   * A senha é conferida sem sinal do Datastar: o `data-on:input` lê o campo
   * irmão pelo DOM e usa `setCustomValidity`.
   *
   * E2E porque `setCustomValidity` e `validationMessage` são do navegador —
   * jsdom aceita a chamada e não faz nada com ela, então lá o guarda passaria
   * verde com a implementação apagada.
   */
  test('a confirmação de senha avisa o typo sem pôr a senha em estado de cliente', async ({
    page,
  }) => {
    await page.goto('/criar-conta?convite=nao-importa')
    await page.locator('#senha').fill('uma senha boa')
    await page.locator('#confirmar').fill('outra coisa')

    expect(
      await page.locator('#confirmar').evaluate((el: HTMLInputElement) => el.validationMessage),
    ).toBe('As senhas não conferem')

    await page.locator('#confirmar').fill('uma senha boa')
    expect(
      await page.locator('#confirmar').evaluate((el: HTMLInputElement) => el.checkValidity()),
    ).toBe(true)
  })
})

test.describe('O Hub (piloto Datastar)', () => {
  test.use({ storageState: '.auth/user.json' })

  test('nenhum texto fica abaixo do mínimo de contraste do AA', async ({ page }) => {
    await page.goto('/')
    await expect(page.getByRole('navigation', { name: 'Menu principal' })).toBeVisible()
    expect(await textoComContrasteBaixo(page), 'texto abaixo do AA no Hub').toEqual([])
  })

  /**
   * A TESE da fatia, medida: o driver de teclado da SPA anda num menu que o
   * SERVIDOR desenhou, sem uma linha de adaptação.
   *
   * O menu só declara a forma dele (`data-nav-region`, `data-nav-layout`) e o
   * `scene-nav.ts` — compilado dos mesmos fontes que a SPA usa — lê isso do
   * DOM. Um DOM vindo do servidor é um DOM.
   *
   * E2E porque foco e geometria são do navegador: em jsdom todo elemento mede
   * zero e o driver não teria como escolher o vizinho.
   */
  test('as setas andam no menu que o servidor desenhou', async ({ page }) => {
    await page.goto('/')
    await page.getByRole('link', { name: 'Meus Heróis' }).focus()

    await page.keyboard.press('ArrowDown')
    await expect(page.locator(':focus')).toHaveAttribute('href', '/campanhas')

    await page.keyboard.press('ArrowUp')
    await expect(page.locator(':focus')).toHaveAttribute('href', '/personagens')
  })

  /**
   * O popover do rodapé é a Popover API NATIVA no lugar do Kobalte, e o que se
   * afirma é o que a biblioteca entregava: camada de topo, `Esc` fecha, e o
   * foco VOLTA para o gatilho.
   *
   * O `Esc` é o guarda que importa. O driver de teclado escuta na CAPTURA para
   * pre-emptar o foco rotativo do Kobalte, e com isso ele também pre-emptava o
   * navegador: media na ALE-231 que o popover não fechava, porque o `Esc` virava
   * "voltar um nível" antes de chegar à dispensa nativa. A lista de "há camada
   * aberta?" do driver era Kobalte-shaped e passou a conhecer `:popover-open` e
   * `dialog[open]`.
   */
  test('o menu do jogador é popover nativo: Esc fecha e devolve o foco', async ({ page }) => {
    await page.goto('/')
    const gatilho = page.getByRole('button', { name: /^Menu de / })
    await gatilho.click()

    const menu = page.locator('#menu-do-jogador')
    await expect(menu).toBeVisible()
    expect(await menu.evaluate((el) => el.matches(':popover-open'))).toBe(true)

    await page.keyboard.press('Escape')
    await expect(menu).toBeHidden()
    await expect(gatilho).toBeFocused()
  })
})

test.describe('O Hub e a SPA dividem a preferência de som', () => {
  test.use({ storageState: '.auth/user.json' })

  /**
   * Enquanto as duas portas existirem, som e volume têm de ser a MESMA
   * preferência: quem liga o som no Hub do servidor não pode achá-lo desligado
   * na cena do tabuleiro. O contrato é a chave e a forma do `localStorage`
   * (`t20-ui` → `{state:{sfx,volume}}`), e o `cena.js` escreve pelo mesmo
   * `persistUi` que o `ui-store` da SPA usa.
   *
   * É este contrato que se afirma — não o rótulo do botão. O rótulo é
   * consequência; a chave é o que uma refatoração distraída quebraria sem que
   * nada mais reclamasse.
   */
  test('ligar o som no Hub grava na chave que a SPA lê', async ({ page }) => {
    // Limpa ANTES de a página carregar, e é aí que estava um defeito de ORDEM
    // deste teste: limpando depois do `goto`, o `data-init` já tinha lido a
    // preferência antiga para o sinal, e o rótulo nascia "Som ligado" se algum
    // teste anterior tivesse ligado. O `addInitScript` roda antes de qualquer
    // script da página, que é o único momento em que limpar significa alguma
    // coisa.
    await page.addInitScript(() => localStorage.removeItem('t20-ui'))
    await page.goto('/')

    await page.getByRole('button', { name: /^Menu de / }).click()
    const alternador = page.locator('#menu-do-jogador button').first()
    await expect(alternador).toHaveText(/Som desligado/)

    await alternador.click()
    await expect(alternador).toHaveText(/Som ligado/)
    // O slider só existe com o som ligado: controle sobre o mudo é controle morto.
    await expect(page.locator('#volume')).toBeVisible()

    expect(await page.evaluate(() => localStorage.getItem('t20-ui'))).toBe(
      '{"state":{"sfx":true,"volume":100}}',
    )
  })
})

test.describe('A cena de campanhas (piloto Datastar)', () => {
  test.use({ storageState: '.auth/user.json' })

  test('nenhum texto fica abaixo do mínimo de contraste do AA', async ({ page }) => {
    await page.goto('/campanhas')
    await expect(page.getByRole('listbox', { name: 'Campanhas' })).toBeVisible()
    expect(await textoComContrasteBaixo(page), 'texto abaixo do AA nas campanhas').toEqual([])
  })

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
   * exata. Os sete casos de `busca_test.go` foram conferidos um a um rodando o
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
    const antes = await page.getByRole('option').count()
    expect(antes, 'a seed precisa de mais de duas campanhas').toBeGreaterThan(2)

    await page.getByRole('searchbox', { name: 'Buscar campanha' }).fill('tauron')
    await expect(page.getByRole('option')).toHaveCount(3)
    await expect(page.getByRole('option', { name: /A Queda de Tauron/ })).toBeVisible()

    await page.getByRole('searchbox', { name: 'Buscar campanha' }).fill('zzzzzz')
    await expect(page.getByText(/Nenhuma campanha combina/)).toBeVisible()
  })
})

test.describe('A folha em branco (piloto Datastar)', () => {
  test.use({ storageState: '.auth/user.json' })

  test('nenhum texto fica abaixo do mínimo de contraste do AA', async ({ page }) => {
    await page.goto('/campanhas/nova')
    await expect(page.getByRole('heading', { name: 'Abrir nova campanha' })).toBeVisible()
    expect(await textoComContrasteBaixo(page), 'texto abaixo do AA na folha em branco').toEqual([])
  })

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

  // O endereço antigo é favorito: ele não pode quebrar.
  test('o endereço antigo /campaigns/new encaminha para a folha nova', async ({ page }) => {
    await page.goto('/campaigns/new')
    await expect(page).toHaveURL(/\/campanhas\/nova$/)
    await expect(page.getByRole('heading', { name: 'Abrir nova campanha' })).toBeVisible()
  })
})

test.describe('A crônica (piloto Datastar)', () => {
  test.use({ storageState: '.auth/user.json' })

  test('nenhum texto fica abaixo do mínimo de contraste do AA', async ({ page }) => {
    await page.goto('/campanhas/1')
    await expect(page.getByRole('navigation', { name: 'Seções da crônica' })).toBeVisible()
    expect(await textoComContrasteBaixo(page), 'texto abaixo do AA na crônica').toEqual([])
  })

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

  // O endereço antigo é o que os jogadores têm salvo, e o `?tab=` viaja junto:
  // perdê-lo quebraria todo link de seção já compartilhado.
  test('o endereço antigo /campaigns/:id encaminha COM a seção', async ({ page }) => {
    await page.goto('/campaigns/1?tab=sessoes')
    await expect(page).toHaveURL(/\/campanhas\/1\?tab=sessoes$/)
    await expect(page.locator('[aria-current="page"]')).toHaveText('Sessões')
  })
})

test.describe('A folha de especificação (piloto Datastar)', () => {
  test.use({ storageState: '.auth/user.json' })

  /**
   * A COLUNA DUPLA MEDE OS DOIS STACKS, e este guarda existe por causa de um
   * defeito real: a primeira versão dela não passava o tamanho ao elemento
   * customizado da SPA, então aquela coluna inteira renderizava no padrão e a
   * linha "xs" comparava um xs do servidor com um default da SPA.
   *
   * O instrumento MENTIA — e instrumento que mente é pior que instrumento
   * nenhum, porque produz confiança em vez de dúvida. Um guarda que só
   * checasse "a seção existe" não teria pego; este exige que as duas colunas
   * tragam MEDIDA, e que a ladeira de tamanhos seja estritamente crescente dos
   * dois lados.
   *
   * E2E porque a coluna da SPA é montada por `solid-element` e medida com
   * `getBoundingClientRect` — nada disso existe sem navegador.
   */
  test('a coluna dupla mede os DOIS stacks, e a ladeira cresce nos dois', async ({ page }) => {
    await page.goto('/grimorio')
    const tamanhos = page.locator('#pecas [data-par]').filter({ hasText: /^(xs|sm|default|lg)/ })
    await expect(tamanhos.first()).toBeVisible()

    const alturas = await page.evaluate(() => {
      const linhas = [...document.querySelectorAll('#pecas [data-par]')]
      const daLinha = (nome: string) => {
        const linha = linhas.find((l) => l.firstElementChild?.textContent?.trim() === nome)
        return [...(linha?.querySelectorAll('[data-medir-cela]') ?? [])].map(
          (e) => Number(/h (\d+)/.exec(e.textContent ?? '')?.[1] ?? 0),
        )
      }
      return { xs: daLinha('xs'), sm: daLinha('sm'), lg: daLinha('lg') }
    })

    for (const [nome, par] of Object.entries(alturas)) {
      expect(par.length, `a linha ${nome} não tem as duas colunas medidas`).toBe(2)
      expect(par[0], `a coluna da SPA não mediu em ${nome}`).toBeGreaterThan(0)
      expect(par[1], `a coluna do servidor não mediu em ${nome}`).toBeGreaterThan(0)
    }
    // A ladeira cresce dos DOIS lados. Com o defeito antigo, a coluna da SPA
    // vinha 36/36/36 — constante, e nenhuma asserção de igualdade a pegaria.
    for (const coluna of [0, 1]) {
      expect(alturas.xs[coluna], 'xs não é menor que sm').toBeLessThan(alturas.sm[coluna] as number)
      expect(alturas.sm[coluna], 'sm não é menor que lg').toBeLessThan(alturas.lg[coluna] as number)
    }
  })

  /**
   * `noShadowDOM()` não é opcional: no shadow root as classes do Tailwind não
   * alcançam, e as peças da SPA renderizariam sem forma nenhuma — numa página
   * cujo trabalho é mostrar como elas são. Pior, as variáveis CSS atravessam o
   * shadow, então o resultado seria parcialmente certo: cores no lugar, forma
   * não.
   */
  test('as peças da SPA montam SEM shadow root, senão o Tailwind não as alcança', async ({
    page,
  }) => {
    await page.goto('/grimorio')
    const botao = page.locator('spa-botao').first()
    await expect(botao.locator('button')).toBeVisible()
    expect(await botao.evaluate((el) => !!el.shadowRoot)).toBe(false)
  })

  // O endereço antigo é o que os dois comentários do index.css mandam abrir.
  test('o endereço antigo /grimorio encaminha para a folha nova', async ({ page }) => {
    await page.goto('/grimorio')
    await expect(page).toHaveURL(/\/grimorio$/)
    await expect(page.getByRole('heading', { name: 'Cor' })).toBeVisible()
  })
})

test.describe('A carta de convite (piloto Datastar)', () => {
  test.use({ storageState: '.auth/user.json' })

  test('nenhum texto fica abaixo do mínimo de contraste do AA', async ({ page }) => {
    await page.goto('/campanhas/entrar')
    await expect(page.getByRole('heading', { name: 'Entrar na mesa' })).toBeVisible()
    expect(await textoComContrasteBaixo(page), 'texto abaixo do AA na carta').toEqual([])
  })

  test('a carta cabe nos seis formatos', async ({ page }) => {
    await page.goto('/campanhas/entrar')
    await expect(page.getByRole('group', { name: /Qual herói/ })).toBeVisible()

    await expectNoHorizontalOverflow(page, VIEWPORTS)
    for (const viewport of VIEWPORTS) {
      await page.setViewportSize({ width: viewport.width, height: viewport.height })
      await expectDentroDaJanela(page)
    }
  })

  /**
   * NENHUM herói vem escolhido, e quem recusa é o NAVEGADOR.
   *
   * Entrar numa mesa cria uma CÓPIA do herói lá dentro (ALE-33) e sair não é um
   * clique — um rádio já marcado faz quem estiver distraído sentar o herói
   * errado. A recusa nativa do `required` diz por que não enviou, o que o botão
   * desabilitado da SPA não dizia (queixa registrada na ALE-80).
   *
   * E2E porque validação de formulário nativa não existe em jsdom: o
   * `validationMessage` é do navegador, e o envio que NÃO acontece também.
   */
  test('sem escolher herói o navegador barra o envio e diz por quê', async ({ page }) => {
    await page.goto('/campanhas/entrar')
    const radios = page.locator('input[name="characterId"]')
    await expect(radios.first()).toBeAttached()
    expect(await page.locator('input[name="characterId"]:checked').count()).toBe(0)

    await page.getByLabel('Número da campanha').fill('1')
    await page.getByRole('button', { name: 'Entrar na mesa' }).click()

    await expect(page, 'o envio passou sem herói escolhido').toHaveURL(
      /\/campanhas\/entrar$/,
    )
    const aviso = await radios.first().evaluate((el: HTMLInputElement) => el.validationMessage)
    expect(aviso, 'o navegador barrou em silêncio').not.toBe('')
  })

  // Convite morto: a carta diz, e NÃO oferece o botão. Um botão que não pode
  // funcionar é uma porta pintada na parede — quem explica é a carta.
  test('convite morto vira frase, e a carta não oferece o botão', async ({ page }) => {
    await page.goto('/campanhas/entrar?token=nao-existe-mesmo')
    await expect(page.getByText(/Convite inválido ou expirado/)).toBeVisible()
    await expect(page.getByRole('button', { name: 'Entrar na mesa' })).toHaveCount(0)
  })

  // O endereço antigo é o destino do `/join/$token`, que é a URL que o mestre
  // ENVIA. Perder o token aqui quebraria todo convite já compartilhado.
  test('o endereço antigo /campaigns/join encaminha COM o token', async ({ page }) => {
    await page.goto('/campaigns/join?token=um-token-qualquer')
    await expect(page).toHaveURL(/\/campanhas\/entrar\?token=um-token-qualquer$/)
    await expect(page.getByRole('heading', { name: 'Entrar na mesa' })).toBeVisible()
  })
})

test.describe('A cena de personagens (piloto Datastar)', () => {
  test.use({ storageState: '.auth/user.json' })

  test('nenhum texto fica abaixo do mínimo de contraste do AA', async ({ page }) => {
    await page.goto('/personagens')
    await expect(page.getByRole('listbox', { name: 'Personagens' })).toBeVisible()
    expect(await textoComContrasteBaixo(page), 'texto abaixo do AA nos personagens').toEqual([])
  })

  /**
   * A tecla `D` abre o dossiê, e a dica `D` na tela diz a verdade.
   *
   * O guarda existe por causa da dica: anunciar um atalho morto ensina errado, e
   * é mais fácil escrever a dica do que ligar a tecla. Aqui os dois andam juntos
   * ou nenhum anda.
   *
   * E2E porque teclado é do navegador — e porque o painel é `position: fixed`, o
   * que já me enganou uma vez: `offsetParent` é NULL para elemento fixo, então
   * um verificador escrito com ele reporta "fechado" com o painel aberto na
   * cara. A checagem é por `display`.
   */
  test('a tecla D abre e fecha o dossiê do herói em cena', async ({ page }) => {
    await page.goto('/personagens')
    await page.getByRole('option').first().focus()

    const dossie = page.locator('aside[aria-label^="Dossiê"]').first()
    await expect(dossie).toBeHidden()

    await page.keyboard.press('d')
    await expect(dossie).toBeVisible()
    // Ele traz o que o servidor já tinha: as habilidades da raça vêm do
    // catálogo EMBUTIDO, e o navegador não baixou catálogo nenhum para isso.
    await expect(dossie.getByText(/HABILIDADES DE/i)).toBeVisible()

    await page.keyboard.press('d')
    await expect(dossie).toBeHidden()
  })

  /**
   * Trocar de herói não pede nada ao servidor — mesmo contrato da cena de
   * campanhas, e o guarda conta as requisições pelo mesmo motivo.
   *
   * Aqui ele vale ainda mais: a Defesa de CADA herói já vem computada na
   * página. Se alguém trocar isso por uma busca sob demanda, andar no elenco
   * passa a custar uma chamada da `ComputeSheetV2` por passo.
   */
  test('as setas trocam de herói sem pedir nada ao servidor', async ({ page }) => {
    await page.goto('/personagens')
    const opcoes = page.getByRole('option')
    await expect(opcoes.first()).toBeVisible()

    let pedidos = 0
    page.on('request', () => {
      pedidos++
    })

    await opcoes.first().focus()
    await page.keyboard.press('ArrowRight')

    await expect(opcoes.nth(1)).toHaveAttribute('aria-selected', 'true')
    await expect(opcoes.first()).toHaveAttribute('aria-selected', 'false')
    expect(pedidos, 'andar no elenco foi à rede — o cursor deixou de ser sinal').toBe(0)
  })

  /**
   * O ⏎ leva à FICHA, e desde a fatia 10 da ALE-272 a ficha é a do servidor: a
   * costura entre os dois stacks que a ALE-239 criou deixou de existir. O caso
   * fica porque a garantia é a mesma — a tecla que abre a ficha tem de abrir a
   * ficha —, e nenhuma outra camada a vê: o guarda em Go conhece só o HTML de
   * um lado.
   */
  test('⏎ no trilho abre a ficha do herói em cena', async ({ page }) => {
    await page.goto('/personagens')
    const primeiro = page.getByRole('option').first()
    await primeiro.focus()
    const nome = (await primeiro.getAttribute('aria-label'))?.split(' · ')[0]

    await page.keyboard.press('Enter')

    await expect(page).toHaveURL(/\/personagens\/\d+$/)
    await expect(page.getByRole('heading', { name: nome, level: 1 }).first()).toBeVisible()
  })

  /**
   * ALE-98: a vaga de criar é POSIÇÃO DE CURSOR, e a seta chega nela. O guarda
   * em Go afirma que ela declara `role=option` e escreve o cursor; que a SETA
   * de fato pare ali e que o ⏎ leve à Forja é do teclado, e teclado é do
   * navegador.
   *
   * Sem o ⏎ aqui a gramática morre na última posição: a tecla que abriu tudo
   * até então não faz nada justamente onde não há ficha para abrir.
   */
  test('a seta alcança a vaga de criar e ⏎ leva à Forja', async ({ page }) => {
    await page.goto('/personagens')
    const vaga = page.getByRole('option', { name: 'Forjar um novo herói' })
    await expect(vaga).toBeVisible()

    await page.getByRole('option').first().focus()
    // Anda até o fim do trilho: a vaga é a última posição, sempre.
    for (let i = 0; i < 30; i++) await page.keyboard.press('ArrowRight')
    await expect(vaga).toHaveAttribute('aria-selected', 'true')
    await expect(page.getByRole('heading', { name: 'Forjar um herói' })).toBeVisible()

    await page.keyboard.press('Enter')
    await expect(page).toHaveURL(/\/personagens\/nova/)
  })

  /**
   * O NOME do vizinho fica sempre à mostra. A versão original escondia a
   * legenda atrás de `group-hover`, e hover não existe no toque nem sob
   * navegação por teclado — que são os dois modos em que duas iniciais não
   * dizem quem vem a seguir.
   *
   * E2E porque a garantia é de CSS COMPUTADO: em jsdom nenhuma folha se aplica,
   * e o guarda em Go só sabe que o texto está no HTML — texto no HTML com
   * `opacity: 0` passaria nos dois.
   */
  test('os peeks mostram o nome do vizinho sem precisar de hover', async ({ page }) => {
    await page.goto('/personagens')
    await page.getByRole('option').first().focus()
    // Um passo para dentro, para haver vizinho dos DOIS lados do palco.
    await page.keyboard.press('ArrowRight')

    for (const lado of [/^Anterior:/, /^Próximo:/]) {
      const peek = page.getByRole('button', { name: lado })
      const legenda = peek.locator('span').last()
      await expect(legenda).not.toHaveText('')
      await expect(legenda).toHaveCSS('opacity', '1')
    }
  })

  /**
   * ALE-99: o retrato não escorrega nas pontas do elenco.
   *
   * No primeiro herói não há vizinho à esquerda, e a caixa vazia entra no lugar
   * dele — sem ela o retrato desliza para a esquerda ao chegar ali, e o palco
   * dança a cada passo. É LAYOUT medido, e por isso é aqui: o guarda em Go que
   * eu tinha escrito primeiro contava `div`s por classe, que é afirmar a forma
   * do DOM e não a garantia.
   */
  test('o retrato fica no mesmo lugar nas pontas do elenco', async ({ page }) => {
    await page.goto('/personagens')
    const retratoVisivel = () =>
      page.locator('a[aria-label^="Abrir ficha de"]:visible').first().boundingBox()

    // A ENTRADA DO PALCO (ALE-235) desloca o retrato por 220ms de propósito, e
    // isso não afrouxa esta garantia: ela é sobre a posição em REPOUSO — o
    // retrato não pode ASSENTAR em lugar diferente ao sair da ponta. Sem a
    // espera, a medição pega o meio de uma animação e mede um instante que
    // ninguém vê parado.
    //
    // A espera é pelas animações DESTA cena, pelo nome: `document.getAnimations()`
    // devolve também os `animate-pulse` da tela, que são INFINITOS — esperar
    // "nenhuma rodando" nunca terminaria.
    const palcoAssentado = () =>
      page.waitForFunction(() =>
        document
          .getAnimations()
          .filter((a) => /^(palcoEntra|placaSobe)/.test((a as CSSAnimation).animationName ?? ''))
          .every((a) => a.playState !== 'running'),
      )

    await page.getByRole('option').first().focus()
    await palcoAssentado()
    const naPonta = await retratoVisivel()

    await page.keyboard.press('ArrowRight')
    await expect(page.getByRole('button', { name: /^Anterior:/ })).toBeVisible()
    await palcoAssentado()
    const noMeio = await retratoVisivel()

    expect(naPonta, 'retrato não medido na ponta').not.toBeNull()
    expect(noMeio, 'retrato não medido no meio').not.toBeNull()
    expect(Math.round(noMeio!.x), 'o retrato escorregou ao sair da ponta').toBe(
      Math.round(naPonta!.x),
    )

    // E a VAGA de criar ocupa a mesma posição de um herói. Ela não tem nome
    // longo, nem vitais, nem resumo — e sem fileiras invisíveis do tamanho
    // deles a coluna centralizada puxa o retrato para cima. Medido antes do
    // conserto: 74px. Este é o passo em que o palco dançava mais.
    for (let i = 0; i < 30; i++) await page.keyboard.press('ArrowRight')
    await palcoAssentado()
    const vaga = await page
      .locator('a[aria-label="Forjar um novo herói"]:visible')
      .first()
      .boundingBox()
    expect(vaga, 'vaga de criar não medida').not.toBeNull()
    expect(Math.round(vaga!.y), 'o palco pulou na vaga de criar').toBe(Math.round(naPonta!.y))
    expect(Math.round(vaga!.x), 'a vaga de criar não está onde os heróis estão').toBe(
      Math.round(naPonta!.x),
    )
  })

  // Tela nova se valida nos seis formatos — regra da casa, e overflow é layout.
  test('personagens: sem scroll horizontal nos seis formatos', async ({ page }) => {
    await page.goto('/personagens')
    await expect(page.getByRole('listbox', { name: 'Personagens' })).toBeVisible()

    await expectNoHorizontalOverflow(page, VIEWPORTS)
  })

  // O endereço antigo é favorito e link de terceiros: ele não pode quebrar. A
  // promessa vivia numa casca da SPA (`routes/characters.index.tsx`) e desceu
  // para o Go na fatia 10 — na SPA ela morreria junto com o `git rm`.
  test('o endereço antigo /characters encaminha para a cena nova', async ({ page }) => {
    await page.goto('/characters')
    await expect(page).toHaveURL(/\/personagens$/)
    await expect(page.getByRole('listbox', { name: 'Personagens' })).toBeVisible()
  })
})

test.describe('O bestiário (piloto Datastar)', () => {
  test.use({ storageState: '.auth/user.json' })

  const BESTIARIO = '/mestre/bestiario'

  test('nenhum texto fica abaixo do mínimo de contraste do AA', async ({ page }) => {
    await page.goto(BESTIARIO)
    await expect(page.getByRole('navigation', { name: 'Ferramentas do mestre' })).toBeVisible()
    expect(await textoComContrasteBaixo(page), 'texto abaixo do AA no bestiário').toEqual([])
  })

  test('o bestiário cabe nos seis formatos', async ({ page }) => {
    await page.goto(BESTIARIO)
    await expect(page.getByRole('heading', { name: 'Bestiário' })).toBeVisible()

    await expectNoHorizontalOverflow(page, VIEWPORTS)
    for (const viewport of VIEWPORTS) {
      await page.setViewportSize({ width: viewport.width, height: viewport.height })
      await expectDentroDaJanela(page)
    }
  })

  /**
   * A METADE ESTREITA, e é o único teste desta cena que precisa mesmo de browser.
   *
   * O `.mesa-painel` só existe a partir de 50rem de CONTÊINER — consulta de
   * contêiner, não de mídia, e nem o jsdom nem uma asserção de classe sabem
   * resolver isso. Abaixo do ponto de troca a ficha tem de estar no diálogo, e
   * acima dela tem de estar no painel; a faixa em que as duas somem, ou em que
   * as duas aparecem, é o defeito que este teste existe para pegar.
   *
   * Ele mede VISIBILIDADE REAL (`toBeVisible`), que é o que resolve a cascata
   * inteira — a consulta de contêiner, o `display:none` da folha e o
   * `data-show` do Datastar decidindo juntos.
   */
  test('a ficha vive no painel quando cabe, e no diálogo quando não cabe', async ({ page }) => {
    await page.goto(BESTIARIO)
    const painel = page.getByRole('region', { name: 'Criatura escolhida' })
    const dialogo = page.getByRole('dialog')

    await page.setViewportSize({ width: 1440, height: 900 })
    await expect(painel, 'o painel sumiu numa largura que comporta duas colunas').toBeVisible()
    await expect(dialogo, 'o diálogo apareceu por cima do painel').toBeHidden()

    // No telefone o painel não cabe: a ficha só é alcançável pelo diálogo, e é
    // ele que impede a lista de virar uma lista sem detalhe nenhum.
    await page.setViewportSize({ width: 390, height: 844 })
    await expect(painel, 'o painel ficou visível onde não cabe').toBeHidden()
    await expect(dialogo, 'o diálogo abriu sozinho').toBeHidden()

    await page.getByRole('listitem').first().getByRole('link').click()
    await expect(dialogo, 'tocar na linha não abriu a ficha no telefone').toBeVisible()

    await page.keyboard.press('Escape')
    await expect(dialogo, 'o Esc não fechou a ficha').toBeHidden()
  })

  /**
   * A CORRIDA ENTRE OS DOIS PEDIDOS DE UM CLIQUE SÓ (ALE-272).
   *
   * O clique do mouse também FOCA, e por uma fatia inteira a linha saía com dois
   * pedidos: o do foco, que só pré-visualiza e não leva `abrir=1`, e o do
   * clique, que leva. Os dois remendam o `#bestiario`, que redeclara
   * `fichaAberta` a cada remendo — então quem CHEGA por último manda, e a ordem
   * de chegada não é a de saída. Na bancada o do clique chegava depois e a ficha
   * abria; no CI a ordem inverteu e o teste acima pegou a criatura escolhida com
   * a ficha fechada, duas vezes seguidas.
   *
   * O teste acima NÃO segura isto: ele passa por sorte de cronometragem, que é o
   * que o deixou verde aqui enquanto o CI ficava vermelho. Este aqui INVERTE a
   * ordem de propósito — atrasa a resposta do pedido sem `abrir` — e aí a
   * garantia deixa de depender de quem é mais rápido.
   *
   * Browser é a única testemunha possível: são dois `fetch` em voo disparados
   * pelo MESMO gesto de ponteiro, e é o `:focus-visible` do navegador que separa
   * o foco do mouse do foco da seta.
   */
  test('a ficha abre no clique mesmo se a resposta do foco chegar depois', async ({ page }) => {
    await page.goto(BESTIARIO)
    await page.setViewportSize({ width: 390, height: 844 })
    await expect(page.getByRole('listitem').first()).toBeVisible()

    // Só o pedido SEM `abrir=1` é atrasado: é o do foco, e é ele que chegaria
    // por último para redeclarar `fichaAberta: false` por cima do clique.
    await page.route('**/mestre/bestiario?*', async (route) => {
      if (new URL(route.request().url()).searchParams.has('abrir')) {
        await route.continue()
        return
      }
      await new Promise((resolve) => setTimeout(resolve, 600))
      await route.continue()
    })

    await page.getByRole('listitem').first().getByRole('link').click()

    // O ESTADO ASSENTADO, e não o primeiro quadro — e esta espera é a diferença
    // entre um teste e um teste que mente. Medido: com o defeito no lugar a
    // ficha ABRE com a resposta do clique e só fecha 600ms depois, quando a do
    // foco chega. Um `toBeVisible` cru passa dentro dessa janela, e a primeira
    // versão deste teste nasceu VERDE sobre o defeito que ela existia para
    // pegar. A janela é de 600ms porque é ESTE teste que a injeta acima, então
    // esperar mais que ela é determinístico e não um palpite de cronometragem.
    await page.waitForTimeout(1500)

    // O CLIQUE ATERRISSOU: sem isto, um seletor que um dia pare de achar a
    // linha faria o teste passar sem nunca ter clicado em nada, que é a forma
    // desta família — ausência de estímulo com cara de conserto. Com o defeito
    // no lugar esta asserção passa e a de baixo falha, e é esse par que
    // distingue "a ficha não abriu" de "o gesto não aconteceu".
    await expect(
      page.getByRole('listitem').first().getByRole('link'),
      'o clique não escolheu a criatura: o gesto não aconteceu',
    ).toHaveAttribute('aria-current', 'true')

    await expect(
      page.getByRole('dialog'),
      'a resposta do foco chegou por último e fechou a ficha que o clique abriu',
    ).toBeVisible()
  })
})

test.describe('Os catálogos (piloto Datastar)', () => {
  test.use({ storageState: '.auth/user.json' })

  const CATALOGOS = '/mestre/condicoes'

  test('nenhum texto fica abaixo do mínimo de contraste do AA', async ({ page }) => {
    await page.goto(CATALOGOS)
    await expect(page.getByRole('navigation', { name: 'Ferramentas do mestre' })).toBeVisible()
    expect(await textoComContrasteBaixo(page), 'texto abaixo do AA nos catálogos').toEqual([])
  })

  /**
   * A ALE-149 DO LADO DO SERVIDOR, e é o único guarda desta cena que precisa
   * mesmo de browser.
   *
   * A cena manda as 992 entradas de uma vez — sem virtualização, por decisão do
   * dono. O que sustenta essa decisão é a lista rolar DENTRO da caixa, e a
   * versão em React já errou exatamente isso: a lista crescia até a altura do
   * conteúdo e vazava 1854–2566px para fora do cartão sem a página rolar, então
   * a asserção "a cena não rola" ficava verde por cima do defeito.
   *
   * Aqui a asserção é a inversa e mede o que aquela não media: o DOCUMENTO não
   * pode ser mais alto que a janela. Com 566 poderes desenhados, um `min-h-0`
   * faltando em qualquer elo da corrente dá uma página de dezenas de milhares
   * de pixels — e é a aba de Poderes que precisa ser visitada, porque é a maior.
   */
  test('com o acervo inteiro na tela, a PÁGINA não cresce em nenhum formato', async ({
    page,
  }) => {
    for (const viewport of VIEWPORTS) {
      await page.setViewportSize({ width: viewport.width, height: viewport.height })
      await page.goto(`${CATALOGOS}?aba=poderes`)
      const altura = await page.evaluate(() => ({
        doc: document.documentElement.scrollHeight,
        janela: window.innerHeight,
      }))
      expect(
        altura.doc,
        `a página cresceu para ${altura.doc}px em ${viewport.width}×${viewport.height} — é a ALE-149`,
      ).toBeLessThanOrEqual(altura.janela + 2)
    }
    await expectNoHorizontalOverflow(page, VIEWPORTS)
  })

  /**
   * O TETO DE TRÊS COLUNAS é medida de leitura (ALE-170), e ele quase sumiu no
   * porte: a primeira versão usava `auto-fill minmax(22rem, 1fr)` puro e MEDIU
   * quatro colunas a 1920. Acima de três a linha fica curta demais para
   * descrição de regra; abaixo de 22rem a prosa vira fita.
   *
   * E2E porque a única testemunha é o `gridTemplateColumns` COMPUTADO — a
   * classe é a mesma em toda largura, então asserção de classe não veria nada.
   */
  test('a grade nunca passa de três colunas, e nunca some', async ({ page }) => {
    for (const [largura, altura, esperado] of [
      [1920, 1080, 3],
      [1440, 900, 3],
      [1024, 768, 2],
      [390, 844, 1],
    ] as const) {
      await page.setViewportSize({ width: largura, height: altura })
      await page.goto(`${CATALOGOS}?aba=condicoes`)
      const colunas = await page.evaluate(() => {
        const grade = document.querySelector('.acervo-em-colunas')
        if (!grade) return 0
        return getComputedStyle(grade).gridTemplateColumns.split(' ').length
      })
      expect(colunas, `${largura}px devia dar ${esperado} coluna(s)`).toBe(esperado)
    }
  })

  /**
   * A busca varre TODOS os catálogos, não só o que está aberto.
   *
   * É a ALE-22: a versão em React filtrava só a aba ativa, então "bola de fogo"
   * digitado na aba Condições dizia "nada encontrado" com a magia existindo.
   *
   * A segunda metade deste guarda — "e a fileira de abas sai de cena" — saiu
   * junto com a fileira: cada catálogo mora hoje no seu endereço e quem troca é
   * o trilho do mestre, que é permanente. Não sobrou fileira para sumir, e uma
   * asserção sobre elemento que não existe passa verde sobre nada.
   */
  test('buscar varre todos os catálogos, não só o que está aberto', async ({ page }) => {
    await page.setViewportSize({ width: 1440, height: 900 })
    await page.goto(`${CATALOGOS}?aba=condicoes`)
    await expect(page.getByRole('navigation', { name: 'Ferramentas do mestre' })).toBeVisible()

    await page.getByRole('searchbox', { name: 'Buscar nos catálogos' }).fill('fogo')

    await expect(
      page.getByRole('region', { name: 'Magias' }),
      'buscar de dentro das condições não alcançou as magias (ALE-22)',
    ).toBeVisible()
  })
})

test.describe('O construtor de encontros (piloto Datastar)', () => {
  test.use({ storageState: '.auth/user.json' })

  const ENCONTROS = '/mestre/encontros'

  test('nenhum texto fica abaixo do mínimo de contraste do AA', async ({ page }) => {
    await page.goto(`${ENCONTROS}?nivel=1&grupo=4&c=ogro:2,goblin-salteador:4`)
    await expect(page.getByRole('heading', { name: 'Construtor de encontros' })).toBeVisible()
    expect(await textoComContrasteBaixo(page), 'texto abaixo do AA nos encontros').toEqual([])
  })

  test('o construtor cabe nos seis formatos', async ({ page }) => {
    await page.goto(`${ENCONTROS}?nivel=1&grupo=4&c=ogro:2,goblin-salteador:4`)
    await expectNoHorizontalOverflow(page, VIEWPORTS)
    for (const viewport of VIEWPORTS) {
      await page.setViewportSize({ width: viewport.width, height: viewport.height })
      await expectDentroDaJanela(page)
    }
  })

  /**
   * O RASCUNHO NÃO MEXE NO HISTÓRICO, e é essa a decisão que o dono tomou.
   *
   * A alternativa — o encontro SER o endereço — foi recusada porque a
   * quantidade muda a cada clique: cada `[+]` viraria uma entrada, e o botão
   * Voltar passaria a desfazer cliques em vez de sair da tela. Este teste é o
   * que impede a decisão de se perder num refactor.
   *
   * E2E porque o histórico é do NAVEGADOR: `history.length` e `goBack` não
   * existem em jsdom, e nenhuma asserção de servidor vê a diferença.
   */
  test('montar o encontro não empilha histórico, e o Voltar sai da tela', async ({ page }) => {
    await page.goto('/')
    await page.goto(ENCONTROS)
    const antes = await page.evaluate(() => history.length)

    await page.getByRole('searchbox', { name: 'Buscar criatura para acrescentar' }).fill('ogro')
    await page.getByRole('button', { name: /^Acrescentar Ogro/ }).first().click()
    await expect(page.getByRole('button', { name: 'Mais um Ogro' })).toBeVisible()
    await page.getByRole('button', { name: 'Mais um Ogro' }).click()
    await page.getByRole('button', { name: 'Mais um Ogro' }).click()

    expect(await page.evaluate(() => history.length), 'os cliques empilharam histórico').toBe(
      antes,
    )
    expect(new URL(page.url()).search, 'o rascunho vazou para a URL').toBe('')

    await page.goBack()
    await expect(page, 'o Voltar desfez um clique em vez de sair da tela').toHaveURL(/\/$/)
  })

  /**
   * O LINK COPIADO reabre o encontro. A ida está prendida em Go
   * (`TestTheCopiedLinkReopensTheEncounter`); o que só o browser vê é o `data-url`
   * que o botão carrega chegando ao endereço certo depois de uma navegação de
   * verdade — que é o que alguém faz ao colar no chat da mesa.
   */
  test('o link do botão de copiar reabre o mesmo encontro', async ({ page }) => {
    await page.goto(`${ENCONTROS}?nivel=3&grupo=4&c=ogro:2`)
    const veredito = page.locator('#encontros')
    const antes = await veredito.textContent()

    const link = await page
      .getByRole('button', { name: 'Copiar link do encontro' })
      .getAttribute('data-url')
    expect(link, 'o botão não carrega o endereço do encontro').toBeTruthy()

    await page.goto(link as string)
    expect(await veredito.textContent(), 'o encontro voltou diferente pelo link').toBe(antes)
  })
})

test.describe('O improviso (piloto Datastar)', () => {
  test.use({ storageState: '.auth/user.json' })

  const IMPROVISO = '/mestre/improviso'

  test('nenhum texto fica abaixo do mínimo de contraste do AA', async ({ page }) => {
    await page.goto(IMPROVISO)
    await expect(page.getByRole('heading', { name: 'Improviso' })).toBeVisible()
    // Com resultado na tela: o número grande em dourado é o texto que mais
    // arrisca contraste, e ele não existe antes da primeira rolagem — medir a
    // cena vazia mediria a metade fácil.
    const ruina = page.getByRole('region', { name: 'Ermos — Ruína' })
    await ruina.getByRole('button', { name: 'Rolar d6' }).click()
    await expect(ruina.locator('[aria-live="polite"]')).toBeVisible()
    expect(await textoComContrasteBaixo(page), 'texto abaixo do AA no improviso').toEqual([])
  })

  test('o improviso cabe nos seis formatos', async ({ page }) => {
    await page.goto(IMPROVISO)
    await expectNoHorizontalOverflow(page, VIEWPORTS)
    for (const viewport of VIEWPORTS) {
      await page.setViewportSize({ width: viewport.width, height: viewport.height })
      await expectDentroDaJanela(page)
    }
  })

  /**
   * ROLAR NÃO EMPILHA HISTÓRICO, e é a mesma decisão do rascunho do encontro
   * (ALE-259): o mestre rola várias vezes seguidas, e cada rolagem virando uma
   * entrada faria o botão Voltar desfazer dados em vez de sair da tela.
   *
   * E2E porque o histórico é do NAVEGADOR — `history.length` e `goBack` não
   * existem em jsdom.
   */
  test('rolar não empilha histórico, e o Voltar sai da tela', async ({ page }) => {
    await page.goto('/')
    await page.goto(IMPROVISO)
    const antes = await page.evaluate(() => history.length)

    for (let i = 0; i < 3; i++) {
      await page.getByRole('button', { name: 'Rolar d20' }).first().click()
      await expect(page.locator('#improviso [aria-live="polite"]').first()).toBeVisible()
    }

    expect(await page.evaluate(() => history.length), 'as rolagens empilharam histórico').toBe(
      antes,
    )
    await page.goBack()
    await expect(page, 'o Voltar desfez uma rolagem em vez de sair').toHaveURL(/\/$/)
  })

  /**
   * O HISTÓRICO CURTO é o que a ferramenta tem de diferente de um botão que
   * mostra o último resultado: o mestre que rola duas vezes na mesma cena quer
   * comparar. Cinco é o fundo, e a sexta rolagem empurra a primeira para fora.
   */
  test('a tabela guarda as rolagens anteriores, e para em cinco', async ({ page }) => {
    await page.goto(IMPROVISO)
    // Pelo NOME da região, e não por contar filhos de `div`: acoplar ao formato
    // do DOM é o que o guia manda não escrever, e a primeira versão deste teste
    // fazia isso — e apontava para o cartão errado.
    const cartao = page.getByRole('region', { name: 'Ermos — Ruína' })

    for (let i = 0; i < 7; i++) {
      await cartao.getByRole('button', { name: 'Rolar d6' }).click()
      await expect(cartao.locator('[aria-live="polite"]')).toBeVisible()
    }
    // Uma manchete + quatro anteriores = os cinco que o fundo permite.
    await expect(cartao.locator('li'), 'o histórico passou do fundo de cinco').toHaveCount(4)
  })
})
