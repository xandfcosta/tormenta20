import {
  RACAS,
  SKILL_IDS,
  computeCharacterSheet,
  getCatalogItem,
  type CharacterEquipment,
  type CharacterInput,
  type ComputedSheet,
  type EquippedArmor,
  type EquippedShield,
  type EquippedWeapon,
  type SkillId,
} from '@tormenta20/t20-data';

/**
 * Backend DB `Character` shape → t20-data `CharacterInput`, then runs
 * the orchestrator to produce a derived sheet.
 *
 * v2 output (Fase A followup) covers:
 *   - attributes + vitals + saves + movement (Fase A)
 *   - trained skills (PT expertise name → SkillId)
 *   - equipment (armor / shield / weapon from equipped catalog items)
 *   - armor penalty derived from armor + shield
 *
 * Still missing (later phases):
 *   - activeEffects (needs modifier decoder from JSON blob)
 *   - improvements/material overlays on equipment stats
 *   - condition-derived mods (Fraco, Cego…)
 */

export type CharacterExpertiseRow = {
  name: string;
  attribute: string;
  trained: boolean;
  custom: boolean;
};

export type CharacterItemRow = {
  catalogId: string | null;
  name: string;
  equipped: string | null;
};

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
  expertises: readonly CharacterExpertiseRow[];
  items: readonly CharacterItemRow[];
};

// ─── Race ────────────────────────────────────────────────────────

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

// ─── Skills ──────────────────────────────────────────────────────

/**
 * PT expertise name → SkillId (kebab lowercase ASCII). Deterministic:
 * strip diacritics + lowercase. Tabela 2-1 (livro p115) tem 29 perícias
 * e todos os SKILL_IDs derivam do nome PT dessa forma.
 */
export function expertiseNameToSkillId(name: string): SkillId | undefined {
  const normalized = name
    .normalize('NFD')
    .replace(/[̀-ͯ]/g, '')
    .toLowerCase()
    .trim();
  if ((SKILL_IDS as readonly string[]).includes(normalized)) {
    return normalized as SkillId;
  }
  return undefined;
}

function trainedSkillsFrom(
  expertises: readonly CharacterExpertiseRow[],
): SkillId[] {
  const out: SkillId[] = [];
  for (const e of expertises) {
    if (!e.trained) continue;
    const id = expertiseNameToSkillId(e.name);
    if (id) out.push(id);
  }
  return out;
}

// ─── Equipment ───────────────────────────────────────────────────

/**
 * Parse equipped catalog items into slots. Only items with a resolved
 * catalog entry contribute (custom items lack the mechanical stats
 * needed for the orchestrator). Overlay stacks (improvements /
 * material) aren't applied — that's a follow-up when the encoder for
 * overlay math ships.
 *
 * Slot assignment rules:
 *   - Armor rows fill `armor` (first match wins if user equipped two —
 *     player error, not the mapper's problem to resolve).
 *   - Shield rows fill `shield`.
 *   - Weapons: `wielded2` (two-handed) → mainHand only, no offHand
 *     (occupies both hands implicitly). `wielded` → mainHand first,
 *     then offHand for the second weapon.
 */
export function equipmentFromRow(
  row: Pick<CharacterDbRow, 'items'>,
): CharacterEquipment | undefined {
  const equipment: CharacterEquipment = {};
  for (const item of row.items) {
    if (!item.catalogId || !item.equipped) continue;
    const catalog = getCatalogItem(item.catalogId);
    if (!catalog) continue;
    if (catalog.armor && !equipment.armor) {
      equipment.armor = toEquippedArmor(catalog.name, catalog.armor);
    }
    if (catalog.shield && !equipment.shield) {
      equipment.shield = toEquippedShield(catalog.name, catalog.shield);
    }
    if (catalog.weapon) {
      const weapon = toEquippedWeapon(catalog.name, catalog.weapon);
      if (!equipment.mainHand) {
        equipment.mainHand = weapon;
      } else if (item.equipped !== 'wielded2' && !equipment.offHand) {
        equipment.offHand = weapon;
      }
    }
  }
  const hasAny =
    equipment.armor ||
    equipment.shield ||
    equipment.mainHand ||
    equipment.offHand;
  return hasAny ? equipment : undefined;
}

function toEquippedArmor(
  name: string,
  stats: { defense: number; penalty: number; heavy: boolean },
): EquippedArmor {
  return {
    name,
    defense: stats.defense,
    penalty: stats.penalty,
    heavy: stats.heavy,
  };
}

function toEquippedShield(
  name: string,
  stats: { defense: number; penalty: number; heavy: boolean },
): EquippedShield {
  return {
    name,
    defense: stats.defense,
    penalty: stats.penalty,
    heavy: stats.heavy,
  };
}

function toEquippedWeapon(
  name: string,
  stats: import('@tormenta20/t20-data').WeaponStats,
): EquippedWeapon {
  return {
    name,
    hand: stats.hand,
    purpose: stats.purpose,
    damage: stats.damage,
    critRange: stats.critRange,
    critMult: stats.critMult,
    damageType: stats.type,
  };
}

// ─── Orchestrator input ──────────────────────────────────────────

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
  const trainedSkills = trainedSkillsFrom(row.expertises);
  const equipment = equipmentFromRow(row);
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
    trainedSkills,
    equipment,
  };
}

export function computeSheetForRow(row: CharacterDbRow): ComputedSheet {
  return computeCharacterSheet(toCharacterInput(row));
}
