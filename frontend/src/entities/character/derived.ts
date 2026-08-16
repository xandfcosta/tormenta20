import { EXPERTISE_NAMES } from '@/shared/api/expertise-names'
import { originModifiers } from '@/shared/rules/abilities-origin-logic'
import { raceModifiers } from '@/shared/rules/abilities-race-logic'
import { conditionModifiers } from '@/shared/rules/condition-modifiers'
import { DEFORMIDADE_PERICIA_BONUS } from '@/shared/rules/deformidade'
import { trainingBonusForLevel } from '@/shared/rules/expertises'
import { conditionalId, resolveStack, statFor } from '@/shared/rules/items-engine'
import { HOMEBREW_VESTED_OK } from '@/shared/rules/items-homebrew'
import { resolveAtributoMod } from '@/shared/rules/racas-attr'
import { carismaLossFromPowers } from '@/shared/rules/tormenta-carisma'
import { requiredProficiency } from '@/shared/api/item-classify'
import type { ClassChoices, ConditionId, Prerequisite } from '@/shared/api/catalog-types'
import type { ExpertiseName } from '@/shared/api/expertise-names'
import type { CatalogItem, ItemEffects } from '@/shared/api/item-types'
import type { ActiveItem, ConditionalEffect, Modifier } from '@/shared/api/item-types'
import { ATTRIBUTE_ABBR, type AttributeKey } from '@/shared/api/attribute-keys'
// The abilities lookups + item catalog now read the fetched catalogs (primed by
// the root loader) instead of the build-time t20-data data — keeps the ~149KB
// abilities chunk + ~44KB item data out of the bundle. `raceModifiers` /
// `originModifiers` stay imported above: they're pure, data-free logic that
// tree-shakes on its own. See project_front_decouple_catalog B.2/B.3.
import { getCatalogItem } from '@/shared/lib/catalog-cache'
import {
  getClassPower,
  getGeneralPower,
  getOrigin,
  getOriginBenefit,
  getRace,
  ownedClassPowers,
  raceWithDeformidade,
} from '@/shared/lib/abilities-cache'
// RACAS + TORMENTA_POWERS now read the fetched catalogs (primed by the loader);
// `resolveAtributoMod` stays a barrel import — data-free after the racas-attr
// split, so it tree-shakes. See project_front_decouple_catalog.
import { racasList } from '@/shared/lib/racas-cache'
import { tormentaPowersRecord } from '@/shared/lib/rules-catalog-cache'
import type { Character, CharacterExpertise, CharacterItem } from '@/shared/api/api'
import {
  areEngineCatalogsPrimed,
  computeEffects as engineComputeEffects,
} from '@/shared/lib/engine-wasm'
import { effectSourceName } from './effect-source'

/**
 * Collect every active modifier source into `ActiveItem[]` — the input the
 * resolution engine (`computeItemEffects`) consumes. Exported so the parity
 * harness can dump it as the Go engine's oracle input (PORT-PLAN.md §3, the
 * catalog-reading collection layer ported in slice 2).
 */
export function activeItemsFor(character: Character): ActiveItem[] {
  const proficiencies = parseProficiencySetFromCharacter(character)
  const items: ActiveItem[] = character.items
    .filter((it) => it.equipped !== null)
    .map((it) => {
      const catalog = it.catalogId ? getCatalogItem(it.catalogId) : undefined
      const baseMods: Modifier[] = catalog?.modifiers ?? []
      const improvementIds = parseImprovementIds(it.improvements)
      const improvementMods = improvementIds.flatMap((id) =>
        overlayModsWithProvenance(getCatalogItem(id)),
      )
      const materialMods = overlayModsWithProvenance(
        it.material ? getCatalogItem(it.material) : undefined,
      )
      const penaltyMods = catalog
        ? nonProficiencyPenalties(catalog, proficiencies)
        : []
      const ownMods = [...baseMods, ...improvementMods, ...materialMods]
      return {
        source: it.name,
        equipped: it.equipped,
        modifiers: [
          ...ownMods,
          ...penaltyMods,
          ...mirrorWeaponAttackMods(catalog, ownMods),
          ...equilibradaHomebrewMods(catalog, improvementIds),
          ...vestedEsotericHomebrewMods(it.equipped, catalog, ownMods),
        ],
      }
    })
  for (const eff of character.activeEffects ?? []) {
    const sourceName = effectSourceName(eff.catalogId)
    const modifiers = parseEffectModifiers(eff.modifiers)
    if (modifiers.length === 0) continue
    items.push({
      source: `${sourceName} (${eff.scope === 'day' ? 'dia' : 'cena'})`,
      equipped: 'vested',
      modifiers,
    })
  }
  const raceMods = raceActiveItems(character)
  items.push(...raceMods)
  const originMods = originActiveItem(character)
  if (originMods) items.push(originMods)
  const classMods = classActiveItems(character)
  items.push(...classMods)
  items.push(...generalPowerActiveItem(character))
  const tormentaCar = tormentaCarismaItem(character)
  if (tormentaCar) items.push(tormentaCar)
  const conditions = conditionActiveItem(character)
  if (conditions) items.push(conditions)
  return items
}

