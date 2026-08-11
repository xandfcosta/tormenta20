/**
 * Format helper — prefixes a positive integer with `+`, leaves
 * negatives (which already carry `-`) alone. Used across the sheet
 * wherever a modifier is displayed inline (attrs, conditionals, dice).
 */
export function signed(n: number): string {
  return n >= 0 ? `+${n}` : String(n)
}
