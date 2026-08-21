import { expect, test } from '@playwright/test'
import { expectDentroDaJanela } from './support/geometry'
import { VIEWPORTS, expectNoHorizontalOverflow } from './support/viewports'

/**
 * O Grimório é a folha de especificação viva do sistema de desenho (ALE-173).
 *
 * Ele não é uma cena de jogo, então o que se afirma aqui não é jornada: é que a
 * folha continua DIZENDO A VERDADE. Uma folha de desenho que apodrece é pior
 * que nenhuma, porque quem consulta acredita nela.
 */
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
