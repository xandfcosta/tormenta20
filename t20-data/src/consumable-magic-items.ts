import type { SpecificItemTier } from './specific-magic-items'
import type { SpellCircle } from './spells'

/** Círculo válido para um item consumível (truques sem item). */
export type ConsumableCircle = Exclude<SpellCircle, 0>

/**
 * Poções + Pergaminhos — PDF Cap 6 (Tesouro), Tabela 8-12, p341.
 *
 * Pricing model per PDF:
 *  - "Poções e pergaminhos são classificados conforme o círculo da
 *    magia que contêm: 1° ou 2° (item mágico menor), 3° ou 4° (médio)
 *    e 5° (maior)."
 *  - **Prices are fixed per row of Tabela 8-12** (not a formula):
 *      - Menor base:    T$ 30
 *      - Médio base:    T$ 270 (entradas aprimoradas começam T$ 120)
 *      - Maior + aprimoramentos extremos: T$ 750 / 1.080 / 1.470 / 3.000
 *
 * Activation (PDF p341):
 *  - Poção: ação padrão para beber.
 *  - Óleo: ação padrão para aplicar no objeto.
 *  - Granada: ação padrão para arremessar em alcance curto.
 *  - Pergaminho: ação padrão OU a ação necessária para lançar a
 *    magia (o que for maior). Exige conhecer a magia OU Misticismo
 *    CD 20 + custo em PM.
 *
 * Tier mapping: `menor` = círculo 1-2; `medio` = círculo 3-4; `maior`
 * = círculo 5 (ou aprimoramento extremo).
 */
export type ConsumableKind = 'pocao' | 'pergaminho'

export type ConsumableMagicItem = {
  id: string
  name: string
  kind: ConsumableKind
  spell: string
  spellCircle: ConsumableCircle
  tier: SpecificItemTier
  priceTibar: number
  effect: string
  bookPage: number
}

/** Tier from circle per PDF p341. */
export function tierForCircle(circle: ConsumableCircle): SpecificItemTier {
  if (circle <= 2) return 'menor'
  if (circle <= 4) return 'medio'
  return 'maior'
}

