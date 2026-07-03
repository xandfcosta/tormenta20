/**
 * Perícia Sobrevivência (SAB) — todos os 4 usos canônicos.
 *
 * PDF Cap 2 Perícias — Sobrevivência (livro p117 / PDF p123).
 *
 * Header verbatim: "Você está em casa nos ermos."
 *
 * Regras globais:
 *  - Acampamento exige equipamento de viagem (-5 sem).
 *  - Só Rastrear é "Apenas Treinado"; os outros 3 usos são non-treinado.
 *  - Sem penalidade de armadura.
 *
 * Cross-ref:
 *  - `travel-marcha.ts` — modelo de viagem km/h/dia usa deslocamento;
 *    Orientar-se falha reduz deslocamento à metade OR perde o dia.
 *  - Cross-uso "Sustento" (Ofício p122) permite Sobrevivência gerar
 *    renda — regra vive na entrada de Ofício, não aqui.
 */

// ─── Types ────────────────────────────────────────────────────────────
export type SobrevivenciaUsageKind =
  | 'acampamento'
  | 'identificar-criatura'
  | 'orientar-se'
  | 'rastrear'

/** Categorias de terreno para Acampamento + Orientar-se (p117). */
export type Terreno =
  | 'planicies-colinas'
  | 'florestas-pantanos'
  | 'desertos-montanhas'
  | 'planares-tormenta'

/** Categorias de solo para Rastrear (p117). */
export type TrackSoil = 'macio' | 'comum' | 'duro'

/** Ação/duração exigida por um uso. */
export type SobrevivenciaAction = 'padrao' | 'por-dia' | 'enquanto-move'

type UsageCommon = {
  id: SobrevivenciaUsageKind
  name: string
  action: SobrevivenciaAction
  trainedOnly: boolean
  effect: string
  bookPage: 117
}

export type SobrevivenciaAcampamento = UsageCommon & {
  kind: 'acampamento'
  /** CD base varia por terreno; ver `TERRAIN_CD`. */
  dc: 'varia-por-terreno'
  /** Verbatim: "exige equipamento de viagem. Sem ele, você sofre -5". */
  requiresTravelKit: true
}

export type SobrevivenciaIdentificarCriatura = UsageCommon & {
  kind: 'identificar-criatura'
  /** CD = 15 + ND da criatura. */
  dcFormula: '15+nd'
}

export type SobrevivenciaOrientarSe = UsageCommon & {
  kind: 'orientar-se'
  /** Mesma tabela de terreno de Acampamento. */
  dc: 'varia-por-terreno'
  /** Falha por 5+ = perde-se e NÃO avança no dia. */
  lostByMargin: 5
}

export type SobrevivenciaRastrear = UsageCommon & {
  kind: 'rastrear'
  /** CD varia por solo; ver `TRACK_CD`. */
  dc: 'varia-por-solo'
  /** Deslocamento reduzido à metade enquanto rastreia. */
  movementFraction: 0.5
  /** Retry após falha exige +1 dia. */
  retryAllowed: true
}

export type SobrevivenciaUsage =
  | SobrevivenciaAcampamento
  | SobrevivenciaIdentificarCriatura
  | SobrevivenciaOrientarSe
  | SobrevivenciaRastrear

// ─── Terrain CD table (Acampamento + Orientar-se) ────────────────────
/**
 * Verbatim p117: "15 para planícies e colinas, 20 para florestas e
 * pântanos, 25 para desertos ou montanhas e 30 para regiões planares
 * perigosas ou áreas de Tormenta."
 */
export const TERRAIN_CD: Readonly<Record<Terreno, number>> = Object.freeze({
  'planicies-colinas': 15,
  'florestas-pantanos': 20,
  'desertos-montanhas': 25,
  'planares-tormenta': 30,
})

/**
 * Verbatim: "Regiões muito áridas ou estéreis e clima ruim impõem
 * penalidade cumulativa de -5" — codificado como aumento de CD.
 */
export const TERRAIN_HARSH_PENALTY = 5

// ─── Track CD table (Rastrear) ────────────────────────────────────────
/**
 * Verbatim p117: "15 para solo macio (neve, lama), 20 para solo comum
 * (grama, terra) e 25 para solo duro (rocha ou piso de interiores)."
 */
export const TRACK_CD: Readonly<Record<TrackSoil, number>> = Object.freeze({
  macio: 15,
  comum: 20,
  duro: 25,
})

/**
 * Verbatim: "-5 se estiver rastreando um grupo grande (dez ou mais
 * indivíduos) ou criaturas Enormes ou Colossais" — CD reduzida.
 */
export const TRACK_LARGE_TARGET_CD_REDUCTION = 5

/** Verbatim: "+5 em visibilidade precária (noite, chuva, neblina...)". */
export const TRACK_POOR_VISIBILITY_CD_INCREASE = 5

/** Verbatim: "a cada dia desde a criação dos rastros, a CD aumenta em +1". */
export const TRACK_CD_PER_DAY_AGE = 1

/** Threshold de grupo grande. */
export const TRACK_LARGE_GROUP_MIN_INDIVIDUOS = 10

