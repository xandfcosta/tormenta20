/**
 * Os lugares que o mestre pode escolher, desenhados em CSS (ALE-124).
 *
 * Sintético e não imagem: o app não tem upload nem armazenamento de arquivo, e
 * a primeira fatia do tabuleiro não pode ficar esperando essa esteira nascer.
 *
 * Aqui mora só o CHÃO. A grade saiu daqui e virou camada própria (ALE-179): ela
 * era repetida seis vezes, uma por terreno, e o `background-size` do quadrado
 * que a página aplicava para alinhar as linhas valia para TODAS as camadas —
 * então o gradiente do chão era ladrilhado a cada 44px e o lugar inteiro virava
 * uma cor chapada. Era por isso que escolher "Cripta" ou "Floresta" não mudava
 * nada visível: a cor estava lá, o RELEVO não.
 */
export const TERRAIN_STYLE: Record<string, string> = {
  pedra: 'bg-[linear-gradient(160deg,#332f2a,#1d1b18_60%,#15140f)]',
  taverna: 'bg-[linear-gradient(160deg,#4a3624,#2a1e14_60%,#1b120c)]',
  floresta: 'bg-[linear-gradient(160deg,#25381f,#162114_60%,#0f170e)]',
  ermo: 'bg-[linear-gradient(160deg,#413726,#241f16_60%,#18140e)]',
  cripta: 'bg-[linear-gradient(160deg,#2b2a33,#181720_60%,#101017)]',
  papel: 'bg-[linear-gradient(160deg,#efe6cf,#ddd0b2_60%,#cbbc9a)]',
}

/**
 * A grade, uma vez só e por cima de qualquer chão. Duas faixas cruzadas de 1px
 * em zero nós — a página é quem dá o `background-size` (o tamanho do quadrado)
 * e o `background-position` (a origem da janela), e é isso que faz a linha cair
 * na borda da célula em qualquer zoom.
 *
 * Clara no escuro e escura no claro seria uma segunda regra por terreno; em vez
 * disso a linha é branca com opacidade baixa e o chão "Papel" ganha a versão
 * escura pelo `GRID_ON_LIGHT`.
 */
export const GRID_LINES =
  'repeating-linear-gradient(0deg,rgba(255,255,255,.07) 0 1px,transparent 1px 100%),repeating-linear-gradient(90deg,rgba(255,255,255,.07) 0 1px,transparent 1px 100%)'

/** A mesma grade para chão claro: linha branca sobre pergaminho é invisível. */
export const GRID_LINES_ON_LIGHT =
  'repeating-linear-gradient(0deg,rgba(0,0,0,.12) 0 1px,transparent 1px 100%),repeating-linear-gradient(90deg,rgba(0,0,0,.12) 0 1px,transparent 1px 100%)'

/** Os chãos claros, que pedem a grade escura. */
const LIGHT_TERRAIN = new Set(['papel'])

export function gridLinesFor(terrain: string): string {
  return LIGHT_TERRAIN.has(terrain) ? GRID_LINES_ON_LIGHT : GRID_LINES
}

/** Rótulo de cada chão, para o seletor do mestre. */
export const TERRAIN_LABEL: Record<keyof typeof TERRAIN_STYLE, string> = {
  pedra: 'Pedra',
  taverna: 'Taverna',
  floresta: 'Floresta',
  ermo: 'Ermo',
  cripta: 'Cripta',
  papel: 'Papel',
}

export const TERRAIN_IDS = Object.keys(TERRAIN_STYLE)
