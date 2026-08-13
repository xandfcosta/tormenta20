/**
 * O vocabulário de CATÁLOGO — as formas que o servidor entrega em `/catalog`.
 * Movido do `t20-data` quando ele foi aposentado (ALE-109).
 *
 * Por que escrito à mão e não gerado das structs Go: o servidor entrega catálogo
 * como BYTES CRUS, sem parsear, então para a maioria destes tipos não existe
 * struct nenhuma no motor, e as três que existem (Spell, Item, Activation) são
 * subconjuntos deliberados do que os caminhos de conjurar e aplicar leem —
 * gerar delas produziria tipos sem `name`, `bookPage` ou `baseEffect` (ALE-108).
 *
 * A validação de que o dado servido casa com estas formas é de SCHEMA, no Go
 * (`catalog/rules_tables_test.go`), que é onde o dado é autorado.
 */
import type { AttributeKey } from './attribute-keys'
import type { DisplayFact } from './display-facts'
import type { ExpertiseName } from './expertise-names'
import type { Modifier } from './item-types'
import type {
  AugmentKind,
  SpellCircle,
  SpellComponent,
  SpellDuration,
  SpellExecution,
  SpellRange,
  SpellResistance,
  SpellSaveType,
  SpellSchool,
} from './spell-types'

// ─── de abilities/classes/ownership.ts ──────────────────────
/** One class's picks in Character.classChoices (per-class JSON blob). */
export type ClassChoiceSelections = { devoto?: string; caminho?: string }

// ─── de abilities/deuses.ts ─────────────────────────────────
/**
 * Per-class persisted choices keyed by className. Stored as JSON on
 * Character.classChoices and merged into prerequisite evaluation.
 */
export type ClassChoiceBlob = {
  /** Deus id chosen as devoto (clérigo/paladino/druida). */
  devoto?: string
  /** Caminho/subpath id (arcanista, paladino L5, cavaleiro L5, ...). */
  caminho?: string
}

export type ClassChoices = Partial<Record<string, ClassChoiceBlob>>

export type Deus = {
  id: string
  name: string
  major: boolean
  paladinoEligible: boolean
  druidaEligible: boolean
  /** PDF p96-105 enrichment fields — empty on Panteão / Paladino do Bem sentinels. */
  portfolio?: string
  energia?: DeusEnergia
  simbolo?: string
  /**
   * Arma preferida. `null` quando o deus proíbe explicitamente Arma
   * Espiritual (Lena, Marah). Nimb usa `'todas'` por sua natureza caótica.
   */
  armaPreferida?: string | null
  /** Sempre 4 poderes concedidos por deus maior (book Cap 2 — Poderes Concedidos). */
  poderesConcedidos?: readonly string[]
  /** Raças + classes elegíveis a devoção (verbatim do "Devotos." line). */
  devotos?: readonly string[]
  bookPage?: number
}

// ─── de abilities/general-powers.ts ─────────────────────────
/**
 * General powers (PDF Cap 4 — power pools shared across classes). Stored
 * separately from class-specific electives because multiple classes draw from
 * the same pool. Same shape as ClassPower minus the `className` field.
 */
export type GeneralPower = {
  id: string
  kind: PowerKind
  name: string
  description: string
  minLevel?: number
  prerequisites?: Prerequisite[]
  modifiers?: Modifier[]
}

/**
 * Power "kinds" mirror the PDF's four general-power pools — Combate, Destino,
 * Magia, Tormenta — plus per-class pools (`barbaro`, `bardo`, etc.). When a
 * class grants a power slot at a given level, the slot's `kinds` list tells
 * the picker which pools the player may draw from.
 */
export type PowerKind =
  | 'combate'
  | 'destino'
  | 'magia'
  | 'tormenta'
  | 'arcanista'
  | 'barbaro'
  | 'bardo'
  | 'bucaneiro'
  | 'cacador'
  | 'cavaleiro'
  | 'clerigo'
  | 'druida'
  | 'guerreiro'
  | 'inventor'
  | 'ladino'
  | 'lutador'
  | 'nobre'
  | 'paladino'

// ─── de abilities/granted-powers.ts ─────────────────────────
export type GrantedPower = {
  id: string
  name: string
  deuses: readonly string[]
  effect: string
  kind: GrantedPowerKind
  bookPage: number
  /** Numeric passives folded into the sheet when the character's `godPower`
   *  names this poder (Bênção do Mana → maxPm por nível ímpar). */
  modifiers?: Modifier[]
}

