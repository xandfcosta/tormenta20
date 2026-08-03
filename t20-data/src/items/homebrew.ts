/**
 * HOMEBREW allowances — deliberate departures from RAW that a table opted
 * into. Each entry documents the RAW rule it bends. Shared here so the
 * frontend equip picker and the backend equip-axis invariant stay in
 * agreement (both consume this set).
 */

/**
 * Esotéricos a table may wear ("Vestido") instead of holding. RAW p159 says
 * an esotérico only works EMPUNHADO — so wearing one is allowed by this
 * list, but its bonus stays off unless the player also switches on the
 * matching homebrew toggle in the Efeitos tab (see the frontend's
 * `medalhaoVestidoHomebrewMods`). Toggle off + vested = carried jewelry,
 * exactly what RAW says.
 *
 * @example HOMEBREW_VESTED_OK.has('medalhao-de-prata') // true
 */
export const HOMEBREW_VESTED_OK: ReadonlySet<string> = new Set([
  'medalhao-de-prata',
])
