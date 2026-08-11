import type { Modifier } from '@tormenta20/t20-data'
import type {
  ActiveEffect,
  ApplyDamageResult,
  ApplyEffectResult,
  Character,
  DisplacedPool,
  ManualPoolCleared,
  PoolSuperseded,
} from '@/shared/api/api'
import { parseEffectModifiers } from './derived'
import { effectSourceName } from './effect-source'

/**
 * Pure temp-PV pool logic (Fase 4 + F2): the REAL pool persisted as
 * ActiveEffect rows with a `tempHp` modifier (Alma de Bronze grant, Campo de
 * Força, manual GM pool…). Damage routes through POST :id/damage server-side;
 * the functions here compute the optimistic prediction and reconcile the
 * cached Character from the server deltas.
 */

export type TempHpSlice = { effectId: number; amount: number; label: string }
export type TempHpPool = { total: number; slices: TempHpSlice[] }
export type TempHpDrainStep = { effectId: number; drained: number; remaining: number }

/** First tempHp modifier of an effect row — the slice the backend drains. */
function tempHpModifierOf(effect: ActiveEffect): Modifier | undefined {
  return parseEffectModifiers(effect.modifiers).find(
    (m) => m.target.k === 'tempHp',
  )
}

/**
 * Sum the character's persisted temp-PV pool from its active effects.
 *
 * @example tempHpPool(barbaro).total // 10 (Alma de Bronze ativa)
 */
export function tempHpPool(character: Character): TempHpPool {
  const slices: TempHpSlice[] = []
  for (const effect of character.activeEffects ?? []) {
    const modifier = tempHpModifierOf(effect)
    if (!modifier || modifier.amount <= 0) continue
    slices.push({
      effectId: effect.id,
      amount: modifier.amount,
      label: effectSourceName(effect.catalogId),
    })
  }
  return { total: slices.reduce((sum, s) => sum + s.amount, 0), slices }
}

/**
 * Pure damage routing: temp-PV soak first, remainder hits hpCurrent.
 * Healing (damage <= 0) never touches the pool.
 *
 * @example routeDamage(7, 5) // { toPool: 5, toHp: 2 }
 */
export function routeDamage(
  damage: number,
  pool: number,
): { toPool: number; toHp: number } {
  if (damage <= 0) return { toPool: 0, toHp: 0 }
  const toPool = Math.min(damage, Math.max(0, pool))
  return { toPool, toHp: damage - toPool }
}

/**
 * Split a pool debit across slices, highest pool first — mirrors the server's
 * drain order in POST :id/damage, so the optimistic patch matches the
 * response. (With vale-o-maior there is normally a single pool.)
 *
 * @example drainPlan([{ effectId: 1, amount: 4, label: 'x' }], 6)
 *          // [{ effectId: 1, drained: 4, remaining: 0 }]
 */
export function drainPlan(
  slices: readonly TempHpSlice[],
  amount: number,
): TempHpDrainStep[] {
  const steps: TempHpDrainStep[] = []
  let left = amount
  for (const slice of [...slices].sort((a, b) => b.amount - a.amount)) {
    if (left <= 0) break
    const drained = Math.min(left, slice.amount)
    steps.push({ effectId: slice.effectId, drained, remaining: slice.amount - drained })
    left -= drained
  }
  return steps
}

/** Rewrite the first tempHp modifier of a serialized blob with the new amount. */
function debitTempHpModifiers(raw: string, remaining: number): string {
  const modifiers = parseEffectModifiers(raw)
  const idx = modifiers.findIndex((m) => m.target.k === 'tempHp')
  if (idx < 0) return raw
  modifiers[idx] = { ...modifiers[idx], amount: remaining }
  return JSON.stringify(modifiers)
}

/** True when every modifier of the row targets tempHp (a pure pool row). */
function isPureTempHpRow(effect: ActiveEffect): boolean {
  const modifiers = parseEffectModifiers(effect.modifiers)
  return (
    modifiers.length > 0 && modifiers.every((m) => m.target.k === 'tempHp')
  )
}

