import {
  ATTRIBUTE_ABBR,
  applyActiveConditionals,
  classPowerModifiers,
  computeItemEffects,
  conditionalId,
  getCatalogItem,
  getClassPower,
  getGeneralPower,
  getOrigin,
  getRace,
  originModifiers,
  raceModifiers,
  requiredProficiency,
  statFor,
  trainingBonusForLevel,
  type ActiveItem,
  type AttributeKey,
  type CatalogItem,
  type ClassChoices,
  type ConditionalEffect,
  type ItemEffects,
  type Modifier,
  type Prerequisite,
} from '@tormenta20/t20-data'
import type { Character, CharacterExpertise, CharacterItem } from '@/shared/api/api'
import { useActiveConditionals } from '@/shared/stores/conditionals-store'

function activeItemsFor(character: Character): ActiveItem[] {
  const proficiencies = parseProficiencySetFromCharacter(character)
  const items: ActiveItem[] = character.items
    .filter((it) => it.equipped !== null)
    .map((it) => {
      const catalog = it.catalogId ? getCatalogItem(it.catalogId) : undefined
      const baseMods: Modifier[] = catalog?.modifiers ?? []
      const improvementIds = parseImprovementIds(it.improvements)
      const improvementMods = improvementIds.flatMap(
        (id) => getCatalogItem(id)?.modifiers ?? [],
      )
      const materialMods = it.material
        ? (getCatalogItem(it.material)?.modifiers ?? [])
        : []
      const penaltyMods = catalog
        ? nonProficiencyPenalties(catalog, proficiencies)
        : []
      return {
        source: it.name,
        equipped: it.equipped,
        modifiers: [
          ...baseMods,
          ...improvementMods,
          ...materialMods,
          ...penaltyMods,
        ],
      }
    })
  for (const eff of character.activeEffects ?? []) {
    const catalog = getCatalogItem(eff.catalogId)
    const sourceName = catalog?.name ?? eff.catalogId
    const modifiers = parseEffectModifiers(eff.modifiers)
    if (modifiers.length === 0) continue
    items.push({
      source: `${sourceName} (${eff.scope === 'day' ? 'dia' : 'cena'})`,
      equipped: 'vested',
      modifiers,
    })
  }
  const raceMods = raceActiveItems(character)
  items.push(...raceMods)
  const originMods = originActiveItem(character)
  if (originMods) items.push(originMods)
  const classMods = classActiveItems(character)
  items.push(...classMods)
  const generalMods = generalPowerActiveItem(character)
  if (generalMods) items.push(generalMods)
  return items
}

function classActiveItems(character: Character): ActiveItem[] {
  const chosen = parseChoiceSet(character.classPowers)
  const out: ActiveItem[] = []
  for (const entry of character.classes) {
    const mods = classPowerModifiers(entry.className, entry.level, chosen)
    if (mods.length === 0) continue
    out.push({
      source: `Classe: ${entry.className} ${entry.level}`,
      equipped: 'vested',
      modifiers: mods,
    })
  }
  return out
}

/**
 * General powers (Poder de Combate, etc.) live in their own catalog and are
 * stored in the same `classPowers` JSON blob — only ids starting with
 * `general.` resolve through getGeneralPower.
 */
function generalPowerActiveItem(character: Character): ActiveItem | null {
  const chosen = parseChoiceSet(character.classPowers)
  const mods: Modifier[] = []
  for (const id of chosen) {
    if (!id.startsWith('general.')) continue
    const power = getGeneralPower(id)
    if (power?.modifiers) mods.push(...power.modifiers)
  }
  if (mods.length === 0) return null
  return {
    source: 'Poderes Gerais',
    equipped: 'vested',
    modifiers: mods,
  }
}

function originActiveItem(character: Character): ActiveItem | null {
  const origin = getOrigin(character.origin)
  if (!origin) return null
  const choices = parseChoiceSet(character.originChoices)
  const mods = originModifiers(origin, choices)
  if (mods.length === 0) return null
  return {
    source: `Origem: ${origin.name}`,
    equipped: 'vested',
    modifiers: mods,
  }
}

