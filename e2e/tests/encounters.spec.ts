import { expect, test } from '@playwright/test'
import { expectDentroDaJanela } from './support/geometry'
import { expectNoHorizontalOverflow, VIEWPORTS } from './support/viewports'

test.describe('O construtor de encontros', () => {
  test.use({ storageState: '.auth/user.json' })

  const ENCOUNTERS = '/mestre/encontros'

  test('o construtor cabe nos seis formatos', async ({ page }) => {
    await page.goto(`${ENCOUNTERS}?nivel=1&grupo=4&c=ogro:2,goblin-salteador:4`)
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
    await page.goto(ENCOUNTERS)
    const before = await page.evaluate(() => history.length)

    await page.getByRole('searchbox', { name: 'Buscar criatura' }).fill('ogro')
    await page.getByRole('button', { name: /^Acrescentar Ogro/ }).first().click()
    await expect(page.getByRole('button', { name: 'Mais um Ogro' })).toBeVisible()
    await page.getByRole('button', { name: 'Mais um Ogro' }).click()
    await page.getByRole('button', { name: 'Mais um Ogro' }).click()

    expect(await page.evaluate(() => history.length), 'os cliques empilharam histórico').toBe(
      before,
    )
    expect(new URL(page.url()).search, 'o rascunho vazou para a URL').toBe('')

    await page.goBack()
    await expect(page, 'o Voltar desfez um clique em vez de sair da tela').toHaveURL(/\/$/)
  })

  /**
   * O LINK COPIADO reabre o encontro. A ida está prendida em Go
   * (`TestTheCopiedLinkReopensTheEncounter`); o que só o browser vê é o `data-url`
   * que a cena carrega chegando ao endereço certo depois de uma navegação de
   * verdade — que é o que alguém faz ao colar no chat da mesa.
   */
  test('o link do botão de copiar reabre o mesmo encontro', async ({ page }) => {
    await page.goto(`${ENCOUNTERS}?nivel=3&grupo=4&c=ogro:2`)
    const verdict = page.locator('#encounters')
    const before = await verdict.textContent()

    await expect(page.getByRole('button', { name: 'Copiar link' })).toBeVisible()
    const link = await verdict.getAttribute('data-url')
    expect(link, 'o botão não carrega o endereço do encontro').toBeTruthy()

    await page.goto(link as string)
    expect(await verdict.textContent(), 'o encontro voltou diferente pelo link').toBe(before)
  })

  /**
   * NO CELULAR o diálogo de acrescentar fica aberto a cada criatura, e o veredito atrás
   * dele acompanha. O remendo passa por baixo do diálogo, e só o navegador diz se ele
   * sobrevive aberto.
   */
  test('no celular o diálogo de acrescentar fica aberto e o veredito acompanha', async ({ page }) => {
    await page.setViewportSize({ width: 390, height: 844 })
    await page.goto(ENCOUNTERS)
    const nd = page.locator('.ds-encounter-nd > b')
    await expect(nd).toHaveText('0')

    await page.getByRole('button', { name: 'Acrescentar criatura' }).click()
    const dialog = page.getByRole('dialog', { name: 'Acrescentar criatura' })
    await dialog.getByRole('button', { name: 'Acrescentar Ogro ao encontro' }).click()
    await expect(nd, 'o veredito não acompanhou: um ogro é ND 4').toHaveText('4')
    await dialog.getByRole('button', { name: 'Acrescentar Ogro ao encontro' }).click()
    await expect(nd, 'dois ogros são ND 6 (p282)').toHaveText('6')

    await expect(dialog, 'o remendo fechou o diálogo').toBeVisible()
    await expect(dialog.getByRole('button', { name: 'Acrescentar Ogro ao encontro' })).toContainText('2×')
  })

  /**
   * O BOTÃO mora no cabeçalho, que o remendo não alcança, e copia o que a cena diz
   * AGORA. Se ele guardasse o endereço de quando a página abriu, o mestre colaria no
   * chat um encontro vazio.
   */
  test('copiar o link depois de montar leva o encontro montado', async ({ page, context }) => {
    await context.grantPermissions(['clipboard-read', 'clipboard-write'])
    await page.goto(ENCOUNTERS)
    await page.getByRole('button', { name: 'Acrescentar Ogro ao encontro' }).click()
    await expect(page.getByRole('button', { name: 'Mais um Ogro' })).toBeVisible()

    await page.getByRole('button', { name: 'Copiar link' }).click()
    await expect(page.getByRole('button', { name: 'Link copiado' })).toBeVisible()
    const copied = await page.evaluate(() => navigator.clipboard.readText())
    expect(decodeURIComponent(copied), 'o link copiado não traz o ogro').toContain('c=ogro:1')
  })

  /**
   * A SETA anda do menu para as linhas do encontro e para a lista, e nunca cai num
   * campo de número: lá ela mudaria o nível do grupo sem ninguém pedir. E2E porque a
   * escolha do vizinho é geometria real, e só existe com o driver ligado.
   */
  test('a seta passa pelas criaturas e não mexe no nível do grupo', async ({ page }) => {
    await page.setViewportSize({ width: 1400, height: 900 })
    await page.goto(`${ENCOUNTERS}?nivel=3&grupo=4&c=ogro:2`)
    await page.waitForLoadState('networkidle')
    const region = () =>
      page.evaluate(() => document.activeElement?.closest('[data-nav-region]')?.getAttribute('data-nav-region'))

    await page.locator('.ds-side-menu-link[aria-current]').focus()
    await page.keyboard.press('ArrowRight')
    await expect.poll(region).toBe('composicao')
    await page.keyboard.press('ArrowRight')
    await page.keyboard.press('ArrowRight')
    await page.keyboard.press('ArrowRight')
    await expect.poll(region).toBe('criaturas')

    await expect(page.getByLabel('Nível do grupo')).toHaveValue('3')
  })
})
