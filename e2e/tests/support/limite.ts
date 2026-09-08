import { expect, type Page } from '@playwright/test'

/**
 * O LIMITE DE UM COMPONENTE contra o que está atrás dele (WCAG 1.4.11).
 *
 * A regra é 3:1, e ela não é sobre TEXTO: é sobre conseguir ver ONDE o botão
 * começa e acaba. O medidor de contraste da casa (`support/contraste.ts`) não
 * responde isso — ele compara a tinta do texto com o fundo efetivo, e um botão
 * cujo preenchimento se confunde com a cena passa por ele sem reclamar, porque
 * a letra continua legível.
 *
 * # O caso que o escreveu (ALE-250)
 *
 * O `secondary` do servidor tem uma borda que o da SPA não tinha, e a folha de
 * especificação media isso como divergência de 2px — "o errado é o nome, não a
 * borda". Medindo o limite, a conclusão virou:
 *
 *   preenchimento do `secondary` contra o fundo da cena   1,30:1   ✗
 *   a borda (`--grimorio-iron-light`)                     3,57:1   ✓
 *
 * **A borda é o conserto, não a divergência** — e o que ela conserta é um
 * defeito que a SPA tinha e ninguém tinha medido. Sem este guarda, alguém tira
 * a borda amanhã em nome da fidelidade e nada acusa: o botão continua clicável,
 * o texto continua legível, e o que se perde é a fronteira.
 *
 * Por que browser: resolver `oklch` para sRGB é coisa que só o navegador faz —
 * a mesma razão que prende o medidor de contraste (ver `contraste.ts`).
 */

/** Uma medição: o que reprovou, e QUANTOS componentes foram olhados. */
export type MedicaoDeLimite = { falhas: string[]; medidos: number }

/** O mínimo do WCAG 1.4.11 para limite de componente. */
export const LIMITE_MINIMO = 3

/**
 * Mede, para cada botão da página, o maior contraste entre o que o desenha —
 * preenchimento OU borda — e o fundo efetivo atrás dele.
 *
 * O MAIOR dos dois, e não a média: basta uma das pontas dar a fronteira. Um
 * botão de preenchimento forte não precisa de borda, e um de borda forte não
 * precisa de preenchimento — é exatamente a diferença entre o `default` e o
 * `secondary` da casa.
 */
export async function medeOLimiteDosBotoes(page: Page): Promise<MedicaoDeLimite> {
  return page.evaluate((minimo) => {
    const tela = document.createElement('canvas')
    tela.width = 1
    tela.height = 1
    const pincel = tela.getContext('2d')
    if (!pincel) return { falhas: ['sem canvas'], medidos: 0 }
    const rgb = (css: string): number[] => {
      pincel.clearRect(0, 0, 1, 1)
      pincel.fillStyle = css
      pincel.fillRect(0, 0, 1, 1)
      return [...pincel.getImageData(0, 0, 1, 1).data]
    }
    const luz = (c: number[]) => {
      const [r, g, b] = c.slice(0, 3).map((v) => {
        const x = v / 255
        return x <= 0.04045 ? x / 12.92 : ((x + 0.055) / 1.055) ** 2.4
      })
      return 0.2126 * (r ?? 0) + 0.7152 * (g ?? 0) + 0.0722 * (b ?? 0)
    }
    const razao = (a: number[], b: number[]) => {
      const [x, y] = [luz(a), luz(b)].sort((p, q) => q - p)
      return ((x ?? 0) + 0.05) / ((y ?? 0) + 0.05)
    }
    // O fundo EFETIVO: sobe a árvore até achar quem é opaco, como o medidor de
    // contraste faz — medir contra um pai transparente daria razão inventada.
    const fundoDe = (el: Element): number[] => {
      let n: Element | null = el.parentElement
      while (n && n !== document.documentElement) {
        const c = rgb(getComputedStyle(n).backgroundColor)
        if ((c[3] ?? 0) > 250) return c
        n = n.parentElement
      }
      return rgb(getComputedStyle(document.body).backgroundColor)
    }

    // OS BOTÕES QUE A CASA PINTA, e o escopo é o guarda inteiro.
    //
    // A primeira versão mediu todo `<button>` da página e reprovou sete que
    // estavam certos: a coluna-MUSEU da folha, que desenha as variantes da SPA
    // de propósito para comparar, e os botões de demonstração sem classe
    // nenhuma. Guarda que reprova o que está certo é guarda que alguém desliga.
    //
    // O critério é o PREENCHIMENTO da casa (`bg-primary`, `bg-secondary`,
    // `bg-destructive`), que é o que `ui.ButtonClasses` emite — e ele pega tanto
    // o `@ui.Button` quanto o `class={ ui.ButtonClasses(…) }` cru, que é metade
    // dos chamadores. Um `ghost` ou um `link` não entram, e não é isenção: eles
    // não têm limite POR DESENHO, e o WCAG não pede fronteira para controle que
    // é só texto.
    const daCasa = (b: Element) =>
      !b.closest('spa-botao') && /\b(bg-primary|bg-secondary|bg-destructive)\b/.test(b.className)

    const olhados: string[] = []
    const falhas: string[] = []
    for (const b of [...document.querySelectorAll('button')].filter(daCasa) as HTMLButtonElement[]) {
      const cs = getComputedStyle(b)
      if (cs.visibility === 'hidden' || cs.display === 'none') continue
      const caixa = b.getBoundingClientRect()
      if (caixa.width < 24 || caixa.height < 16) continue
      // DESABILITADO fica de fora: o WCAG isenta componente inativo, e o app o
      // desenha com 50% de opacidade DE PROPÓSITO — cobrá-lo faria o guarda
      // pedir que "indisponível" parecesse disponível.
      if (b.disabled) continue
      const fundo = fundoDe(b)
      const preenchimento = rgb(cs.backgroundColor)
      const opaco = (preenchimento[3] ?? 0) > 250
      const temBorda = Number.parseFloat(cs.borderTopWidth) > 0
      const doPreenchimento = opaco ? razao(preenchimento, fundo) : 1
      const daBorda = temBorda ? razao(rgb(cs.borderTopColor), fundo) : 1
      const melhor = Math.max(doPreenchimento, daBorda)
      const rotulo = (b.getAttribute('aria-label') || b.textContent || '').trim().slice(0, 24)
      olhados.push(rotulo)
      if (melhor < minimo) {
        falhas.push(
          `"${rotulo}" tem limite de ${melhor.toFixed(2)}:1 ` +
            `(preenchimento ${doPreenchimento.toFixed(2)}, borda ${daBorda.toFixed(2)})`,
        )
      }
    }
    return { falhas, medidos: olhados.length }
  }, LIMITE_MINIMO)
}

/**
 * A asserção completa, com o denominador embutido pela mesma razão do medidor
 * de tipografia: quem afirma "nada reprovou" afirma junto quantos foram olhados,
 * senão "verde" e "não mediu" são a mesma cor.
 */
export async function expectBotoesComLimiteVisivel(page: Page, onde: string): Promise<void> {
  const { falhas, medidos } = await medeOLimiteDosBotoes(page)
  expect(
    medidos,
    `${onde}: o medidor não achou botão nenhum — o seletor parou de casar, e a asserção seguinte não seria evidência`,
  ).toBeGreaterThan(0)
  expect(falhas, `botão sem limite visível de ${LIMITE_MINIMO}:1 contra o fundo ${onde}`).toEqual([])
}
