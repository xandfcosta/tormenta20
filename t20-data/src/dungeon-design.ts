/**
 * Masmorras — PDF Cap 6 p263 ("Ambientes de aventura → masmorras").
 *
 * Duas ferramentas de GM:
 *
 *   1. Tamanhos de masmorra (pequena/média/grande) + regras derivadas:
 *      - número de salas
 *      - "1 ameaça por 3 salas" (padrão único, escala com tamanho)
 *      - objetivos secundários por tamanho (média até 1, grande até 3)
 *      - objetivos opcionais por tamanho (pequena 1, média 2, grande 3)
 *      - pacing (parte de sessão / sessão inteira / aventura inteira)
 *      - livro recomenda não passar de "grande"
 *
 *   2. Tabela 6-2: Ideias de Masmorras (p263) — d20 rolável, 20 conceitos.
 *      Serve de gerador de brainstorm; o "conceito" da masmorra é depois
 *      preenchido com objetivos + ameaças + mapa (livro p263).
 *
 * O livro descreve várias outras etapas ("Conceito", "Objetivo Principal",
 * "Pontos de Interesse", "Mapa", "Descrição e Regras") mas essas são
 * puramente prosa de GM e ficam no PDF.
 */

export const DUNGEON_SIZES = ['pequena', 'media', 'grande'] as const

export type DungeonSize = (typeof DUNGEON_SIZES)[number]

export type DungeonSizeRow = {
  size: DungeonSize
  label: string
  minRooms: number
  maxRooms: number
  /**
   * Duração narrativa esperada. Livro p263:
   *   - pequena  = "parte de uma sessão"
   *   - média    = "uma sessão inteira"
   *   - grande   = "escopo de toda uma aventura"
   */
  pacing: 'parte-de-sessao' | 'sessao-inteira' | 'aventura-inteira'
  /**
   * Total de objetivos secundários possíveis (p263):
   *   - pequena: 0 (apenas médias/grandes tem secundários)
   *   - média: até 1
   *   - grande: até 3
   */
  maxSecondaryObjectives: number
  /**
   * Objetivos opcionais recomendados (p263, "Objetivos Opcionais"):
   *   - pequena: 1
   *   - média:  2
   *   - grande: 3
   */
  optionalObjectives: number
}

export const DUNGEON_SIZE_TABLE: readonly DungeonSizeRow[] = Object.freeze([
  {
    size: 'pequena',
    label: 'Pequena',
    minRooms: 3,
    maxRooms: 6,
    pacing: 'parte-de-sessao',
    maxSecondaryObjectives: 0,
    optionalObjectives: 1,
  },
  {
    size: 'media',
    label: 'Média',
    minRooms: 7,
    maxRooms: 20,
    pacing: 'sessao-inteira',
    maxSecondaryObjectives: 1,
    optionalObjectives: 2,
  },
  {
    size: 'grande',
    label: 'Grande',
    minRooms: 21,
    maxRooms: 50,
    pacing: 'aventura-inteira',
    maxSecondaryObjectives: 3,
    optionalObjectives: 3,
  },
])

export function dungeonSizeRow(size: DungeonSize): DungeonSizeRow {
  const row = DUNGEON_SIZE_TABLE.find((r) => r.size === size)
  if (!row) throw new Error(`Unknown DungeonSize: ${size}`)
  return row
}

/**
 * Ameaças a distribuir na masmorra. Livro p263: "Calcule uma ameaça
 * para cada três salas, com um misto de cenas de ação (monstros,
 * armadilhas) e exploração (labirintos, enigmas)."
 *
 * Uso: `plannedThreats(numRooms)` → arredonda pra cima (uma masmorra
 * de 5 salas tem 2 ameaças; uma de 6 salas tem 2 ameaças; uma de 7
 * salas tem 3).
 */
export const ROOMS_PER_THREAT = 3

export function plannedThreats(numRooms: number): number {
  if (!Number.isFinite(numRooms) || numRooms <= 0) {
    throw new Error(`numRooms must be > 0, got ${numRooms}`)
  }
  return Math.ceil(numRooms / ROOMS_PER_THREAT)
}

/**
 * Menor tamanho de masmorra que comporta `numRooms` salas. `null`
 * quando o número passa do limite superior de "grande" (50) — livro
 * recomenda não crescer além disso.
 */
export function classifyDungeonSize(numRooms: number): DungeonSize | null {
  if (!Number.isFinite(numRooms) || numRooms < 1) {
    throw new Error(`numRooms must be >= 1, got ${numRooms}`)
  }
  for (const row of DUNGEON_SIZE_TABLE) {
    if (numRooms >= row.minRooms && numRooms <= row.maxRooms) return row.size
  }
  return null
}

// ─── Tabela 6-2: Ideias de Masmorras (p263) ──────────────────────────

export type DungeonIdea = { roll: number; label: string }

export const DUNGEON_IDEA_TABLE: readonly DungeonIdea[] = Object.freeze([
  { roll: 1, label: 'Complexo de cavernas subterrâneas' },
  { roll: 2, label: 'Mina abandonada' },
  { roll: 3, label: 'Templo de um deus maligno' },
  { roll: 4, label: 'Esgotos da cidade' },
  { roll: 5, label: 'Castelo de um déspota' },
  { roll: 6, label: 'Torre de um mago louco' },
  { roll: 7, label: 'Moinho da vila' },
  { roll: 8, label: 'Armazém no porto' },
  { roll: 9, label: 'Ruínas de uma civilização perdida' },
  { roll: 10, label: 'Fortaleza anã abandonada' },
  { roll: 11, label: 'Mansão assombrada' },
  { roll: 12, label: 'Prisão da cidade' },
  { roll: 13, label: 'Caverna submersa' },
  { roll: 14, label: 'Gruta usada como covil por um monstro' },
  { roll: 15, label: 'Biblioteca mágica' },
  { roll: 16, label: 'Galeão encalhado' },
  { roll: 17, label: 'Labirinto feito para proteger um tesouro' },
  { roll: 18, label: 'Manicômio repleto de vilões insanos' },
  { roll: 19, label: 'Vulcão inativo' },
  { roll: 20, label: 'Castelo nas nuvens' },
])

/**
 * Resolve um d20 → ideia de masmorra. Aceita 1-20; lança fora do range
 * (Tabela 6-2 é d20 puro, sem 21+).
 */
export function dungeonIdeaFromRoll(d20: number): DungeonIdea {
  if (!Number.isInteger(d20) || d20 < 1 || d20 > 20) {
    throw new Error(`d20 roll must be an integer 1-20, got ${d20}`)
  }
  const row = DUNGEON_IDEA_TABLE[d20 - 1]!
  return row
}
