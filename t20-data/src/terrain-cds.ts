/**
 * Terrenos — PDF Cap 6 p268-269. Oito biomas (colinas, desertos, florestas,
 * montanhas, pântanos, planícies, ártico, aquático), cada um com um ou
 * mais elementos mecânicos: CD de perícia, dano, condição imposta ou
 * cobertura/camuflagem oferecida.
 *
 * Cada elemento carrega o campo `feature: string` como discriminante.
 * O objetivo aqui é dar ao GM uma lookup rápida quando o grupo entra
 * em um bioma — CDs, dano, tamanho de acidente etc. Ficam de fora:
 *
 *   - Prosa narrativa ("um lugar árido e quente…").
 *   - "Perigos Ambientais" (avalanches) — Cap 7, já em `environmental-hazards.ts`.
 *   - Objetos usados como cobertura em masmorras (portas, pilares) —
 *     Cap 6 p264-265, `object-statistics.ts`.
 */

export const BIOMES = [
  'colinas',
  'desertos',
  'florestas',
  'montanhas',
  'pantanos',
  'planicies',
  'artico',
  'aquatico',
] as const

export type Biome = (typeof BIOMES)[number]

// ─── Colinas (p268) ──────────────────────────────────────────────────

export const COLINAS_FEATURES = [
  'inclinacao-suave',
  'inclinacao-ingreme',
  'penhasco',
] as const

export type ColinasFeature = (typeof COLINAS_FEATURES)[number]

export type ColinasFeatureRow =
  | {
      feature: 'inclinacao-suave'
      label: 'Inclinação Suave'
      /** Personagens no lado superior recebem bônus por terreno elevado. */
      grantsHighGroundBonus: true
    }
  | {
      feature: 'inclinacao-ingreme'
      label: 'Inclinação Íngreme'
      /** Terreno difícil para subir. */
      difficultToClimb: true
      /** Descer correndo/investindo exige teste (Acrobacia ou Cavalgar). */
      descentTestCd: 10
      /** Falha: cai + rola 1d4×1,5m + 1d6/1,5m impacto. */
      failureConsequence: {
        knockdown: true
        rollDice: '1d4'
        damagePer1_5m: '1d6'
        damageType: 'impacto'
      }
    }
  | {
      feature: 'penhasco'
      label: 'Penhasco'
      heightDice: '1d6'
      heightMultiplierMeters: 3
      /** Escalar exige teste de Atletismo. */
      climbCd: 15
    }

export const COLINAS_TABLE: readonly ColinasFeatureRow[] = Object.freeze([
  {
    feature: 'inclinacao-suave',
    label: 'Inclinação Suave',
    grantsHighGroundBonus: true,
  },
  {
    feature: 'inclinacao-ingreme',
    label: 'Inclinação Íngreme',
    difficultToClimb: true,
    descentTestCd: 10,
    failureConsequence: {
      knockdown: true,
      rollDice: '1d4',
      damagePer1_5m: '1d6',
      damageType: 'impacto',
    },
  },
  {
    feature: 'penhasco',
    label: 'Penhasco',
    heightDice: '1d6',
    heightMultiplierMeters: 3,
    climbCd: 15,
  },
])

// ─── Desertos (p268) ─────────────────────────────────────────────────

export type DesertosFeatureRow = {
  feature: 'dunas'
  label: 'Dunas'
  /** Funcionam como inclinação íngreme, mas cair não causa dano. */
  worksAsInclinacaoIngreme: true
  fallDamage: 'nenhum'
}

export const DESERTOS_TABLE: readonly DesertosFeatureRow[] = Object.freeze([
  {
    feature: 'dunas',
    label: 'Dunas',
    worksAsInclinacaoIngreme: true,
    fallDamage: 'nenhum',
  },
])

// ─── Florestas (p268) ────────────────────────────────────────────────

export const FLORESTAS_FEATURES = [
  'arvore-estreita',
  'arvore-larga',
  'folhagens',
  'vegetacao-rasteira',
] as const

export type FlorestasFeature = (typeof FLORESTAS_FEATURES)[number]

export type FlorestasFeatureRow =
  | {
      feature: 'arvore-estreita'
      label: 'Árvore Estreita'
      widthMax: '<1,5m'
      damageReduction: 5
      hp: 100
      /** Personagem no mesmo espaço ganha cobertura leve. */
      cover: 'leve'
      climbCd: 15
      /** No topo, precisa se equilibrar. */
      balanceCd: 15
    }
  | {
      feature: 'arvore-larga'
      label: 'Árvore Larga'
      widthMin: '>1,5m'
      damageReduction: 5
      hp: 500
      /** Não pode ficar no mesmo espaço; atrás ganha cobertura leve. */
      cover: 'leve'
      climbCd: 15
      balanceCd: 15
      /** No topo, camuflagem leve contra criaturas no solo. */
      topGrantsConcealment: 'leve'
    }
  | {
      feature: 'folhagens'
      label: 'Folhagens'
      difficultTerrain: true
      concealmentInside: 'leve'
    }
  | {
      feature: 'vegetacao-rasteira'
      label: 'Vegetação Rasteira'
      difficultTerrain: true
      /** Penalidade em Furtividade por folhas secas / galhos caídos. */
      stealthPenalty: -2
    }

