import type { Page } from '@playwright/test'

/**
 * OS PISOS DO SISTEMA TEMPESTADE (ALE-479).
 *
 * A UI está migrando tela a tela (ALE-497), e o sistema novo subiu dois pisos:
 * texto de no mínimo 12px e alvo de toque de no mínimo 44×44. Os dois valem
 * SÓ dentro de `.scene-ds` — as telas do grimório continuam com os pisos delas
 * até a vez de cada uma, e cobrá-las aqui reprovaria a transição inteira no
 * primeiro dia.
 *
 * O alvo é 44 nos DOIS lados, e não o círculo de 24px do WCAG 2.5.8 que o
 * `touch-targets.ts` mede: o dono decidiu o piso do 2.5.5 (AAA) para o sistema
 * novo. Por isso este medidor é outro, e não um parâmetro daquele — a regra do
 * espaçamento de lá não existe aqui.
 *
 * Os dois devolvem `{falhas, medidos}`: uma cena que não carregou e uma cena
 * perfeita devolvem a mesma lista vazia.
 */

/** O piso de texto do sistema Tempestade, em pixels. */
export const DS_TEXT_FLOOR = 12

/** O piso de alvo no toque, em pixels, nos dois lados. */
export const DS_TOUCH_FLOOR = 44

export type FloorReport = { falhas: string[]; medidos: number; escopos: number }

/** Mede todo texto desenhado dentro de `.scene-ds` e devolve o que está abaixo do piso. */
export async function measureDsText(page: Page): Promise<FloorReport> {
  return page.evaluate((floor) => {
    const scopes = [...document.querySelectorAll('.scene-ds')]
    let measured = 0
    const failures: string[] = []
    for (const scope of scopes) {
      for (const el of [scope, ...scope.querySelectorAll('*')]) {
        const text = [...el.childNodes]
          .filter((n) => n.nodeType === Node.TEXT_NODE)
          .map((n) => n.textContent ?? '')
          .join('')
          .trim()
        if (!text) continue
        const r = el.getBoundingClientRect()
        const cs = getComputedStyle(el)
        if (r.width === 0 || r.height === 0 || cs.visibility === 'hidden') continue
        measured++
        const px = Number.parseFloat(cs.fontSize)
        if (px < floor) failures.push(`${px}px: "${text.slice(0, 32)}"`)
      }
    }
    return { falhas: failures, medidos: measured, escopos: scopes.length }
  }, DS_TEXT_FLOOR)
}

/** Mede todo alvo interativo visível dentro de `.scene-ds`. */
export async function measureDsTouchTargets(page: Page): Promise<FloorReport> {
  return page.evaluate((floor) => {
    const SELECTOR =
      'a[href], button, input:not([type=hidden]), select, textarea, summary, [role=button], [tabindex]:not([tabindex="-1"])'
    const scopes = [...document.querySelectorAll('.scene-ds')]
    let measured = 0
    const failures: string[] = []
    for (const scope of scopes) {
      for (const el of scope.querySelectorAll(SELECTOR)) {
        const r = el.getBoundingClientRect()
        if (r.width === 0 || r.height === 0 || getComputedStyle(el).visibility === 'hidden') continue
        measured++
        // Meio pixel de folga: o arredondamento de `rem` e de borda produz
        // 43,99, e um guarda que reprova arredondamento é desligado na semana
        // seguinte.
        if (r.width + 0.5 < floor || r.height + 0.5 < floor) {
          const name = (el.getAttribute('aria-label') ?? el.textContent ?? '').trim().slice(0, 32)
          failures.push(`<${el.tagName.toLowerCase()}> «${name}» ${Math.round(r.width)}×${Math.round(r.height)}`)
        }
      }
    }
    return { falhas: failures, medidos: measured, escopos: scopes.length }
  }, DS_TOUCH_FLOOR)
}
