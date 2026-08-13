const NUMERALS: readonly [number, string][] = [
  [1000, 'M'],
  [900, 'CM'],
  [500, 'D'],
  [400, 'CD'],
  [100, 'C'],
  [90, 'XC'],
  [50, 'L'],
  [40, 'XL'],
  [10, 'X'],
  [9, 'IX'],
  [5, 'V'],
  [4, 'IV'],
  [1, 'I'],
]

/**
 * Roman numeral for display (chapter/step numbering in the grimório scenes).
 * Outside 1..3999 there is no numeral, so the arabic number comes back as-is —
 * a caller passing a bad value gets a readable screen, not an empty one.
 *
 * Always pair it with a plain-language `sr-only` line: "III" is decoration, and
 * a screen reader saying "eye eye eye" is not a step counter.
 *
 * @example romanNumeral(3) // 'III'
 */
export function romanNumeral(value: number): string {
  if (!Number.isInteger(value) || value < 1 || value > 3999) return String(value)
  let left = value
  let out = ''
  for (const [amount, numeral] of NUMERALS) {
    while (left >= amount) {
      out += numeral
      left -= amount
    }
  }
  return out
}
