import type {
  ChaseEventRow,
  DungeonIdea,
  DungeonSizeRow,
  RewardCastigoRow,
  RuinaRow,
} from '@/shared/api/api'
import { dungeonDesignData, gmTablesData } from '@/shared/lib/rules-tables-cache'

/**
 * Resolvedores das tabelas de rolagem da Mesa do Mestre — Cap 6 (p263-274).
 *
 * Moravam no `t20-data` GRUDADOS nas tabelas, e por isso as tabelas iam para o
 * bundle. As tabelas desceram para o catálogo servido; aqui ficou só a regra de
 * "qual linha esta rolagem acerta" (ALE-102).
 *
 * A cobertura contígua das faixas é garantida no servidor
 * (`catalog/rules_tables_test.go`), então uma rolagem válida SEMPRE acha linha —
 * é por isso que estas funções podem tratar "não achou" como erro de programação
 * em vez de estado normal.
 */

function assertRoll(value: number, sides: number): void {
  if (!Number.isInteger(value) || value < 1 || value > sides) {
    throw new Error(`rolagem de d${sides} deve ser inteiro de 1 a ${sides}, veio ${value}`)
  }
}

function rowForRoll<T extends { rollMin: number; rollMax: number }>(
  rows: readonly T[],
  roll: number,
  tabela: string,
): T {
  const row = rows.find((r) => roll >= r.rollMin && roll <= r.rollMax)
  if (!row) throw new Error(`${tabela}: nenhuma linha cobre a rolagem ${roll}`)
  return row
}

/** Tabela 6-4: Ruínas (d6, p272). */
export function ruinaFromRoll(d6: number): RuinaRow {
  assertRoll(d6, 6)
  return rowForRoll(gmTablesData().ruina, d6, 'ruina')
}

/** Tabela 6-5: Eventos de Perseguição (d20, p274). */
export function chaseEventFromRoll(d20: number): ChaseEventRow {
  assertRoll(d20, 20)
  return rowForRoll(gmTablesData().chaseEvents, d20, 'chaseEvents')
}

/** Tabela de recompensa/castigo (d6). */
export function rewardCastigoFromRoll(d6: number): RewardCastigoRow {
  assertRoll(d6, 6)
  const row = gmTablesData().rewardCastigo.find((r) => r.roll === d6)
  if (!row) throw new Error(`rewardCastigo: nenhuma linha para a rolagem ${d6}`)
  return row
}

export function rewardLabel(kind: string): string {
  return gmTablesData().rewardLabels[kind] ?? kind
}

export function castigoLabel(kind: string): string {
  return gmTablesData().castigoLabels[kind] ?? kind
}

/** Tabela 6-2: Ideias de Masmorra (d20, p263). */
export function dungeonIdeaFromRoll(d20: number): DungeonIdea {
  assertRoll(d20, 20)
  const ideas = dungeonDesignData().ideas
  const row = ideas[d20 - 1]
  if (!row) throw new Error(`dungeon ideas: nenhuma linha para a rolagem ${d20}`)
  return row
}

export function dungeonSizeRow(size: string): DungeonSizeRow {
  const row = dungeonDesignData().sizeTable.find((r) => r.size === size)
  if (!row) throw new Error(`tamanho de masmorra desconhecido: ${size}`)
  return row
}

/**
 * Ameaças a distribuir. Livro p263: "Calcule uma ameaça para cada três salas,
 * com um misto de cenas de ação e exploração."
 */
export function plannedThreats(numRooms: number): number {
  if (!Number.isFinite(numRooms) || numRooms <= 0) {
    throw new Error(`numRooms deve ser > 0, veio ${numRooms}`)
  }
  return Math.ceil(numRooms / dungeonDesignData().roomsPerThreat)
}

/**
 * Menor tamanho que comporta `numRooms` salas; `null` acima do teto de "grande"
 * (50), que é onde o livro recomenda parar.
 */
export function classifyDungeonSize(numRooms: number): string | null {
  if (!Number.isFinite(numRooms) || numRooms < 1) {
    throw new Error(`numRooms deve ser >= 1, veio ${numRooms}`)
  }
  for (const row of dungeonDesignData().sizeTable) {
    if (numRooms >= row.minRooms && numRooms <= row.maxRooms) return row.size
  }
  return null
}
