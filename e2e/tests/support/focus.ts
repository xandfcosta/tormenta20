import { expect, type Page } from '@playwright/test'

/**
 * O MEDIDOR DO ANEL DE FOCO da casa: uma receita só, em toda cena.
 *
 * A regra é global e mora no `index.css` — `2px solid var(--grimorio-gold)` a
 * `outline-offset: 1px`, sobre tudo que recebe foco dentro de uma cena. O offset
 * é 1px por medição: o anel é `outline` e não `box-shadow` porque o contêiner que
 * rola corta o realce, e offset grande sofre o mesmo corte.
 *
 * Ele mora AQUI e não dentro de um spec: instrumento que mora dentro de um
 * chamador tem exatamente um chamador, e cobertura é função de onde o guarda
 * CHEGA.
 *
 * # A TRANSIÇÃO é medida como transição, e não adivinhada por um INSTANTE
 *
 * Escolher um instante não resolve, e as duas pontas foram medidas: ler na
 * mesma tarefa do `focus()` devolve o valor de PARTIDA de toda propriedade em
 * transição — um botão do kit responde `3px solid off:0px`, que ele nunca
 * pinta —, e ler no quadro seguinte ainda pega a primeira amostra de uma
 * transição de 150ms. Todo corte no tempo produz lista de falhas com cara de
 * descoberta: uma medição assim já acusou TRÊS anéis diferentes em 131 botões,
 * e os três eram o MESMO anel em três instantes do trajeto.
 *
 * O que este medidor faz é PERGUNTAR ao navegador se existe transição:
 * `getAnimations()` devolve uma `CSSTransition` por propriedade, com o nome
 * dela. **Anel que anima é a falha**, dita com o nome da propriedade; e as
 * transições de contorno são FINALIZADAS antes da leitura da aparência, para
 * que a divergência de cara e a de tempo não se contaminem.
 *
 * O CONTROLE POSITIVO conhecido: devolvendo o repouso do `outline-color` ao ouro
 * a 50% que o shadcn traz (`* { outline-ring/50 }`), este medidor acusa
 * `outline-color` em 29 de 40 focáveis da aba de Perícias. É isso que torna o
 * silêncio dele evidência.
 *
 * # Os três pontos cegos que ele fecha
 *
 *   - **O nó SEM anel nenhum**, que um `if (outlineStyle === 'none') continue`
 *     pularia — o pior caso da família: quem navega por teclado perde o lugar
 *     (WCAG 2.4.7).
 *   - **O anel que mora no ANCESTRAL.** A carta de rádio da forja e a de entrar
 *     na mesa escondem o `<input>` com `sr-only` e desenham o realce no
 *     `<label>` com `has-[:focus-visible]:outline-*`. Um sweep de focáveis
 *     nunca olha para lá.
 *   - **O DENOMINADOR.** Uma lista de falhas vazia e um seletor que não casa
 *     com nada se parecem no terminal.
 */

/**
 * Uma medição: o que reprovou, quantas paradas de foco foram olhadas, e as caras
 * achadas.
 *
 * As TRÊS CHAVES ficam em português, contra a regra de idioma, porque é a
 * grafia dos outros medidores da casa (`contrast.ts`, `typography.ts`,
 * `touch-targets.ts`): renomear só estas forkaria a convenção em duas grafias,
 * que é pior. Trocá-las é varredura, e essa é decisão própria.
 */
export type FocusRingMeasurement = { falhas: string[]; medidos: number; caras: string[] }

/**
 * Percorre todo focável VISÍVEL da página, foca cada um e lê o anel pintado —
 * no próprio nó e nos ancestrais dele, porque `:has()` põe o realce no pai.
 *
 * O primeiro `Tab` é de VERDADE e serve para ligar o modo teclado do navegador:
 * a partir dele o foco programático também casa `:focus-visible`, e dá para
 * varrer a cena inteira em vez das primeiras paradas. Sem ele a sonda mediria
 * "sem contorno" em toda parte e passaria verde sobre qualquer coisa — foi
 * assim que a primeira versão do guarda anterior passou sobre uma sabotagem.
 *
 * O laço roda INTEIRO dentro de um `page.evaluate` síncrono, e isso não é
 * economia de ida e volta: um `data-on:focus` de cena pede ao servidor, e o
 * remendo do Datastar chega assíncrono. Sem ceder o laço, nenhum morph acontece
 * no meio da varredura e as referências não se soltam debaixo dela.
 */
