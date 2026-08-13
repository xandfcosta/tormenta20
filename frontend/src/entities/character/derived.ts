import { applyActiveConditionals, barbaroRdForLevel, CAVALEIRO_BASTIAO_RD, carismaLossFromPowers, CLASS_SPELLCASTING_ATTRIBUTE, computeItemEffects, conditionModifiers, conditionalId, type ConditionId, DEFORMIDADE_PERICIA_BONUS, EXPERTISE_NAMES, type ExpertiseName, HOMEBREW_VESTED_OK, originModifiers, raceModifiers, requiredProficiency, resolveAtributoMod, resolveStack, spellcastingAttributeFor, spellSaveDc, statFor, trainingBonusForLevel, type ActiveItem, type CatalogItem, type ClassChoices, type ConditionalEffect, type ItemEffects, type Modifier, type Prerequisite } from '@tormenta20/t20-data'
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
 * In PRODUCTION/dev the engine runs the whole heavy derive (collection +
 * resolution) in Go. The TS branch below is TEST-ONLY: it keeps the TS derive as
 * the parity oracle + vitest backing so the unit suite needs no wasm. Because
 * `import.meta.env.MODE` is statically `'production'` in the app build, that
 * branch — and everything only it reaches (`activeItemsFor` + the whole
 * collection layer, `computeItemEffects`, `applyActiveConditionals`) — is
 * dead-code-eliminated, so the front bundle ships ONLY the Go engine (task #8).
 * Parity is proven byte-equal by the `itemEffects` oracle across the 16 seeds.
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

/**
 * A mesma resolução em TypeScript — a IMPLEMENTAÇÃO DE REFERÊNCIA que gera o
 * oráculo de paridade.
 *
 * Chamada EXPLICITAMENTE, nunca por `if` de ambiente: quando o harness passou a
 * atravessar o choke point, o oráculo virou "o Go dizendo o que o Go acha", que
 * é a ilusão que a fatia 5 evita enquanto ainda existem duas implementações.
 * Morre com o `t20-data` (ALE-109).
 */
export function tsCharacterEffects(
  character: Character,
  activeConditionals: ReadonlySet<string> = EMPTY_SET,
): ItemEffects {
  return applyActiveConditionals(
    computeItemEffects(activeItemsFor(character)),
    activeConditionals,
  )
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

/**
 * Defense total: 10 + Dex (capped if heavy armor) + armor + shield + other typed mods.
 */
export function defenseTotal(
  character: Character,
  effects: ItemEffects,
): {
  base: number
  itemBonus: number
  total: number
  dexApplied: boolean
  vsMelee: number
  vsRanged: number
  contributions: { source: string; amount: number; note?: string }[]
} {
  const stat = statFor(effects, { k: 'defense' })
  const dexApplied = !effects.flags.has('cannot-apply-dex-to-defense')
  const effectiveDex = attributeTotal(character, 'dexterity', effects)
  const base = 10 + (dexApplied ? effectiveDex : 0)
  // Defesa DIRECIONAL: igual ao total na maioria das fichas, separada quando
  // algo distingue a direção do ataque — hoje só o Caído (p394).
  const melee = statFor(effects, { k: 'defense', scope: 'melee' })
  const ranged = statFor(effects, { k: 'defense', scope: 'ranged' })
  const total = base + stat.total
  return {
    base,
    itemBonus: stat.total,
    total,
    dexApplied,
    vsMelee: total + melee.total,
    vsRanged: total + ranged.total,
    contributions: [...stat.contributions, ...melee.contributions, ...ranged.contributions].map(
      (c) => ({
        source: c.source,
        amount: c.amount,
        ...(c.note ? { note: c.note } : {}),
      }),
    ),
  }
}

export function displacementTotal(
  character: Character,
  effects: ItemEffects,
): {
  base: number
  itemBonus: number
  total: number
  contributions: { source: string; amount: number; note?: string }[]
} {
  const stat = statFor(effects, { k: 'displacement' })
  return {
    base: character.displacement,
    itemBonus: stat.total,
    total: Math.max(0, character.displacement + stat.total),
    contributions: stat.contributions.map((c) => ({
      source: c.source,
      amount: c.amount,
      ...(c.note ? { note: c.note } : {}),
    })),
  }
}

/**
 * Fly speed (metros) granted by active effects — Voo and similar. Characters
 * have no innate fly base, so 0 means "can't fly" and the movement line hides
 * it. Shown, not folded into ground displacement.
 */
export function flySpeedTotal(effects: ItemEffects): number {
  return Math.max(0, statFor(effects, { k: 'flySpeed' }).total)
}

export function inventorySlotsTotal(
  character: Character,
  effects: ItemEffects,
): number {
  const effStr = attributeTotal(character, 'strength', effects)
  // PDF p141 (Carga): "10 espaços, +2 por ponto de Força (ou –1 por ponto de
  // Força negativo)". Math.abs here inflated negative-Str carriers (−2 → 14).
  const base = effStr >= 0 ? 10 + 2 * effStr : 10 + effStr
  const stat = statFor(effects, { k: 'inventorySlots' })
  return base + stat.total
}

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

export function attributeContributions(
  attr: AttributeKey,
  effects: ItemEffects,
): { source: string; amount: number; note?: string }[] {
  return statFor(effects, { k: 'attribute', name: attr }).contributions.map(
    (c) => ({ source: c.source, amount: c.amount }),
  )
}

export function armorPenaltyTotal(effects: ItemEffects): number {
  return statFor(effects, { k: 'armorPenalty' }).total
}

/**
 * Caster level for the per-spell PM cap — PDF p224: "o máximo de PM que você
 * pode gastar por uso é igual ao seu nível NA CLASSE que fornece a
 * habilidade". Multiclass casters take the best spellcasting-class level;
 * non-casters fall back to character level (their Limite PM box is hidden).
 * Ex.: Guerreiro 4 / Arcanista 4 → 4 (not ½ do nível 8 do personagem).
 */
export function casterLevelForPmLimit(character: Character): number {
  const casterLevels = character.classes
    .filter((c) => CLASS_SPELLCASTING_ATTRIBUTE[c.className] !== undefined)
    .map((c) => c.level)
  if (casterLevels.length === 0) return character.level
  return Math.max(...casterLevels)
}

export function pmLimitTotal(
  character: Character,
  effects: ItemEffects,
): {
  base: number
  itemBonus: number
  total: number
  contributions: { source: string; amount: number; note?: string }[]
} {
  const base = Math.max(1, casterLevelForPmLimit(character))
  const stat = statFor(effects, { k: 'pmLimit' })
  return {
    base,
    itemBonus: stat.total,
    total: base + stat.total,
    contributions: stat.contributions.map((c) => ({
      source: c.source,
      amount: c.amount,
      ...(c.note ? { note: c.note } : {}),
    })),
  }
}

/**
 * Best spell save CD across the character's caster classes — PDF p173:
 * CD = 10 + metade do nível DO PERSONAGEM + modificador do atributo-chave.
 * Uses the FINAL attribute (attributeTotal), so racial/item bonuses count — the
 * raw stored attribute understated the CD (Necromante Osteon: 21 shown, 22
 * correct) — and resolves the Arcanista's atributo-chave through its Caminho
 * (ALE-113).
 * Ex.: bestBaseSpellCd(arcanista12ComIntFinal6, effects) === 22
 */
export function bestBaseSpellCd(
  character: Character,
  effects: ItemEffects,
): number | null {
  const caminho = arcanistaCaminhoOf(character)
  let best: number | null = null
  for (const entry of character.classes) {
    const attr = spellcastingAttributeFor(entry.className, caminho)
    if (!attr) continue
    const dc = spellSaveDc(character.level, attributeTotal(character, attr, effects))
    if (best === null || dc > best) best = dc
  }
  return best
}

export function spellDCBonus(effects: ItemEffects): {
  total: number
  contributions: { source: string; amount: number; note?: string }[]
} {
  const stat = statFor(effects, { k: 'spellDC' })
  return {
    total: stat.total,
    contributions: stat.contributions.map((c) => ({
      source: c.source,
      amount: c.amount,
      ...(c.note ? { note: c.note } : {}),
    })),
  }
}

export function pmCostMod(effects: ItemEffects): {
  total: number
  contributions: { source: string; amount: number; note?: string }[]
} {
  const stat = statFor(effects, { k: 'pmCost' })
  return {
    total: stat.total,
    contributions: stat.contributions.map((c) => ({
      source: c.source,
      amount: c.amount,
      ...(c.note ? { note: c.note } : {}),
    })),
  }
}

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
/** "Especialização em Armadura" (Cavaleiro p54, Guerreiro p65): poder escolhido,
 *  pré-requisito de 12º nível na classe, RD 5 fixa com armadura pesada. */
const ESPECIALIZACAO_ARMADURA_RD = 5
const ESPECIALIZACAO_ARMADURA_LEVEL = 12

export function characterDamageReduction(
  character: Character,
  effects: ItemEffects,
): { total: number; sources: { source: string; amount: number }[] } {
  const heavy = effects.flags.has('armadura-pesada')
  const chosen = parseChoiceSet(character.classPowers)
  // Qualificado por classe: casar só pelo sufixo deixaria a escolha de uma
  // classe satisfazer o ramo da outra num multiclasse.
  const hasPower = (className: string, power: string) =>
    [...chosen].some((id) => id === `class.${className}.${power}` || id === power)

  const sources: { source: string; amount: number }[] = []
  for (const entry of character.classes) {
    if (entry.className === 'Bárbaro') {
      const rd = barbaroRdForLevel(entry.level)
      if (rd > 0) sources.push({ source: 'Bárbaro (p42)', amount: rd })
    }
    if (entry.className === 'Guerreiro' && heavy) {
      if (
        entry.level >= ESPECIALIZACAO_ARMADURA_LEVEL &&
        hasPower('guerreiro', 'especializacao-em-armadura')
      )
        sources.push({
          source: 'Especialização em Armadura',
          amount: ESPECIALIZACAO_ARMADURA_RD,
        })
    }
    if (entry.className === 'Cavaleiro' && heavy) {
      if (entry.level >= 5 && hasPower('cavaleiro', 'caminho-bastiao'))
        sources.push({ source: 'Bastião — armadura pesada', amount: CAVALEIRO_BASTIAO_RD })
      if (
        entry.level >= ESPECIALIZACAO_ARMADURA_LEVEL &&
        hasPower('cavaleiro', 'especializacao-em-armadura')
      )
        sources.push({
          source: 'Especialização em Armadura',
          amount: ESPECIALIZACAO_ARMADURA_RD,
        })
    }
  }
  // As de CLASSE competem entre si; a Especialização soma por cima, cumulativa
  // por texto explícito.
  const especializacao = sources
    .filter((s) => s.source === 'Especialização em Armadura')
    .reduce((sum, s) => sum + s.amount, 0)
  const general = Math.max(
    0,
    ...sources
      .filter((s) => s.source !== 'Especialização em Armadura')
      .map((s) => s.amount),
  )

  // RD concedida por MODIFICADOR — hoje só a do Petrificado (p394). Soma por
  // cima: a p226 diz que efeitos de origens diferentes acumulam, e uma estátua
  // de bárbaro tem as duas.
  const granted = statFor(effects, { k: 'damageReduction' })
  for (const c of granted.contributions) {
    sources.push({ source: c.source, amount: c.amount })
  }

  if (sources.length === 0) return { total: 0, sources }
  return { total: general + especializacao + granted.total, sources }
}

/** True when the Fúria stance is switched on in the Efeitos tab. */
export function isFuriaActive(entries: readonly ConditionalEntry[]): boolean {
  return entries.some((e) => e.effect.flag === 'furia' && e.active)
}

/**
 * PV temporários concedidos por poderes disparados por postura ativa.
 * Coberto: Alma de Bronze (Bárbaro p41) — "quando entra em fúria, recebe
 * PV temporários = nível + Força". Exibição apenas (a pool não é
 * persistida; o jogador abate dano dela manualmente).
 *
 * @example tempHpFromPowers(barbaroComAlma, effects, true).total // nível + For
 */
export function tempHpFromPowers(
  character: Character,
  effects: ItemEffects,
  furiaActive: boolean,
): { total: number; sources: { source: string; amount: number }[] } {
  if (!furiaActive) return { total: 0, sources: [] }
  const chosen = parseChoiceSet(character.classPowers)
  const owns = [...chosen].some(
    (id) => id === 'alma-de-bronze' || id.endsWith('.alma-de-bronze'),
  )
  if (!owns) return { total: 0, sources: [] }
  const amount = character.level + attributeTotal(character, 'strength', effects)
  return {
    total: amount,
    sources: [{ source: 'Alma de Bronze (Fúria, p41)', amount }],
  }
}
