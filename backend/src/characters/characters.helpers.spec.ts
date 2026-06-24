import { BadRequestException } from '@nestjs/common';
import {
  assertCharacterRules,
  assertOverlaysCompatible,
  assertSlotsMultiple,
  rollAverage,
  sanitizeClassChoices,
} from './characters.helpers';
import type { CreateCharacterDto } from './dto/character.dto';

/**
 * Pure-function specs. No Prisma, no Nest — every helper takes plain data
 * and throws BadRequestException with a `fieldErrors` payload that the
 * frontend turns into per-field error messages.
 */

function baseDto(overrides: Partial<CreateCharacterDto> = {}): CreateCharacterDto {
  return {
    name: 'Hero',
    races: ['Humano'],
    classes: [{ className: 'Guerreiro', level: 1 }],
    origin: 'Soldado',
    god: null,
    hpMax: 12,
    hpCurrent: 12,
    mpMax: 4,
    mpCurrent: 4,
    strength: 3,
    dexterity: 2,
    constitution: 2,
    intelligence: 0,
    wisdom: 1,
    charisma: 1,
    size: 'M',
    displacement: 9,
    ...overrides,
  } as CreateCharacterDto;
}

describe('assertCharacterRules', () => {
  it('accepts a valid DTO without throwing', () => {
    expect(() => assertCharacterRules(baseDto())).not.toThrow();
  });

  it('rejects hpCurrent > hpMax with a fieldErrors payload', () => {
    try {
      assertCharacterRules(baseDto({ hpCurrent: 13, hpMax: 12 }));
      fail('should have thrown');
    } catch (e) {
      expect(e).toBeInstanceOf(BadRequestException);
      const body = (e as BadRequestException).getResponse() as {
        fieldErrors: Record<string, string[]>;
      };
      expect(body.fieldErrors.hpCurrent).toBeDefined();
    }
  });

  it('rejects mpCurrent > mpMax', () => {
    try {
      assertCharacterRules(baseDto({ mpCurrent: 10, mpMax: 4 }));
      fail('should have thrown');
    } catch (e) {
      const body = (e as BadRequestException).getResponse() as {
        fieldErrors: Record<string, string[]>;
      };
      expect(body.fieldErrors.mpCurrent).toBeDefined();
    }
  });

  it('rejects duplicate class entries with the offending index', () => {
    const dto = baseDto({
      classes: [
        { className: 'Guerreiro', level: 1 },
        { className: 'Guerreiro', level: 2 },
      ],
    });
    try {
      assertCharacterRules(dto);
      fail('should have thrown');
    } catch (e) {
      const body = (e as BadRequestException).getResponse() as {
        fieldErrors: Record<string, string[]>;
      };
      expect(body.fieldErrors['classes.1.className']).toBeDefined();
    }
  });

  it('reports multiple errors in a single throw', () => {
    const dto = baseDto({
      hpCurrent: 99,
      mpCurrent: 99,
    });
    try {
      assertCharacterRules(dto);
      fail('should have thrown');
    } catch (e) {
      const body = (e as BadRequestException).getResponse() as {
        fieldErrors: Record<string, string[]>;
      };
      expect(Object.keys(body.fieldErrors).sort()).toEqual([
        'hpCurrent',
        'mpCurrent',
      ]);
    }
  });
});

describe('sanitizeClassChoices', () => {
  it('returns empty object for null/undefined/non-object input', () => {
    expect(sanitizeClassChoices(null)).toEqual({});
    expect(sanitizeClassChoices(undefined)).toEqual({});
    expect(sanitizeClassChoices('string')).toEqual({});
  });

  it('drops empty/missing fields without throwing', () => {
    expect(sanitizeClassChoices({ Guerreiro: {} })).toEqual({});
    expect(
      sanitizeClassChoices({ Clérigo: { devoto: '', caminho: '' } }),
    ).toEqual({});
  });

  it('accepts a valid devoto for Clérigo', () => {
    const result = sanitizeClassChoices({ Clérigo: { devoto: 'khalmyr' } });
    expect(result.Clérigo?.devoto).toBe('khalmyr');
  });

  it('throws on unknown deus id', () => {
    try {
      sanitizeClassChoices({ Clérigo: { devoto: 'not-a-god' } });
      fail('should have thrown');
    } catch (e) {
      const body = (e as BadRequestException).getResponse() as {
        fieldErrors: Record<string, string[]>;
      };
      expect(body.fieldErrors['classChoices.Clérigo.devoto']).toBeDefined();
    }
  });

  it('accepts a valid caminho for Arcanista', () => {
    // Arcanista caminhos include 'mago', 'feiticeiro', 'bruxo' per t20-data.
    const result = sanitizeClassChoices({
      Arcanista: { caminho: 'mago' },
    });
    expect(result.Arcanista?.caminho).toBe('mago');
  });

  it('throws on caminho not valid for the class', () => {
    try {
      sanitizeClassChoices({ Arcanista: { caminho: 'not-a-path' } });
      fail('should have thrown');
    } catch (e) {
      const body = (e as BadRequestException).getResponse() as {
        fieldErrors: Record<string, string[]>;
      };
      expect(body.fieldErrors['classChoices.Arcanista.caminho']).toBeDefined();
    }
  });

  it('ignores bogus className keys silently (frontend gate)', () => {
    // Frontend never proposes a key like 'Hexer', so we tolerate it and
    // let it become inert at evaluation time. Only the *values* validate.
    expect(
      sanitizeClassChoices({ Hexer: { devoto: 'khalmyr' } }),
    ).toEqual({ Hexer: { devoto: 'khalmyr' } });
  });
});

