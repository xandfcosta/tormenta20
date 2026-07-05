/**
 * GM random tables — PDF Cap 6. Quatro tabelas de rolagem que o mestre
 * usa para gerar conteúdo procedural durante uma sessão:
 *
 *   - Ermos: Ruína d6 (p269)
 *   - Tabela 6-5: Eventos de Perseguições d20 (p274)
 *   - Tabela 6-6: Desafios de Buscas 2d12 (p279)
 *   - Tabela 6-7: Consequências de Buscas + Recompensas/Castigos d6 (p279)
 *
 * A `Tabela 6-2: Ideias de Masmorras d20` fica em `dungeon-design.ts`
 * — está tematicamente ligada ao design de masmorra, não é usada como
 * gerador rápido de mesa.
 */

// ─── Ermos: Ruína d6 (p269) ──────────────────────────────────────────

export type RuinaOutcome =
  | 'ameaca'
  | 'vazia'
  | 'ameaca-e-tesouro'
  | 'tesouro'

export type RuinaRow = {
  rollMin: number
  rollMax: number
  outcome: RuinaOutcome
  label: string
}

export const RUINA_TABLE: readonly RuinaRow[] = Object.freeze([
  {
    rollMin: 1,
    rollMax: 2,
    outcome: 'ameaca',
    label: 'Uma ameaça (armadilha ou monstro, a critério do mestre)',
  },
  { rollMin: 3, rollMax: 4, outcome: 'vazia', label: 'Vazia' },
  {
    rollMin: 5,
    rollMax: 5,
    outcome: 'ameaca-e-tesouro',
    label: 'Uma ameaça e um tesouro',
  },
  { rollMin: 6, rollMax: 6, outcome: 'tesouro', label: 'Apenas um tesouro' },
])

export function ruinaFromRoll(d6: number): RuinaRow {
  if (!Number.isInteger(d6) || d6 < 1 || d6 > 6) {
    throw new Error(`d6 roll must be an integer 1-6, got ${d6}`)
  }
  const row = RUINA_TABLE.find((r) => d6 >= r.rollMin && d6 <= r.rollMax)!
  return row
}

// ─── Tabela 6-5: Eventos de Perseguições d20 (p274) ──────────────────

export type ChaseEventKind = 'nenhum' | 'obstaculo' | 'atalho'

export type ChaseEventRow = {
  rollMin: number
  rollMax: number
  kind: ChaseEventKind
  /** Perícia ou teste de resistência exigido (null = nenhum). */
  test:
    | null
    | 'Força'
    | 'Acrobacia'
    | 'Reflexos'
    | 'Intimidação'
    | 'Adestramento'
    | 'Percepção'
  cd: number | null
  example: string
}

export const CHASE_EVENT_TABLE: readonly ChaseEventRow[] = Object.freeze([
  {
    rollMin: 1,
    rollMax: 6,
    kind: 'nenhum',
    test: null,
    cd: null,
    example: '—',
  },
  {
    rollMin: 7,
    rollMax: 8,
    kind: 'obstaculo',
    test: 'Força',
    cd: 15,
    example: 'Pilha de caixotes bloqueia o caminho.',
  },
  {
    rollMin: 9,
    rollMax: 10,
    kind: 'obstaculo',
    test: 'Acrobacia',
    cd: 20,
    example: 'Frutas caídas deixam o piso escorregadio.',
  },
  {
    rollMin: 11,
    rollMax: 12,
    kind: 'obstaculo',
    test: 'Reflexos',
    cd: 20,
    example: 'Barris rolam pela rua.',
  },
  {
    rollMin: 13,
    rollMax: 14,
    kind: 'obstaculo',
    test: 'Intimidação',
    cd: 20,
    example: 'Multidão impede a passagem.',
  },
  {
    rollMin: 15,
    rollMax: 16,
    kind: 'atalho',
    test: 'Adestramento',
    cd: 20,
    example: 'Carroça na qual se pode tentar subir.',
  },
  {
    rollMin: 17,
    rollMax: 18,
    kind: 'atalho',
    test: 'Força',
    cd: 15,
    example: 'Caminho mais curto, mas bloqueado.',
  },
  {
    rollMin: 19,
    rollMax: 20,
    kind: 'atalho',
    test: 'Percepção',
    cd: 20,
    example: 'Ruelas labirínticas, nas quais se pode cortar caminho ou se perder.',
  },
])

export function chaseEventFromRoll(d20: number): ChaseEventRow {
  if (!Number.isInteger(d20) || d20 < 1 || d20 > 20) {
    throw new Error(`d20 roll must be an integer 1-20, got ${d20}`)
  }
  const row = CHASE_EVENT_TABLE.find(
    (r) => d20 >= r.rollMin && d20 <= r.rollMax,
  )!
  return row
}

// ─── Tabela 6-6: Desafios de Buscas 2d12 (p279) ──────────────────────

export type BuscaChallengeRow = {
  roll: number
  skill: string
  example: string
}

