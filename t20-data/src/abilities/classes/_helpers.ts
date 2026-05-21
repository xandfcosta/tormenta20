import type { AttributeKey } from '../../attributes'
import type { ExpertiseName } from '../../expertises'
import type { Modifier } from '../../items/types'
import type { ClassPower, Prerequisite } from '../types'

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
  }
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

/** Free-form gate displayed verbatim — used for devotion, caminho, etc. */
export function note(description: string): Prerequisite {
  return { kind: 'note', description }
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