export const FLORESTAS_TABLE: readonly FlorestasFeatureRow[] = Object.freeze([
  {
    feature: 'arvore-estreita',
    label: 'Árvore Estreita',
    widthMax: '<1,5m',
    damageReduction: 5,
    hp: 100,
    cover: 'leve',
    climbCd: 15,
    balanceCd: 15,
  },
  {
    feature: 'arvore-larga',
    label: 'Árvore Larga',
    widthMin: '>1,5m',
    damageReduction: 5,
    hp: 500,
    cover: 'leve',
    climbCd: 15,
    balanceCd: 15,
    topGrantsConcealment: 'leve',
  },
  {
    feature: 'folhagens',
    label: 'Folhagens',
    difficultTerrain: true,
    concealmentInside: 'leve',
  },
  {
    feature: 'vegetacao-rasteira',
    label: 'Vegetação Rasteira',
    difficultTerrain: true,
    stealthPenalty: -2,
  },
])

// ─── Montanhas (p268) ────────────────────────────────────────────────

export const MONTANHAS_FEATURES = [
  'abismo',
  'altitude',
  'paredao',
  'seixos',
] as const

export type MontanhasFeature = (typeof MONTANHAS_FEATURES)[number]

export type MontanhasFeatureRow =
  | {
      feature: 'abismo'
      label: 'Abismo'
      widthDice: '1d4'
      widthMultiplierMeters: 1.5
      depthDice: '2d4'
      depthMultiplierMeters: 3
      climbOutCd: 20
    }
  | {
      feature: 'altitude'
      label: 'Altitude'
      /** Fort CD 15 (cumulativo +1 por teste anterior), por dia. */
      fortCdBase: 15
      period: 'dia'
      /** Falha aplica a condição fatigado (ou exausto se já fatigado). */
      failureEffect: 'fatigado-ou-exausto'
    }
  | {
      feature: 'paredao'
      label: 'Paredão'
      heightDice: '2d6'
      heightMultiplierMeters: 3
      climbCd: 25
    }
  | {
      feature: 'seixos'
      label: 'Seixos'
      /** Substitui a CD de descer inclinação íngreme (10 → 15). */
      descentTestCd: 15
    }

export const MONTANHAS_TABLE: readonly MontanhasFeatureRow[] = Object.freeze([
  {
    feature: 'abismo',
    label: 'Abismo',
    widthDice: '1d4',
    widthMultiplierMeters: 1.5,
    depthDice: '2d4',
    depthMultiplierMeters: 3,
    climbOutCd: 20,
  },
  {
    feature: 'altitude',
    label: 'Altitude',
    fortCdBase: 15,
    period: 'dia',
    failureEffect: 'fatigado-ou-exausto',
  },
  {
    feature: 'paredao',
    label: 'Paredão',
    heightDice: '2d6',
    heightMultiplierMeters: 3,
    climbCd: 25,
  },
  {
    feature: 'seixos',
    label: 'Seixos',
    descentTestCd: 15,
  },
])

// ─── Pântanos (p268) ─────────────────────────────────────────────────

export type PantanosFeatureRow = {
  feature: 'lodacal'
  label: 'Lodaçal'
  difficultTerrain: true
  conditionApplied: 'vulneravel'
}

export const PANTANOS_TABLE: readonly PantanosFeatureRow[] = Object.freeze([
  {
    feature: 'lodacal',
    label: 'Lodaçal',
    difficultTerrain: true,
    conditionApplied: 'vulneravel',
  },
])

// ─── Planícies (p268) ────────────────────────────────────────────────

export type PlaniciesFeatureRow = {
  feature: 'trincheira'
  label: 'Trincheira'
  /** Ocupante recebe cobertura leve contra ataques à distância. */
  cover: 'leve'
  coverScope: 'ranged'
  /** Sair da trincheira conta como terreno difícil. */
  exitAsDifficultTerrain: true
}

export const PLANICIES_TABLE: readonly PlaniciesFeatureRow[] = Object.freeze([
  {
    feature: 'trincheira',
    label: 'Trincheira',
    cover: 'leve',
    coverScope: 'ranged',
    exitAsDifficultTerrain: true,
  },
])

// ─── Ártico (p269) ───────────────────────────────────────────────────

export const ARTICO_FEATURES = ['gelo', 'rio-congelado'] as const

export type ArticoFeature = (typeof ARTICO_FEATURES)[number]