export const BUSCA_CHALLENGE_TABLE: readonly BuscaChallengeRow[] = Object.freeze([
  { roll: 2, skill: 'Misticismo', example: 'Decifrar uma runa' },
  { roll: 3, skill: 'Adestramento', example: 'Acalmar uma fera' },
  { roll: 4, skill: 'Conhecimento', example: 'Traduzir um texto antigo' },
  { roll: 5, skill: 'Enganação', example: 'Participar de uma intriga' },
  { roll: 6, skill: 'Cura', example: 'Tratar um veneno' },
  { roll: 7, skill: 'Iniciativa', example: 'Perseguir um bandido' },
  { roll: 8, skill: 'Intimidação', example: 'Negociar com um criminoso' },
  { roll: 9, skill: 'Investigação', example: 'Descobrir uma localização' },
  { roll: 10, skill: 'Reflexos', example: 'Evitar um desmoronamento' },
  { roll: 11, skill: 'Atletismo', example: 'Escalar um penhasco' },
  { roll: 12, skill: 'Percepção', example: 'Evitar uma emboscada' },
  { roll: 13, skill: 'Sobrevivência', example: 'Atravessar os ermos' },
  { roll: 14, skill: 'Fortitude', example: 'Tolerar clima ruim' },
  { roll: 15, skill: 'Diplomacia', example: 'Negociar com um mercador' },
  { roll: 16, skill: 'Furtividade', example: 'Infiltrar-se num lugar' },
  { roll: 17, skill: 'Acrobacia', example: 'Atravessar uma ravina' },
  { roll: 18, skill: 'Intuição', example: 'Elucidar um enigma' },
  { roll: 19, skill: 'Vontade', example: 'Resistir a uma maldição' },
  { roll: 20, skill: 'Luta', example: 'Defender-se de um monstro' },
  { roll: 21, skill: 'Jogatina', example: 'Apostar com as fadas' },
  { roll: 22, skill: 'Nobreza', example: 'Participar de um baile' },
  { roll: 23, skill: 'Religião', example: 'Entender um presságio' },
  { roll: 24, skill: 'Guerra', example: 'Atravessar um campo de batalha' },
])

export function buscaChallengeFromRoll(twoD12: number): BuscaChallengeRow {
  if (!Number.isInteger(twoD12) || twoD12 < 2 || twoD12 > 24) {
    throw new Error(`2d12 roll must be an integer 2-24, got ${twoD12}`)
  }
  const row = BUSCA_CHALLENGE_TABLE.find((r) => r.roll === twoD12)!
  return row
}

/**
 * CD do teste da busca. Livro p278: "CD 20 + metade do nível do
 * personagem". Meio nível arredondado pra baixo (padrão T20 para
 * "metade do nível" em outros lugares como Fortitude/Reflexos/Vontade
 * treinada).
 */
export function buscaTestCd(characterLevel: number): number {
  if (!Number.isInteger(characterLevel) || characterLevel < 1) {
    throw new Error(`characterLevel must be an integer >= 1, got ${characterLevel}`)
  }
  return 20 + Math.floor(characterLevel / 2)
}

// ─── Tabela 6-7: Consequências de Buscas + Recompensas/Castigos d6 (p279) ─

export type BuscaOutcome =
  | { successes: 0; result: '1-castigo' }
  | { successes: 1; result: 'nenhuma' }
  | { successes: 2; result: '1-recompensa' }
  | { successes: 3; result: '2-recompensas' }

export const BUSCA_OUTCOME_TABLE: readonly BuscaOutcome[] = Object.freeze([
  { successes: 0, result: '1-castigo' },
  { successes: 1, result: 'nenhuma' },
  { successes: 2, result: '1-recompensa' },
  { successes: 3, result: '2-recompensas' },
])

export function buscaOutcomeFromSuccesses(successes: 0 | 1 | 2 | 3): BuscaOutcome {
  return BUSCA_OUTCOME_TABLE[successes]!
}

export type RewardKind =
  | 'tesouro-riqueza'
  | 'favor'
  | 'tesouro-item'
  | 'informacao'
  | 'tesouro-ambos'
  | 'poder'

export type CastigoKind =
  | 'ruina-menor'
  | 'abalo'
  | 'complicacao'
  | 'ferimento'
  | 'maldicao'
  | 'ruina-maior'

export type RewardCastigoRow = {
  roll: number
  reward: RewardKind
  castigo: CastigoKind
}

export const REWARD_CASTIGO_TABLE: readonly RewardCastigoRow[] = Object.freeze([
  { roll: 1, reward: 'tesouro-riqueza', castigo: 'ruina-menor' },
  { roll: 2, reward: 'favor', castigo: 'abalo' },
  { roll: 3, reward: 'tesouro-item', castigo: 'complicacao' },
  { roll: 4, reward: 'informacao', castigo: 'ferimento' },
  { roll: 5, reward: 'tesouro-ambos', castigo: 'maldicao' },
  { roll: 6, reward: 'poder', castigo: 'ruina-maior' },
])

export function rewardCastigoFromRoll(d6: number): RewardCastigoRow {
  if (!Number.isInteger(d6) || d6 < 1 || d6 > 6) {
    throw new Error(`d6 roll must be an integer 1-6, got ${d6}`)
  }
  const row = REWARD_CASTIGO_TABLE.find((r) => r.roll === d6)!
  return row
}
