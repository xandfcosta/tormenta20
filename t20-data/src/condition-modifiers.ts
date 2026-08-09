import type { AttributeKey } from './attributes'
import type { ConditionId } from './conditions'
import type { ExpertiseName } from './expertises'
import type { Modifier, ModifierTarget } from './items/types'

/**
 * Mechanical modifiers each status condition (p394) applies to the derived
 * sheet. Encoded as v2 `Modifier`s so they flow through the same engine the
 * sheet already runs (collect → resolveStack), reaching Defesa, perícias,
 * resistências (que SÃO perícias no motor — Fortitude/Reflexos/Vontade) e
 * ataque. ALE-28.
 *
 * All condition mods use `bonusType: 'condition'`: book p394 says "condições com
 * os mesmos efeitos não se acumulam; aplique apenas o mais severo" — the typed
 * bucket in resolveStack keeps the highest-abs per target, so two conditions
 * hitting Defesa give the worst of the two, not their sum, while still stacking
 * with item/other bonuses.
 *
 * Only conditions with an unambiguous, sheet-global numeric effect are here.
 * Situational effects (deslocamento pela metade, economia de ações, Defesa
 * direcional do Caído, "50% acerta alvo errado"…) stay display-only reminders —
 * `conditionModifiers` returns `[]` for them.
 *
 * Known limitation: a skill hit by two conditions through *different* targets
 * (e.g. Abalado's `expertiseAll` and Fraco's `expertiseByAttribute`) sums both,
 * since "mais severo" only dedups within one target key. Rare in play.
 */

const CONDITION_BONUS = 'condition' as const

function mod(target: ModifierTarget, amount: number): Modifier {
  return { target, amount, bonusType: CONDITION_BONUS }
}

const defense = (n: number) => mod({ k: 'defense' }, n)
const allSkills = (n: number) => mod({ k: 'expertiseAll' }, n)
const skill = (name: ExpertiseName, n: number) => mod({ k: 'expertise', name }, n)
const byAttr = (attribute: AttributeKey, n: number) =>
  mod({ k: 'expertiseByAttribute', attribute }, n)
const attack = (n: number) => mod({ k: 'attack', scope: 'all' }, n)

// "-N em testes de Força/Destreza/Constituição e perícias baseadas" (Fraco…).
const forDesCon = (n: number) => [
  byAttr('strength', n),
  byAttr('dexterity', n),
  byAttr('constitution', n),
]
// "-N em Inteligência/Sabedoria/Carisma e perícias baseadas" (Frustrado…).
const intSabCar = (n: number) => [
  byAttr('intelligence', n),
  byAttr('wisdom', n),
  byAttr('charisma', n),
]

export const CONDITION_MODIFIERS: Partial<Record<ConditionId, Modifier[]>> = {
  // Medo — penalidade global em perícias (inclui as resistências, que o motor
  // trata como perícias — casa com a expectativa do relator do ALE-28).
  abalado: [allSkills(-2)],
  apavorado: [allSkills(-5)],

  // Defesa (e Reflexos, uma perícia dex).
  vulneravel: [defense(-2)],
  desprevenido: [defense(-5), skill('Reflexos', -5)],
  indefeso: [defense(-10)],

  // Penalidade em testes de atributo + perícias baseadas.
  fraco: forDesCon(-2),
  debilitado: forDesCon(-5),
  frustrado: intSabCar(-2),
  esmorecido: intSabCar(-5),
  // Compostos: Fatigado = Fraco + Vulnerável; Exausto = Debilitado + Vulnerável
  // (o "lento" fica lembrete).
  fatigado: [...forDesCon(-2), defense(-2)],
  exausto: [...forDesCon(-5), defense(-2)],
  // Cego: Desprevenido (Defesa −5, e Reflexos via Destreza) + −5 perícias de
  // Força/Destreza (o "lento" e "camuflagem total" ficam lembrete).
  cego: [defense(-5), byAttr('strength', -5), byAttr('dexterity', -5)],

  // Perícia específica / ataque.
  ofuscado: [attack(-2), skill('Percepção', -2)],
  fascinado: [skill('Percepção', -5)],
  surdo: [skill('Iniciativa', -5)],
  enredado: [defense(-2), attack(-2)], // Vulnerável + −2 ataques (lento = lembrete)
  agarrado: [defense(-5), skill('Reflexos', -5), attack(-2)], // Desprevenido + −2 ataques
  // Caído: −5 ataques corpo-a-corpo = −5 Luta (a Defesa direcional não é
  // modelável num único número).
  caido: [skill('Luta', -5)],
}

/** Modifiers a condition applies to the sheet; `[]` for display-only conditions. */
export function conditionModifiers(id: ConditionId): Modifier[] {
  return CONDITION_MODIFIERS[id] ?? []
}

const ATTR_ABBR: Record<AttributeKey, string> = {
  strength: 'For',
  dexterity: 'Des',
  constitution: 'Con',
  intelligence: 'Int',
  wisdom: 'Sab',
  charisma: 'Car',
}

function targetLabel(t: ModifierTarget): string {
  switch (t.k) {
    case 'defense':
      return 'Def'
    case 'expertiseAll':
      return 'perícias'
    case 'expertise':
      return t.name
    case 'attack':
      return 'ataque'
    default:
      return t.k
  }
}

const signed = (n: number) => (n >= 0 ? `+${n}` : `−${-n}`)

/**
 * Short human label of a condition's mechanical effect for the sheet chip
 * ("−2 Def · −5 Reflexos"), or "lembrete" when it changes no sheet number.
 * Groups the `expertiseByAttribute` mods by amount ("−2 For/Des/Con").
 */
export function conditionEffectSummary(id: ConditionId): string {
  const mods = CONDITION_MODIFIERS[id]
  if (!mods || mods.length === 0) return 'lembrete'
  const parts: string[] = []
  const byAttrByAmount = new Map<number, string[]>()
  for (const m of mods) {
    if (m.target.k === 'expertiseByAttribute') {
      const group = byAttrByAmount.get(m.amount) ?? []
      group.push(ATTR_ABBR[m.target.attribute])
      byAttrByAmount.set(m.amount, group)
      continue
    }
    parts.push(`${signed(m.amount)} ${targetLabel(m.target)}`)
  }
  for (const [amount, attrs] of byAttrByAmount) {
    parts.push(`${signed(amount)} ${attrs.join('/')}`)
  }
  return parts.join(' · ')
}
