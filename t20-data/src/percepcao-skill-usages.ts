/**
 * Perícia Percepção (SAB, aberta, sem penalidade de armadura) — 2 usos.
 *
 * PDF Cap 2 Perícias — Percepção (livro p122). Header verbatim:
 * "PERCEPÇÃO — SAB".
 * Intro verbatim: "Você nota coisas usando seus sentidos."
 *
 * Usos:
 *  - Observar (p122) — CDs 15 (difícil) a 30 (quase invisível); CD ler
 *    lábios 20; para pessoa/item escondido, CD = resultado do teste de
 *    Furtividade OU Ladinagem-Ocultar de quem escondeu.
 *  - Ouvir (p122) — CD 0 conversa casual (auto sem penalidade); CD 15
 *    sussurro; +10 na CD do outro lado de porta; -10 no teste dormindo
 *    (sucesso acorda); perceber criatura invisível CD 20 OU +10 sobre
 *    Furtividade dela (o maior).
 *
 * Nota Tabela 2-1 p115: NÃO é somente treinada; NÃO sofre penalidade de
 * armadura. Cross-ref: [[furtividade-skill-usages]] (Esconder-se) e
 * [[ladinagem-skill-usages]] (Ocultar).
 */

// ─── Types ────────────────────────────────────────────────────────────
export type PercepcaoUsageKind = 'observar' | 'ouvir'

/** Alvo de observação (p122). */
export type ObservarTarget =
  | { kind: 'dificil' }
  | { kind: 'quase-invisivel' }
  | { kind: 'ler-labios' }
  | { kind: 'escondido-por-furtividade'; furtividadeRoll: number }
  | { kind: 'escondido-por-ladinagem-ocultar'; ladinagemRoll: number }

/** Fonte sonora (p122). */
export type OuvirSource =
  | { kind: 'conversa-casual' }
  | { kind: 'sussurro' }
  | { kind: 'criatura-invisivel'; furtividadeRoll: number }

type UsageCommon = {
  id: PercepcaoUsageKind
  name: string
  effect: string
  bookPage: 122
}

export type PercepcaoObservar = UsageCommon & {
  kind: 'observar'
  cdDificil: 15
  cdQuaseInvisivel: 30
  cdLerLabios: 20
  /** CD contra pessoa/item escondido = resultado do teste de Furtividade ou Ladinagem-Ocultar. */
  cdHiddenEqualsHiderRoll: true
}

export type PercepcaoOuvir = UsageCommon & {
  kind: 'ouvir'
  cdConversaCasual: 0
  cdSussurro: 15
  throughDoorCdIncrease: 10
  sleepingPenalty: -10
  /** Sucesso enquanto dormindo acorda o personagem. */
  sleepingSuccessWakes: true
  invisibleCreatureBaseCd: 20
  invisibleCreatureFurtividadeBonus: 10
  /** Penalidades por lutar sem ver o inimigo permanecem mesmo com sucesso. */
  keepsBlindFightPenalties: true
}

export type PercepcaoUsage = PercepcaoObservar | PercepcaoOuvir

// ─── Constantes ──────────────────────────────────────────────────────
// Observar (p122 verbatim)
export const OBSERVAR_CD_DIFICIL = 15
export const OBSERVAR_CD_QUASE_INVISIVEL = 30
export const OBSERVAR_LER_LABIOS_CD = 20

// Ouvir (p122 verbatim)
export const OUVIR_CD_CONVERSA_CASUAL = 0
export const OUVIR_CD_SUSSURRO = 15
export const OUVIR_THROUGH_DOOR_CD_INCREASE = 10
export const OUVIR_SLEEPING_PENALTY = -10
export const OUVIR_CRIATURA_INVISIVEL_BASE_CD = 20
export const OUVIR_CRIATURA_FURTIVIDADE_BONUS = 10

// Flags Tabela 2-1 p115
export const PERCEPCAO_TRAINED_ONLY = false
export const PERCEPCAO_ARMOR_PENALTY = false

