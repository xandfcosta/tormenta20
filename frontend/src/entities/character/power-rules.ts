/**
 * The RULES half of the React app's `use-power-action.ts`: deciding whether a
 * power can be used, what scope limits it, resolving its activation spec and
 * reading a stance's flag group. All framework-free — the React file mixed
 * these with a `useMutation` hook, so a caller that only wanted to ask "can
 * this be used?" had to drag React Query in with it.
 *
 * The acting half (paying PM, bumping counters, flipping stance flags) lands
 * with the Poderes block (ALE-87) as a Solid primitive.
 */

import { FLAG_ACTIVATIONS } from '@/shared/rules/flag-activations'
import { maxStepsForLevel } from '@/shared/rules/power-activation'
import type { ActivationSpec } from '@/shared/api/catalog-types'
import {
  activationSpecs,
  findActivationByName,
  getActivation,
} from '@/shared/lib/activation-cache'
import type { Character } from '@/shared/api/api'
import { parseChoiceSet } from './derived'

/** Scene- or day-limited uses; the counter store lands with ALE-87. */
export type PowerUseScope = 'scene' | 'day'

export type PowerUseDecision = { ok: boolean; reason?: string }

export type PowerUseContext = {
  mpCurrent: number
  usedScene: number
  usedDay: number
  activeFlags: ReadonlySet<string>
}

/**
 * Can this INSTANT power fire right now? Pure so the button/disabled logic is
 * unit-testable without the hook. 'rodada' and numeric limits ship unenforced
 * (badge-only) — the table tracks rounds, the sheet doesn't.
 *
 * @example powerUseDecision(spec, { mpCurrent: 0, usedScene: 0, usedDay: 0, activeFlags: new Set() })
 *          // { ok: false, reason: 'PM insuficiente' }
 */
export function powerUseDecision(
  spec: ActivationSpec,
  ctx: PowerUseContext,
): PowerUseDecision {
  if (spec.pmCost === 'variavel') return { ok: false, reason: 'custo variável' }
  if (spec.requiresFlag && !ctx.activeFlags.has(spec.requiresFlag)) {
    return { ok: false, reason: `requer ${spec.requiresFlag}` }
  }
  if (spec.uses === 'cena' && ctx.usedScene >= 1) {
    return { ok: false, reason: 'limite por cena atingido' }
  }
  if (spec.uses === 'dia' && ctx.usedDay >= 1) {
    return { ok: false, reason: 'limite por dia atingido' }
  }
  if (spec.pmCost > ctx.mpCurrent) return { ok: false, reason: 'PM insuficiente' }
  return { ok: true }
}

/** Store scope for a limited spec, or undefined when the limit is unenforced. */
export function enforcedScopeOf(
  spec: ActivationSpec,
): PowerUseScope | undefined {
  if (spec.uses === 'cena') return 'scene'
  if (spec.uses === 'dia') return 'day'
  return undefined
}

/**
 * Registry lookup for an ability row: exact id first (class powers already use
 * the `class.<x>.<slug>` convention), then name fallback for surfaces whose
 * local ids don't follow it (race `humano-versatil`, origin `poder-sortudo`).
 *
 * @example resolveActivationSpec('Golpe Poderoso', 'class.barbaro.golpe-poderoso')
 */
export function resolveActivationSpec(
  name: string,
  id?: string,
): ActivationSpec | undefined {
  const byId = id ? getActivation(id) : undefined
  if (byId) return byId
  const byName = findActivationByName(name)
  if (byName) return byName
  // Tier autos ("Inspiração +1", "Fúria +3") share one stance spec named
  // without the tier suffix — retry stripped so every tier row resolves to
  // the same activation instead of falling through as a silent passive.
  const tierless = name.replace(/\s\+\d+$/, '')
  return tierless !== name ? findActivationByName(tierless) : undefined
}

