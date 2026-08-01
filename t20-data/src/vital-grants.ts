/**
 * Passive max-PV/PM pipeline. Powers/traits that raise the permanent pool
 * ("+1 PM por nível", Anão "+3 PV +1/nível", "soma Sabedoria no PM total")
 * carry a `maxPv`/`maxPm` modifier with a `scale`; this module resolves the
 * abilities a character owns, evaluates each scale against level + attributes,
 * and returns the summed bonus for `computeVitals` to fold in.
 *
 * Only PERMANENT max-pool grants live here. Temporary/scene PV/PM (Alma de
 * Bronze, Missas, Estandarte…) are a separate expiring pool, not a max change.
 *
 * NOT yet covered: god-granted powers (devoto), e.g. Wynna's "Bênção do Mana"
 * (+1 PM a cada nível ímpar). `GrantedPower` carries no `modifiers` field and
 * devoto power selection isn't threaded into the sheet — deferred follow-up.
 */
import type { AttributeKey } from './attributes'
import { getOrigin, getRace, raceModifiers } from './abilities/catalog'
import { classPowerModifiers } from './abilities/classes'
import { getGeneralPower } from './abilities/general-powers'
import { originModifiers } from './abilities/origins'
import type { Modifier, VitalScale } from './items/types'
import { racaById } from './racas'

export type VitalGrantContext = {
  level: number
  className: string
  raceId?: string
  /** Chosen variant ids for race abilities (Suraggel ascendência etc.). */
  raceAbilityChoices?: readonly string[]
  /** Ids of picked class electives + general powers (character.classPowers). */
  powerIds?: readonly string[]
  /** Origin id/name (character.origin). */
  origin?: string
  /** Ids of picked origin benefits (character.originChoices). */
  originChoices?: readonly string[]
  /** Final attribute totals (post-race/buff), for `attribute`-scaled grants. */
  attrTotals: Record<AttributeKey, number>
}

export type VitalGrants = { pv: number; pm: number }

/** Evaluate one modifier's scaled amount. Omitted scale ⇒ flat. */
export function evalVitalScale(
  amount: number,
  scale: VitalScale | undefined,
  level: number,
  attrTotals: Record<AttributeKey, number>,
): number {
  if (!scale || scale.per === 'flat') return amount
  if (scale.per === 'level') return amount * level
  if (scale.per === 'levelStep') {
    const steps = level / scale.step
    return amount * (scale.round === 'up' ? Math.ceil(steps) : Math.floor(steps))
  }
  return amount * (attrTotals[scale.attribute] ?? 0)
}

/**
 * `raceId` is the racas.ts slug ('anao'); the abilities catalog keys races by
 * display name ('Anão'). Bridge slug → name, tolerating an unknown slug
 * (`racaById` throws) and an abilities-name passed straight through.
 */
function abilitiesRaceKey(raceId: string): string {
  try {
    return racaById(raceId).name
  } catch {
    return raceId
  }
}

/** All modifiers from the abilities the character owns (any target). */
function ownedModifiers(ctx: VitalGrantContext): Modifier[] {
  const out: Modifier[] = []
  const powers = new Set(ctx.powerIds ?? [])
  if (ctx.raceId) {
    const race = getRace(abilitiesRaceKey(ctx.raceId))
    if (race) out.push(...raceModifiers(race, new Set(ctx.raceAbilityChoices ?? [])))
  }
  // Auto-granted class powers fold in by className+level; electives need the id.
  out.push(...classPowerModifiers(ctx.className, ctx.level, powers))
  for (const id of powers) {
    const general = getGeneralPower(id)
    if (general?.modifiers) out.push(...general.modifiers)
  }
  if (ctx.origin) {
    const origin = getOrigin(ctx.origin)
    if (origin) out.push(...originModifiers(origin, new Set(ctx.originChoices ?? [])))
  }
  return out
}

/** Sum every `maxPv`/`maxPm` grant the character owns (untyped ⇒ stacks). */
export function collectVitalGrants(ctx: VitalGrantContext): VitalGrants {
  let pv = 0
  let pm = 0
  for (const m of ownedModifiers(ctx)) {
    if (m.target.k === 'maxPv') {
      pv += evalVitalScale(m.amount, m.scale, ctx.level, ctx.attrTotals)
    } else if (m.target.k === 'maxPm') {
      pm += evalVitalScale(m.amount, m.scale, ctx.level, ctx.attrTotals)
    }
  }
  return { pv, pm }
}
