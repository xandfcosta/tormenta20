import type { AttributeKey } from './attributes'

/**
 * Magia — PDF book Cap 4 (p168-211). This module pins the **mechanics**
 * of casting: PM cost per circle, save CD formula, execution / range /
 * duration / area taxonomy, components, schools, save outcomes, and
 * augment validation. The spell catalog (individual magias) is a
 * separate, larger artifact and is intentionally not encoded here.
 */

/**
 * Círculos de magia. Truques are círculo 0 — they cost 0 PM and ignore
 * the augment system. Círculos 1-5 follow the cost table on p171.
 */
export type SpellCircle = 0 | 1 | 2 | 3 | 4 | 5

/**
 * PDF p171 — Custo em PM por círculo. Truques are free. The cost is
 * what's paid for the *base* effect; +PM aprimoramentos add on top.
 */
export const SPELL_BASE_PM_COST: Readonly<Record<SpellCircle, number>> = {
  0: 0,
  1: 1,
  2: 3,
  3: 6,
  4: 10,
  5: 15,
}

/**
 * The 8 escolas de magia (PDF p170). Mechanically interchangeable for
 * resolution; modifier targets and class restrictions reference them
 * (e.g., Mago de Evocação dá +1 dano por dado em magias de Evocação).
 */
export const SPELL_SCHOOLS = [
  'abjuracao',
  'adivinhacao',
  'convocacao',
  'encantamento',
  'evocacao',
  'ilusao',
  'necromancia',
  'transmutacao',
] as const
export type SpellSchool = (typeof SPELL_SCHOOLS)[number]

/**
 * Execução — how a spell fits into the action economy. PDF p172.
 * Most magias are padrão; some are livre (Escudo Arcano) or reação.
 * Sustentar a magia is itself livre but costs 1 PM/turn (see DURATION).
 */
export type SpellExecution =
  | 'padrao'
  | 'movimento'
  | 'livre'
  | 'reacao'
  | 'completa'

/**
 * Alcance — PDF p172. Distances measured to nearest square edge.
 *   pessoal  — caster only
 *   toque    — adjacente, requires hand contact
 *   curto    — 9m (6 squares)
 *   medio    — 30m (20 squares)
 *   longo    — 90m (60 squares)
 *   ilimitado — same plane
 */
export type SpellRange =
  | 'pessoal'
  | 'toque'
  | 'curto'
  | 'medio'
  | 'longo'
  | 'ilimitado'

/** Range → meters. `ilimitado` returns Infinity. `pessoal`/`toque` = 0. */
export const SPELL_RANGE_METERS: Readonly<Record<SpellRange, number>> = {
  pessoal: 0,
  toque: 0,
  curto: 9,
  medio: 30,
  longo: 90,
  ilimitado: Infinity,
}

/**
 * Duração — PDF p172-173.
 *   instantanea — effect resolves immediately, nothing persists
 *   cena        — until end of current cena/encontro
 *   sustentada  — caster spends 1 PM at start of each of their turns;
 *                 only one sustained magia active at a time
 *   definida    — explicit per-spell duration (e.g., 1 min, 5 rounds)
 *   dia         — 1 day
 *   permanente  — until dissipated
 */
export type SpellDuration =
  | 'instantanea'
  | 'cena'
  | 'sustentada'
  | 'definida'
  | 'dia'
  | 'permanente'

/** PM upkeep cost per turn for sustained magias. PDF p173. */
export const SUSTAIN_PM_PER_TURN = 1

/**
 * Componentes (gestos + palavras). PDF p173: "Toda magia tem componentes
 * verbais e gestuais." A character that is gagged/silenced or whose
 * hands are all occupied/bound cannot cast magia. Some magias add
 * material/focus components; those are pinned per-spell, not here.
 */
export type SpellComponent = 'verbal' | 'gestual' | 'material' | 'foco'

export const DEFAULT_SPELL_COMPONENTS: readonly SpellComponent[] = [
  'verbal',
  'gestual',
]

/**
 * Áreas de efeito — PDF p173. Standard shape catalog used by area
 * magias. Per-spell sizes override the defaults below.
 */
export type AreaShape = 'cilindro' | 'cone' | 'esfera' | 'linha' | 'quadrado' | 'cubo'

