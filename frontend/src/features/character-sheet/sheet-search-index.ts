import { EXPERTISES } from '@/shared/api/expertise-names'
import {
  conditionsRecord,
  tormentaPowersRecord,
} from '@/shared/lib/rules-catalog-cache'
import {
  getClassPower,
  getGeneralPower,
  getOrigin,
  getRace,
  ownedClassPowers,
} from '@/shared/lib/abilities-cache'
import { spellById } from '@/shared/lib/spell-cache'
import type { Character } from '@/shared/api/api'
import {
  computedSheetFor,
  expertiseFromSheet,
} from '@/entities/character/computed-sheet'
import { parseClassChoices } from '@/entities/character/derived'
import { parseActiveConditions } from './active-conditions'

export type SheetSearchEntry = {
  /** Display name ("Furtividade", "Totem Espiritual", "Adaga"). */
  name: string
  /** The inline ANSWER — total, quantity, rule text… lookup without routing. */
  detail: string
  /** Badge: where this fact lives ("Perícia", "Raça", "Item"…). */
  source: string
  /** Sheet tab that owns the fact (palette navigates there on select). */
  tab: string
  /** Catalog id of an owned power/ability — lets the flat Poderes results
   *  resolve its ActivationSpec by id before falling back to the name. */
  powerId?: string
}

function parseIds(raw: string): string[] {
  try {
    const parsed = JSON.parse(raw)
    return Array.isArray(parsed)
      ? parsed.filter((x): x is string => typeof x === 'string')
      : []
  } catch {
    return []
  }
}

/**
 * Everything on the sheet, flattened for the 3-second lookup: perícias with
 * their live totals, owned powers/abilities with rule text, items with
 * quantities, learned spells, active conditions. Built on demand when the
 * palette opens (cheap: a few hundred entries max).
 */
export function buildSheetSearchIndex(character: Character): SheetSearchEntry[] {
  const sheet = computedSheetFor(character)
  const out: SheetSearchEntry[] = []

  for (const def of EXPERTISES) {
    const total = expertiseFromSheet(sheet, def.name)?.total ?? 0
    out.push({
      name: def.name,
      detail: `${total >= 0 ? '+' : ''}${total}`,
      source: 'Perícia',
      tab: 'expertises',
    })
  }

  for (const abilityEntry of ownedAbilities(character)) out.push(abilityEntry)

  for (const item of character.items) {
    out.push({
      name: item.name,
      detail: `×${item.quantity}${item.equipped ? ' · equipado' : ''}`,
      source: 'Item',
      tab: item.equipped ? 'equipment' : 'inventory',
    })
  }

  for (const spell of character.spells) {
    try {
      const cat = spellById(spell.catalogSpellId)
      out.push({
        name: cat.name,
        detail: `${cat.circle}º círculo · ${cat.school}`,
        source: 'Magia',
        tab: 'spells',
      })
    } catch {
      // unknown catalog id — skip
    }
  }

  const conditions = conditionsRecord()
  for (const id of parseActiveConditions(character.activeConditions)) {
    out.push({
      name: conditions[id].name,
      detail: conditions[id].description,
      source: 'Condição',
      tab: 'conditionals',
    })
  }

  return out
}

/** Owned abilities/powers across every source, with rule text as the answer. */
const EMPTY_IDS: ReadonlySet<string> = new Set()

export function ownedAbilities(character: Character): SheetSearchEntry[] {
  const out: SheetSearchEntry[] = []
  for (const { race } of character.races) {
    const def = getRace(race)
    for (const ability of def?.abilities ?? []) {
      out.push({
        name: ability.name,
        detail: ability.description,
        source: `Raça · ${race}`,
        tab: 'abilities',
        powerId: ability.id,
      })
    }
  }
  const origin = getOrigin(character.origin)
  if (origin) {
    const chosen = new Set(parseIds(character.originChoices))
    for (const benefit of [...origin.benefits, origin.poderUnico]) {
      if (!chosen.has(benefit.id)) continue
      out.push({
        name: benefit.name,
        detail: benefit.description,
        source: `Origem · ${origin.name}`,
        tab: 'abilities',
        powerId: benefit.id,
      })
    }
  }
  // Class AUTOS (Fúria, Instinto Selvagem…) are owned by level, not chosen —
  // without this loop searching "fúria" found nothing (2026-08 bug report).
  // classChoices surfaces grantedByChoice rows ("Caminho: Mago").
  const classChoices = parseClassChoices(character.classChoices)
  for (const entry of character.classes) {
    for (const power of ownedClassPowers(
      entry.className,
      entry.level,
      EMPTY_IDS,
      classChoices[entry.className],
    )) {
      out.push({
        name: power.name,
        detail: power.description,
        source: `Classe · ${entry.className}`,
        tab: 'abilities',
        powerId: power.id,
      })
    }
  }
  const tormenta = tormentaPowersRecord()
  for (const id of parseIds(character.classPowers)) {
    const power =
      getClassPower(id) ??
      getGeneralPower(id) ??
      (id in tormenta ? tormenta[id as keyof typeof tormenta] : undefined)
    if (!power) continue
    const source = getClassPower(id)
      ? `Classe · ${(power as { className?: string }).className ?? ''}`
      : id in tormenta
        ? 'Poder da Tormenta'
        : 'Poder geral'
    out.push({ name: power.name, detail: power.description, source, tab: 'abilities', powerId: id })
  }
  return out
}