// ─── de abilities/types.ts ──────────────────────────────────
export type ClassPower = {
  id: string
  className: string
  name: string
  description: string
  /** When non-null, this power is automatically granted at this class level. */
  grantedAtLevel?: number
  /** Typed prerequisites — power refs, trained perícias, attribute mins, or
   *  free-form notes for gates not yet machine-checked. */
  prerequisites?: Prerequisite[]
  /** Minimum class level required to pick (separate from `grantedAtLevel`). */
  minLevel?: number
  modifiers?: Modifier[]
  /** Sub-choice the player resolves when taking this power (totem/school/…). */
  choice?: PowerChoice
  /**
   * Owned automatically when `Character.classChoices[className][field] ===
   * value` — no power slot spent. Models the Caminho do Arcanista rows (p36):
   * picking the caminho grants its habilidade; the other two stay locked.
   * Distinct from a `classChoice` PREREQUISITE, which only gates an elective
   * that still costs a slot (e.g. Autoridade Eclesiástica).
   */
  grantedByChoice?: { field: 'devoto' | 'caminho'; value: string }
}

/**
 * Origens (PDF Chapter 2). Each origin lists 4 perícias and 2 poderes the
 * player picks 2 benefits from, plus one exclusive poder único.
 */
export type OriginBenefit = {
  id: string
  name: string
  kind: 'pericia' | 'poder'
  description: string
  /** If kind='pericia', the expertise it trains. */
  expertise?: ExpertiseName
  modifiers?: Modifier[]
  /** Free-pick benefit ("um poder de combate/da Tormenta a sua escolha") —
   *  names the pool the player picks the concrete power from. */
  powerPick?: 'combate' | 'tormenta'
}

export type OriginDefinition = {
  id: string
  name: string
  /** Full benefit pool (typically 4 perícias + 2 poderes). */
  benefits: OriginBenefit[]
  /** Exclusive poder único — chosen as one of the two slots. */
  poderUnico: OriginBenefit
}

/**
 * A structured sub-choice a power requires when taken — Bárbaro's Totem
 * Espiritual picks an animal, Arcanista's Especialista em Escola picks a
 * school, Guerreiro's Foco em Arma picks a weapon. `options` enumerates the
 * choices; when omitted (weapon) the consumer sources them from the item
 * catalog. `repeatable` marks powers takeable multiple times with distinct
 * choices.
 */
export type PowerChoice = {
  kind: 'totem' | 'school' | 'companion' | 'weapon' | 'attribute'
  label: string
  options?: { id: string; name: string; note?: string }[]
  repeatable?: boolean
  /** When set, each option's `note` names a SPELL the power teaches, cast with
   *  this key attribute (Bárbaro Totem Espiritual: Sab, PDF p42). Consumers
   *  resolve `note` → catalog spell via `spellByName`. */
  grantsSpellAttribute?: AttributeKey
}

/**
 * Typed prerequisite for class powers and general powers. Covers the four
 * common gates plus a `note` escape hatch for prereqs not yet machine-checked
 * (Ofício sub-crafts, spell knowledge, etc.) — UI displays note verbatim.
 *
 *  - power: must own a specific other power (by id).
 *  - anyPower: must own at least one of the listed ids (used for "any
 *    armadilha", "any missa" rules).
 *  - trained: must be trained in the named expertise.
 *  - attribute: attribute (post-race-mods, raw character.X) must meet min.
 *  - classChoice: a per-class choice (devoto/caminho) stored in
 *    Character.classChoices must satisfy `allowed`/`forbidden`. When neither
 *    is set, any non-empty value satisfies (i.e., "must be devoto").
 *  - note: free-form description shown in UI; not auto-resolved.
 */
export type Prerequisite =
  | { kind: 'power'; id: string }
  | { kind: 'anyPower'; ids: string[] }
  | { kind: 'trained'; expertise: ExpertiseName }
  | { kind: 'attribute'; attr: AttributeKey; min: number }
  | {
      kind: 'classChoice'
      class: string
      field: 'devoto' | 'caminho'
      allowed?: string[]
      forbidden?: string[]
      /** Human-readable hint shown in UI ("Devoto, exceto Lena/Marah"). */
      label: string
    }
  | { kind: 'note'; description: string }

