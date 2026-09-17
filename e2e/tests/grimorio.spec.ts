import { expect, test } from '@playwright/test'
import { expectBotoesComLimiteVisivel } from './support/boundary'
import { expectDentroDaJanela } from './support/geometry'
import { expectCinzelAcimaDoPiso } from './support/typography'
import { expectNoHorizontalOverflow, VIEWPORTS } from './support/viewports'

/**
 * O Grimório é a folha de especificação viva do sistema de desenho.
 *
 * Ele não é uma cena de jogo, então o que se afirma aqui não é jornada: é que a
 * folha continua DIZENDO A VERDADE. Uma folha de desenho que apodrece é pior
 * que nenhuma, porque quem consulta acredita nela.
 *
 * O que ele NÃO é: a varredura das superfícies da casa. A lista de cenas que
 * escrevem tinta semântica, e as duas medições que a percorrem, moram no
 * `surfaces.spec.ts` — enterradas aqui, a instrução "cena nova entra na lista"
 * só alcançaria quem já tivesse aberto o arquivo da folha.
 */

test.describe('Grimório — a folha de especificação', () => {
  /**
   * A ladeira do raio é estritamente crescente e começa em zero.
   *
   * A escala do shadcn é derivada de `--radius` por `sm = R−4`: com um R de 4px,
   * `sm` cai em ZERO e passa a significar "quadrado", que é trabalho do
   * `rounded-none`. Dois degraus valendo a mesma coisa não se enxerga lendo o
   * código.
   *
   * A asserção é a FORMA da ladeira e não os números: prender 2/4/6/10 seria
   * prender uma decisão de desenho que pode mudar.
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
   * 12px, pensado para formulário) porque a mesa é densa. Enquanto um degrau não
   * tem nome, ninguém percebe quando dois passam a valer a mesma coisa.
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
   * de existir no CSS — o sintoma da armadilha do guia: classe usada só num
   * arquivo NOVO não entra no bundle até o servidor reiniciar, e a folha fica
   * bonita e vazia.
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
   * Toda TINTA alcança 4.5:1 contra o painel.
   *
   * Esta é a razão de os quatro papéis terem duas cores: as de BLOCO foram
   * afinadas para preencher — barra de vida, fundo de botão — e ficam abaixo do
   * mínimo da WCAG para texto pequeno.
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
   * Todo botão PREENCHIDO é legível sobre o próprio preenchimento.
   *
   * O guarda vizinho afirma que a TINTA alcança texto contra o painel. Este
   * afirma a outra metade: quando o botão tem fundo próprio, quem decide a
   * legibilidade é o par fundo+texto DELE, não o painel atrás. Foi assim que o
   * botão destrutivo — o único vermelho da tela, e o que APAGA — passou anos
   * abaixo dos 4,5 do AA.
   *
   * Afirma a REGRA e não os valores: a paleta pode mudar de matiz, de croma ou
   * de luminosidade sem deixar de ser legível, e prender o oklch tornaria
   * qualquer repintura impossível sem tocar no teste.
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
   * A Cinzel não desce abaixo do piso de 14px.
   *
   * O medidor mora em `support/typography.ts` e não aqui dentro: instrumento que
   * vive dentro de um chamador tem exatamente um chamador, e enquanto ele era
   * inline neste `test()` visitava um endereço só — quatro violações viveram em
   * três cenas com ele no ar o tempo todo.
   *
   * Este caso FICA e não virou redundante: a folha é a superfície onde a decisão
   * foi tomada, e ele é uma das cenas medidas em vez de a única.
   */
  test('a Cinzel não desce abaixo do piso de leitura', async ({ page }) => {
    await page.goto('/grimorio')
    await expect(page.getByRole('heading', { name: 'Grimório' })).toBeVisible()

    await expectCinzelAcimaDoPiso(page, 'na folha de especificação')
  })

  /**
   * TODO BOTÃO TEM LIMITE VISÍVEL contra o fundo (WCAG 1.4.11).
   *
   * Ele mora aqui, e o lugar é o argumento: a folha de especificação desenha
   * TODAS as variantes e TODOS os tamanhos lado a lado, então medir esta tela é
   * medir a família inteira por AMOSTRAGEM em vez de por enumeração.
   *
   * O que ele prende é a BORDA do `secondary`: o preenchimento sozinho dá 1,30:1
   * contra o fundo da cena e a borda dá 3,57:1. Sem este guarda, tirá-la não
   * quebra nada que alguém veja — o botão continua clicável e o texto continua
   * legível, e o que some é a fronteira.
   */
  test('todo botão tem limite visível contra o fundo', async ({ page }) => {
    await page.goto('/grimorio')
    await expect(page.getByRole('heading', { name: 'Grimório' })).toBeVisible()

    await expectBotoesComLimiteVisivel(page, 'na folha de especificação')
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

  /**
   * A CELA TRAZ MEDIDA, e a LADEIRA CRESCE.
   *
   * Um guarda que só checasse "a seção existe" passaria verde sobre uma coluna
   * CONSTANTE, que foi o sintoma do defeito original: a legenda dizia 36/36/36
   * para xs/sm/lg porque o tamanho nunca chegava ao elemento medido.
   *
   * E2E porque a legenda é escrita com `getBoundingClientRect` e
   * `getComputedStyle` depois do layout assentar — nada disso existe sem
   * navegador.
   */
  test('cada peça do kit sai com a medida do navegador, e a ladeira cresce', async ({ page }) => {
    await page.goto('/grimorio')
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

    // O DENOMINADOR: uma linha que perdeu a cela — ou um nome que deixou de
    // existir — devolve lista vazia, e lista vazia passaria calada em toda
    // asserção de ladeira abaixo.
    for (const [nome, celas] of Object.entries(alturas)) {
      expect(celas.length, `a linha ${nome} não tem exatamente uma cela medida`).toBe(1)
      expect(celas[0], `a cela de ${nome} não mediu`).toBeGreaterThan(0)
    }
    // A ladeira é o que pega a coluna CONSTANTE: nenhuma asserção de "mediu"
    // separa 36/36/36 de três tamanhos de verdade.
    expect(alturas.xs[0], 'xs não é menor que sm').toBeLessThan(alturas.sm[0] as number)
    expect(alturas.sm[0], 'sm não é menor que lg').toBeLessThan(alturas.lg[0] as number)
  })

  // Não há caso de SHADOW ROOT aqui de propósito: ele prendia o `noShadowDOM()`
  // dos elementos customizados da SPA, e as peças desta folha são `templ`
  // renderizado no servidor — HTML de servidor não tem shadow root onde
  // esconder o Tailwind.

})
