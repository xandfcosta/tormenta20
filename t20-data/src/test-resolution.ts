/**
 * Test Resolution — regras gerais de teste (Cap 5 Jogando p220-223, p227).
 *
 * Cobre:
 *  - Tabela 5-1 (escala de dificuldades — CD 5..40) — p220
 *  - Sucessos/falhas automáticos (20 nat / 1 nat) — p221
 *  - Escolher 0 / 10 / 20 — p222
 *  - Ajudar (CD 10, +1 base, +1 por 10 acima) — p221
 *  - Testes estendidos (Tabela 5-2, sucessos 3/5/7 antes de 3 falhas) — p223
 *  - Ferramentas ausentes (-5) — p221
 *  - Fórmula CD de resistência (10 + nível/2 + atributo) — p227
 *  - Tipos de sucesso em resistência (Anula/Parcial/Metade/Desacredita) — p227
 */

// ─── Types ────────────────────────────────────────────────────────────
/** Categorias de dificuldade (Tabela 5-1 p220). Não há tier CD 35. */
export type CdDifficultyTier =
  | 'facil'
  | 'media'
  | 'dificil'
  | 'desafiadora'
  | 'formidavel'
  | 'heroica'
  | 'quase-impossivel'

/** Complexidade de teste estendido (Tabela 5-2 p223). */
export type ExtendedComplexity = 'baixa' | 'media' | 'alta'

/** Tipo de sucesso em teste de resistência (p227). */
export type ResistanceSaveOutcome = 'anula' | 'parcial' | 'metade' | 'desacredita'

/** Estado de um teste estendido em progresso. */
export type ExtendedTestState = 'success' | 'in-progress' | 'failure'

// ─── Constantes ──────────────────────────────────────────────────────
/** Tabela 5-1 verbatim (p220). */
export const DIFFICULTY_CD: Readonly<Record<CdDifficultyTier, number>> =
  Object.freeze({
    facil: 5,
    media: 10,
    dificil: 15,
    desafiadora: 20,
    formidavel: 25,
    heroica: 30,
    'quase-impossivel': 40,
  })

/** Tabela 5-2 verbatim (p223): sucessos exigidos por complexidade. */
export const EXTENDED_SUCCESSES_REQUIRED: Readonly<
  Record<ExtendedComplexity, number>
> = Object.freeze({
  baixa: 3,
  media: 5,
  alta: 7,
})

/** Máximo de falhas em teste estendido antes de falha total (p223). */
export const EXTENDED_MAX_FAILURES = 3

/** Opcional: CD aumenta +2 por teste feito em estendido (p223). */
export const EXTENDED_CUMULATIVE_CD_INCREMENT = 2

/** Opcional: cada falha impõe -2 cumulativo em testes seguintes (p223). */
export const EXTENDED_CUMULATIVE_FAILURE_PENALTY = -2

/** Escolher 10 — resultado = 10 + modificadores (p222). */
export const TAKE_TEN_VALUE = 10

/** Escolher 20 — resultado = 20 + modificadores (p222). */
export const TAKE_TWENTY_VALUE = 20

/** Escolher 20 leva 20× o tempo normal (p222). */
export const TAKE_TWENTY_TIME_MULTIPLIER = 20

/** Ajudar tem CD 10 (p221). */
export const AID_ANOTHER_CD = 10

/** Bônus base ao ajudar = +1 no sucesso (p221). */
export const AID_ANOTHER_BASE_BONUS = 1

/** Bônus adicional por cada 10 acima da CD (p221). */
export const AID_ANOTHER_INCREMENT_PER_10_ABOVE = 1

/** Penalidade -5 sem ferramenta adequada (p221). */
export const TOOLS_MISSING_PENALTY = -5

/** Base 10 na fórmula de CD de resistência (p227). */
export const RESISTANCE_CD_BASE = 10

/** Sucesso automático no natural 20 (p221). */
export const NATURAL_20_ALWAYS_SUCCEEDS = true

/** Falha automática no natural 1 (p221). */
export const NATURAL_1_ALWAYS_FAILS = true

// ─── Helpers — CD por tier ──────────────────────────────────────────
/** CD por dificuldade da Tabela 5-1. */
export function difficultyCd(tier: CdDifficultyTier): number {
  return DIFFICULTY_CD[tier]
}

// ─── Helpers — CD de resistência (p227) ─────────────────────────────
/**
 * CD de resistência canônica (p227):
 * `CD = 10 + floor(nível / 2) + modificador do atributo`
 *
 * Exemplo verbatim livro: nobre nível 10 CAR 4 → CD 19 (10 + 5 + 4).
 */
