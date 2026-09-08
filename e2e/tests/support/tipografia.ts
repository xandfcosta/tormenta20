import { expect, type Page } from '@playwright/test'

/**
 * O MEDIDOR DE TIPOGRAFIA da casa: a Cinzel não desce abaixo de 14px (ALE-173).
 *
 * Ela é serifada de display — contraste de traço alto e olhos pequenos —, e em
 * 11px maiúscula com espaçamento largo vira desenho antes de virar texto. O dono
 * apontou isso olhando a folha de especificação, e a casa já tinha tomado a
 * mesma decisão um degrau abaixo: o rótulo de campo, em 10px, nunca usou Cinzel.
 *
 * # Ele mora aqui pelo mesmo motivo que o `contraste.ts` (ALE-252)
 *
 * A versão anterior era função INLINE dentro de um `test()` do
 * `grimorio.spec.ts`, e visitava um endereço só: `/grimorio`. Instrumento que
 * mora dentro de um chamador tem exatamente um chamador, e isso não aparece em
 * revisão de diff nenhuma — foi assim que quatro violações minhas viveram em
 * três cenas com o guarda no ar o tempo todo.
 *
 * O guarda da folha funcionava por AMOSTRAGEM: toda tipografia da SPA passava
 * pelos componentes de verdade, então vigiar uma tela vigiava 43. Cena em templ
 * escreve a classe à mão, e a amostragem virou enumeração — uma entrada por
 * cena, para sempre. Este módulo é a metade barata do conserto; a outra metade,
 * que RESTAURA a amostragem, é as cenas passarem pelos componentes da casa.
 *
 * Por que browser: a face resolvida só existe num navegador. Em jsdom o
 * `font-family` devolve a string do CSS — a lista inteira de fallbacks —, e não
 * a fonte que o navegador de fato usou.
 */

/** Uma medição: o que reprovou, e QUANTOS textos em Cinzel foram olhados. */
export type MedicaoDeTipografia = { falhas: string[]; medidos: number }

/** O piso de leitura da Cinzel, em pixels (ALE-173). */
export const PISO_DA_CINZEL = 14

/**
 * Mede todo texto desenhado em Cinzel e reprova o que estiver abaixo do piso.
 *
 * O `medidos` existe para o CONTROLE de quem chama, e aqui ele é mais do que
 * zelo: o filtro é por FONTE, então uma página onde a Cinzel não carregou não
 * tem nenhum candidato — e "nenhuma Cinzel pequena" e "nenhuma Cinzel" produzem
 * a mesma lista vazia. A versão inline deste medidor não devolvia denominador
 * nenhum, e passaria verde com a fonte fora do ar.
 *
 * Ele olha o TEXTO PRÓPRIO de cada nó, e não o `textContent`: o tamanho é
 * herdado, então um `<section>` em Cinzel 24px que contém um rótulo de 11px
 * seria acusado no lugar do rótulo, com o texto do filho na mensagem.
 */
export async function medeATipografia(page: Page): Promise<MedicaoDeTipografia> {
  return page.evaluate((piso) => {
    const olhados: string[] = []
    const falhas = [...document.querySelectorAll('*')]
      .map((el) => {
        const cs = getComputedStyle(el)
        if (!cs.fontFamily.startsWith('Cinzel')) return null
        if (cs.visibility === 'hidden' || cs.display === 'none') return null
        const texto = [...el.childNodes]
          .filter((n) => n.nodeType === 3)
          .map((n) => n.textContent?.trim() ?? '')
          .join('')
          .trim()
        if (!texto) return null
        // O MONOGRAMA DO AVATAR não é texto, é marca — decisão do dono
        // (2026-09-08). O piso existe porque a Cinzel pequena "vira desenho
        // antes de virar texto", e num monograma virar desenho é o objetivo:
        // são duas letras dentro de um círculo colorido (ver `ui.Monogram`),
        // com o nome escrito por extenso ao lado em texto de verdade.
        //
        // A exceção é a CLASSE e não uma heurística, e essa escolha é o que a
        // separa de afrouxar o guarda: `.monograma` é a receita da casa, então
        // quem escreve um monograma novo herda a isenção junto com o desenho, e
        // quem escrever Cinzel pequena em qualquer outro lugar continua sendo
        // pego. Um `aria-hidden` no lugar dela isentaria toda Cinzel decorativa,
        // que é largo demais.
        if (el.closest('.monograma')) return null
        const px = Number.parseFloat(cs.fontSize)
        olhados.push(texto)
        return px < piso ? `${Math.round(px)}px: "${texto.slice(0, 32)}"` : null
      })
      .filter((x): x is string => x !== null)
    return { falhas, medidos: olhados.length }
  }, PISO_DA_CINZEL)
}

/**
 * A ASSERÇÃO COMPLETA, e ela existe para que o denominador não seja opcional.
 *
 * Quem chama o medidor cru pode afirmar `falhas` e esquecer `medidos` — e uma
 * lista de falhas vazia é indistinguível de "o filtro não achou nada", que é
 * como este guarda passaria verde com a Cinzel fora do ar. Embrulhando as duas
 * numa chamada só, o controle vem junto de graça e não há caminho para
 * esquecê-lo.
 *
 * O `onde` entra na mensagem porque estes casos são uma ENUMERAÇÃO: uma linha
 * por cena, e a que alguém esquecer nasce sem medição. Quando a falha aparecer,
 * ela precisa dizer em qual cena — senão a busca começa do zero.
 */
export async function expectCinzelAcimaDoPiso(page: Page, onde: string): Promise<void> {
  const { falhas, medidos } = await medeATipografia(page)
  expect(
    medidos,
    `${onde}: o medidor não achou NENHUM texto em Cinzel — ou a fonte não carregou, ou o filtro parou de casar, e a asserção seguinte não seria evidência de nada`,
  ).toBeGreaterThan(0)
  expect(falhas, `Cinzel abaixo do piso de ${PISO_DA_CINZEL}px ${onde}`).toEqual([])
}
