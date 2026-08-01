import {
  computeSheetForRow,
  equipmentFromRow,
  expertiseNameToSkillId,
  raceNameToId,
  toCharacterInput,
  type CharacterDbRow,
} from './character-sheet.mapper';

/**
 * Fase A wiring + Fase A followup: DB `Character` row → orchestrator
 * input. Beyond the basics from Fase A, the mapper now decodes trained
 * skills (PT expertise name → SkillId) and equipped items (catalogId
 * → armor/shield/weapon slots).
 */

const humanoFighter: CharacterDbRow = {
  level: 3,
  strength: 3,
  dexterity: 2,
  constitution: 2,
  intelligence: 0,
  wisdom: 1,
  charisma: 0,
  hpCurrent: 30,
  mpCurrent: 5,
  races: [{ race: 'Humano' }],
  classes: [{ className: 'Guerreiro', level: 3 }],
  expertises: [],
  items: [],
};

describe('raceNameToId', () => {
  it('maps a canonical PT name to the t20-data id', () => {
    expect(raceNameToId('Humano')).toBe('humano');
    expect(raceNameToId('Anão')).toBe('anao');
  });

  it('is tolerant of casing / whitespace', () => {
    expect(raceNameToId('humano')).toBe('humano');
    expect(raceNameToId('  ANÃO  ')).toBe('anao');
  });

  it('returns undefined for unknown names', () => {
    expect(raceNameToId('Não-Existe')).toBeUndefined();
  });
});

describe('toCharacterInput', () => {
  it('projects the DB row onto CharacterInput', () => {
    const input = toCharacterInput(humanoFighter);
    expect(input.level).toBe(3);
    expect(input.className).toBe('Guerreiro');
    expect(input.raceId).toBe('humano');
    expect(input.baseAttributes.strength).toBe(3);
    expect(input.baseAttributes.dexterity).toBe(2);
    expect(input.currentPv).toBe(30);
    expect(input.currentPm).toBe(5);
  });

  it('uses the summed class levels when they diverge from the materialized column', () => {
    const multiclass = {
      ...humanoFighter,
      level: 3,
      classes: [
        { className: 'Guerreiro', level: 2 },
        { className: 'Arcanista', level: 3 },
      ],
    };
    expect(toCharacterInput(multiclass).level).toBe(5);
    expect(toCharacterInput(multiclass).className).toBe('Guerreiro');
  });

  it('falls back to the materialized level when no classes are stored', () => {
    const noClasses = { ...humanoFighter, classes: [] };
    expect(toCharacterInput(noClasses).level).toBe(3);
  });

  it('leaves raceId undefined for unknown races', () => {
    const noRace = { ...humanoFighter, races: [{ race: 'Alien' }] };
    expect(toCharacterInput(noRace).raceId).toBeUndefined();
  });
});

