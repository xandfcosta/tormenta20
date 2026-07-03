/**
 * Montarias — parceiros do tipo montaria (PDF Cap 6 p261-262).
 *
 * T20 modela parceiros como bloco de bônus aplicado ao jogador, não
 * NPCs autônomos. Portanto o "stat block" das 6 montarias reduz-se a:
 *  - tamanho + deslocamento por patamar
 *  - modificadores (ataque/defesa/dano CaC/resistências)
 *  - sentidos/manobras concedidos
 *  - preço em T$ (quando presente na Tabela de equipamentos Cap 4)
 *
 * Cross-refs:
 *  - `parceiro-rules.ts` — 3 patamares (iniciante/veterano/mestre)
 *  - Cap 4 (tabela de equipamentos) — preços de cavalo/pônei/cão/trobo
 *
 * Casos especiais:
 *  - Grifo iniciante: é filhote, `ridavel = false` (só combate a pé).
 *  - Cavalo compartilha tiers com pôneis (tamanho Médio).
 *  - Lobo-das-cavernas compartilha com lobo comum (tamanho Médio).
 */

import type { ParceiroTier } from './parceiro-rules'

export type MontariaId =
  | 'cavalo'
  | 'cao-de-caca'
  | 'lobo-das-cavernas'
  | 'grifo'
  | 'gorlogg'
  | 'trobo'

export type MontariaSize = 'pequeno' | 'medio' | 'grande'
export type MovementType = 'terrestre' | 'voo'

export type MontariaDanoCaC = {
  die: '+1d6' | '+1d8' | '+1d10' | '+1d12' | '+2d8'
  timesPerRound: 1
}

export type MontariaTierData = {
  /** Deslocamento em metros no patamar. 0 quando não montável (grifo iniciante). */
  deslocamentoM: number
  movementType: MovementType
  acoesMovimentoExtras: 0 | 1 | 2
  bonusAtaqueCaC: 0 | 2
  bonusDefesa: 0 | 2
  bonusResistencias: 0 | 1 | 2 | 5
  bonusDanoCaC?: MontariaDanoCaC
  sentidosGanhos?: 'faro'
  manobrasLivres?: 'derrubar'
  ridavel: boolean
  /** Prosa verbatim do PDF por patamar. */
  verbatim: string
}

export type MontariaAliasSize = {
  name: string
  size: MontariaSize
}

export type MontariaEntry = {
  id: MontariaId
  name: string
  tamanho: MontariaSize
  aliasWithSize?: MontariaAliasSize
  /** Preço em T$ (Tabela de equipamentos Cap 4). null = ausente. */
  precoTS: number | null
  notaPreco?: string
  tiers: Readonly<Record<ParceiroTier, MontariaTierData>>
  bookPage: 262
}

// ─── Combate montado — constantes (PDF p261-262) ──────────────────
export const CAVALGAR_CD_GUIAR = 10
export const CAVALGAR_MARGEM_QUEDA = 5
export const CAVALGAR_DANO_QUEDA = '1d6'
export const PENALIDADE_ATAQUE_DISTANCIA_MONTADO = -2
export const MONTAR_ACAO = 'movimento' as const
export const DESMONTAR_ACAO = 'movimento' as const
export const COMPRA_PATAMAR_PADRAO: ParceiroTier = 'iniciante'

/** CD do teste de Cavalgar quando a montaria sofre dano (p261: CD = dano sofrido). */
export function cavalgarCdAposDano(dano: number): number {
  if (dano < 0) {
    throw new Error(`cavalgarCdAposDano: dano must be ≥ 0, got ${dano}`)
  }
  return dano
}

