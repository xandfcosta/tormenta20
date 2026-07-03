/**
 * Diseases + Poisons — regras auxiliares além do catálogo em
 * `diseases-poisons.ts`. Complementa entidades já pinnadas com:
 *
 *  - Tratamento via Cura (PDF p117 — Perícia Cura, Tratamento).
 *  - Aplicação de veneno em arma (PDF p161 — Contato).
 *  - Modelo do save de veneno + condição "envenenada" (PDF p161).
 *  - Regra de exposição não cumulativa (PDF p318 — Doenças).
 *
 * Cross-ref:
 *  - `diseases-poisons.ts:advanceDisease` — progressão ladder + cura por
 *    2 sucessos seguidos. Não duplica aqui.
 *
 * NOTA (não corrigido neste módulo): `poisonResistCd(p, int)` em
 * `diseases-poisons.ts` mistura CD-fabricação (20 fixo, p161) com CD do
 * save. O PDF p161 verbatim diz que a CD do save é "definida pelo
 * aplicador do veneno, atributo-chave Int" — sem fórmula fechada nesse
 * box. Modificadores por-veneno (Beladona/Pó de Lich +5) aplicam-se a
 * ambas as CDs. Revisar em módulo futuro; helpers atuais preservados.
 */

// ─── Cura — Tratamento (PDF p117) ────────────────────────────────────
/**
 * Bônus concedido no próximo teste de Fortitude da vítima ao tratador
 * passar no teste de Cura.
 *
 * Verbatim: "Se você passar, o paciente recebe +5 em seu próximo teste
 * de Fortitude contra esse efeito."
 */
export const CURA_TREATMENT_BONUS_NEXT_FORT = 5

/** Ação para tratamento (verbatim "ação completa"). */
export const CURA_TREATMENT_ACTION = 'completa' as const

/** Perícia Cura é "Apenas Treinado" — tratamento herda o gate. */
export const CURA_TREATMENT_TRAINED_ONLY = true

/**
 * Penalidade no teste de Cura sem maleta de medicamentos (p117
 * verbatim: "Sem ela, você sofre –5 no teste").
 */
export const CURA_NO_MALETA_PENALTY = -5

/**
 * Penalidade adicional ao aplicar Cura em si mesmo (p117 verbatim:
 * "Você pode usar a perícia Cura em si mesmo, mas sofre –5 no teste").
 * Cumulativo com penalidade de maleta ausente.
 */
export const CURA_SELF_TREATMENT_PENALTY = -5

/**
 * Penalidade agregada no teste de Cura conforme condições ambientais.
 * Máximo teórico com ambas as penalidades: -10.
 */
export function curaTreatmentPenalty(opts: {
  withoutKit?: boolean
  selfTreatment?: boolean
}): number {
  let penalty = 0
  if (opts.withoutKit) penalty += CURA_NO_MALETA_PENALTY
  if (opts.selfTreatment) penalty += CURA_SELF_TREATMENT_PENALTY
  return penalty
}

// ─── Doenças — regras estruturais (PDF p318) ─────────────────────────
/**
 * PDF p318 verbatim: "Contaminação não é cumulativa; uma vez que
 * contraia a doença, o personagem não sofre efeitos adicionais por ser
 * atingido novamente."
 */
export const DISEASE_EXPOSICAO_CUMULATIVA = false

/**
 * Re-exposição a uma doença que já contaminou a vítima é no-op.
 * `alreadyInfected: true` → nenhum novo teste é rolado; efeito não
 * escala.
 */
export function isDiseaseReexposicaoNoOp(alreadyInfected: boolean): boolean {
  return alreadyInfected && !DISEASE_EXPOSICAO_CUMULATIVA
}

// ─── Venenos — aplicação em arma (PDF p161) ──────────────────────────
/**
 * PDF p161 verbatim: "Aplicar um veneno em uma arma exige uma ação de
 * movimento e uma rolagem de 1d6."
 */
export const POISON_WEAPON_APPLY_ACTION = 'movimento' as const

/** Face do 1d6 que causa autocontaminação ao aplicar veneno em arma. */
export const POISON_WEAPON_SELF_CONTAM_ROLL = 1

/** Dado rolado ao aplicar veneno em arma (verbatim "1d6"). */
export const POISON_WEAPON_APPLY_DIE = 'd6' as const

/**
 * PDF p161 verbatim: "O veneno permanece na arma até acertar um ataque
 * ou até o fim da cena (o que acontecer primeiro)."
 */
export const POISON_WEAPON_DURATION = 'ate-acerto-ou-fim-cena' as const

/** True se rolagem 1d6 causa autocontaminação (rolagem = 1). */
export function isPoisonSelfContamination(d6Roll: number): boolean {
  if (d6Roll < 1 || d6Roll > 6) {
    throw new Error(
      `isPoisonSelfContamination: d6Roll must be in [1,6], got ${d6Roll}`,
    )
  }
  return d6Roll === POISON_WEAPON_SELF_CONTAM_ROLL
}

// ─── Venenos — save + condição envenenada (PDF p161) ─────────────────
/** Tipo do save contra veneno (verbatim). */
export const POISON_SAVE_TYPE = 'fortitude' as const

/**
 * Modelo do save: um único teste por exposição. Efeitos recorrentes
 * NÃO exigem novo teste por rodada — a condição envenenada persiste
 * automaticamente até ser curada.
 */
export const POISON_SAVES_PER_EXPOSURE = 1

/** Condição aplicada por venenos não-instantâneos (verbatim). */
export const POISON_CONDITION_NAME = 'envenenada' as const

/** Tipos possíveis de efeito de veneno (para gate da condição). */
export type PoisonEffectKind = 'instantaneo' | 'recorrente' | 'condicao'

/**
 * PDF p161: "Efeitos que não sejam instantâneos, como perda de PV
 * recorrente ou condições, deixam a vítima com a condição envenenada".
 * Efeitos instantâneos NÃO aplicam a condição.
 */
export function poisonAppliesEnvenenadaCondition(
  effectKind: PoisonEffectKind,
): boolean {
  return effectKind !== 'instantaneo'
}

/**
 * PDF p161: "curar esta condição encerra quaisquer efeitos de veneno
 * (mas não recupera PV perdidos)". Curar a condição encerra a
 * sequência mas PV já perdidos permanecem perdidos.
 */
export const POISON_CURE_RESTORES_LOST_HP = false
