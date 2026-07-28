import type { AttributeKey } from '../../attributes'
import type { ExpertiseName } from '../../expertises'
import type { Modifier } from '../../items/types'
import type { ClassPower, PowerChoice, Prerequisite } from '../types'

/**
 * Builders for class powers. `autoPower` is granted automatically at the given
 * class level. `electivePower` shows up as a pickable option at `minLevel`
 * (default 1) — the player chooses one each level the class grants a free
 * "Poder de X" slot. Both share the same ClassPower shape; the engine just
 * checks `grantedAtLevel` vs the picker set to decide whether the power is
 * always-on or opt-in.
 */
export function autoPower(
  className: string,
  level: number,
  name: string,
  description: string,
  modifiers?: Modifier[],
): ClassPower {
  return {
    id: classPowerId(className, name),
    className,
    name,
    description,
    grantedAtLevel: level,
    ...(modifiers ? { modifiers } : {}),
  }
}

export function electivePower(
  className: string,
  name: string,
  description: string,
  opts: {
    minLevel?: number
    prerequisites?: Prerequisite[]
    modifiers?: Modifier[]
    choice?: PowerChoice
  } = {},
): ClassPower {
  return {
    id: classPowerId(className, name),
    className,
    name,
    description,
    ...(opts.minLevel !== undefined ? { minLevel: opts.minLevel } : {}),
    ...(opts.prerequisites ? { prerequisites: opts.prerequisites } : {}),
    ...(opts.modifiers ? { modifiers: opts.modifiers } : {}),
    ...(opts.choice ? { choice: opts.choice } : {}),
  }
}

/**
 * "Aumento de Atributo" — the +1-attribute elective every base class shares
 * (PDF Cap 2). Repeatable: taken multiple times, each pick raising a different
 * attribute (once per patamar per attribute). Shared builder so all 14 classes
 * carry the same attribute sub-choice.
 */
export const ATTRIBUTE_BOOST_CHOICE: PowerChoice = {
  kind: 'attribute',
  label: 'Atributo',
  repeatable: true,
  options: [
    { id: 'strength', name: 'Força' },
    { id: 'dexterity', name: 'Destreza' },
    { id: 'constitution', name: 'Constituição' },
    { id: 'intelligence', name: 'Inteligência' },
    { id: 'wisdom', name: 'Sabedoria' },
    { id: 'charisma', name: 'Carisma' },
  ],
}

export function attributeBoostPower(className: string): ClassPower {
  return electivePower(
    className,
    'Aumento de Atributo',
    '+1 em um atributo. Apenas uma vez por patamar para um mesmo atributo. Pode ser escolhido várias vezes.',
    { choice: ATTRIBUTE_BOOST_CHOICE },
  )
}

/** Require a specific other power by id. */
export function power(id: string): Prerequisite {
  return { kind: 'power', id }
}

/** Require at least one of the listed power ids (e.g., "any armadilha"). */
export function anyPower(ids: string[]): Prerequisite {
  return { kind: 'anyPower', ids }
}

/** Require trained in the named perícia. */
export function trained(expertise: ExpertiseName): Prerequisite {
  return { kind: 'trained', expertise }
}

/** Require attribute (raw character.X) ≥ min. */
export function attr(key: AttributeKey, min: number): Prerequisite {
  return { kind: 'attribute', attr: key, min }
}

/** Free-form gate displayed verbatim — escape hatch for non-typed gates. */
export function note(description: string): Prerequisite {
  return { kind: 'note', description }
}

/**
 * Per-class choice gate. `field` indexes into Character.classChoices[class].
 *  - `allowed`: value must be in this set.
 *  - `forbidden`: value must NOT be in this set.
 *  - neither: any non-empty value satisfies (i.e., "must be a devoto").
 *
 * `label` is the human-readable rule shown in the UI when displayed.
 */
export function classChoice(
  className: string,
  field: 'devoto' | 'caminho',
  label: string,
  opts: { allowed?: string[]; forbidden?: string[] } = {},
): Prerequisite {
  return {
    kind: 'classChoice',
    class: className,
    field,
    label,
    ...(opts.allowed ? { allowed: opts.allowed } : {}),
    ...(opts.forbidden ? { forbidden: opts.forbidden } : {}),
  }
}

/** Stable id like `class.barbaro.furia` — matches origin/race id convention. */
export function classPowerId(className: string, powerName: string): string {
  return `class.${slug(className)}.${slug(powerName)}`
}

function slug(text: string): string {
  return text
    .toLowerCase()
    .normalize('NFD')
    .replace(/[̀-ͯ]/g, '')
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')
}