export const CONSUMABLE_MAGIC_ITEMS: readonly ConsumableMagicItem[] =
  Object.freeze([
    // ─── Poções menores (círculo 1-2) — T$ 30 ───────────────────────
    {
      id: 'pocao-de-abencoar-alimentos',
      name: 'Poção de Abençoar Alimentos',
      kind: 'pocao',
      spell: 'Abençoar Alimentos',
      spellCircle: 1,
      tier: 'menor',
      priceTibar: 30,
      effect:
        'Ao ser bebida, purifica e abençoa os alimentos consumidos pelo usuário (efeito da magia Abençoar Alimentos).',
      bookPage: 341,
    },
    {
      id: 'pocao-de-area-escorregadia',
      name: 'Poção de Área Escorregadia (granada)',
      kind: 'pocao',
      spell: 'Área Escorregadia',
      spellCircle: 1,
      tier: 'menor',
      priceTibar: 30,
      effect:
        'Arremessada em alcance curto, gera área escorregadia centrada no ponto de impacto.',
      bookPage: 341,
    },
    {
      id: 'pocao-de-arma-magica',
      name: 'Poção de Arma Mágica (óleo)',
      kind: 'pocao',
      spell: 'Arma Mágica',
      spellCircle: 1,
      tier: 'menor',
      priceTibar: 30,
      effect: 'Aplicada como óleo sobre uma arma, encanta-a como Arma Mágica.',
      bookPage: 341,
    },
    {
      id: 'pocao-de-curar-ferimentos-menor',
      name: 'Poção de Curar Ferimentos (2d8+2 PV)',
      kind: 'pocao',
      spell: 'Curar Ferimentos',
      spellCircle: 1,
      tier: 'menor',
      priceTibar: 30,
      effect: 'Ao ser bebida, cura 2d8+2 pontos de vida do usuário.',
      bookPage: 341,
    },
    {
      id: 'pocao-de-disfarce-ilusorio',
      name: 'Poção de Disfarce Ilusório',
      kind: 'pocao',
      spell: 'Disfarce Ilusório',
      spellCircle: 1,
      tier: 'menor',
      priceTibar: 30,
      effect: 'Ao ser bebida, gera o efeito de Disfarce Ilusório no usuário.',
      bookPage: 341,
    },
    {
      id: 'pocao-de-escuridao',
      name: 'Poção de Escuridão (óleo)',
      kind: 'pocao',
      spell: 'Escuridão',
      spellCircle: 1,
      tier: 'menor',
      priceTibar: 30,
      effect:
        'Aplicada como óleo em um objeto, fá-lo emanar trevas mágicas (efeito de Escuridão).',
      bookPage: 341,
    },
    {
      id: 'pocao-de-luz',
      name: 'Poção de Luz (óleo)',
      kind: 'pocao',
      spell: 'Luz',
      spellCircle: 1,
      tier: 'menor',
      priceTibar: 30,
      effect:
        'Aplicada como óleo em um objeto, fá-lo brilhar como uma tocha (efeito de Luz).',
      bookPage: 341,
    },
    {
      id: 'pocao-de-nevoa',
      name: 'Poção de Névoa (granada)',
      kind: 'pocao',
      spell: 'Névoa',
      spellCircle: 1,
      tier: 'menor',
      priceTibar: 30,
      effect:
        'Arremessada em alcance curto, gera nuvem de névoa centrada no ponto de impacto.',
      bookPage: 341,
    },
    {
      id: 'pocao-de-protecao-divina',
      name: 'Poção de Proteção Divina',
      kind: 'pocao',
      spell: 'Proteção Divina',
      spellCircle: 1,
      tier: 'menor',
      priceTibar: 30,
      effect: 'Ao ser bebida, gera o efeito de Proteção Divina no usuário.',
      bookPage: 341,
    },
    {
      id: 'pocao-de-sono',
      name: 'Poção de Sono',
      kind: 'pocao',
      spell: 'Sono',
      spellCircle: 1,
      tier: 'menor',
      priceTibar: 30,
      effect: 'Ao ser bebida, faz o usuário cair em sono mágico.',
      bookPage: 341,
    },
    {
      id: 'pocao-de-visao-mistica',
      name: 'Poção de Visão Mística',
      kind: 'pocao',
      spell: 'Visão Mística',
      spellCircle: 1,
      tier: 'menor',
      priceTibar: 30,
      effect: 'Ao ser bebida, concede ao usuário o efeito de Visão Mística.',
      bookPage: 341,
    },
    {
      id: 'pocao-de-vitalidade-fantasma',
      name: 'Poção de Vitalidade Fantasma',
      kind: 'pocao',
      spell: 'Vitalidade Fantasma',
      spellCircle: 1,
      tier: 'menor',
      priceTibar: 30,
      effect:
        'Ao ser bebida, concede PV temporários (efeito de Vitalidade Fantasma).',
      bookPage: 341,
    },

    // ─── Poções médias (círculo 2-3) — T$ 120 / 270 ─────────────────
    {
      id: 'pocao-de-escudo-da-fe-aprimorada',
      name: 'Poção de Escudo da Fé (duração cena)',
      kind: 'pocao',
      spell: 'Escudo da Fé',
      spellCircle: 2,
      tier: 'menor',
      priceTibar: 120,
      effect:
        'Ao ser bebida, concede Escudo da Fé com aprimoramento de duração de cena.',
      bookPage: 341,
    },
    {
      id: 'pocao-de-bola-de-fogo',
      name: 'Poção de Bola de Fogo (granada)',
      kind: 'pocao',
      spell: 'Bola de Fogo',
      spellCircle: 3,
      tier: 'medio',
      priceTibar: 270,
      effect:
        'Arremessada em alcance curto, detona Bola de Fogo centrada no ponto de impacto.',
      bookPage: 341,
    },
    {
      id: 'pocao-de-curar-ferimentos-medio',
      name: 'Poção de Curar Ferimentos (4d8+4 PV)',
      kind: 'pocao',
      spell: 'Curar Ferimentos',
      spellCircle: 3,
      tier: 'medio',
      priceTibar: 270,
      effect: 'Ao ser bebida, cura 4d8+4 pontos de vida do usuário.',
      bookPage: 341,
    },
    {
      id: 'pocao-de-metamorfose',
      name: 'Poção de Metamorfose',
      kind: 'pocao',
      spell: 'Metamorfose',
      spellCircle: 3,
      tier: 'medio',
      priceTibar: 270,
      effect: 'Ao ser bebida, transforma o usuário em outra criatura.',
      bookPage: 341,
    },
    {
      id: 'pocao-de-velocidade',
      name: 'Poção de Velocidade',
      kind: 'pocao',
      spell: 'Velocidade',
      spellCircle: 3,
      tier: 'medio',
      priceTibar: 270,
      effect: 'Ao ser bebida, concede ao usuário o efeito de Velocidade.',
      bookPage: 341,
    },
    {
      id: 'pocao-de-voz-divina',
      name: 'Poção de Voz Divina',
      kind: 'pocao',
      spell: 'Voz Divina',
      spellCircle: 3,
      tier: 'medio',
      priceTibar: 270,
      effect: 'Ao ser bebida, concede ao usuário o efeito de Voz Divina.',
      bookPage: 341,
    },

    // ─── Poções maiores (aprimoramentos extremos) ───────────────────
    {
      id: 'pocao-de-curar-ferimentos-maior',
      name: 'Poção de Curar Ferimentos (7d8+7 PV)',
      kind: 'pocao',
      spell: 'Curar Ferimentos',
      spellCircle: 4,
      tier: 'medio',
      priceTibar: 1080,
      effect: 'Ao ser bebida, cura 7d8+7 pontos de vida do usuário.',
      bookPage: 341,
    },
    {
      id: 'pocao-de-invisibilidade',
      name: 'Poção de Invisibilidade (duração cena)',
      kind: 'pocao',
      spell: 'Invisibilidade',
      spellCircle: 3,
      tier: 'medio',
      priceTibar: 1080,
      effect: 'Ao ser bebida, torna o usuário invisível pela duração da cena.',
      bookPage: 341,
    },
    {
      id: 'pocao-de-curar-ferimentos-suprema',
      name: 'Poção de Curar Ferimentos (11d8+11 PV)',
      kind: 'pocao',
      spell: 'Curar Ferimentos',
      spellCircle: 5,
      tier: 'maior',
      priceTibar: 3000,
      effect: 'Ao ser bebida, cura 11d8+11 pontos de vida do usuário.',
      bookPage: 341,
    },

    // ─── Pergaminhos menores (círculo 1) — T$ 30 ────────────────────
    {
      id: 'pergaminho-de-compreensao',
      name: 'Pergaminho de Compreensão',
      kind: 'pergaminho',
      spell: 'Compreensão',
      spellCircle: 1,
      tier: 'menor',
      priceTibar: 30,
      effect:
        'Ao ser lido, conjura Compreensão (entende qualquer idioma falado ou escrito).',
      bookPage: 341,
    },
    {
      id: 'pergaminho-de-curar-ferimentos-menor',
      name: 'Pergaminho de Curar Ferimentos (2d8+2 PV)',
      kind: 'pergaminho',
      spell: 'Curar Ferimentos',
      spellCircle: 1,
      tier: 'menor',
      priceTibar: 30,
      effect:
        'Ao ser lido, conjura Curar Ferimentos restaurando 2d8+2 PV em um alvo.',
      bookPage: 341,
    },
    {
      id: 'pergaminho-de-primor-atletico',
      name: 'Pergaminho de Primor Atlético',
      kind: 'pergaminho',
      spell: 'Primor Atlético',
      spellCircle: 1,
      tier: 'menor',
      priceTibar: 30,
      effect: 'Ao ser lido, conjura Primor Atlético no alvo.',
      bookPage: 341,
    },
    {
      id: 'pergaminho-de-resistencia-a-energia',
      name: 'Pergaminho de Resistência a Energia',
      kind: 'pergaminho',
      spell: 'Resistência a Energia',
      spellCircle: 1,
      tier: 'menor',
      priceTibar: 30,
      effect: 'Ao ser lido, conjura Resistência a Energia em um alvo.',
      bookPage: 341,
    },
    {
      id: 'pergaminho-de-suporte-ambiental',
      name: 'Pergaminho de Suporte Ambiental',
      kind: 'pergaminho',
      spell: 'Suporte Ambiental',
      spellCircle: 1,
      tier: 'menor',
      priceTibar: 30,
      effect:
        'Ao ser lido, conjura Suporte Ambiental protegendo aliados de condições climáticas.',
      bookPage: 341,
    },
    {
      id: 'pergaminho-de-tranca-arcana',
      name: 'Pergaminho de Tranca Arcana',
      kind: 'pergaminho',
      spell: 'Tranca Arcana',
      spellCircle: 1,
      tier: 'menor',
      priceTibar: 30,
      effect:
        'Ao ser lido, conjura Tranca Arcana em uma porta ou objeto fechado.',
      bookPage: 341,
    },

    // ─── Pergaminhos médios (círculo 2-3) — T$ 270 ──────────────────
    {
      id: 'pergaminho-de-alterar-tamanho',
      name: 'Pergaminho de Alterar Tamanho',
      kind: 'pergaminho',
      spell: 'Alterar Tamanho',
      spellCircle: 2,
      tier: 'menor',
      priceTibar: 270,
      effect: 'Ao ser lido, conjura Alterar Tamanho no alvo escolhido.',
      bookPage: 341,
    },
    {
      id: 'pergaminho-de-aparencia-perfeita',
      name: 'Pergaminho de Aparência Perfeita',
      kind: 'pergaminho',
      spell: 'Aparência Perfeita',
      spellCircle: 2,
      tier: 'menor',
      priceTibar: 270,
      effect: 'Ao ser lido, conjura Aparência Perfeita no alvo.',
      bookPage: 341,
    },
    {
      id: 'pergaminho-de-armamento-da-natureza',
      name: 'Pergaminho de Armamento da Natureza',
      kind: 'pergaminho',
      spell: 'Armamento da Natureza',
      spellCircle: 2,
      tier: 'menor',
      priceTibar: 270,
      effect: 'Ao ser lido, conjura Armamento da Natureza em uma arma.',
      bookPage: 341,
    },
    {
      id: 'pergaminho-de-camuflagem-ilusoria',
      name: 'Pergaminho de Camuflagem Ilusória',
      kind: 'pergaminho',
      spell: 'Camuflagem Ilusória',
      spellCircle: 2,
      tier: 'menor',
      priceTibar: 270,
      effect: 'Ao ser lido, conjura Camuflagem Ilusória no alvo.',
      bookPage: 341,
    },
    {
      id: 'pergaminho-de-concentracao-de-combate',
      name: 'Pergaminho de Concentração de Combate (duração cena)',
      kind: 'pergaminho',
      spell: 'Concentração de Combate',
      spellCircle: 2,
      tier: 'menor',
      priceTibar: 270,
      effect:
        'Ao ser lido, conjura Concentração de Combate com aprimoramento de duração cena.',
      bookPage: 341,
    },
    {
      id: 'pergaminho-de-fisico-divino',
      name: 'Pergaminho de Físico Divino',
      kind: 'pergaminho',
      spell: 'Físico Divino',
      spellCircle: 2,
      tier: 'menor',
      priceTibar: 270,
      effect: 'Ao ser lido, conjura Físico Divino no alvo.',
      bookPage: 341,
    },
    {
      id: 'pergaminho-de-mente-divina',
      name: 'Pergaminho de Mente Divina',
      kind: 'pergaminho',
      spell: 'Mente Divina',
      spellCircle: 2,
      tier: 'menor',
      priceTibar: 270,
      effect: 'Ao ser lido, conjura Mente Divina no alvo.',
      bookPage: 341,
    },
    {
      id: 'pergaminho-de-purificacao',
      name: 'Pergaminho de Purificação',
      kind: 'pergaminho',
      spell: 'Purificação',
      spellCircle: 2,
      tier: 'menor',
      priceTibar: 270,
      effect: 'Ao ser lido, conjura Purificação removendo condições negativas.',
      bookPage: 341,
    },
    {
      id: 'pergaminho-de-vestimenta-da-fe',
      name: 'Pergaminho de Vestimenta da Fé',
      kind: 'pergaminho',
      spell: 'Vestimenta da Fé',
      spellCircle: 2,
      tier: 'menor',
      priceTibar: 270,
      effect: 'Ao ser lido, conjura Vestimenta da Fé.',
      bookPage: 341,
    },

    // ─── Pergaminho aprimorado ──────────────────────────────────────
    {
      id: 'pergaminho-de-arma-magica-aprimorada',
      name: 'Pergaminho de Arma Mágica (óleo; bônus +3)',
      kind: 'pergaminho',
      spell: 'Arma Mágica',
      spellCircle: 1,
      tier: 'menor',
      priceTibar: 750,
      effect:
        'Ao ser lido e aplicado, conjura Arma Mágica aprimorada (bônus +3) em uma arma.',
      bookPage: 341,
    },
  ])

const byId = new Map(CONSUMABLE_MAGIC_ITEMS.map((i) => [i.id, i]))

export function consumableById(
  id: string,
): ConsumableMagicItem | undefined {
  return byId.get(id)
}

export function consumablesByKind(
  kind: ConsumableKind,
): readonly ConsumableMagicItem[] {
  return CONSUMABLE_MAGIC_ITEMS.filter((i) => i.kind === kind)
}

export function consumablesByCircle(
  circle: ConsumableCircle,
): readonly ConsumableMagicItem[] {
  return CONSUMABLE_MAGIC_ITEMS.filter((i) => i.spellCircle === circle)
}

export function consumablesBySpell(
  spell: string,
): readonly ConsumableMagicItem[] {
  return CONSUMABLE_MAGIC_ITEMS.filter((i) => i.spell === spell)
}
