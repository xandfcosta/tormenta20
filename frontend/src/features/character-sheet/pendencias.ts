import {
  caminhoSlotFor,
  classPowersFor,
  devotoOptionsFor,
  generalPowersByKinds,
  getOrigin,
  getRace,
  slotsForClassLevel,
  unlockedKinds,
} from '@tormenta20/t20-data'
import type { Character } from '@/shared/api/api'
import { parseClassChoices } from '@/entities/character/derived'
import { parseChoices } from './parse-choices'

/** Which sub-tab a pendência belongs to — matches the source tab values. */
export type PendenciaSource = 'raca' | 'origem' | 'classe'

/**
 * One still-unmade required choice on the Habilidades tab. `cardId` targets the
 * collapsible card that owns the choice, so clicking a pendência can jump to
 * the source tab and open + scroll to that card.
 */
export type Pendencia = {
  source: PendenciaSource
  label: string
  cardId: string
}

const ORIGIN_BENEFIT_LIMIT = 2

/**
 * Derives the list of unmade choices from query data alone (no server call).
 * A source that isn't in the catalog emits no pendência — we can't validate
 * choices for data we don't have.
 *
 * @example
 * const pendencias = computePendencias(character)
 * // [{ source: 'origem', label: 'Origem: faltam 2 benefícios', cardId: 'origem' }]
 */
export function computePendencias(character: Character): Pendencia[] {
  return [
    ...racePendencias(character),
    ...originPendencias(character),
    ...classPendencias(character),
  ]
}

function racePendencias(character: Character): Pendencia[] {
  const choices = parseChoices(character.raceAbilityChoices)
  const out: Pendencia[] = []
  for (const { race: raceId } of character.races) {
    const race = getRace(raceId)
    if (!race) continue
    for (const ability of race.abilities) {
      if (!ability.variants) continue
      const picked = ability.variants.some((v) => choices.includes(v.id))
      if (picked) continue
      out.push({
        source: 'raca',
        label: `Raça: escolher variante de ${ability.name}`,
        cardId: `raca:${race.id}`,
      })
    }
  }
  return out
}

function originPendencias(character: Character): Pendencia[] {
  const origin = getOrigin(character.origin)
  if (!origin) return []
  const benefitIds = new Set(
    [...origin.benefits, origin.poderUnico].map((b) => b.id),
  )
  const chosen = parseChoices(character.originChoices).filter((id) =>
    benefitIds.has(id),
  )
  const remaining = ORIGIN_BENEFIT_LIMIT - chosen.length
  if (remaining <= 0) return []
  const noun = remaining === 1 ? 'benefício' : 'benefícios'
  return [
    {
      source: 'origem',
      label: `Origem: faltam ${remaining} ${noun}`,
      cardId: 'origem',
    },
  ]
}

function classPendencias(character: Character): Pendencia[] {
  const allChosen = parseChoices(character.classPowers)
  const classChoices = parseClassChoices(character.classChoices)
  const out: Pendencia[] = []
  for (const entry of character.classes) {
    const { className, level } = entry
    const pool = classPowersFor(className)
    if (pool.length === 0) continue
    const cardId = `classe:${className}`

    const remaining = slotsRemainingFor(className, level, allChosen)
    if (remaining > 0) {
      const noun = remaining === 1 ? 'poder' : 'poderes'
      out.push({
        source: 'classe',
        label: `Classe ${className}: ${remaining} ${noun} por escolher`,
        cardId,
      })
    }

    const blob = classChoices[className] ?? {}
    if (devotoOptionsFor(className) !== null && !blob.devoto) {
      out.push({
        source: 'classe',
        label: `Classe ${className}: escolher devoto`,
        cardId,
      })
    }
    const caminhoSlot = caminhoSlotFor(className)
    if (caminhoSlot !== null && level >= caminhoSlot.minLevel && !blob.caminho) {
      out.push({
        source: 'classe',
        label: `Classe ${className}: escolher caminho`,
        cardId,
      })
    }
  }
  return out
}

/** Open power slots for a class = granted slots minus elective/general picks. */
function slotsRemainingFor(
  className: string,
  level: number,
  allChosen: string[],
): number {
  const pool = classPowersFor(className)
  const electiveIds = new Set(
    pool.filter((p) => p.grantedAtLevel === undefined).map((p) => p.id),
  )
  const generalIds = new Set(
    generalPowersByKinds(unlockedKinds(className, level)).map((p) => p.id),
  )
  const picks = allChosen.filter(
    (id) => electiveIds.has(id) || generalIds.has(id),
  ).length
  return Math.max(0, slotsForClassLevel(className, level).length - picks)
}
