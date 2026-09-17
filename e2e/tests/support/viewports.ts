import { type Page, expect } from '@playwright/test'

// As asserções de RELAÇÃO (alinhamento, proporção, containment, alcance) moram
// em `geometry.ts`. Aqui ficam a lista de formatos e as duas asserções de
// DOCUMENTO, que são de outra natureza: globais e negativas.

/** Os seis formatos que toda cena da casa tem de sobreviver. */
export const VIEWPORTS = [
  { name: 'desktop', width: 1920, height: 1080 },
  { name: 'laptop', width: 1440, height: 900 },
  { name: 'tablet-landscape', width: 1024, height: 768 },
  { name: 'tablet-portrait', width: 768, height: 1024 },
  { name: 'mobile-landscape', width: 844, height: 390 },
  { name: 'mobile-portrait', width: 390, height: 844 },
] as const

/** Do laptop para cima — onde a camada de teclado responde (`≥xl` + `pointer: fine`). */
export const DESK_VIEWPORTS = VIEWPORTS.filter((v) => v.width >= 1280)

/**
 * Redimensiona por todos os formatos na página ATUAL e falha no primeiro em que
 * o documento rola para o lado.
 *
 * Um `goto` e seis redimensionamentos, e não seis navegações: as consultas de
 * mídia da casa chaveiam só por LARGURA, então o leiaute se refaz ao vivo e não
 * há o que rebuscar entre os tamanhos. Uma carga de página por formato por cena
 * custava 198s — 48% da suíte e2e inteira — por uma expressão repetida.
 *
 * Honesto sobre o que prova: só que o BODY não rola na horizontal. Ele não prova
 * que o conteúdo não é recortado dentro de um contêiner, nem que a cena "preenche
 * a tela" — isso seria outra asserção.
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
 * tela e o mestre não caça informação rolando no meio do combate.
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
