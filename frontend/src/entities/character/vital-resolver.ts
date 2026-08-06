import {
  originModifiers as computeOriginModifiers,
  raceModifiers as computeRaceModifiers,
  type Modifier,
  type VitalAbilitiesResolver,
} from '@tormenta20/t20-data'
import {
  classPowerModifiers,
  getGeneralPower,
  getOrigin,
  getRace,
  grantedPowerByName,
} from '@/shared/lib/abilities-cache'
import { racaById } from '@/shared/lib/racas-cache'

/**
 * Cache-backed `VitalAbilitiesResolver` for the FRONT — the twin of t20-data's
 * `defaultVitalResolver`, but reading the fetched-and-cached abilities catalog
 * (via `abilities-cache`) instead of the bundled data. Injected into
 * `collectVitalGrants` by the optimism helpers (level-vitals, draft-vitals) so
 * the client runs the exact same vital pipeline as the server without bundling
 * the ~149KB abilities chunk (project_front_decouple_catalog B.3). The pure
 * modifier builders (raceModifiers/originModifiers) come straight from t20-data
 * — they tree-shake on their own. `racaById` is racas.ts (t20-classes), already
 * in the bundle, and bridges the racas slug → abilities race name.
 */
function abilitiesRaceKey(raceId: string): string {
  try {
    return racaById(raceId).name
  } catch {
    return raceId
  }
}

export const frontVitalResolver: VitalAbilitiesResolver = {
  raceModifiers(raceId: string, variantChoices: ReadonlySet<string>): Modifier[] {
    const race = getRace(abilitiesRaceKey(raceId))
    return race ? computeRaceModifiers(race, variantChoices) : []
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
    return origin ? computeOriginModifiers(origin, choiceSet) : []
  },
}
