/**
 * Parceiro benefits matrix — 12 tipos × 3 patamares (PDF Cap 6 p260-261).
 *
 * Cada tipo fornece um shape distinto de bônus mecânico: discriminated
 * union por `kind`. Complementa `parceiro-rules.ts` (que só carrega os
 * enums + limite + tier ladder de classe).
 *
 * Encoding assumptions:
 *  - Descrições em prosa verbatim ficam em `PARCEIRO_BENEFIT_DESCRIPTIONS`
 *    para UI. Campos mecânicos são exclusivamente estruturados.
 *  - Prerequisites (bardo/nobre/conjurador…) são flavor — regra p260
 *    explicita "classificação é abstrata, não indicando classe e nível".
 *  - Perseguidor/Vigilante têm mistura de skill-bonus (iniciante) +
 *    grantedPower (veterano/mestre) — modelado como campos opcionais.
 */

import type { ParceiroTier, ParceiroType } from './parceiro-rules'

export type AdeptoBenefit = {
  kind: 'adepto'
  /** Círculos de magia com custo de PM reduzido em 1. */
  pmDiscountCirculos: readonly number[]
  /** Se a redução acumula com outras reduções (Especialista, etc). */
  cumulative: boolean
}

export type AjudanteBenefit = {
  kind: 'ajudante'
  /** Valor do bônus aplicado às perícias escolhidas. */
  skillBonus: number
  /** Quantas perícias distintas recebem o bônus. */
  skillCount: number
  /** Perícias vedadas ao bônus (p260 exclui Luta/Pontaria). */
  excludedSkills: readonly ['Luta', 'Pontaria']
}

export type AssassinoBenefit = {
  kind: 'assassino'
  /** Dado extra de Ataque Furtivo (cumulativo com furtivo já possuído). */
  furtivoBonusDice: '+1d6' | '+2d6'
  /** Bônus por flanquear vs 1 inimigo/rodada (veterano+). */
  hasFlankBonus: boolean
  /** Flanquear também facilita Furtivo (mestre). */
  flankAssistsFurtivo: boolean
}

export type AtiradorBenefit = {
  kind: 'atirador'
  /** Dado extra em uma rolagem de dano à distância, 1×/rodada. */
  damageDie: '+1d6' | '+1d10' | '+2d8'
}

export type CombatenteBenefit = {
  kind: 'combatente'
  /** Bônus em testes de ataque. */
  attackBonus: 2 | 3 | 4
  /** Se mestre, PM para 1 ataque extra/rodada (mestre = 5). */
  extraAttackPm?: 5
}

export type DestruidorOption = {
  pmCost: number
  damageDice: string
  /** Alcance/área quando difere do padrão (alcance curto, um alvo). */
  area?: string
}

export type DestruidorBenefit = {
  kind: 'destruidor'
  /** Dano é escolhido do menu ácido/eletricidade/fogo/frio. */
  damageTypeMenu: readonly ['acido', 'eletricidade', 'fogo', 'frio']
  /** Opções cumulativas — mestre tem 3 slots, veterano 2, iniciante 1. */
  options: readonly DestruidorOption[]
}

export type FortaoBenefit = {
  kind: 'fortao'
  /** Dado extra em uma rolagem de dano corpo a corpo, 1×/rodada. */
  damageDie: '+1d8' | '+1d12' | '+3d6'
}

export type GuardiaoBenefit = {
  kind: 'guardiao'
  defesaBonus: 2 | 3 | 4
  /** Bônus em testes de resistência (só mestre). */
  resistBonus?: 2
}

export type MagivocadorBenefit = {
  kind: 'magivocador'
  /** Dados extras de dano do mesmo tipo. */
  damageDicePlus: 1 | 2
  /** Bônus na CD para resistir às magias. */
  cdPlus: 0 | 1 | 2
}