describe('assertSlotsMultiple', () => {
  it.each([0, 0.5, 1, 1.5, 2, 5])('%d is valid', (slots) => {
    expect(() => assertSlotsMultiple(slots)).not.toThrow();
  });

  it.each([0.25, 0.7, 1.1, Number.NaN, Number.POSITIVE_INFINITY])(
    '%d is rejected',
    (slots) => {
      expect(() => assertSlotsMultiple(slots)).toThrow(BadRequestException);
    },
  );
});

describe('assertOverlaysCompatible', () => {
  it('no overlays → no-op (custom items allowed)', () => {
    expect(() => assertOverlaysCompatible(null, [], null)).not.toThrow();
    expect(() => assertOverlaysCompatible(null, undefined, '')).not.toThrow();
  });

  it('rejects overlays on custom items (no catalogId)', () => {
    expect(() =>
      assertOverlaysCompatible(null, ['melhoria-certeira'], null),
    ).toThrow(BadRequestException);
  });

  it('rejects unknown catalogId', () => {
    expect(() =>
      assertOverlaysCompatible('not-a-real-id', ['melhoria-certeira'], null),
    ).toThrow(BadRequestException);
  });

  it('rejects overlays on consumable items', () => {
    expect(() =>
      assertOverlaysCompatible(
        'balsamo-restaurador',
        ['melhoria-certeira'],
        null,
      ),
    ).toThrow(BadRequestException);
  });

  it('accepts weapon + weapon-only improvement (Certeira on Adaga)', () => {
    expect(() =>
      assertOverlaysCompatible('adaga', ['melhoria-certeira'], null),
    ).not.toThrow();
  });

  it('rejects weapon-only improvement on armor (Certeira on Cota de malha)', () => {
    try {
      assertOverlaysCompatible('cota-malha', ['melhoria-certeira'], null);
      fail('should have thrown');
    } catch (e) {
      const body = (e as BadRequestException).getResponse() as {
        fieldErrors: Record<string, string[]>;
      };
      expect(body.fieldErrors.improvements).toBeDefined();
    }
  });

  it('accepts armor + armor-compatible material (Mitral on Cota de malha)', () => {
    expect(() =>
      assertOverlaysCompatible('cota-malha', [], 'material-mitral'),
    ).not.toThrow();
  });

  it('rejects armor + weapon-only material (Aço-rubi on Cota de malha)', () => {
    try {
      assertOverlaysCompatible('cota-malha', [], 'material-aco-rubi');
      fail('should have thrown');
    } catch (e) {
      const body = (e as BadRequestException).getResponse() as {
        fieldErrors: Record<string, string[]>;
      };
      expect(body.fieldErrors.material).toBeDefined();
    }
  });

  it('rejects when material is not a material catalog entry', () => {
    try {
      // Adaga (weapon) used as material on a weapon — wrong category.
      assertOverlaysCompatible('espada-longa', [], 'adaga');
      fail('should have thrown');
    } catch (e) {
      const body = (e as BadRequestException).getResponse() as {
        fieldErrors: Record<string, string[]>;
      };
      expect(body.fieldErrors.material).toBeDefined();
    }
  });
});

