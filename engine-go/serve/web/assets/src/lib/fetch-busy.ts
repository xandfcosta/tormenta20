/**
 * O CONTROLE DIZ QUE ESTÁ TRABALHANDO enquanto o pedido dele não volta
 * (ALE-520).
 *
 * O servidor responde em ~30ms, mas o toque até a tela mudar leva 75–160ms num
 * celular, e nesse meio o botão não mudava nada — parecia que o toque não
 * tinha pegado. O Datastar avisa o documento a cada pedido (`datastar-fetch`,
 * com o tipo e o elemento que disparou), e aqui o elemento ganha
 * `aria-busy="true"` até a resposta; o CSS do kit desenha o resto.
 *
 * SÓ CONTROLES, e não qualquer elemento: o fluxo da página também é um pedido
 * do Datastar, pendurado num nó que não é botão, e ele NÃO TERMINA — um
 * `aria-busy` eterno ali faria o leitor de tela tratar a página inteira como
 * "carregando".
 *
 * A CONTA por elemento existe porque o −1 se toca em sequência: dois pedidos
 * do mesmo botão em curso, e o primeiro a voltar não pode apagar a marca do
 * segundo.
 */
const CONTROLS = 'button, a, input, select, textarea, [role="switch"]'

type FetchDetail = { type: string; el: unknown }

export function attachFetchBusy(doc: Document): void {
  const inFlight = new WeakMap<Element, number>()
  doc.addEventListener('datastar-fetch', (event) => {
    const { type, el } = (event as CustomEvent<FetchDetail>).detail
    if (!(el instanceof Element) || !el.matches(CONTROLS)) return
    const pending = inFlight.get(el) ?? 0
    if (type === 'started') {
      inFlight.set(el, pending + 1)
      el.setAttribute('aria-busy', 'true')
      return
    }
    if (type !== 'finished') return
    const left = Math.max(0, pending - 1)
    inFlight.set(el, left)
    if (left === 0) el.removeAttribute('aria-busy')
  })
}
