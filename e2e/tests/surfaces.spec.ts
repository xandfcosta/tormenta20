import { expect, type Page, test } from '@playwright/test'
import { expectOneFocusRing } from './support/focus'

/**
 * AS SUPERFÍCIES DA CASA — onde ela ESCREVE com tinta semântica —, e as duas
 * medições que percorrem todas elas.
 *
 * **Cena nova onde a casa escreva tinta semântica entra nesta lista, no MESMO
 * commit que a cria**: o regime aqui é ENUMERAÇÃO, e a que alguém esquecer nasce
 * sem medição, em silêncio. Um guarda só mede o que ele VISITA.
 */
const HOUSE_SURFACES = [
  {
    where: 'na folha do grimório',
    visit: async (page: Page) => {
      await page.goto('/grimorio')
      await expect(page.getByRole('heading', { name: 'Grimório' })).toBeVisible()
    },
  },
  {
    where: 'no menu do jogador',
    visit: async (page: Page) => {
      // O PAINEL ELEVADO, que é o `--popover`. Ele só existe depois do clique,
      // e é por isso que nenhum guarda o tinha medido.
      await page.goto('/')
      await page.getByRole('button', { name: 'Menu de Mestre' }).click()
      await expect(page.getByRole('button', { name: 'Sair' })).toBeVisible()
    },
  },
  {
    where: 'no veredito do encontro',
    visit: async (page: Page) => {
      // O veredito do encontro escreve com a família VITAL, e é a única tela
      // fora da folha onde ela vira texto grande.
      //
      // Ele mede o tom que estiver na tela — com o rascunho vazio, "Trivial" — e
      // NÃO exercita o "Mortal" de propósito: montar um encontro mortal custaria
      // semear criaturas, e o token daquele tom já é medido nas outras cenas por
      // identidade.
      await page.goto('/mestre/encontros')
      await expect(page.getByRole('heading', { name: 'Encontros' })).toBeVisible()
    },
  },
  {
    where: 'no livro de campanhas',
    visit: async (page: Page) => {
      // O PERGAMINHO, que é creme e inverte a conta inteira. Ancorado no
      // cabeçalho e NÃO em "Sessão ao vivo": exigir uma sessão viva seria uma
      // asserção que mede o banco.
      await page.goto('/campanhas')
      await expect(page.getByRole('heading', { name: 'Campanhas' })).toBeVisible()
    },
  },
  {
    where: 'na aba de lugares da crônica',
    visit: async (page: Page) => {
      // O ACERVO DA CAMPANHA, sobre o PERGAMINHO da crônica — creme, que
      // inverte a conta inteira como a lista de campanhas acima.
      await page.goto('/campanhas/4?tab=lugares')
      // Ancorado no GESTO e não num título: as seções da crônica são abertas por
      // sobrancelha (`SectionLabel`, um `<p>`) e não por `<h2>`, então exigir um
      // heading aqui seria exigir uma forma que a cena não tem.
      await expect(page.getByRole('button', { name: 'Novo lugar' })).toBeVisible()
      await expect(page.getByText('Cripta de Thwor')).toBeVisible()
    },
  },
  {
    where: 'na janela das notas',
    visit: async (page: Page) => {
      // A CENA DAS NOTAS FORA DA MESA: mesmo painel, sem mapa em volta.
      //
      // Endereço direto e não navegado: o gesto que a abre é um `window.open`,
      // e uma janela nova não é a `page` deste guarda.
      await page.goto('/mesa/1/4/notas')
      await expect(page.getByRole('heading', { name: /Notas/ })).toBeVisible()
    },
  },
  {
    where: 'no rascunho de lugar',
    visit: async (page: Page) => {
      // O RASCUNHO é a superfície do TABULEIRO fora da sessão, e escreve tinta
      // que nenhuma outra entrada desta lista alcança: a tarja dourada do modo,
      // o nome da peça sobre o chão de cripta, e o terreno difícil por baixo.
      //
      // A cena vem da SEMENTE e não de um clique de criação: criar o lugar aqui
      // deixaria uma linha para trás no banco compartilhado da corrida.
      //
      // MAS O CAMINHO É NAVEGADO, e não um id no endereço: id de AUTOINCREMENT
      // numa fixture compartilhada quebra por uma mudança sem relação nenhuma, e
      // o erro ("heading não encontrado") não aponta para a causa.
      await page.goto('/campanhas/4?tab=lugares')
      await page.getByRole('link', { name: 'Montar' }).first().click()
      // O NÍVEL importa: o nome do lugar aparece DUAS vezes na tela — no `<h1>`
      // da moldura e no `<h2>` da cena, que é o mesmo desenho da Mesa. Sem o
      // nível o seletor casa com os dois e estoura em `strict mode`.
      await expect(page.getByRole('heading', { level: 1, name: 'Cripta de Thwor' })).toBeVisible()
      // E o CONTROLE do que esta entrada existe para medir: a tarja do modo, que
      // é a tinta que nenhuma outra cena desta lista alcança.
      await expect(page.getByText('a mesa não vê')).toBeVisible()
    },
  },
] as const

/**
 * A razão de cada texto crimson VISÍVEL contra o fundo EFETIVO — subindo a
 * árvore até o primeiro fundo opaco, que é exatamente onde os dois defeitos se
 * escondiam: o elemento do texto é transparente e o painel de trás não é o que
 * se imagina.
 *
 * Por que e2e: converter oklch para sRGB é trabalho do navegador; em jsdom o
 * `getComputedStyle` devolve o oklch cru e ler aqueles três números como RGB
 * dá razão inventada.
 */
