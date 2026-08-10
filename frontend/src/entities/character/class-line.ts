/**
 * A character's classes as the roster's subtitle line, e.g.
 * "Guerreiro 3 / Arcanista 1". Empty array → '' so the caller can fall back to
 * a plain level. Shared by the party roster and the campaign muster so the
 * class/level string is formatted one way everywhere.
 *
 * @example classLevelLine([{ className: 'Bárbaro', level: 2 }]) // 'Bárbaro 2'
 */
export function classLevelLine(
  classes: readonly { className: string; level: number }[],
): string {
  return classes.map((c) => `${c.className} ${c.level}`).join(' / ')
}
