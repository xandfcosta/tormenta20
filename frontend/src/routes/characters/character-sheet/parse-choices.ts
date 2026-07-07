/**
 * Parse a persisted `string[]` payload out of a Character JSON blob
 * column. Returns an empty array for any malformed content — corrupt
 * blobs should not crash the sheet, they should degrade to "no
 * choices made yet".
 */
export function parseChoices(raw: string): string[] {
  try {
    const parsed = JSON.parse(raw)
    if (Array.isArray(parsed)) {
      return parsed.filter((x): x is string => typeof x === 'string')
    }
    return []
  } catch {
    return []
  }
}