export async function measureFocusRing(page: Page): Promise<FocusRingMeasurement> {
  await page.keyboard.press('Tab')
  return page.evaluate(() => {
    const failures: string[] = []
    // A cara → um EXEMPLO dela. Guarda de varredura falha com o nome do caso,
    // que é a diferença entre "conserte isto" e "procure".
    const looks = new Map<string, string>()
    let measured = 0

    const isVisible = (e: Element) =>
      !!(e as HTMLElement).offsetWidth || !!(e as HTMLElement).offsetHeight || e.getClientRects().length > 0
    const describe = (e: Element) =>
      `<${e.tagName.toLowerCase()}> «${(e.getAttribute('aria-label') || e.textContent || '').trim().slice(0, 32)}»`
    const ringOf = (cs: CSSStyleDeclaration) =>
      `${cs.outlineWidth} ${cs.outlineStyle} ${cs.outlineColor} off:${cs.outlineOffset}`
    const isPainted = (cs: CSSStyleDeclaration) => cs.outlineStyle !== 'none' && cs.outlineWidth !== '0px'

    // A transição do CONTORNO, e só dela: `opacity` e `background-color` animam
    // de propósito em várias peças, e o que a casa promete instantâneo é o anel.
    const outlineTransitionsOf = (e: Element) =>
      e.getAnimations().filter((a) => (a as CSSTransition).transitionProperty?.startsWith('outline'))

    const targets = [
      ...document.querySelectorAll('a[href], button, input, select, textarea, summary, [tabindex]:not([tabindex="-1"])'),
    ].filter((e) => isVisible(e) && !(e as HTMLInputElement).disabled)

    for (const target of targets) {
      ;(target as HTMLElement).focus()
      if (document.activeElement !== target) continue
      measured++

      // O CURSOR DE NAVEGAÇÃO é outro estado — "você está pilotando por aqui" —
      // e diz isso com brilho em vez de contorno (`[data-nav-region]` no
      // `index.css`). Ele não fica de fora: o que se cobra dele é o brilho,
      // senão "sem anel" e "sem realce nenhum" ficariam iguais.
      //
      // A condição COPIA o seletor do `index.css` em vez de aproximá-lo por
      // "está dentro de um trilho", e a diferença não é teórica: só
      // `closest('[data-nav-region]')` acusava 37 nós na folha da forja — os
      // rádios `sr-only` das cartas e o contêiner rolável estão dentro do trilho
      // mas não são item dele, e recebem o anel normal. Guarda que aproxima um
      // seletor mede outro seletor.
      if (target.closest('[data-nav-region]') && target.matches('a, button, [data-nav-item]')) {
        if (getComputedStyle(target).boxShadow === 'none') {
          failures.push(`${describe(target)} está num trilho de navegação e não acende nem anel nem brilho`)
        }
        continue
      }

      // O CONTROLE que torna o resto evidência: sem `:focus-visible` casado, o
      // computado é o de repouso e a leitura não diz nada sobre realce.
      if (!target.matches(':focus-visible')) {
        failures.push(`${describe(target)} recebeu foco e NÃO casa :focus-visible — a medição abaixo não seria sobre realce`)
        continue
      }

      // O anel pode morar no nó ou num ANCESTRAL (`has-[:focus-visible]`).
      // Subir a árvore é o que enxerga a carta de rádio, cujo `<input>` é
      // `sr-only` e cujo realce é do `<label>`.
      let found = false
      for (let node: Element | null = target; node && node !== document.documentElement; node = node.parentElement) {
        for (const a of outlineTransitionsOf(node)) {
          const prop = (a as CSSTransition).transitionProperty
          const ms = Math.round(Number(a.effect?.getComputedTiming().duration ?? 0))
          failures.push(
            `${describe(node)} ACENDE o anel em vez de desenhá-lo: ${prop} em transição de ${ms}ms` +
              (node === target ? '' : ` (ancestral de ${describe(target)})`),
          )
          // Finalizada para que a leitura de aparência abaixo seja sobre a CARA
          // e não sobre o instante: duas falhas diferentes não devem se misturar.
          a.finish()
        }
        const cs = getComputedStyle(node)
        if (!isPainted(cs)) continue
        found = true
        const look = ringOf(cs)
        if (!looks.has(look)) looks.set(look, node === target ? describe(target) : `${describe(node)} (ancestral de ${describe(target)})`)
      }
      if (!found) {
        failures.push(`${describe(target)} recebeu foco e não desenhou anel nenhum (WCAG 2.4.7)`)
      }
    }

    if (looks.size > 1) {
      failures.push(
        `o foco tem ${looks.size} caras e a casa promete UMA:\n` +
          [...looks].map(([look, example]) => `        ${look}  —  ${example}`).join('\n'),
      )
    }
    return { falhas: failures, medidos: measured, caras: [...looks.keys()] }
  })
}

/**
 * A ASSERÇÃO COMPLETA, e ela existe para que o denominador não seja opcional.
 *
 * Quem chama o medidor cru pode afirmar `falhas` e esquecer `medidos` — e é
 * assim que este guarda passaria verde numa cena que deixou de renderizar. O
 * `onde` entra na mensagem porque os chamadores são uma ENUMERAÇÃO de cenas: a
 * falha precisa dizer em qual, senão a busca começa do zero.
 */
export async function expectOneFocusRing(page: Page, where: string, floor = 1): Promise<void> {
  const { falhas: failures, medidos: measured } = await measureFocusRing(page)
  expect(
    measured,
    `${where}: o medidor não conseguiu focar NENHUM elemento — ou a cena não desenhou, ou o seletor parou de casar, e a asserção seguinte não seria evidência de nada`,
  ).toBeGreaterThanOrEqual(floor)
  expect(failures, `o realce de foco ${where}`).toEqual([])
}
