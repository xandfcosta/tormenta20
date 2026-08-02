import {
  EXPERTISES,
  getClassPower,
  getGeneralPower,
  getOrigin,
  getRace,
  TORMENTA_POWERS,
} from '@tormenta20/t20-data'
import { spellById } from '@tormenta20/t20-data'
import type { Character } from '@/shared/api/api'
import {
  characterEffects,
  expertiseTotalWithItems,
} from '@/entities/character/derived'
import { expertiseStateFor } from '@/entities/character/expertise'
import { parseActiveConditions } from './conditions-section'
import { CONDITIONS } from '@tormenta20/t20-data'

export type SheetSearchEntry = {
  /** Display name ("Furtividade", "Totem Espiritual", "Adaga"). */
  name: string
  /** The inline ANSWER — total, quantity, rule text… lookup without routing. */
  detail: string
  /** Badge: where this fact lives ("Perícia", "Raça", "Item"…). */
  source: string
  /** Sheet tab that owns the fact (palette navigates there on select). */
  tab: string
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
  const effects = characterEffects(character)
  const out: SheetSearchEntry[] = []

  for (const def of EXPERTISES) {
    const state = expertiseStateFor(character, {
      name: def.name,
      attribute: def.attribute,
      abbr: def.name.slice(0, 3).toUpperCase(),
    })
    const total = expertiseTotalWithItems(character, state, effects).total
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

  for (const id of parseActiveConditions(character.activeConditions)) {
    out.push({
      name: CONDITIONS[id].name,
      detail: CONDITIONS[id].description,
      source: 'Condição',
      tab: 'conditionals',
    })
  }

  return out
}

/** Owned abilities/powers across every source, with rule text as the answer. */
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
      })
    }
  }
  for (const id of parseIds(character.classPowers)) {
    const power =
      getClassPower(id) ??
      getGeneralPower(id) ??
      (id in TORMENTA_POWERS
        ? TORMENTA_POWERS[id as keyof typeof TORMENTA_POWERS]
        : undefined)
    if (!power) continue
    const source = getClassPower(id)
      ? `Classe · ${(power as { className?: string }).className ?? ''}`
      : id in TORMENTA_POWERS
        ? 'Poder da Tormenta'
        : 'Poder geral'
    out.push({ name: power.name, detail: power.description, source, tab: 'abilities' })
  }
  return out
}
