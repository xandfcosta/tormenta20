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
