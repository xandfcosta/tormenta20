/**
 * Deterministic hue (0–359) from a character name, so every character gets a
 * stable, distinct splash tint without any stored image/color. Same name →
 * same hue across reloads. Used by the character-select splash gradient +
 * keystone; the app has no per-character image field (reskin-only).
 *
 * @example hueFromName('Thorvald') // => 214 (stable)
 */
export function hueFromName(name: string): number {
  let hash = 0
  for (const ch of name) {
    // 31-based rolling hash, kept unsigned so the modulo is always positive.
    hash = (hash * 31 + ch.charCodeAt(0)) >>> 0
  }
  return hash % 360
}