function raceActiveItems(character: Character): ActiveItem[] {
  const variantChoices = parseChoiceSet(character.raceAbilityChoices)
  const result: ActiveItem[] = []
  for (const entry of character.races) {
    const race = getRace(entry.race)
    if (!race) continue
    const mods = raceModifiers(race, variantChoices)
    if (mods.length === 0) continue
    result.push({
      source: `Raça: ${race.name}`,
      equipped: 'vested',
      modifiers: mods,
    })
  }
  return result
}

function parseChoiceSet(raw: string): Set<string> {
  try {
    const parsed = JSON.parse(raw)
    if (Array.isArray(parsed)) {
      return new Set(parsed.filter((x): x is string => typeof x === 'string'))
    }
    return new Set()
  } catch {
    return new Set()
  }
}

export type PrerequisiteCheck = {
  prereq: Prerequisite
  met: boolean
  reason: string
}

/**
 * Auto-checks a single Prerequisite against the character. `power`/`anyPower`
 * test the chosen-class-power set; `trained` looks at the perícia row;
 * `attribute` compares raw character attribute; `classChoice` reads from
 * the parsed classChoices blob (devoto/caminho picks). `note` cannot be
 * auto-checked — returns `met:true` so it never blocks selection; UI shows
 * reason text as info hint.
 */
export function evaluatePrerequisite(
  prereq: Prerequisite,
  character: Character,
  chosenPowerIds: ReadonlySet<string>,
  classChoices: ClassChoices,
): PrerequisiteCheck {
  switch (prereq.kind) {
    case 'power': {
      const met = chosenPowerIds.has(prereq.id)
      return { prereq, met, reason: powerLabel(prereq.id) }
    }
    case 'anyPower': {
      const met = prereq.ids.some((id) => chosenPowerIds.has(id))
      const reason = prereq.ids.map(powerLabel).join(' ou ')
      return { prereq, met, reason }
    }
    case 'trained': {
      const exp = character.expertises.find((e) => e.name === prereq.expertise)
      return {
        prereq,
        met: exp?.trained === true,
        reason: `Treinado em ${prereq.expertise}`,
      }
    }
    case 'attribute': {
      const value = character[prereq.attr]
      return {
        prereq,
        met: value >= prereq.min,
        reason: `${ATTRIBUTE_ABBR[prereq.attr]} ${prereq.min}+`,
      }
    }
    case 'classChoice': {
      const value = classChoices[prereq.class]?.[prereq.field]
      let met = !!value
      if (met && prereq.allowed) met = prereq.allowed.includes(value!)
      if (met && prereq.forbidden) met = !prereq.forbidden.includes(value!)
      return { prereq, met, reason: prereq.label }
    }
    case 'note':
      return { prereq, met: true, reason: prereq.description }
  }
}

/**
 * Parses Character.classChoices JSON. Missing/malformed → empty object,
 * so the rest of the engine treats the character as having no choices yet.
 */
export function parseClassChoices(raw: string): ClassChoices {
  try {
    const parsed = JSON.parse(raw)
    if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
      return parsed as ClassChoices
    }
    return {}
  } catch {
    return {}
  }
}

function powerLabel(id: string): string {
  return getClassPower(id)?.name ?? getGeneralPower(id)?.name ?? id
}

function parseEffectModifiers(raw: string): Modifier[] {
  try {
    const parsed = JSON.parse(raw)
    if (Array.isArray(parsed)) return parsed as Modifier[]
    return []
  } catch {
    return []
  }
}

export function parseImprovementIds(raw: string): string[] {
  try {
    const parsed = JSON.parse(raw)
    if (Array.isArray(parsed)) {
      return parsed.filter((x): x is string => typeof x === 'string')
    }
    return []
  } catch {
    return []
  }
}

function parseProficiencySetFromCharacter(character: Character): Set<string> {
  try {
    const parsed = JSON.parse(character.proficiencies)
    if (Array.isArray(parsed)) {
      return new Set(parsed.filter((x): x is string => typeof x === 'string'))
    }
    return new Set()
  } catch {
    return new Set()
  }
}

/**
 * T20 p142: non-proficient weapon use → -5 attack. Armor/shield without
 * proficiency → cannot apply Dex to Defense and the armor penalty extends to
 * all expertise tests. Penalties are emitted as synthetic modifiers attached
 * to the offending ActiveItem so the standard engine resolves them alongside
 * catalog mods. Attack rolls are mapped onto the Luta/Pontaria perícia (T20
 * resolves attacks as expertise tests) so the penalty becomes visible in the
 * expertise breakdown rather than hiding inside a per-weapon target.
 */
