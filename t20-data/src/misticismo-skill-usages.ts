/**
 * Perícia Misticismo (INT, treinada, sem penalidade de armadura) — 6 usos.
 *
 * PDF Cap 2 Perícias — Misticismo (livro p121). Header verbatim:
 * "MISTICISMO — INT · TREINADA".
 * Intro verbatim: "Esta perícia envolve o conhecimento de magias, itens
 * mágicos e fenômenos sobrenaturais."
 *
 * Usos:
 *  - Detectar Magia (p121) — CD 15, ação completa, alcance curto;
 *    aura por círculo/categoria; barreira -5 (madeira/pedra) / -10
 *    (ferro/chumbo).
 *  - Identificar Criatura (p121) — CD 15 + ND; ação completa; +1
 *    característica por 5 acima da CD; falha 5+ = informação errada.
 *  - Identificar Item Mágico (p121) — CD 20/25/30 por categoria;
 *    1 hora; -10 opcional para reduzir a ação completa.
 *  - Identificar Magia (p121) — CD 15 + custo em PM da magia; reação.
 *  - Informação (p121) — sem teste (simples); CD 20 (complexa); CD 30
 *    (mistério/enigma).
 *  - Lançar Magia de Armadura (p121) — CD 20 + custo em PM; sofre
 *    penalidade de armadura; falha consome PM mesmo assim; apenas
 *    magia arcana.
 *
 * Nota Tabela 2-1 p115: APENAS TREINADA; NÃO sofre penalidade de armadura
 * (exceto pelo uso Lançar Magia de Armadura, que a aplica por texto
 * explícito). Cross-ref: [[religiao-skill-usages]] delega Identificar
 * Criatura/Item Mágico de origem divina para cá.
 */

import type { SpellCircle } from './spells'

// ─── Types ────────────────────────────────────────────────────────────
export type MisticismoUsageKind =
  | 'detectar-magia'
  | 'identificar-criatura'
  | 'identificar-item-magico'
  | 'identificar-magia'
  | 'informacao'
  | 'lancar-magia-de-armadura'

/** Intensidade da aura mágica (p121). */
export type AuraIntensity = 'tenue' | 'moderada' | 'poderosa' | 'avassaladora'

/** Círculo de magia efetivo para aura (1-5; truque não gera aura pela tabela). */
export type MisticismoSpellCircle = Exclude<SpellCircle, 0>

/** Categoria de item mágico (p121, Cap 6). */
export type ItemMagicoCategory = 'menor' | 'medio' | 'maior'

/** Origem especial (deus menor ou artefato) para aura avassaladora. */
export type SpecialMagicSource = 'deus-menor' | 'artefato'

/** Barreira entre observador e aura (p121). */
export type DetectarMagiaBarrier =
  | 'nenhuma'
  | 'madeira-ou-pedra'
  | 'ferro-ou-chumbo'

/** Dificuldade da pergunta em Informação (p121). */
export type InformacaoDifficulty = 'simples' | 'complexa' | 'misterio-ou-enigma'

type UsageCommon = {
  id: MisticismoUsageKind
  name: string
  effect: string
  bookPage: 121
}

export type MisticismoDetectarMagia = UsageCommon & {
  kind: 'detectar-magia'
  action: 'completa'
  dc: 15
  range: 'curto'
  barreiraMadeiraPedraPenalty: -5
  barreiraFerroChumboPenalty: -10
}

export type MisticismoIdentificarCriatura = UsageCommon & {
  kind: 'identificar-criatura'
  action: 'completa'
  cdBase: 15
  cdIncludesNd: true
  extraTraitPerMargin: 5
  wrongInfoMargin: 5
  requiresSeeingCreature: true
}

export type MisticismoIdentificarItemMagico = UsageCommon & {
  kind: 'identificar-item-magico'
  action: '1-hora'
  cdMenor: 20
  cdMedio: 25
  cdMaior: 30
  rushedPenalty: -10
  rushedAction: 'completa'
}

