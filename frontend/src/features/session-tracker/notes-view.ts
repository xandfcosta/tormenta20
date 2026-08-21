/**
 * O modo de visualização das notas da sessão (ALE-139).
 *
 * Eram lado a lado FIXO, sempre 50/50: com uma nota de duas linhas, duas
 * colunas quase vazias; com uma nota longa, escrever numa coluna estreita.
 * O mestre escolhe, e a escolha GRUDA — é preferência de trabalho, não estado
 * da sessão, e ele não vai reescolher a cada aba que abre.
 */
export type NotesView = 'escrever' | 'ler' | 'duplo'

export const NOTES_VIEW_KEY = 't20:notas-view'

/** O padrão em tela larga é o de hoje: ninguém perde o arranjo que já usava. */
const PADRAO: NotesView = 'duplo'

const VALIDOS: readonly NotesView[] = ['escrever', 'ler', 'duplo']

/**
 * O modo guardado, ou o padrão quando o armazenamento não diz nada útil.
 *
 * Tolerante por desenho: um valor estranho no `localStorage` — de uma versão
 * antiga, de outra aba, de um dedo no console — não pode deixar a aba de notas
 * sem arranjo nenhum.
 *
 * @example readNotesView('ler') // 'ler'
 * @example readNotesView('paisagem') // 'duplo'
 */
export function readNotesView(raw: string | null): NotesView {
  return VALIDOS.includes(raw as NotesView) ? (raw as NotesView) : PADRAO
}

/**
 * A largura de contêiner a partir da qual duas colunas cabem — o `@2xl` do
 * Tailwind, que é onde o arranjo lado a lado já vivia antes desta issue.
 */
const LARGURA_PARA_DUAS_COLUNAS = 672

/**
 * Se a região tem espaço para as duas colunas.
 *
 * Largura zero significa "ainda não medi" e responde SIM, que é o arranjo de
 * sempre: a alternativa faria a aba piscar do editor sozinho para o lado a lado
 * no primeiro quadro depois da medição.
 *
 * @example cabeLadoALado(0) // true — ainda não medi
 * @example cabeLadoALado(400) // false
 */
export function cabeLadoALado(larguraDaRegiao: number): boolean {
  return larguraDaRegiao === 0 || larguraDaRegiao >= LARGURA_PARA_DUAS_COLUNAS
}

/**
 * O modo que a região consegue MOSTRAR, dado quanto espaço ela tem.
 *
 * Lado a lado não cabe num contêiner estreito, e ali ele vira "escrever" — não
 * "ler", porque quem estava no modo duplo estava escrevendo e lendo ao mesmo
 * tempo, e tirar o editor de quem digita é pior que tirar a prévia. Quem
 * escolheu "ler" continua lendo: essa escolha é explícita e a largura não a
 * contradiz.
 *
 * A largura é do CONTÊINER e não da janela: esta região é 7/12 da tela na cena
 * do mestre e a tela inteira na gaveta, e um limiar de viewport prometeria duas
 * colunas onde cabe uma só — a ALE-122 tropeçou nisso três vezes.
 *
 * @example efetivo('duplo', false) // 'escrever'
 * @example efetivo('ler', false) // 'ler'
 */
export function efetivo(escolhido: NotesView, cabeLadoALado: boolean): NotesView {
  if (escolhido === 'duplo' && !cabeLadoALado) return 'escrever'
  return escolhido
}
