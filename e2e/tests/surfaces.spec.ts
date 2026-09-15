import { expect, type Page, test } from '@playwright/test'
import { expectOneFocusRing } from './support/focus'

/**
 * AS SUPERFÍCIES DA CASA, e as duas medições que percorrem todas elas.
 *
 * A lista morava dentro do `grimorio.spec.ts`, e o lugar era a contradição: ela
 * não é do grimório — ela abre o menu do jogador, o livro de campanhas, as notas
 * e o rascunho de lugar. "Cena nova que escreve crimson entra aqui" só alcança
 * quem já está no arquivo da folha de especificação, que é o último lugar onde
 * se procura ao criar uma cena. É a lição da ALE-272 aplicada à LISTA e não ao
 * medidor: o que mora dentro de um chamador tem exatamente um chamador.
 *
 * **Cena nova onde a casa escreva tinta semântica entra na lista abaixo, no
 * MESMO commit que a cria** — o regime aqui é ENUMERAÇÃO, e a que alguém
 * esquecer nasce sem medição, em silêncio (ALE-252).
 */
/**
 * As superfícies onde a casa ESCREVE com tinta semântica, e como chegar a cada
 * uma (ALE-237, ALE-240). Cena nova que escreva crimson entra aqui, ou nasce sem medição —
 * a lição da issue é que um guarda de contraste só mede o que ele VISITA, e
 * dois defeitos velhos sobreviveram anos porque nenhum guarda abria um popover
 * nem entrava no livro de campanhas.
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
      // e é literalmente por isso que nenhum guarda o tinha medido.
      //
      // Na base da migração (ALE-225) o `/` encaminha para o Hub do SERVIDOR e
      // este popover é o nativo, não o do Kobalte. O guarda atravessa a virada
      // sem mudar porque ele mede TINTA CONTRA FUNDO, e isso independe de quem
      // desenhou — e é o desfecho certo: ele passou a medir a tela de verdade.
      await page.goto('/')
      await page.getByRole('button', { name: 'Menu de Mestre' }).click()
      await expect(page.getByRole('button', { name: 'Sair' })).toBeVisible()
    },
  },
  {
    where: 'no veredito do encontro',
    visit: async (page: Page) => {
      // O veredito do encontro escreve com a família VITAL, e é a única tela
      // fora da folha onde ela vira texto grande (ALE-240).
      //
      // O que ele mede é o tom que estiver na tela: com o rascunho vazio isso é
      // "Trivial". Ele NÃO exercita o "Mortal" — montar um encontro mortal aqui
      // custaria semear criaturas, e o token daquele tom já é a tinta da casa,
      // medida nas outras cenas por identidade. O que esta entrada garante é
      // que a família vital continue legível ONDE ela escreve grande, e que a
      // tela entre na varredura quando alguém puser tinta nova nela.
      //
      // O endereço era o do construtor DENTRO da sessão da SPA
      // (`/campanhas/1/sessoes/4`, num diálogo do trilho do mestre). Com a SPA
      // apagada (ALE-272, fatia 10c) o construtor é cena própria do servidor, e
      // é ela que passa a ser medida — o guarda mede TINTA CONTRA FUNDO, e isso
      // independe de quem desenhou.
      await page.goto('/mestre/encontros')
      await expect(page.getByRole('heading', { name: 'Encontros' })).toBeVisible()
    },
  },
  {
    where: 'no livro de campanhas',
    visit: async (page: Page) => {
      // O PERGAMINHO, que é creme e inverte a conta inteira. Ancorado no
      // cabeçalho e NÃO em "Sessão ao vivo": exigir uma sessão viva seria uma
      // asserção que mede o banco (ALE-238).
      //
      // Na base da migração (ALE-234) este endereço encaminha para a cena do
      // servidor, e é ela que passa a ser medida. Vale o mesmo do popover
      // acima: o guarda não sabe nem precisa saber qual stack desenhou.
      await page.goto('/campanhas')
      await expect(page.getByRole('heading', { name: 'Campanhas' })).toBeVisible()
    },
  },
  {
    where: 'na aba de lugares da crônica',
    visit: async (page: Page) => {
      // O ACERVO DA CAMPANHA (ALE-292), sobre o PERGAMINHO da crônica — que é
      // creme e inverte a conta inteira, como a lista de campanhas acima.
      //
      // Cena nova entra aqui NO MESMO commit que a cria, e não depois: o regime
      // desta lista é ENUMERAÇÃO, então a que alguém esquecer nasce sem medição,
      // em silêncio, que é a marca desta família (ALE-252).
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
      // A CENA DAS NOTAS FORA DA MESA (ALE-218): mesmo painel, sem mapa em
      // volta. Ela entra aqui no MESMO commit que a cria, pela razão que a
      // entrada do rascunho registra logo abaixo — o regime desta lista é
      // ENUMERAÇÃO, e a cena que alguém esquecer nasce sem medição.
      //
      // Endereço direto e não navegado: o gesto que a abre é um `window.open`,
      // e uma janela nova não é a `page` deste guarda. O id da sessão vem da
      // seed e é o mesmo que os outros specs da mesa usam.
      await page.goto('/mesa/1/4/notas')
      await expect(page.getByRole('heading', { name: /Notas/ })).toBeVisible()
    },
  },
  {
    where: 'no rascunho de lugar',
    visit: async (page: Page) => {
      // O RASCUNHO (ALE-292) é a superfície do TABULEIRO fora da sessão, e ela
      // escreve tinta que nenhuma outra entrada desta lista alcança: a tarja
      // dourada do modo, o nome da peça sobre o chão de cripta, e o terreno
      // difícil por baixo dela.
      //
      // A cena vem da SEMENTE e não de um clique de criação: criar o lugar aqui
      // deixaria uma linha para trás no banco compartilhado da corrida, e o
      // guarda de contraste não é lugar para escrever.
      //
      // MAS O CAMINHO É NAVEGADO, e não um id no endereço. O id era `lugares/1`
      // e virou `lugares/4` no dia em que a seed ganhou o acervo da campanha 1
      // (ALE-271) — um id de AUTOINCREMENT numa fixture compartilhada é um
      // acoplamento que quebra por uma mudança sem relação nenhuma, e o erro
      // ("heading não encontrado") não aponta para a causa.
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
     *  coisa avermelhada": a primeira versão deste guarda casava por aparência
     *  e não sabia dizer QUAL token estava falhando.
     *
     *  A lista tem duas famílias porque o defeito é o mesmo dos dois lados. Os
     *  três crimsons são tinta por construção (ALE-237). Os quatro VITAIS não
     *  são: eles foram escolhidos como cor de BARRA, e escrevem em alguns
     *  lugares por hábito — o verde e o azul passam por sorte (5,34 e 6,34
     *  medidos), e o `--hp-critical` não passava em superfície nenhuma. Por
     *  isso eles entram: o dia em que alguém escrever com um deles numa tela
     *  nova, é aqui que se descobre (ALE-240). */
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
   * O CRIMSON É LEGÍVEL EM TODA SUPERFÍCIE ONDE ELE POUSA (ALE-237).
   *
   * Os dois guardas do `grimorio.spec.ts` medem contra UMA superfície cada: o
   * das tintas contra `--grimorio-panel`, o dos botões contra o preenchimento do
   * próprio botão. O app tem três superfícies, e foi nas outras duas que dois
   * defeitos moraram sem ninguém ver:
   *
   * | superfície            | onde                       | rendia   |
   * | --------------------- | -------------------------- | -------- |
   * | `panel`               | a folha do grimório        | 4,61:1   |
   * | `panel-raised`        | "Sair", no menu do jogador | **4,07** |
   * | `parchment`           | "Sessão ao vivo", no livro | **3,95** |
   *
   * A lição não é "faltavam dois consertos", é que **um guarda só mede o que
   * ele VISITA**. Os dois defeitos eram velhos e o app tinha guarda de
   * contraste o tempo todo — ele só nunca tinha ABERTO um popover nem entrado
   * na cena de campanhas. Por isso este aqui NAVEGA: cena nova onde a casa
   * escrever crimson entra na lista do topo do arquivo, ou nasce sem medição.
   *
   * Mede o fundo EFETIVO, subindo a árvore até o primeiro fundo opaco — é
   * exatamente aí que os dois se escondiam, porque o elemento do texto é
   * transparente e o painel de trás não é o que se imagina.
   *
   * Por que e2e: converter oklch para sRGB é trabalho do navegador; em jsdom o
   * `getComputedStyle` devolve o oklch cru e ler aqueles três números como RGB
   * dá razão inventada. E o popover só existe depois de um clique de verdade.
   */
  /**
   * UM TESTE POR SUPERFÍCIE, e isso não é estilo: visitar `/grimorio` e depois
   * navegar mais duas vezes no MESMO contexto derruba a terceira página em
   * branco, com `net::ERR_INSUFFICIENT_RESOURCES` no console. A folha puxa o
   * sistema de desenho inteiro como módulos soltos no servidor de
   * desenvolvimento, e o Chromium estoura o limite de recursos por página.
   * Isolado em quatro combinações: `/campaigns` sozinho e `/` → `/campaigns`
   * montam; qualquer caminho que comece na folha e navegue duas vezes, não.
   * É artefato do dev server — em produção a SPA sai empacotada —, mas ele
   * mente igual, e um `expect` cansado dentro de um teste desses acusaria o
   * app. Cada teste ganha página nova.
   */
  for (const surface of HOUSE_SURFACES) {
    test(`a tinta da casa alcança texto ${surface.where}`, async ({ page }) => {
      await surface.visit(page)
      const weak = await weakTints(page)
      expect(weak, `tinta abaixo do mínimo de texto ${surface.where}`).toEqual([])
    })
  }

  /**
   * Todo foco da casa tem a MESMA cara (ALE-173, P4), em TODA superfície.
   *
   * Havia três gramáticas em 12 combinações — o anel do shadcn em sete
   * arquivos do kit, o contorno dourado em três, quatro variantes avulsas — e
   * 80 dos 84 `<button>` do app sem tratamento nenhum, caindo no contorno
   * padrão do navegador. Quem navega por teclado pagava a cada Tab: o realce
   * mudava de cara conforme a tela.
   *
   * # O que mudou na ALE-318, e por que ele deixou de ser um `test()` só
   *
   * Aqui morava a varredura INLINE, e ela abria `/grimorio` e mais nada — a
   * folha de especificação, onde tudo passa pelo kit por construção. Ela
   * também injetava `transition: none` antes de medir e PULAVA todo nó sem
   * contorno, que são os dois pontos cegos que a ALE-318 fechou. O medidor
   * virou `support/focus.ts`; o que ficou aqui é a lista de superfícies, a
   * mesma que o guarda de tinta percorre — cena nova entra numa linha e ganha
   * as duas medições.
   *
   * O caminhar pelas SETE ABAS da ficha mora no `sheet.spec.ts`, junto
   * do contraste e da tipografia, pela razão que aquele caso registra: à parte
   * ele seria enumeração, e a aba que nascer amanhã nasceria sem medição.
   */
  for (const surface of HOUSE_SURFACES) {
    test(`o realce de foco é o mesmo ${surface.where}`, async ({ page }) => {
      await surface.visit(page)
      await expectOneFocusRing(page, surface.where)
    })
  }
})
