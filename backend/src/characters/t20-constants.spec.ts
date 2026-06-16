import {
  CLASSES,
  GODS,
  ORIGINS,
  RACES,
  EXPERTISES,
  EXPERTISE_NAMES,
} from './t20-constants';
import { DEUSES, RACES_CATALOG, ORIGINS_CATALOG } from '@tormenta20/t20-data';

/**
 * Backend ships its own duplicate enums (RACES, CLASSES, ORIGINS, GODS) for
 * DTO validation since class-validator can't import const-arrays cleanly.
 * This spec is a drift-detector: if t20-data adds a new race / class /
 * origin / deus, the backend list must catch up so DTO validation doesn't
 * reject legitimate input.
 *
 * Known intentional divergence (kept as TODO, not a test failure):
 *   - GODS includes 'Tauron' (per-book deceased per PDF p96 — Aharadak
 *     killed him). Backend keeps Tauron for legacy character rows; this
 *     spec only checks that every t20-data deus id is *present* in the
 *     backend list, not the converse.
 */
describe('Backend t20-constants vs @tormenta20/t20-data', () => {
  it('RACES list has 17 entries (PDF Cap 1)', () => {
    expect(RACES).toHaveLength(17);
  });

  it('RACES list covers every race in t20-data catalog', () => {
    const catalogIds = RACES_CATALOG.map((r) => r.id);
    for (const id of catalogIds) {
      expect(RACES).toContain(id);
    }
  });

  it('CLASSES list has all 14 base classes', () => {
    expect([...CLASSES].sort()).toEqual(
      [
        'Arcanista',
        'Bárbaro',
        'Bardo',
        'Bucaneiro',
        'Caçador',
        'Cavaleiro',
        'Clérigo',
        'Druida',
        'Guerreiro',
        'Inventor',
        'Ladino',
        'Lutador',
        'Nobre',
        'Paladino',
      ].sort(),
    );
  });

  it('ORIGINS list has 35 entries (PDF Tabela 1-19)', () => {
    expect(ORIGINS).toHaveLength(35);
  });

  it('ORIGINS list covers every origin in t20-data catalog', () => {
    const catalogIds = ORIGINS_CATALOG.map((o) => o.id);
    for (const id of catalogIds) {
      expect(ORIGINS).toContain(id);
    }
  });

  it('GODS list covers every deus from t20-data DEUSES (by id)', () => {
    // Note: backend may also carry legacy / placeholder names (e.g.,
    // 'Tauron'). This test asserts coverage one-way.
    const lowerGods = (GODS as readonly string[]).map((g) => g.toLowerCase());
    for (const deus of DEUSES) {
      // Compare against the human name field (backend stores PT-BR display
      // name), not the catalog id.
      expect(lowerGods).toContain(deus.name.toLowerCase());
    }
  });

  it('GODS list includes Aharadak (added in PR #1 Phase 3 alignment)', () => {
    expect(GODS).toContain('Aharadak');
  });

  it('EXPERTISES re-export matches name list', () => {
    expect(EXPERTISES).toHaveLength(EXPERTISE_NAMES.length);
    for (const e of EXPERTISES) {
      expect(EXPERTISE_NAMES).toContain(e.name);
    }
  });
});

describe('Backend t20-constants — list invariants', () => {
  it('every list has unique entries', () => {
    expect(new Set(RACES).size).toBe(RACES.length);
    expect(new Set(CLASSES).size).toBe(CLASSES.length);
    expect(new Set(ORIGINS).size).toBe(ORIGINS.length);
    expect(new Set(GODS).size).toBe(GODS.length);
  });

  it('every list entry is a non-empty string', () => {
    for (const list of [RACES, CLASSES, ORIGINS, GODS]) {
      for (const name of list) {
        expect(typeof name).toBe('string');
        expect(name.length).toBeGreaterThan(0);
      }
    }
  });
});
