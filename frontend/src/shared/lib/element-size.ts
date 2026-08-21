import { type Accessor, createEffect, createSignal, onCleanup } from 'solid-js'

export type ElementSize = { width: number; height: number }

/**
 * O tamanho em px de um elemento, acompanhado por `ResizeObserver`.
 *
 * Existe porque nem toda decisão de arranjo cabe no CSS. Duas situações:
 *
 * 1. Quando o número de colunas muda os DADOS — e muda, numa lista
 *    virtualizada, que agrupa as entradas antes de renderizá-las (ALE-170) — a
 *    consulta de contêiner não alcança: ela pinta, não conta.
 * 2. Quando a decisão atravessa um PORTAL. Um diálogo do Kobalte nasce no
 *    `body`, fora do palco, e consulta de contêiner só vale para descendentes
 *    (ALE-146).
 *
 * Mede o elemento, nunca a janela. O painel de ferramentas do /gm divide a tela
 * com a trilha lateral, então largura de janela mente por centenas de pixels —
 * é o mesmo defeito que a ALE-172 consertou trocando `lg:` por consulta de
 * contêiner.
 *
 * Devolve zeros onde não há `ResizeObserver` nem leiaute — jsdom é os dois — e
 * quem consome trata zero como "ainda não sei".
 *
 * @example const tamanho = createElementSize(alvo)
 * @example const baixo = () => tamanho().height > 0 && tamanho().height < 416
 */
export function createElementSize(
  alvo: Accessor<HTMLElement | undefined>,
): Accessor<ElementSize> {
  const [tamanho, setTamanho] = createSignal<ElementSize>({ width: 0, height: 0 })

  createEffect(() => {
    const elemento = alvo()
    if (!elemento) return

    const caixa = elemento.getBoundingClientRect()
    setTamanho({ width: caixa.width, height: caixa.height })
    if (typeof ResizeObserver === 'undefined') return

    const observador = new ResizeObserver(([entrada]) => {
      if (entrada) {
        setTamanho({
          width: entrada.contentRect.width,
          height: entrada.contentRect.height,
        })
      }
    })
    observador.observe(elemento)
    onCleanup(() => observador.disconnect())
  })

  return tamanho
}

/** Só a largura — a metade que os catálogos usam para contar colunas. */
export function createElementWidth(alvo: Accessor<HTMLElement | undefined>): Accessor<number> {
  const tamanho = createElementSize(alvo)
  return () => tamanho().width
}
