import { type FactCategory, getRace } from '@tormenta20/t20-data'
import type { Character } from '@/shared/api/api'

/** Uppercased primary class + level — the T20 analog to a Valorant "role" tag.
 *  Falls back to the origin when a character has no class yet. */
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

export function signed(n: number): string {
  return n >= 0 ? `+${n}` : String(n)
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

/** The one genuine prose string available: the first race ability's text. */
export function raceLoreLine(character: Character): string | null {
  const race = getRace(character.races[0]?.race ?? '')
  return race?.abilities[0]?.description ?? null
}

export type AbilityBlurb = {
  id: string
  name: string
  description: string
  category: FactCategory | null
}

/**
 * Race abilities as the ability-blurb row. Chosen over class/general powers
 * because they are fixed grants (no choice-parse), always carry name + a
 * one-line description, and are the flavor-richest. Capped to `limit`.
 */
export function raceAbilityBlurbs(
  character: Character,
  limit: number,
): AbilityBlurb[] {
  const race = getRace(character.races[0]?.race ?? '')
  if (!race) return []
  return race.abilities.slice(0, limit).map((ability) => ({
    id: ability.id,
    name: ability.name,
    description: ability.description,
    category: ability.facts?.[0]?.category ?? null,
  }))
}
