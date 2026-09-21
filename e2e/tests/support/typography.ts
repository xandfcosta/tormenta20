import { expect, type Page } from '@playwright/test'

/**
 * O MEDIDOR DE TIPOGRAFIA da casa: a Cinzel não desce abaixo de 14px.
 *
 * Ela é serifada de display — contraste de traço alto e olhos pequenos —, e em
 * 11px maiúscula com espaçamento largo vira desenho antes de virar texto. A casa
 * já tinha a mesma decisão um degrau abaixo: o rótulo de campo, em 10px, nunca
 * usou Cinzel.
 *
 * # Ele mora AQUI, e não dentro de um `test()`
 *
 * Instrumento que mora dentro de um chamador tem exatamente um chamador, e isso
 * não aparece em revisão de diff nenhuma: a versão inline deste medidor visitava
 * um endereço só, e quatro violações viveram em três cenas com o guarda no ar.
 *
 * E a cobertura hoje é ENUMERAÇÃO, não amostragem — cena em templ escreve a
 * classe à mão, então é uma entrada por cena, para sempre. Este módulo é a
 * metade barata do conserto; a que RESTAURA a amostragem é as cenas passarem
 * pelos componentes da casa.
 *
 * Por que browser: a face resolvida só existe num navegador. Em jsdom o
 * `font-family` devolve a string do CSS — a lista inteira de fallbacks —, e não
 * a fonte que o navegador de fato usou.
 */

/** Uma medição: o que reprovou, e QUANTOS textos em Cinzel foram olhados. */
export type MedicaoDeTipografia = { falhas: string[]; medidos: number }

/** O piso de leitura da Cinzel, em pixels. */
export const PISO_DA_CINZEL = 14

/**
 * Mede todo texto desenhado em Cinzel e reprova o que estiver abaixo do piso.
 *
 * O `medidos` existe para o CONTROLE de quem chama, e aqui ele é mais do que
 * zelo: o filtro é por FONTE, então uma página onde a Cinzel não carregou não
 * tem nenhum candidato — e "nenhuma Cinzel pequena" e "nenhuma Cinzel" produzem
 * a mesma lista vazia.
 *
 * Ele olha o TEXTO PRÓPRIO de cada nó, e não o `textContent`: o tamanho é
 * herdado, então um `<section>` em Cinzel 24px que contém um rótulo de 11px
 * seria acusado no lugar do rótulo, com o texto do filho na mensagem.
 */
export async function medeATipografia(page: Page): Promise<MedicaoDeTipografia> {
  return page.evaluate((floor) => {
    const checkedTags: string[] = []
    const failures = [...document.querySelectorAll('*')]
      .map((el) => {
        const cs = getComputedStyle(el)
        if (!cs.fontFamily.startsWith('Cinzel')) return null
        if (cs.visibility === 'hidden' || cs.display === 'none') return null
        const text = [...el.childNodes]
          .filter((n) => n.nodeType === 3)
          .map((n) => n.textContent?.trim() ?? '')
          .join('')
          .trim()
        if (!text) return null
        // O MONOGRAMA DO AVATAR não é texto, é marca (decisão do dono): o piso
        // existe porque a Cinzel pequena "vira desenho antes de virar texto", e
        // num monogram virar desenho é o objetivo — o nome por extenso está ao
        // lado, em texto de verdade.
        //
        // A exceção é a CLASSE e não uma heurística, e é isso que a separa de
        // afrouxar o guarda: quem escreve um monogram novo herda a isenção
        // junto com o desenho, e Cinzel pequena em qualquer outro lugar continua
        // sendo pega. Um `aria-hidden` isentaria toda Cinzel decorativa, que é
        // largo demais.
        if (el.closest('.monogram')) return null
        const px = Number.parseFloat(cs.fontSize)
        checkedTags.push(text)
        return px < floor ? `${Math.round(px)}px: "${text.slice(0, 32)}"` : null
      })
      .filter((x): x is string => x !== null)
    return { falhas: failures, medidos: checkedTags.length }
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
export async function expectCinzelAcimaDoPiso(page: Page, at: string): Promise<void> {
  const { falhas: failures, medidos: measured } = await medeATipografia(page)
  expect(
    measured,
    `${at}: o medidor não achou NENHUM texto em Cinzel — ou a fonte não carregou, ou o filtro parou de casar, e a asserção seguinte não seria evidência de nada`,
  ).toBeGreaterThan(0)
  expect(failures, `Cinzel abaixo do piso de ${PISO_DA_CINZEL}px ${at}`).toEqual([])
}
