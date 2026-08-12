import { originGrant } from './grant-helpers'
import { origemRolledMoneySum } from './starting-equipment'
import type { CharacterFormValues } from './wizard-steps'

type OriginSwitchValues = Pick<
  CharacterFormValues,
  'origin' | 'originChoices' | 'originItemPicks' | 'powerChoices' | 'tibar'
>

export type OriginSwitchPatch = Pick<
  CharacterFormValues,
  'origin' | 'originChoices' | 'originItemPicks' | 'powerChoices'
> & { tibar: number }

/**
 * Everything that has to move when the player changes origin. Picking a new
 * origin invalidates the benefits, the item picks and the money the previous
 * one had already rolled (T$ 2d6 do último salário) — the roll is subtracted so
 * coin cannot leak from an origin the character no longer has.
 *
 * A patch rather than a set of writes: swapping origin is ONE decision, and
 * five separate `setValue` calls is five chances to forget the fifth.
 *
 * @example originSwitchPatch(draft.values, 'Batedor') // → aplicar com draft.patch()
 */
export function originSwitchPatch(
  values: OriginSwitchValues,
  nextOrigin: string,
): OriginSwitchPatch {
  const rolled = values.origin
    ? origemRolledMoneySum(values.origin, values.originItemPicks ?? {})
    : 0
  return {
    origin: nextOrigin,
    originChoices: [],
    originItemPicks: {},
    powerChoices: withoutOriginPicks(values.powerChoices ?? {}, values.origin),
    tibar: Math.max(0, (values.tibar ?? 0) - rolled),
  }
}

/**
 * Power picks are keyed by the benefit that granted them, so leaving an origin
 * must take its keys along — otherwise a power chosen for a benefit the
 * character no longer has rides silently into the create payload.
 */
function withoutOriginPicks(
  powerChoices: Record<string, string[]>,
  previousOrigin: string,
): Record<string, string[]> {
  const grant = previousOrigin ? originGrant(previousOrigin) : null
  if (!grant) return { ...powerChoices }
  const owned = new Set(grant.benefits.map((b) => b.id))
  if (grant.poderUnico) owned.add(grant.poderUnico.id)
  return Object.fromEntries(
    Object.entries(powerChoices).filter(([benefitId]) => !owned.has(benefitId)),
  )
}