/**
 * Tipo de resistência a magia — PDF p173.
 *   anula        — successful save negates the entire effect
 *   parcial      — save reduces but doesn't negate
 *   metade       — save halves numeric effect (typical for dano)
 *   desacredita  — alvo só recebe novo teste se interagir/ser convencido
 */
export type SpellResistance = 'anula' | 'parcial' | 'metade' | 'desacredita'

/** Save type targeted by the magia. */
export type SpellSaveType = 'fortitude' | 'reflexos' | 'vontade' | 'none'

/**
 * Atributo-chave de conjuração por classe (PDF p168, class entries).
 * Used to compute the spell save CD. Multi-class casters pick per
 * spell list at cast time.
 */
export const CLASS_SPELLCASTING_ATTRIBUTE: Readonly<Record<string, AttributeKey>> = {
  Arcanista: 'intelligence',
  Bardo: 'charisma',
  Clérigo: 'wisdom',
  Druida: 'wisdom',
  Paladino: 'wisdom',
}

/**
 * Spell save CD per PDF p171:
 *   CD = 10 + ½ nível do conjurador + modificador do atributo-chave
 * The "½ nível" is rounded down per T20 convention.
 */
export function spellSaveDc(
  casterLevel: number,
  keyAttributeMod: number,
): number {
  if (casterLevel < 1) {
    throw new Error(
      `spellSaveDc: casterLevel must be ≥ 1, got ${casterLevel}`,
    )
  }
  return 10 + Math.floor(casterLevel / 2) + keyAttributeMod
}

/**
 * Total PM cost = base circle cost + sum of additional augment PM
 * spent. Truques (círculo 0) always cost 0 and reject augments — PDF
 * p171: "Truques não podem receber aprimoramentos."
 */
export function spellPmCost(circle: SpellCircle, augmentPm: number = 0): number {
  if (augmentPm < 0) {
    throw new Error(
      `spellPmCost: augmentPm must be ≥ 0, got ${augmentPm}`,
    )
  }
  if (circle === 0 && augmentPm > 0) {
    throw new Error(
      `spellPmCost: truques cannot receive augments (augmentPm=${augmentPm})`,
    )
  }
  return SPELL_BASE_PM_COST[circle] + augmentPm
}

/**
 * PDF p173: Concentração para manter uma magia sustentada quando o
 * conjurador sofre dano. Vontade test against CD that scales with the
 * type of distraction.
 *   normal     — CD 15
 *   ruim       — CD 15 + cost of the magia
 *   terrivel   — CD 20 + cost of the magia
 */
export type ConcentrationDifficulty = 'normal' | 'ruim' | 'terrivel'

export function concentrationCd(
  difficulty: ConcentrationDifficulty,
  spellTotalPm: number,
): number {
  switch (difficulty) {
    case 'normal':
      return 15
    case 'ruim':
      return 15 + spellTotalPm
    case 'terrivel':
      return 20 + spellTotalPm
  }
}

/**
 * Aprimoramento (augment) kind. PDF p171-172:
 *   aumenta — same effect can be taken multiple times, cumulative
 *   muda    — alters the spell's behaviour; cannot stack with itself
 */
export type AugmentKind = 'aumenta' | 'muda'

export type SpellAugment = {
  id: string
  kind: AugmentKind
  /** Extra PM the augment costs on top of the base circle cost. */
  pmCost: number
  description: string
}

/**
 * Validate a selected set of augments per PDF rules:
 *  - Truques cannot take any augment.
 *  - 'muda' augments cannot be taken twice (same id) — at most 1 stack.
 *  - 'aumenta' augments may stack (any count ≥ 1).
 *
 * Returns total +PM spent on augments. Throws on invalid combination.
 */
export function totalAugmentPm(
  circle: SpellCircle,
  augments: readonly { augment: SpellAugment; stacks: number }[],
): number {
  if (circle === 0 && augments.length > 0) {
    throw new Error('totalAugmentPm: truques cannot receive aprimoramentos')
  }
  let total = 0
  for (const { augment, stacks } of augments) {
    if (stacks < 1) {
      throw new Error(
        `totalAugmentPm: stacks must be ≥ 1 for ${augment.id}, got ${stacks}`,
      )
    }
    if (augment.kind === 'muda' && stacks > 1) {
      throw new Error(
        `totalAugmentPm: 'muda' augment ${augment.id} cannot stack (got ${stacks})`,
      )
    }
    total += augment.pmCost * stacks
  }
  return total
}
