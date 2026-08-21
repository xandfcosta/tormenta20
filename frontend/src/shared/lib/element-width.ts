import { type Accessor, createEffect, createSignal, onCleanup } from 'solid-js'

/**
 * A largura em px de um elemento, acompanhada por `ResizeObserver`.
 *
 * Existe porque nem toda decisão de arranjo cabe no CSS. Quando o número de
 * colunas muda os DADOS — e muda, numa lista virtualizada, que agrupa as
 * entradas antes de renderizá-las (ALE-170) — a consulta de contêiner não
 * alcança: ela pinta, não conta. Aí é preciso o número.
 *
 * Mede o CONTÊINER, nunca a janela. O painel de ferramentas do /gm divide a
 * tela com a trilha lateral, então largura de janela mente por centenas de
 * pixels — é o mesmo defeito que a ALE-172 consertou trocando `lg:` por
 * consulta de contêiner.
 *
 * Devolve 0 onde não há `ResizeObserver` nem leiaute — jsdom é os dois — e
 * quem consome trata 0 como "ainda não sei", caindo no arranjo mais estreito.
 *
 * @example const largura = createElementWidth(alvo)
 * @example const colunas = () => (largura() > 900 ? 2 : 1)
 */
export function createElementWidth(alvo: Accessor<HTMLElement | undefined>): Accessor<number> {
  const [largura, setLargura] = createSignal(0)

  createEffect(() => {
    const elemento = alvo()
    if (!elemento) return

    setLargura(elemento.getBoundingClientRect().width)
    if (typeof ResizeObserver === 'undefined') return

    const observador = new ResizeObserver(([entrada]) => {
      if (entrada) setLargura(entrada.contentRect.width)
    })
    observador.observe(elemento)
    onCleanup(() => observador.disconnect())
  })

  return largura
}