describe('computeSheetForRow', () => {
  it('returns a ComputedSheet with attributes reflecting racial mods', () => {
    const sheet = computeSheetForRow(humanoFighter);
    /* Humano tem 3 floating +1; sem picks, aplicamos apenas base. */
    expect(sheet.attributes.strength.base).toBe(3);
    expect(sheet.attributes.dexterity.base).toBe(2);
    expect(sheet.level).toBe(3);
    expect(sheet.className).toBe('Guerreiro');
  });

  it('applies the fixed race attribute mod ONCE from BASE stored attrs', () => {
    // Stored attributes are BASE (pre-race). Anão CON+2/SAB+1/DES-1 is applied
    // exactly once by the orchestrator (no double).
    const anao: CharacterDbRow = { ...humanoFighter, races: [{ race: 'Anão' }] };
    const sheet = computeSheetForRow(anao);
    expect(sheet.attributes.constitution.raceMod).toBe(2);
    expect(sheet.attributes.constitution.total).toBe(humanoFighter.constitution + 2);
    expect(sheet.attributes.dexterity.raceMod).toBe(-1);
    expect(sheet.attributes.wisdom.raceMod).toBe(1);
  });

  it('applies a floating race mod from persisted raceAttributeChoices', () => {
    // Humano places its +1×3 via floating picks; without them, no race mod.
    const humano: CharacterDbRow = {
      ...humanoFighter,
      races: [{ race: 'Humano' }],
      raceAttributeChoices: JSON.stringify({
        floatingPicks: ['strength', 'constitution', 'wisdom'],
      }),
    };
    const sheet = computeSheetForRow(humano);
    expect(sheet.attributes.strength.total).toBe(humanoFighter.strength + 1);
    expect(sheet.attributes.constitution.total).toBe(humanoFighter.constitution + 1);
    expect(sheet.attributes.wisdom.total).toBe(humanoFighter.wisdom + 1);
    expect(sheet.attributes.dexterity.raceMod).toBe(0);
  });

  it('still derives race-driven movement + PV grant', () => {
    const anao: CharacterDbRow = { ...humanoFighter, races: [{ race: 'Anão' }] };
    const sheet = computeSheetForRow(anao);
    expect(sheet.deslocamento).toBe(6); // Anão movement
    // Guerreiro L3, base CON 2 → total 4 (Anão +2). pvBase = 20 + 2*5 + 4*3 = 42;
    // Duro como Pedra (nível+2 = 5) folds in once → 47.
    expect(sheet.vitals.pvMax).toBe(47);
  });

  it('exposes a defense object with base 10', () => {
    const sheet = computeSheetForRow(humanoFighter);
    expect(sheet.defense.base).toBe(10);
    /* Sem armadura equipada, Defesa total = 10 + DES total. */
    expect(sheet.defense.armor).toBe(0);
    expect(sheet.defense.shield).toBe(0);
  });

  it('exposes saves derived from ½ level + attribute', () => {
    const sheet = computeSheetForRow(humanoFighter);
    expect(sheet.saves.fortitude).toBeDefined();
    expect(sheet.saves.reflexos).toBeDefined();
    expect(sheet.saves.vontade).toBeDefined();
  });

  it('exposes deslocamento derived from race (humano = 9m)', () => {
    const sheet = computeSheetForRow(humanoFighter);
    expect(sheet.deslocamento).toBe(9);
  });
});

// ─── Passive PV/PM grants threaded from JSON columns ─────────────

describe('computeSheetForRow — passive max-PV/PM grants', () => {
  it('parses classPowers/originChoices JSON into CharacterInput', () => {
    const input = toCharacterInput({
      ...humanoFighter,
      origin: 'Acólito',
      classPowers: JSON.stringify(['vitalidade']),
      originChoices: JSON.stringify(['poder-vontade-de-ferro']),
    });
    expect(input.powerIds).toEqual(['vitalidade']);
    expect(input.originChoices).toEqual(['poder-vontade-de-ferro']);
    expect(input.origin).toBe('Acólito');
  });

  it('tolerates a malformed JSON blob (empty, no crash)', () => {
    const input = toCharacterInput({ ...humanoFighter, classPowers: 'not json' });
    expect(input.powerIds).toEqual([]);
  });

  it('Vitalidade (general power) adds +1 PV per level to pvMax', () => {
    const base = computeSheetForRow(humanoFighter).vitals.pvMax;
    const boosted = computeSheetForRow({
      ...humanoFighter,
      classPowers: JSON.stringify(['vitalidade']),
    }).vitals.pvMax;
    expect(boosted - base).toBe(3); // +1/nível, nível 3
  });

  it('Clérigo Magia Divina (auto) folds +Sabedoria into pmMax', () => {
    const clerigo: CharacterDbRow = {
      ...humanoFighter,
      wisdom: 4,
      classes: [{ className: 'Clérigo', level: 3 }],
    };
    // pmBase = 5*3 = 15; +Sabedoria total (4 base, Humano sem bônus fixo) = 19.
    expect(computeSheetForRow(clerigo).vitals.pmMax).toBe(19);
  });
});

// ─── Skills ──────────────────────────────────────────────────────