/**
 * Status conditions (p394) as a synthetic ActiveItem, so their p394 numeric
 * penalties flow through the same engine as items (ALE-28). Parsed cache-free
 * (unknown/no-modifier ids yield no mods) so the Go mirror collects byte-equal.
 * Appended last — must match the Go `conditionActiveItem` position for parity.
 */
function conditionActiveItem(character: Character): ActiveItem | null {
  const ids = parseConditionIds(character.activeConditions)
  const modifiers = ids.flatMap((id) => conditionModifiers(id))
  if (modifiers.length === 0) return null
  return { source: 'Condições', equipped: 'vested', modifiers }
}

/** Parse the persisted ConditionId[] blob to ids (bad blob ⇒ none). */
function parseConditionIds(raw: string | undefined): ConditionId[] {
  if (!raw) return []
  try {
    const parsed = JSON.parse(raw)
    if (!Array.isArray(parsed)) return []
    return parsed.filter((x): x is ConditionId => typeof x === 'string')
  } catch {
    return []
  }
}

function classActiveItems(character: Character): ActiveItem[] {
  const chosen = parseChoiceSet(character.classPowers)
  const choices = parseClassChoices(character.classChoices)
  const out: ActiveItem[] = []
  for (const entry of character.classes) {
    // One ActiveItem PER POWER so breakdown dialogs name the actual poder
    // ("Pele de Ferro +4"), not an opaque "Classe: Bárbaro 6" bundle.
    // classChoices resolves grantedByChoice rows (Caminho do Arcanista).
    for (const power of ownedClassPowers(
      entry.className,
      entry.level,
      chosen,
      choices[entry.className],
    )) {
      if (!power.modifiers || power.modifiers.length === 0) continue
      out.push({
        source: power.name,
        equipped: 'vested',
        modifiers: power.modifiers,
      })
    }
  }
  return out
}

/**
 * General powers (Poder de Combate, etc.) live in their own catalog and are
 * stored in the same `classPowers` JSON blob by their bare catalog id
 * (e.g. `esquiva`) — the same ids the picker toggles. We resolve every
 * chosen id through getGeneralPower; non-general ids (class electives)
 * return undefined and are skipped. (Earlier this filtered on a `general.`
 * prefix that the picker never writes, so power modifiers never applied.)
 */
function generalPowerActiveItem(character: Character): ActiveItem[] {
  const chosen = parseChoiceSet(character.classPowers)
  const out: ActiveItem[] = []
  for (const id of chosen) {
    const power = getGeneralPower(id)
    if (!power?.modifiers || power.modifiers.length === 0) continue
    // Per-power source — same provenance rule as class powers.
    out.push({ source: power.name, equipped: 'vested', modifiers: power.modifiers })
  }
  return out
}

function originActiveItem(character: Character): ActiveItem | null {
  const origin = getOrigin(character.origin)
  if (!origin) return null
  const choices = parseChoiceSet(character.originChoices)
  const mods = [
    ...originModifiers(origin, choices),
    // Free-pick benefits ("um poder de combate a sua escolha"): the concrete
    // power picked lives in powerChoices — fold its modifiers as origem's.
    ...originPickedPowerIds(character).flatMap(
      (id) => getGeneralPower(id)?.modifiers ?? [],
    ),
  ]
  if (mods.length === 0) return null
  return {
    source: `Origem: ${origin.name}`,
    equipped: 'vested',
    modifiers: mods,
  }
}