/**
 * Optimistic cache patch mirroring the server rule: an emptied PURE pool row
 * is dropped, an emptied mixed buff (Heroísmo) keeps its other modifiers with
 * the pool rewritten to 0, partial drains rewrite the amount in place.
 */
export function applyDrainToEffects(
  effects: readonly ActiveEffect[],
  plan: readonly TempHpDrainStep[],
): ActiveEffect[] {
  const byId = new Map(plan.map((s) => [s.effectId, s]))
  const out: ActiveEffect[] = []
  for (const effect of effects) {
    const step = byId.get(effect.id)
    if (!step) {
      out.push(effect)
      continue
    }
    if (step.remaining <= 0 && isPureTempHpRow(effect)) continue
    out.push({
      ...effect,
      modifiers: debitTempHpModifiers(effect.modifiers, step.remaining),
    })
  }
  return out
}

export function isPoolSuperseded(
  result: ApplyEffectResult,
): result is PoolSuperseded {
  return 'superseded' in result
}

export function isManualPoolCleared(
  result: ApplyEffectResult,
): result is ManualPoolCleared {
  return 'cleared' in result
}

/** Drop removed rows / zero kept rows for pools displaced under vale-o-maior. */
function applyDisplacedToEffects(
  effects: readonly ActiveEffect[],
  displaced: readonly DisplacedPool[],
): ActiveEffect[] {
  const byId = new Map(displaced.map((d) => [d.effectId, d]))
  const out: ActiveEffect[] = []
  for (const effect of effects) {
    const hit = byId.get(effect.id)
    if (!hit) {
      out.push(effect)
      continue
    }
    if (hit.removed) continue
    out.push({ ...effect, modifiers: debitTempHpModifiers(effect.modifiers, 0) })
  }
  return out
}

/** Replace/insert an upserted effect row (server upserts on catalogId+scope). */
function upsertEffectRow(
  effects: readonly ActiveEffect[],
  effect: ActiveEffect,
): ActiveEffect[] {
  const others = effects.filter(
    (e) =>
      e.id !== effect.id &&
      !(e.catalogId === effect.catalogId && e.scope === effect.scope),
  )
  return [...others, effect]
}

/**
 * Merge a POST :id/active-effects result into the cached character — handles
 * all four outcomes (plain row, pool applied + displaced, superseded no-op,
 * manual pool cleared).
 *
 * @example qc.setQueryData(key, (prev) => prev && applyPoolResult(prev, result))
 */
export function applyPoolResult(
  prev: Character,
  result: ApplyEffectResult,
): Character {
  if (isPoolSuperseded(result)) return prev
  if (isManualPoolCleared(result)) {
    const removed = new Set(result.removedEffectIds)
    return {
      ...prev,
      activeEffects: prev.activeEffects.filter((e) => !removed.has(e.id)),
    }
  }
  const applied = 'effect' in result ? result : { effect: result, displaced: [] }
  return {
    ...prev,
    activeEffects: upsertEffectRow(
      applyDisplacedToEffects(prev.activeEffects, applied.displaced),
      applied.effect,
    ),
  }
}

/**
 * Reconcile the cached character from the POST :id/damage response — the
 * server is authoritative: hpCurrent replaces the optimistic value and each
 * drained pool is dropped (removed) or rewritten to `newAmount`.
 */
export function reconcileDamageResult(
  prev: Character,
  result: ApplyDamageResult,
): Character {
  let effects: readonly ActiveEffect[] = prev.activeEffects
  for (const step of result.drained) {
    effects = step.removed
      ? effects.filter((e) => e.id !== step.effectId)
      : effects.map((e) =>
          e.id === step.effectId
            ? { ...e, modifiers: debitTempHpModifiers(e.modifiers, step.newAmount) }
            : e,
        )
  }
  return { ...prev, hpCurrent: result.hpCurrent, activeEffects: [...effects] }
}