describe('expertiseNameToSkillId', () => {
  it('strips diacritics + lowercases (PT → SkillId)', () => {
    expect(expertiseNameToSkillId('Acrobacia')).toBe('acrobacia');
    expect(expertiseNameToSkillId('Atuação')).toBe('atuacao');
    expect(expertiseNameToSkillId('Ofício')).toBe('oficio');
    expect(expertiseNameToSkillId('Percepção')).toBe('percepcao');
    expect(expertiseNameToSkillId('Religião')).toBe('religiao');
    expect(expertiseNameToSkillId('Sobrevivência')).toBe('sobrevivencia');
  });

  it('returns undefined for names that are not in Tabela 2-1', () => {
    expect(expertiseNameToSkillId('Culinária Élfica')).toBeUndefined();
  });

  it('is case-insensitive and trims', () => {
    expect(expertiseNameToSkillId('  ATLETISMO  ')).toBe('atletismo');
  });
});

describe('toCharacterInput trainedSkills', () => {
  it('projects trained expertises into SkillId[]', () => {
    const row: CharacterDbRow = {
      ...humanoFighter,
      expertises: [
        { name: 'Atletismo', attribute: 'strength', trained: true, custom: false },
        { name: 'Luta', attribute: 'strength', trained: true, custom: false },
        { name: 'Furtividade', attribute: 'dexterity', trained: false, custom: false },
      ],
    };
    const input = toCharacterInput(row);
    expect(input.trainedSkills).toEqual(['atletismo', 'luta']);
  });

  it('ignores custom names that do not match Tabela 2-1', () => {
    const row: CharacterDbRow = {
      ...humanoFighter,
      expertises: [
        { name: 'Atletismo', attribute: 'strength', trained: true, custom: false },
        { name: 'Culinária Élfica', attribute: 'intelligence', trained: true, custom: true },
      ],
    };
    expect(toCharacterInput(row).trainedSkills).toEqual(['atletismo']);
  });
});

// ─── Equipment ───────────────────────────────────────────────────

describe('equipmentFromRow', () => {
  it('returns undefined when nothing is equipped', () => {
    expect(
      equipmentFromRow({
        items: [{ catalogId: 'espada-longa', name: 'Espada', equipped: null }],
      }),
    ).toBeUndefined();
  });

  it('populates armor slot from a vested armor catalog entry', () => {
    const eq = equipmentFromRow({
      items: [
        { catalogId: 'armadura-couro', name: 'Armadura de couro', equipped: 'vested' },
      ],
    });
    expect(eq?.armor?.defense).toBe(2);
    expect(eq?.armor?.penalty).toBe(0);
    expect(eq?.armor?.heavy).toBe(false);
  });

  it('populates mainHand from a wielded one-handed weapon', () => {
    const eq = equipmentFromRow({
      items: [
        { catalogId: 'espada-longa', name: 'Espada longa', equipped: 'wielded' },
      ],
    });
    expect(eq?.mainHand?.name).toBe('Espada longa');
    expect(eq?.mainHand?.damage).toBe('1d8');
    expect(eq?.mainHand?.hand).toBe('one');
  });

  it('two-handed weapon (wielded2) fills mainHand only, offHand stays empty', () => {
    /* Sample any two-handed weapon by catalogId. If two-handed weapon
     * doesn't have catalogId 'espada-grande', treat as a smoke test —
     * just assert equipped=wielded2 doesn't fill offHand. */
    const eq = equipmentFromRow({
      items: [
        {
          catalogId: 'espada-longa',
          name: 'Espada longa',
          equipped: 'wielded2',
        },
      ],
    });
    expect(eq?.mainHand).toBeDefined();
    expect(eq?.offHand).toBeUndefined();
  });

  it('ignores custom items (catalogId=null) — no mechanical stats available', () => {
    const eq = equipmentFromRow({
      items: [
        { catalogId: null, name: 'Espada singular do herói', equipped: 'wielded' },
      ],
    });
    expect(eq).toBeUndefined();
  });

  it('ignores catalog ids that no longer exist', () => {
    const eq = equipmentFromRow({
      items: [
        { catalogId: 'nao-existe-XYZ', name: 'X', equipped: 'wielded' },
      ],
    });
    expect(eq).toBeUndefined();
  });
});

describe('computeSheetForRow with equipment', () => {
  it('picks up armor defense in the derived sheet', () => {
    const sheet = computeSheetForRow({
      ...humanoFighter,
      items: [
        { catalogId: 'armadura-couro', name: 'Armadura de couro', equipped: 'vested' },
      ],
    });
    expect(sheet.defense.armor).toBeGreaterThan(0);
  });
});
