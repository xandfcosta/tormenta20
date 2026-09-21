import type { Page } from '@playwright/test'

/**
 * O MEDIDOR DE CONTRASTE da casa, e ele mora num arquivo PRÓPRIO de propósito:
 * instrumento que vive dentro de um chamador tem exatamente um chamador, e foi
 * assim que a ficha atravessou duas fatias sem uma única medição.
 *
 * Contraste exige converter oklch para sRGB, e só o navegador faz isso: em jsdom
 * o `getComputedStyle` devolve o oklch cru, e ler aqueles três números como RGB
 * dá razão inventada. É o que prende esta medição ao browser.
 *
 * ELE MEDE O QUE ESTÁ ESCONDIDO POR UM ANCESTRAL, e isso é para saber: o
 * descarte olha o `display`/`visibility` do PRÓPRIO nó, e um filho de um pai com
 * `display: none` tem `display: block` seu. Todo diálogo que o Datastar esconde
 * pelo `data-show` do pai já entra na conta com a caixa fechada — bom para a
 * cobertura, e péssimo para quem quiser usar `medidos` como prova de que um
 * diálogo ABRIU: o número não muda.
 */

/** Uma medição: o que reprovou, e QUANTOS textos foram olhados. */
export type MedicaoDeContraste = { falhas: string[]; medidos: number }

/**
 * Mede o contraste de todo texto visível contra o fundo EFETIVO, subindo a
 * árvore até o primeiro fundo opaco.
 *
 * Contra o fundo efetivo e não contra o painel porque foi exatamente aí que os
 * DOIS defeitos que este guarda pegou se escondiam: um crachá e um botão, os
 * dois com preenchimento próprio. Medir contra o painel teria dado verde nos
 * dois.
 *
 * O `medidos` existe para o CONTROLE de quem chama: uma lista de falhas vazia é
 * indistinguível de "o seletor não achou nada", e as duas leituras se parecem no
 * terminal. Quem afirma ausência tem de afirmar o denominador junto.
 */
export async function medeOContraste(page: Page): Promise<MedicaoDeContraste> {
  return page.evaluate(() => {

    const readerCanvas = document.createElement('canvas')
    readerCanvas.width = 1
    readerCanvas.height = 1
    const ctx = readerCanvas.getContext('2d')
    if (!ctx) return { falhas: ['sem canvas'], medidos: 0 }

    const rgb = (css: string): number[] => {
      ctx.clearRect(0, 0, 1, 1)
      ctx.fillStyle = css
      ctx.fillRect(0, 0, 1, 1)
      return [...ctx.getImageData(0, 0, 1, 1).data]
    }
    const glow = (c: number[]) => {
      const [r, g, b] = c.slice(0, 3).map((v) => {
        const x = v / 255
        return x <= 0.04045 ? x / 12.92 : ((x + 0.055) / 1.055) ** 2.4
      })
      return 0.2126 * (r ?? 0) + 0.7152 * (g ?? 0) + 0.0722 * (b ?? 0)
    }
    const bgOf = (el: Element): number[] => {
      let n: Element | null = el
      while (n && n !== document.documentElement) {
        const c = rgb(getComputedStyle(n).backgroundColor)
        if ((c[3] ?? 0) > 250) return c
        n = n.parentElement
      }
      return rgb(getComputedStyle(document.body).backgroundColor)
    }
    const ratio = (a: number[], b: number[]) => {
      const [x, y] = [glow(a), glow(b)].sort((p, q) => q - p)
      return ((x ?? 0) + 0.05) / ((y ?? 0) + 0.05)
    }

    const checked: string[] = []
    const fails = [...document.querySelectorAll('.scene-grimorio *')]
      .map((el) => {
        const text = [...el.childNodes]
          .filter((n) => n.nodeType === 3)
          .map((n) => n.textContent?.trim() ?? '')
          .join('')
        if (!text) return null
        const cs = getComputedStyle(el)
        if (cs.visibility === 'hidden' || cs.display === 'none') return null
        // Texto DECORATIVO não entra na conta, e isto não é afrouxar o guarda:
        // o WCAG isenta texto que não é exposto, e `aria-hidden` é exatamente
        // essa declaração. As iniciais gigantes em `text-white/15` sobre um
        // emblema são substituto de ARTE, com o nome escrito ao lado em texto de
        // verdade — medi-las é medir a ilustração.
        //
        // O perigo aqui é esconder defeito atrás de `aria-hidden`, e a proteção
        // é a regra que já vale: se o texto CARREGA informação, escondê-lo do
        // leitor de tela é um defeito PIOR que o de contraste — e é o guarda
        // errado que estaria reclamando.
        if (el.closest('[aria-hidden="true"]')) return null
        const px = Number.parseFloat(cs.fontSize)
        const weight = Number.parseInt(cs.fontWeight, 10) || 400
        // A regra do AA: texto grande (24px, ou 18.66px em negrito) pede 3:1.
        const min = px >= 24 || (px >= 18.66 && weight >= 700) ? 3 : 4.5
        checked.push(text)
        const r = ratio(rgb(cs.color), bgOf(el))
        return r < min ? `"${text.slice(0, 24)}" dá ${r.toFixed(2)}:1 (pede ${min})` : null
      })
      .filter((x): x is string => x !== null)
    return { falhas: fails, medidos: checked.length }
  })
}

/** A lista de reprovados, para quem já garante o denominador de outro jeito. */
export async function textoComContrasteBaixo(page: Page): Promise<string[]> {
  return (await medeOContraste(page)).falhas
}
