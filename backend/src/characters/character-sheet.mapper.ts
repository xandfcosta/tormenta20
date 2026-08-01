import {
  ATTRIBUTE_KEYS,
  type AttributeKey,
  RACAS,
  SKILL_IDS,
  computeCharacterSheet,
  getCatalogItem,
  raceWithDeformidade,
  type CharacterEquipment,
  type CharacterInput,
  type ComputedSheet,
  type DeformidadeChoice,
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
  /** Origin name (matches an ORIGINS_CATALOG id). */
  origin?: string | null;
  /** JSON string[] columns — picked class/general powers, origin benefits,
   *  and chosen race-ability variant ids. Feed the passive PV/PM pipeline. */
  classPowers?: string | null;
  originChoices?: string | null;
  raceAbilityChoices?: string | null;
  /** JSON { floatingPicks?: string[]; ascendencia?: string } for the primary
   *  race — derives the racial attribute mod from the BASE stored attributes. */
  raceAttributeChoices?: string | null;
  /** JSON { race, floatingPicks?, ascendencia? }[] — opted-in secondary races. */
  secondaryRaceChoices?: string | null;
  races: readonly { race: string }[];
  classes: readonly { className: string; level: number }[];
  expertises: readonly CharacterExpertiseRow[];
  items: readonly CharacterItemRow[];
};

/** Parse a JSON-encoded `string[]` column defensively (bad blob ⇒ empty). */
function jsonStringArray(raw: string | null | undefined): string[] {
  if (!raw) return [];
  try {
    const parsed: unknown = JSON.parse(raw);
    return Array.isArray(parsed)
      ? parsed.filter((x): x is string => typeof x === 'string')
      : [];
  } catch {
    return [];
  }
}

const ATTRIBUTE_KEY_SET: ReadonlySet<string> = new Set(ATTRIBUTE_KEYS);

/** Parse opted-in secondary races → CharacterInput.additionalRaces (name→slug). */
function parseSecondaryRaces(
  raw: string | null | undefined,
): { raceId: string; floatingPicks?: AttributeKey[]; ascendencia?: string }[] {
  if (!raw) return [];
  let parsed: unknown;
  try {
    parsed = JSON.parse(raw);
  } catch {
    return [];
  }
  if (!Array.isArray(parsed)) return [];
  const out: {
    raceId: string;
    floatingPicks?: AttributeKey[];
    ascendencia?: string;
  }[] = [];
  for (const entry of parsed) {
    const e = entry as {
      race?: unknown;
      floatingPicks?: unknown;
      ascendencia?: unknown;
    };
    if (typeof e.race !== 'string') continue;
    const raceId = raceNameToId(e.race);
    if (!raceId) continue;
    const floatingPicks = Array.isArray(e.floatingPicks)
      ? e.floatingPicks.filter(
          (x): x is AttributeKey =>
            typeof x === 'string' && ATTRIBUTE_KEY_SET.has(x),
        )
      : undefined;
    const ascendencia =
      typeof e.ascendencia === 'string' && e.ascendencia
        ? e.ascendencia
        : undefined;
    out.push({ raceId, floatingPicks, ascendencia });
  }
  return out;
}

/** Parse the race attribute-choices JSON into typed floating picks + ascendência. */
function parseRaceAttributeChoices(raw: string | null | undefined): {
  floatingPicks: AttributeKey[];
  ascendencia?: string;
  deformidade?: DeformidadeChoice;
} {
  if (!raw) return { floatingPicks: [] };
  try {
    const p = JSON.parse(raw) as {
      floatingPicks?: unknown;
      ascendencia?: unknown;
      deformidade?: unknown;
    };
    const floatingPicks = Array.isArray(p.floatingPicks)
      ? p.floatingPicks.filter(
          (x): x is AttributeKey =>
            typeof x === 'string' && ATTRIBUTE_KEY_SET.has(x),
        )
      : [];
    const ascendencia =
      typeof p.ascendencia === 'string' && p.ascendencia
        ? p.ascendencia
        : undefined;
    return { floatingPicks, ascendencia, deformidade: parseDeformidade(p.deformidade) };
  } catch {
    return { floatingPicks: [] };
  }
}

/** Parse a persisted deformidade blob (bad shape ⇒ undefined; engine validates ids). */
function parseDeformidade(raw: unknown): DeformidadeChoice | undefined {
  if (!raw || typeof raw !== 'object') return undefined;
  const d = raw as { pericias?: unknown; tormentaPower?: unknown };
  if (!Array.isArray(d.pericias)) return undefined;
  const pericias = d.pericias.filter((x): x is string => typeof x === 'string');
  const tormentaPower =
    typeof d.tormentaPower === 'string' && d.tormentaPower
      ? d.tormentaPower
      : undefined;
  return { pericias, tormentaPower } as DeformidadeChoice;
}

/**
 * Deformidade escolhida — da raça primária ou da primeira secundária aplicada
 * que realmente possua a habilidade (Lefou, p23). Blobs em raças sem a
 * habilidade são ignorados (dados stale, não erro).
 */
function deformidadeFromRow(
  row: CharacterDbRow,
  primary: { deformidade?: DeformidadeChoice },
): DeformidadeChoice | undefined {
  const primaryName = row.races[0]?.race;
  if (primaryName && raceWithDeformidade([primaryName]) && primary.deformidade) {
    return primary.deformidade;
  }
  return secondaryDeformidade(row.secondaryRaceChoices);
}

function secondaryDeformidade(
  raw: string | null | undefined,
): DeformidadeChoice | undefined {
  if (!raw) return undefined;
  let parsed: unknown;
  try {
    parsed = JSON.parse(raw);
  } catch {
    return undefined;
  }
  if (!Array.isArray(parsed)) return undefined;
  for (const entry of parsed) {
    const e = entry as { race?: unknown; deformidade?: unknown };
    if (typeof e.race !== 'string' || !raceWithDeformidade([e.race])) continue;
    const choice = parseDeformidade(e.deformidade);
    if (choice) return choice;
  }
  return undefined;
}

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
  // Stored attributes are BASE (pre-race); the orchestrator applies the racial
  // mod ONCE from the persisted floating-pick / ascendência choices.
  const raceAttr = parseRaceAttributeChoices(row.raceAttributeChoices);
  const additionalRaces = parseSecondaryRaces(row.secondaryRaceChoices);
  return {
    level: totalLevel > 0 ? totalLevel : row.level,
    className: primaryClass,
    raceId,
    raceFloatingPicks: raceAttr.floatingPicks,
    raceAscendencia: raceAttr.ascendencia,
    additionalRaces,
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
    origin: row.origin ?? undefined,
    powerIds: jsonStringArray(row.classPowers),
    originChoices: jsonStringArray(row.originChoices),
    raceAbilityChoices: jsonStringArray(row.raceAbilityChoices),
    deformidade: deformidadeFromRow(row, raceAttr),
  };
}

export function computeSheetForRow(row: CharacterDbRow): ComputedSheet {
  return computeCharacterSheet(toCharacterInput(row));
}
