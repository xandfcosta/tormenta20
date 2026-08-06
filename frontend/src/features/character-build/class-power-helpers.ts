import {
  caminhoSlotFor,
  type ClassChoices,
  type PowerChoice,
  type Prerequisite,
  slotsForClassLevel,
  spellEffectByName,
  TORMENTA_POWERS,
} from '@tormenta20/t20-data'
import {
  allGeneralPowers,
  classPowersFor,
  devotoOptionsFor,
} from '@/shared/lib/abilities-cache'
import { catalogWeapons } from '@/shared/lib/catalog-cache'

export type ClassEntry = { className: string; level: number }

export type PowerOption = {
  id: string
  name: string
  description: string
  minLevel: number
  prerequisites: Prerequisite[]
  source: 'class' | 'general' | 'tormenta'
  /** Sub-choice this power requires when taken (totem/school/…). */
  choice?: PowerChoice
}

/**
 * The poderes da Tormenta pool as pickable options. Only offered when the
 * character's race grants Tormenta access (Lefou) — the caller gates on
 * `racesGrantTormenta`. Prereqs: a specific power (Larva ← Dentes Afiados) is
 * enforced; the "N outros poderes da Tormenta" gate is advisory (note).
 */
export function tormentaPowerOptions(): PowerOption[] {
  return Object.values(TORMENTA_POWERS).map((p): PowerOption => {
    const prerequisites: Prerequisite[] = []
    if (p.requiresPower) prerequisites.push({ kind: 'power', id: p.requiresPower })
    if (p.requiresOtherPowers > 0) {
      prerequisites.push({
        kind: 'note',
        description: `Requer ${p.requiresOtherPowers} outro(s) poder(es) da Tormenta`,
      })
    }
    return {
      id: p.id,
      name: p.name,
      description: p.description,
      minLevel: 1,
      prerequisites,
      source: 'tormenta',
    }
  })
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
    return catalogWeapons().map((w) => ({ id: w.id, name: w.name }))
  }
  return []
}

/**
 * Candidate powers a slot can be spent on for one class entry: the class's own
 * elective powers (those WITHOUT `grantedAtLevel` — auto powers are excluded)
 * plus every general power (any slot may substitute one; prereqs gate the pick).
 */
export function classPowerCandidates(
  className: string,
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
  // Any class-power slot can be spent on any general power (PDF p33); the
  // pool is level-independent — prereqs/level gate each pick at selection.
  const generalPowers = allGeneralPowers().map(
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
    // A selected repeatable with no sub-choice yet still occupies its slot
    // (min 1) — otherwise it reads as picked while consuming nothing.
    if (opt?.choice?.repeatable) n += Math.max(1, powerChoices[id]?.length ?? 0)
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

// ─── Resumo (choices, not pools) ─────────────────────────────────────

export type ChosenPowerLine = {
  id: string
  name: string
  description: string
  source: PowerOption['source']
  /** Resolved sub-choice names (totem animal, escola, arma…). */
  choices: string[]
}

/**
 * The player's CHOSEN elective powers as display lines — names resolved across
 * every pool a pick can come from (class electives of each class, generals,
 * tormenta), with any sub-choice ids mapped to their option names. Unknown ids
 * degrade to the raw id instead of vanishing. Feeds the Resumo step.
 */
export function chosenPowerLines(
  classes: ClassEntry[],
  chosenIds: string[],
  powerChoices: Record<string, string[]>,
): ChosenPowerLine[] {
  const byId = new Map<string, PowerOption>()
  for (const c of classes) {
    const { classPowers, generalPowers } = classPowerCandidates(c.className)
    for (const o of [...classPowers, ...generalPowers]) byId.set(o.id, o)
  }
  for (const o of tormentaPowerOptions()) byId.set(o.id, o)
  return chosenIds.map((id) => {
    const o = byId.get(id)
    if (!o) return { id, name: id, description: '', source: 'general', choices: [] }
    const picked = powerChoices[id] ?? []
    const options = o.choice ? powerChoiceOptions(o.choice) : []
    const choices = picked.map(
      (cid) => options.find((opt) => opt.id === cid)?.name ?? cid,
    )
    return { id, name: o.name, description: o.description, source: o.source, choices }
  })
}

/**
 * One class's caminho/devoto picks as a display line ("caminho: Mago · devoto
 * de Khalmyr") — ids resolved to names via the same catalogs the pickers use.
 * Null when the class made no such pick.
 */
export function classChoiceSummary(
  className: string,
  blob: { caminho?: string; devoto?: string } | undefined,
): string | null {
  const parts: string[] = []
  if (blob?.caminho) {
    const name =
      caminhoSlotFor(className)?.options.find((o) => o.id === blob.caminho)
        ?.name ?? blob.caminho
    parts.push(`caminho: ${name}`)
  }
  if (blob?.devoto) {
    const name =
      devotoOptionsFor(className)?.find((d) => d.id === blob.devoto)?.name ??
      blob.devoto
    parts.push(`devoto de ${name}`)
  }
  return parts.length > 0 ? parts.join(' · ') : null
}

export type PowerPickOption = { value: string; label: string; description: string }

/**
 * Pickable pool for a free-pick origem benefit ("um poder de combate/da
 * Tormenta a sua escolha"). Prereqs are advisory here — the benefit text says
 * they apply; the GM arbitrates at the table.
 */
export function powerPickOptions(
  pool: 'combate' | 'tormenta',
): PowerPickOption[] {
  if (pool === 'tormenta') {
    return Object.values(TORMENTA_POWERS).map((p) => ({
      value: p.id,
      label: p.name,
      description: p.description,
    }))
  }
  return allGeneralPowers()
    .filter((p) => p.kind === 'combate')
    .map((p) => ({ value: p.id, label: p.name, description: p.description }))
}
