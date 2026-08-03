import { BadRequestException } from '@nestjs/common';
import {
  allowedEquipStates,
  assertEquipAxisAllowed,
} from './equip-axis.helpers';

/**
 * Equip-axis invariant (regression for the 2026-08 live UI audit): the API
 * persisted `equipped: 'vested'` for Escudo pesado (catalog equip 'wielded'),
 * an impossible state the engine then ignored. Every equip write must match
 * the catalog item's equip axis; custom items (no catalogId) skip the check.
 */
describe('allowedEquipStates', () => {
  it("maps 'vested' → only 'vested'", () => {
    expect(allowedEquipStates('vested')).toEqual(['vested']);
  });

  it("maps 'wielded' → 'wielded' or two-hand grip 'wielded2'", () => {
    expect(allowedEquipStates('wielded')).toEqual(['wielded', 'wielded2']);
  });

  it("maps 'either' (consumables/gear/overlays) → stow-only", () => {
    expect(allowedEquipStates('either')).toEqual([]);
  });
});

describe('assertEquipAxisAllowed', () => {
  it('accepts armor on the vested axis (armadura-couro → vested)', () => {
    expect(() =>
      assertEquipAxisAllowed('armadura-couro', 'vested'),
    ).not.toThrow();
  });

  it('accepts a weapon on the wielded axis (espada-curta → wielded)', () => {
    expect(() =>
      assertEquipAxisAllowed('espada-curta', 'wielded'),
    ).not.toThrow();
  });

  it('accepts a two-hand grip on a wieldable (machado-batalha → wielded2)', () => {
    expect(() =>
      assertEquipAxisAllowed('machado-batalha', 'wielded2'),
    ).not.toThrow();
  });

  it('rejects a shield worn as apparel (escudo-pesado → vested)', () => {
    try {
      assertEquipAxisAllowed('escudo-pesado', 'vested');
      fail('should have thrown');
    } catch (e) {
      expect(e).toBeInstanceOf(BadRequestException);
      const body = (e as BadRequestException).getResponse() as {
        message: string;
        fieldErrors: Record<string, string[]>;
      };
      // Root rule: message carries the offending value + expected shape.
      expect(body.message).toContain("'vested'");
      expect(body.message).toContain('Escudo pesado');
      expect(body.message).toContain("'wielded'");
      expect(body.fieldErrors.equipped).toBeDefined();
    }
  });

  it('rejects armor wielded as a weapon (armadura-completa → wielded)', () => {
    expect(() =>
      assertEquipAxisAllowed('armadura-completa', 'wielded'),
    ).toThrow(BadRequestException);
  });

  it('rejects any equip state on a consumable (balsamo-restaurador)', () => {
    for (const state of ['vested', 'wielded', 'wielded2'] as const) {
      expect(() =>
        assertEquipAxisAllowed('balsamo-restaurador', state),
      ).toThrow(BadRequestException);
    }
  });

  it('skips validation for custom items (no catalogId)', () => {
    expect(() => assertEquipAxisAllowed(null, 'vested')).not.toThrow();
    expect(() => assertEquipAxisAllowed(undefined, 'wielded2')).not.toThrow();
  });

  it('skips validation for an unknown catalogId (addItem validates it separately)', () => {
    expect(() => assertEquipAxisAllowed('nao-existe', 'vested')).not.toThrow();
  });

  describe('homebrew registry (HOMEBREW_VESTED_OK)', () => {
    it('allows wearing the Medalhão de prata despite the wielded axis', () => {
      expect(() =>
        assertEquipAxisAllowed('medalhao-de-prata', 'vested'),
      ).not.toThrow();
    });

    it('keeps a non-registry esotérico wielded-only (orbe-cristalino)', () => {
      expect(() =>
        assertEquipAxisAllowed('orbe-cristalino', 'vested'),
      ).toThrow(BadRequestException);
    });

    it('the registry does not open other impossible states (medalhão as null-only axis stays intact)', () => {
      // wielded/wielded2 remain valid via the normal axis path.
      expect(() =>
        assertEquipAxisAllowed('medalhao-de-prata', 'wielded'),
      ).not.toThrow();
    });
  });
});
