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
// O único modificador de condição que não é penalidade em teste: o Alquebrado
// encarece as habilidades do personagem.
const pmCost = (n: number) => mod({ k: 'pmCost' }, n)
// O Petrificado concede RD 8 (p394) — por modificador, como qualquer fonte.
const damageReduction = (n: number) => mod({ k: 'damageReduction' }, n)
// A Defesa DIRECIONAL do Caído (p394) — escopo separado para não competir com a
// Defesa geral das outras condições, que é o que o "cumulativos" do livro pede.
const defenseVs = (scope: 'melee' | 'ranged', n: number) =>
  mod({ k: 'defense', scope }, n)

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

/**
 * As condições que a p394 usa para DEFINIR outras. Compor em vez de copiar os
 * números é o que impede uma derivada de ficar com metade do efeito — foi assim
 * que o cego perdeu a penalidade de Reflexos (ALE-112) e que cinco condições
 * ficaram sem efeito nenhum (ALE-115).
 */
const VULNERAVEL = [defense(-2)]
const DESPREVENIDO = [defense(-5), skill('Reflexos', -5)]
const FRACO = forDesCon(-2)
const DEBILITADO = forDesCon(-5)
/**
 * "INDEFESO. O personagem fica desprevenido, MAS sofre −10 na Defesa, falha
 * automaticamente em testes de Reflexos […]" — o "mas" faz o −10 SUBSTITUIR o −5
 * do desprevenido. A falha automática em Reflexos não é um número e não vira
 * modificador (ALE-115).
 */
const INDEFESO = [defense(-10)]

export const CONDITION_MODIFIERS: Partial<Record<ConditionId, Modifier[]>> = {
  // Medo — penalidade global em perícias (inclui as resistências, que o motor
  // trata como perícias — casa com a expectativa do relator do ALE-28).
  abalado: [allSkills(-2)],
  apavorado: [allSkills(-5)],

  // Defesa (e Reflexos, uma perícia dex).
  vulneravel: VULNERAVEL,
  desprevenido: DESPREVENIDO,
  indefeso: INDEFESO,

  // Penalidade em testes de atributo + perícias baseadas.
  fraco: FRACO,
  debilitado: DEBILITADO,
  frustrado: intSabCar(-2),
  esmorecido: intSabCar(-5),

  // As derivadas — cada uma cita o texto da p394 que a obriga. O "lento" e o
  // "imóvel" ficam lembrete: deslocamento não é número de ficha.
  // "ATORDOADO. O personagem fica desprevenido e não pode fazer ações."
  atordoado: DESPREVENIDO,
  // "SURPREENDIDO. O personagem fica desprevenido e não pode fazer ações."
  surpreendido: DESPREVENIDO,
  // "PARALISADO. Fica imóvel e indefeso […]"
  paralisado: INDEFESO,
  // "INCONSCIENTE. O personagem fica indefeso e não pode fazer ações […]"
  inconsciente: INDEFESO,
  // "PETRIFICADO. O personagem fica inconsciente e recebe redução de dano 8."
  petrificado: [...INDEFESO, damageReduction(8)],
  // "FATIGADO. O personagem fica fraco e vulnerável."
  fatigado: [...FRACO, ...VULNERAVEL],
  // "EXAUSTO. O personagem fica debilitado, lento e vulnerável."
  exausto: [...DEBILITADO, ...VULNERAVEL],
  // "CEGO. O personagem fica desprevenido e lento […] e sofre −5 em testes de
  // perícias baseadas em Força ou Destreza." As DUAS metades do desprevenido,
  // não só a Defesa (ALE-112).
  cego: [...DESPREVENIDO, byAttr('strength', -5), byAttr('dexterity', -5)],
  // "AGARRADO. O personagem fica desprevenido e imóvel, sofre −2 em ataque."
  agarrado: [...DESPREVENIDO, attack(-2)],
  // "ENREDADO. O personagem fica lento, vulnerável e sofre −2 em ataque."
  enredado: [...VULNERAVEL, attack(-2)],

  // "ALQUEBRADO. O custo em pontos de mana das habilidades do personagem
  // aumenta em +1." Aumento, não redução — soma normalmente (p226).
  alquebrado: [pmCost(1)],

  // Perícia específica / ataque.
  ofuscado: [attack(-2), skill('Percepção', -2)],
  fascinado: [skill('Percepção', -5)],
  surdo: [skill('Iniciativa', -5)],
  // "CAÍDO. O personagem sofre −5 na Defesa contra ataques corpo a corpo e
  // recebe +5 na Defesa contra ataques à distância (cumulativos com outras
  // condições). Além disso, sofre −5 em ataques corpo a corpo" — Luta no motor.
  caido: [skill('Luta', -5), defenseVs('melee', -5), defenseVs('ranged', 5)],
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