export type ArticoFeatureRow =
  | {
      feature: 'gelo'
      label: 'Gelo'
      /** Metade do deslocamento sem teste. */
      halfSpeedFree: true
      /**
       * Deslocamento normal, corrida, investida, ou sofrer dano exige
       * teste de Acrobacia. `movement` = movimento normal (CD 15);
       * `damage` = sofrer dano (CD igual ao dano sofrido).
       */
      acrobaticsCd: { movement: 15; damage: 'equal-to-damage' }
      /** Falha: cai + desliza 1d4×1,5m. */
      failureSlide: { rollDice: '1d4'; multiplierMeters: 1.5 }
    }
  | {
      feature: 'rio-congelado'
      label: 'Rio Congelado'
      /** Comporta-se como gelo. */
      inheritsFromGelo: true
      /** Se rolar 1 no d4 de desliza, o gelo quebra. */
      iceCracksOnSlideRoll: 1
      /** Se afundar: 1d6 frio/rodada + precisa nadar. */
      submergedDamagePerRound: { dice: '1d6'; type: 'frio' }
      /** Abrir buraco: causar 10 pontos de dano de impacto ou fogo. */
      breakThroughDamage: { amount: 10; types: readonly ['impacto', 'fogo'] }
    }

export const ARTICO_TABLE: readonly ArticoFeatureRow[] = Object.freeze([
  {
    feature: 'gelo',
    label: 'Gelo',
    halfSpeedFree: true,
    acrobaticsCd: { movement: 15, damage: 'equal-to-damage' },
    failureSlide: { rollDice: '1d4', multiplierMeters: 1.5 },
  },
  {
    feature: 'rio-congelado',
    label: 'Rio Congelado',
    inheritsFromGelo: true,
    iceCracksOnSlideRoll: 1,
    submergedDamagePerRound: { dice: '1d6', type: 'frio' },
    breakThroughDamage: { amount: 10, types: ['impacto', 'fogo'] },
  },
])

// ─── Aquático (p269) ─────────────────────────────────────────────────

export const AQUATICO_FEATURES = [
  'agua-corrente',
  'agua-parada',
  'submerso',
] as const

export type AquaticoFeature = (typeof AQUATICO_FEATURES)[number]

export type AquaticoFeatureRow =
  | {
      feature: 'agua-corrente'
      label: 'Água Corrente'
      currentSpeedDice: '1d6'
      currentMultiplierMeters: 3
      /** CD de Atletismo para nadar num rio (varia com a correnteza). */
      swimCd: { slowCorrenteza: 15; fastCorrenteza: 20 }
      /** Sair de correnteza ≥15m/rodada exige Atletismo CD 20 num ponto de apoio. */
      exitCd: 20
    }
  | {
      feature: 'agua-parada'
      label: 'Água Parada'
      /** Sem modificadores extras — usa CD padrão da perícia Atletismo. */
      requiresPlainSwim: true
    }
  | {
      feature: 'submerso'
      label: 'Personagens Submersos'
      /** Não pode falar (portanto, não lança magias verbais). */
      cannotSpeak: true
      attackPenalty: -2
      perceptionPenalty: -5
      /** Ataques à distância proibidos, exceto perfuração/bestas/redes. */
      rangedForbiddenExcept: readonly ['arremesso-perfuracao', 'bestas', 'redes']
      /** Corte/impacto (exceto arma natural) causam metade do dano. */
      slashImpactHalfDamage: true
      /** Camuflagem + cobertura leves contra atacantes fora d'água. */
      againstAboveWaterConcealment: 'leve'
      againstAboveWaterCover: 'leve'
    }

export const AQUATICO_TABLE: readonly AquaticoFeatureRow[] = Object.freeze([
  {
    feature: 'agua-corrente',
    label: 'Água Corrente',
    currentSpeedDice: '1d6',
    currentMultiplierMeters: 3,
    swimCd: { slowCorrenteza: 15, fastCorrenteza: 20 },
    exitCd: 20,
  },
  {
    feature: 'agua-parada',
    label: 'Água Parada',
    requiresPlainSwim: true,
  },
  {
    feature: 'submerso',
    label: 'Personagens Submersos',
    cannotSpeak: true,
    attackPenalty: -2,
    perceptionPenalty: -5,
    rangedForbiddenExcept: ['arremesso-perfuracao', 'bestas', 'redes'],
    slashImpactHalfDamage: true,
    againstAboveWaterConcealment: 'leve',
    againstAboveWaterCover: 'leve',
  },
])

// ─── Índice cruzado ──────────────────────────────────────────────────

export type BiomeFeatureRow =
  | ColinasFeatureRow
  | DesertosFeatureRow
  | FlorestasFeatureRow
  | MontanhasFeatureRow
  | PantanosFeatureRow
  | PlaniciesFeatureRow
  | ArticoFeatureRow
  | AquaticoFeatureRow

export function biomeFeatures(biome: Biome): readonly BiomeFeatureRow[] {
  switch (biome) {
    case 'colinas':
      return COLINAS_TABLE
    case 'desertos':
      return DESERTOS_TABLE
    case 'florestas':
      return FLORESTAS_TABLE
    case 'montanhas':
      return MONTANHAS_TABLE
    case 'pantanos':
      return PANTANOS_TABLE
    case 'planicies':
      return PLANICIES_TABLE
    case 'artico':
      return ARTICO_TABLE
    case 'aquatico':
      return AQUATICO_TABLE
  }
}