/**
 * Concrete powers picked via free-pick origem benefits (`powerPick`): for each
 * CHOSEN benefit, its powerChoices entry names the picked power id. Owned for
 * all downstream rules (modifiers, Tormenta Carisma loss).
 */
export function originPickedPowerIds(character: Character): string[] {
  const chosen = parseChoiceSet(character.originChoices)
  if (chosen.size === 0) return []
  let blob: unknown
  try {
    blob = JSON.parse(character.powerChoices)
  } catch {
    return []
  }
  if (!blob || typeof blob !== 'object') return []
  const record = blob as Record<string, unknown>
  const out: string[] = []
  for (const benefitId of chosen) {
    if (!getOriginBenefit(benefitId)?.powerPick) continue
    const picked = record[benefitId]
    if (!Array.isArray(picked)) continue
    out.push(...picked.filter((x): x is string => typeof x === 'string'))
  }
  return out
}

// racasList() is primed by the loader gate; a module-level Map would evaluate
// before priming (empty). 17 racas → linear find is cheap.
function racaByName(name: string) {
  return racasList().find((r) => r.name === name)
}

function parseRaceAttributeChoices(raw: string): {
  floatingPicks: AttributeKey[]
  ascendencia?: string
  deformidade?: DeformidadeStored
} {
  try {
    const p = JSON.parse(raw) as {
      floatingPicks?: unknown
      ascendencia?: unknown
      deformidade?: unknown
    }
    return {
      floatingPicks: Array.isArray(p.floatingPicks)
        ? (p.floatingPicks.filter((x) => typeof x === 'string') as AttributeKey[])
        : [],
      ascendencia: typeof p.ascendencia === 'string' ? p.ascendencia : undefined,
      deformidade: parseDeformidadeStored(p.deformidade),
    }
  } catch {
    return { floatingPicks: [] }
  }
}

type RaceAttrChoice = {
  floatingPicks?: AttributeKey[]
  ascendencia?: string
  deformidade?: DeformidadeStored
}

type DeformidadeStored = { pericias: string[]; tormentaPower?: string }

function parseDeformidadeStored(raw: unknown): DeformidadeStored | undefined {
  if (!raw || typeof raw !== 'object') return undefined
  const d = raw as { pericias?: unknown; tormentaPower?: unknown }
  if (!Array.isArray(d.pericias)) return undefined
  return {
    pericias: d.pericias.filter((x): x is string => typeof x === 'string'),
    tormentaPower:
      typeof d.tormentaPower === 'string' && d.tormentaPower
        ? d.tormentaPower
        : undefined,
  }
}

/**
 * Deformidade (Lefou p23) as engine modifiers: +2 in each chosen perícia.
 * The Carisma loss lives in `tormentaCarismaItem` — it must escalate over the
 * TOTAL real power count (p136), so it can't be emitted per-source here.
 */
function deformidadeModifiers(
  raceName: string,
  draft: DeformidadeStored | undefined,
): Modifier[] {
  if (!draft || !raceWithDeformidade([raceName])) return []
  return draft.pericias
    .filter((n) => (EXPERTISE_NAMES as readonly string[]).includes(n))
    .map((n) => ({
      target: { k: 'expertise', name: n as ExpertiseName },
      amount: DEFORMIDADE_PERICIA_BONUS,
      bonusType: 'untyped',
      note: 'Deformidade',
    }))
}

/** The Deformidade-swapped poder da Tormenta, from either race blob. */
export function deformidadeHeldPower(character: Character): string | undefined {
  const primaryRace = character.races[0]?.race
  const primary = parseRaceAttributeChoices(character.raceAttributeChoices)
  if (primaryRace && raceWithDeformidade([primaryRace])) {
    const p = primary.deformidade?.tormentaPower
    if (p) return p
  }
  for (const [race, choice] of parseSecondaryRaceChoices(
    character.secondaryRaceChoices,
  )) {
    if (raceWithDeformidade([race]) && choice.deformidade?.tormentaPower) {
      return choice.deformidade.tormentaPower
    }
  }
  return undefined
}

/**
 * Carisma loss from ALL real poderes da Tormenta (p136): the Deformidade swap
 * plus any picked in the Poderes pool (stored in classPowers). The loss
 * escalates with the total (1→1, 2→2, 3→4…), so it's one modifier computed
 * over the count — never summed per source. Deformidade perícia bonuses don't
 * count (p23).
 */