describe('rollAverage', () => {
  it('returns the bonus for empty / "0" inputs', () => {
    expect(rollAverage('')).toBe(0);
    expect(rollAverage('0')).toBe(0);
    expect(rollAverage('0', 5)).toBe(5);
  });

  it('treats a flat integer as a flat bonus', () => {
    expect(rollAverage('5')).toBe(5);
    expect(rollAverage('5', 2)).toBe(7);
  });

  it('averages NdF as floor(N * (F+1) / 2) plus bonus', () => {
    // 2d4 → floor(2 * 5 / 2) = 5
    expect(rollAverage('2d4')).toBe(5);
    // 1d6 → floor(1 * 7 / 2) = 3
    expect(rollAverage('1d6')).toBe(3);
    // 3d6 → floor(3 * 7 / 2) = 10
    expect(rollAverage('3d6')).toBe(10);
  });

  it('adds bonus to dice average', () => {
    expect(rollAverage('2d4', 3)).toBe(8);
  });

  it('handles whitespace around the dice expression', () => {
    expect(rollAverage('  1d6  ')).toBe(3);
  });

  it('throws BadRequestException on malformed expression', () => {
    expect(() => rollAverage('xyz')).toThrow(BadRequestException);
    expect(() => rollAverage('2d')).toThrow(BadRequestException);
    expect(() => rollAverage('d6')).toThrow(BadRequestException);
  });
});

/**
 * Sibling describe blocks below pin behaviors the original spec only
 * brushed against — making sure a future refactor can't silently change
 * field-error shapes that the frontend already binds to.
 */
describe('assertCharacterRules — duplicate-class indexing', () => {
  it('flags only the later index when a class is duplicated', () => {
    // First occurrence is the canonical one; second flagged so the UI
    // surfaces the error on the duplicate row, not the original.
    const dto = baseDto({
      classes: [
        { className: 'Guerreiro', level: 1 },
        { className: 'Guerreiro', level: 2 },
      ],
    });
    try {
      assertCharacterRules(dto);
      fail('should have thrown');
    } catch (e) {
      const body = (e as BadRequestException).getResponse() as {
        fieldErrors: Record<string, string[]>;
      };
      expect(body.fieldErrors['classes.0.className']).toBeUndefined();
      expect(body.fieldErrors['classes.1.className']).toBeDefined();
    }
  });

  it('flags both later indices in a triple-duplicate run', () => {
    const dto = baseDto({
      classes: [
        { className: 'Guerreiro', level: 1 },
        { className: 'Guerreiro', level: 2 },
        { className: 'Guerreiro', level: 3 },
      ],
    });
    try {
      assertCharacterRules(dto);
      fail('should have thrown');
    } catch (e) {
      const body = (e as BadRequestException).getResponse() as {
        fieldErrors: Record<string, string[]>;
      };
      expect(Object.keys(body.fieldErrors).sort()).toEqual([
        'classes.1.className',
        'classes.2.className',
      ]);
    }
  });

  it('does not flag a non-adjacent duplicate that comes second', () => {
    const dto = baseDto({
      classes: [
        { className: 'Guerreiro', level: 1 },
        { className: 'Bardo', level: 1 },
        { className: 'Guerreiro', level: 1 },
      ],
    });
    try {
      assertCharacterRules(dto);
      fail('should have thrown');
    } catch (e) {
      const body = (e as BadRequestException).getResponse() as {
        fieldErrors: Record<string, string[]>;
      };
      expect(body.fieldErrors['classes.2.className']).toBeDefined();
      expect(body.fieldErrors['classes.1.className']).toBeUndefined();
    }
  });
});

describe('sanitizeClassChoices — combined fields + partial-reject semantics', () => {
  it('returns both devoto and caminho when both are valid', () => {
    const result = sanitizeClassChoices({
      Clérigo: { devoto: 'khalmyr' },
      Arcanista: { caminho: 'mago' },
    });
    expect(result.Clérigo?.devoto).toBe('khalmyr');
    expect(result.Arcanista?.caminho).toBe('mago');
  });

  it('throws with both fieldErrors when devoto and caminho are both unknown', () => {
    try {
      sanitizeClassChoices({
        Clérigo: { devoto: 'not-a-god' },
        Arcanista: { caminho: 'not-a-path' },
      });
      fail('should have thrown');
    } catch (e) {
      const body = (e as BadRequestException).getResponse() as {
        fieldErrors: Record<string, string[]>;
      };
      expect(body.fieldErrors['classChoices.Clérigo.devoto']).toBeDefined();
      expect(body.fieldErrors['classChoices.Arcanista.caminho']).toBeDefined();
    }
  });

  it('rejects a non-string devoto value', () => {
    try {
      sanitizeClassChoices({ Clérigo: { devoto: 42 } });
      fail('should have thrown');
    } catch (e) {
      const body = (e as BadRequestException).getResponse() as {
        fieldErrors: Record<string, string[]>;
      };
      expect(body.fieldErrors['classChoices.Clérigo.devoto']).toBeDefined();
    }
  });

  it('skips entries where the blob itself is not an object', () => {
    // `Guerreiro: null` and `Guerreiro: 7` are tolerated (no entry in `out`).
    expect(sanitizeClassChoices({ Guerreiro: null })).toEqual({});
    expect(sanitizeClassChoices({ Guerreiro: 7 })).toEqual({});
  });

  it('does not return a partial result alongside a thrown error', () => {
    // One valid + one invalid → should throw without returning the valid half.
    // Catches a future refactor that accidentally returns the half-baked object.
    try {
      sanitizeClassChoices({
        Clérigo: { devoto: 'khalmyr' },
        Arcanista: { caminho: 'not-a-path' },
      });
      fail('should have thrown');
    } catch (e) {
      expect(e).toBeInstanceOf(BadRequestException);
    }
  });
});

