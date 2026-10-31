import { expect, test } from '@playwright/test'
import { expectDentroDaJanela } from './support/geometry'
import { VIEWPORTS, expectNoHorizontalOverflow } from './support/viewports'

/**
 * A porta do jogo SEGUE O TEMA DO APARELHO (decisão do dono, ALE-524): ela
 * veste o sistema novo, e o modo claro ou escuro é o do sistema operacional —
 * não há seletor antes do login.
 *
 * O que se afirma é o que o jogador VÊ: a luminância do fundo que o navegador
 * pintou, nos dois modos. Não é afirmação de classe.
 *
 * Por que e2e: `prefers-color-scheme` e o `light-dark()` dos tokens só
 * resolvem num navegador. Em jsdom toda superfície mede transparente.
 */
test.describe('A porta do jogo', () => {
  test.use({ storageState: { cookies: [], origins: [] } })

  for (const [route, heading] of [
    ['/entrar', 'Entrar'],
    ['/redefinir-senha', 'Escolher nova senha'],
  ] as const) {
    test(`${route} segue o tema do aparelho`, async ({ page }) => {
      const luminance = async () =>
        page.evaluate(() => {
          const scene = document.querySelector('[data-slot=scene-shell].scene-ds')
          if (!scene) return null
          const [r, g, b] = getComputedStyle(scene).backgroundColor.match(/\d+(\.\d+)?/g)!.map(Number)
          return (0.2126 * r + 0.7152 * g + 0.0722 * b) / 255
        })

      await page.emulateMedia({ colorScheme: 'light' })
      await page.goto(route)
      await expect(page.getByRole('heading', { name: heading })).toBeVisible()
      const light = await luminance()
      expect(light, 'a porta não montou dentro de uma cena do sistema novo').not.toBeNull()
      expect(light!, 'com o aparelho no claro, a porta não ficou clara').toBeGreaterThan(0.7)

      await page.emulateMedia({ colorScheme: 'dark' })
      await page.goto(route)
      expect(await luminance(), 'com o aparelho no escuro, a porta não ficou escura').toBeLessThan(0.2)
    })
  }

  // NÃO há caso de "criar conta só abre com convite", e a ausência é
  // deliberada: a porta é do SERVIDOR e está presa em Go
  // (`TestRegisterRequiresAnInvite`, `TestRegisterWithAnInviteWorksExactlyOnce`,
  // `TestExpiredInviteIsRejected`). Esconder a tela é UX sobre uma porta já
  // fechada, e afirmar UI não move a fronteira de segurança.

  test('a porta cabe nos seis formatos', async ({ page }) => {
    await page.goto('/entrar')
    await expect(page.getByRole('heading', { name: 'Entrar' })).toBeVisible()

    await expectNoHorizontalOverflow(page, VIEWPORTS)
    for (const viewport of VIEWPORTS) {
      await page.setViewportSize({ width: viewport.width, height: viewport.height })
      await expectDentroDaJanela(page)
    }
  })
})