export type MisticismoIdentificarMagia = UsageCommon & {
  kind: 'identificar-magia'
  action: 'reacao'
  cdBase: 15
  cdIncludesSpellPMCost: true
}

export type MisticismoInformacao = UsageCommon & {
  kind: 'informacao'
  cdComplexa: 20
  cdMisterio: 30
  /** Perguntas simples não exigem teste. */
  simplesRequiresNoTest: true
}

export type MisticismoLancarMagiaArmadura = UsageCommon & {
  kind: 'lancar-magia-de-armadura'
  cdBase: 20
  cdIncludesSpellPMCost: true
  armorPenaltyApplies: true
  /** Falha consome PM mesmo assim. */
  failureStillConsumesPM: true
  /** Apenas para magias arcanas. */
  arcaneOnly: true
}

export type MisticismoUsage =
  | MisticismoDetectarMagia
  | MisticismoIdentificarCriatura
  | MisticismoIdentificarItemMagico
  | MisticismoIdentificarMagia
  | MisticismoInformacao
  | MisticismoLancarMagiaArmadura

// ─── Constantes ──────────────────────────────────────────────────────
// Detectar Magia (p121 verbatim)
export const DETECTAR_MAGIA_CD = 15
export const DETECTAR_MAGIA_BARREIRA_MADEIRA_PEDRA_PENALTY = -5
export const DETECTAR_MAGIA_BARREIRA_FERRO_CHUMBO_PENALTY = -10

/** Aura por círculo de magia (verbatim p121). */
export const AURA_BY_SPELL_CIRCLE: Readonly<Record<MisticismoSpellCircle, AuraIntensity>> =
  Object.freeze({
    1: 'tenue',
    2: 'tenue',
    3: 'moderada',
    4: 'moderada',
    5: 'poderosa',
  })

/** Aura por categoria de item mágico (verbatim p121). */
export const AURA_BY_ITEM_MAGICO_CATEGORY: Readonly<
  Record<ItemMagicoCategory, AuraIntensity>
> = Object.freeze({
  menor: 'tenue',
  medio: 'moderada',
  maior: 'poderosa',
})

// Identificar Criatura (p121 verbatim)
export const IDENTIFICAR_CRIATURA_CD_BASE = 15
export const IDENTIFICAR_CRIATURA_EXTRA_TRAIT_MARGIN = 5
export const IDENTIFICAR_CRIATURA_WRONG_INFO_MARGIN = 5

// Identificar Item Mágico (p121 verbatim)
export const IDENTIFICAR_ITEM_MAGICO_CD_MENOR = 20
export const IDENTIFICAR_ITEM_MAGICO_CD_MEDIO = 25
export const IDENTIFICAR_ITEM_MAGICO_CD_MAIOR = 30
export const IDENTIFICAR_ITEM_MAGICO_RUSHED_PENALTY = -10

// Identificar Magia (p121 verbatim)
export const IDENTIFICAR_MAGIA_CD_BASE = 15

// Informação (p121 verbatim)
export const INFORMACAO_CD_COMPLEXA = 20
export const INFORMACAO_CD_MISTERIO = 30

// Lançar Magia de Armadura (p121 verbatim)
export const LANCAR_MAGIA_ARMADURA_CD_BASE = 20

// Flags Tabela 2-1 p115
export const MISTICISMO_TRAINED_ONLY = true
export const MISTICISMO_ARMOR_PENALTY = false

