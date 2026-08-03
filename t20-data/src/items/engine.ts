import type { BonusType, Modifier, ModifierTarget } from './types'

export type Contribution = {
  source: string
  bonusType: BonusType
  amount: number
  /** The modifier's own `note`, when present — the WHY behind the number
   *  ("desbalanceada: -2 em ataque"). Breakdown dialogs render it under
   *  the source so a row explains itself. */
  note?: string
}

export type AggregatedStat = {
  total: number
  contributions: Contribution[]
}

export type ConditionalEffect = {
  source: string
  bonusType: BonusType
  amount: number
  /** human-readable condition note ("terreno compatível", "ao usar manobra X") */
  note: string
  target: ModifierTarget
  /** Present when the condition is a `flagOn`; UI groups all conditionals
   *  sharing the same `flag` under a single toggle. */
  flag?: string
}

export type ItemEffects = {
  /** Aggregated numeric modifiers keyed by target identity. */
  byTarget: Record<string, AggregatedStat>
  /** Set of active flag names ("lethal-unarmed", "fatigue-on-sleep", ...). */
  flags: Set<string>
  /** Modifiers requiring player choice / situational opt-in. */
  conditional: ConditionalEffect[]
}

export type ActiveItem = {
  source: string
  /** Wear state declared by the player. */
  equipped: 'vested' | 'wielded' | 'wielded2' | null
  modifiers: Modifier[]
}

/**
 * Stable identity for a modifier target — used to bucket modifiers for
 * non-stacking resolution.
 */
export function targetKey(t: ModifierTarget): string {
  switch (t.k) {
    case 'expertise':
      return `expertise:${t.name}`
    case 'expertiseAll':
      return 'expertiseAll'
    case 'expertiseRemovePenalty':
      return `expertiseRemovePenalty:${t.name}`
    case 'expertiseByAttribute':
      return `expertiseByAttribute:${t.attribute}`
    case 'attribute':
      return `attribute:${t.name}`
    case 'defense':
      return 'defense'
    case 'defenseDexCap':
      return 'defenseDexCap'
    case 'resistance':
      return 'resistance'
    case 'fearResistance':
      return 'fearResistance'
    case 'attack':
      return `attack:${t.scope}`
    case 'damage':
      return `damage:${t.scope}`
    case 'critRange':
      return 'critRange'
    case 'critMult':
      return 'critMult'
    case 'pmLimit':
      return 'pmLimit'
    case 'pmCost':
      return 'pmCost'
    case 'catalyst':
      return `catalyst:${t.school}`
    case 'spellDC':
      return 'spellDC'
    case 'inventorySlots':
      return 'inventorySlots'
    case 'displacement':
      return 'displacement'
    case 'flySpeed':
      return 'flySpeed'
    case 'armorPenalty':
      return 'armorPenalty'
    case 'armorPenaltyExpertises':
      return 'armorPenaltyExpertises'
    case 'tempHp':
      return 'tempHp'
    case 'tempMp':
      return 'tempMp'
    case 'maxPv':
      return 'maxPv'
    case 'maxPm':
      return 'maxPm'
    case 'maneuver':
      return `maneuver:${t.name}`
    case 'flag':
      return `flag:${t.name}`
  }
}

function isUnconditional(m: Modifier): boolean {
  if (!m.condition) return true
  switch (m.condition.c) {
    case 'always':
    case 'wielded':
    case 'vested':
      return true
    case 'terrain':
    case 'against':
    case 'context':
    case 'flagOn':
      return false
    case 'flagOff':
      // Auto-evaluated against the collected flags in the main pass —
      // never a user-toggled conditional.
      return true
  }
}

function conditionMet(
  m: Modifier,
  equipped: ActiveItem['equipped'],
): boolean {
  if (!m.condition || m.condition.c === 'always') return true
  if (m.condition.c === 'wielded')
    return equipped === 'wielded' || equipped === 'wielded2'
  if (m.condition.c === 'vested') return equipped === 'vested'
  // flagOff passed the flags gate in the main loop — treat as met here.
  if (m.condition.c === 'flagOff') return true
  return false
}

function describeCondition(m: Modifier): string {
  if (!m.condition) return ''
  switch (m.condition.c) {
    case 'terrain':
      return `terreno: ${m.condition.type}`
    case 'against':
      return `contra: ${m.condition.trait}`
    case 'context':
      return m.condition.note
    case 'flagOn':
    case 'flagOff':
      return m.condition.label
    default:
      return ''
  }
}

/**
 * T20 non-stacking rules:
 *  - For a given target, modifiers sharing the same `bonusType` keep only
 *    the entry with the highest absolute amount (positive or negative).
 *  - 'untyped' modifiers stack freely with each other and with typed bonuses.
 *  - Result is the sum of one representative per typed bucket plus all
 *    untyped entries.
 */
function resolveStack(contribs: Contribution[]): AggregatedStat {
  const byType = new Map<BonusType, Contribution[]>()
  for (const c of contribs) {
    const arr = byType.get(c.bonusType)
    if (arr) arr.push(c)
    else byType.set(c.bonusType, [c])
  }

  const kept: Contribution[] = []
  for (const [type, list] of byType) {
    if (type === 'untyped') {
      kept.push(...list)
      continue
    }
    let best = list[0]
    for (const entry of list) {
      if (Math.abs(entry.amount) > Math.abs(best.amount)) best = entry
    }
    kept.push(best)
  }

  const total = kept.reduce((s, c) => s + c.amount, 0)
  return { total, contributions: kept }
}

