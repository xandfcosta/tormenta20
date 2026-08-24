import { type Page, expect, test } from '@playwright/test'
import { VIEWPORTS, expectNoHorizontalOverflow } from './support/viewports'
import { expectDentroDaJanela } from './support/geometry'

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
 * `/piloto` do `vite.config.ts` que a alcança. Se o piloto for apagado, este
 * arquivo vai junto.
 */
/**
 * Mede o contraste de todo texto visível contra o fundo EFETIVO, subindo a
 * árvore até o primeiro fundo opaco.
 *
 * Contra o fundo efetivo e não contra o painel porque foi exatamente aí que os
 * DOIS defeitos que este arquivo pegou se escondiam: um crachá e um botão, os
 * dois com preenchimento próprio. Medir contra o painel teria dado verde nos
 * dois.
 */
async function textoComContrasteBaixo(page: Page): Promise<string[]> {
  return page.evaluate(() => {
    const tela = document.createElement('canvas')
    tela.width = 1
    tela.height = 1
    const ctx = tela.getContext('2d')
    if (!ctx) return ['sem canvas']

    const rgb = (css: string): number[] => {
      ctx.clearRect(0, 0, 1, 1)
      ctx.fillStyle = css
      ctx.fillRect(0, 0, 1, 1)
      return [...ctx.getImageData(0, 0, 1, 1).data]
    }
    const luz = (c: number[]) => {
      const [r, g, b] = c.slice(0, 3).map((v) => {
        const x = v / 255
        return x <= 0.04045 ? x / 12.92 : ((x + 0.055) / 1.055) ** 2.4
      })
      return 0.2126 * (r ?? 0) + 0.7152 * (g ?? 0) + 0.0722 * (b ?? 0)
    }
    const fundoDe = (el: Element): number[] => {
      let n: Element | null = el
      while (n && n !== document.documentElement) {
        const c = rgb(getComputedStyle(n).backgroundColor)
        if ((c[3] ?? 0) > 250) return c
        n = n.parentElement
      }
      return rgb(getComputedStyle(document.body).backgroundColor)
    }
    const razao = (a: number[], b: number[]) => {
      const [x, y] = [luz(a), luz(b)].sort((p, q) => q - p)
      return ((x ?? 0) + 0.05) / ((y ?? 0) + 0.05)
    }

    return [...document.querySelectorAll('.scene-grimorio *')]
      .map((el) => {
        const texto = [...el.childNodes]
          .filter((n) => n.nodeType === 3)
          .map((n) => n.textContent?.trim() ?? '')
          .join('')
        if (!texto) return null
        const cs = getComputedStyle(el)
        if (cs.visibility === 'hidden' || cs.display === 'none') return null
        // Texto DECORATIVO não entra na conta, e isto não é afrouxar o guarda:
        // o WCAG isenta texto que não é exposto, e `aria-hidden` é exatamente
        // essa declaração. O caso que trouxe a regra foi o monograma do livro
        // de campanhas — as iniciais gigantes em `text-white/15` sobre o emblema
        // são um substituto de ARTE, com o nome da campanha escrito ao lado em
        // texto de verdade. Medi-las é medir a ilustração.
        //
        // O perigo aqui é esconder defeito atrás de `aria-hidden`, e a proteção
        // é a regra que já vale: se o texto CARREGA informação, escondê-lo do
        // leitor de tela é um defeito PIOR que o de contraste — e é o guarda
        // errado que estaria reclamando.
        if (el.closest('[aria-hidden="true"]')) return null
        const px = Number.parseFloat(cs.fontSize)
        const peso = Number.parseInt(cs.fontWeight, 10) || 400
        // A regra do AA: texto grande (24px, ou 18.66px em negrito) pede 3:1.
        const minimo = px >= 24 || (px >= 18.66 && peso >= 700) ? 3 : 4.5
        const r = razao(rgb(cs.color), fundoDe(el))
        return r < minimo ? `"${texto.slice(0, 24)}" dá ${r.toFixed(2)}:1 (pede ${minimo})` : null
      })
      .filter((x): x is string => x !== null)
  })
}

