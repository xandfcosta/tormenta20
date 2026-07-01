/**
 * Redução de Dano (RD) — mecânica de subtração de dano.
 *
 * PDF refs:
 *  - RD core rules: Cap 5 p228-229 (definições Imunidade/Resistência/RD)
 *  - Order of operations: p225 (saves → halve → RD last)
 *  - Typed RD (redução de fogo/frio/corte): p230
 *  - Bárbaro/Guerreiro RD tables: p47/p56
 *  - Cavaleiro Bastião + Especialização (stack): p54-57
 *  - Adamante RD: p133
 *
 * Regras chave:
 *  - RD subtraída do dano FINAL por hit (após halving por save/hab).
 *  - `RD X/tipo` — tipos após barra bypassam a RD.
 *  - `redução de fogo N` — RD tipada; só aplica àquele damage type.
 *  - Imunidade = total (0 dano), separada de RD.
 *  - Perda de Vida bypassa RD (p228).
 *  - Sem regra "mínimo 1 dano" — RD pode zerar hit.
 *  - Múltiplas fontes empilham (soma).
 */

// ─── Types ──────────────────────────────────────────────────────────
export type RdSource =
  | {
      kind: 'flat'
      amount: number
      /** Tags/tipos que bypassam esta RD (ex: ['magico', 'luz']). */
      bypass?: readonly string[]
    }
  | {
      kind: 'typed'
      amount: number
      /** Só reduz dano deste tipo (ex: 'fogo', 'corte'). */
      damageType: string
    }

export type DamageResistanceProfile = {
  rdSources: readonly RdSource[]
  /** Damage types com imunidade total (dano vira 0). */
  immunities: readonly string[]
}

export type IncomingDamage = {
  amount: number
  damageType: string
  /** Tags do ataque (ex: ['magico'] pra arma mágica, ['luz'] pra magia divina). */
  tags?: readonly string[]
  /** Perda de Vida bypassa RD inteira. */
  isPerdaDeVida?: boolean
}

// ─── RD sum helpers ─────────────────────────────────────────────────
/**
 * Retorna total de RD aplicável a um hit, considerando bypass e tipo.
 * Ignora imunidade — caller checa antes.
 */
export function applicableRd(
  sources: readonly RdSource[],
  damageType: string,
  tags: readonly string[] = [],
): number {
  const tagSet = new Set(tags)
  let total = 0
  for (const src of sources) {
    if (src.kind === 'typed') {
      if (src.damageType === damageType) total += src.amount
      continue
    }
    // flat: aplica exceto se algum bypass tag presente
    const bypassed = (src.bypass ?? []).some((b) => tagSet.has(b))
    if (!bypassed) total += src.amount
  }
  return total
}

// ─── Damage application ────────────────────────────────────────────
/**
 * Aplica RD + imunidade a um hit. Retorna dano final (mín 0).
 *
 * Ordem:
 *  1. Perda de Vida → passa direto (ignora RD)
 *  2. Imunidade → 0
 *  3. RD → max(0, amount - applicable)
 */
export function applyDamageReduction(
  incoming: IncomingDamage,
  profile: DamageResistanceProfile,
): number {
  if (incoming.amount < 0) {
    throw new Error(
      `applyDamageReduction: amount must be ≥ 0, got ${incoming.amount}`,
    )
  }
  if (incoming.isPerdaDeVida) return incoming.amount
  if (profile.immunities.includes(incoming.damageType)) return 0
  const rd = applicableRd(profile.rdSources, incoming.damageType, incoming.tags)
  return Math.max(0, incoming.amount - rd)
}

// ─── Ignore-RD abilities ────────────────────────────────────────────
/**
 * Guerreiro Romper Resistências: paga 1 PM pra ignorar 10 pontos de RD.
 * Ladino Encontrar Fraqueza (7º) ignora RD de OBJETOS por completo —
 * caller passa `Infinity` pra `rdIgnored` nesse caso.
 */
export function applyDamageWithRdIgnore(
  incoming: IncomingDamage,
  profile: DamageResistanceProfile,
  rdIgnored: number,
): number {
  if (rdIgnored < 0) {
    throw new Error(
      `applyDamageWithRdIgnore: rdIgnored must be ≥ 0, got ${rdIgnored}`,
    )
  }
  if (incoming.isPerdaDeVida) return incoming.amount
  if (profile.immunities.includes(incoming.damageType)) return 0
  const rd = applicableRd(profile.rdSources, incoming.damageType, incoming.tags)
  const effectiveRd = Math.max(0, rd - rdIgnored)
  return Math.max(0, incoming.amount - effectiveRd)
}

// ─── Class RD tables ────────────────────────────────────────────────
/**
 * Bárbaro Redução de Dano (Tabela p47):
 *  - 5º: 2, 8º: 4, 11º: 6, 14º: 8, 17º: 10
 *  - Antes de 5º: 0
 */
export function barbaroRdForLevel(level: number): number {
  if (level < 1) {
    throw new Error(`barbaroRdForLevel: level must be ≥ 1, got ${level}`)
  }
  if (level >= 17) return 10
  if (level >= 14) return 8
  if (level >= 11) return 6
  if (level >= 8) return 4
  if (level >= 5) return 2
  return 0
}

/**
 * Guerreiro Redução de Dano — starts nível 5 com RD 2 (só em armadura
 * pesada), +2 cada 3 níveis. Max RD 10 no 17º.
 */
export function guerreiroRdForLevel(level: number, heavyArmor: boolean): number {
  if (level < 1) {
    throw new Error(`guerreiroRdForLevel: level must be ≥ 1, got ${level}`)
  }
  if (!heavyArmor) return 0
  if (level >= 17) return 10
  if (level >= 14) return 8
  if (level >= 11) return 6
  if (level >= 8) return 4
  if (level >= 5) return 2
  return 0
}

/** Cavaleiro Bastião (5º): RD 5 em armadura pesada. */
export const CAVALEIRO_BASTIAO_RD = 5

/**
 * Cavaleiro Especialização em Armadura (12º): RD 5 empilhável com
 * Bastião. Requer armadura pesada.
 */
export const CAVALEIRO_ESPECIALIZACAO_RD = 5

export function cavaleiroTotalRd(
  level: number,
  heavyArmor: boolean,
): number {
  if (!heavyArmor) return 0
  let total = 0
  if (level >= 5) total += CAVALEIRO_BASTIAO_RD
  if (level >= 12) total += CAVALEIRO_ESPECIALIZACAO_RD
  return total
}

// ─── Material RD constants ──────────────────────────────────────────
/** Adamante armadura leve/escudo (p133): RD 2. */
export const ADAMANTE_ARMADURA_LEVE_RD = 2

/** Adamante armadura pesada (p133): RD 5. */
export const ADAMANTE_ARMADURA_PESADA_RD = 5
