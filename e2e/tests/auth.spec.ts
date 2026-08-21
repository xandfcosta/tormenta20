import { expect, test } from '@playwright/test'
import { expectDentroDaJanela } from './support/geometry'
import { VIEWPORTS, expectNoHorizontalOverflow } from './support/viewports'

/**
 * A porta do jogo fala a língua da casa (ALE-173).
 *
 * /login, /register e /redefinir-senha eram o split-screen do shadcn — painel
 * de marca à esquerda, formulário à direita, tokens claros — e eram a última
 * superfície do app que ainda falava a língua do template. Agora são cena.
 *
 * O que se afirma é o que o jogador VÊ: a porta é escura como a mesa. Não é
 * afirmação de classe: é a cor que o navegador pintou, e o defeito que ela
 * impede é o template genérico voltar por cima.
 *
 * Por que e2e: cor computada de token que depende de escopo. Em jsdom nenhuma
 * variável CSS resolve e toda superfície mede transparente.
 */
test.describe('A porta do jogo', () => {
  test.use({ storageState: { cookies: [], origins: [] } })

  for (const rota of ['/login', '/register', '/redefinir-senha']) {
    test(`${rota} é escura como a mesa`, async ({ page }) => {
      await page.goto(rota)
      await expect(page.getByRole('heading', { name: 'Tormenta 20' })).toBeVisible()

      // Compara token com token, lidos do mesmo nó: prender o oklch literal
      // seria prender uma decisão de paleta que pode mudar sem que a porta
      // deixe de ser a mesa.
      const cores = await page.evaluate(() => {
        const cena = document.querySelector('[data-slot=scene-shell]')
        if (!cena) return null
        const cs = getComputedStyle(cena)
        return {
          fundo: cs.getPropertyValue('--background').trim(),
          mesa: cs.getPropertyValue('--grimorio-bg').trim(),
          painel: cs.getPropertyValue('--popover').trim(),
        }
      })

      expect(cores, 'a porta não montou dentro de uma cena').not.toBeNull()
      expect(
        cores?.fundo,
        'a porta não pinta a cor da mesa — voltou a resolver o tema claro do template',
      ).toBe(cores?.mesa)
    })
  }

  test('a porta cabe nos seis formatos', async ({ page }) => {
    await page.goto('/login')
    await expect(page.getByRole('heading', { name: 'Tormenta 20' })).toBeVisible()

    await expectNoHorizontalOverflow(page, VIEWPORTS)
    for (const viewport of VIEWPORTS) {
      await page.setViewportSize({ width: viewport.width, height: viewport.height })
      await expectDentroDaJanela(page)
    }
  })
})
