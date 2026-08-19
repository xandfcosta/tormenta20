/**
 * O teclado da superfície do tabuleiro (ALE-194).
 *
 * Traduz a tecla em INTENÇÃO e para por aí: quem sabe se há peça na mão, se
 * quem está olhando pode movê-la e onde a janela está é a cena, não a tabela de
 * teclas. É o que deixa a mesma gramática servir o tabuleiro vivo e o editor de
 * lugar sem duplicar o mapeamento.
 *
 * A dívida que isto paga está escrita no commit `c95d502`: ao tirar as quatro
 * setas e o −/+ do cabeçalho — porque arrastar e a roda fazem melhor —, mover e
 * ampliar passaram a existir SÓ no ponteiro, e quem usa teclado perdeu o que os
 * botões davam.
 *
 * `Alt+seta` do Roll20 (mover um PIXEL) não existe aqui de propósito: o estado
 * é em QUADRADOS inteiros (T20 p236), e meio quadrado não é uma posição.
 *
 * @example const acao = boardKeyAction(event); if (acao?.kind === 'step') …
 */
export type BoardKeyAction =
  /** Um quadrado numa direção: a peça na mão, ou a janela se não há peça. */
  | { kind: 'step'; dx: number; dy: number }
  /** Enquadrar as peças — o mesmo que o botão "Centralizar" faz. */
  | { kind: 'fit' }
  /** Aproximar ou afastar, em pixels por quadrado. */
  | { kind: 'zoom'; deltaPx: number }

const PASSO: Record<string, { dx: number; dy: number }> = {
  ArrowLeft: { dx: -1, dy: 0 },
  ArrowRight: { dx: 1, dy: 0 },
  ArrowUp: { dx: 0, dy: -1 },
  ArrowDown: { dx: 0, dy: 1 },
}

/**
 * O passo do zoom, em pixels por quadrado. É o mesmo 8 que os botões −/+ usavam
 * antes de saírem do cabeçalho: a tecla herda o degrau que a mesa já conhecia.
 */
const ZOOM_STEP_PX = 8

export function boardKeyAction(event: KeyboardEvent): BoardKeyAction | null {
  // Com modificador a tecla é do BROWSER ou do sistema: Ctrl+Home vai ao topo
  // do documento e Ctrl+− diminui a página. Roubá-las seria quebrar o que a
  // pessoa já sabe fazer para ganhar um atalho que ela não pediu.
  if (event.ctrlKey || event.metaKey || event.altKey) return null
  const passo = PASSO[event.key]
  if (passo) return { kind: 'step', ...passo }
  if (event.key === 'Home') return { kind: 'fit' }
  // O `=` entra junto do `+` porque no teclado ABNT2 e no US o mais exige
  // Shift: exigir a tecla exata seria exigir duas mãos para aproximar.
  if (event.key === '+' || event.key === '=') return { kind: 'zoom', deltaPx: ZOOM_STEP_PX }
  if (event.key === '-') return { kind: 'zoom', deltaPx: -ZOOM_STEP_PX }
  return null
}
