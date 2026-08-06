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
 * God-granted powers (devoto) fold in via `ctx.godPower` (the chosen poder's
 * NAME) → `GrantedPower.modifiers` (e.g. Wynna's Bênção do Mana, +1 PM a
 * cada nível ímpar). Limitation: one poder concedido per character (single
 * `godPower` column) — Clérigo/Druida's "DOIS poderes concedidos" (p56/p60)
 * still collapses to one pick.
 */
import type { AttributeKey } from './attributes'
import { getOrigin, getRace, raceModifiers } from './abilities/catalog'
import {
  classPowerModifiers,
  type ClassChoiceSelections,
} from './abilities/classes'
import { getGeneralPower } from './abilities/general-powers'
import { grantedPowerByName } from './abilities/granted-powers'
import { originModifiers } from './abilities/origins'
import type { Modifier, VitalScale } from './items/types'
import { racaById } from './racas'

export type VitalGrantContext = {
  level: number
  className: string
  /** Full class list for multiclass — when present, class-power grants are
   *  collected per class (each at ITS level); `className`/`level` stay the
   *  single-class fallback. */
  classes?: readonly { className: string; level: number }[]
  raceId?: string
  /** Chosen variant ids for race abilities (Suraggel ascendência etc.). */
  raceAbilityChoices?: readonly string[]
  /** Ids of picked class electives + general powers (character.classPowers). */
  powerIds?: readonly string[]
  /** Per-class caminho/devoto picks (character.classChoices) — resolves
   *  grantedByChoice rows (Caminho do Arcanista → +atributo-chave no PM).
   *  Partial: parsed JSON blobs may hold undefined per-class entries. */
  classChoices?: Readonly<Partial<Record<string, ClassChoiceSelections>>>
  /** Poder concedido escolhido na devoção (character.godPower — NAME, o
   *  picker grava o nome do poder). Bênção do Mana → +1 PM/nível ímpar. */
  godPower?: string
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
  // Auto-granted class powers fold in by className+level; electives need the
  // id; caminho rows resolve via classChoices (grantedByChoice). Multiclass:
  // every class contributes at ITS class level (p35).
  const classEntries =
    ctx.classes && ctx.classes.length > 0
      ? ctx.classes
      : [{ className: ctx.className, level: ctx.level }]
  for (const c of classEntries) {
    out.push(
      ...classPowerModifiers(
        c.className,
        c.level,
        powers,
        ctx.classChoices?.[c.className],
      ),
    )
  }
  for (const id of powers) {
    const general = getGeneralPower(id)
    if (general?.modifiers) out.push(...general.modifiers)
  }
  if (ctx.godPower) {
    const granted = grantedPowerByName(ctx.godPower)
    if (granted?.modifiers) out.push(...granted.modifiers)
  }
  if (ctx.origin) {
    const origin = getOrigin(ctx.origin)
    if (origin) out.push(...originModifiers(origin, new Set(ctx.originChoices ?? [])))
  }
  return out
}

/**
 * Sum every `maxPv`/`maxPm` grant the character owns (untyped ⇒ stacks),
 * EXCEPT attribute-scaled grants of the same attribute+target, which apply
 * once — p225: "um personagem clérigo/druida não soma duas vezes sua
 * Sabedoria nos pontos de mana". Distinct scales (Poder Mágico per-level +
 * caminho attribute) still stack.
 */
export function collectVitalGrants(ctx: VitalGrantContext): VitalGrants {
  let pv = 0
  let pm = 0
  const seenAttrGrants = new Set<string>()
  for (const m of ownedModifiers(ctx)) {
    if (m.target.k !== 'maxPv' && m.target.k !== 'maxPm') continue
    if (m.scale?.per === 'attribute') {
      const key = `${m.target.k}:${m.scale.attribute}`
      if (seenAttrGrants.has(key)) continue
      seenAttrGrants.add(key)
    }
    const amount = evalVitalScale(m.amount, m.scale, ctx.level, ctx.attrTotals)
    if (m.target.k === 'maxPv') pv += amount
    else pm += amount
  }
  return { pv, pm }
}
