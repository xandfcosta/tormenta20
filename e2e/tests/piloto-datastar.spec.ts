import { type Page, expect, test } from '@playwright/test'

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
    await expect(page.locator(':focus')).toHaveAttribute('href', '/characters')
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
})