export function resistanceCd(level: number, attributeModifier: number): number {
  if (level < 1) {
    throw new Error(`resistanceCd: level must be ≥ 1, got ${level}`)
  }
  return RESISTANCE_CD_BASE + Math.floor(level / 2) + attributeModifier
}

// ─── Helpers — Rolagem automática (p221) ────────────────────────────
/** Resolve 20 nat / 1 nat conforme p221. Retorna undefined se não se aplica. */
export function automaticOutcome(
  d20Roll: number,
): 'auto-success' | 'auto-failure' | undefined {
  if (d20Roll === 20) return 'auto-success'
  if (d20Roll === 1) return 'auto-failure'
  return undefined
}

// ─── Helpers — Escolher 0 / 10 / 20 (p222) ──────────────────────────
/**
 * Escolher 10: resultado = 10 + modificador total. Permitido sem pressão.
 * Este helper apenas calcula — checagem de precondição fica com o caller.
 */
export function takeTenResult(totalModifier: number): number {
  return TAKE_TEN_VALUE + totalModifier
}

/**
 * Escolher 20: resultado = 20 + modificador total. Leva 20× o tempo
 * normal. Não permitido se a falha tiver consequência.
 */
export function takeTwentyResult(totalModifier: number): number {
  return TAKE_TWENTY_VALUE + totalModifier
}

/** Tempo total ao Escolher 20 dado o tempo base do teste. */
export function takeTwentyTime(baseTimeUnits: number): number {
  if (baseTimeUnits <= 0) {
    throw new Error(
      `takeTwentyTime: baseTimeUnits must be > 0, got ${baseTimeUnits}`,
    )
  }
  return baseTimeUnits * TAKE_TWENTY_TIME_MULTIPLIER
}

/**
 * Escolher 0: se o bônus total já cobre a CD, passa automaticamente sem
 * rolar (p222). Rolar só faz sentido para tentar grau maior (arrisca 1
 * natural).
 */
export function takeZeroSucceeds(totalModifier: number, cd: number): boolean {
  return totalModifier >= cd
}

// ─── Helpers — Ajudar (p221) ────────────────────────────────────────
/**
 * Bônus concedido ao líder pelo ajudante:
 *  - roll < 10 (falha CD 10) → 0.
 *  - roll ≥ 10 → 1 + floor((roll - 10) / 10).
 *
 * Exemplos verbatim: natural 20 → +2; ≥30 → +3.
 */
export function aidAnotherBonus(helperRollResult: number): number {
  if (helperRollResult < AID_ANOTHER_CD) return 0
  const extra = Math.floor(
    (helperRollResult - AID_ANOTHER_CD) / 10,
  )
  return AID_ANOTHER_BASE_BONUS + extra
}

// ─── Helpers — Testes Estendidos (p223) ─────────────────────────────
/** Sucessos exigidos por complexidade. */
export function extendedSuccessesRequired(
  complexity: ExtendedComplexity,
): number {
  return EXTENDED_SUCCESSES_REQUIRED[complexity]
}

/**
 * Estado atual de teste estendido:
 *  - falhas ≥ 3 → failure.
 *  - sucessos ≥ exigidos → success.
 *  - caso contrário → in-progress.
 *
 * Empate (ambos limites atingidos na mesma rodada) resolve como
 * success — sucessos são contabilizados primeiro em grupo (p223).
 */
export function extendedTestState(
  successes: number,
  failures: number,
  complexity: ExtendedComplexity,
): ExtendedTestState {
  if (successes < 0 || failures < 0) {
    throw new Error(
      `extendedTestState: successes/failures must be ≥ 0 (got ${successes}, ${failures})`,
    )
  }
  const required = extendedSuccessesRequired(complexity)
  if (successes >= required) return 'success'
  if (failures >= EXTENDED_MAX_FAILURES) return 'failure'
  return 'in-progress'
}

/**
 * CD ajustada em teste estendido com dificuldades cumulativas (opcional):
 * CD original + 2 × número de testes já feitos (successes + failures).
 */
export function extendedCumulativeCd(
  baseCd: number,
  testsAlreadyMade: number,
): number {
  if (testsAlreadyMade < 0) {
    throw new Error(
      `extendedCumulativeCd: testsAlreadyMade must be ≥ 0, got ${testsAlreadyMade}`,
    )
  }
  return baseCd + EXTENDED_CUMULATIVE_CD_INCREMENT * testsAlreadyMade
}

/**
 * Penalidade cumulativa opcional após N falhas (p223):
 * cada falha impõe -2 aos testes seguintes.
 */
