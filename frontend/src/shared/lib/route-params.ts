import { z } from 'zod'

/**
 * A positive-integer path param (campaign / session / character id). TanStack
 * hands path params through as strings; coercing + validating here rejects a
 * malformed URL (`/campaigns/abc`, `/characters/-1`) at the route boundary so a
 * `NaN` never reaches a query key and silently poisons the cache.
 */
export const idParamSchema = z.coerce.number().int().positive()

/**
 * Build a route `params` config (`parse` + `stringify`) for the named path
 * params: inbound strings are coerced + validated to positive integers via
 * {@link idParamSchema} (a bad value throws → route error boundary), and the
 * matching `stringify` turns the numbers back into strings when building URLs
 * (required whenever `parse` changes the param type).
 *
 * @example
 * createFileRoute('/characters/$id')({ params: idParams('id') })
 */
export function idParams<K extends string>(...keys: K[]) {
  return {
    parse: (raw: Record<string, string>): Record<K, number> => {
      const out = {} as Record<K, number>
      for (const key of keys) out[key] = idParamSchema.parse(raw[key])
      return out
    },
    stringify: (parsed: Record<string, number>): Record<K, string> => {
      const out = {} as Record<K, string>
      for (const key of keys) out[key] = String(parsed[key])
      return out
    },
  }
}
