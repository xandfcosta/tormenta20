/**
 * O vocabulário de ITEM e MODIFICADOR — movido do `t20-data` quando ele foi
 * aposentado (ALE-109).
 *
 * Por que NÃO vem do gerador de tipos da fronteira: o `ModifierTarget` é uma
 * UNIÃO DISCRIMINADA — `name` só existe quando `k` é `expertise`, `attribute` só
 * quando é `expertiseByAttribute` — e a struct Go é plana, com todos os campos
 * sempre presentes. Gerar daqui destruiria a discriminação e, com ela, as
 * switches exaustivas que garantem que um alvo novo não passe sem rótulo. É a
 * mesma razão do `AttributeKey` e do `BonusType`: o Go não expressa união, e
 * aqui a união é quem carrega a regra.
 */
import type { AttributeKey } from './attribute-keys'
import type { BonusType, EquippedSlot } from './bonus-types'
import type { DisplayFact } from './display-facts'
import type { ExpertiseName } from './expertise-names'
import type { SpellSchool } from './spell-types'

export type ItemCategory =
  | 'weapon-simple'
  | 'weapon-martial'
  | 'weapon-exotic'
  | 'weapon-firearm'
  | 'armor-light'
  | 'armor-heavy'
  | 'shield'
  | 'apparel'
  | 'consumable'
  | 'meal'
  | 'catalyst'
  | 'improvement'
  | 'material'
  | 'animal'
  | 'vehicle'

/**
 * Coarse grouping used to gate which catalog improvements / materials may
 * be attached to a given base item.
 *  - 'weapon'     — any weapon-* category.
 *  - 'armor'      — armor-light / armor-heavy.
 *  - 'shield'     — shields.
 *  - 'apparel'    — clothing, esotéricos, ferramentas.
 *  - 'any'        — applies to any base item.
 */
export type ItemFamily = 'weapon' | 'armor' | 'shield' | 'apparel' | 'any'

export type WeaponHand = 'light' | 'one' | 'two'

export type WeaponPurpose = 'melee' | 'thrown' | 'ranged'

export type WeaponTrait =
  | 'agil'
  | 'alongada'
  | 'desbalanceada'
  | 'dupla'
  | 'versatil'
  | 'adaptavel'

export type DamageType =
  | 'corte'
  | 'impacto'
  | 'perfuracao'
  | 'corte-perfuracao'

export type WeaponRange = 'curto' | 'medio' | 'longo'

export type WeaponStats = {
  damage: string
  critRange: number
  critMult: number
  type: DamageType
  hand: WeaponHand
  purpose: WeaponPurpose
  range?: WeaponRange
  traits: WeaponTrait[]
  /** Weapon that inherently lets you use Destreza instead of Força on the
   *  attack roll (Adaga, p145) — no power required. Attack only, not damage.
   *  (The "Acuidade com Arma" power grants Des on attack AND damage for light/
   *  thrown/ágil weapons; that's derived from the power + these stats.) */
  finesse?: boolean
}

export type ArmorStats = {
  defense: number
  penalty: number
  heavy: boolean
}

export type ShieldStats = {
  defense: number
  penalty: number
  heavy: boolean
}

/**
 * Bonus categories used to enforce non-stacking rules from the rulebook.
 * Two modifiers sharing the same `bonusType` and same `target` keep only
 * the highest. `untyped` always stacks.
 *
 * Mapped to T20 conventions:
 *  - 'armor'    — body-armor Defense bonus (don't stack with itself).
 *  - 'shield'   — shield Defense bonus. Distinct from 'armor' because the
 *                 book stacks armadura + escudo (p226 sidebar) — a shared
 *                 bucket made resolveStack silently drop the shield.
 *  - 'item'     — generic item bonus (most apparel +1 expertise).
 *  - 'training' — training bonus (level-based).
 *  - 'morale'   — temporary morale (e.g. Bardo Inspiração).
 *  - 'enhancement' — magical enhancement (melhorias).
 *  - 'condition' — status-condition penalty (Abalado, Vulnerável…). Book p394:
 *                  "condições com os mesmos efeitos não se acumulam; aplique
 *                  apenas o mais severo" — so condition mods sharing a target
 *                  keep the highest-abs (like any typed bucket) instead of
 *                  summing, while still stacking with item/other bonuses.
 *  - 'untyped'  — stacks freely (situational bonuses, alchemicals).
 */