function tormentaCarismaItem(character: Character): ActiveItem | null {
  const picked = [
    ...new Set([
      ...parseChoiceSet(character.classPowers),
      ...originPickedPowerIds(character),
    ]),
  ].filter((id) => id in tormentaPowersRecord())
  const held = deformidadeHeldPower(character)
  const count = picked.length + (held && !picked.includes(held) ? 1 : 0)
  if (count === 0) return null
  return {
    source: 'Poderes da Tormenta',
    equipped: 'vested',
    modifiers: [
      {
        target: { k: 'attribute', name: 'charisma' },
        amount: -carismaLossFromPowers(count),
        bonusType: 'untyped',
        note: `${count} poder(es) da Tormenta (p136)`,
      },
    ],
  }
}

/**
 * A race's attribute mod as `{k:'attribute'}` modifiers, derived from its
 * floating-pick / ascendência choices (the abilities catalog's
 * `attributeBonuses` only covers fixed races). Stored attributes are BASE, so
 * this is applied exactly once. Returns `[]` on incomplete choices.
 */
function raceAttributeMods(raceName: string, choice: RaceAttrChoice): Modifier[] {
  const raca = racaByName(raceName)
  if (!raca) return []
  let deltas: Partial<Record<AttributeKey, number>>
  try {
    deltas = resolveAtributoMod(raca, choice)
  } catch {
    return []
  }
  return Object.entries(deltas)
    .filter(([, amount]) => amount)
    .map(([attr, amount]) => ({
      target: { k: 'attribute', name: attr as AttributeKey },
      amount: amount as number,
      bonusType: 'untyped',
      note: raca.name,
    }))
}

/** Opted-in secondary races → their attribute choices, keyed by race name. */
function parseSecondaryRaceChoices(raw: string): Map<string, RaceAttrChoice> {
  try {
    const arr = JSON.parse(raw) as unknown
    if (!Array.isArray(arr)) return new Map()
    return new Map(
      arr
        .filter((e): e is { race: string } => typeof e?.race === 'string')
        .map((e) => {
          const x = e as {
            race: string
            floatingPicks?: unknown
            ascendencia?: unknown
            deformidade?: unknown
          }
          return [
            x.race,
            {
              floatingPicks: Array.isArray(x.floatingPicks)
                ? (x.floatingPicks.filter(
                    (a) => typeof a === 'string',
                  ) as AttributeKey[])
                : undefined,
              ascendencia:
                typeof x.ascendencia === 'string' ? x.ascendencia : undefined,
              deformidade: parseDeformidadeStored(x.deformidade),
            },
          ] as const
        }),
    )
  } catch {
    return new Map()
  }
}

/**
 * Race modifiers folded into the sheet: the primary race always, plus any
 * opted-in secondary (GM-negotiated). Attribute mods come from each race's
 * persisted choices; non-attribute mods (PV/PM, perícias) via `raceModifiers`.
 * Non-applied secondary races contribute nothing.
 */
function raceActiveItems(character: Character): ActiveItem[] {
  const variantChoices = parseChoiceSet(character.raceAbilityChoices)
  const primaryChoice = parseRaceAttributeChoices(character.raceAttributeChoices)
  const secondaries = parseSecondaryRaceChoices(character.secondaryRaceChoices)
  const result: ActiveItem[] = []
  character.races.forEach((entry, i) => {
    const choice = i === 0 ? primaryChoice : secondaries.get(entry.race)
    if (choice === undefined) return // non-applied secondary → no mechanics
    const race = getRace(entry.race)
    if (!race) return
    // Strip raceModifiers' own attribute mods (fixed-race duplicates); attrs
    // come from the persisted choices so floating picks apply exactly once.
    const nonAttr = raceModifiers(race, variantChoices).filter(
      (m) => m.target.k !== 'attribute',
    )
    const mods = [
      ...raceAttributeMods(entry.race, choice),
      ...nonAttr,
      ...deformidadeModifiers(entry.race, choice.deformidade),
    ]
    if (mods.length === 0) return
    result.push({
      source: `Raça: ${race.name}`,
      equipped: 'vested',
      modifiers: mods,
    })
  })
  return result
}

/** Parse a JSON string[] choice column (classPowers, originChoices…) into a
 *  Set — exported for grant-ownership checks in use-power-action (Fase 4). */