export type MedicoOption = {
  pmCost: 1 | 3 | 5
  healDice: string
  /** Remove condição prejudicial (abalado/fatigado) — opção veterano. */
  removesCondition?: boolean
}

export type MedicoBenefit = {
  kind: 'medico'
  options: readonly MedicoOption[]
}

export type PerseguidorBenefit = {
  kind: 'perseguidor'
  skillBonus?: { skills: readonly string[]; amount: number }
  grantedPower?: 'Sentidos Aguçados' | 'Percepção às Cegas'
}

export type VigilanteBenefit = {
  kind: 'vigilante'
  skillBonus?: { skills: readonly string[]; amount: number }
  grantedPower?: 'Esquiva Sobrenatural' | 'Olhos nas Costas'
}

export type ParceiroBenefit =
  | AdeptoBenefit
  | AjudanteBenefit
  | AssassinoBenefit
  | AtiradorBenefit
  | CombatenteBenefit
  | DestruidorBenefit
  | FortaoBenefit
  | GuardiaoBenefit
  | MagivocadorBenefit
  | MedicoBenefit
  | PerseguidorBenefit
  | VigilanteBenefit

type BenefitMatrix = Readonly<
  Record<ParceiroType, Readonly<Record<ParceiroTier, ParceiroBenefit>>>
>

const DAMAGE_MENU = ['acido', 'eletricidade', 'fogo', 'frio'] as const
const AJUDANTE_EXCLUDED = ['Luta', 'Pontaria'] as const

