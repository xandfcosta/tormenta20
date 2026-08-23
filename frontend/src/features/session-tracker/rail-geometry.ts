/**
 * Altura de um item da fila no trilho, em px.
 *
 * CONSTANTE e não medida, e isso é deliberado: o que precisa de observador é a
 * altura do TRILHO, que muda com a janela; a altura de um item é decidida pelo
 * CSS deste projeto e só muda quando alguém a reescreve. Medir cada item
 * custaria um `ResizeObserver` por combatente para responder um número que já
 * está escrito.
 *
 * O preço de ela mentir é BAIXO por construção: a conta usa `Math.floor` e a
 * lista recorta o que passa, então um valor pequeno demais mostra um vizinho a
 * menos e um valor grande demais nunca transborda. Se o item mudar de tamanho,
 * este número muda junto — eles vivem no mesmo commit ou a fila passa a mentir
 * sobre quantos vizinhos cabem.
 *
 * 52 = o item (iniciais, filete de vida, respiro e borda) MAIS o vão que vem
 * depois dele. Contar o vão dentro do item sobra um vão no fim da lista, o que
 * empurra a conta para BAIXO — que é o lado seguro dos dois.
 */
export const ITEM_DA_FILA_PX = 52

/** O cabeçalho de um bloco do trilho (o botão que abre a gaveta). */
export const CABECALHO_DO_BLOCO_PX = 32

/**
 * Quantos combatentes cabem no bloco da fila, dada a altura do TRILHO INTEIRO.
 *
 * A conta parte do trilho e não do bloco, e essa escolha é o que impede um
 * laço: o bloco é dimensionado pelo conteúdo até o teto de metade, então medir
 * o BLOCO para decidir quantos itens desenhar faria a resposta mudar o que a
 * pergunta media — desenha menos, o bloco encolhe, cabem mais, desenha mais.
 * A altura do trilho é estado de fora e não se mexe com o que a fila decide.
 *
 * Altura zero significa "ainda não medi", e a resposta é ZERO vizinhos — não um
 * palpite. Quem chama mostra a fila inteira nesse quadro; ver demais por um
 * quadro é melhor que afirmar uma vizinhança inventada.
 *
 * @example cabemNaFila(900) // 8
 */
export function cabemNaFila(alturaDoTrilho: number): number {
  if (alturaDoTrilho <= 0) return 0
  const doBloco = alturaDoTrilho / 2 - CABECALHO_DO_BLOCO_PX
  return Math.max(0, Math.floor(doBloco / ITEM_DA_FILA_PX))
}

/** A faixa de índices da fila que o trilho desenha, inclusiva nas duas pontas. */
export type JanelaDaFila = { inicio: number; fim: number }

/**
 * Qual pedaço da fila o trilho mostra, centrado em QUEM ESTÁ NA VEZ.
 *
 * A fila do trilho não rola (decisão do dono): ela mostra o que couber, com
 * quem já agiu acima e quem ainda vai abaixo. Esta é a conta que decide o que
 * couber, e ela é regra e não enfeite — com um número errado de vizinhos, o
 * trilho MENTE sobre quem já jogou.
 *
 * Centrada e GRAMPEADA nas pontas, o que faz a vez sair do centro no começo e
 * no fim da rodada. Isso é a resposta certa e não uma limitação: no turno 1 não
 * há ninguém acima porque ninguém agiu, e reservar espaço vazio ali seria
 * desenhar um passado que não existe. O grampo devolve a sobra ao outro lado.
 *
 * NÃO é circular, e essa é a diferença para o `upcomingTurns` (ALE-179), que é
 * circular de propósito. A razão de lá — "cortar no fim deixaria a tira vazia
 * justamente no turno em que saber quem vem depois mais importa" — não vale
 * aqui, porque quem responde isso é o AVANÇO na faixa do turno, que diz
 * "Próximo: Fulano" dando a volta e está sempre na tela. O trilho paga o preço
 * de girar (o mestre perde a âncora de "o Ogro é o terceiro") sem comprar nada
 * que a faixa já não entregue.
 *
 * @example janelaDaFila({ total: 9, turnIndex: 4, cabem: 5 }) // { inicio: 2, fim: 6 }
 * @example janelaDaFila({ total: 9, turnIndex: 0, cabem: 5 }) // { inicio: 0, fim: 4 }
 */
export function janelaDaFila(input: {
  total: number
  turnIndex: number
  cabem: number
}): JanelaDaFila {
  const { total, turnIndex, cabem } = input
  if (total <= 0) return { inicio: 0, fim: -1 }
  // `cabem` zero é "ainda não medi": mostrar a fila inteira por um quadro é
  // melhor que esconder todo mundo, e o recorte do CSS impede o transbordo.
  if (cabem <= 0 || cabem >= total) return { inicio: 0, fim: total - 1 }
  // Fora de combate (turnIndex −1) não há centro: a fila começa do topo, que é
  // a ordem em que ela vai ser jogada.
  const centro = turnIndex < 0 ? 0 : turnIndex
  const acima = Math.floor((cabem - 1) / 2)
  const inicio = Math.min(Math.max(centro - acima, 0), total - cabem)
  return { inicio, fim: inicio + cabem - 1 }
}