async function weakTints(page: Page): Promise<string[]> {
  return page.evaluate(() => {
    const canvas = document.createElement('canvas')
    canvas.width = 1
    canvas.height = 1
    const ctx = canvas.getContext('2d')
    if (!ctx) return ['sem canvas']

    const rgb = (css: string): [number, number, number, number] => {
      ctx.clearRect(0, 0, 1, 1)
      ctx.fillStyle = css
      ctx.fillRect(0, 0, 1, 1)
      const [r, g, b, a] = ctx.getImageData(0, 0, 1, 1).data
      return [r ?? 0, g ?? 0, b ?? 0, a ?? 0]
    }
    const luminance = (c: [number, number, number, number]) => {
      const [r, g, b] = [c[0], c[1], c[2]].map((v) => {
        const x = v / 255
        return x <= 0.04045 ? x / 12.92 : ((x + 0.055) / 1.055) ** 2.4
      })
      return 0.2126 * (r ?? 0) + 0.7152 * (g ?? 0) + 0.0722 * (b ?? 0)
    }

    /** As tintas da casa, resolvidas em sRGB pelo navegador, e não "qualquer
     *  coisa avermelhada": casar por aparência não sabe dizer QUAL token falhou.
     *
     *  Os três crimsons são tinta por construção. Os quatro VITAIS foram
     *  escolhidos como cor de BARRA e escrevem em alguns lugares por hábito —
     *  entram para que o dia em que alguém escrever com um deles numa tela nova
     *  seja descoberto aqui. */
    const root = getComputedStyle(document.documentElement)
    const TINT_TOKENS = [
      '--grimorio-crimson',
      '--grimorio-crimson-bright',
      '--grimorio-parchment-crimson',
      '--hp-full',
      '--hp-hurt',
      '--hp-critical',
      '--mp-arcane',
    ]
    const tints = TINT_TOKENS.map((token) => ({
      token,
      key: rgb(root.getPropertyValue(token).trim()).slice(0, 3).join(','),
    }))

    const effectiveBackground = (el: HTMLElement) => {
      let node: HTMLElement | null = el
      while (node) {
        const c = rgb(getComputedStyle(node).backgroundColor)
        if (c[3] >= 250) return c
        node = node.parentElement
      }
      return rgb('#000')
    }

    return [...document.querySelectorAll<HTMLElement>('*')]
      .filter((el) => {
        // Só quem PINTA texto próprio: um contêiner herda a cor e mediria a
        // mesma tinta dez vezes, com o mesmo veredito.
        const ownText = [...el.childNodes].some(
          (n) => n.nodeType === Node.TEXT_NODE && n.textContent?.trim(),
        )
        return ownText && el.getBoundingClientRect().height > 0
      })
      .map((el) => {
        const color = rgb(getComputedStyle(el).color)
        const hit = tints.find((t) => t.key === [color[0], color[1], color[2]].join(','))
        if (!hit) return null
        const [a, b] = [luminance(color), luminance(effectiveBackground(el))].sort((x, y) => y - x)
        const ratio = ((a ?? 0) + 0.05) / ((b ?? 0) + 0.05)
        return {
          token: hit.token,
          text: el.textContent?.trim().slice(0, 24) ?? '',
          ratio: Number(ratio.toFixed(2)),
        }
      })
      .filter((t) => t !== null && t.ratio < 4.5)
      .map((t) => `${t?.token} em "${t?.text}" dá ${t?.ratio}:1`)
  })
}

test.describe('As superfícies da casa', () => {
  /**
   * O CRIMSON É LEGÍVEL EM TODA SUPERFÍCIE ONDE ELE POUSA.
   *
   * UM TESTE POR SUPERFÍCIE, e isso não é estilo: visitar `/grimorio` e depois
   * navegar mais duas vezes no MESMO contexto derruba a terceira página em
   * branco, com `net::ERR_INSUFFICIENT_RESOURCES` no console — o Chromium
   * estoura o limite de recursos por página. Um `expect` cansado dentro de um
   * teste desses acusaria o app. Cada teste ganha página nova.
   */
  for (const surface of HOUSE_SURFACES) {
    test(`a tinta da casa alcança texto ${surface.where}`, async ({ page }) => {
      await surface.visit(page)
      const weak = await weakTints(page)
      expect(weak, `tinta abaixo do mínimo de texto ${surface.where}`).toEqual([])
    })
  }

  /**
   * Todo foco da casa tem a MESMA cara, em TODA superfície.
   *
   * O medidor mora em `support/focus.ts` e não aqui: instrumento dentro de um
   * chamador tem exatamente um chamador. O que fica aqui é a lista de
   * superfícies, a mesma do guarda de tinta — cena nova entra numa linha e ganha
   * as duas medições.
   *
   * O caminhar pelas SETE ABAS da ficha NÃO mora aqui de propósito: ele vive no
   * `sheet.spec.ts`, junto do contraste e da tipografia, porque à parte seria
   * enumeração e a aba que nascer amanhã nasceria sem medição.
   */
  for (const surface of HOUSE_SURFACES) {
    test(`o realce de foco é o mesmo ${surface.where}`, async ({ page }) => {
      await surface.visit(page)
      await expectOneFocusRing(page, surface.where)
    })
  }
})
