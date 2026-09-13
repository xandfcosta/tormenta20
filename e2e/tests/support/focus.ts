import { expect, type Page } from '@playwright/test'

/**
 * O MEDIDOR DO ANEL DE FOCO da casa: uma receita só, em toda cena (ALE-318).
 *
 * A regra é global e mora no `index.css` desde a ALE-173 (P4) — `2px solid
 * var(--grimorio-gold)` a `outline-offset: 1px`, sobre tudo que recebe foco
 * dentro de uma cena. O offset é 1px por medição: a ALE-150 trocou `box-shadow`
 * por `outline` porque o contêiner que rola cortava o anel, e offset grande
 * sofre o mesmo corte.
 *
 * # Ele mora aqui porque o anterior media UMA página
 *
 * A versão anterior era um `test()` dentro do `grimorio.spec.ts` que abria
 * `/grimorio` e mais nada — a folha de especificação, onde tudo passa pelo kit
 * por construção. É a família da ALE-237, da ALE-252 e da ALE-272 pela terceira
 * vez: **instrumento que mora dentro de um chamador tem exatamente um
 * chamador**, e a cobertura é função de onde o guarda CHEGA.
 *
 * # A TRANSIÇÃO é medida como transição, e não adivinhada por um INSTANTE
 *
 * O guarda anterior injetava `transition: none` antes de medir, com o
 * comentário explicando que as peças do kit têm `transition-all` e o contorno
 * "vai mudando de alfa no caminho". O diagnóstico estava certo, e o preço de
 * ele morar num comentário de teste apareceu na ALE-316: uma medição nova da
 * ficha leu o computado no instante do foco, achou TRÊS anéis diferentes em 131
 * botões, e virou issue de alta prioridade. Os três eram o MESMO anel em três
 * instantes do trajeto.
 *
 * Escolher um instante não resolve, e as duas pontas foram medidas: ler na
 * mesma tarefa do `focus()` devolve o valor de PARTIDA de toda propriedade em
 * transição — um botão do kit responde `3px solid off:0px`, que ele nunca
 * pinta —, e ler no quadro seguinte ainda pega a primeira amostra de uma
 * transição de 150ms. Todo corte no tempo produz lista de falhas com cara de
 * descoberta.
 *
 * O que este medidor faz é PERGUNTAR ao navegador se existe transição:
 * `getAnimations()` devolve uma `CSSTransition` por propriedade, com o nome
 * dela. **Anel que anima é a falha**, dita com o nome da propriedade; e as
 * transições de contorno são FINALIZADAS antes da leitura da aparência, para
 * que a divergência de cara e a de tempo não se contaminem.
 *
 * O conserto da ALE-318 foi tirar o trajeto: o repouso do `outline-color` era
 * ouro a 50% (o `* { outline-ring/50 }` que o shadcn traz), então havia o que
 * interpolar. Medido com o repouso devolvido ao que era, este medidor acusa
 * `outline-color` em 29 de 40 focáveis da aba de Perícias — o controle positivo
 * que torna o silêncio dele evidência.
 *
 * # O que ele mede que o anterior não media
 *
 *   - **O nó SEM anel nenhum.** O anterior tinha `if (outlineStyle === 'none')
 *     continue` — um botão que não recebe realce era PULADO, e é o pior caso
 *     desta família: quem navega por teclado perde o lugar (WCAG 2.4.7).
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
 * As TRÊS CHAVES seguem a grafia dos outros medidores da casa
 * (`support/contrast.ts`, `support/typography.ts`, `support/touch-targets.ts`),
 * que é a que o `CLAUDE.md` da raiz nomeia ao descrever o denominador
 * obrigatório. Os identificadores deste arquivo nascem em inglês, como a regra
 * de idioma manda; renomear só estas três aqui forkaria a convenção em duas
 * grafias, que é pior — trocá-las é varredura das 56 ocorrências em dez
 * arquivos, e essa é decisão própria.
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
      // `index.css`). Ele não fica de fora da medição: o que se cobra dele é o
      // brilho, senão "sem anel" e "sem realce nenhum" ficariam iguais.
      //
      // A condição COPIA o seletor do `index.css` em vez de aproximá-lo por
      // "está dentro de um trilho", e a diferença não é teórica: a primeira
      // versão dizia só `closest('[data-nav-region]')` e acusou 37 nós na folha
      // da forja — os rádios `sr-only` das cartas e o próprio contêiner
      // rolável, que estão dentro do trilho, não são item dele e recebem o anel
      // normal da casa. Guarda que aproxima um seletor mede outro seletor.
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
export async function expectOneFocusRing(page: Page, onde: string, piso = 1): Promise<void> {
  const { falhas, medidos } = await measureFocusRing(page)
  expect(
    medidos,
    `${onde}: o medidor não conseguiu focar NENHUM elemento — ou a cena não desenhou, ou o seletor parou de casar, e a asserção seguinte não seria evidência de nada`,
  ).toBeGreaterThanOrEqual(piso)
  expect(falhas, `o realce de foco ${onde}`).toEqual([])
}