test.use({ storageState: '.auth/player.json' })

test.describe('Mesa do jogador (piloto Datastar)', () => {
  test.use({ storageState: '.auth/player.json' })

  test('nenhum texto fica abaixo do mínimo de contraste do AA', async ({ page }) => {
    await page.goto('/piloto/mesa/1/4')
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

test.describe('Administração (piloto Datastar)', () => {
  // A tela é do ADMIN, então o estado de login é o do mestre — o `requireAdmin`
  // responde 403 para o jogador, e é o servidor que decide, não a tela.
  test.use({ storageState: '.auth/user.json' })

  test('nenhum texto fica abaixo do mínimo de contraste do AA', async ({ page }) => {
    await page.goto('/piloto/admin')
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
  // O endereço antigo é favorito de quem administra: ele não pode quebrar. Sem
  // `requireAdmin` no cliente de propósito — a rota do servidor tem o mesmo
  // guarda, e essa é a fronteira.
  test('o endereço antigo /admin encaminha para a cena nova', async ({ page }) => {
    await page.goto('/admin')
    await expect(page).toHaveURL(/\/piloto\/admin$/)
    await expect(page.getByRole('heading', { name: 'Administração' })).toBeVisible()
  })

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
    await page.goto('/piloto/admin')
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
    await page.goto('/piloto/admin')
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
      ['/piloto/entrar', 'Entrar'],
      ['/piloto/criar-conta?convite=nao-importa', 'Criar conta'],
      ['/piloto/redefinir-senha', 'Escolher nova senha'],
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
    await page.goto('/piloto/criar-conta?convite=nao-importa')
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
    await page.goto('/piloto/')
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
    await page.goto('/piloto/')
    await page.getByRole('link', { name: 'Meus Heróis' }).focus()

    await page.keyboard.press('ArrowDown')
    await expect(page.locator(':focus')).toHaveAttribute('href', '/piloto/campanhas')

    await page.keyboard.press('ArrowUp')
    await expect(page.locator(':focus')).toHaveAttribute('href', '/piloto/personagens')
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
    await page.goto('/piloto/')
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
    await page.goto('/piloto/')

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
    await page.goto('/piloto/campanhas')
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
    await page.goto('/piloto/campanhas')
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
    await page.keyboard.press('ArrowDown')

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
    await page.goto('/piloto/campanhas')
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
    await page.goto('/piloto/campanhas/nova')
    await expect(page.getByRole('heading', { name: 'Abrir nova campanha' })).toBeVisible()
    expect(await textoComContrasteBaixo(page), 'texto abaixo do AA na folha em branco').toEqual([])
  })

  // Tela nova se valida nos seis formatos. Aqui importa mais que de costume: a
  // folha hospeda campos de texto, e o espaçamento dela encolhe com a
  // ORIENTAÇÃO justamente porque num telefone deitado o botão de enviar caía
  // para fora da tela (ALE-176).
  test('a folha cabe nos seis formatos', async ({ page }) => {
    await page.goto('/piloto/campanhas/nova')
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
    await page.goto('/piloto/campanhas/nova')
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
    await expect(page).toHaveURL(/\/piloto\/campanhas\/nova$/)
    await expect(page.getByRole('heading', { name: 'Abrir nova campanha' })).toBeVisible()
  })
})

test.describe('A crônica (piloto Datastar)', () => {
  test.use({ storageState: '.auth/user.json' })

  test('nenhum texto fica abaixo do mínimo de contraste do AA', async ({ page }) => {
    await page.goto('/piloto/campanhas/1')
    await expect(page.getByRole('navigation', { name: 'Seções da crônica' })).toBeVisible()
    expect(await textoComContrasteBaixo(page), 'texto abaixo do AA na crônica').toEqual([])
  })

  test('a crônica cabe nos seis formatos', async ({ page }) => {
    await page.goto('/piloto/campanhas/1')
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
    await page.goto('/piloto/campanhas/1?tab=membros')
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
    await page.goto('/piloto/campanhas/1?tab=config')
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
    await expect(page).toHaveURL(/\/piloto\/campanhas\/1\?tab=sessoes$/)
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
    await page.goto('/piloto/grimorio')
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
    await page.goto('/piloto/grimorio')
    const botao = page.locator('spa-botao').first()
    await expect(botao.locator('button')).toBeVisible()
    expect(await botao.evaluate((el) => !!el.shadowRoot)).toBe(false)
  })

  // O endereço antigo é o que os dois comentários do index.css mandam abrir.
  test('o endereço antigo /grimorio encaminha para a folha nova', async ({ page }) => {
    await page.goto('/grimorio')
    await expect(page).toHaveURL(/\/piloto\/grimorio$/)
    await expect(page.getByRole('heading', { name: 'Cor' })).toBeVisible()
  })
})

test.describe('A carta de convite (piloto Datastar)', () => {
  test.use({ storageState: '.auth/user.json' })

  test('nenhum texto fica abaixo do mínimo de contraste do AA', async ({ page }) => {
    await page.goto('/piloto/campanhas/entrar')
    await expect(page.getByRole('heading', { name: 'Entrar na mesa' })).toBeVisible()
    expect(await textoComContrasteBaixo(page), 'texto abaixo do AA na carta').toEqual([])
  })

  test('a carta cabe nos seis formatos', async ({ page }) => {
    await page.goto('/piloto/campanhas/entrar')
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
    await page.goto('/piloto/campanhas/entrar')
    const radios = page.locator('input[name="characterId"]')
    await expect(radios.first()).toBeAttached()
    expect(await page.locator('input[name="characterId"]:checked').count()).toBe(0)

    await page.getByLabel('Número da campanha').fill('1')
    await page.getByRole('button', { name: 'Entrar na mesa' }).click()

    await expect(page, 'o envio passou sem herói escolhido').toHaveURL(
      /\/piloto\/campanhas\/entrar$/,
    )
    const aviso = await radios.first().evaluate((el: HTMLInputElement) => el.validationMessage)
    expect(aviso, 'o navegador barrou em silêncio').not.toBe('')
  })

  // Convite morto: a carta diz, e NÃO oferece o botão. Um botão que não pode
  // funcionar é uma porta pintada na parede — quem explica é a carta.
  test('convite morto vira frase, e a carta não oferece o botão', async ({ page }) => {
    await page.goto('/piloto/campanhas/entrar?token=nao-existe-mesmo')
    await expect(page.getByText(/Convite inválido ou expirado/)).toBeVisible()
    await expect(page.getByRole('button', { name: 'Entrar na mesa' })).toHaveCount(0)
  })

  // O endereço antigo é o destino do `/join/$token`, que é a URL que o mestre
  // ENVIA. Perder o token aqui quebraria todo convite já compartilhado.
  test('o endereço antigo /campaigns/join encaminha COM o token', async ({ page }) => {
    await page.goto('/campaigns/join?token=um-token-qualquer')
    await expect(page).toHaveURL(/\/piloto\/campanhas\/entrar\?token=um-token-qualquer$/)
    await expect(page.getByRole('heading', { name: 'Entrar na mesa' })).toBeVisible()
  })
})

test.describe('A cena de personagens (piloto Datastar)', () => {
  test.use({ storageState: '.auth/user.json' })

  test('nenhum texto fica abaixo do mínimo de contraste do AA', async ({ page }) => {
    await page.goto('/piloto/personagens')
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
    await page.goto('/piloto/personagens')
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
    await page.goto('/piloto/personagens')
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
   * O ⏎ atravessa a FRONTEIRA entre os dois stacks: a cena é do servidor, a
   * ficha ainda é da SPA. É a costura que a virada da ALE-239 criou, e nenhuma
   * outra camada a vê — o guarda em Go conhece só o HTML de um lado.
   */
  test('⏎ no trilho abre a ficha do herói em cena, que ainda é da SPA', async ({ page }) => {
    await page.goto('/piloto/personagens')
    const primeiro = page.getByRole('option').first()
    await primeiro.focus()
    const nome = (await primeiro.getAttribute('aria-label'))?.split(' · ')[0]

    await page.keyboard.press('Enter')

    await expect(page).toHaveURL(/\/characters\/\d+$/)
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
    await page.goto('/piloto/personagens')
    const vaga = page.getByRole('option', { name: 'Forjar um novo herói' })
    await expect(vaga).toBeVisible()

    await page.getByRole('option').first().focus()
    // Anda até o fim do trilho: a vaga é a última posição, sempre.
    for (let i = 0; i < 30; i++) await page.keyboard.press('ArrowRight')
    await expect(vaga).toHaveAttribute('aria-selected', 'true')
    await expect(page.getByRole('heading', { name: 'Forjar um herói' })).toBeVisible()

    await page.keyboard.press('Enter')
    await expect(page).toHaveURL(/\/characters\/new/)
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
    await page.goto('/piloto/personagens')
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
    await page.goto('/piloto/personagens')
    const retratoVisivel = () =>
      page.locator('a[aria-label^="Abrir ficha de"]:visible').first().boundingBox()

    await page.getByRole('option').first().focus()
    const naPonta = await retratoVisivel()

    await page.keyboard.press('ArrowRight')
    await expect(page.getByRole('button', { name: /^Anterior:/ })).toBeVisible()
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
    await page.goto('/piloto/personagens')
    await expect(page.getByRole('listbox', { name: 'Personagens' })).toBeVisible()

    await expectNoHorizontalOverflow(page, VIEWPORTS)
  })

  // O endereço antigo é favorito e link de terceiros: ele não pode quebrar. É a
  // promessa escrita em `routes/characters.index.tsx`, e ela vive na SPA.
  test('o endereço antigo /characters encaminha para a cena nova', async ({ page }) => {
    await page.goto('/characters')
    await expect(page).toHaveURL(/\/piloto\/personagens$/)
    await expect(page.getByRole('listbox', { name: 'Personagens' })).toBeVisible()
  })
})

test.describe('O bestiário (piloto Datastar)', () => {
  test.use({ storageState: '.auth/user.json' })

  const BESTIARIO = '/piloto/mestre/bestiario'

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
})

test.describe('Os catálogos (piloto Datastar)', () => {
  test.use({ storageState: '.auth/user.json' })

  const CATALOGOS = '/piloto/mestre/catalogos'

  test('nenhum texto fica abaixo do mínimo de contraste do AA', async ({ page }) => {
    await page.goto(CATALOGOS)
    await expect(page.getByRole('navigation', { name: 'Catálogos' })).toBeVisible()
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
   * A busca varre os QUATRO catálogos, e a aba some enquanto ela dura.
   *
   * É a ALE-22: a versão em React filtrava só a aba ativa, então "bola de fogo"
   * digitado na aba Condições dizia "nada encontrado" com a magia existindo.
   */
  test('buscar varre tudo e a fileira de abas sai de cena', async ({ page }) => {
    await page.setViewportSize({ width: 1440, height: 900 })
    await page.goto(`${CATALOGOS}?aba=condicoes`)
    await expect(page.getByRole('navigation', { name: 'Catálogos' })).toBeVisible()

    await page.getByRole('searchbox', { name: 'Buscar nos catálogos' }).fill('fogo')

    await expect(page.getByRole('region', { name: 'Magias' })).toBeVisible()
    await expect(
      page.getByRole('navigation', { name: 'Catálogos' }),
      'a fileira de abas ficou acesa durante a busca',
    ).toBeHidden()
  })
})
