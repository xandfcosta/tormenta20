import { expect, type Page } from '@playwright/test'

/**
 * O LIMITE DE UM COMPONENTE contra o que está atrás dele (WCAG 1.4.11).
 *
 * A regra é 3:1, e ela não é sobre TEXTO: é sobre conseguir ver ONDE o botão
 * começa e acaba. O medidor de contraste da casa (`support/contrast.ts`) não
 * responde isso — ele compara a tinta do texto com o fundo efetivo, e um botão
 * cujo preenchimento se confunde com a cena passa por ele sem reclamar, porque
 * a letra continua legível.
 *
 * Medido no `secondary`: o preenchimento contra o fundo da cena dá 1,30:1, e a
 * borda (`--grimorio-iron-light`) dá 3,57:1. **A borda é o conserto** — sem este
 * guarda, tirá-la não acusa nada: o botão continua clicável, o texto continua
 * legível, e o que se perde é a fronteira.
 *
 * Por que browser: resolver `oklch` para sRGB é coisa que só o navegador faz —
 * a mesma razão que prende o medidor de contraste (ver `contrast.ts`).
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
  return page.evaluate((minimum) => {
    const canvasEl = document.createElement('canvas')
    canvasEl.width = 1
    canvasEl.height = 1
    const brush = canvasEl.getContext('2d')
    if (!brush) return { falhas: ['sem canvas'], medidos: 0 }
    const rgb = (css: string): number[] => {
      brush.clearRect(0, 0, 1, 1)
      brush.fillStyle = css
      brush.fillRect(0, 0, 1, 1)
      return [...brush.getImageData(0, 0, 1, 1).data]
    }
    const light = (c: number[]) => {
      const [r, g, b] = c.slice(0, 3).map((v) => {
        const x = v / 255
        return x <= 0.04045 ? x / 12.92 : ((x + 0.055) / 1.055) ** 2.4
      })
      return 0.2126 * (r ?? 0) + 0.7152 * (g ?? 0) + 0.0722 * (b ?? 0)
    }
    const reason = (a: number[], b: number[]) => {
      const [x, y] = [light(a), light(b)].sort((p, q) => q - p)
      return ((x ?? 0) + 0.05) / ((y ?? 0) + 0.05)
    }
    // O fundo EFETIVO: sobe a árvore até achar quem é opaco, como o medidor de
    // contraste faz — medir contra um pai transparente daria razão inventada.
    const backgroundOf = (el: Element): number[] => {
      let n: Element | null = el.parentElement
      while (n && n !== document.documentElement) {
        const c = rgb(getComputedStyle(n).backgroundColor)
        if ((c[3] ?? 0) > 250) return c
        n = n.parentElement
      }
      return rgb(getComputedStyle(document.body).backgroundColor)
    }

    // OS BOTÕES QUE A CASA PINTA, e o escopo é o guarda inteiro: medir todo
    // `<button>` da página reprova botão de demonstração sem classe nenhuma, e
    // guarda que reprova o que está certo é guarda que alguém desliga.
    //
    // O critério é o PREENCHIMENTO da casa (`bg-primary`, `bg-secondary`,
    // `bg-destructive`), que é o que `ui.ButtonClasses` emite — e ele pega tanto
    // o `@ui.Button` quanto o `class={ ui.ButtonClasses(…) }` cru, que é metade
    // dos chamadores. Um `ghost` ou um `link` não entram, e não é isenção: eles
    // não têm limite POR DESENHO, e o WCAG não pede fronteira para controle que
    // é só texto.
    const fromSquare = (b: Element) =>
      /\b(bg-primary|bg-secondary|bg-destructive)\b/.test(b.className)

    const looked: string[] = []
    const failures: string[] = []
    for (const b of [...document.querySelectorAll('button')].filter(fromSquare) as HTMLButtonElement[]) {
      const cs = getComputedStyle(b)
      if (cs.visibility === 'hidden' || cs.display === 'none') continue
      const box = b.getBoundingClientRect()
      if (box.width < 24 || box.height < 16) continue
      // DESABILITADO fica de fora: o WCAG isenta componente inativo, e o app o
      // desenha com 50% de opacidade DE PROPÓSITO — cobrá-lo faria o guarda
      // pedir que "indisponível" parecesse disponível.
      if (b.disabled) continue
      const bg = backgroundOf(b)
      const fill = rgb(cs.backgroundColor)
      const opaque = (fill[3] ?? 0) > 250
      const hasBorder = Number.parseFloat(cs.borderTopWidth) > 0
      const fillOf = opaque ? reason(fill, bg) : 1
      const fromEdge = hasBorder ? reason(rgb(cs.borderTopColor), bg) : 1
      const best = Math.max(fillOf, fromEdge)
      const caption = (b.getAttribute('aria-label') || b.textContent || '').trim().slice(0, 24)
      looked.push(caption)
      if (best < minimum) {
        failures.push(
          `"${caption}" tem limite de ${best.toFixed(2)}:1 ` +
            `(preenchimento ${fillOf.toFixed(2)}, borda ${fromEdge.toFixed(2)})`,
        )
      }
    }
    return { falhas: failures, medidos: looked.length }
  }, LIMITE_MINIMO)
}

/**
 * A asserção completa, com o denominador embutido pela mesma razão do medidor
 * de tipografia: quem afirma "nada reprovou" afirma junto quantos foram olhados,
 * senão "verde" e "não mediu" são a mesma cor.
 */
export async function expectBotoesComLimiteVisivel(page: Page, where: string): Promise<void> {
  const { falhas: failures, medidos: measured } = await medeOLimiteDosBotoes(page)
  expect(
    measured,
    `${where}: o medidor não achou botão nenhum — o seletor parou de casar, e a asserção seguinte não seria evidência`,
  ).toBeGreaterThan(0)
  expect(failures, `botão sem limite visível de ${LIMITE_MINIMO}:1 contra o fundo ${where}`).toEqual([])
}