export type ModifierTarget =
  | { k: 'expertise'; name: ExpertiseName }
  | { k: 'expertiseAll' }
  | { k: 'expertiseRemovePenalty'; name: ExpertiseName }
  | { k: 'expertiseByAttribute'; attribute: AttributeKey }
  | { k: 'attribute'; name: AttributeKey }
  | { k: 'defense'; scope?: 'all' | 'melee' | 'ranged' }
  | { k: 'defenseDexCap' }
  | { k: 'resistance' }
  | { k: 'fearResistance' }
  | { k: 'attack'; scope: 'this' | 'all' }
  | { k: 'damage'; scope: 'this' | 'all' }
  | { k: 'critRange' }
  | { k: 'critMult' }
  | { k: 'pmLimit' }
  | { k: 'pmCost' }
  /** Redução de dano concedida por modificador (Petrificado, p394). */
  | { k: 'damageReduction' }
  /**
   * Catalisador — one-shot pmCost discount on the next magia of the
   * given school. Consumed on cast; the item consumer creates a
   * scene-scoped ActiveEffect that the cast engine looks up + deletes.
   */
  | { k: 'catalyst'; school: SpellSchool }
  | { k: 'spellDC' }
  | { k: 'inventorySlots' }
  | { k: 'displacement' }
  | { k: 'flySpeed' }
  | { k: 'armorPenalty' }
  | { k: 'armorPenaltyExpertises' }
  | { k: 'tempHp' }
  | { k: 'tempMp' }
  /**
   * Permanent max-pool grants (PV/PM total), e.g. race traits (Anão Duro como
   * Pedra) and powers ("+1 PM por nível", "soma Sabedoria no PM total"). The
   * amount is scaled by `Modifier.scale` and folded into vitals — see
   * `vital-grants.ts`. Distinct from `tempHp`/`tempMp` (expiring pools).
   */
  | { k: 'maxPv' }
  | { k: 'maxPm' }
  | { k: 'maneuver'; name: 'derrubar' | 'desarmar' | 'quebrar' | 'agarrar' | 'empurrar' }
  | { k: 'flag'; name: ItemFlag }

export type ItemFlag =
  | 'lethal-unarmed'
  | 'cannot-apply-dex-to-defense'
  | 'fatigue-on-sleep'
  | 'reach-extends'
  /** Emitted by every armor-heavy piece while vested — read by flagOff
   *  conditions like Pele de Ferro ("se não estiver usando armadura pesada"). */
  | 'armadura-pesada'
  /** Falha AUTOMÁTICA em Reflexos — o Indefeso e tudo que o livro define como
   *  indefeso (p394). Booleano porque não é um número: a ficha mostra "falha
   *  automática" na linha em vez de um total. */
  | 'auto-fail-reflexos'

export type ModifierCondition =
  | { c: 'always' }
  | { c: 'wielded' }
  | { c: 'vested' }
  | { c: 'terrain'; type: string }
  | { c: 'against'; trait: string }
  | { c: 'context'; note: string }
  | { c: 'flagOn'; flag: string; label: string }
  /** Auto-evaluated: applies only while the flag is ABSENT (e.g. Pele de
   *  Ferro's "+4 Defesa se não estiver usando armadura pesada"). */
  | { c: 'flagOff'; flag: string; label: string }

/**
 * How a `maxPv`/`maxPm` modifier's `amount` scales with the character. Only
 * interpreted by the vitals collector (`vital-grants.ts`); the item engine
 * ignores it. Omitted `scale` ⇒ `{ per: 'flat' }`.
 *  - flat        — amount as-is (e.g. Anão "+3 PV no 1º nível").
 *  - level       — amount × nível do personagem ("+1 PM por nível").
 *  - levelStep   — amount × ⌊/⌉(nível / step): floor for "a cada dois níveis"
 *                  (Vontade de Ferro), ceil for "a cada nível ímpar" (Bênção
 *                  do Mana).
 *  - attribute   — amount × total do atributo ("soma Sabedoria no PM total").
 */
export type VitalScale =
  | { per: 'flat' }
  | { per: 'level' }
  | { per: 'levelStep'; step: number; round: 'down' | 'up' }
  | { per: 'attribute'; attribute: AttributeKey }