export function parseChoiceSet(raw: string): Set<string> {
  try {
    const parsed = JSON.parse(raw)
    if (Array.isArray(parsed)) {
      return new Set(parsed.filter((x): x is string => typeof x === 'string'))
    }
    return new Set()
  } catch {
    return new Set()
  }
}

export type PrerequisiteCheck = {
  prereq: Prerequisite
  met: boolean
  reason: string
}

/**
 * Auto-checks a single Prerequisite against the character. `power`/`anyPower`
 * test the chosen-class-power set; `trained` looks at the perícia row;
 * `attribute` compares raw character attribute; `classChoice` reads from
 * the parsed classChoices blob (devoto/caminho picks). `note` cannot be
 * auto-checked — returns `met:true` so it never blocks selection; UI shows
 * reason text as info hint.
 */
export function evaluatePrerequisite(
  prereq: Prerequisite,
  character: Character,
  chosenPowerIds: ReadonlySet<string>,
  classChoices: ClassChoices,
): PrerequisiteCheck {
  switch (prereq.kind) {
    case 'power': {
      const met = chosenPowerIds.has(prereq.id)
      return { prereq, met, reason: powerLabel(prereq.id) }
    }
    case 'anyPower': {
      const met = prereq.ids.some((id) => chosenPowerIds.has(id))
      const reason = prereq.ids.map(powerLabel).join(' ou ')
      return { prereq, met, reason }
    }
    case 'trained': {
      const exp = character.expertises.find((e) => e.name === prereq.expertise)
      return {
        prereq,
        met: exp?.trained === true,
        reason: `Treinado em ${prereq.expertise}`,
      }
    }
    case 'attribute': {
      const value = character[prereq.attr]
      return {
        prereq,
        met: value >= prereq.min,
        reason: `${ATTRIBUTE_ABBR[prereq.attr]} ${prereq.min}+`,
      }
    }
    case 'classChoice': {
      const value = classChoices[prereq.class]?.[prereq.field]
      let met = !!value
      if (met && prereq.allowed) met = prereq.allowed.includes(value!)
      if (met && prereq.forbidden) met = !prereq.forbidden.includes(value!)
      return { prereq, met, reason: prereq.label }
    }
    case 'note':
      return { prereq, met: true, reason: prereq.description }
  }
}

/**
 * Parses Character.classChoices JSON. Missing/malformed → empty object,
 * so the rest of the engine treats the character as having no choices yet.
 */
export function parseClassChoices(raw: string): ClassChoices {
  try {
    const parsed = JSON.parse(raw)
    if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
      return parsed as ClassChoices
    }
    return {}
  } catch {
    return {}
  }
}

/**
 * The character's Caminho do Arcanista, if any — the pick that decides the
 * atributo-chave das magias (p37). `undefined` for a non-Arcanista or a sheet
 * that has not made the 1st-level choice yet.
 *
 * @example arcanistaCaminhoOf(samira) // 'feiticeiro'
 */
export function arcanistaCaminhoOf(character: Character): string | undefined {
  return parseClassChoices(character.classChoices).Arcanista?.caminho
}

function powerLabel(id: string): string {
  return getClassPower(id)?.name ?? getGeneralPower(id)?.name ?? id
}

export function parseEffectModifiers(raw: string): Modifier[] {
  try {
    const parsed = JSON.parse(raw)
    if (Array.isArray(parsed)) return parsed as Modifier[]
    return []
  } catch {
    return []
  }
}

export function parseImprovementIds(raw: string): string[] {
  try {
    const parsed = JSON.parse(raw)
    if (Array.isArray(parsed)) {
      return parsed.filter((x): x is string => typeof x === 'string')
    }
    return []
  } catch {
    return []
  }
}

function parseProficiencySetFromCharacter(character: Character): Set<string> {
  try {
    const parsed = JSON.parse(character.proficiencies)
    if (Array.isArray(parsed)) {
      return new Set(parsed.filter((x): x is string => typeof x === 'string'))
    }
    return new Set()
  } catch {
    return new Set()
  }
}

/**
 * Mirror a weapon's own `{k:'attack', scope:'this'}` mods (desbalanceada -2,
 * melhoria Certeira +1, materials…) onto its attack perícia (Luta/Pontaria) —
 * the same route `nonProficiencyPenalties` takes for the -5. T20 resolves
 * attacks as expertise tests, so this is what makes the penalty land in the
 * "Ataque Corpo a Corpo" box and its breakdown (named per weapon) instead of
 * hiding inside a per-weapon target no total reads. Mirrors are 'untyped' so
 * they never collide with genuine expertise bonuses in `resolveStack`; the
 * original mod keeps its own bonusType for the item chip.
 */
