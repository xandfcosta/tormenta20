import type { Modifier } from '@tormenta20/t20-data'

// Attribute keys arrive raw (e.g. 'charisma') on catalog modifiers — map to
// the pt-BR name so a row reads "Carisma +2" instead of "charisma +2".
const ATTRIBUTE_PT: Record<string, string> = {
  strength: 'Força',
  dexterity: 'Destreza',
  constitution: 'Constituição',
  intelligence: 'Inteligência',
  wisdom: 'Sabedoria',
  charisma: 'Carisma',
}

/**
 * Human-readable label for a Modifier target — used inside the
 * conditional rows so a player can read "Ataque +2" instead of
 * `{k:'attack', scope:'all'}`. Kept exhaustive so a missed case is a
 * TypeScript error rather than a silently-empty row.
 *
 * @example describeConditionalTarget({ k: 'attack', scope: 'all' }) // 'Ataque'
 */
export function describeConditionalTarget(target: Modifier['target']): string {
  switch (target.k) {
    case 'expertise':
      return target.name
    case 'expertiseAll':
      return 'todas perícias'
    case 'expertiseRemovePenalty':
      return `${target.name} (sem penalidade)`
    case 'expertiseByAttribute':
      return `Perícias de ${target.attribute}`
    case 'attribute':
      return ATTRIBUTE_PT[target.name] ?? target.name
    case 'defense':
      return 'Defesa'
    case 'defenseDexCap':
      return 'limite de Des na Defesa'
    case 'resistance':
      return 'Resistência'
    case 'fearResistance':
      return 'Resistência (medo)'
    case 'attack':
      return target.scope === 'this' ? 'Ataque (esta arma)' : 'Ataque'
    case 'damage':
      return target.scope === 'this' ? 'Dano (esta arma)' : 'Dano'
    case 'critRange':
      return 'Margem de crítico'
    case 'critMult':
      return 'Multiplicador de crítico'
    case 'pmLimit':
      return 'Limite de PM'
    case 'pmCost':
      return 'Custo de PM'
    case 'damageReduction':
      return 'Redução de dano'
    case 'catalyst':
      return `Catalisador ${target.school}`
    case 'spellDC':
      return 'CD Magia'
    case 'inventorySlots':
      return 'Espaços'
    case 'displacement':
      return 'Deslocamento'
    case 'flySpeed':
      return 'Voo'
    case 'armorPenalty':
      return 'Penalidade de armadura'
    case 'armorPenaltyExpertises':
      return 'Penalidade em perícias afetadas'
    case 'tempHp':
      return 'PV temp.'
    case 'tempMp':
      return 'PM temp.'
    case 'maxPv':
      return 'PV máximo'
    case 'maxPm':
      return 'PM máximo'
    case 'maneuver':
      return `Manobra: ${target.name}`
    case 'flag':
      return `Estado: ${target.name}`
  }
}

/**
 * Short labels for the one-line stance summary ("Atq/Dano/Fort/Von +3").
 * Falls back to the full label for anything without a common abbreviation.
 */
export function abbreviateConditionalTarget(target: Modifier['target']): string {
  if (target.k === 'attack' && target.scope === 'all') return 'Atq'
  if (target.k === 'damage' && target.scope === 'all') return 'Dano'
  if (target.k === 'defense') return 'Def'
  if (target.k === 'expertise') {
    if (target.name === 'Fortitude') return 'Fort'
    if (target.name === 'Vontade') return 'Von'
    if (target.name === 'Reflexos') return 'Ref'
  }
  return describeConditionalTarget(target)
}
