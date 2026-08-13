/**
 * Tipos das tabelas de livro que o SERVIDOR autora e serve (`/catalog/...`).
 *
 * Elas viviam em `@tormenta20/t20-data` e entravam no bundle em tempo de build —
 * eram os últimos bytes de catálogo embarcados (ALE-102). A forma do dado é
 * validada no Go, em `catalog/rules_tables_test.go`; aqui só existe o contrato
 * que o front lê.
 */

/** Perícias que uma classe treina — Cap 1, entradas de classe (p36-83). */
export type ClassTrainedExpertises = {
  /** Sempre treinadas. */
  fixed: string[]
  /** Escolha UMA: as linhas "Luta ou Pontaria" / "Diplomacia ou Intimidação". */
  eitherOr?: { options: [string, string] }
  /** Quantas perícias adicionais o jogador escolhe de `choosePool`. */
  chooseCount: number
  choosePool: string[]
}

/**
 * Termos de devoto (p96) — a linha "Devotos." de cada deus é texto verbatim do
 * livro (plurais, sentinelas), então o mapa traduz termo → nomes do app.
 */
export type DevotoTerms = {
  /** Sentinelas que admitem todo mundo ("Quaisquer"). */
  openTerms: string[]
  termToNames: Record<string, string[]>
}

/** Uma linha de tabela de rolagem: cobre a faixa `rollMin..rollMax`. */
export type RollRangeRow = { rollMin: number; rollMax: number }

export type RuinaRow = RollRangeRow & { outcome: string; label: string }

export type ChaseEventRow = RollRangeRow & {
  kind: string
  test: string | null
  cd: number | null
  example: string
}

export type RewardCastigoRow = { roll: number; reward: string; castigo: string }

export type GmTables = {
  ruina: RuinaRow[]
  chaseEvents: ChaseEventRow[]
  rewardCastigo: RewardCastigoRow[]
  rewardLabels: Record<string, string>
  castigoLabels: Record<string, string>
}

export type DungeonSizeRow = {
  size: string
  label: string
  minRooms: number
  maxRooms: number
  pacing: string
  maxSecondaryObjectives: number
  optionalObjectives: number
}

export type DungeonIdea = { roll: number; label: string }

export type DungeonDesign = {
  sizes: string[]
  sizeTable: DungeonSizeRow[]
  /** Salas por ameaça planejada — divisor, nunca zero (validado no Go). */
  roomsPerThreat: number
  ideas: DungeonIdea[]
}