export function extendedCumulativePenalty(failureCount: number): number {
  if (failureCount < 0) {
    throw new Error(
      `extendedCumulativePenalty: failureCount must be ≥ 0, got ${failureCount}`,
    )
  }
  if (failureCount === 0) return 0
  return failureCount * EXTENDED_CUMULATIVE_FAILURE_PENALTY
}

// ─── Helpers — Ferramentas (p221) ───────────────────────────────────
/** Penalidade -5 se ferramenta ausente. */
export function toolsMissingPenalty(hasProperTools: boolean): number {
  return hasProperTools ? 0 : TOOLS_MISSING_PENALTY
}

// ─── Helpers — Testes Opostos (p220) ────────────────────────────────
export type OpposedTestOutcome = 'attacker-wins' | 'defender-wins' | 'reroll'

/**
 * Resolve teste oposto:
 *  - maior valor vence.
 *  - empate: maior modificador vence.
 *  - se ainda empatar, rerola (retorna 'reroll').
 *
 * `attackerRoll` = resultado total (d20+mod); `attackerModifier` = só o
 * modificador (para desempate).
 */
export function opposedTestOutcome(
  attackerRoll: number,
  attackerModifier: number,
  defenderRoll: number,
  defenderModifier: number,
): OpposedTestOutcome {
  if (attackerRoll > defenderRoll) return 'attacker-wins'
  if (attackerRoll < defenderRoll) return 'defender-wins'
  if (attackerModifier > defenderModifier) return 'attacker-wins'
  if (attackerModifier < defenderModifier) return 'defender-wins'
  return 'reroll'
}

// ─── Fórmulas de teste (p220) ────────────────────────────────────────
/**
 * Teste de atributo (p220 verbatim): `1d20 + Atributo`. Usado quando
 * nenhuma perícia se aplica (erguer objeto pesado com Força etc.).
 */
export function attributeTestResult(
  d20Roll: number,
  attributeValue: number,
): number {
  if (d20Roll < 1 || d20Roll > 20) {
    throw new Error(
      `attributeTestResult: d20Roll must be 1-20, got ${d20Roll}`,
    )
  }
  return d20Roll + attributeValue
}

/**
 * Teste de perícia (p220 verbatim): `1d20 + Valor de Perícia`. Funciona
 * como teste de atributo, mas usa o valor total da perícia (que já
 * inclui atributo-chave + treino + demais bônus).
 */
export function skillTestResult(
  d20Roll: number,
  skillValue: number,
): number {
  if (d20Roll < 1 || d20Roll > 20) {
    throw new Error(`skillTestResult: d20Roll must be 1-20, got ${d20Roll}`)
  }
  return d20Roll + skillValue
}

// ─── Auto-pass Fácil por decisão do mestre (p220 rodapé Tabela 5-1) ─
/**
 * Tarefas Fáceis (CD 5) aparecem na tabela para escala. Normalmente
 * não exigem teste — mestre pode conceder auto-pass para acelerar
 * o jogo. Distinto de Escolher 0 (que exige modificador ≥ CD).
 */
export const EASY_TASK_AUTO_PASS_ON_GM_DISCRETION = true

// ─── Testes Mistos (p220 — comum + oposto) ───────────────────────────
/**
 * Teste misto: todos rolam contra uma CD (todos os que passam
 * "atravessam o lago"); entre os aprovados, o maior valor é o
 * vencedor (chega primeiro).
 */
export type HybridTestOutcome = {
  passed: readonly number[]
  /**
   * Índice do maior valor dentre os aprovados; null se ninguém
   * passar.
   */
  winnerIndex: number | null
}

export function hybridTestOutcome(
  rolls: readonly number[],
  cd: number,
): HybridTestOutcome {
  const passed: number[] = []
  let winnerIndex: number | null = null
  let winnerRoll = -Infinity
  for (let i = 0; i < rolls.length; i++) {
    const roll = rolls[i]
    if (roll >= cd) {
      passed.push(i)
      if (roll > winnerRoll) {
        winnerRoll = roll
        winnerIndex = i
      }
    }
  }
  return { passed, winnerIndex }
}

// ─── Condições Favoráveis/Desfavoráveis (p221) ───────────────────────
/**
 * Circunstâncias favoráveis/desfavoráveis podem alterar o teste em
 * ±2 ou mais, à decisão do mestre.
 */
export const CIRCUMSTANCE_MIN_MAGNITUDE = 2

export type CircumstanceKind = 'favorable' | 'unfavorable'

/**
 * Aplica um modificador circunstancial ao total do teste. `magnitude`
 * deve ser ≥ CIRCUMSTANCE_MIN_MAGNITUDE; favoráveis somam, desfavoráveis
 * subtraem.
 */
