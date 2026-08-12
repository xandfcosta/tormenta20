import type { Character, RaceDefinition } from '@/shared/api/api'

/**
 * Uppercased primary class + level — the T20 analog to a Valorant "role" tag.
 * Falls back to the origin when a character has no class yet.
 *
 * @example primaryRole(char) // 'GUERREIRO 10'
 */
export function primaryRole(character: Character): string {
  const first = character.classes[0]
  if (!first) return character.origin.toUpperCase()
  return `${first.className} ${first.level}`.toUpperCase()
}

/** Class + level as shown in the splash name overlay (mixed case). */
export function primaryClass(character: Character): string {
  const first = character.classes[0]
  if (!first) return character.origin
  return `${first.className} ${first.level}`
}

/**
 * Assembled flavor line for the info panel. Characters have NO freeform bio
 * field, so we compose one from structured fields: races • origin • deity •
 * size • level. `god` is nullable and dropped when absent.
 */
export function characterFlavor(character: Character): string {
  const races = character.races.map((r) => r.race).join(', ')
  return [
    races,
    character.origin,
    character.god ? `devoto de ${character.god}` : null,
    character.size,
    `Nível ${character.level}`,
  ]
    .filter(Boolean)
    .join(' • ')
}

export type AbilityBlurb = {
  id: string
  name: string
  description: string
}

/**
 * Race abilities as the dossier's ability row. Chosen over class/general
 * powers because they're fixed grants (no choice-parse), always carry name + a
 * one-line description, and are the flavor-richest. Capped to `limit`.
 *
 * The React version reached into a module-level catalog cache; here the
 * catalog comes in as a parameter, so the rule is pure and the caller owns the
 * fetching.
 *
 * @example raceAbilityBlurbs(raceDefs(), character, 8)
 */
export function raceAbilityBlurbs(
  raceDefs: readonly RaceDefinition[],
  character: Character,
  limit: number,
): AbilityBlurb[] {
  const raceName = character.races[0]?.race ?? ''
  const race = raceDefs.find((r) => r.name === raceName || r.id === raceName)
  if (!race) return []
  return race.abilities.slice(0, limit).map((ability) => ({
    id: ability.id,
    name: ability.name,
    description: ability.description,
  }))
}
