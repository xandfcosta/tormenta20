/**
 * Combate — PDF book p230-235.
 *
 * Core resolution rules pinned here:
 *
 *  - Ataque: `1d20 + bônus de ataque ≥ Defesa do alvo` → acerta. (p230)
 *  - Acerto crítico: rolling within the weapon's *margem* (default 20)
 *    on a hit is automatically a crit — **no confirmation roll**. (p231)
 *  - Dano de crítico: `dado da arma × multiplicador` + non-doubled
 *    bonuses (atributo, dados extras como Ataque Furtivo). (p231)
 *  - Dano corpo a corpo / arremesso: `arma + Força`. Dano à distância:
 *    `arma` apenas (sem Força). (p230)
 *  - Iniciativa: rolled once per combat; order stays the same.
 *    Tie-break: highest Iniciativa value, then reroll between tied. (p231)
 *  - Action economy: 1 padrão + 1 movimento por turno; padrão pode ser
 *    trocada por movimento, NÃO o inverso. (p233)
 *  - Investida (full-round): up to 2× deslocamento em linha reta, +2
 *    no ataque, -2 na Defesa até o próximo turno. (p235)
 */

/** Bônus de ataque corpo a corpo + Investida. */
export const INVESTIDA_ATTACK_BONUS = 2

/** Penalidade na Defesa durante o turno após uma Investida. */
export const INVESTIDA_DEFENSE_PENALTY = -2

export type ActionKind = 'padrao' | 'movimento' | 'livre' | 'reacao' | 'completa'

/**
 * Does the d20 attack roll + bonus meet or exceed the defender's Defesa?
 * PDF: "Se o resultado é igual ou maior que a Defesa do alvo, você
 * acerta." (p230)
 */
export function isAttackHit(
  d20Result: number,
  attackBonus: number,
  defense: number,
): boolean {
  if (d20Result < 1 || d20Result > 20) return false
  return d20Result + attackBonus >= defense
}

/**
 * Is the hit also a critical?
 *
 * PDF p231: "Você faz um acerto crítico quando acerta um ataque rolando
 * um valor igual ou maior que a margem de ameaça do ataque." Note this
 * checks the **raw d20 result**, not the modified total. T20 has no
 * confirmation re-roll.
 *
 * Default weapon critRange is 20 (no margem column on weapon → only a
 * natural 20 crits).
 */
export function isCriticalHit(
  d20Result: number,
  weaponCritRange: number,
  hit: boolean,
): boolean {
  if (!hit) return false
  return d20Result >= weaponCritRange
}

/**
 * Resolve damage for a single attack. Multiplies the weapon dice (and
 * weapon dice ONLY) by the critical multiplier when `isCrit` is true.
 * `attributeBonus` and `extraDice` are PDF: "Bônus numéricos de dano,
 * assim como dados extras (como pela habilidade Ataque Furtivo) não são
 * multiplicados." (p231)
 */
export function damageTotal(args: {
  weaponDice: number
  attributeBonus: number
  extraDice: number
  isCrit: boolean
  critMultiplier: number
}): number {
  const { weaponDice, attributeBonus, extraDice, isCrit, critMultiplier } = args
  const multiplied = isCrit ? weaponDice * critMultiplier : weaponDice
  return Math.max(0, multiplied + attributeBonus + extraDice)
}

/**
 * Base weapon-damage bonus from the attacker's attribute, given the
 * weapon's purpose. PDF p230: corpo a corpo / arremesso → Força; disparo
 * → sem bônus.
 */
export function attackAttributeBonus(
  weaponPurpose: 'melee' | 'thrown' | 'ranged',
  attackerStrength: number,
): number {
  if (weaponPurpose === 'ranged') return 0
  return attackerStrength
}

/**
 * Iniciativa tie-break per PDF p231. Returns the **index** in the
 * `entries` array that goes first, or `null` when both characters have
 * identical totals AND identical Iniciativa modifiers — caller rerolls
 * Iniciativa between them.
 *
 * `entries[i].total` is the rolled Iniciativa result.
 * `entries[i].modifier` is the static bonus used to break tied totals.
 */
export function initiativeOrder<T extends { total: number; modifier: number }>(
  entries: readonly T[],
): T[] {
  return [...entries].sort((a, b) => {
    if (b.total !== a.total) return b.total - a.total
    if (b.modifier !== a.modifier) return b.modifier - a.modifier
    return 0
  })
}

/**
 * Action-economy validator. PDF p233: "Você pode trocar sua ação padrão
 * por uma ação de movimento, para fazer duas ações de movimento, mas
 * não pode fazer o inverso." Ação completa consumes both. Reactions and
 * free actions live outside the budget.
 */
export function isLegalTurn(used: {
  padrao: number
  movimento: number
  completa: number
}): boolean {
  const { padrao, movimento, completa } = used
  if (padrao < 0 || movimento < 0 || completa < 0) return false
  if (completa > 1) return false
  if (completa === 1) {
    // Ação completa consumes the standard + move budget entirely.
    return padrao === 0 && movimento === 0
  }
  // Without ação completa: 1 padrão + 1 movimento max; trading padrão
  // for movimento allows 0 padrão + 2 movimento.
  if (padrao > 1 || movimento > 2) return false
  if (padrao === 1 && movimento > 1) return false
  return true
}
