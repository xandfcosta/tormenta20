import type { Modifier } from '../items/types'
import { racaById } from '../racas'
import type { VitalAbilitiesResolver } from '../vital-grants'
import { getOrigin, getRace, raceModifiers } from './catalog'
import { classPowerModifiers } from './classes'
import { getGeneralPower } from './general-powers'
import { grantedPowerByName } from './granted-powers'
import { originModifiers } from './origins'

/**
 * Data-backed `VitalAbilitiesResolver` for the engine/backend — binds the vital
 * pipeline to the real t20-data catalog. Kept in its OWN module (not
 * vital-grants.ts) so importing `collectVitalGrants` doesn't drag the abilities
 * catalog into the frontend bundle; the front injects a cache-backed resolver
 * instead (project_front_decouple_catalog B.3).
 */

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

export const defaultVitalResolver: VitalAbilitiesResolver = {
  raceModifiers(raceId: string, variantChoices: ReadonlySet<string>): Modifier[] {
    const race = getRace(abilitiesRaceKey(raceId))
    return race ? raceModifiers(race, variantChoices) : []
  },
  classPowerModifiers,
  generalPowerModifiers(id: string): Modifier[] {
    return getGeneralPower(id)?.modifiers ?? []
  },
  godPowerModifiers(godPowerName: string): Modifier[] {
    return grantedPowerByName(godPowerName)?.modifiers ?? []
  },
  originModifiers(originId: string, choiceSet: ReadonlySet<string>): Modifier[] {
    const origin = getOrigin(originId)
    return origin ? originModifiers(origin, choiceSet) : []
  },
}
