/**
 * Manobras de combate — PDF book p234 (mecânica principal) + p239
 * (Tabela 5-4 Quebrando Objetos) + p228-229 (interações Habilidades
 * Gerais: Agarrar Aprimorado, deslocamento de voo).
 *
 * Definição verbatim (p234): "Uma manobra é um ataque corpo a corpo
 * para fazer algo diferente de causar dano — como arrancar a arma do
 * oponente ou empurrá-lo para um abismo. Não é possível fazer manobras
 * de combate com ataques à distância."
 *
 * Núcleo: manobra substitui um ataque corpo a corpo (Agredir, p233).
 * Faça um teste de manobra (teste de ataque corpo a corpo) oposto com
 * a criatura. Alvo com arma à distância ainda usa Luta no teste
 * oposto. Empate → maior bônus vence; empate de bônus → refazer teste.
 */

// ─── Global rules — verbatim ─────────────────────────────────────────
export const MANEUVER_DEFINITION =
  'Uma manobra é um ataque corpo a corpo para fazer algo diferente de causar dano — como arrancar a arma do oponente ou empurrá-lo para um abismo. Não é possível fazer manobras de combate com ataques à distância.'

/** Empate: maior bônus vence; empate de bônus → refazer teste. */
export const MANEUVER_TIEBREAK_RULE =
  'Em caso de empate, o personagem com o maior bônus vence. Se os bônus forem iguais, outro teste deve ser feito.'

/** Manobras de combate não podem ser feitas com ataques à distância. */
export const MANEUVER_RANGED_ALLOWED = false

/**
 * Ataque à distância contra alvo envolvido em Agarrar tem 50% de
 * chance de acertar o alvo errado (p234, parágrafo final).
 */
export const RANGED_AT_GRABBED_TARGET_MISS_CHANCE_PERCENT = 50

// ─── Rules específicas — verbatim/derivadas ──────────────────────────
/** Derrubar em beirada: alvo pode fazer Reflexos CD 20 para se agarrar. */
export const DERRUBAR_LEDGE_REFLEXOS_CD = 20

/**
 * Criatura voando derrubada cai 1d6 × 1,5m antes de recuperar o voo
 * (p229, deslocamento de voo).
 */
export const DERRUBAR_FLIGHT_FALL_DICE = '1d6 x 1,5m'

/** Agarrar arrasta o alvo, mas o atacante move metade do deslocamento. */
export const AGARRAR_DRAG_SPEED_MULTIPLIER = 0.5

/**
 * Agarrar renovado (esmagar/sufocar): substituir um ataque por novo
 * teste de agarrar contra a criatura já agarrada; ao vencer, causa
 * dano de impacto igual a um ataque desarmado ou arma natural.
 */
export const AGARRAR_RENEW_DEALS_UNARMED_DAMAGE = true

/**
 * Objeto em movimento recebe +5 na Defesa quando alvo da manobra
 * Quebrar / ataque contra objeto (p239 Quebrando Objetos).
 */
export const QUEBRAR_MOVING_OBJECT_DEFENSE_BONUS = 5

// ─── IDs ─────────────────────────────────────────────────────────────
export const MANEUVER_IDS = [
  'agarrar',
  'derrubar',
  'desarmar',
  'empurrar',
  'quebrar',
] as const

export type ManeuverId = (typeof MANEUVER_IDS)[number]

export type ManeuverAction = 'padrao' | 'livre'

/**
 * Contra Luta na maior parte das manobras. Quebrar contra objeto
 * segurado usa Luta do portador; contra objeto solto usa ataque
 * contra Defesa do objeto (não modelado como opposed roll aqui).
 */
export type ManeuverDefenderRoll = 'luta' | 'item-defesa'

export type Maneuver = {
  id: ManeuverId
  name: string
  action: ManeuverAction
  defenderRoll: ManeuverDefenderRoll
  /** Verbatim PDF success text. */
  successEffect: string
  /**
   * Verbatim "vencer por 5+" text (Derrubar, Desarmar). null se não
   * há bônus discreto por margem (Agarrar, Quebrar) ou se a escala é
   * contínua (Empurrar).
   */
  fiveOverEffect: string | null
  /**
   * True se margem ≥ 5 dispara efeito adicional discreto. Empurrar
   * escala continuamente (+1,5m por 5 de margem) — não disparo
   * one-shot.
   */
  hasFiveOverBonus: boolean
  /** Verbatim regra de escape/duração da condição, se aplicável. */
  escapeRules: string | null
  /** Verbatim requisitos de arma/mão. */
  weaponRequirement: string | null
  /** Interações verbatim (avançar junto, arrastar, ataques ao agarrado). */
  specialInteractions: string | null
  bookPage: number
}

