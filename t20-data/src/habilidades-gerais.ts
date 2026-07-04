/**
 * Habilidades Gerais — catálogo de criaturas.
 *
 * PDF Cap 5 Jogando p228-229 (seção Habilidades). Cobre 16 habilidades
 * comuns a criaturas/monstros. RD numérica vive em [[damage-reduction]];
 * aqui a RD entra apenas como entrada do catálogo (parâmetros).
 *
 * Alcance curto padrão = 9m (p224).
 */

// ─── Types ────────────────────────────────────────────────────────────
export type HabilidadeGeralKind =
  | 'agarrar-aprimorado'
  | 'cura-acelerada'
  | 'deslocamento-escalada'
  | 'deslocamento-escavacao'
  | 'deslocamento-natacao'
  | 'deslocamento-voo'
  | 'faro'
  | 'fortificacao'
  | 'imunidade'
  | 'incorporeo'
  | 'percepcao-as-cegas'
  | 'reducao-de-dano'
  | 'resistencia-a-efeito'
  | 'visao-na-penumbra'
  | 'visao-no-escuro'
  | 'vulnerabilidade-dano'

/** Alvo aceito por Imunidade (p229). */
export type ImunidadeTarget =
  | 'tipo-de-dano'
  | 'tipo-de-efeito'
  | 'condicao'
  | 'habilidade'

type Common = {
  id: HabilidadeGeralKind
  name: string
  effect: string
  bookPage: 228 | 229
}

export type HabilidadeAgarrarAprimorado = Common & {
  kind: 'agarrar-aprimorado'
  /** Requer arma natural especificada. */
  requiresSpecifiedNaturalWeapon: true
  /** Manobra agarrar como ação livre no acerto. */
  actionOnHit: 'livre'
}

export type HabilidadeCuraAcelerada = Common & {
  kind: 'cura-acelerada'
  /** PV recuperados no início do turno. */
  hpPerTurn: number
  /** Sufixo /tipo bloqueia regeneração daquele tipo. Ex: /ácido. */
  blockedByDamageType?: string
  /** Não cura perda de PV máximo (só dano). */
  healsMaxHpLoss: false
}

export type HabilidadeDeslocamento = Common & {
  kind:
    | 'deslocamento-escalada'
    | 'deslocamento-escavacao'
    | 'deslocamento-natacao'
    | 'deslocamento-voo'
  /** Deslocamento em metros. */
  speedMeters: number
}

export type HabilidadeFaro = Common & {
  kind: 'faro'
  /** Alvo em alcance curto (9m) sofre 20% falha em vez de 50%. */
  camuflagemTotalFailureRate: 0.2
  range: 'curto'
}

export type HabilidadeFortificacao = Common & {
  kind: 'fortificacao'
  /** Chance de ignorar dano extra de crítico/ataque furtivo. */
  ignoreExtraDamageChance: number
}

export type HabilidadeImunidade = Common & {
  kind: 'imunidade'
  target: ImunidadeTarget
  /** Descrição do alvo específico (ex: 'fogo', 'sono', 'curado'). */
  specificTarget: string
  /** Imunidade a crítico converte crítico em acerto normal. */
  criticalConvertsToNormal: boolean
}

export type HabilidadeIncorporeo = Common & {
  kind: 'incorporeo'
  /** Só afetada por armas/efeitos mágicos ou outras criaturas incorpóreas. */
  onlyAffectedByMagical: true
  /** Força efetivamente nula. */
  hasNoStrength: true
}

export type HabilidadePercepcaoAsCegas = Common & {
  kind: 'percepcao-as-cegas'
  /** Alcance curto por padrão salvo indicação da criatura. */
  range: 'curto'
  /** Escuridão e invisibilidade não afetam. */
  ignoresDarknessAndInvisibility: true
}

export type HabilidadeReducaoDeDano = Common & {
  kind: 'reducao-de-dano'
  /** Valor de RD (mecânica em damage-reduction.ts). */
  value: number
  /** Tipo/exceção do sufixo /tipo (ex: 'magico', 'contundente'). */
  bypassedBy?: string
}

export type HabilidadeResistenciaAEfeito = Common & {
  kind: 'resistencia-a-efeito'
  /** Bônus em Fort/Ref/Vontade contra o efeito. */
  bonus: number
  /** Descrição do tipo de efeito (ex: 'magia', 'frio'). */
  effectType: string
}

export type HabilidadeVisaoNaPenumbra = Common & {
  kind: 'visao-na-penumbra'
  range: 'curto'
  /** Não funciona em escuridão mágica. */
  worksInMagicalDarkness: false
}

export type HabilidadeVisaoNoEscuro = Common & {
  kind: 'visao-no-escuro'
  range: 'curto'
  worksInMagicalDarkness: false
}

export type HabilidadeVulnerabilidadeDano = Common & {
  kind: 'vulnerabilidade-dano'
  damageType: string
  /** Multiplicador de dano (1.5×). */
  damageMultiplier: 1.5
}

