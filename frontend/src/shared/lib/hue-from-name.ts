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

/**
 * The 155° gradient that fills a portrait or emblem frame until real art lands.
 * Same name always yields the same colours, so a hero or a chronicle is
 * recognisable by its palette across every surface that shows it.
 *
 * `lightness` and `chroma` are the first stop's — character portraits sit a hair
 * brighter than campaign emblems, and that difference is deliberate, so it stays
 * a parameter instead of a second copy of the formula.
 *
 * @example hueGradient('Thal, o Errante', 0.55, 0.15)
 */
export function hueGradient(name: string, lightness: number, chroma: number): string {
  const hue = hueFromName(name)
  return `linear-gradient(155deg, oklch(${lightness} ${chroma} ${hue}) 0%, oklch(0.30 0.09 ${hue}) 70%, oklch(0.22 0.06 ${hue}) 100%)`
}
