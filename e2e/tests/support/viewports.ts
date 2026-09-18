import { type Page, expect } from '@playwright/test'

// As asserções de RELAÇÃO (alinhamento, proporção, containment, alcance) moram
// em `geometry.ts`. Aqui ficam a lista de formatos e as duas asserções de
// TRANSBORDO, que são de outra natureza: negativas, e varrendo os seis formatos
// numa carga de página só.

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
 * Falha no primeiro formato em que algo ACIMA do rolador transborda.
 *
 * "A PÁGINA não cresce" é a asserção que se escreve sozinha aqui, e nesta casa
 * ela é INERTE: a casca é `h-dvh overflow-hidden`, então o documento tem a
 * altura da janela aconteça o que acontecer por dentro. Medido arrancando o
 * `overflow-y-auto` do rolador dos catálogos: 25.187px de lista dentro de um
 * elemento de 756px, e `documentElement.scrollHeight` ficou em 900 — a mesma
 * altura da janela (ALE-332).
 *
 * Quem cresce é o ANCESTRAL, e é ele que esta varredura lê: da mãe do rolador
 * até o `<html>`, nenhum elemento pode ter mais conteúdo do que caixa. Rolagem
 * é privilégio do rolador; qualquer outro elemento rolando é a lista vazando
 * para fora do cartão (ALE-149).
 *
 * O CONTROLE vem de graça e é obrigatório: o rolador tem de estar ROLANDO. Uma
 * cena que desenhou pouco não transborda em lugar nenhum, e aí a varredura dos
 * ancestrais fica verde sobre nada — que é a mesma cor de "não mediu".
 *
 * @example await expectOnlyTheScrollerScrolls(page, '#catalogs [role="region"]', VIEWPORTS)
 */
export async function expectOnlyTheScrollerScrolls(
  page: Page,
  scroller: string,
  viewports: readonly { name: string; width: number; height: number }[],
): Promise<void> {
  for (const vp of viewports) {
    await page.setViewportSize({ width: vp.width, height: vp.height })
    const medida = await page.evaluate((seletor) => {
      const rolador = document.querySelector(seletor)
      if (!rolador) return null
      const transbordando: string[] = []
      for (let n = rolador.parentElement; n; n = n.parentElement) {
        const sobra = n.scrollHeight - n.clientHeight
        if (sobra > 2) {
          // A CLASSE vai na mensagem porque `div` não endereça nada: o
          // primeiro vermelho deste instrumento foi um `lg:flex-row` da casca
          // do mestre, e sem as classes a falha dizia "procure" em vez de
          // "conserte isto".
          transbordando.push(
            `${n.tagName.toLowerCase()}${n.id ? `#${n.id}` : ''} tem ${n.scrollHeight}px de conteúdo em ${n.clientHeight}px de caixa — class="${n.className}"`,
          )
        }
      }
      return { rola: rolador.scrollHeight - rolador.clientHeight, transbordando }
    }, scroller)

    expect(medida, `nenhum "${scroller}" na cena @ ${vp.name} (${vp.width}×${vp.height})`).not.toBeNull()
    const { rola, transbordando } = medida as { rola: number; transbordando: string[] }
    expect(
      rola,
      `o rolador não está rolando @ ${vp.name} (${vp.width}×${vp.height}) — sem conteúdo sobrando não há transbordo possível, e a varredura dos ancestrais não mede nada`,
    ).toBeGreaterThan(0)
    expect(
      transbordando,
      `algo acima do rolador transbordou @ ${vp.name} (${vp.width}×${vp.height}) — a lista está vazando para fora do cartão (ALE-149)`,
    ).toEqual([])
  }
}