const RAW = {
  adepto: {
    iniciante: { kind: 'adepto', pmDiscountCirculos: [1], cumulative: false },
    veterano: { kind: 'adepto', pmDiscountCirculos: [1, 2], cumulative: false },
    mestre: { kind: 'adepto', pmDiscountCirculos: [1, 2], cumulative: true },
  },
  ajudante: {
    iniciante: {
      kind: 'ajudante',
      skillBonus: 2,
      skillCount: 2,
      excludedSkills: AJUDANTE_EXCLUDED,
    },
    veterano: {
      kind: 'ajudante',
      skillBonus: 2,
      skillCount: 3,
      excludedSkills: AJUDANTE_EXCLUDED,
    },
    mestre: {
      kind: 'ajudante',
      skillBonus: 4,
      skillCount: 3,
      excludedSkills: AJUDANTE_EXCLUDED,
    },
  },
  assassino: {
    iniciante: {
      kind: 'assassino',
      furtivoBonusDice: '+1d6',
      hasFlankBonus: false,
      flankAssistsFurtivo: false,
    },
    veterano: {
      kind: 'assassino',
      furtivoBonusDice: '+1d6',
      hasFlankBonus: true,
      flankAssistsFurtivo: false,
    },
    mestre: {
      kind: 'assassino',
      furtivoBonusDice: '+2d6',
      hasFlankBonus: true,
      flankAssistsFurtivo: true,
    },
  },
  atirador: {
    iniciante: { kind: 'atirador', damageDie: '+1d6' },
    veterano: { kind: 'atirador', damageDie: '+1d10' },
    mestre: { kind: 'atirador', damageDie: '+2d8' },
  },
  combatente: {
    iniciante: { kind: 'combatente', attackBonus: 2 },
    veterano: { kind: 'combatente', attackBonus: 3 },
    mestre: { kind: 'combatente', attackBonus: 4, extraAttackPm: 5 },
  },
  destruidor: {
    iniciante: {
      kind: 'destruidor',
      damageTypeMenu: DAMAGE_MENU,
      options: [{ pmCost: 1, damageDice: '2d6' }],
    },
    veterano: {
      kind: 'destruidor',
      damageTypeMenu: DAMAGE_MENU,
      options: [
        { pmCost: 1, damageDice: '2d6' },
        { pmCost: 2, damageDice: '4d6' },
      ],
    },
    mestre: {
      kind: 'destruidor',
      damageTypeMenu: DAMAGE_MENU,
      options: [
        { pmCost: 1, damageDice: '2d6' },
        { pmCost: 2, damageDice: '4d6' },
        { pmCost: 4, damageDice: '6d6', area: 'raio 6m alcance médio' },
      ],
    },
  },
  fortao: {
    iniciante: { kind: 'fortao', damageDie: '+1d8' },
    veterano: { kind: 'fortao', damageDie: '+1d12' },
    mestre: { kind: 'fortao', damageDie: '+3d6' },
  },
  guardiao: {
    iniciante: { kind: 'guardiao', defesaBonus: 2 },
    veterano: { kind: 'guardiao', defesaBonus: 3 },
    mestre: { kind: 'guardiao', defesaBonus: 4, resistBonus: 2 },
  },
  magivocador: {
    iniciante: { kind: 'magivocador', damageDicePlus: 1, cdPlus: 0 },
    veterano: { kind: 'magivocador', damageDicePlus: 1, cdPlus: 1 },
    mestre: { kind: 'magivocador', damageDicePlus: 2, cdPlus: 2 },
  },
  medico: {
    iniciante: {
      kind: 'medico',
      options: [{ pmCost: 1, healDice: '1d8+1' }],
    },
    veterano: {
      kind: 'medico',
      options: [
        { pmCost: 1, healDice: '1d8+1' },
        { pmCost: 3, healDice: '3d8+3', removesCondition: true },
      ],
    },
    mestre: {
      kind: 'medico',
      options: [
        { pmCost: 1, healDice: '1d8+1' },
        { pmCost: 3, healDice: '3d8+3', removesCondition: true },
        { pmCost: 5, healDice: '6d8+6' },
      ],
    },
  },
  perseguidor: {
    iniciante: {
      kind: 'perseguidor',
      skillBonus: { skills: ['Percepção', 'Sobrevivência'], amount: 2 },
    },
    veterano: { kind: 'perseguidor', grantedPower: 'Sentidos Aguçados' },
    mestre: { kind: 'perseguidor', grantedPower: 'Percepção às Cegas' },
  },
  vigilante: {
    iniciante: {
      kind: 'vigilante',
      skillBonus: { skills: ['Percepção', 'Iniciativa'], amount: 2 },
    },
    veterano: { kind: 'vigilante', grantedPower: 'Esquiva Sobrenatural' },
    mestre: { kind: 'vigilante', grantedPower: 'Olhos nas Costas' },
  },
} as const satisfies BenefitMatrix

/**
 * Deep-freeze o RAW: tenta congelar o objeto raiz + cada tipo + cada
 * patamar. Preserva literais de `kind` via `as const satisfies`.
 */
function deepFreezeBenefits(): BenefitMatrix {
  for (const type of Object.keys(RAW) as ParceiroType[]) {
    Object.freeze(RAW[type])
  }
  return Object.freeze(RAW)
}

export const PARCEIRO_BENEFITS: BenefitMatrix = deepFreezeBenefits()

/** Descrições verbatim para UI (PDF p260-261). */
export const PARCEIRO_BENEFIT_DESCRIPTIONS: Readonly<
  Record<ParceiroType, Readonly<Record<ParceiroTier, string>>>