export type HabilidadeGeral =
  | HabilidadeAgarrarAprimorado
  | HabilidadeCuraAcelerada
  | HabilidadeDeslocamento
  | HabilidadeFaro
  | HabilidadeFortificacao
  | HabilidadeImunidade
  | HabilidadeIncorporeo
  | HabilidadePercepcaoAsCegas
  | HabilidadeReducaoDeDano
  | HabilidadeResistenciaAEfeito
  | HabilidadeVisaoNaPenumbra
  | HabilidadeVisaoNoEscuro
  | HabilidadeVulnerabilidadeDano

// ─── Constantes ──────────────────────────────────────────────────────
export const CURTO_RANGE_METERS = 9

/** Vulnerabilidade multiplica o dano do tipo por 1.5 (p229). */
export const VULNERABILIDADE_DAMAGE_MULTIPLIER = 1.5

/** Alvo em alcance curto sofre 20% falha em vez de 50% contra Faro. */
export const FARO_CAMUFLAGEM_TOTAL_FAILURE_RATE = 0.2

/** Queda de 150m/rodada ao perder voo (p229). */
export const DESLOCAMENTO_VOO_FALL_PER_ROUND_M = 150

/** Alvo de manobra derrubar bem-sucedida cai 1d6 × 1,5m antes de recuperar voo. */
export const DESLOCAMENTO_VOO_TRIP_FALL_METERS_PER_D6 = 1.5

// ─── Fábricas para o catálogo (parâmetros variam por criatura) ──────
/**
 * Constrói entrada de Cura Acelerada. Passe `blockedByDamageType` para
 * `Cura Acelerada X/tipo`.
 */
export function habilidadeCuraAcelerada(
  hpPerTurn: number,
  blockedByDamageType?: string,
): HabilidadeCuraAcelerada {
  if (hpPerTurn <= 0) {
    throw new Error(
      `habilidadeCuraAcelerada: hpPerTurn must be > 0, got ${hpPerTurn}`,
    )
  }
  return {
    id: 'cura-acelerada',
    kind: 'cura-acelerada',
    name: 'Cura Acelerada',
    hpPerTurn,
    blockedByDamageType,
    healsMaxHpLoss: false,
    effect: `Recupera ${hpPerTurn} PV no início do turno${
      blockedByDamageType ? ` (bloqueada por dano de ${blockedByDamageType})` : ''
    }; não cura perda de PV máximo.`,
    bookPage: 228,
  }
}

/** Constrói entrada de Deslocamento (escalada/escavação/natação/voo). */
export function habilidadeDeslocamento(
  kind:
    | 'deslocamento-escalada'
    | 'deslocamento-escavacao'
    | 'deslocamento-natacao'
    | 'deslocamento-voo',
  speedMeters: number,
): HabilidadeDeslocamento {
  if (speedMeters <= 0) {
    throw new Error(
      `habilidadeDeslocamento: speedMeters must be > 0, got ${speedMeters}`,
    )
  }
  return {
    id: kind,
    kind,
    name: displacementDisplayName(kind),
    speedMeters,
    effect: `${displacementDisplayName(kind)} ${speedMeters}m.`,
    bookPage: kind === 'deslocamento-voo' ? 229 : 228,
  }
}

function displacementDisplayName(
  kind: HabilidadeDeslocamento['kind'],
): string {
  switch (kind) {
    case 'deslocamento-escalada':
      return 'Deslocamento de Escalada'
    case 'deslocamento-escavacao':
      return 'Deslocamento de Escavação'
    case 'deslocamento-natacao':
      return 'Deslocamento de Natação'
    case 'deslocamento-voo':
      return 'Deslocamento de Voo'
  }
}

/** Constrói entrada de Imunidade. */
export function habilidadeImunidade(
  target: ImunidadeTarget,
  specificTarget: string,
): HabilidadeImunidade {
  const criticalConvertsToNormal =
    target === 'habilidade' && specificTarget === 'acerto-critico'
  return {
    id: 'imunidade',
    kind: 'imunidade',
    name: `Imunidade a ${specificTarget}`,
    target,
    specificTarget,
    criticalConvertsToNormal,
    effect: `Imune a ${specificTarget}${
      criticalConvertsToNormal ? '; críticos viram acertos normais.' : '.'
    }`,
    bookPage: 229,
  }
}

/** Constrói entrada de Redução de Dano (mecânica delegada a damage-reduction.ts). */
export function habilidadeReducaoDeDano(
  value: number,
  bypassedBy?: string,
): HabilidadeReducaoDeDano {
  if (value <= 0) {
    throw new Error(
      `habilidadeReducaoDeDano: value must be > 0, got ${value}`,
    )
  }
  return {
    id: 'reducao-de-dano',
    kind: 'reducao-de-dano',
    name: 'Redução de Dano',
    value,
    bypassedBy,
    effect: `Ignora ${value} de dano${
      bypassedBy ? ` (exceto contra ${bypassedBy})` : ''
    }; delega a damage-reduction.`,
    bookPage: 229,
  }
}

/** Constrói entrada de Resistência a Efeito. */
export function habilidadeResistenciaAEfeito(
  effectType: string,
  bonus: number,
): HabilidadeResistenciaAEfeito {
  return {
    id: 'resistencia-a-efeito',
    kind: 'resistencia-a-efeito',
    name: `Resistência a ${effectType}`,
    bonus,
    effectType,
    effect: `+${bonus} em Fort/Ref/Vontade contra ${effectType}.`,
    bookPage: 229,
  }
}