function mirrorWeaponAttackMods(
  catalog: CatalogItem | undefined,
  ownMods: readonly Modifier[],
): Modifier[] {
  if (!catalog?.weapon) return []
  const expertise = catalog.weapon.purpose === 'melee' ? 'Luta' : 'Pontaria'
  return ownMods
    .filter((m) => m.target.k === 'attack' && m.target.scope === 'this')
    .map((m) => ({
      target: { k: 'expertise', name: expertise },
      amount: m.amount,
      bonusType: 'untyped',
      condition: m.condition ?? { c: 'wielded' },
      // Breakdown rows already name the item (Contribution.source) — the
      // note only carries the WHY ("desbalanceada: -2 em ataque").
      note: m.note ?? 'bônus desta arma',
    }))
}

/**
 * An overlay's modifiers with the overlay's NAME folded into each note, so
 * breakdown rows say WHICH melhoria/material a bonus came from — "Couraça +1"
 * with note "Reforçada: +1 Defesa" instead of a bare "+1 Defesa" (the
 * Contribution.source only carries the host item's name).
 */
function overlayModsWithProvenance(
  overlay: CatalogItem | undefined,
): Modifier[] {
  if (!overlay) return []
  return overlay.modifiers.map((m) => ({
    ...m,
    note: m.note?.includes(overlay.name)
      ? m.note
      : `${overlay.name}${m.note ? `: ${m.note}` : ''}`,
  }))
}

/**
 * HOMEBREW (opt-in, not RAW): esotéricos in the shared HOMEBREW_VESTED_OK
 * registry (Medalhão de prata) may be WORN. RAW p159 grants the bonus only
 * EMPUNHADO, so while vested every wielded-gated modifier the item carries —
 * its own catalog line AND its overlays (melhoria Vigilante's +2 Defesa,
 * materials) — stays off; this offers them back as ONE Efeitos toggle
 * (flagOn groups all of them under a single switch). Same bonusType as the
 * originals, so non-stacking absorbs any double route.
 */
function vestedEsotericHomebrewMods(
  equipped: ActiveItem['equipped'],
  catalog: CatalogItem | undefined,
  ownMods: readonly Modifier[],
): Modifier[] {
  if (equipped !== 'vested') return []
  if (!catalog || !HOMEBREW_VESTED_OK.has(catalog.id)) return []
  return ownMods
    .filter((m) => m.condition?.c === 'wielded')
    .map((m) => ({
      ...m,
      condition: {
        c: 'flagOn',
        flag: `homebrew-vestido-${catalog.id}`,
        label:
          'Homebrew: esotérico vestido mantém o bônus (RAW exige empunhar, p159)',
      },
    }))
}

/**
 * HOMEBREW (opt-in, not RAW): some tables rule that the Equilibrada melhoria
 * cancels the weapon's desbalanceada trait. The book keeps both (-2 ataque
 * p149; +2 manobras p164), so the counter ships as a CONDITIONAL +2 on the
 * weapon's attack perícia — it surfaces as a toggle in the Efeitos tab and
 * only nets the -2 out while the player keeps it switched on.
 */
function equilibradaHomebrewMods(
  catalog: CatalogItem | undefined,
  improvementIds: readonly string[],
): Modifier[] {
  if (!catalog?.weapon?.traits.includes('desbalanceada')) return []
  if (!improvementIds.includes('melhoria-equilibrada')) return []
  const expertise = catalog.weapon.purpose === 'melee' ? 'Luta' : 'Pontaria'
  return [
    {
      target: { k: 'expertise', name: expertise },
      amount: 2,
      bonusType: 'untyped',
      condition: {
        c: 'context',
        note: 'Homebrew: Equilibrada anula a desbalanceada (-2 → 0)',
      },
      note: 'anula desbalanceada',
    },
  ]
}

/**
 * T20 p142: non-proficient weapon use → -5 attack. Armor/shield without
 * proficiency → cannot apply Dex to Defense and the armor penalty extends to
 * all expertise tests. Penalties are emitted as synthetic modifiers attached
 * to the offending ActiveItem so the standard engine resolves them alongside
 * catalog mods. Attack rolls are mapped onto the Luta/Pontaria perícia (T20
 * resolves attacks as expertise tests) so the penalty becomes visible in the
 * expertise breakdown rather than hiding inside a per-weapon target.
 */
