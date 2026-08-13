import { totalSlots } from '@/entities/character/class-powers'
import type { CreateCharacterInput, RaceAttributeChoicesInput } from '@/shared/api/types'
import { totalClassLevel } from './class-entries'
import { deriveDraftVitals } from './draft-vitals'
import { type RaceChoiceState, deformidadePayload } from './grant-helpers'
import {
  purchasesPayload,
  purchasesTotal,
  startingItemsPayload,
  startingLoadout,
} from './starting-equipment'
import type { CharacterFormValues } from './wizard-steps'

/**
 * The draft as the create body the Go handler expects
 * (`api/character_create.go`). Everything the wizard tracks only for its own
 * sake — the kit pickers, the shop, the "already rolled" flag — is turned into
 * `items[]` and a remaining purse, then dropped.
 *
 * The pools are RECOMPUTED here rather than read off the draft: the last screen
 * may have changed a level, and the character that gets saved must agree with
 * the sheet that opens next. (The server heals vitals again after the insert —
 * this only keeps the two from disagreeing on the way in.)
 *
 * @example api.characters.create(createCharacterPayload(draft.values, draft.raceChoices))
 */
export function createCharacterPayload(
  values: CharacterFormValues,
  raceChoices: RaceChoiceState,
): CreateCharacterInput & {
  items: NonNullable<CreateCharacterInput['items']>
  raceAttributeChoices: RaceAttributeChoicesInput
  secondaryRaceChoices: NonNullable<CreateCharacterInput['secondaryRaceChoices']>
} {
  const {
    startingWeaponSimple,
    startingWeaponMartial,
    startingArmor,
    startingShield,
    startingPurchases,
    originItemPicks,
    startingMoneyRolled: _rolled,
    ...rest
  } = values

  const level = totalClassLevel(values.classes) || 1
  const primary = values.classes[0]?.className
  const purchases = startingPurchases ?? {}
  const { pvMax, pmMax } = deriveDraftVitals(values, raceChoices)

  const kitItems = primary
    ? startingItemsPayload(
        {
          weaponSimple: startingWeaponSimple ?? '',
          weaponMartial: startingWeaponMartial ?? '',
          armor: startingArmor ?? '',
          shield: startingShield ?? false,
        },
        startingLoadout(primary, level).kit,
        values.origin,
        originItemPicks ?? {},
      )
    : []

  return {
    ...rest,
    items: [...kitItems, ...purchasesPayload(purchases)],
    // What is SAVED is the money left: the starting purse bought the items
    // (p140), so keeping the full amount would mint coin.
    tibar: roundTibar(Math.max(0, (values.tibar ?? 0) - purchasesTotal(purchases))),
    hpMax: pvMax,
    mpMax: pmMax,
    hpCurrent: Math.min(values.hpCurrent, pvMax),
    mpCurrent: Math.min(values.mpCurrent, pmMax),
    // Last guard: never save more elective powers than the levels earn. Covers
    // lowering the level after the Poderes step and then skipping back past it.
    classPowers: values.classPowers.slice(0, totalSlots(values.classes)),
    god: values.god || undefined,
    // A poder concedido only means anything with a god behind it (p96).
    godPower: values.god && values.godPower ? values.godPower : undefined,
    raceAttributeChoices: primaryRaceChoices(values.races, raceChoices),
    secondaryRaceChoices: optedInSecondaryRaces(values.races, raceChoices),
  }
}

/** Centavo-safe: T$ is stored with two decimals and 0.1+0.2 is not 0.3. */
function roundTibar(value: number): number {
  return Math.round(value * 100) / 100
}

function primaryRaceChoices(
  races: string[],
  raceChoices: RaceChoiceState,
): RaceAttributeChoicesInput {
  const primary = races[0]
  const choice = primary ? raceChoices[primary] : undefined
  return {
    floatingPicks: choice?.floatingPicks ?? [],
    ascendencia: choice?.ascendencia,
    deformidade: primary ? deformidadePayload(primary, choice) : undefined,
  }
}

/**
 * Secondary races contribute only once the player opted in — a second lineage
 * is flavor the table agreed to make mechanical, never an automatic bonus.
 */
function optedInSecondaryRaces(
  races: string[],
  raceChoices: RaceChoiceState,
): NonNullable<CreateCharacterInput['secondaryRaceChoices']> {
  return races
    .slice(1)
    .filter((race) => raceChoices[race]?.applied)
    .map((race) => ({
      race,
      floatingPicks: raceChoices[race]?.floatingPicks ?? [],
      ascendencia: raceChoices[race]?.ascendencia,
      deformidade: deformidadePayload(race, raceChoices[race]),
    }))
}
