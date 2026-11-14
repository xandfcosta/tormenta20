import { expect, test } from '@playwright/test'
import { expectNoHorizontalOverflow, VIEWPORTS } from './support/viewports'

test.describe('O bestiário', () => {
  test.use({ storageState: '.auth/user.json' })

  const BESTIARY = '/mestre/bestiario'

  test('o bestiário cabe nos seis formatos', async ({ page }) => {
    await page.goto(BESTIARY)
    await expect(page.getByRole('heading', { name: 'Bestiário' })).toBeVisible()

    await expectNoHorizontalOverflow(page, VIEWPORTS)
  })

  /**
   * A METADE ESTREITA, e é o único teste desta cena que precisa mesmo de browser.
   *
   * O painel da criatura só existe a partir de 50rem de CONTÊINER — a área
   * principal da moldura —, e nem o jsdom nem uma asserção de classe resolvem
   * consulta de contêiner. Abaixo do ponto de
   * troca a ficha tem de estar no diálogo, e acima dela tem de estar no painel;
   * a faixa em que as duas somem, ou em que as duas aparecem, é o defeito que
   * este teste existe para pegar.
   *
   * Ele mede VISIBILIDADE REAL (`toBeVisible`), que é o que resolve a cascata
   * inteira — a consulta de contêiner, o `display:none` da folha e o
   * `data-show` do Datastar decidindo juntos.
   */
  test('a ficha vive no painel quando cabe, e no diálogo quando não cabe', async ({ page }) => {
    await page.goto(BESTIARY)
    const panel = page.getByRole('region', { name: 'Criatura escolhida' })
    const dialog = page.getByRole('dialog')

    await page.setViewportSize({ width: 1440, height: 900 })
    await expect(panel, 'o painel sumiu numa largura que comporta duas colunas').toBeVisible()
    await expect(dialog, 'o diálogo apareceu por cima do painel').toBeHidden()

    // No telefone o painel não cabe: a ficha só é alcançável pelo diálogo, e é
    // ele que impede a lista de virar uma lista sem detalhe nenhum.
    await page.setViewportSize({ width: 390, height: 844 })
    await expect(panel, 'o painel ficou visível onde não cabe').toBeHidden()
    await expect(dialog, 'o diálogo abriu sozinho').toBeHidden()

    await page.locator('#bestiary a[href*="criatura="]').first().click()
    await expect(dialog, 'tocar na linha não abriu a ficha no telefone').toBeVisible()

    await page.keyboard.press('Escape')
    await expect(dialog, 'o Esc não fechou a ficha').toBeHidden()
  })

})