function nonProficiencyPenalties(
  catalog: CatalogItem,
  proficiencies: ReadonlySet<string>,
): Modifier[] {
  const required = requiredProficiency(catalog)
  if (!required) return []
  if (proficiencies.has(required)) return []
  if (catalog.category.startsWith('weapon-')) {
    const purpose = catalog.weapon?.purpose
    const attackExpertise = purpose === 'melee' ? 'Luta' : 'Pontaria'
    return [
      {
        target: { k: 'attack', scope: 'this' },
        amount: -5,
        bonusType: 'untyped',
        condition: { c: 'wielded' },
        note: 'sem proficiência',
      },
      {
        target: { k: 'expertise', name: attackExpertise },
        amount: -5,
        bonusType: 'untyped',
        condition: { c: 'wielded' },
        note: `${catalog.name} sem proficiência`,
      },
    ]
  }
  const armorPenaltyMod = catalog.modifiers?.find(
    (m) => m.target.k === 'armorPenalty',
  )
  const basePenalty = armorPenaltyMod?.amount ?? -1
  return [
    {
      target: { k: 'flag', name: 'cannot-apply-dex-to-defense' },
      amount: 1,
      bonusType: 'untyped',
      condition: { c: 'vested' },
      note: 'sem proficiência',
    },
    {
      target: { k: 'expertiseAll' },
      amount: basePenalty,
      bonusType: 'untyped',
      condition: { c: 'vested' },
      note: `${catalog.name} sem proficiência`,
    },
  ]
}

/** Does the character have the proficiency required to use this item without penalty? */
export function isItemProficient(
  character: Character,
  item: CharacterItem,
): boolean {
  if (!item.catalogId) return true
  const catalog = getCatalogItem(item.catalogId)
  if (!catalog) return true
  const required = requiredProficiency(catalog)
  if (!required) return true
  const set = parseProficiencySetFromCharacter(character)
  return set.has(required)
}

export function characterEffects(
  character: Character,
  activeConditionals: ReadonlySet<string> = EMPTY_SET,
): ItemEffects {
  const base = computeItemEffects(activeItemsFor(character))
  return applyActiveConditionals(base, activeConditionals)
}

const EMPTY_SET: ReadonlySet<string> = new Set()

export function useCharacterEffects(character: Character): ItemEffects {
  const active = useActiveConditionals(character.id)
  return characterEffects(character, active)
}

export type ConditionalEntry = {
  id: string
  effect: ConditionalEffect
  active: boolean
}

export function useAllConditionals(character: Character): ConditionalEntry[] {
  const active = useActiveConditionals(character.id)
  const raw = computeItemEffects(activeItemsFor(character))
  return raw.conditional.map((effect) => {
    const id = conditionalId(effect)
    return { id, effect, active: active.has(id) }
  })
}

/**
 * Expertise total: ½ level + attribute + training + item modifiers
 * (typed, with non-stacking rules already resolved).
 */
const ARMOR_PENALTY_EXPERTISES = new Set([
  'Acrobacia',
  'Furtividade',
  'Ladinagem',
])

export function expertiseTotalWithItems(
  character: Character,
  state: CharacterExpertise,
  effects: ItemEffects,
): {
  base: number
  itemBonus: number
  total: number
  halfLevel: number
  attrValue: number
  training: number
  itemContributions: { source: string; amount: number }[]
  armorPenaltyApplied: number
} {
  const halfLevel = Math.floor(character.level / 2)
  const attrValue = attributeTotal(character, state.attribute, effects)
  const training = state.trained ? trainingBonusForLevel(character.level) : 0
  const base = halfLevel + attrValue + training
  const stat = statFor(effects, { k: 'expertise', name: state.name as never })
  const allStat = statFor(effects, { k: 'expertiseAll' })
  const byAttrStat = statFor(effects, {
    k: 'expertiseByAttribute',
    attribute: state.attribute,
  })
  const itemContributions = [
    ...stat.contributions,
    ...allStat.contributions,
    ...byAttrStat.contributions,
  ].map((c) => ({
    source: c.source,
    amount: c.amount,
  }))

  let armorPenaltyApplied = 0
  if (ARMOR_PENALTY_EXPERTISES.has(state.name)) {
    armorPenaltyApplied = armorPenaltyTotal(effects)
    if (armorPenaltyApplied !== 0) {
      itemContributions.push({
        source: 'Penalidade de armadura',
        amount: armorPenaltyApplied,
      })
    }
  }

  const itemBonus = stat.total + allStat.total + byAttrStat.total
  return {
    base,
    itemBonus: itemBonus + armorPenaltyApplied,
    total: base + itemBonus + armorPenaltyApplied,
    halfLevel,
    attrValue,
    training,
    itemContributions,
    armorPenaltyApplied,
  }
}