/**
 * Catalog entry for a racial ability. Mirrors how PDF Chapter 1 (Races)
 * presents each trait — a fixed grant per race, sometimes with sub-options
 * the player picks at creation (e.g., elf Linhagem).
 */
export type RaceAbility = {
  id: string
  raceId: string
  name: string
  description: string
  /** True when player must pick one variant from `options`. */
  variants?: RaceAbilityVariant[]
  /** Numeric modifiers folded into the engine when this ability is owned. */
  modifiers?: Modifier[]
  /** Display-only facts (visão no escuro, RD, imunidades) — shown as reference
   *  chips, not computed. Companion-app affordance for non-numeric traits. */
  facts?: DisplayFact[]
}

export type RaceDefinition = {
  id: string
  name: string
  /** Attribute deltas applied at creation (T20 racial bonuses). */
  attributeBonuses: Partial<Record<AttributeKey, number>>
  /** Innate abilities granted by this race. */
  abilities: RaceAbility[]
  /** Tormenta-touched race (Lefou) — unlocks the poderes da Tormenta pool at
   *  creation (book p18/p136). */
  grantsTormentaPowers?: boolean
  /** Race owns the Deformidade ability (p23): 2 chosen +2 perícia bonuses,
   *  one swappable for a poder da Tormenta. See `deformidade.ts`. */
  hasDeformidade?: boolean
}

// ─── de bestiary.ts ─────────────────────────────────────────
export type Monster = {
  id: string
  name: string
  nd: number
  tipo: MonsterTipo
  size: MonsterSize
  hp: number
  defesa: number
  forca: number
  destreza: number
  constituicao: number
  inteligencia: number
  sabedoria: number
  carisma: number
  fortitude: number
  reflexos: number
  vontade: number
  deslocamento: string
  attacks: readonly MonsterAttack[]
  specialAbilities: readonly string[]
  treasureXp: number
  bookPage: number
}

export type MonsterTipo =
  | 'humanoide'
  | 'animal'
  | 'monstro'
  | 'morto-vivo'
  | 'construto'
  | 'espirito'
  | 'planar'

// ─── de class-spellcasting.ts ───────────────────────────────
export type SpellcasterClass =
  | 'Arcanista'
  | 'Bardo'
  | 'Clérigo'
  | 'Druida'
  | 'Paladino'

// ─── de class-starting-kits.ts ──────────────────────────────
export type StartingKit = {
  baseItems: readonly string[]
  weapons: StartingWeaponGrant
  armor: StartingArmorChoice
  /** Granted only when the class is proficient in escudos. */
  shieldLeve: boolean
  /** Always `'4d6'` per Tabela 3-1. */
  tibarDice: string
  extras: readonly StartingExtra[]
}

// ─── de conditions.ts ───────────────────────────────────────
export type Condition = {
  id: ConditionId
  name: string
  description: string
  tags: ConditionTag[]
  /** When applying this condition to a target who already has it, the
   *  condition is *replaced* by `upgradesTo` (PDF p394 sidebar). */
  upgradesTo?: ConditionId
}

export type ConditionId = (typeof CONDITION_IDS)[number]

// ─── de divine-power-mechanics.ts ───────────────────────────
export type GrantedPowerOption = DivinePower & { description: string }

// ─── de origem-item-parse.ts ────────────────────────────────
/**
 * Structured view of `Origem.itensIniciais` (book p85-95 "Itens" lines).
 * The catalog stores the book's verbatim strings; several are CHOICES, not
 * fixed grants ("Arma marcial", "Um item estrangeiro (até T$ 100)",
 * "Estojo de disfarces OU gazua"). This parser classifies each entry so the
 * creation UI can render a picker instead of a dead text row.
 *
 * PURE logic — NO ORIGENS catalog. Split out of `./origem-item-grants` so the
 * frontend can call it against an origem from its fetched cache without
 * anchoring the ORIGENS table (project_front_decouple_catalog).
 *
 * Parsing conventions (encoded in the data strings):
 *  - literal 'Arma simples' / 'Arma marcial' / 'Arma marcial ou exótica'
 *    → weapon pick by category.
 *  - '(até T$ N)' → any-item pick with a price cap.
 *  - ' OU ' (uppercase) → one-of between fixed alternatives; lowercase 'ou'
 *    inside parentheses is descriptive text, never a split point.
 *  - '(escolha)' suffix → one-of over the comma/'ou'-separated list.
 *  - 'T$ <dice>' → starting-money bonus, not an item.
 *  - anything else → fixed grant.
 */
