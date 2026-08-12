import type { Character, RaceDefinition } from '@/shared/api/api'
import { hueFromName } from '@/shared/lib/hue-from-name'

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

/**
 * Up to two initials — the monogram standing in for portrait art everywhere in
 * this scene (stage, side peeks, filmstrip chips). One definition on purpose:
 * three copies of it drifted apart once already.
 *
 * @example initials('Thal, o Errante') // 'TO'
 */
export function initials(name: string): string {
  const parts = name.trim().split(/\s+/).filter(Boolean)
  if (parts.length === 0) return '?'
  return parts
    .slice(0, 2)
    .map((word) => word[0])
    .join('')
    .toUpperCase()
}

/**
 * The hue-derived gradient that fills a portrait frame until real art lands.
 * Same name always yields the same colours, so a hero is recognisable by its
 * palette across the stage, the peeks and the filmstrip.
 */
export function portraitGradient(name: string): string {
  const hue = hueFromName(name)
  return `linear-gradient(155deg, oklch(0.55 0.15 ${hue}) 0%, oklch(0.30 0.09 ${hue}) 70%, oklch(0.22 0.06 ${hue}) 100%)`
}

/**
 * The roster cursor runs over the heroes PLUS one trailing slot: the "+" that
 * opens the Forge. Modelling it as a real position is what lets the keyboard
 * reach creation — before this the arrows stopped at the last hero and the "+"
 * was mouse-only (ALE-98).
 *
 * The slot sits at index `count` (one past the last hero), so it is reached by
 * pressing → from the end and left again by pressing ←.
 *
 * @example stepRosterIndex(4, 1, 5) // 5 — the create slot
 */
export function stepRosterIndex(current: number, delta: number, count: number): number {
  const createSlot = count
  return Math.min(createSlot, Math.max(0, current + delta))
}

/** True when the cursor sits on the trailing "+" instead of a hero. */
export function isCreateSlot(index: number, count: number): boolean {
  return index >= count
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