export const MANEUVERS: Record<ManeuverId, Maneuver> = {
  agarrar: {
    id: 'agarrar',
    name: 'Agarrar',
    action: 'padrao',
    defenderRoll: 'luta',
    successEffect:
      'Você segura o alvo (por seu braço, sua roupa etc.). Uma criatura agarrada fica desprevenida e imóvel, sofre -2 nos testes de ataque e só pode atacar com armas leves.',
    fiveOverEffect: null,
    hasFiveOverBonus: false,
    escapeRules:
      'Ela pode se soltar com uma ação padrão, vencendo um teste de manobra oposto.',
    weaponRequirement:
      'Você só pode agarrar com um ataque desarmado ou arma natural e, enquanto agarra, fica com essa mão ou arma natural ocupada.',
    specialInteractions:
      'Move-se metade do deslocamento normal, mas arrasta a criatura agarrada. Pode soltá-la com uma ação livre. Pode atacar a criatura agarrada com a mão livre. Substituindo um ataque por novo teste de agarrar contra a criatura já agarrada: ao vencer, causa dano de impacto igual a um ataque desarmado ou arma natural (esmagar/sufocar). Um personagem fazendo um ataque à distância contra um alvo envolvido na manobra agarrar tem 50% de chance de acertar o alvo errado.',
    bookPage: 234,
  },
  derrubar: {
    id: 'derrubar',
    name: 'Derrubar',
    action: 'padrao',
    defenderRoll: 'luta',
    successEffect: 'Você deixa o alvo caído. Esta queda normalmente não causa dano.',
    fiveOverEffect:
      'Se você vencer o teste oposto por 5 pontos ou mais, derruba o oponente com tanta força que também o empurra um quadrado em uma direção a sua escolha.',
    hasFiveOverBonus: true,
    escapeRules: null,
    weaponRequirement: null,
    specialInteractions:
      'Se é jogado além de um parapeito ou precipício, ele pode fazer um teste de Reflexos (CD 20) para se agarrar numa beirada. Uma criatura voando que sofra uma manobra derrubar bem-sucedida cai 1d6 x 1,5m antes de recuperar o voo (p229).',
    bookPage: 234,
  },
  desarmar: {
    id: 'desarmar',
    name: 'Desarmar',
    action: 'padrao',
    defenderRoll: 'luta',
    successEffect:
      'Você derruba um item que a criatura esteja segurando. Normalmente o item cai no mesmo lugar em que ela está (a menos que o alvo esteja voando, sobre uma ponte etc.).',
    fiveOverEffect:
      'Se você vencer o teste oposto por 5 ou mais, derruba o item com tanta força que também o empurra um quadrado em uma direção a sua escolha.',
    hasFiveOverBonus: true,
    escapeRules: null,
    weaponRequirement: null,
    specialInteractions: null,
    bookPage: 234,
  },
  empurrar: {
    id: 'empurrar',
    name: 'Empurrar',
    action: 'padrao',
    defenderRoll: 'luta',
    successEffect:
      'Você empurra a criatura 1,5m. Para cada 5 pontos de diferença entre os testes, você empurra o alvo mais 1,5m.',
    fiveOverEffect: null,
    hasFiveOverBonus: false,
    escapeRules: null,
    weaponRequirement: null,
    specialInteractions:
      'Você pode gastar uma ação de movimento para avançar junto com a criatura (até o limite de seu deslocamento).',
    bookPage: 234,
  },
  quebrar: {
    id: 'quebrar',
    name: 'Quebrar',
    action: 'padrao',
    defenderRoll: 'luta',
    successEffect:
      'Você atinge um item que a criatura esteja segurando. Veja as estatísticas de objetos na página 239.',
    fiveOverEffect: null,
    hasFiveOverBonus: false,
    escapeRules: null,
    weaponRequirement: null,
    specialInteractions:
      'Contra objeto solto: faça um ataque contra a Defesa do objeto (Tabela 5-4 p239). Objeto em movimento recebe +5 na Defesa. Dano normal, com RD do material; objeto reduzido a 0 PV é destruído.',
    bookPage: 234,
  },
}

export type ManeuverOutcome = {
  success: boolean
  /** Margin = attackerTotal - defenderTotal; only meaningful on success. */
  margin: number
  /**
   * True quando a manobra concede seu bônus por vencer por 5+
   * (`hasFiveOverBonus && success && margin >= 5`).
   */
  fiveOverBonus: boolean
}

/**
 * Resolve o resultado de uma manobra a partir dos totais rolados.
 *
 *   Empates vão ao maior bônus por PDF; o caller resolve o empate
 *   antes de passar os *totais* aqui (i.e., números pós-desempate).
 *   Se ainda empatarem após o desempate, o PDF diz "refazer teste";
 *   nesse caso, tratamos como *falha* até o caller re-executar com
 *   as novas rolagens.
 */
export function maneuverOutcome(
  id: ManeuverId,
  attackerTotal: number,
  defenderTotal: number,
): ManeuverOutcome {
  const success = attackerTotal > defenderTotal
  const margin = success ? attackerTotal - defenderTotal : 0
  const entry = MANEUVERS[id]
  return {
    success,
    margin,
    fiveOverBonus: success && entry.hasFiveOverBonus && margin >= 5,
  }
}

/**
 * Distância em metros que Empurrar move o alvo dada a margem do teste
 * oposto (não-negativa). Base 1,5m + 1,5m para cada 5 pontos de margem.
 * Ex: margin 0 → 1,5m; margin 5 → 3m; margin 10 → 4,5m.
 */
export function empurrarDistanceMeters(margin: number): number {
  if (margin < 0) {
    throw new Error(
      `empurrarDistanceMeters: margin must be >= 0, got ${margin}`,
    )
  }
  return 1.5 * (1 + Math.floor(margin / 5))
}

/**
 * Defesa efetiva do objeto contra a manobra Quebrar / ataque contra
 * objeto solto (p239). Objeto em movimento recebe +5.
 */
export function quebrarObjectEffectiveDefense(
  baseDefense: number,
  moving: boolean,
): number {
  return baseDefense + (moving ? QUEBRAR_MOVING_OBJECT_DEFENSE_BONUS : 0)
}

/**
 * Ataque à distância contra alvo envolvido em Agarrar erra o alvo em
 * 50% dos casos (p234). Passe uma rolagem 1-100.
 */
export function rangedAttackAtGrabbedTargetHits(roll1d100: number): boolean {
  if (roll1d100 < 1 || roll1d100 > 100) {
    throw new Error(
      `rangedAttackAtGrabbedTargetHits: roll must be 1-100, got ${roll1d100}`,
    )
  }
  return roll1d100 > RANGED_AT_GRABBED_TARGET_MISS_CHANCE_PERCENT
}
