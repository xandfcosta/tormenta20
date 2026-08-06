import {
  ATTRIBUTE_KEYS,
  type AttributeKey,
  type CharacterEquipment,
  type CharacterInput,
  type DeformidadeChoice,
  type EquippedArmor,
  type EquippedShield,
  type EquippedWeapon,
  expertiseNameToSkillId,
  type SkillId,
} from '@tormenta20/t20-data'
import { getOriginBenefit, raceWithDeformidade } from '@/shared/lib/abilities-cache'
import { getCatalogItem } from '@/shared/lib/catalog-cache'
import { racasRecord } from '@/shared/lib/racas-cache'
import type { Character, CharacterExpertise } from '@/shared/api/api'

/**
 * Front mirror of the backend `toCharacterInput` (character-sheet.mapper.ts):
 * `Character` (API shape) → t20-data `CharacterInput`, the input the Go/WASM
 * engine consumes. Same field names as the DB row, so this tracks the backend
 * mapper 1:1 — a parity test (to-character-input.test) guards drift against the
 * bench payloads (which ARE the backend's output). Catalog lookups go through
 * the front caches (getCatalogItem/getOriginBenefit/racasRecord) instead of the
 * bundled data (project_front_decouple_catalog Fase 3 → WASM engine).
 */
const ATTRIBUTE_KEY_SET: ReadonlySet<string> = new Set(ATTRIBUTE_KEYS)

/** Race display name → racas slug, off the primed racas cache (name-insensitive
 *  fallback like the backend). */
function raceNameToId(name: string): string | undefined {
  const byName = racasRecord()
  for (const r of Object.values(byName)) if (r.name === name) return r.id
  const norm = name.trim().toLowerCase()
  for (const r of Object.values(byName)) {
    if (r.name.toLowerCase() === norm) return r.id
  }
  return undefined
}

function jsonStringArray(raw: string | null | undefined): string[] {
  if (!raw) return []
  try {
    const parsed: unknown = JSON.parse(raw)
    return Array.isArray(parsed)
      ? parsed.filter((x): x is string => typeof x === 'string')
      : []
  } catch {
    return []
  }
}

function trainedSkillsFrom(expertises: readonly CharacterExpertise[]): SkillId[] {
  const out: SkillId[] = []
  for (const e of expertises) {
    if (!e.trained) continue
    const id = expertiseNameToSkillId(e.name)
    if (id) out.push(id)
  }
  return out
}

function toEquippedArmor(
  name: string,
  s: { defense: number; penalty: number; heavy: boolean },
): EquippedArmor {
  return { name, defense: s.defense, penalty: s.penalty, heavy: s.heavy }
}
function toEquippedShield(
  name: string,
  s: { defense: number; penalty: number; heavy: boolean },
): EquippedShield {
  return { name, defense: s.defense, penalty: s.penalty, heavy: s.heavy }
}
function toEquippedWeapon(
  name: string,
  s: NonNullable<ReturnType<typeof getCatalogItem>>['weapon'] & object,
): EquippedWeapon {
  return {
    name,
    hand: s.hand,
    purpose: s.purpose,
    damage: s.damage,
    critRange: s.critRange,
    critMult: s.critMult,
    damageType: s.type,
  }
}

/** Equipped items → the engine's CharacterEquipment (armor/shield/2 hands),
 *  resolving stats from the primed item cache. */
function equipmentFrom(
  items: Character['items'],
): CharacterEquipment | undefined {
  const equipment: CharacterEquipment = {}
  for (const item of items) {
    if (!item.catalogId || !item.equipped) continue
    const catalog = getCatalogItem(item.catalogId)
    if (!catalog) continue
    if (catalog.armor && !equipment.armor) {
      equipment.armor = toEquippedArmor(catalog.name, catalog.armor)
    }
    if (catalog.shield && !equipment.shield) {
      equipment.shield = toEquippedShield(catalog.name, catalog.shield)
    }
    if (catalog.weapon) {
      const weapon = toEquippedWeapon(catalog.name, catalog.weapon)
      if (!equipment.mainHand) equipment.mainHand = weapon
      else if (item.equipped !== 'wielded2' && !equipment.offHand)
        equipment.offHand = weapon
    }
  }
  const hasAny =
    equipment.armor || equipment.shield || equipment.mainHand || equipment.offHand
  return hasAny ? equipment : undefined
}

function parseDeformidade(raw: unknown): DeformidadeChoice | undefined {
  if (!raw || typeof raw !== 'object') return undefined
  const d = raw as { pericias?: unknown; tormentaPower?: unknown }
  if (!Array.isArray(d.pericias)) return undefined
  const pericias = d.pericias.filter((x): x is string => typeof x === 'string')
  const tormentaPower =
    typeof d.tormentaPower === 'string' && d.tormentaPower
      ? d.tormentaPower
      : undefined
  return { pericias, tormentaPower } as DeformidadeChoice
}

function parseRaceAttributeChoices(raw: string | null | undefined): {
  floatingPicks: AttributeKey[]
  ascendencia?: string
  deformidade?: DeformidadeChoice
} {
  if (!raw) return { floatingPicks: [] }
  try {
    const p = JSON.parse(raw) as {
      floatingPicks?: unknown
      ascendencia?: unknown
      deformidade?: unknown
    }
    const floatingPicks = Array.isArray(p.floatingPicks)
      ? p.floatingPicks.filter(
          (x): x is AttributeKey =>
            typeof x === 'string' && ATTRIBUTE_KEY_SET.has(x),
        )
      : []
    const ascendencia =
      typeof p.ascendencia === 'string' && p.ascendencia ? p.ascendencia : undefined
    return { floatingPicks, ascendencia, deformidade: parseDeformidade(p.deformidade) }
  } catch {
    return { floatingPicks: [] }
  }
}