/**
 * Non-stacking resolution for DISPLAYING a set of conditional effects (an
 * active stance's rows): buckets by target identity, runs the same
 * per-bonusType resolution as the main aggregation, and returns only the
 * surviving `{target, amount}` rows — so a Bárbaro 6's Fúria shows the +3
 * tier without the superseded +2 duplicates.
 *
 * @example
 * resolveConditionalDisplay([
 *   { target: { k: 'attack', scope: 'all' }, bonusType: 'morale', amount: 2 },
 *   { target: { k: 'attack', scope: 'all' }, bonusType: 'morale', amount: 3 },
 * ]) // → [{ target: { k: 'attack', scope: 'all' }, amount: 3 }]
 */
export function resolveConditionalDisplay(
  effects: { target: ModifierTarget; bonusType: BonusType; amount: number }[],
): { target: ModifierTarget; amount: number }[] {
  const byKey = new Map<string, { target: ModifierTarget; contribs: Contribution[] }>()
  for (const e of effects) {
    const key = targetKey(e.target)
    const bucket = byKey.get(key) ?? { target: e.target, contribs: [] }
    bucket.contribs.push({ source: '', bonusType: e.bonusType, amount: e.amount })
    byKey.set(key, bucket)
  }
  const kept: { target: ModifierTarget; amount: number }[] = []
  for (const { target, contribs } of byKey.values()) {
    for (const c of resolveStack(contribs).contributions) {
      kept.push({ target, amount: c.amount })
    }
  }
  return kept
}

export function computeItemEffects(items: ActiveItem[]): ItemEffects {
  const buckets: Record<string, Contribution[]> = {}
  const flags = new Set<string>()
  const conditional: ConditionalEffect[] = []

  // Pre-pass: collect flags from every equipped item FIRST, so auto-evaluated
  // flag conditions (flagOff) don't depend on item iteration order (the armor
  // may come after the power that reads its flag).
  for (const item of items) {
    if (item.equipped === null) continue
    for (const m of item.modifiers) {
      if (m.target.k === 'flag' && conditionMet(m, item.equipped)) {
        flags.add(m.target.name)
      }
    }
  }

  for (const item of items) {
    if (item.equipped === null) continue
    for (const m of item.modifiers) {
      // flagOff: book-passive that switches off while the flag is set
      // (Pele de Ferro vs armadura pesada) — resolved here, not user-toggled.
      if (m.condition?.c === 'flagOff' && flags.has(m.condition.flag)) continue
      if (!isUnconditional(m)) {
        conditional.push({
          source: item.source,
          bonusType: m.bonusType,
          amount: m.amount,
          note: describeCondition(m) || (m.note ?? ''),
          target: m.target,
          ...(m.condition?.c === 'flagOn' ? { flag: m.condition.flag } : {}),
        })
        continue
      }
      if (!conditionMet(m, item.equipped)) continue

      if (m.target.k === 'flag') {
        flags.add(m.target.name)
        continue
      }

      const key = targetKey(m.target)
      const list = buckets[key] ?? (buckets[key] = [])
      list.push({
        source: item.source,
        bonusType: m.bonusType,
        amount: m.amount,
        ...(m.note ? { note: m.note } : {}),
      })
    }
  }

  const byTarget: Record<string, AggregatedStat> = {}
  for (const key of Object.keys(buckets)) {
    byTarget[key] = resolveStack(buckets[key])
  }

  return { byTarget, flags, conditional }
}

export function statFor(
  effects: ItemEffects,
  target: ModifierTarget,
): AggregatedStat {
  const stat = effects.byTarget[targetKey(target)]
  if (stat) return stat
  return { total: 0, contributions: [] }
}

/**
 * Stable identifier for a single conditional effect — used to persist
 * which conditional opt-ins are currently toggled on by the player.
 */
export function conditionalId(c: ConditionalEffect): string {
  return [
    c.source,
    targetKey(c.target),
    c.note,
    c.amount,
    c.bonusType,
  ].join('::')
}

/**
 * Fold the conditional effects whose ids appear in `activeIds` into
 * `effects.byTarget`, re-running non-stacking resolution per target.
 * Flag conditionals are ignored (no UI for opt-in flags yet).
 */
export function applyActiveConditionals(
  effects: ItemEffects,
  activeIds: ReadonlySet<string>,
): ItemEffects {
  if (activeIds.size === 0) return effects
  const buckets: Record<string, Contribution[]> = {}
  for (const [key, agg] of Object.entries(effects.byTarget)) {
    buckets[key] = [...agg.contributions]
  }
  const remaining: ConditionalEffect[] = []
  for (const c of effects.conditional) {
    if (!activeIds.has(conditionalId(c))) {
      remaining.push(c)
      continue
    }
    if (c.target.k === 'flag') continue
    const key = targetKey(c.target)
    const list = buckets[key] ?? (buckets[key] = [])
    list.push({
      source: `${c.source} (cond.)`,
      bonusType: c.bonusType,
      amount: c.amount,
      ...(c.note ? { note: c.note } : {}),
    })
  }
  const byTarget: Record<string, AggregatedStat> = {}
  for (const k of Object.keys(buckets)) {
    byTarget[k] = resolveStack(buckets[k])
  }
  return { byTarget, flags: effects.flags, conditional: remaining }
}
