/**
 * Ability duration lifecycle.
 *
 * PDF Cap 5 Jogando p227 — seção "Duração" + subseções:
 *  - "Descarregar" (dormant-until-event)
 *  - "Encerrando suas habilidades" (owner-terminated)
 *  - "Morte e Duração"
 *  - "Duração e Áreas"
 *  - Sustentar (mesma página)
 *
 * Tipos de duração no livro: Instantânea, Cena, Sustentada, Definida
 * (rodadas/horas/dias), Permanente, Descarregar.
 *
 * Regras principais:
 *  - Sustentar: 1 PM/turno; falha em pagar = habilidade termina.
 *  - Múltiplas habilidades sustentadas OK; **apenas 1 magia sustentada
 *    por vez**.
 *  - Morte do dono: habilidades sustentadas terminam; não-sustentadas
 *    continuam até fim da duração.
 *  - Descarregar: gasto PM na ativação da habilidade dormente; encerra
 *    quando o evento gatilho ocorre OU quando a duração original passa.
 *  - Encerrando: dono pode encerrar a qualquer momento; PM não devolve;
 *    efeitos ativos cessam imediatamente.
 *  - Área: persiste no local durante a duração; alvos entram/saem
 *    conforme presença dentro da área.
 *  - Instantânea: efeito acaba na ativação, mas consequências podem
 *    persistir (ex.: cura permanente após magia terminar).
 */

// ─── Types ────────────────────────────────────────────────────────────
export type DurationKind =
  | 'instantanea'
  | 'cena'
  | 'sustentada'
  | 'definida'
  | 'permanente'
  | 'descarregar'

/** Causa do encerramento de uma habilidade ativa (p227). */
export type LifecycleEndCause =
  | 'duration-elapsed'
  | 'owner-ended'
  | 'owner-death'
  | 'sustain-failed'
  | 'discharged'

/** Estado presente da habilidade Descarregar (p227). */
export type DescarregarState =
  | 'dormant'
  | 'discharged'
  | 'expired-without-trigger'

// ─── Constantes ──────────────────────────────────────────────────────
/** Custo de sustento por turno do lançador (p227). */
export const SUSTAIN_COST_PM_PER_TURN = 1

/** Máximo de MAGIAS sustentadas simultaneamente (p227). */
export const MAX_CONCURRENT_SUSTAINED_SPELLS = 1

/**
 * Dono encerrando a habilidade não devolve PM (custo é perdido).
 * Livro p227 não descreve reembolso.
 */
export const OWNER_END_REFUNDS_PM = false

/**
 * Encerrar habilidade é implicitamente ação livre (livro não especifica
 * ação; convenção T20 para efeitos autoencerráveis pelo dono).
 */
export const OWNER_END_ACTION = 'livre' as const

// ─── Helpers — Morte e Duração ──────────────────────────────────────
/**
 * True se a morte do dono encerra a habilidade.
 * Apenas sustentadas terminam por morte (p227).
 */
export function endsOnOwnerDeath(duration: DurationKind): boolean {
  return duration === 'sustentada'
}

/**
 * True se a área da habilidade persiste no local após a morte do
 * criador (habilidades não-sustentadas + não-instantâneas).
 */
export function areaContinuesAfterOwnerDeath(duration: DurationKind): boolean {
  if (duration === 'sustentada') return false
  if (duration === 'instantanea') return false
  return true
}

// ─── Helpers — Sustentar ────────────────────────────────────────────
/**
 * Resolve pagamento do sustento no início do turno.
 * `sustainedCount` = número de habilidades sustentadas ativas.
 * Retorna PM restante e quais habilidades expiraram por falta de PM.
 *
 * Regra simples: cada habilidade sustentada exige 1 PM. Se o dono não
 * tem PM suficiente para todas, ele escolhe quais manter — este helper
 * assume ordem de input (primeiras têm prioridade); ajuste no caller
 * se a lógica de escolha for outra.
 */
export function paySustainCost(
  currentPm: number,
  sustainedAbilityIds: readonly string[],
): {
  pmAfter: number
  kept: readonly string[]
  endedForFailure: readonly string[]
} {
  if (currentPm < 0) {
    throw new Error(
      `paySustainCost: currentPm must be ≥ 0, got ${currentPm}`,
    )
  }
  const kept: string[] = []
  const endedForFailure: string[] = []
  let pm = currentPm
  for (const id of sustainedAbilityIds) {
    if (pm >= SUSTAIN_COST_PM_PER_TURN) {
      pm -= SUSTAIN_COST_PM_PER_TURN
      kept.push(id)
    } else {
      endedForFailure.push(id)
    }
  }
  return { pmAfter: pm, kept, endedForFailure }
}

/**
 * True se o caster pode iniciar outra magia sustentada.
 * Livro (p227): apenas 1 magia sustentada por vez. Habilidades
 * (não-magia) sustentadas são ilimitadas.
 */
export function canSustainAnotherSpell(
  activeSustainedSpellCount: number,
): boolean {
  return activeSustainedSpellCount < MAX_CONCURRENT_SUSTAINED_SPELLS
}

// ─── Helpers — Descarregar ──────────────────────────────────────────
/**
 * Estado de habilidade Descarregar após um evento potencialmente
 * gatilho + tempo decorrido.
 */
export function resolveDescarregar(
  triggerFired: boolean,
  durationExpired: boolean,
): DescarregarState {
  if (triggerFired) return 'discharged'
  if (durationExpired) return 'expired-without-trigger'
  return 'dormant'
}

// ─── Helpers — Encerrando pelo dono ─────────────────────────────────
/**
 * Simula encerramento manual pelo dono. Retorna causa e se o PM
 * gasto originalmente é devolvido (p227: nunca é devolvido).
 */
export function ownerEndAbility(): {
  cause: LifecycleEndCause
  refundsPm: boolean
} {
  return {
    cause: 'owner-ended',
    refundsPm: OWNER_END_REFUNDS_PM,
  }
}

// ─── Helpers — Área ─────────────────────────────────────────────────
/**
 * True se um alvo dentro da área é afetado pelo efeito naquele momento.
 * Áreas só afetam quem está dentro DURANTE a duração (p227).
 */
export function isAffectedByArea(
  targetInsideArea: boolean,
  effectStillActive: boolean,
): boolean {
  return targetInsideArea && effectStillActive
}

/**
 * True se as consequências de uma habilidade Instantânea podem persistir
 * após ela terminar (ex.: cura). Livro p227: sim para instantâneas.
 */
export function instantaneousConsequencesPersist(
  duration: DurationKind,
): boolean {
  return duration === 'instantanea'
}