> = Object.freeze({
  adepto: Object.freeze({
    iniciante: 'O custo para lançar suas magias de 1º círculo diminui –1 PM.',
    veterano: 'Como acima, mas também reduz o custo de suas magias de 2º círculo.',
    mestre: 'Como acima, e esta redução se torna cumulativa com outras reduções.',
  }),
  ajudante: Object.freeze({
    iniciante: 'Você recebe +2 em duas perícias.',
    veterano: 'Muda para +2 em três perícias.',
    mestre:
      'Muda para +4 em três perícias. As perícias são definidas pelo parceiro. Um ajudante não pode fornecer bônus em Luta ou Pontaria.',
  }),
  assassino: Object.freeze({
    iniciante:
      'Você pode usar a habilidade Ataque Furtivo +1d6. Se já possui a habilidade, o bônus é cumulativo.',
    veterano:
      'Além do Ataque Furtivo, fornece bônus por flanquear contra um inimigo por rodada.',
    mestre:
      'Muda o dano do Ataque Furtivo para +2d6. Note que, além de fornecer +2 em testes de ataque corpo a corpo, o bônus por flanquear facilita que o personagem use seu Ataque Furtivo.',
  }),
  atirador: Object.freeze({
    iniciante:
      'Uma vez por rodada, você recebe +1d6 em uma rolagem de dano à distância.',
    veterano: 'Muda para +1d10.',
    mestre: 'Muda para +2d8.',
  }),
  combatente: Object.freeze({
    iniciante: '+2 em testes de ataque.',
    veterano: 'Muda para +3 em testes de ataque.',
    mestre:
      'Muda para +4 em testes de ataque, e uma vez por rodada, você pode gastar 5 PM para fazer um ataque extra.',
  }),
  destruidor: Object.freeze({
    iniciante:
      'Uma vez por rodada, como uma ação livre, você pode gastar 1 PM para causar 2d6 pontos de dano de ácido, eletricidade, fogo ou frio (de acordo com o parceiro) em um alvo em alcance curto.',
    veterano:
      'Como acima, mas você também pode gastar 2 PM para causar 4d6 pontos de dano.',
    mestre:
      'Como acima, mas você também pode gastar 4 PM para causar 6d6 pontos de dano em uma área de 6m de raio em alcance médio.',
  }),
  fortao: Object.freeze({
    iniciante:
      'Uma vez por rodada, você recebe +1d8 em uma rolagem de dano corpo a corpo.',
    veterano: 'Muda para +1d12.',
    mestre: 'Muda para +3d6.',
  }),
  guardiao: Object.freeze({
    iniciante: 'Você recebe +2 na Defesa.',
    veterano: 'Muda para +3.',
    mestre: 'Muda para +4 na Defesa e +2 em testes de resistência.',
  }),
  magivocador: Object.freeze({
    iniciante: 'O dano de suas magias aumenta em +1 dado do mesmo tipo.',
    veterano: 'Como acima, e a CD para resistir a suas magias aumenta em +1.',
    mestre:
      'Como acima, mas dobra os bônus (para um total de +2 dados de dano e +2 na CD).',
  }),
  medico: Object.freeze({
    iniciante:
      'Uma vez por rodada você pode gastar 1 PM para curar 1d8+1 PV de uma criatura adjacente.',
    veterano:
      'Como acima, mas você pode gastar 3 PM para curar 3d8+3 PV ou remover uma condição prejudicial (como abalado ou fatigado).',
    mestre:
      'Como acima, mas você também pode gastar 5 PM para curar 6d8+6 PV.',
  }),
  perseguidor: Object.freeze({
    iniciante: '+2 em Percepção e Sobrevivência.',
    veterano: 'Você pode usar Sentidos Aguçados.',
    mestre: 'Você pode usar Percepção às Cegas.',
  }),
  vigilante: Object.freeze({
    iniciante: '+2 em Percepção e Iniciativa.',
    veterano: 'Você pode usar Esquiva Sobrenatural.',
    mestre: 'Você pode usar Olhos nas Costas.',
  }),
})

/** Lookup do bônus de parceiro por (tipo, patamar). */
export function parceiroBenefit(
  type: ParceiroType,
  tier: ParceiroTier,
): ParceiroBenefit {
  return PARCEIRO_BENEFITS[type][tier]
}

/** Descrição em prosa PT-BR do bônus (p260-261 verbatim). */
export function parceiroBenefitDescription(
  type: ParceiroType,
  tier: ParceiroTier,
): string {
  return PARCEIRO_BENEFIT_DESCRIPTIONS[type][tier]
}
