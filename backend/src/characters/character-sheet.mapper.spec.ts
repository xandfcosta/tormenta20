import {
  computeSheetForRow,
  raceNameToId,
  toCharacterInput,
  type CharacterDbRow,
} from './character-sheet.mapper';

/**
 * Fase A wiring: DB `Character` row → t20-data orchestrator input.
 *
 * DB stores race as name (PT display), t20-data expects id (kebab).
 * Level authority: sum of class levels (materialized `level` column is
 * derived); first class is treated as the primary for orchestrator
 * `className`.
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

  it('anão gets Constituição +2 / Sabedoria +1 / Destreza -1 from race mods', () => {
    const anao: CharacterDbRow = {
      ...humanoFighter,
      races: [{ race: 'Anão' }],
    };
    const sheet = computeSheetForRow(anao);
    expect(sheet.attributes.constitution.raceMod).toBe(2);
    expect(sheet.attributes.wisdom.raceMod).toBe(1);
    expect(sheet.attributes.dexterity.raceMod).toBe(-1);
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
