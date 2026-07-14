/**
 * Shared domain-rule validation. These are the invariants the backend enforces
 * on writes (class-level sum ≤ 20, equip caps, spell-cast preconditions, …),
 * extracted as pure functions so the frontend can pre-validate a mutation and
 * apply it optimistically without drifting from server enforcement.
 *
 * Every validator returns `ValidationError[]` — empty means valid. `field`
 * matches the backend `fieldErrors` key so a client can surface the same
 * message the server would have returned.
 */
export type ValidationError = { field: string; message: string }

/** True when the rule holds (no errors). */
export function isValid(errors: readonly ValidationError[]): boolean {
  return errors.length === 0
}

/** First error message, or null when valid — handy for a toast/inline hint. */
export function firstErrorMessage(
  errors: readonly ValidationError[],
): string | null {
  return errors[0]?.message ?? null
}