export type OrigemItemGrant =
  | { kind: 'fixed'; name: string }
  | {
      kind: 'weapon'
      categories: readonly ('weapon-simple' | 'weapon-martial' | 'weapon-exotic')[]
      label: string
    }
  | { kind: 'anyItem'; maxPrice: number; label: string }
  | { kind: 'oneOf'; options: readonly string[]; label: string }
  | { kind: 'money'; dice: string; label: string }

// ─── de origens.ts ──────────────────────────────────────────
export type Origem = {
  id: string
  name: string
  pericias: readonly string[]
  poderes: readonly string[]
  /** Always === poderes[poderes.length - 1] for non-GM-driven origens. */
  poderUnico: string
  poderChoiceCategory?: OrigemPoderChoiceCategory
  itensIniciais: readonly string[]
  bookPage: number
  /** Set on Amnésico — perícias / poderes are picked by the GM. */
  gmDriven?: boolean
  /** Always-applied power (Amnésico's Lembranças Graduais). */
  poderObrigatorio?: string
}

// ─── de power-activation.ts ─────────────────────────────────
export type ActivationAction =
  | 'padrao'
  | 'movimento'
  | 'livre'
  | 'reacao'
  | 'gratuita'
  | 'completa'
  | 'passivo'
  | 'varia'

export type ActivationKind =
  | 'passive'
  | 'triggered-passive'
  | 'stance'
  | 'instant'

export type ActivationSpec = {
  id: string
  name: string
  kind: ActivationKind
  action: ActivationAction
  pmCost: number | 'variavel'
  uses: ActivationUses
  /** Flag de postura exigida pra ativar/disparar ('furia', 'inspiracao'). */
  requiresFlag?: string
  /** Presente só em posturas/escaláveis (hand-authored). */
  scaling?: ActivationScaling
  /** Efeito duradouro concedido ao usar/disparar (hand-authored). */
  grant?: ActivationGrant
  bookPage: number
}

export type ActivationUses = null | 'cena' | 'rodada' | 'dia' | number

// ─── de proficiencies.ts ────────────────────────────────────
export type ProficiencyEntry = {
  category: ProficiencyCategory
  label: string
  granted: boolean
  sources: string[]
}

// ─── de racas.ts ────────────────────────────────────────────
export type Raca = {
  id: string
  name: string
  tier: 'comum' | 'extra'
  atributoMod: AtributoMod
  tamanho: Tamanho
  deslocamento: number
  visaoNoEscuro: boolean
  visaoNaPenumbra: boolean
  abilities: readonly RacaAbility[]
  /** Ascendência options (Qareen elemental, Golem fonte, Suraggel Aggelus/Sulfure). */
  ascendencias?: readonly string[]
  bookPage: number
}

// ─── de spell-catalog.ts ────────────────────────────────────
export type CatalogSpell = {
  id: string
  name: string
  circle: SpellCircle
  school: SpellSchool
  execution: SpellExecution
  range: SpellRange
  duration: SpellDuration
  /** Free-text duration when `duration === 'definida'`. */
  durationNote?: string
  saveType: SpellSaveType
  /** `null` when the magia offers no resistance roll at all. */
  resistance: SpellResistance | null
  components: SpellComponent[]
  classes: SpellClassName[]
  baseEffect: string
  augments: CatalogAugment[]
  bookPage: number
  /** Present only for self/ally buff spells — see {@link SpellBuff}. */
  buff?: SpellBuff
}

export type SpellClassName = 'Arcanista' | 'Bardo' | 'Clérigo' | 'Druida' | 'Paladino'

// ─── de tormenta.ts ─────────────────────────────────────────
export type TormentaPower = {
  id: TormentaPowerId
  name: string
  /**
   * Verbatim rule text lifted from PDF Cap 2 p135-137. Every power in
   * this catalog now carries a description so consumers can render
   * without a second lookup. See `feat(t20-data): 22 tormenta descs`.
   */
  description: string
  /** Number of OTHER Tormenta powers required to unlock this one. */
  requiresOtherPowers: number
  /** Specific prerequisite power, when stated. */
  requiresPower?: TormentaPowerId
  /** Book page anchor (135, 136 or 137). */
  bookPage: 135 | 136 | 137
}

