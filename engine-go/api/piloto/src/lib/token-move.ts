import { podeAnimar } from '@/lib/turn-juice'

/**
 * A PEÇA QUE MOVEU — o terceiro disparo da ALE-174, e o único que ainda faltava.
 *
 * Confirmar um movimento no tabuleiro era TELEPORTE: a peça sumia de uma casa e
 * reaparecia em outra, e a mesa inteira está olhando para o mapa justamente
 * nessa hora. O `PendingPath` já cobre a PROPOSTA (a seta viva durante o
 * arrasto); isto cobre a confirmação.
 *
 * # Por que FLIP, e por que não uma transição de CSS
 *
 * A peça é posicionada por `left`/`top` calculados de `--col`/`--lin`
 * (`piloto.src.css:982`). A issue previa o perigo de transicioná-los e nomeou a
 * causa errada: ela dizia que o PAN também os muda. Não muda — o pan é
 * `transform` do contêiner da cena (`--vista-x`), e a peça não sabe dele.
 *
 * **Quem os muda é o ZOOM.** O `--quadrado` entra no `calc` de toda peça, então
 * uma `transition: left, top` faria as nove escorregarem ao aproximar, que é
 * pior que o teleporte: um movimento que ninguém fez.
 *
 * O FLIP é imune por CONSTRUÇÃO, e não por cuidado: ele dispara na mudança de
 * `--col`/`--lin`, e o zoom não os toca. Não há o que lembrar de não fazer.
 *
 * # O gatilho é a MUTAÇÃO, não o nó
 *
 * Medido no navegador: quando o servidor remenda a Mesa, o `<button>` da peça
 * SOBREVIVE e o `style` dele é reescrito no lugar — a mutação chega com
 * `oldValue` (`--col:3` → `--col:7`) sobre o mesmo alvo. É disso que o delta
 * sai, e é por isso que nada aqui depende de casar peça velha com peça nova.
 *
 * Se um dia o morph passar a TROCAR o nó, não haverá mutação de `style` com
 * valor antigo e a peça simplesmente aparecerá no destino — degradação para o
 * comportamento de hoje, nunca para uma animação errada.
 *
 * # O que NÃO tem teste unitário, e por quê
 *
 * A conta do delta é coberta pelo e2e e não por um spec de unidade: este pacote
 * não tem runner de JavaScript desde que a SPA saiu, e somar um por causa de
 * duas multiplicações seria pagar ferramenta nova pelo que a faixa de cima já
 * segura. A linha do tempo da animação, que é o que importa aqui, só o navegador
 * tem de qualquer forma.
 */

/** A casa em que a peça está, lida de um `style` cru. Nulo quando não dá. */
export function squareFromStyle(estilo: string | null): { col: number; lin: number } | null {
  if (!estilo) return null
  const col = /--col:\s*(-?\d+)/.exec(estilo)
  const lin = /--lin:\s*(-?\d+)/.exec(estilo)
  if (!col || !lin) return null
  return { col: Number(col[1]), lin: Number(lin[1]) }
}

/**
 * Desliza a peça da casa velha para a nova.
 *
 * FLIP clássico, na ordem que o nome diz: o navegador já pintou o DESTINO
 * quando esta função roda, então ela INVERTE (põe a peça de volta na origem por
 * `transform`) e TOCA (volta a zero). Só `transform`, que mora no compositor.
 *
 * `quadrado` é o lado da casa em pixels — o `--quadrado` da cena, que é também
 * o que o zoom muda. Passá-lo em vez de lê-lo aqui dentro é o que mantém esta
 * função sem dependência do DOM em volta.
 *
 * @example slideTheToken(botao, { col: 3, lin: 2 }, { col: 7, lin: 2 }, 44)
 */
export function slideTheToken(
  alvo: Element | null | undefined,
  de: { col: number; lin: number },
  para: { col: number; lin: number },
  quadrado: number,
): void {
  if (!podeAnimar(alvo)) return
  const dx = (de.col - para.col) * quadrado
  const dy = (de.lin - para.lin) * quadrado
  // Peça que não andou não anima. Acontece a cada remendo que mexe em outra
  // coisa da peça — o PV, o selo, a vez —, e sem esta linha toda mudança de
  // estado viraria uma animação de zero pixel roubando um quadro.
  if (dx === 0 && dy === 0) return

  alvo.animate([{ transform: `translate(${dx}px, ${dy}px)` }, { transform: 'translate(0, 0)' }], {
    // 200ms é o que a issue mediu como legível sem atrasar quem está jogando —
    // e é a mesma faixa da entrada do palco (220ms), para a casa ter uma
    // cadência só.
    duration: 200,
    easing: 'cubic-bezier(0.22, 1, 0.36, 1)',
  })
}
