import type { WeaponStats } from '@tormenta20/t20-data'
import { getCatalogItem } from '@/shared/lib/catalog-cache'
import type { Character } from '@/shared/api/api'

export type WieldedWeaponEntry = { name: string; weapon: WeaponStats }

/**
 * The character's wielded items resolved to catalog weapons, capped at two
 * (one per hand — the formula cards have room for exactly that). Per-weapon
 * attack mods (scope:'this') are NOT summed here: `mirrorWeaponAttackMods`
 * in entities/character/derived already lands them in Luta/Pontaria, which
 * every attack display reads.
 *
 * @example wieldedWeaponEntries(character)[0]?.weapon.damage // '1d8'
 */
export function wieldedWeaponEntries(
  character: Pick<Character, 'items'>,
): WieldedWeaponEntry[] {
  return character.items
    .filter((i) => i.equipped === 'wielded' || i.equipped === 'wielded2')
    .flatMap((i) => {
      const catalog = i.catalogId ? getCatalogItem(i.catalogId) : undefined
      return catalog?.weapon ? [{ name: i.name, weapon: catalog.weapon }] : []
    })
    .slice(0, 2)
}

/**
 * True when at least one wielded item is a catalog weapon. Drives the
 * contextual Vitais blocks: hybrids (Paladino, Druida, Bardo…) keep their
 * weapon attack row alongside the magic block instead of losing it to the
 * caster-only branch (UI audit task 17).
 *
 * @example hasWieldedWeapon(paladino) // true → weapon cards AND magic stats
 */
export function hasWieldedWeapon(character: Pick<Character, 'items'>): boolean {
  return wieldedWeaponEntries(character).length > 0
}