/**
 * The flag a stance spec raises when activated. Stance specs don't carry their
 * flag (only `requiresFlag` for triggered powers), but every stance is born
 * from a FLAG_ACTIVATIONS entry — match back by name.
 *
 * @example stanceFlagOf(getActivation('class.barbaro.furia')!) // 'furia'
 */
export function stanceFlagOf(spec: ActivationSpec): string | undefined {
  return Object.values(FLAG_ACTIVATIONS).find((a) => a.name === spec.name)?.flag
}

/** Accent/case-insensitive slug so 'Bárbaro' matches the id segment 'barbaro'. */
function classNameSlug(name: string): string {
  return name
    .normalize('NFD')
    .replace(/[\u0300-\u036f]/g, '')
    .toLowerCase()
    .replace(/\s+/g, '-')
}

/**
 * Level in the class that OWNS the stance (p40: multiclass counts only the
 * Bárbaro levels for Fúria's scaling). 0 for non-class specs / class not taken.
 *
 * @example stanceClassLevel(furiaSpec, [{ className: 'Bárbaro', level: 6 }]) // 6
 */
export function stanceClassLevel(
  spec: ActivationSpec,
  classes: readonly { className: string; level: number }[],
): number {
  const [scope, owner] = spec.id.split('.')
  if (scope !== 'class') return 0
  return classes.find((c) => classNameSlug(c.className) === owner)?.level ?? 0
}

/**
 * Stepper ceiling for a scaling stance — 0 when the spec doesn't scale.
 *
 * @example stanceMaxSteps(furiaSpec, [{ className: 'Bárbaro', level: 6 }]) // 1
 */
export function stanceMaxSteps(
  spec: ActivationSpec,
  classes: readonly { className: string; level: number }[],
): number {
  if (!spec.scaling) return 0
  return maxStepsForLevel(spec.scaling, stanceClassLevel(spec, classes))
}

/**
 * Total PM to enter the stance with N extra steps. Non-scaling stances cost
 * their flat pmCost; a 'variavel' cost yields NaN so the decision refuses it.
 *
 * @example stanceTotalPm(furiaSpec, 1) // 3 (base 2 + 1×1)
 */
export function stanceTotalPm(spec: ActivationSpec, steps: number): number {
  if (spec.scaling) return spec.scaling.basePm + steps * spec.scaling.stepPm
  return spec.pmCost === 'variavel' ? Number.NaN : spec.pmCost
}

/**
 * Can the stance be entered with these steps and this pool? Pure so the
 * dialog's disabled state and the hook share one gate.
 *
 * @example stanceActivationDecision(furiaSpec, 1, 2) // { ok: false, reason: 'PM insuficiente' }
 */
export function stanceActivationDecision(
  spec: ActivationSpec,
  steps: number,
  mpCurrent: number,
): PowerUseDecision {
  const total = stanceTotalPm(spec, steps)
  if (Number.isNaN(total)) return { ok: false, reason: 'custo variável' }
  if (total > mpCurrent) return { ok: false, reason: 'PM insuficiente' }
  return { ok: true }
}

/**
 * OWNED powers whose grant fires with this stance flag (Fase 4): Alma de
 * Bronze materializes its temp-PV pool when Fúria switches on. Ownership
 * checks classPowers, tolerating bare slugs and full convention ids.
 *
 * @example grantPowersForFlag(barbaro, 'furia').map((s) => s.id)
 *          // ['class.barbaro.alma-de-bronze']
 */
export function grantPowersForFlag(
  character: Character,
  flag: string,
): ActivationSpec[] {
  const chosen = parseChoiceSet(character.classPowers)
  return activationSpecs().filter((spec) => {
    if (spec.requiresFlag !== flag || !spec.grant) return false
    const tail = spec.id.split('.').pop() ?? spec.id
    if (chosen.has(spec.id) || chosen.has(tail)) return true
    return [...chosen].some((id) => id.endsWith(`.${tail}`))
  })
}