// ─── Catálogo ─────────────────────────────────────────────────────────
export const MISTICISMO_USAGES: readonly MisticismoUsage[] = Object.freeze([
  {
    id: 'detectar-magia',
    kind: 'detectar-magia',
    name: 'Detectar Magia',
    action: 'completa',
    dc: 15,
    range: 'curto',
    barreiraMadeiraPedraPenalty: -5,
    barreiraFerroChumboPenalty: -10,
    effect:
      'Ação completa; CD 15; alcance curto; detecta presença/intensidade de auras; barreira -5 madeira/pedra ou -10 ferro/chumbo.',
    bookPage: 121,
  },
  {
    id: 'identificar-criatura',
    kind: 'identificar-criatura',
    name: 'Identificar Criatura',
    action: 'completa',
    cdBase: 15,
    cdIncludesNd: true,
    extraTraitPerMargin: 5,
    wrongInfoMargin: 5,
    requiresSeeingCreature: true,
    effect:
      'Ação completa; CD 15 + ND; +1 característica por 5 acima da CD; falha 5+ = informação errada; precisa ver a criatura.',
    bookPage: 121,
  },
  {
    id: 'identificar-item-magico',
    kind: 'identificar-item-magico',
    name: 'Identificar Item Mágico',
    action: '1-hora',
    cdMenor: 20,
    cdMedio: 25,
    cdMaior: 30,
    rushedPenalty: -10,
    rushedAction: 'completa',
    effect:
      '1 hora; CD 20/25/30 (menor/médio/maior); opcional -10 para reduzir a ação completa.',
    bookPage: 121,
  },
  {
    id: 'identificar-magia',
    kind: 'identificar-magia',
    name: 'Identificar Magia',
    action: 'reacao',
    cdBase: 15,
    cdIncludesSpellPMCost: true,
    effect:
      'Reação; CD 15 + custo em PM da magia; identifica magia por gestos/palavras.',
    bookPage: 121,
  },
  {
    id: 'informacao',
    kind: 'informacao',
    name: 'Informação',
    cdComplexa: 20,
    cdMisterio: 30,
    simplesRequiresNoTest: true,
    effect:
      'Perguntas simples sem teste; complexas CD 20; mistérios/enigmas CD 30.',
    bookPage: 121,
  },
  {
    id: 'lancar-magia-de-armadura',
    kind: 'lancar-magia-de-armadura',
    name: 'Lançar Magia de Armadura',
    cdBase: 20,
    cdIncludesSpellPMCost: true,
    armorPenaltyApplies: true,
    failureStillConsumesPM: true,
    arcaneOnly: true,
    effect:
      'Lançar magia arcana com armadura; CD 20 + custo em PM; sofre penalidade de armadura; falha consome PM mesmo assim.',
    bookPage: 121,
  },
])

const usagesByKind = new Map<MisticismoUsageKind, MisticismoUsage>(
  MISTICISMO_USAGES.map((u) => [u.kind, u]),
)

export function misticismoUsageByKind(
  kind: MisticismoUsageKind,
): MisticismoUsage {
  const usage = usagesByKind.get(kind)
  if (!usage) {
    throw new Error(`misticismoUsageByKind: unknown kind ${kind}`)
  }
  return usage
}

// ─── Helpers — Detectar Magia ────────────────────────────────────────
/** Penalidade no teste por barreira entre observador e aura. */
export function detectarMagiaBarrierPenalty(
  barrier: DetectarMagiaBarrier,
): number {
  switch (barrier) {
    case 'nenhuma':
      return 0
    case 'madeira-ou-pedra':
      return DETECTAR_MAGIA_BARREIRA_MADEIRA_PEDRA_PENALTY
    case 'ferro-ou-chumbo':
      return DETECTAR_MAGIA_BARREIRA_FERRO_CHUMBO_PENALTY
  }
}

/** Aura por círculo de magia (1-5). */
export function auraForSpellCircle(circle: MisticismoSpellCircle): AuraIntensity {
  return AURA_BY_SPELL_CIRCLE[circle]
}

/** Aura por categoria de item mágico. */
export function auraForItemMagicoCategory(
  category: ItemMagicoCategory,
): AuraIntensity {
  return AURA_BY_ITEM_MAGICO_CATEGORY[category]
}

