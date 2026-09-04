import { type Page, expect, test } from '@playwright/test'
import { expectDentroDaJanela } from './support/geometry'
import { VIEWPORTS, expectNoHorizontalOverflow } from './support/viewports'

/**
 * O Grimório é a folha de especificação viva do sistema de desenho (ALE-173).
 *
 * Ele não é uma cena de jogo, então o que se afirma aqui não é jornada: é que a
 * folha continua DIZENDO A VERDADE. Uma folha de desenho que apodrece é pior
 * que nenhuma, porque quem consulta acredita nela.
 */
/**
 * As superfícies onde a casa ESCREVE com tinta semântica, e como chegar a cada
 * uma (ALE-237, ALE-240). Cena nova que escreva crimson entra aqui, ou nasce sem medição —
 * a lição da issue é que um guarda de contraste só mede o que ele VISITA, e
 * dois defeitos velhos sobreviveram anos porque nenhum guarda abria um popover
 * nem entrava no livro de campanhas.
 */
const SUPERFICIES_COM_TINTA = [
  {
    onde: 'na folha do grimório',
    abre: async (page: Page) => {
      await page.goto('/grimorio')
      await expect(page.getByRole('heading', { name: 'Grimório' })).toBeVisible()
    },
  },
  {
    onde: 'no menu do jogador',
    abre: async (page: Page) => {
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
    onde: 'no veredito do encontro',
    abre: async (page: Page) => {
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
      // (`/campaigns/1/sessions/4`, num diálogo do trilho do mestre). Com a SPA
      // apagada (ALE-272, fatia 10c) o construtor é cena própria do servidor, e
      // é ela que passa a ser medida — o guarda mede TINTA CONTRA FUNDO, e isso
      // independe de quem desenhou.
      await page.goto('/mestre/encontros')
      await expect(page.getByRole('heading', { name: 'Encontros' })).toBeVisible()
    },
  },
  {
    onde: 'no livro de campanhas',
    abre: async (page: Page) => {
      // O PERGAMINHO, que é creme e inverte a conta inteira. Ancorado no
      // cabeçalho e NÃO em "Sessão ao vivo": exigir uma sessão viva seria uma
      // asserção que mede o banco (ALE-238).
      //
      // Na base da migração (ALE-234) este endereço encaminha para a cena do
      // servidor, e é ela que passa a ser medida. Vale o mesmo do popover
      // acima: o guarda não sabe nem precisa saber qual stack desenhou.
      await page.goto('/campaigns')
      await expect(page.getByRole('heading', { name: 'Campanhas' })).toBeVisible()
    },
  },
  {
    onde: 'na aba de lugares da crônica',
    abre: async (page: Page) => {
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
    onde: 'no rascunho de lugar',
    abre: async (page: Page) => {
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
async function tintasFracas(page: Page): Promise<string[]> {
  return page.evaluate(() => {
    const tela = document.createElement('canvas')
    tela.width = 1
    tela.height = 1
    const ctx = tela.getContext('2d')
    if (!ctx) return ['sem canvas']

    const rgb = (css: string): [number, number, number, number] => {
      ctx.clearRect(0, 0, 1, 1)
      ctx.fillStyle = css
      ctx.fillRect(0, 0, 1, 1)
      const [r, g, b, a] = ctx.getImageData(0, 0, 1, 1).data
      return [r ?? 0, g ?? 0, b ?? 0, a ?? 0]
    }
    const luz = (c: [number, number, number, number]) => {
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
    const raiz = getComputedStyle(document.documentElement)
    const TINTAS = [
      '--grimorio-crimson',
      '--grimorio-crimson-bright',
      '--grimorio-parchment-crimson',
      '--hp-full',
      '--hp-hurt',
      '--hp-critical',
      '--mp-arcane',
    ]
    const tintas = TINTAS.map((token) => ({
      token,
      chave: rgb(raiz.getPropertyValue(token).trim()).slice(0, 3).join(','),
    }))

    const fundoEfetivo = (el: HTMLElement) => {
      let no: HTMLElement | null = el
      while (no) {
        const c = rgb(getComputedStyle(no).backgroundColor)
        if (c[3] >= 250) return c
        no = no.parentElement
      }
      return rgb('#000')
    }

    return [...document.querySelectorAll<HTMLElement>('*')]
      .filter((el) => {
        // Só quem PINTA texto próprio: um contêiner herda a cor e mediria a
        // mesma tinta dez vezes, com o mesmo veredito.
        const proprio = [...el.childNodes].some(
          (n) => n.nodeType === Node.TEXT_NODE && n.textContent?.trim(),
        )
        return proprio && el.getBoundingClientRect().height > 0
      })
      .map((el) => {
        const cor = rgb(getComputedStyle(el).color)
        const casa = tintas.find((t) => t.chave === [cor[0], cor[1], cor[2]].join(','))
        if (!casa) return null
        const [a, b] = [luz(cor), luz(fundoEfetivo(el))].sort((x, y) => y - x)
        const razao = ((a ?? 0) + 0.05) / ((b ?? 0) + 0.05)
        return {
          token: casa.token,
          texto: el.textContent?.trim().slice(0, 24) ?? '',
          razao: Number(razao.toFixed(2)),
        }
      })
      .filter((t) => t !== null && t.razao < 4.5)
      .map((t) => `${t?.token} em "${t?.texto}" dá ${t?.razao}:1`)
  })
}

test.describe('Grimório — a folha de especificação', () => {
  /**
   * A ladeira do raio é estritamente crescente e começa em zero.
   *
   * Este é o defeito que a página nasceu documentando: a escala do shadcn é
   * derivada de `--radius` por `sm = R−4`, e com o R antigo, de 4px, `sm` caía
   * em ZERO — passando a significar "quadrado", que é trabalho do
   * `rounded-none`. Ninguém conseguia prever, lendo o TSX, se `rounded-sm` ia
   * desenhar canto.
   *
   * A asserção é a FORMA da ladeira e não os números: prender 2/4/6/10 seria
   * prender uma decisão de desenho que pode mudar. O que não pode voltar é dois
   * degraus valendo a mesma coisa.
   *
   * Por que e2e: `--radius` só resolve em browser. Em jsdom não há `calc` de
   * variável CSS e todo degrau mede zero.
   */
  test('a ladeira do raio é estritamente crescente e começa no quadrado', async ({ page }) => {
    await page.goto('/grimorio')
    await expect(page.getByRole('heading', { name: 'Grimório' })).toBeVisible()

    const nomes = ['rounded-none', 'rounded-sm', 'rounded-md', 'rounded-lg', 'rounded-xl']
    const degraus = await page.evaluate(
      (esperados) =>
        [...document.querySelectorAll('#raio figure')]
          .map((f) => ({
            nome: f.querySelector('p')?.textContent?.trim() ?? '',
            px: Number.parseFloat(getComputedStyle(f.firstElementChild as Element).borderRadius),
          }))
          .filter((d) => esperados.includes(d.nome)),
      nomes,
    )

    expect(degraus.length, 'a folha não desenhou os cinco degraus').toBe(5)
    expect(degraus[0]?.px, 'o primeiro degrau tem de ser o canto quadrado').toBe(0)
    for (let i = 1; i < degraus.length; i++) {
      expect(
        degraus[i]?.px ?? -1,
        `${degraus[i]?.nome} não é maior que ${degraus[i - 1]?.nome} — a escala degenerou, e é o defeito que a ALE-173 consertou`,
      ).toBeGreaterThan(degraus[i - 1]?.px ?? 0)
    }
  })

  /**
   * A ladeira de tamanho é estritamente decrescente, sem dois degraus iguais.
   *
   * A casa acrescentou três tamanhos abaixo do piso do shadcn (`text-xs`, de
   * 12px, pensado para formulário) porque a mesa é densa. Eles eram 321 valores
   * arbitrários sem nome, e é justamente enquanto um degrau não tem nome que
   * ninguém percebe quando dois passam a valer a mesma coisa — foi o que
   * aconteceu com o raio antes da ALE-173.
   *
   * A asserção é a FORMA e não os números, pela mesma razão do guarda do raio.
   */
  test('a ladeira de tamanho não tem dois degraus iguais', async ({ page }) => {
    await page.goto('/grimorio')
    await expect(page.getByRole('heading', { name: 'Grimório' })).toBeVisible()

    const ordem = ['text-xs', 'text-2xs', 'text-3xs', 'text-4xs']
    const degraus = await page.evaluate((nomes) => {
      const cena = document.querySelector('.scene-grimorio') ?? document.body
      return nomes.map((nome) => {
        const alvo = document.createElement('span')
        alvo.className = nome
        cena.appendChild(alvo)
        const px = Number.parseFloat(getComputedStyle(alvo).fontSize)
        alvo.remove()
        return { nome, px }
      })
    }, ordem)

    for (let i = 1; i < degraus.length; i++) {
      expect(
        degraus[i]?.px ?? -1,
        `${degraus[i]?.nome} não é menor que ${degraus[i - 1]?.nome} — dois degraus valendo o mesmo`,
      ).toBeLessThan(degraus[i - 1]?.px ?? 0)
    }
  })

  /**
   * As legendas vêm do navegador, não da mão de quem escreveu a página.
   *
   * Se uma amostra ficar sem cor resolvida, o utilitário que ela desenha deixou
   * de existir no CSS — o sintoma exato da armadilha registrada no guia, a de
   * que classe usada só num arquivo NOVO não entra no bundle até o servidor
   * reiniciar. A folha ficaria bonita e vazia, e foi assim que ela nasceu.
   */
  test('nenhuma amostra de cor fica sem valor', async ({ page }) => {
    await page.goto('/grimorio')
    await expect(page.getByRole('heading', { name: 'Grimório' })).toBeVisible()

    const transparentes = await page.evaluate(() =>
      [...document.querySelectorAll('#cor figure')]
        .filter((f) => {
          const fundo = getComputedStyle(f.firstElementChild as Element).backgroundColor
          return fundo === 'rgba(0, 0, 0, 0)' || fundo === 'transparent'
        })
        .map((f) => f.querySelector('p')?.textContent?.trim() ?? '?'),
    )

    expect(transparentes, 'amostra sem cor: o utilitário não existe no CSS').toEqual([])
  })

  /**
   * Toda TINTA alcança 4.5:1 contra o painel (ALE-173, P3).
   *
   * Esta é a razão de os quatro papéis terem duas cores. As de bloco foram
   * afinadas para preencher — barra de vida, fundo de botão — e ficam entre
   * 3,2 e 4,6:1, abaixo do mínimo da WCAG para texto pequeno. Era por isso que
   * a cena escrevia com 74 cores CRUAS do Tailwind: não era desleixo, era
   * compensação, e nenhuma delas tinha nome.
   *
   * O guarda afirma só a metade que é REGRA — tinta serve de texto. Não prende
   * o valor de nenhuma: a paleta pode mudar de matiz, de croma ou de
   * luminosidade sem deixar de ser legível, e prender o oklch tornaria isso
   * impossível sem tocar no teste.
   *
   * Por que e2e: contraste exige converter oklch para sRGB, e só o navegador
   * faz isso — `getComputedStyle` devolve o oklch cru, e ler aqueles três
   * números como RGB dá razão inventada.
   */
  test('nenhuma tinta fica abaixo do mínimo de texto', async ({ page }) => {
    await page.goto('/grimorio')
    await expect(page.getByRole('heading', { name: 'Grimório' })).toBeVisible()

    const fracas = await page.evaluate(() => {
      const cena = document.querySelector('.scene-grimorio')
      const tela = document.createElement('canvas')
      tela.width = 1
      tela.height = 1
      const ctx = tela.getContext('2d')
      if (!ctx || !cena) return ['a cena não montou']

      const rgb = (css: string): [number, number, number] => {
        ctx.clearRect(0, 0, 1, 1)
        ctx.fillStyle = css
        ctx.fillRect(0, 0, 1, 1)
        const [r, g, b] = ctx.getImageData(0, 0, 1, 1).data
        return [r ?? 0, g ?? 0, b ?? 0]
      }
      const luz = (c: [number, number, number]) => {
        const [r, g, b] = c.map((v) => {
          const x = v / 255
          return x <= 0.04045 ? x / 12.92 : ((x + 0.055) / 1.055) ** 2.4
        })
        return 0.2126 * (r ?? 0) + 0.7152 * (g ?? 0) + 0.0722 * (b ?? 0)
      }
      const estilo = getComputedStyle(cena)
      const painel = luz(rgb(estilo.getPropertyValue('--grimorio-panel').trim()))

      return ['--bonus-ink', '--arcane-ink', '--penalty-ink', '--warning-ink']
        .map((token) => {
          const valor = estilo.getPropertyValue(token).trim()
          const [a, b] = [luz(rgb(valor)), painel].sort((x, y) => y - x)
          const razao = ((a ?? 0) + 0.05) / ((b ?? 0) + 0.05)
          return { token, razao: Number(razao.toFixed(2)) }
        })
        .filter((t) => t.razao < 4.5)
        .map((t) => `${t.token} dá ${t.razao}:1`)
    })

    expect(fracas, 'tinta que não alcança texto — ela é cor de BLOCO').toEqual([])
  })

  /**
   * Todo botão PREENCHIDO é legível sobre o próprio preenchimento (ALE-200).
   *
   * O guarda vizinho afirma que a TINTA alcança texto contra o painel. Este
   * afirma a outra metade, que ninguém estava olhando: quando o botão tem fundo
   * próprio, quem decide a legibilidade é o par fundo+texto DELE, não o painel
   * atrás.
   *
   * Foi assim que o destrutivo passou despercebido. Branco sobre o
   * crimson-bright dava **3,72:1** — abaixo dos 4,5 do AA —, e ele é o único
   * vermelho da tela: o botão que APAGA era o menos legível do app. Trocado o
   * preenchimento para o crimson base, o mesmo branco dá 5,35:1.
   *
   * Afirma a REGRA e não os valores: a paleta pode mudar de matiz, de croma ou
   * de luminosidade sem deixar de ser legível. Prender o oklch tornaria
   * qualquer repintura impossível sem tocar no teste — foi o cuidado que a
   * ALE-173 registrou no guarda das tintas, e vale igual aqui.
   *
   * Por que e2e: converter oklch para sRGB é trabalho do navegador. Em jsdom o
   * `getComputedStyle` devolve a variável CRUA, e ler aqueles três números como
   * RGB dá uma razão inventada.
   */
  test('nenhum botão preenchido fica abaixo do mínimo de texto', async ({ page }) => {
    await page.goto('/grimorio')
    await expect(page.getByRole('heading', { name: 'Grimório' })).toBeVisible()

    const fracos = await page.evaluate(() => {
      const tela = document.createElement('canvas')
      tela.width = 1
      tela.height = 1
      const ctx = tela.getContext('2d')
      if (!ctx) return ['sem canvas']

      const rgb = (css: string): [number, number, number, number] => {
        ctx.clearRect(0, 0, 1, 1)
        ctx.fillStyle = css
        ctx.fillRect(0, 0, 1, 1)
        const [r, g, b, a] = ctx.getImageData(0, 0, 1, 1).data
        return [r ?? 0, g ?? 0, b ?? 0, a ?? 0]
      }
      const luz = (c: [number, number, number, number]) => {
        const [r, g, b] = [c[0], c[1], c[2]].map((v) => {
          const x = v / 255
          return x <= 0.04045 ? x / 12.92 : ((x + 0.055) / 1.055) ** 2.4
        })
        return 0.2126 * (r ?? 0) + 0.7152 * (g ?? 0) + 0.0722 * (b ?? 0)
      }

      return [...document.querySelectorAll<HTMLElement>('[data-slot="button"]')]
        .map((botao) => {
          const estilo = getComputedStyle(botao)
          const fundo = rgb(estilo.backgroundColor)
          // Só os PREENCHIDOS: `ghost`, `outline` e `link` são transparentes, e
          // quem decide a legibilidade deles é o painel — que o guarda das
          // tintas já cobre.
          if (fundo[3] < 250) return null
          const [a, b] = [luz(rgb(estilo.color)), luz(fundo)].sort((x, y) => y - x)
          const razao = ((a ?? 0) + 0.05) / ((b ?? 0) + 0.05)
          return { variante: botao.dataset.variant ?? '?', razao: Number(razao.toFixed(2)) }
        })
        .filter((b) => b !== null && b.razao < 4.5)
        .map((b) => `${b?.variante} dá ${b?.razao}:1`)
    })

    expect(fracos, 'botão preenchido cujo texto não alcança o mínimo de leitura').toEqual([])
  })

  /**
   * O CRIMSON É LEGÍVEL EM TODA SUPERFÍCIE ONDE ELE POUSA (ALE-237).
   *
   * Os dois guardas acima medem contra UMA superfície cada: o das tintas contra
   * `--grimorio-panel`, o dos botões contra o preenchimento do próprio botão. O
   * app tem três superfícies, e foi nas outras duas que dois defeitos moraram
   * sem ninguém ver:
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
   * escrever crimson entra na lista abaixo, ou nasce sem medição.
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
  for (const cena of SUPERFICIES_COM_TINTA) {
    test(`a tinta da casa alcança texto ${cena.onde}`, async ({ page }) => {
      await cena.abre(page)
      const fracos = await tintasFracas(page)
      expect(fracos, `tinta abaixo do mínimo de texto ${cena.onde}`).toEqual([])
    })
  }

  /**
   * Todo foco da casa tem a MESMA cara (ALE-173, P4).
   *
   * Havia três gramáticas em 12 combinações — o anel do shadcn em sete
   * arquivos do kit, o contorno dourado em três, quatro variantes avulsas — e
   * 80 dos 84 `<button>` do app sem tratamento nenhum, caindo no contorno
   * padrão do navegador. Quem navega por teclado pagava a cada Tab: o realce
   * mudava de cara conforme a tela.
   *
   * Navega com TAB de verdade, e não com `.focus()`: foco programático não
   * dispara `:focus-visible` num botão, então a sonda mediria "sem contorno" e
   * passaria verde sobre qualquer coisa.
   *
   * O que se afirma é que os realces são IGUAIS ENTRE SI, não que sejam de uma
   * cor específica: a paleta pode mudar sem a casa deixar de ter uma voz só.
   * O cursor de navegação fica de fora de propósito — ele é outro estado,
   * "você está pilotando por aqui", e diz isso com brilho em vez de contorno.
   *
   * Uma coisa que este guarda ENSINOU ao ser provado: pôr de volta uma receita
   * num componente não o quebra, porque a regra global tem especificidade
   * maior que um utilitário e vence. Isso é feição e não acaso — uma peça não
   * consegue mais divergir sozinha. O que ele pega é a perda da uniformidade
   * na RAIZ: tirar um tipo de elemento da regra, ou alguém escrever algo mais
   * específico. Foi assim que ele ficou vermelho.
   */
  test('o realce de foco é o mesmo em toda a folha', async ({ page }) => {
    await page.goto('/grimorio')
    await expect(page.getByRole('heading', { name: 'Grimório' })).toBeVisible()

    // UM Tab de verdade liga o modo teclado do navegador; a partir daí o foco
    // programático também dispara `:focus-visible`, e dá para varrer a folha
    // inteira em vez dos primeiros doze paradas. A primeira versão deste
    // guarda tabulava doze vezes, só alcançava a trilha, e passou VERDE quando
    // sabotei o botão do kit com uma receita própria.
    await page.keyboard.press('Tab')

    // Desliga a transição para MEDIR. As peças do kit têm `transition-all`, e o
    // contorno vai mudando de alfa no caminho: sem isto a varredura devolve
    // cinco "caras" que são o mesmo realce em cinco instantes do trajeto.
    // Esperar cada uma assentar também funcionaria e custaria dez segundos.
    await page.addStyleTag({ content: '*, *::before, *::after { transition: none !important }' })

    const realces = await page.evaluate(async () => {
      const focaveis = [...document.querySelectorAll('a, button, input, select, textarea')].filter(
        (n) => (n as HTMLElement).offsetParent !== null && !n.closest('[data-nav-region]'),
      )
      const vistos = new Set<string>()
      for (const alvo of focaveis) {
        ;(alvo as HTMLElement).focus()
        const cs = getComputedStyle(alvo)
        if (cs.outlineStyle === 'none') continue
        vistos.add(`${cs.outlineWidth} ${cs.outlineStyle} ${cs.outlineColor} off:${cs.outlineOffset}`)
      }
      return [...vistos]
    })

    expect(realces.length, 'ninguém recebeu realce — a varredura não achou foco').toBeGreaterThan(0)
    expect(
      realces,
      'o foco tem mais de uma cara: voltou a haver receita por componente',
    ).toHaveLength(1)
  })

  /**
   * A Cinzel não desce abaixo de 14px (ALE-173).
   *
   * Ela é serifada de display — contraste de traço alto e olhos pequenos —, e
   * em 11px maiúscula com espaçamento largo vira desenho antes de virar texto.
   * O dono apontou isso olhando esta folha, e a casa já tinha tomado a mesma
   * decisão um degrau abaixo: o rótulo de campo, em 10px, nunca usou Cinzel.
   * O cabeçalho de bloco era a exceção solta.
   *
   * O guarda vive aqui porque a folha desenha a família inteira com os
   * componentes de verdade — se um deles voltar a usar Cinzel pequena, ela
   * aparece aqui antes de aparecer em 43 telas.
   *
   * Por que e2e: a face resolvida só existe em browser. Em jsdom
   * `font-family` devolve a string do CSS, não o que foi de fato usado.
   */
  test('a Cinzel não desce abaixo do piso de leitura', async ({ page }) => {
    await page.goto('/grimorio')
    await expect(page.getByRole('heading', { name: 'Grimório' })).toBeVisible()

    const pequenas = await page.evaluate(() =>
      [...document.querySelectorAll('*')]
        .filter((n) => {
          const cs = getComputedStyle(n)
          if (!cs.fontFamily.startsWith('Cinzel')) return false
          if (!(n.textContent ?? '').trim()) return false
          return Number.parseFloat(cs.fontSize) < 14
        })
        .map((n) => `${Math.round(Number.parseFloat(getComputedStyle(n).fontSize))}px: ${(n.textContent ?? '').trim().slice(0, 24)}`)
        .slice(0, 6),
    )

    expect(
      pequenas,
      'Cinzel abaixo de 14px — ela não se lê nesse tamanho, e o piso da casa é 14',
    ).toEqual([])
  })

  test('a folha cabe nos seis formatos', async ({ page }) => {
    await page.goto('/grimorio')
    await expect(page.getByRole('heading', { name: 'Grimório' })).toBeVisible()

    await expectNoHorizontalOverflow(page, VIEWPORTS)
    for (const viewport of VIEWPORTS) {
      await page.setViewportSize({ width: viewport.width, height: viewport.height })
      await expectDentroDaJanela(page)
    }
  })
})