function nonProficiencyPenalties(
  catalog: CatalogItem,
  proficiencies: ReadonlySet<string>,
): Modifier[] {
  const required = requiredProficiency(catalog)
  if (!required) return []
  if (proficiencies.has(required)) return []
  if (catalog.category.startsWith('weapon-')) {
    const purpose = catalog.weapon?.purpose
    const attackExpertise = purpose === 'melee' ? 'Luta' : 'Pontaria'
    return [
      {
        target: { k: 'attack', scope: 'this' },
        amount: -5,
        bonusType: 'untyped',
        condition: { c: 'wielded' },
        note: 'sem proficiência',
      },
      {
        target: { k: 'expertise', name: attackExpertise },
        amount: -5,
        bonusType: 'untyped',
        condition: { c: 'wielded' },
        note: 'sem proficiência: -5 em testes de ataque (p142)',
      },
    ]
  }
  const armorPenaltyMod = catalog.modifiers?.find(
    (m) => m.target.k === 'armorPenalty',
  )
  const basePenalty = armorPenaltyMod?.amount ?? -1
  return [
    {
      target: { k: 'flag', name: 'cannot-apply-dex-to-defense' },
      amount: 1,
      bonusType: 'untyped',
      condition: { c: 'vested' },
      note: 'sem proficiência',
    },
    {
      target: { k: 'expertiseAll' },
      amount: basePenalty,
      bonusType: 'untyped',
      condition: { c: 'vested' },
      note: `${catalog.name} sem proficiência`,
    },
  ]
}

/** Does the character have the proficiency required to use this item without penalty? */
export function isItemProficient(
  character: Character,
  item: CharacterItem,
): boolean {
  if (!item.catalogId) return true
  const catalog = getCatalogItem(item.catalogId)
  if (!catalog) return true
  const required = requiredProficiency(catalog)
  if (!required) return true
  const set = parseProficiencySetFromCharacter(character)
  return set.has(required)
}

/**
 * The sheet derive's CHOKE POINT (Inc.2 task #7/#8): resolve a character's
 * `ItemEffects` through the Go/WASM engine — the single source of truth.
 *
 * O motor Go faz a derivação inteira (coleta + resolução) em TODOS os ambientes,
 * inclusive nos testes: o `test-setup` do vitest carrega o mesmo `.wasm` que a
 * produção usa. Não existe mais ramo TS de fallback nem `import.meta.env.MODE`
 * aqui — eles morreram com o `t20-data` (ALE-104/117), e o que restou de TS
 * neste arquivo é só o que a UI ainda monta por conta própria.
 */
function resolveEffects(
  character: Character,
  activeConditionals: ReadonlySet<string>,
): ItemEffects {
  if (!areEngineCatalogsPrimed()) {
    throw new Error(
      'sheet derive: WASM engine not primed — ensureEngineCatalogs() must resolve before any sheet renders',
    )
  }
  return engineComputeEffects(character, [...activeConditionals])
}

export function characterEffects(
  character: Character,
  activeConditionals: ReadonlySet<string> = EMPTY_SET,
): ItemEffects {
  return resolveEffects(character, activeConditionals)
}


const EMPTY_SET: ReadonlySet<string> = new Set()

/**
 * Ported from the React `useCharacterEffects`, which only existed to read the
 * toggled conditionals out of a Zustand store and hand them to
 * `characterEffects`. The set comes in as an argument here, so this file stays
 * framework-free and the Solid side wires the store where it belongs.
 */
export function characterEffectsWith(
  character: Character,
  active: ReadonlySet<string>,
): ItemEffects {
  return characterEffects(character, active)
}

export type ConditionalEntry = {
  id: string
  effect: ConditionalEffect
  active: boolean
}

/** Every opt-in the character could toggle, flagged with what's on right now. */
export function allConditionals(
  character: Character,
  active: ReadonlySet<string>,
): ConditionalEntry[] {
  // Base effects (no conditionals applied) — the `conditional` list enumerates
  // every opt-in the character could toggle. Same choke point as the sheet.
  const raw = resolveEffects(character, EMPTY_SET)
  return raw.conditional.map((effect) => {
    const id = conditionalId(effect)
    return { id, effect, active: active.has(id) }
  })
}