// ─── Catálogo ─────────────────────────────────────────────────────────
export const SOBREVIVENCIA_USAGES: readonly SobrevivenciaUsage[] =
  Object.freeze([
    {
      id: 'acampamento',
      kind: 'acampamento',
      name: 'Acampamento',
      action: 'por-dia',
      dc: 'varia-por-terreno',
      requiresTravelKit: true,
      trainedOnly: false,
      effect:
        'Consegue abrigo e alimento para você e seu grupo por um dia (caçando, pescando, colhendo frutos).',
      bookPage: 117,
    },
    {
      id: 'identificar-criatura',
      kind: 'identificar-criatura',
      name: 'Identificar Criatura',
      action: 'padrao',
      dcFormula: '15+nd',
      trainedOnly: false,
      effect: 'Identifica animal ou monstro. Veja Misticismo.',
      bookPage: 117,
    },
    {
      id: 'orientar-se',
      kind: 'orientar-se',
      name: 'Orientar-se',
      action: 'por-dia',
      dc: 'varia-por-terreno',
      lostByMargin: 5,
      trainedOnly: false,
      effect:
        'Passa: avança deslocamento normal. Falha: metade. Falha por 5+: perde-se e não avança no dia.',
      bookPage: 117,
    },
    {
      id: 'rastrear',
      kind: 'rastrear',
      name: 'Rastrear',
      action: 'enquanto-move',
      dc: 'varia-por-solo',
      movementFraction: 0.5,
      retryAllowed: true,
      trainedOnly: true,
      effect:
        'Identifica e segue rastros; deslocamento à metade; falha permite nova tentativa em +1 dia.',
      bookPage: 117,
    },
  ])

const usagesByKind = new Map<SobrevivenciaUsageKind, SobrevivenciaUsage>(
  SOBREVIVENCIA_USAGES.map((u) => [u.kind, u]),
)

export function sobrevivenciaUsageByKind(
  kind: SobrevivenciaUsageKind,
): SobrevivenciaUsage {
  const usage = usagesByKind.get(kind)
  if (!usage) {
    throw new Error(`sobrevivenciaUsageByKind: unknown kind ${kind}`)
  }
  return usage
}

// ─── Helpers — Acampamento ────────────────────────────────────────────
/**
 * CD do teste de Sobrevivência para Acampamento.
 * `harshWeatherOrTerrain: true` aplica +5 cumulativo.
 */
export function acampamentoCd(
  terrain: Terreno,
  opts: { harshWeatherOrTerrain?: boolean } = {},
): number {
  let cd = TERRAIN_CD[terrain]
  if (opts.harshWeatherOrTerrain) cd += TERRAIN_HARSH_PENALTY
  return cd
}

// ─── Helpers — Orientar-se ────────────────────────────────────────────
/** Mesma tabela de terreno de Acampamento (verbatim redir). */
export function orientarSeCd(
  terrain: Terreno,
  opts: { harshWeatherOrTerrain?: boolean } = {},
): number {
  return acampamentoCd(terrain, opts)
}

export type OrientarSeOutcome = 'ok' | 'metade' | 'perdido'

/**
 * Resolve resultado do teste de Orientar-se.
 * - Sucesso → 'ok' (deslocamento normal).
 * - Falha < 5 → 'metade' (avança metade).
 * - Falha ≥ 5 → 'perdido' (não avança no dia).
 */
export function orientarSeOutcome(
  checkResult: number,
  cd: number,
): OrientarSeOutcome {
  const delta = checkResult - cd
  if (delta >= 0) return 'ok'
  if (Math.abs(delta) >= 5) return 'perdido'
  return 'metade'
}

// ─── Helpers — Identificar Criatura ───────────────────────────────────
/** CD = 15 + ND da criatura (verbatim). */
export function identificarCriaturaCd(nd: number): number {
  if (nd < 0) {
    throw new Error(
      `identificarCriaturaCd: nd must be ≥ 0, got ${nd}`,
    )
  }
  return 15 + nd
}

// ─── Helpers — Rastrear ───────────────────────────────────────────────
export type RastrearContext = {
  ageInDays?: number
  largeGroupOrHugeCreature?: boolean
  poorVisibility?: boolean
}

/**
 * CD final para Rastrear.
 * Base por solo + `TRACK_CD_PER_DAY_AGE` por dia de idade
 * - `TRACK_LARGE_TARGET_CD_REDUCTION` se alvo grande
 * + `TRACK_POOR_VISIBILITY_CD_INCREASE` se visibilidade precária.
 */
export function rastrearCd(
  soil: TrackSoil,
  ctx: RastrearContext = {},
): number {
  const age = ctx.ageInDays ?? 0
  if (age < 0) {
    throw new Error(`rastrearCd: ageInDays must be ≥ 0, got ${age}`)
  }
  let cd = TRACK_CD[soil] + age * TRACK_CD_PER_DAY_AGE
  if (ctx.largeGroupOrHugeCreature) cd -= TRACK_LARGE_TARGET_CD_REDUCTION
  if (ctx.poorVisibility) cd += TRACK_POOR_VISIBILITY_CD_INCREASE
  return cd
}
