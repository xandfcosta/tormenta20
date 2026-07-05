import {
  RACAS,
  computeCharacterSheet,
  type CharacterInput,
  type ComputedSheet,
} from '@tormenta20/t20-data';

/**
 * Backend DB `Character` shape (subset) → t20-data `CharacterInput`, then
 * runs the orchestrator to produce a derived sheet.
 *
 * Missing today (moves to later phases):
 *   - trainedSkills (needs Expertise PT name → SkillId map)
 *   - equipment (needs Item catalogId → armor/weapon spec resolver)
 *   - activeEffects (needs modifier decoder from JSON blob)
 *
 * v1 output covers: attributes, vitals, defense (base+attribute only,
 * armor/shield=0 for now), saves, movement.
 */

export type CharacterDbRow = {
  level: number;
  strength: number;
  dexterity: number;
  constitution: number;
  intelligence: number;
  wisdom: number;
  charisma: number;
  hpCurrent: number;
  mpCurrent: number;
  races: readonly { race: string }[];
  classes: readonly { className: string; level: number }[];
};

const RACE_NAME_TO_ID: ReadonlyMap<string, string> = new Map(
  Object.values(RACAS).map((r) => [r.name, r.id]),
);

/**
 * Resolve stored race name → t20-data raça id. Case-insensitive fallback
 * because DB casing isn't enforced. Returns `undefined` when no match —
 * caller decides whether to pass it as raceId (skips racial mod).
 */
export function raceNameToId(name: string): string | undefined {
  const direct = RACE_NAME_TO_ID.get(name);
  if (direct) return direct;
  const norm = name.trim().toLowerCase();
  for (const [rname, rid] of RACE_NAME_TO_ID) {
    if (rname.toLowerCase() === norm) return rid;
  }
  return undefined;
}

/**
 * Build the CharacterInput expected by t20-data's orchestrator from a
 * DB row. Level uses the sum of `classes[].level` — the top-level
 * `level` column is a materialized total but classes are authoritative.
 * className picks the first class in the list (single-class characters
 * are the norm; multiclass is not yet resolved).
 */
export function toCharacterInput(row: CharacterDbRow): CharacterInput {
  const totalLevel = row.classes.reduce((sum, c) => sum + c.level, 0);
  const primaryClass = row.classes[0]?.className ?? 'Guerreiro';
  const raceId = row.races[0]
    ? raceNameToId(row.races[0].race)
    : undefined;
  return {
    level: totalLevel > 0 ? totalLevel : row.level,
    className: primaryClass,
    raceId,
    baseAttributes: {
      strength: row.strength,
      dexterity: row.dexterity,
      constitution: row.constitution,
      intelligence: row.intelligence,
      wisdom: row.wisdom,
      charisma: row.charisma,
    },
    currentPv: row.hpCurrent,
    currentPm: row.mpCurrent,
  };
}

export function computeSheetForRow(row: CharacterDbRow): ComputedSheet {
  return computeCharacterSheet(toCharacterInput(row));
}
