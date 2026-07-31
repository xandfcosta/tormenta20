import {
  caminhoSlotFor,
  classPowersFor,
  devotoOptionsFor,
  generalPowersByKinds,
  type ClassChoices,
  type PowerChoice,
  type Prerequisite,
  slotsForClassLevel,
  spellEffectByName,
  unlockedKinds,
  WEAPONS,
} from '@tormenta20/t20-data'

export type ClassEntry = { className: string; level: number }

export type PowerOption = {
  id: string
  name: string
  description: string
  minLevel: number
  prerequisites: Prerequisite[]
  source: 'class' | 'general'
  /** Sub-choice this power requires when taken (totem/school/…). */
  choice?: PowerChoice
}

export type ChoiceOption = {
  id: string
  name: string
  note?: string
  /** What the option does — resolved from the catalog (e.g. a totem's granted
   *  spell effect). Shown when the option is selected. */
  desc?: string
}

/**
 * Resolve the selectable options for a power's sub-choice. Enumerated kinds
 * (totem/school/companion) carry their own list; `weapon` is sourced from the
 * item catalog. When an option's `note` names a spell (Bárbaro totems grant
 * one), its effect is attached as `desc` so the picker can explain the pick.
 */
export function powerChoiceOptions(choice: PowerChoice): ChoiceOption[] {
  if (choice.options) {
    return choice.options.map((o) => ({
      ...o,
      desc: (o.note && spellEffectByName(o.note)) || undefined,
    }))
  }
  if (choice.kind === 'weapon') {
    return WEAPONS.map((w) => ({ id: w.id, name: w.name }))
  }
  return []
}

/**
 * Candidate powers a slot can be spent on for one class entry: the class's own
 * elective powers (those WITHOUT `grantedAtLevel` — auto powers are excluded)
 * plus the general powers whose kind is unlocked at this level.
 */
export function classPowerCandidates(
  className: string,
  level: number,
): { classPowers: PowerOption[]; generalPowers: PowerOption[] } {
  const classPowers = classPowersFor(className)
    .filter((p) => p.grantedAtLevel === undefined)
    .map(
      (p): PowerOption => ({
        id: p.id,
        name: p.name,
        description: p.description,
        minLevel: p.minLevel ?? 1,
        prerequisites: p.prerequisites ?? [],
        source: 'class',
        choice: p.choice,
      }),
    )
  const generalPowers = generalPowersByKinds(unlockedKinds(className, level)).map(
    (p): PowerOption => ({
      id: p.id,
      name: p.name,
      description: p.description,
      minLevel: p.minLevel ?? 1,
      prerequisites: p.prerequisites ?? [],
      source: 'general',
    }),
  )
  return { classPowers, generalPowers }
}

/** Elective power slots earned by a class at a level (1 per level, L2..L20). */
export function classSlotCount(className: string, level: number): number {
  return slotsForClassLevel(className, level).length
}

/** Total elective slots across every class entry. */
export function totalSlots(classes: ClassEntry[]): number {
  return classes.reduce((n, c) => n + classSlotCount(c.className, c.level), 0)
}

/**
 * Slots actually consumed by the current picks. A repeatable power (Aumento de
 * Atributo, Especialização em Arma/Escola) can be taken once per sub-choice —
 * each pick eats a slot — so it counts by the number of choices made; every
 * other picked power counts as one.
 */
export function usedSlots(
  chosenIds: string[],
  powerChoices: Record<string, string[]>,
  candidatesById: Map<string, PowerOption>,
): number {
  let n = 0
  for (const id of chosenIds) {
    const opt = candidatesById.get(id)
    if (opt?.choice?.repeatable) n += powerChoices[id]?.length ?? 0
    else n += 1
  }
  return n
}

/** True while fewer power picks have been made than slots earned. */
export function anyClassElectivePending(
  classes: ClassEntry[],
  chosenIds: string[],
): boolean {
  return chosenIds.length < totalSlots(classes)
}

/**
 * A class still owes a caminho and/or devoto choice (Arcanista caminho at L1,
 * Cavaleiro/Paladino caminho at L5, Clérigo/Druida/Paladino devoto).
 */
export function anyClassChoicePending(
  classes: ClassEntry[],
  classChoices: ClassChoices,
): boolean {
  return classes.some((c) => {
    const cam = caminhoSlotFor(c.className)
    if (cam && c.level >= cam.minLevel && !classChoices[c.className]?.caminho) {
      return true
    }
    return (
      devotoOptionsFor(c.className) !== null && !classChoices[c.className]?.devoto
    )
  })
}

export type PrereqStatus = { met: boolean; enforced: boolean; reason: string }

/**
 * Wizard-light prerequisite check. Enforces the prereqs whose inputs are known
 * at the Poderes step (other powers picked, caminho/devoto, level). Attribute
 * and trained-perícia prereqs come from LATER steps, so they're reported as
 * info (met, with a reason to show) rather than blocking the pick.
 */
export function evalPowerPrereq(
  prereq: Prerequisite,
  ctx: { chosenIds: Set<string>; classChoices: ClassChoices },
): PrereqStatus {
  switch (prereq.kind) {
    case 'power':
      return {
        enforced: true,
        met: ctx.chosenIds.has(prereq.id),
        reason: 'Requer outro poder',
      }
    case 'anyPower':
      return {
        enforced: true,
        met: prereq.ids.some((id) => ctx.chosenIds.has(id)),
        reason: 'Requer um poder pré-requisito',
      }
    case 'classChoice': {
      const value = ctx.classChoices[prereq.class]?.[prereq.field]
      const ok = value
        ? (prereq.allowed?.includes(value) ?? true) &&
          !prereq.forbidden?.includes(value)
        : false
      return { enforced: true, met: ok, reason: prereq.label }
    }
    case 'attribute':
      return { enforced: false, met: true, reason: `Requer ${prereq.attr} ${prereq.min}` }
    case 'trained':
      return { enforced: false, met: true, reason: `Requer treino em ${prereq.expertise}` }
    case 'note':
      return { enforced: false, met: true, reason: prereq.description }
  }
}

/** A power is pickable given level + enforced prereqs (slot availability is
 *  checked by the caller against remaining count). */
export function powerBlockedReason(
  option: PowerOption,
  level: number,
  ctx: { chosenIds: Set<string>; classChoices: ClassChoices },
): string | null {
  if (level < option.minLevel) return `≥ Nv ${option.minLevel}`
  for (const p of option.prerequisites) {
    const s = evalPowerPrereq(p, ctx)
    if (s.enforced && !s.met) return s.reason
  }
  return null
}
