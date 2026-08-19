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
 * O AMBIENTE de cada lugar (ALE-188): a camada entre o chão e as peças.
 *
 * Com o relevo consertado na ALE-179 o chão deixou de ser cor chapada, mas
 * continuava um degradê simples — e o resto do app não é assim. O `.grimorio-stone`
 * do `index.css` é receita de três camadas (ambiente radial, calor, grão), e é
 * dessa gramática que estes vêm: a cripta tem o frio roxo caindo do alto e a
 * vinheta fechando os cantos; a taverna tem a LAREIRA (radial âmbar fora do
 * centro) e as pranchas do assoalho; a floresta tem duas clareiras verdes
 * descentradas, porque luz de mata não vem de um ponto só.
 *
 * O "papel" fica LIMPO de propósito: ele é o modo mapa de batalha, onde o que
 * importa é a grade e a peça, e ambiente ali seria sujeira sobre a régua.
 *
 * Continua sendo FUNDO: UM nó por camada, nunca um por quadrado — a restrição
 * que segura o custo de um plano infinito.
 */
export const TERRAIN_AMBIENCE: Record<string, string> = {
  pedra:
    'radial-gradient(120% 80% at 50% -10%, rgba(255,255,255,.06), transparent 60%), radial-gradient(100% 100% at 50% 120%, rgba(0,0,0,.45), transparent 55%)',
  taverna:
    'radial-gradient(45% 40% at 22% 30%, rgba(255,176,86,.18), transparent 70%), repeating-linear-gradient(90deg, rgba(0,0,0,.16) 0 2px, transparent 2px 92px), radial-gradient(120% 110% at 50% 120%, rgba(0,0,0,.42), transparent 60%)',
  floresta:
    'radial-gradient(38% 34% at 28% 24%, rgba(150,220,120,.14), transparent 70%), radial-gradient(30% 28% at 72% 62%, rgba(120,200,110,.10), transparent 72%), radial-gradient(120% 110% at 50% 115%, rgba(0,0,0,.45), transparent 58%)',
  ermo:
    'radial-gradient(80% 50% at 50% 0%, rgba(255,214,148,.10), transparent 65%), radial-gradient(120% 110% at 50% 120%, rgba(0,0,0,.40), transparent 60%)',
  cripta:
    'radial-gradient(60% 45% at 50% -5%, rgba(150,130,255,.16), transparent 70%), radial-gradient(110% 100% at 50% 115%, rgba(0,0,0,.55), transparent 55%)',
  papel: '',
}

/**
 * A grade, uma vez só e por cima de qualquer chão. Duas faixas cruzadas de 1px
 * em zero nós — a página é quem dá o `background-size` (o tamanho do quadrado)
 * e o `background-position` (a origem da janela), e é isso que faz a linha cair
 * na borda da célula em qualquer zoom.
 *
 * Clara no escuro e escura no claro seria uma segunda regra por terreno; em vez
 * disso a linha é OURO quase apagado — a assinatura da casa em vez do branco
 * neutro de planilha (ALE-188) — e o chão "Papel" ganha a versão escura pelo
 * `GRID_LINES_ON_LIGHT`.
 */
export const GRID_LINES =
  'repeating-linear-gradient(0deg,oklch(0.8 0.11 85 / 0.08) 0 1px,transparent 1px 100%),repeating-linear-gradient(90deg,oklch(0.8 0.11 85 / 0.08) 0 1px,transparent 1px 100%)'

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
