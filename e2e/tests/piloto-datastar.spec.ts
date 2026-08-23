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
    await expect(page.getByRole('heading', { name: 'Iniciativa' })).toBeVisible()

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