describe('assertOverlaysCompatible — multi-field + miscategorized overlay ids', () => {
  it('aggregates errors for improvement + material in a single throw', () => {
    try {
      assertOverlaysCompatible(
        'cota-malha',
        ['melhoria-certeira'],
        'material-aco-rubi',
      );
      fail('should have thrown');
    } catch (e) {
      const body = (e as BadRequestException).getResponse() as {
        fieldErrors: Record<string, string[]>;
      };
      expect(body.fieldErrors.improvements).toBeDefined();
      expect(body.fieldErrors.material).toBeDefined();
    }
  });

  it('flags only the unknown improvement id when others are valid', () => {
    try {
      assertOverlaysCompatible(
        'adaga',
        ['melhoria-certeira', 'not-a-real-improvement'],
        null,
      );
      fail('should have thrown');
    } catch (e) {
      const body = (e as BadRequestException).getResponse() as {
        fieldErrors: Record<string, string[]>;
      };
      expect(body.fieldErrors.improvements).toHaveLength(1);
      expect(body.fieldErrors.improvements?.[0]).toContain(
        'not-a-real-improvement',
      );
    }
  });

  it('rejects a material id passed in the improvements list (wrong category)', () => {
    try {
      assertOverlaysCompatible('adaga', ['material-aco-rubi'], null);
      fail('should have thrown');
    } catch (e) {
      const body = (e as BadRequestException).getResponse() as {
        fieldErrors: Record<string, string[]>;
      };
      expect(body.fieldErrors.improvements).toBeDefined();
    }
  });

  it('rejects an improvement id passed in the material field (wrong category)', () => {
    try {
      assertOverlaysCompatible('adaga', [], 'melhoria-certeira');
      fail('should have thrown');
    } catch (e) {
      const body = (e as BadRequestException).getResponse() as {
        fieldErrors: Record<string, string[]>;
      };
      expect(body.fieldErrors.material).toBeDefined();
    }
  });

  it('treats empty-string material the same as null (no overlay)', () => {
    // hasOverlays check uses `material !== '' && material !== null` — an
    // empty string from the frontend's clear-field path must short-circuit.
    expect(() => assertOverlaysCompatible(null, [], '')).not.toThrow();
  });
});

describe('assertSlotsMultiple — boundary values', () => {
  it('accepts negative half-slot values (no lower bound in the rule)', () => {
    // assertSlotsMultiple only enforces the 0.5 multiple — it does not
    // gate negative slots. DTO validation handles >= 0 separately.
    expect(() => assertSlotsMultiple(-0.5)).not.toThrow();
    expect(() => assertSlotsMultiple(-2)).not.toThrow();
  });

  it('rejects very small non-multiples (rounding precision)', () => {
    expect(() => assertSlotsMultiple(0.499)).toThrow(BadRequestException);
  });
});

describe('rollAverage — extra coverage', () => {
  it('handles large dice expressions', () => {
    // 4d10 → floor(4 * 11/2) = 22
    expect(rollAverage('4d10')).toBe(22);
    // 1d20 → floor(1 * 21/2) = 10
    expect(rollAverage('1d20')).toBe(10);
  });

  it('passes through negative bonus values', () => {
    // 2d4 (avg 5) - 2 = 3 — used for nerf consumables.
    expect(rollAverage('2d4', -2)).toBe(3);
  });

  it('accepts uppercase D in the dice expression', () => {
    // Regex uses /i flag — '2D6' must parse like '2d6'.
    expect(rollAverage('2D6')).toBe(7);
  });
});