/** Aura de fontes especiais (magia de deus menor ou artefato). */
export function auraForSpecialSource(_source: SpecialMagicSource): AuraIntensity {
  return 'avassaladora'
}

// ─── Helpers — Identificar Criatura ──────────────────────────────────
/** CD para Identificar Criatura de ND informado. */
export function misticismoIdentificarCriaturaCd(nd: number): number {
  return IDENTIFICAR_CRIATURA_CD_BASE + nd
}

/**
 * Quantas características são lembradas:
 *  - roll < CD → 0.
 *  - roll ≥ CD → 1 + floor((roll - CD) / 5).
 */
export function identificarCriaturaTraitsRecalled(
  rollResult: number,
  cd: number,
): number {
  if (rollResult < cd) return 0
  return 1 + Math.floor((rollResult - cd) / IDENTIFICAR_CRIATURA_EXTRA_TRAIT_MARGIN)
}

export type IdentificarCriaturaOutcome =
  | 'wrong-info'
  | 'failed'
  | 'traits-recalled'

/**
 * Resolve Identificar Criatura:
 *  - roll ≥ CD → traits-recalled.
 *  - falha por < 5 → failed.
 *  - falha por ≥ 5 → wrong-info.
 */
export function identificarCriaturaOutcome(
  rollResult: number,
  cd: number,
): IdentificarCriaturaOutcome {
  const delta = rollResult - cd
  if (delta >= 0) return 'traits-recalled'
  if (Math.abs(delta) >= IDENTIFICAR_CRIATURA_WRONG_INFO_MARGIN) return 'wrong-info'
  return 'failed'
}

// ─── Helpers — Identificar Item Mágico ──────────────────────────────
/** CD por categoria de item mágico (menor 20 / médio 25 / maior 30). */
export function identificarItemMagicoCd(category: ItemMagicoCategory): number {
  switch (category) {
    case 'menor':
      return IDENTIFICAR_ITEM_MAGICO_CD_MENOR
    case 'medio':
      return IDENTIFICAR_ITEM_MAGICO_CD_MEDIO
    case 'maior':
      return IDENTIFICAR_ITEM_MAGICO_CD_MAIOR
  }
}

/** Penalidade -10 se optar por reduzir 1h para ação completa. */
export function identificarItemMagicoRushedPenalty(rushed: boolean): number {
  return rushed ? IDENTIFICAR_ITEM_MAGICO_RUSHED_PENALTY : 0
}

// ─── Helpers — Identificar Magia ────────────────────────────────────
/** CD = 15 + custo em PM da magia observada. */
export function identificarMagiaCd(spellPMCost: number): number {
  return IDENTIFICAR_MAGIA_CD_BASE + spellPMCost
}

// ─── Helpers — Informação ───────────────────────────────────────────
/**
 * CD por dificuldade da pergunta:
 *  - simples → null (não exige teste)
 *  - complexa → 20
 *  - mistério/enigma → 30
 */
export function informacaoCd(difficulty: InformacaoDifficulty): number | null {
  switch (difficulty) {
    case 'simples':
      return null
    case 'complexa':
      return INFORMACAO_CD_COMPLEXA
    case 'misterio-ou-enigma':
      return INFORMACAO_CD_MISTERIO
  }
}

// ─── Helpers — Lançar Magia de Armadura ─────────────────────────────
/** CD = 20 + custo em PM da magia arcana lançada com armadura. */
export function lancarMagiaArmaduraCd(spellPMCost: number): number {
  return LANCAR_MAGIA_ARMADURA_CD_BASE + spellPMCost
}

export type LancarMagiaArmaduraOutcome = 'success' | 'failed-still-consumes-pm'

/** Sucesso = magia funciona. Falha = não funciona MAS ainda consome PM. */
export function lancarMagiaArmaduraOutcome(
  rollResult: number,
  cd: number,
): LancarMagiaArmaduraOutcome {
  return rollResult >= cd ? 'success' : 'failed-still-consumes-pm'
}
