import { expect, test } from '@playwright/test'
import { expectDentroDaJanela } from './support/geometry'
import { expectNoHorizontalOverflow, VIEWPORTS } from './support/viewports'

test.describe('O construtor de encontros', () => {
  test.use({ storageState: '.auth/user.json' })

  const ENCONTROS = '/mestre/encontros'

  test('o construtor cabe nos seis formatos', async ({ page }) => {
    await page.goto(`${ENCONTROS}?nivel=1&grupo=4&c=ogro:2,goblin-salteador:4`)
    await expectNoHorizontalOverflow(page, VIEWPORTS)
    for (const viewport of VIEWPORTS) {
      await page.setViewportSize({ width: viewport.width, height: viewport.height })
      await expectDentroDaJanela(page)
    }
  })

  /**
   * O RASCUNHO NÃO MEXE NO HISTÓRICO, e é essa a decisão que o dono tomou.
   *
   * A alternativa — o encontro SER o endereço — foi recusada porque a
   * quantidade muda a cada clique: cada `[+]` viraria uma entrada, e o botão
   * Voltar passaria a desfazer cliques em vez de sair da tela. Este teste é o
   * que impede a decisão de se perder num refactor.
   *
   * E2E porque o histórico é do NAVEGADOR: `history.length` e `goBack` não
   * existem em jsdom, e nenhuma asserção de servidor vê a diferença.
   */
  test('montar o encontro não empilha histórico, e o Voltar sai da tela', async ({ page }) => {
    await page.goto('/')
    await page.goto(ENCONTROS)
    const antes = await page.evaluate(() => history.length)

    await page.getByRole('searchbox', { name: 'Buscar criatura para acrescentar' }).fill('ogro')
    await page.getByRole('button', { name: /^Acrescentar Ogro/ }).first().click()
    await expect(page.getByRole('button', { name: 'Mais um Ogro' })).toBeVisible()
    await page.getByRole('button', { name: 'Mais um Ogro' }).click()
    await page.getByRole('button', { name: 'Mais um Ogro' }).click()

    expect(await page.evaluate(() => history.length), 'os cliques empilharam histórico').toBe(
      antes,
    )
    expect(new URL(page.url()).search, 'o rascunho vazou para a URL').toBe('')

    await page.goBack()
    await expect(page, 'o Voltar desfez um clique em vez de sair da tela').toHaveURL(/\/$/)
  })

  /**
   * O LINK COPIADO reabre o encontro. A ida está prendida em Go
   * (`TestTheCopiedLinkReopensTheEncounter`); o que só o browser vê é o `data-url`
   * que o botão carrega chegando ao endereço certo depois de uma navegação de
   * verdade — que é o que alguém faz ao colar no chat da mesa.
   */
  test('o link do botão de copiar reabre o mesmo encontro', async ({ page }) => {
    await page.goto(`${ENCONTROS}?nivel=3&grupo=4&c=ogro:2`)
    const veredito = page.locator('#encounters')
    const antes = await veredito.textContent()

    const link = await page
      .getByRole('button', { name: 'Copiar link do encontro' })
      .getAttribute('data-url')
    expect(link, 'o botão não carrega o endereço do encontro').toBeTruthy()

    await page.goto(link as string)
    expect(await veredito.textContent(), 'o encontro voltou diferente pelo link').toBe(antes)
  })
})
