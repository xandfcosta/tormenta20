import { BadRequestException } from '@nestjs/common';
import {
  getCatalogItem,
  HOMEBREW_VESTED_OK,
  type EquippedState,
} from '@tormenta20/t20-data';

/**
 * Equip-axis invariant. Regression for the 2026-08 live UI audit: the API
 * persisted `equipped: 'vested'` for Escudo pesado (catalog equip axis
 * 'wielded') — an impossible state the sheet engine then silently ignored.
 * The catalog `equip` field says how an item is carried; the stored
 * `equipped` state must stay on that axis (or null = stowed).
 */

/**
 * Equipped states a catalog item may legally occupy, per its `equip` axis:
 *  - 'vested'  (armor, apparel)   → only 'vested'
 *  - 'wielded' (weapons, shields) → 'wielded' or the two-hand grip 'wielded2'
 *  - 'either'  (consumables, gear, overlays, mounts) → none: stow-only.
 *    In the catalog, 'either' means "works however carried", i.e. the item
 *    has no equip slot at all.
 * The 4-vested / 2-hands caps are a separate rule (`validateEquipChange`).
 *
 * @example allowedEquipStates('wielded') // → ['wielded', 'wielded2']
 */
export function allowedEquipStates(
  equip: 'vested' | 'wielded' | 'either',
): EquippedState[] {
  if (equip === 'vested') return ['vested'];
  if (equip === 'wielded') return ['wielded', 'wielded2'];
  return [];
}

/**
 * Throw when `equipped` is off the catalog item's equip axis. Custom items
 * (no catalogId) and unknown ids skip — there is no catalog axis to enforce
 * (addItem already rejects unknown ids on its own).
 *
 * @example assertEquipAxisAllowed('escudo-pesado', 'vested') // throws 400
 */
export function assertEquipAxisAllowed(
  catalogId: string | null | undefined,
  equipped: EquippedState,
): void {
  const catalog = catalogId ? getCatalogItem(catalogId) : undefined;
  if (!catalog) return;
  // Homebrew allowance (shared registry): some esotéricos may be WORN even
  // though their RAW axis is wielded-only — the bonus still requires the
  // matching Efeitos toggle on the frontend, so RAW users lose nothing.
  if (equipped === 'vested' && HOMEBREW_VESTED_OK.has(catalog.id)) return;
  const allowed = allowedEquipStates(catalog.equip);
  if (allowed.includes(equipped)) return;
  const expected =
    allowed.length > 0
      ? `null | ${allowed.map((s) => `'${s}'`).join(' | ')}`
      : 'null (item is not equippable)';
  throw new BadRequestException({
    statusCode: 400,
    error: 'Bad Request',
    message: `equipped '${equipped}' is invalid for "${catalog.name}" (equip axis '${catalog.equip}') — expected ${expected}`,
    fieldErrors: {
      equipped: [
        allowed.length > 0
          ? `"${catalog.name}" só aceita ${allowed.join(' ou ')}`
          : `"${catalog.name}" não é equipável`,
      ],
    },
  });
}