function parseSecondaryRaces(
  raw: string | null | undefined,
): { raceId: string; floatingPicks?: AttributeKey[]; ascendencia?: string }[] {
  if (!raw) return []
  let parsed: unknown
  try {
    parsed = JSON.parse(raw)
  } catch {
    return []
  }
  if (!Array.isArray(parsed)) return []
  const out: { raceId: string; floatingPicks?: AttributeKey[]; ascendencia?: string }[] = []
  for (const entry of parsed) {
    const e = entry as { race?: unknown; floatingPicks?: unknown; ascendencia?: unknown }
    if (typeof e.race !== 'string') continue
    const raceId = raceNameToId(e.race)
    if (!raceId) continue
    const floatingPicks = Array.isArray(e.floatingPicks)
      ? e.floatingPicks.filter(
          (x): x is AttributeKey => typeof x === 'string' && ATTRIBUTE_KEY_SET.has(x),
        )
      : undefined
    const ascendencia =
      typeof e.ascendencia === 'string' && e.ascendencia ? e.ascendencia : undefined
    out.push({ raceId, floatingPicks, ascendencia })
  }
  return out
}

function secondaryDeformidade(
  raw: string | null | undefined,
): DeformidadeChoice | undefined {
  if (!raw) return undefined
  let parsed: unknown
  try {
    parsed = JSON.parse(raw)
  } catch {
    return undefined
  }
  if (!Array.isArray(parsed)) return undefined
  for (const entry of parsed) {
    const e = entry as { race?: unknown; deformidade?: unknown }
    if (typeof e.race !== 'string' || !raceWithDeformidade([e.race])) continue
    const choice = parseDeformidade(e.deformidade)
    if (choice) return choice
  }
  return undefined
}

function classChoicesFrom(
  raw: string | null | undefined,
): Record<string, { devoto?: string; caminho?: string }> | undefined {
  if (!raw) return undefined
  try {
    const parsed: unknown = JSON.parse(raw)
    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) return undefined
    const out: Record<string, { devoto?: string; caminho?: string }> = {}
    for (const [cls, blob] of Object.entries(parsed)) {
      if (!blob || typeof blob !== 'object') continue
      const b = blob as { devoto?: unknown; caminho?: unknown }
      out[cls] = {
        ...(typeof b.devoto === 'string' ? { devoto: b.devoto } : {}),
        ...(typeof b.caminho === 'string' ? { caminho: b.caminho } : {}),
      }
    }
    return out
  } catch {
    return undefined
  }
}

/** Origin-benefit powerPick sub-choices → owned power ids (mirror backend). */
function originPickedPowerIds(c: Character): string[] {
  const chosen = jsonStringArray(c.originChoices)
  if (chosen.length === 0 || !c.powerChoices) return []
  let blob: unknown
  try {
    blob = JSON.parse(c.powerChoices)
  } catch {
    return []
  }
  if (!blob || typeof blob !== 'object') return []
  const choices = blob as Record<string, unknown>
  const out: string[] = []
  for (const benefitId of chosen) {
    if (!getOriginBenefit(benefitId)?.powerPick) continue
    const picked = choices[benefitId]
    if (!Array.isArray(picked)) continue
    out.push(...picked.filter((x): x is string => typeof x === 'string'))
  }
  return out
}

function deformidadeFrom(
  c: Character,
  primary: { deformidade?: DeformidadeChoice },
): DeformidadeChoice | undefined {
  const primaryName = c.races[0]?.race
  if (primaryName && raceWithDeformidade([primaryName]) && primary.deformidade) {
    return primary.deformidade
  }
  return secondaryDeformidade(c.secondaryRaceChoices)
}

/**
 * Build the CharacterInput the Go/WASM engine consumes from an API `Character`.
 * 1:1 mirror of backend `toCharacterInput`; requires the item/abilities/racas
 * caches to be primed (loader gate).
 */
export function characterToInput(c: Character): CharacterInput {
  const totalLevel = c.classes.reduce((sum, cl) => sum + cl.level, 0)
  const primaryClass = c.classes[0]?.className ?? 'Guerreiro'
  const raceId = c.races[0] ? raceNameToId(c.races[0].race) : undefined
  const raceAttr = parseRaceAttributeChoices(c.raceAttributeChoices)
  return {
    level: totalLevel > 0 ? totalLevel : c.level,
    className: primaryClass,
    classes: c.classes.map((cl) => ({ className: cl.className, level: cl.level })),
    raceId,
    raceFloatingPicks: raceAttr.floatingPicks,
    raceAscendencia: raceAttr.ascendencia,
    additionalRaces: parseSecondaryRaces(c.secondaryRaceChoices),
    baseAttributes: {
      strength: c.strength,
      dexterity: c.dexterity,
      constitution: c.constitution,
      intelligence: c.intelligence,
      wisdom: c.wisdom,
      charisma: c.charisma,
    },
    currentPv: c.hpCurrent,
    currentPm: c.mpCurrent,
    trainedSkills: trainedSkillsFrom(c.expertises),
    equipment: equipmentFrom(c.items),
    origin: c.origin ?? undefined,
    godPower: c.godPower || undefined,
    powerIds: [...new Set([...jsonStringArray(c.classPowers), ...originPickedPowerIds(c)])],
    originChoices: jsonStringArray(c.originChoices),
    raceAbilityChoices: jsonStringArray(c.raceAbilityChoices),
    classChoices: classChoicesFrom(c.classChoices),
    activeConditions: jsonStringArray(
      c.activeConditions,
    ) as CharacterInput['activeConditions'],
    deformidade: deformidadeFrom(c, raceAttr),
  }
}