/**
 * Expertise total: ½ level + attribute + training + item modifiers
 * (typed, with non-stacking rules already resolved).
 */
const ARMOR_PENALTY_EXPERTISES = new Set([
  'Acrobacia',
  'Furtividade',
  'Ladinagem',
])

/**
 * Redução de Dano agregada do personagem, para exibir junto da Defesa.
 * Fontes cobertas (todas passivas e deriváveis do estado da ficha):
 *  - Bárbaro: tabela p42 (2/4/6/8/10 nos níveis 5/8/11/14/17)
 *  - Cavaleiro Caminho do Bastião (p55): RD 5 em armadura pesada, escolhido no
 *    5º nível
 *  - Especialização em Armadura: poder ESCOLHIDO com pré-requisito de 12º nível
 *    na classe, RD 5 fixa em armadura pesada — existe igual para Cavaleiro
 *    (p54) e Guerreiro (p65), e as duas descrições dizem que é CUMULATIVA com
 *    Bastião. O Guerreiro NÃO tem RD passiva: o motor dava a ele a progressão do
 *    Bárbaro desde o 5º nível, o que não existe no livro (ALE-111).
 * RD geral não acumula entre fontes (vale a maior, p290) — exceto a
 * cumulatividade explícita acima.
 *
 * @example characterDamageReduction(barbaro8, effects).total // 4
 */
/**
 * Effective attribute value = raw character attribute + sum of `attribute`
 * target modifiers (race bonuses, items, active effects). Negative bonuses
 * apply too.
 */
export function attributeTotal(
  character: Character,
  attr: AttributeKey,
  effects: ItemEffects,
): number {
  const raw = character[attr]
  const stat = statFor(effects, { k: 'attribute', name: attr })
  return raw + stat.total
}

export function expertiseTotalWithItems(
  character: Character,
  state: CharacterExpertise,
  effects: ItemEffects,
): {
  base: number
  itemBonus: number
  total: number
  halfLevel: number
  attrValue: number
  training: number
  itemContributions: { source: string; amount: number; note?: string }[]
  armorPenaltyApplied: number
} {
  const halfLevel = Math.floor(character.level / 2)
  const attrValue = attributeTotal(character, state.attribute, effects)
  const training = state.trained ? trainingBonusForLevel(character.level) : 0
  const base = halfLevel + attrValue + training
  const stat = statFor(effects, { k: 'expertise', name: state.name as never })
  const allStat = statFor(effects, { k: 'expertiseAll' })
  const byAttrStat = statFor(effects, {
    k: 'expertiseByAttribute',
    attribute: state.attribute,
  })
  // Uma perícia recebe modificador por TRÊS alvos, e somar os três totais deixa
  // efeitos do mesmo tipo acumularem entre si — o que a p226 proíbe. Um
  // personagem exausto E atordoado levava −10 em Reflexos (−5 pelas perícias de
  // Destreza do debilitado, −5 pelo Reflexos do desprevenido) onde o livro manda
  // aplicar só o mais severo. Resolver a CONCATENAÇÃO faz os três alvos
  // competirem como um só número, que é o que a ficha mostra (ALE-116).
  const merged = resolveStack([
    ...stat.contributions,
    ...allStat.contributions,
    ...byAttrStat.contributions,
  ])
  const itemContributions = merged.contributions.map((c) => ({
    source: c.source,
    amount: c.amount,
    ...(c.note ? { note: c.note } : {}),
  }))

  let armorPenaltyApplied = 0
  if (ARMOR_PENALTY_EXPERTISES.has(state.name)) {
    armorPenaltyApplied = armorPenaltyTotal(effects)
    if (armorPenaltyApplied !== 0) {
      itemContributions.push({
        source: 'Penalidade de armadura',
        amount: armorPenaltyApplied,
      })
    }
  }

  const itemBonus = merged.total
  return {
    base,
    itemBonus: itemBonus + armorPenaltyApplied,
    total: base + itemBonus + armorPenaltyApplied,
    halfLevel,
    attrValue,
    training,
    itemContributions,
    armorPenaltyApplied,
  }
}

function armorPenaltyTotal(effects: ItemEffects): number {
  return statFor(effects, { k: 'armorPenalty' }).total
}
