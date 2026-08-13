import { ATTRIBUTE_ABBR } from '@/entities/character/expertise'
import type {
  ComputedSheetV2,
  ExpertiseBreakdown,
  TotalContribs,
  WeaponCard,
} from '@/shared/lib/computed-sheet-v2'

/** One line of a breakdown dialog: where the number came from, and how much. */
export type StatRow = { label: string; amount: number; muted?: boolean; note?: string }

/** Contribution lists all carry the same three fields — one mapper for all. */
function contributionRows(contributions: TotalContribs['contributions']): StatRow[] {
  return contributions.map((c) => ({ label: c.source, amount: c.amount, note: c.note }))
}

/**
 * Defense, line by line. Heavy armor BLOCKS Destreza — that shows as a muted
 * zero row instead of silently missing, because "why is my Defesa low?" is the
 * question the dialog exists to answer.
 *
 * @example defenseRows(sheet)[1].label // "Destreza (bloqueada por armadura pesada)"
 */
export function defenseRows(sheet: ComputedSheetV2): StatRow[] {
  const dexterity = sheet.defense.dexApplied
    ? { label: 'Destreza', amount: sheet.attributes.dexterity.total }
    : { label: 'Destreza (bloqueada por armadura pesada)', amount: 0, muted: true }
  return [
    { label: 'Base', amount: 10 },
    dexterity,
    ...contributionRows(sheet.defense.contributions),
    ...directionalDefenseRows(sheet),
  ]
}

/**
 * As duas Defesas DIRECIONAIS, mostradas só quando divergem da geral — hoje
 * apenas o Caído as separa (p394: −5 contra corpo a corpo, +5 contra à
 * distância). Um personagem em pé vê a mesma linha de sempre; um caído vê os
 * dois números que de fato valem contra cada tipo de ataque.
 */
function directionalDefenseRows(sheet: ComputedSheetV2): StatRow[] {
  const { total, vsMelee, vsRanged } = sheet.defense
  if (vsMelee === total && vsRanged === total) return []
  return [
    { label: 'Contra corpo a corpo', amount: vsMelee - total },
    { label: 'Contra ataques à distância', amount: vsRanged - total },
  ]
}

/**
 * An attack or a resistance, line by line: ½ nível, the attribute, training,
 * then the item contributions and (for attacks) the global attack modifiers.
 *
 * `attackAll` is scope:'all' only — Fúria and friends. Weapon-specific
 * (scope:'this') mods stay out: the non-proficiency penalty already comes
 * through the expertise path, and folding it again would double-count it.
 */
export function expertiseRows(
  expertise: ExpertiseBreakdown,
  attrAbbr: string,
  attackAll?: TotalContribs,
): StatRow[] {
  return [
    { label: '½ nível', amount: expertise.halfLevel },
    { label: attrAbbr, amount: expertise.attrValue },
    ...(expertise.training ? [{ label: 'Treino', amount: expertise.training }] : []),
    ...contributionRows(expertise.itemContributions),
    ...(attackAll ? contributionRows(attackAll.contributions) : []),
  ]
}

/** The three saves, in the order a table calls for them. */
export const SAVES = [
  { name: 'Fortitude', attribute: 'constitution', abbr: 'CON' },
  { name: 'Reflexos', attribute: 'dexterity', abbr: 'DES' },
  { name: 'Vontade', attribute: 'wisdom', abbr: 'SAB' },
] as const

/**
 * Attack + damage rows for a weapon card. The Go engine owns the NUMBERS
 * (`WeaponCard`); this only applies the structural labels. The FOR damage row
 * shows for melee/thrown (Luta) and never for ranged, where `strDamage` is 0.
 *
 * @example weaponCardRows(machadoCard).damageRows[0] // { label: 'FOR', amount: 4 }
 */
export function weaponCardRows(card: WeaponCard): {
  attackRows: StatRow[]
  damageRows: StatRow[]
} {
  return {
    attackRows: expertiseRows(
      card.expertise,
      ATTRIBUTE_ABBR[card.attribute],
      card.attackAll,
    ),
    damageRows: [
      ...(card.skill === 'Luta' ? [{ label: 'FOR', amount: card.strDamage }] : []),
      ...contributionRows(card.damageAll.contributions),
    ],
  }
}

/** "19-20/x3" — a crit range of 20 is written plain, not as "20-20". */
export function critLabel(card: WeaponCard): string {
  const range = card.critRange < 20 ? `${card.critRange}-20` : '20'
  return `${range}/x${card.critMult}`
}

/** PM ceiling per spell: caster level plus whatever raises it. */
export function pmLimitRows(sheet: ComputedSheetV2): StatRow[] {
  return [
    { label: 'Nível de conjurador', amount: sheet.pmLimit.base },
    ...contributionRows(sheet.pmLimit.contributions),
  ]
}

/**
 * Spell DC rows. The base comes from the ENGINE (level + key attribute), so the
 * key attribute is the FINAL value — reading the raw attribute understated
 * Osteon casters by 1.
 */
export function spellDcRows(sheet: ComputedSheetV2): StatRow[] {
  const bonus = sheet.spellDCBonus
  return [
    { label: 'CD base (nível + atributo-chave)', amount: sheet.bestBaseSpellCd ?? 0 },
    ...(bonus.total !== 0
      ? contributionRows(bonus.contributions)
      : [{ label: 'Sem bônus de itens', amount: 0, muted: true }]),
  ]
}

export function pmCostRows(sheet: ComputedSheetV2): StatRow[] {
  const cost = sheet.pmCostMod
  return cost.total !== 0
    ? contributionRows(cost.contributions)
    : [{ label: 'Sem mod de itens', amount: 0, muted: true }]
}
