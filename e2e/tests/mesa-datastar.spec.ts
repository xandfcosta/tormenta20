import { expect, test } from '@playwright/test'

/**
 * A Mesa renderizada pelo SERVIDOR — o piloto Datastar (ALE-219).
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
 * `/mesa` do `vite.config.ts` que a alcança. Se o piloto for apagado, este
 * arquivo vai junto.
 */
test.use({ storageState: '.auth/player.json' })

test.describe('Mesa do jogador (piloto Datastar)', () => {
  test('nenhum texto fica abaixo do mínimo de contraste do AA', async ({ page }) => {
    await page.goto('/mesa/1/4')
    await expect(page.getByRole('heading', { name: 'Iniciativa' })).toBeVisible()

    const fracos = await page.evaluate(() => {
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
      // O fundo EFETIVO, subindo a árvore: metade do defeito original era o
      // crachá ter fundo próprio, e medir contra o painel teria dado verde.
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

      return [...document.querySelectorAll('#mesa *')]
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

    expect(fracos, 'texto abaixo do mínimo do AA na Mesa do servidor').toEqual([])
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