export type Modifier = {
  target: ModifierTarget
  amount: number
  bonusType: BonusType
  condition?: ModifierCondition
  note?: string
  /** Scaling for `maxPv`/`maxPm` targets. Ignored for every other target. */
  scale?: VitalScale
}

/**
 * Lifetime of an effect produced by consuming a single-use item.
 *  - 'instant'  — direct numeric mutation on consume (HP/MP). No ActiveEffect row.
 *  - 'scene'    — modifiers active until end of current scene/encounter.
 *  - 'day'      — modifiers active until next long rest. Pratos especiais use this
 *                 and are limited to 1 active per character per day.
 */
export type ConsumableScope = 'instant' | 'scene' | 'day'

/**
 * Direct numeric effect of an instant consumable (potion, healing salve, etc.).
 * Engine clients turn these into vitals patches; they don't produce modifiers.
 */
export type InstantEffect = {
  hp?: { dice: string; bonus?: number }
  mp?: { dice: string; bonus?: number }
}

export type ConsumableSpec = {
  scope: ConsumableScope
  /** True when only one instance of this item type may be active per day. */
  oncePerDay?: boolean
  /** Modifiers granted while the resulting ActiveEffect is alive. */
  modifiers?: Modifier[]
  /** Direct vitals mutation applied at consumption time. */
  instant?: InstantEffect
}

export type CatalogItem = {
  id: string
  name: string
  category: ItemCategory
  price: number
  slots: number
  /** how the item must be carried to grant its modifiers */
  equip: EquippedSlot | 'either'
  /** hands occupied when wielded ('one' or 'two'); undefined when not wieldable */
  hands?: 1 | 2
  weapon?: WeaponStats
  armor?: ArmorStats
  shield?: ShieldStats
  modifiers: Modifier[]
  /**
   * Display-only mechanical facts (RD bypass, immunities, senses) the engine
   * can't compute — shown as reference chips. Replaces the old `amount: 0`
   * fake-modifier hack (see adamante). Kept separate from `modifiers`, which
   * stays strictly numeric.
   */
  displayFacts?: DisplayFact[]
  /** Present when the item is single-use. Consuming decrements quantity. */
  consumable?: ConsumableSpec
  /**
   * For catalog entries that overlay onto a base item — improvements
   * (melhorias) and materiais especiais. Lists which item families this
   * overlay can be attached to.
   */
  appliesTo?: ItemFamily[]
}

// ─── Efeitos resolvidos ──────────────────────────────────────────────
//
// A forma EM MEMÓRIA, que não é a mesma do fio: aqui `flags` é um `Set`, e o
// motor serializa um array ordenado. O contrato de fio é o `ItemEffects` de
// `engine-types.ts`, gerado da struct Go; o wrapper do WASM converte entre os
// dois. Manter os dois nomes é intencional — colapsá-los faria o app tratar um
// array como Set (ALE-109).

export type Contribution = {
  source: string
  bonusType: BonusType
  amount: number
  /** O `note` do próprio modificador — o PORQUÊ do número ("desbalanceada: −2
   *  em ataque"). Os diálogos de composição o mostram sob a fonte, para a linha
   *  se explicar sozinha. */
  note?: string
}

export type AggregatedStat = {
  total: number
  contributions: Contribution[]
}

export type ConditionalEffect = {
  source: string
  bonusType: BonusType
  amount: number
  /** Nota legível da condição ("terreno compatível", "ao usar a manobra X"). */
  note: string
  target: ModifierTarget
  /** Presente quando a condição é um `flagOn`; a UI agrupa num só interruptor
   *  todos os condicionais que compartilham a mesma `flag`. */
  flag?: string
}

export type ItemEffects = {
  /** Modificadores numéricos agregados, por identidade de alvo. */
  byTarget: Record<string, AggregatedStat>
  /** Flags ativas ("lethal-unarmed", "fatigue-on-sleep", …). */
  flags: Set<string>
  /** Modificadores que dependem de escolha ou situação. */
  conditional: ConditionalEffect[]
}

export type ActiveItem = {
  source: string
  /** Estado de uso declarado pelo jogador. */
  equipped: EquippedSlot | null
  modifiers: Modifier[]
}