// ─── Catálogo ─────────────────────────────────────────────────────────
export const PERCEPCAO_USAGES: readonly PercepcaoUsage[] = Object.freeze([
  {
    id: 'observar',
    kind: 'observar',
    name: 'Observar',
    cdDificil: 15,
    cdQuaseInvisivel: 30,
    cdLerLabios: 20,
    cdHiddenEqualsHiderRoll: true,
    effect:
      'Vê coisas discretas/escondidas; CD 15 difícil, CD 30 quase invisível; CD ler lábios 20; escondido = CD igual ao roll de Furtividade/Ladinagem.',
    bookPage: 122,
  },
  {
    id: 'ouvir',
    kind: 'ouvir',
    name: 'Ouvir',
    cdConversaCasual: 0,
    cdSussurro: 15,
    throughDoorCdIncrease: 10,
    sleepingPenalty: -10,
    sleepingSuccessWakes: true,
    invisibleCreatureBaseCd: 20,
    invisibleCreatureFurtividadeBonus: 10,
    keepsBlindFightPenalties: true,
    effect:
      'Escuta sons; conversa casual auto, sussurro CD 15; +10 CD porta; -10 dormindo (sucesso acorda); criatura invisível CD 20 ou Furt+10 (o maior).',
    bookPage: 122,
  },
])

const usagesByKind = new Map<PercepcaoUsageKind, PercepcaoUsage>(
  PERCEPCAO_USAGES.map((u) => [u.kind, u]),
)

export function percepcaoUsageByKind(kind: PercepcaoUsageKind): PercepcaoUsage {
  const usage = usagesByKind.get(kind)
  if (!usage) {
    throw new Error(`percepcaoUsageByKind: unknown kind ${kind}`)
  }
  return usage
}

// ─── Helpers — Observar ──────────────────────────────────────────────
/**
 * CD de Observar conforme alvo:
 *  - `dificil` → 15
 *  - `quase-invisivel` → 30
 *  - `ler-labios` → 20
 *  - `escondido-por-furtividade` → resultado do teste de Furtividade
 *  - `escondido-por-ladinagem-ocultar` → resultado do teste de Ladinagem
 */
export function observarCd(target: ObservarTarget): number {
  switch (target.kind) {
    case 'dificil':
      return OBSERVAR_CD_DIFICIL
    case 'quase-invisivel':
      return OBSERVAR_CD_QUASE_INVISIVEL
    case 'ler-labios':
      return OBSERVAR_LER_LABIOS_CD
    case 'escondido-por-furtividade':
      return target.furtividadeRoll
    case 'escondido-por-ladinagem-ocultar':
      return target.ladinagemRoll
  }
}

// ─── Helpers — Ouvir ─────────────────────────────────────────────────
/** Modificador na CD por barreira (porta fechada +10, ar livre 0). */
export function ouvirThroughDoorModifier(throughDoor: boolean): number {
  return throughDoor ? OUVIR_THROUGH_DOOR_CD_INCREASE : 0
}

/** Penalidade no teste se o personagem estiver dormindo. */
export function ouvirSleepingPenalty(isSleeping: boolean): number {
  return isSleeping ? OUVIR_SLEEPING_PENALTY : 0
}

/**
 * CD para perceber criatura invisível/não vista:
 * CD 20 OU +10 sobre o teste de Furtividade da criatura, o que for maior.
 */
export function ouvirCriaturaInvisivelCd(criaturaFurtividadeRoll: number): number {
  return Math.max(
    OUVIR_CRIATURA_INVISIVEL_BASE_CD,
    criaturaFurtividadeRoll + OUVIR_CRIATURA_FURTIVIDADE_BONUS,
  )
}

/** CD de Ouvir por fonte + porta fechada. */
export function ouvirCd(source: OuvirSource, throughDoor: boolean = false): number {
  const base = ouvirSourceBaseCd(source)
  return base + ouvirThroughDoorModifier(throughDoor)
}

function ouvirSourceBaseCd(source: OuvirSource): number {
  switch (source.kind) {
    case 'conversa-casual':
      return OUVIR_CD_CONVERSA_CASUAL
    case 'sussurro':
      return OUVIR_CD_SUSSURRO
    case 'criatura-invisivel':
      return ouvirCriaturaInvisivelCd(source.furtividadeRoll)
  }
}