// ─── Catálogo — 6 montarias × 3 patamares ─────────────────────────
const RAW: readonly MontariaEntry[] = [
  {
    id: 'cavalo',
    name: 'Cavalo',
    tamanho: 'grande',
    aliasWithSize: { name: 'Pônei', size: 'medio' },
    precoTS: 75,
    notaPreco:
      'Tabela Cap 4: cavalo T$ 75, cavalo de guerra T$ 400, pônei T$ 5, pônei de guerra T$ 30.',
    tiers: Object.freeze({
      iniciante: {
        deslocamentoM: 12,
        movementType: 'terrestre',
        acoesMovimentoExtras: 1,
        bonusAtaqueCaC: 0,
        bonusDefesa: 0,
        bonusResistencias: 0,
        ridavel: true,
        verbatim:
          'seu deslocamento muda para 12m e você recebe uma ação de movimento extra por turno (apenas para se deslocar)',
      },
      veterano: {
        deslocamentoM: 15,
        movementType: 'terrestre',
        acoesMovimentoExtras: 1,
        bonusAtaqueCaC: 2,
        bonusDefesa: 0,
        bonusResistencias: 0,
        ridavel: true,
        verbatim:
          'como acima, mas seu deslocamento muda para 15m e você recebe +2 em ataques corpo a corpo',
      },
      mestre: {
        deslocamentoM: 15,
        movementType: 'terrestre',
        acoesMovimentoExtras: 2,
        bonusAtaqueCaC: 2,
        bonusDefesa: 0,
        bonusResistencias: 0,
        ridavel: true,
        verbatim:
          'como acima, mas você recebe uma segunda ação de movimento extra por turno (novamente, apenas para se deslocar)',
      },
    }),
    bookPage: 262,
  },
  {
    id: 'cao-de-caca',
    name: 'Cão de Caça',
    tamanho: 'medio',
    aliasWithSize: { name: 'Cão de Caça (Pequeno)', size: 'pequeno' },
    precoTS: 150,
    tiers: Object.freeze({
      iniciante: {
        deslocamentoM: 9,
        movementType: 'terrestre',
        acoesMovimentoExtras: 1,
        bonusAtaqueCaC: 0,
        bonusDefesa: 0,
        bonusResistencias: 0,
        sentidosGanhos: 'faro',
        ridavel: true,
        verbatim:
          'seu deslocamento muda para 9m, você pode usar faro e recebe uma ação de movimento extra por turno (apenas para se deslocar)',
      },
      veterano: {
        deslocamentoM: 12,
        movementType: 'terrestre',
        acoesMovimentoExtras: 1,
        bonusAtaqueCaC: 0,
        bonusDefesa: 2,
        bonusResistencias: 0,
        sentidosGanhos: 'faro',
        ridavel: true,
        verbatim:
          'como acima, mas seu deslocamento muda para 12m e você recebe +2 na Defesa',
      },
      mestre: {
        deslocamentoM: 12,
        movementType: 'terrestre',
        acoesMovimentoExtras: 1,
        bonusAtaqueCaC: 0,
        bonusDefesa: 2,
        bonusResistencias: 0,
        sentidosGanhos: 'faro',
        manobrasLivres: 'derrubar',
        ridavel: true,
        verbatim:
          'como acima; além disso, uma vez por rodada, quando acerta um ataque corpo a corpo, você pode fazer a manobra derrubar como uma ação livre',
      },
    }),
    bookPage: 262,
  },
  {
    id: 'lobo-das-cavernas',
    name: 'Lobo-das-cavernas',
    tamanho: 'grande',
    aliasWithSize: { name: 'Lobo comum', size: 'medio' },
    precoTS: null,
    notaPreco: 'Não consta na tabela de equipamentos; obtido por captura ou habilidade de classe.',
    tiers: Object.freeze({
      iniciante: {
        deslocamentoM: 12,
        movementType: 'terrestre',
        acoesMovimentoExtras: 1,
        bonusAtaqueCaC: 0,
        bonusDefesa: 0,
        bonusResistencias: 0,
        ridavel: true,
        verbatim:
          'seu deslocamento muda para 12m e você recebe uma ação de movimento extra por turno (apenas para se deslocar)',
      },
      veterano: {
        deslocamentoM: 15,
        movementType: 'terrestre',
        acoesMovimentoExtras: 1,
        bonusAtaqueCaC: 0,
        bonusDefesa: 0,
        bonusResistencias: 0,
        bonusDanoCaC: { die: '+1d8', timesPerRound: 1 },
        ridavel: true,
        verbatim:
          'como acima, mas seu deslocamento muda para 15m, uma vez por rodada, você recebe +1d8 em uma rolagem de dano corpo a corpo',
      },
      mestre: {
        deslocamentoM: 15,
        movementType: 'terrestre',
        acoesMovimentoExtras: 1,
        bonusAtaqueCaC: 0,
        bonusDefesa: 0,
        bonusResistencias: 0,
        bonusDanoCaC: { die: '+1d8', timesPerRound: 1 },
        manobrasLivres: 'derrubar',
        ridavel: true,
        verbatim:
          'como acima; além disso, uma vez por rodada, quando acerta um ataque corpo a corpo, você pode fazer a manobra derrubar como uma ação livre',
      },
    }),
    bookPage: 262,
  },
  {
    id: 'grifo',
    name: 'Grifo',
    tamanho: 'grande',
    precoTS: null,
    notaPreco: 'Encontrar um grifo à venda é quase impossível!',
    tiers: Object.freeze({
      iniciante: {
        deslocamentoM: 0,
        movementType: 'terrestre',
        acoesMovimentoExtras: 0,
        bonusAtaqueCaC: 0,
        bonusDefesa: 0,
        bonusResistencias: 0,
        bonusDanoCaC: { die: '+1d8', timesPerRound: 1 },
        ridavel: false,
        verbatim:
          'uma vez por rodada, você recebe +1d8 em uma rolagem de dano corpo a corpo (um grifo iniciante é um filhote e não pode ser usado como montaria)',
      },
      veterano: {
        deslocamentoM: 18,
        movementType: 'voo',
        acoesMovimentoExtras: 0,
        bonusAtaqueCaC: 0,
        bonusDefesa: 0,
        bonusResistencias: 0,
        bonusDanoCaC: { die: '+1d8', timesPerRound: 1 },
        ridavel: true,
        verbatim:
          'como acima, mas pode ser usado como montaria, mudando seu deslocamento para voo 18m',
      },
      mestre: {
        deslocamentoM: 18,
        movementType: 'voo',
        acoesMovimentoExtras: 1,
        bonusAtaqueCaC: 0,
        bonusDefesa: 0,
        bonusResistencias: 0,
        bonusDanoCaC: { die: '+1d8', timesPerRound: 1 },
        ridavel: true,
        verbatim:
          'como acima, mas você recebe uma ação de movimento extra por turno (apenas para se deslocar)',
      },
    }),
    bookPage: 262,
  },
  {
    id: 'gorlogg',
    name: 'Gorlogg',
    tamanho: 'grande',
    precoTS: null,
    notaPreco: 'Não consta na tabela de equipamentos; obtido por captura ou habilidade de classe.',
    tiers: Object.freeze({
      iniciante: {
        deslocamentoM: 12,
        movementType: 'terrestre',
        acoesMovimentoExtras: 0,
        bonusAtaqueCaC: 0,
        bonusDefesa: 0,
        bonusResistencias: 0,
        bonusDanoCaC: { die: '+1d6', timesPerRound: 1 },
        ridavel: true,
        verbatim:
          'seu deslocamento muda para 12m e, uma vez por rodada, você recebe +1d6 em uma rolagem de dano corpo a corpo',
      },
      veterano: {
        deslocamentoM: 12,
        movementType: 'terrestre',
        acoesMovimentoExtras: 0,
        bonusAtaqueCaC: 0,
        bonusDefesa: 0,
        bonusResistencias: 0,
        bonusDanoCaC: { die: '+1d10', timesPerRound: 1 },
        ridavel: true,
        verbatim:
          'como acima, mas o bônus em rolagens de dano corpo a corpo muda para +1d10',
      },
      mestre: {
        deslocamentoM: 15,
        movementType: 'terrestre',
        acoesMovimentoExtras: 0,
        bonusAtaqueCaC: 0,
        bonusDefesa: 0,
        bonusResistencias: 0,
        bonusDanoCaC: { die: '+2d8', timesPerRound: 1 },
        ridavel: true,
        verbatim:
          'seu deslocamento muda para 15m e o bônus em rolagens de dano corpo a corpo muda para +2d8',
      },
    }),
    bookPage: 262,
  },
  {
    id: 'trobo',
    name: 'Trobo',
    tamanho: 'grande',
    precoTS: 60,
    tiers: Object.freeze({
      iniciante: {
        deslocamentoM: 9,
        movementType: 'terrestre',
        acoesMovimentoExtras: 1,
        bonusAtaqueCaC: 0,
        bonusDefesa: 0,
        bonusResistencias: 1,
        ridavel: true,
        verbatim:
          'seu deslocamento muda para 9m e você recebe uma ação de movimento extra por turno (apenas para se deslocar) e +1 em testes de resistência',
      },
      veterano: {
        deslocamentoM: 12,
        movementType: 'terrestre',
        acoesMovimentoExtras: 1,
        bonusAtaqueCaC: 0,
        bonusDefesa: 0,
        bonusResistencias: 2,
        ridavel: true,
        verbatim:
          'como acima, mas seu deslocamento muda para 12m e o bônus em testes de resistência muda para +2',
      },
      mestre: {
        deslocamentoM: 12,
        movementType: 'terrestre',
        acoesMovimentoExtras: 1,
        bonusAtaqueCaC: 0,
        bonusDefesa: 0,
        bonusResistencias: 5,
        ridavel: true,
        verbatim:
          'como acima, mas o bônus em testes de resistência muda para +5',
      },
    }),
    bookPage: 262,
  },
]

export const MONTARIA_CATALOG: readonly MontariaEntry[] = Object.freeze(RAW)

export function montariaById(id: MontariaId): MontariaEntry | undefined {
  return MONTARIA_CATALOG.find((m) => m.id === id)
}

export function montariaTier(
  id: MontariaId,
  tier: ParceiroTier,
): MontariaTierData | undefined {
  return montariaById(id)?.tiers[tier]
}