/** Constrói entrada de Vulnerabilidade a Dano. */
export function habilidadeVulnerabilidadeDano(
  damageType: string,
): HabilidadeVulnerabilidadeDano {
  return {
    id: 'vulnerabilidade-dano',
    kind: 'vulnerabilidade-dano',
    name: `Vulnerabilidade a ${damageType}`,
    damageType,
    damageMultiplier: VULNERABILIDADE_DAMAGE_MULTIPLIER,
    effect: `Sofre ${VULNERABILIDADE_DAMAGE_MULTIPLIER}× dano de ${damageType}.`,
    bookPage: 229,
  }
}

/** Constrói entrada de Fortificação com percentual. */
export function habilidadeFortificacao(
  percent: number,
): HabilidadeFortificacao {
  if (percent <= 0 || percent > 100) {
    throw new Error(
      `habilidadeFortificacao: percent must be in (0, 100], got ${percent}`,
    )
  }
  return {
    id: 'fortificacao',
    kind: 'fortificacao',
    name: `Fortificação ${percent}%`,
    ignoreExtraDamageChance: percent / 100,
    effect: `${percent}% de chance de ignorar dano extra de críticos e ataques furtivos.`,
    bookPage: 229,
  }
}

// ─── Entradas paramétricas fixas (não variam por criatura) ──────────
export const HABILIDADE_AGARRAR_APRIMORADO: HabilidadeAgarrarAprimorado = Object.freeze({
  id: 'agarrar-aprimorado',
  kind: 'agarrar-aprimorado',
  name: 'Agarrar Aprimorado',
  requiresSpecifiedNaturalWeapon: true,
  actionOnHit: 'livre',
  effect:
    'Ao acertar arma natural, faz manobra agarrar como ação livre; não desfere outros ataques com essa arma enquanto agarra.',
  bookPage: 228,
})

export const HABILIDADE_FARO: HabilidadeFaro = Object.freeze({
  id: 'faro',
  kind: 'faro',
  name: 'Faro',
  camuflagemTotalFailureRate: 0.2,
  range: 'curto',
  effect: 'Alcance curto (9m); camuflagem total contra Faro causa apenas 20% de falha.',
  bookPage: 229,
})

export const HABILIDADE_INCORPOREO: HabilidadeIncorporeo = Object.freeze({
  id: 'incorporeo',
  kind: 'incorporeo',
  name: 'Incorpóreo',
  onlyAffectedByMagical: true,
  hasNoStrength: true,
  effect: 'Só afetada por armas/efeitos mágicos ou criaturas incorpóreas; Força nula.',
  bookPage: 229,
})

export const HABILIDADE_PERCEPCAO_AS_CEGAS: HabilidadePercepcaoAsCegas = Object.freeze({
  id: 'percepcao-as-cegas',
  kind: 'percepcao-as-cegas',
  name: 'Percepção às Cegas',
  range: 'curto',
  ignoresDarknessAndInvisibility: true,
  effect: 'Alcance curto (9m); escuridão e invisibilidade não afetam Percepção.',
  bookPage: 229,
})

export const HABILIDADE_VISAO_NA_PENUMBRA: HabilidadeVisaoNaPenumbra = Object.freeze({
  id: 'visao-na-penumbra',
  kind: 'visao-na-penumbra',
  name: 'Visão na Penumbra',
  range: 'curto',
  worksInMagicalDarkness: false,
  effect: 'Enxerga em escuridão leve não-mágica em alcance curto; ignora camuflagem leve.',
  bookPage: 229,
})

export const HABILIDADE_VISAO_NO_ESCURO: HabilidadeVisaoNoEscuro = Object.freeze({
  id: 'visao-no-escuro',
  kind: 'visao-no-escuro',
  name: 'Visão no Escuro',
  range: 'curto',
  worksInMagicalDarkness: false,
  effect: 'Enxerga em escuridão total não-mágica em alcance curto; ignora camuflagem total.',
  bookPage: 229,
})

// ─── Helpers ────────────────────────────────────────────────────────
/** Aplica multiplicador de vulnerabilidade (1.5×). */
export function vulnerabilidadeDamage(baseDamage: number): number {
  if (baseDamage < 0) {
    throw new Error(
      `vulnerabilidadeDamage: baseDamage must be ≥ 0, got ${baseDamage}`,
    )
  }
  return Math.floor(baseDamage * VULNERABILIDADE_DAMAGE_MULTIPLIER)
}

/**
 * Queda ao ser derrubado voando: 1d6 × 1,5m (input é o resultado do d6).
 */
export function voarQuedaDerrubadoM(d6Roll: number): number {
  if (d6Roll < 1 || d6Roll > 6) {
    throw new Error(`voarQuedaDerrubadoM: d6Roll must be 1..6, got ${d6Roll}`)
  }
  return d6Roll * DESLOCAMENTO_VOO_TRIP_FALL_METERS_PER_D6
}