/**
 * Defense total: 10 + Dex (capped if heavy armor) + armor + shield + other typed mods.
 */
export function defenseTotal(
  character: Character,
  effects: ItemEffects,
): {
  base: number
  itemBonus: number
  total: number
  dexApplied: boolean
  contributions: { source: string; amount: number }[]
} {
  const stat = statFor(effects, { k: 'defense' })
  const dexApplied = !effects.flags.has('cannot-apply-dex-to-defense')
  const effectiveDex = attributeTotal(character, 'dexterity', effects)
  const base = 10 + (dexApplied ? effectiveDex : 0)
  return {
    base,
    itemBonus: stat.total,
    total: base + stat.total,
    dexApplied,
    contributions: stat.contributions.map((c) => ({
      source: c.source,
      amount: c.amount,
    })),
  }
}

export function displacementTotal(
  character: Character,
  effects: ItemEffects,
): {
  base: number
  itemBonus: number
  total: number
  contributions: { source: string; amount: number }[]
} {
  const stat = statFor(effects, { k: 'displacement' })
  return {
    base: character.displacement,
    itemBonus: stat.total,
    total: Math.max(0, character.displacement + stat.total),
    contributions: stat.contributions.map((c) => ({
      source: c.source,
      amount: c.amount,
    })),
  }
}

export function inventorySlotsTotal(
  character: Character,
  effects: ItemEffects,
): number {
  const effStr = attributeTotal(character, 'strength', effects)
  const base = 10 + 2 * Math.abs(effStr)
  const stat = statFor(effects, { k: 'inventorySlots' })
  return base + stat.total
}

/**
 * Effective attribute value = raw character attribute + sum of `attribute`
 * target modifiers (race bonuses, items, active effects). Negative bonuses
 * apply too.
 */
export function attributeTotal(
  character: Character,
  attr: AttributeKey,
  effects: ItemEffects,
): number {
  const raw = character[attr]
  const stat = statFor(effects, { k: 'attribute', name: attr })
  return raw + stat.total
}

export function attributeContributions(
  attr: AttributeKey,
  effects: ItemEffects,
): { source: string; amount: number }[] {
  return statFor(effects, { k: 'attribute', name: attr }).contributions.map(
    (c) => ({ source: c.source, amount: c.amount }),
  )
}

export function armorPenaltyTotal(effects: ItemEffects): number {
  return statFor(effects, { k: 'armorPenalty' }).total
}

export function pmLimitTotal(
  character: Character,
  effects: ItemEffects,
): {
  base: number
  itemBonus: number
  total: number
  contributions: { source: string; amount: number }[]
} {
  const base = Math.max(1, Math.floor(character.level / 2))
  const stat = statFor(effects, { k: 'pmLimit' })
  return {
    base,
    itemBonus: stat.total,
    total: base + stat.total,
    contributions: stat.contributions.map((c) => ({
      source: c.source,
      amount: c.amount,
    })),
  }
}

export function spellDCBonus(effects: ItemEffects): {
  total: number
  contributions: { source: string; amount: number }[]
} {
  const stat = statFor(effects, { k: 'spellDC' })
  return {
    total: stat.total,
    contributions: stat.contributions.map((c) => ({
      source: c.source,
      amount: c.amount,
    })),
  }
}

export function pmCostMod(effects: ItemEffects): {
  total: number
  contributions: { source: string; amount: number }[]
} {
  const stat = statFor(effects, { k: 'pmCost' })
  return {
    total: stat.total,
    contributions: stat.contributions.map((c) => ({
      source: c.source,
      amount: c.amount,
    })),
  }
}
