import type { Modifier } from '@tormenta20/t20-data'

/**
 * Human-readable labels for the two Modifier fields that need to be
 * displayed inside the item info + catalog dialogs. Kept exhaustive
 * so a new Modifier variant surfaces as a TS error rather than a
 * silently-empty catalog row.
 */
export function describeModifierTarget(t: Modifier['target']): string {
  switch (t.k) {
    case 'expertise':
      return `Perícia ${t.name}`
    case 'expertiseAll':
      return 'Todas perícias'
    case 'expertiseRemovePenalty':
      return `Remove penalidade em ${t.name}`
    case 'expertiseByAttribute':
      return `Perícias de ${t.attribute}`
    case 'attribute':
      return `Atributo ${t.name}`
    case 'defense':
      return 'Defesa'
    case 'defenseDexCap':
      return 'Limite de Des na Defesa'
    case 'resistance':
      return 'Resistências'
    case 'fearResistance':
      return 'Resistência a medo'
    case 'attack':
      return `Ataque (${t.scope})`
    case 'damage':
      return `Dano (${t.scope})`
    case 'critRange':
      return 'Margem de ameaça'
    case 'critMult':
      return 'Multiplicador crítico'
    case 'pmLimit':
      return 'Limite de PM por magia'
    case 'pmCost':
      return 'Custo em PM'
    case 'spellDC':
      return 'CD de magias'
    case 'inventorySlots':
      return 'Espaços de carga'
    case 'displacement':
      return 'Deslocamento'
    case 'armorPenalty':
      return 'Penalidade de armadura'
    case 'armorPenaltyExpertises':
      return 'Penalidade em perícias'
    case 'tempHp':
      return 'PV temporários'
    case 'tempMp':
      return 'PM temporários'
    case 'maneuver':
      return `Manobra ${t.name}`
    case 'flag':
      return `Efeito: ${t.name}`
  }
}

export function describeCondition(m: Modifier): string | null {
  if (!m.condition) return null
  switch (m.condition.c) {
    case 'always':
      return null
    case 'wielded':
      return 'enquanto empunhado'
    case 'vested':
      return 'enquanto vestido'
    case 'terrain':
      return `terreno: ${m.condition.type}`
    case 'against':
      return `contra: ${m.condition.trait}`
    case 'context':
      return m.condition.note
    case 'flagOn':
      return m.condition.label
  }
}

/** Number formatter shared between inventory + catalog dialogs. */
export function formatLoad(n: number): string {
  if (Number.isInteger(n)) return String(n)
  return n.toFixed(1).replace('.', ',')
}