/**
 * Poder da Tormenta catalog (book p127-128 + p135-137). Each is a
 * permanent voluntary mutation. Some require N other powers already
 * taken before they unlock.
 */
export type TormentaPowerId =
  | 'anatomia-insana'
  | 'antenas'
  | 'armamento-aberrante'
  | 'articulacoes-flexiveis'
  | 'asas-insetoides'
  | 'carapaca'
  | 'corpo-aberrante'
  | 'cuspir-enxame'
  | 'dentes-afiados'
  | 'desprezar-a-realidade'
  | 'empunhadura-rubra'
  | 'fome-de-mana'
  | 'larva-explosiva'
  | 'legiao-aberrante'
  | 'maos-membranosas'
  | 'membros-estendidos'
  | 'membros-extras'
  | 'mente-aberrante'
  | 'olhos-vermelhos'
  | 'pele-corrompida'
  | 'sangue-acido'
  | 'visco-rubro'


// ─── tipos transitivos ───────────────────────────────────

/**
 * Efeito PERSISTENTE que usar o poder concede (Fase 4) — vira um ActiveEffect
 * no backend (`POST /characters/:id/active-effects` com `{powerId}`).
 *
 *  - 'temp-hp'      → pool de PV temporários; o SERVIDOR calcula o total como
 *                     nível + atributo FINAL (`attribute`) via computeSheet.
 *  - 'active-effect'→ modifiers persistidos verbatim, escopo cena/dia.
 */
export type ActivationGrant =
  | { kind: 'temp-hp'; scope: 'scene'; attribute: AttributeKey }
  | { kind: 'active-effect'; scope: 'scene' | 'day'; modifiers: Modifier[] }

export type ActivationScaling = {
  basePm: number
  stepPm: number
  /** O que cada passo compra ("+1 no bônus de Fúria"). */
  stepLabel: string
  /**
   * Nível NA CLASSE que destrava o primeiro passo, e de quantos em quantos
   * níveis vem o próximo.
   *
   * DADO, não função: o catálogo é servido por HTTP e uma função não sobrevive
   * ao JSON — o front recebia `scaling` sem o método e derrubava a cena ao
   * abrir qualquer postura que escala. Leia com `maxStepsForLevel`.
   */
  firstStepLevel: number
  stepEveryLevels: number
}

export type AtributoMod =
  | { kind: 'fixed'; mods: Partial<Record<AttributeKey, number>> }
  | {
      kind: 'floating'
      count: number
      value: number
      exclude?: AttributeKey
      penalty?: { attribute: AttributeKey; value: number }
    }
  | {
      kind: 'subraca-gated'
      variants: Readonly<Record<string, Partial<Record<AttributeKey, number>>>>
    }

export type CatalogAugment = {
  pmCost: number
  kind: AugmentKind
  description: string
  requiresCircle?: SpellCircle
  classOnly?: 'arcanos' | 'divinos' | 'druidas'
}

export type ConditionTag = (typeof CONDITION_TAGS)[number]

/**
 * Energia canalizada — book uses Positiva / Negativa / Qualquer (book p97).
 * 'qualquer' = devoto picks one at character creation (irreversible).
 *
 * Note: T20 explicitly drops D&D-style 2-axis alinhamento. No alinhamento
 * field here. The clérigo's behavioural restrictions come from the deus's
 * doutrina prose, not a numeric grid.
 */
export type DeusEnergia = 'positiva' | 'negativa' | 'qualquer'

export type DivinePower = {
  deusId: string
  name: string
  action: DivinePowerAction
  /** PM fixo, ou 'variavel' quando concede magia (custo = base spell PM). */
  pmCost: number | 'variavel'
  uses: DivinePowerUses
  bookPage: number
}

export type GrantedPowerKind =
  | 'ataque'
  | 'defesa'
  | 'utilidade'
  | 'sentido'
  | 'social'
  | 'movimento'
  | 'magia'

export type MonsterAttack = {
  name: string
  attackBonus: number
  damage: string
  special?: string
}

export type MonsterSize =
  | 'minusculo'
  | 'pequeno'
  | 'medio'
  | 'grande'
  | 'enorme'
  | 'colossal'

