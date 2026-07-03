/**
 * Perícia Cura — todos os 4 usos canônicos do PDF p117 (Cap 2 Perícias).
 *
 * Cura (SAB): "Você sabe tratar ferimentos, doenças e venenos."
 *
 * Regra global do kit + auto-uso (aplicam a TODOS os usos):
 *  - Sem maleta de medicamentos: -5 no teste.
 *  - Auto-tratamento: -5 no teste (cumulativo com falta de kit).
 *
 * Estes modificadores já são constantes em `diseases-poisons-rules.ts`:
 *  - `CURA_NO_MALETA_PENALTY = -5`
 *  - `CURA_SELF_TREATMENT_PENALTY = -5`
 *  - `curaTreatmentPenalty({withoutKit, selfTreatment})` — helper agregador
 *
 * Este módulo cataloga os 4 usos (discriminated union) + helpers
 * específicos por uso (dimensionamento por nível para Cuidados
 * Prolongados, CD rara para Necropsia, etc).
 */

// ─── Types ────────────────────────────────────────────────────────────
/** Ação/duração exigida por um uso da perícia Cura. */
export type CuraAction = 'padrao' | 'completa' | '10-minutos' | '1-hora'

/** Chave discriminante dos 4 usos canônicos de Cura (p117). */
export type CuraUsageKind =
  | 'primeiros-socorros'
  | 'tratamento'
  | 'cuidados-prolongados'
  | 'necropsia'

type CuraUsageCommon = {
  id: CuraUsageKind
  name: string
  action: CuraAction
  trainedOnly: boolean
  /** Sempre true na p117 — kit é regra global da perícia. */
  requiresMaleta: true
  /** Verbatim curto do efeito (frase-síntese; texto completo no livro). */
  effect: string
  bookPage: 117
}

/** Primeiros Socorros: estabiliza sangrando. CD 15 fixa. */
export type CuraUsagePrimeirosSocorros = CuraUsageCommon & {
  kind: 'primeiros-socorros'
  dc: 15
}

/**
 * Tratamento: assiste vítima de doença/veneno contínuo. CD = CD da
 * própria doença/veneno. Sucesso concede +5 no próximo Fort da vítima.
 */
export type CuraUsageTratamento = CuraUsageCommon & {
  kind: 'tratamento'
  /** CD variável — usar CD da doença/veneno alvo. */
  dc: 'doenca-ou-veneno-cd'
  bonusNextFortitude: 5
}

/**
 * Cuidados Prolongados: acelera recuperação diária de PV. CD 15 fixa.
 * Duração 1 hora. Bônus escala por nível do curador; teto de pacientes
 * simultâneos = nível do curador.
 */
export type CuraUsageCuidadosProlongados = CuraUsageCommon & {
  kind: 'cuidados-prolongados'
  dc: 15
  /** +1 PV por nível do curador por dia (verbatim). */
  hpBonusPerLevel: 1
}

/**
 * Necropsia: determina causa/momento da morte. CD 20 base; CD 30 para
 * causas raras/extraordinárias (veneno, maldição). Duração 10 minutos.
 */
export type CuraUsageNecropsia = CuraUsageCommon & {
  kind: 'necropsia'
  dcBase: 20
  dcRareCause: 30
}

export type CuraUsage =
  | CuraUsagePrimeirosSocorros
  | CuraUsageTratamento
  | CuraUsageCuidadosProlongados
  | CuraUsageNecropsia

// ─── Catálogo ─────────────────────────────────────────────────────────
export const CURA_USAGES: readonly CuraUsage[] = Object.freeze([
  {
    id: 'primeiros-socorros',
    kind: 'primeiros-socorros',
    name: 'Primeiros Socorros',
    action: 'padrao',
    dc: 15,
    trainedOnly: false,
    requiresMaleta: true,
    effect:
      'Estabiliza um personagem adjacente que esteja sangrando.',
    bookPage: 117,
  },
  {
    id: 'tratamento',
    kind: 'tratamento',
    name: 'Tratamento',
    action: 'completa',
    dc: 'doenca-ou-veneno-cd',
    bonusNextFortitude: 5,
    trainedOnly: true,
    requiresMaleta: true,
    effect:
      'Assiste vítima de doença ou veneno contínuo; +5 no próximo Fort da vítima.',
    bookPage: 117,
  },
  {
    id: 'cuidados-prolongados',
    kind: 'cuidados-prolongados',
    name: 'Cuidados Prolongados',
    action: '1-hora',
    dc: 15,
    hpBonusPerLevel: 1,
    trainedOnly: true,
    requiresMaleta: true,
    effect:
      'Acelera recuperação: +1 PV por nível do curador nesse dia. Teto de pacientes = nível do curador.',
    bookPage: 117,
  },
  {
    id: 'necropsia',
    kind: 'necropsia',
    name: 'Necropsia',
    action: '10-minutos',
    dcBase: 20,
    dcRareCause: 30,
    trainedOnly: true,
    requiresMaleta: true,
    effect:
      'Determina causa e momento aproximado da morte examinando cadáver.',
    bookPage: 117,
  },
])

// ─── Helpers ──────────────────────────────────────────────────────────
const usagesByKind = new Map<CuraUsageKind, CuraUsage>(
  CURA_USAGES.map((u) => [u.kind, u]),
)

export function curaUsageByKind(kind: CuraUsageKind): CuraUsage {
  const usage = usagesByKind.get(kind)
  if (!usage) {
    throw new Error(`curaUsageByKind: unknown kind ${kind}`)
  }
  return usage
}

/**
 * CD final da Necropsia. `rareCause: true` (veneno, maldição, etc)
 * eleva CD para 30; senão 20 (verbatim p117).
 */
export function necropsiaCd(rareCause: boolean): number {
  const usage = curaUsageByKind('necropsia') as CuraUsageNecropsia
  return rareCause ? usage.dcRareCause : usage.dcBase
}

/**
 * Bônus de PV concedido por Cuidados Prolongados em 1 dia:
 * `curatorLevel × 1` (verbatim "+1 por nível").
 */
export function cuidadosProlongadosHpBonus(curatorLevel: number): number {
  if (curatorLevel < 1) {
    throw new Error(
      `cuidadosProlongadosHpBonus: curatorLevel must be ≥ 1, got ${curatorLevel}`,
    )
  }
  return curatorLevel
}

/**
 * Número máximo de pacientes que o curador pode assistir simultaneamente
 * em Cuidados Prolongados (verbatim: "igual ao seu nível").
 */
export function cuidadosProlongadosMaxPatients(curatorLevel: number): number {
  if (curatorLevel < 1) {
    throw new Error(
      `cuidadosProlongadosMaxPatients: curatorLevel must be ≥ 1, got ${curatorLevel}`,
    )
  }
  return curatorLevel
}