export function applyCircumstance(
  totalBeforeCircumstance: number,
  kind: CircumstanceKind,
  magnitude: number,
): number {
  if (magnitude < CIRCUMSTANCE_MIN_MAGNITUDE) {
    throw new Error(
      `applyCircumstance: magnitude must be ≥ ${CIRCUMSTANCE_MIN_MAGNITUDE}, got ${magnitude}`,
    )
  }
  return kind === 'favorable'
    ? totalBeforeCircumstance + magnitude
    : totalBeforeCircumstance - magnitude
}

// ─── Novas Tentativas (p221 — retry framework) ───────────────────────
/**
 * Por padrão, testes podem ser refeitos em caso de falha. Alguns testes
 * têm consequência de falha (armadilha dispara, criatura cai) que o
 * caller registra.
 */
export const RETRIES_ALLOWED_BY_DEFAULT = true

/**
 * Falha "por 5 ou mais" pode disparar consequência catastrófica em
 * alguns testes (ex: Atletismo escalada). Padrão de margem de falha.
 */
export const CATASTROPHIC_FAILURE_MARGIN = 5

/**
 * Verifica se uma falha foi por margem catastrófica (≥ 5 de diferença).
 * `rollResult < cd` obrigatório (retorna false se passou).
 */
export function isCatastrophicFailure(rollResult: number, cd: number): boolean {
  if (rollResult >= cd) return false
  return cd - rollResult >= CATASTROPHIC_FAILURE_MARGIN
}

// ─── Testes Estendidos: extensões (p223) ─────────────────────────────
/**
 * Testes Estendidos Abertos (p223): cada teste avulso precisa ser
 * feito com uma perícia diferente. Jogadores escolhem a perícia por
 * teste e explicam como se aplica.
 */
export const EXTENDED_OPEN_REQUIRES_DIFFERENT_SKILL_PER_TEST = true

/**
 * Ajudar em teste estendido (p223): a perícia usada para ajudar não
 * pode mais ser usada no teste estendido (para ajudar de novo ou para
 * o teste principal).
 */
export const EXTENDED_AID_CONSUMES_SKILL = true

/**
 * Interrupção de teste estendido pode contar como falha simples ou
 * falha completa (à decisão do mestre).
 */
export const EXTENDED_INTERRUPT_GM_DECIDES = true

/**
 * Retorna true se a perícia proposta pode ser usada no estendido
 * aberto — regra: cada teste avulso precisa ser feito com perícia
 * diferente.
 */
export function extendedOpenSkillAllowed(
  proposedSkill: string,
  alreadyUsedSkills: readonly string[],
): boolean {
  if (!EXTENDED_OPEN_REQUIRES_DIFFERENT_SKILL_PER_TEST) return true
  return !alreadyUsedSkills.includes(proposedSkill)
}

/**
 * Ajudar em estendido: retorna true se a perícia do ajudante ainda
 * pode ser usada no estendido; false se já foi consumida (ajuda ou
 * teste principal).
 */
export function extendedAidSkillAllowed(
  aidSkill: string,
  alreadyUsedSkills: readonly string[],
): boolean {
  if (!EXTENDED_AID_CONSUMES_SKILL) return true
  return !alreadyUsedSkills.includes(aidSkill)
}

/**
 * Testes Estendidos em Grupo (p223): a cada "rodada", cada jogador
 * faz um teste. Soma sucessos e falhas do grupo inteiro para definir
 * estado (usa a mesma regra base `extendedTestState`).
 *
 * `roundResults[i]` = 'success' | 'failure' por jogador na rodada.
 */
export function extendedGroupRoundTally(
  roundResults: readonly ('success' | 'failure')[],
): { successes: number; failures: number } {
  let successes = 0
  let failures = 0
  for (const r of roundResults) {
    if (r === 'success') successes++
    else failures++
  }
  return { successes, failures }
}

/**
 * Testes Estendidos Multi-perícia (p223): definição de contagens por
 * perícia. Ex: {atletismo: 1, furtividade: 2} exige 1 sucesso em
 * Atletismo + 2 em Furtividade antes de 3 falhas totais.
 */
export type MultiSkillExtendedRequirement = Readonly<Record<string, number>>

export function multiSkillExtendedComplete(
  requirement: MultiSkillExtendedRequirement,
  successesBySkill: Readonly<Record<string, number>>,
): boolean {
  for (const skill of Object.keys(requirement)) {
    const need = requirement[skill]
    const got = successesBySkill[skill] ?? 0
    if (got < need) return false
  }
  return true
}