export type OrigemPoderChoiceCategory = 'combate' | 'tormenta'

export type ProficiencyCategory = (typeof PROFICIENCY_CATEGORIES)[number]

export type RacaAbility = {
  name: string
  summary: string
}

export type RaceAbilityVariant = {
  id: string
  name: string
  description: string
  modifiers?: Modifier[]
}

/**
 * Structured buff a spell can apply as a scoped ActiveEffect. Only spells that
 * grant a machine-modelable, self/ally-targetable bonus carry one; the rest
 * stay free-text `baseEffect`. `defaultScope` maps the spell's duration onto
 * the ActiveEffect scopes the effect lifecycle understands (`scene`/`day`).
 */
export type SpellBuff = {
  defaultScope: 'scene' | 'day'
  modifiers: Modifier[]
  /**
   * Display-only mechanical facts (RD, immunities, senses, movement modes,
   * action economy) — shown as reference chips but not computed. Lets a spell
   * with no computable modifier still appear in the "Aplicar efeito" dialog,
   * and lets buffs carry non-numeric sub-effects their modifiers drop.
   */
  facts?: DisplayFact[]
}

export type StartingArmorChoice =
  /** L1 Arcanistas começam sem armadura (exceção explícita p140). */
  | 'nenhuma'
  /** Player picks among the three light options. */
  | 'leve-a-escolha'
  /** Proficiência em pesadas substitui leve por brunea. */
  | 'brunea'

export type StartingExtra = {
  /** Classe que concede o item extra. */
  source: string
  description: string
  /** Teto de preço do item, em Tibares. `null` = item sem teto explícito. */
  maxValueTibar: number | null
}

export type StartingWeaponGrant = 'simples' | 'simples+marcial'

export type Tamanho =
  | 'Minúsculo'
  | 'Pequeno'
  | 'Médio'
  | 'Grande'
  | 'Enorme'
  | 'Colossal'

// ─── tipos transitivos ───────────────────────────────────

export type DivinePowerAction =
  | 'padrao'
  | 'movimento'
  | 'livre'
  | 'reacao'
  | 'gratuita'
  | 'completa'
  | 'passivo'
  | 'varia'

/** Limite de uso. null = ilimitado (só limitado por PM). */
export type DivinePowerUses = null | 'cena' | 'rodada' | number

export const CONDITION_IDS = [
  'abalado',
  'agarrado',
  'alquebrado',
  'apavorado',
  'atordoado',
  'caido',
  'cego',
  'confuso',
  'debilitado',
  'desprevenido',
  'doente',
  'em-chamas',
  'enfeitiçado',
  'enjoado',
  'enredado',
  'envenenado',
  'esmorecido',
  'exausto',
  'fascinado',
  'fatigado',
  'fraco',
  'frustrado',
  'imovel',
  'inconsciente',
  'indefeso',
  'lento',
  'ofuscado',
  'paralisado',
  'pasmo',
  'petrificado',
  'sangrando',
  'sobrecarregado',
  'surdo',
  'surpreendido',
  'vulneravel',
] as const

/**
 * Lista de Condições (PDF book p394-395). Mechanical effects pinned from
 * the rulebook. T20 condition system rules:
 *
 *   - "Condições com os mesmos efeitos não se acumulam; aplique apenas
 *     o mais severo."
 *   - "A menos que especificado, condições terminam no fim da cena."
 *   - Some conditions carry a *tipo de efeito* (Medo, Mental, …) — used
 *     by other rules (immunities, recovery sources, etc.).
 *   - A handful of conditions "upgrade" if applied again: a fresh
 *     Abalado on an already-Abalado target becomes Apavorado, etc.
 *
 * Upgrade chains:
 *   Fraco → Debilitado → Inconsciente
 *   Frustrado → Esmorecido
 *   Fatigado → Exausto → Inconsciente
 *   Abalado → Apavorado
 */
export const CONDITION_TAGS = [
  'medo',
  'mental',
  'movimento',
  'metabolismo',
  'sentidos',
  'cansaco',
  'veneno',
  'metamorfose',
] as const

export const PROFICIENCY_CATEGORIES = [
  'armas-simples',
  'armas-marciais',
  'armas-exoticas',
  'armas-de-fogo',
  'armaduras-leves',
  'armaduras-pesadas',
  'escudos',
] as const
