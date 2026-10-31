import { expect, test } from '@playwright/test'
import { expectNoHorizontalOverflow, VIEWPORTS } from './support/viewports'

test.describe('A cena de personagens', () => {
  test.use({ storageState: '.auth/user.json' })

  /**
   * A BUSCA NÃO ROLA COM A LISTA (ALE-528, revisão do dono): ela fica presa no
   * topo de quem rola, e os cartões passam por baixo dela.
   *
   * E2E porque `position: sticky` só existe com leiaute de verdade: o servidor
   * escreve a mesma barra presa ou solta, e a diferença é o rolador.
   *
   * O CONTROLE é o rolador ter rolado, e o primeiro cartão ter saído de baixo
   * da busca: sem isso, uma lista curta deixaria a busca no lugar sem rolar
   * nada, e o caso passaria verde sobre o nada.
   */
  test('a busca fica à vista quando a lista rola', async ({ page }) => {
    await page.setViewportSize({ width: 390, height: 844 })
    await page.goto('/personagens')
    const search = page.getByRole('searchbox', { name: 'Buscar personagem' })
    await expect(search).toBeVisible()

    const scroller = page.locator('[data-slot="scene-content"]')
    const scrolled = await scroller.evaluate((el) => {
      el.scrollTop = el.scrollHeight
      return el.scrollTop
    })
    expect(scrolled, 'a lista não rolou — o elenco da semente é curto demais para o caso').toBeGreaterThan(100)

    const box = await search.boundingBox()
    expect(box, 'busca não medida').not.toBeNull()
    expect(box!.y, 'a busca saiu da tela junto com a lista').toBeGreaterThanOrEqual(0)

    // Ninguém por cima dela: o ponto do meio do campo é o próprio campo.
    const onTop = await page.evaluate(
      ([x, y]) => (document.elementFromPoint(x, y) as HTMLElement | null)?.getAttribute('aria-label'),
      [box!.x + box!.width / 2, box!.y + box!.height / 2],
    )
    expect(onTop, 'um cartão passou por cima da busca').toBe('Buscar personagem')

    const firstCard = await page.locator('.ds-cast-card').first().boundingBox()
    expect(firstCard!.y, 'o primeiro cartão não passou por baixo da busca').toBeLessThan(box!.y)
  })

  // Tela nova se valida nos seis formatos — regra da casa, e overflow é layout.
  test('personagens: sem scroll horizontal nos seis formatos', async ({ page }) => {
    await page.goto('/personagens')
    await expect(page.getByRole('searchbox', { name: 'Buscar personagem' })).toBeVisible()

    await expectNoHorizontalOverflow(page, VIEWPORTS)
  })
})
