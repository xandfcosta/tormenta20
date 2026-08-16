import { type Page, expect } from '@playwright/test'

/** The six form factors `frontend/CLAUDE.md` requires every scene to survive. */
export const VIEWPORTS = [
  { name: 'desktop', width: 1920, height: 1080 },
  { name: 'laptop', width: 1440, height: 900 },
  { name: 'tablet-landscape', width: 1024, height: 768 },
  { name: 'tablet-portrait', width: 768, height: 1024 },
  { name: 'mobile-landscape', width: 844, height: 390 },
  { name: 'mobile-portrait', width: 390, height: 844 },
] as const

/** Laptop and up — where the keyboard layer answers (`≥xl` + `pointer: fine`). */
export const DESK_VIEWPORTS = VIEWPORTS.filter((v) => v.width >= 1280)

/**
 * Resizes through every form factor on the CURRENT page and fails on the first
 * one that makes the document scroll sideways.
 *
 * One `goto`, six resizes — not six navigations. The house rule is that media
 * queries switch on WIDTH only (`frontend/CLAUDE.md`), so the layout re-flows
 * live and there is nothing to re-fetch between sizes. The old shape paid a
 * full page load per viewport per scene: 26 tests and 198s, 48% of the whole
 * E2E suite, for one repeated expression.
 *
 * Honest about what it proves: only that the BODY doesn't scroll horizontally.
 * It does not prove content isn't clipped inside a container, and it does not
 * prove the scene "fills the screen" — that would be a different assertion.
 *
 * @example await expectNoHorizontalOverflow(page, VIEWPORTS)
 */
export async function expectNoHorizontalOverflow(
  page: Page,
  viewports: readonly { name: string; width: number; height: number }[],
): Promise<void> {
  for (const vp of viewports) {
    await page.setViewportSize({ width: vp.width, height: vp.height })
    const overflow = await page.evaluate(
      () => document.documentElement.scrollWidth - document.documentElement.clientWidth,
    )
    expect(overflow, `rolagem horizontal @ ${vp.name} (${vp.width}×${vp.height})`).toBeLessThanOrEqual(1)
  }
}

/**
 * Falha no primeiro formato em que a PÁGINA rola verticalmente.
 *
 * Isto é premissa de produto, não estética: a cena de jogo mostra tudo numa
 * tela e o mestre não caça informação rolando no meio do combate. Estava
 * afirmada em mensagem de commit ("verificado: não rola em 1920, 1024 nem
 * 390") e em lugar nenhum que rodasse de novo — a próxima mudança de layout a
 * quebraria em silêncio.
 *
 * Só um browser testemunha: em jsdom `scrollHeight` e `clientHeight` são ambos
 * zero e a asserção passaria verde sobre uma cena de três telas de altura.
 *
 * @example await expectPageDoesNotScroll(page, VIEWPORTS)
 */
export async function expectPageDoesNotScroll(
  page: Page,
  viewports: readonly { name: string; width: number; height: number }[],
): Promise<void> {
  for (const vp of viewports) {
    await page.setViewportSize({ width: vp.width, height: vp.height })
    const overflow = await page.evaluate(
      () => document.documentElement.scrollHeight - document.documentElement.clientHeight,
    )
    expect(overflow, `a página rola @ ${vp.name} (${vp.width}×${vp.height})`).toBeLessThanOrEqual(1)
  }
}
