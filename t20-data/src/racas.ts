import type { AttributeKey } from './attributes'

/**
 * Raças — PDF book p18-31 (Capítulo 1).
 *
 * T20 ships 17 raças in two tiers:
 *  - **Comuns** (p18-26): Humano, Anão, Dahllan, Elfo, Goblin, Lefou,
 *    Minotauro, Qareen.
 *  - **Extras** (p27-31): Golem, Hynne, Kliren, Medusa, Osteon,
 *    Sereia/Tritão, Sílfide, Suraggel, Trog.
 *
 * Defaults applied when omitted: `tamanho='Médio'`, `deslocamento=9`,
 * `visaoNoEscuro=false`, `visaoNaPenumbra=false`.
 *
 * Atributo modifiers come in 3 shapes:
 *  - **fixed**: a static set (e.g. Anão Con +2 / Sab +1 / Des −1).
 *  - **floating**: pick N atributos to receive +1; optionally with a
 *    forbidden attribute and/or a static penalty (Humano / Lefou /
 *    Osteon / Sereia).
 *  - **subraca-gated**: variants per ascendência (only Suraggel).
 */

export type Tamanho =
  | 'Minúsculo'
  | 'Pequeno'
  | 'Médio'
  | 'Grande'
  | 'Enorme'
  | 'Colossal'

export type AtributoMod =
  | { kind: 'fixed'; mods: Partial<Record<AttributeKey, number>> }
  | {
      kind: 'floating'
      count: number
      value: number
      exclude?: AttributeKey
      penalty?: { attribute: AttributeKey; value: number }
    }
  | {
      kind: 'subraca-gated'
      variants: Readonly<Record<string, Partial<Record<AttributeKey, number>>>>
    }

export type RacaAbility = {
  name: string
  summary: string
}

export type Raca = {
  id: string
  name: string
  tier: 'comum' | 'extra'
  atributoMod: AtributoMod
  tamanho: Tamanho
  deslocamento: number
  visaoNoEscuro: boolean
  visaoNaPenumbra: boolean
  abilities: readonly RacaAbility[]
  /** Ascendência options (Qareen elemental, Golem fonte, Suraggel Aggelus/Sulfure). */
  ascendencias?: readonly string[]
  bookPage: number
}

const RACAS_LIST: readonly Raca[] = [
  {
    id: 'humano',
    name: 'Humano',
    tier: 'comum',
    atributoMod: { kind: 'floating', count: 3, value: 1 },
    tamanho: 'Médio',
    deslocamento: 9,
    visaoNoEscuro: false,
    visaoNaPenumbra: false,
    abilities: [
      { name: '+1 em Três Atributos Diferentes', summary: 'Filhos de Valkaria escolhem livremente onde colocar três +1.' },
      { name: 'Versátil', summary: 'Treinado em 2 perícias livres; pode trocar uma por um poder geral.' },
    ],
    bookPage: 19,
  },
  {
    id: 'anao',
    name: 'Anão',
    tier: 'comum',
    atributoMod: { kind: 'fixed', mods: { constitution: 2, wisdom: 1, dexterity: -1 } },
    tamanho: 'Médio',
    deslocamento: 6,
    visaoNoEscuro: true,
    visaoNaPenumbra: false,
    abilities: [
      { name: 'Conhecimento das Rochas', summary: 'Visão no escuro + +2 em Percepção e Sobrevivência no subterrâneo.' },
      { name: 'Devagar e Sempre', summary: 'Deslocamento 6m, imune a redução por armadura ou carga.' },
      { name: 'Duro como Pedra', summary: '+3 PV no 1º nível e +1 PV por nível seguinte.' },
      { name: 'Tradição de Heredrimm', summary: 'Machados/martelos/marretas/picaretas viram armas simples; +2 nos ataques com elas.' },
    ],
    bookPage: 20,
  },
  {
    id: 'dahllan',
    name: 'Dahllan',
    tier: 'comum',
    atributoMod: { kind: 'fixed', mods: { wisdom: 2, dexterity: 1, intelligence: -1 } },
    tamanho: 'Médio',
    deslocamento: 9,
    visaoNoEscuro: false,
    visaoNaPenumbra: false,
    abilities: [
      { name: 'Amiga das Plantas', summary: 'Lança Controlar Plantas (atrib-chave Sab); −1 PM se aprender de novo.' },
      { name: 'Armadura de Allihanna', summary: 'Ação de movimento + 1 PM: pele vira casca, +2 Defesa até fim da cena.' },
      { name: 'Empatia Selvagem', summary: 'Comunica-se com animais; usa Adestramento para persuasão; +2 em Adestramento se aprender de novo.' },
    ],
    bookPage: 21,
  },
  {
    id: 'elfo',
    name: 'Elfo',
    tier: 'comum',
    atributoMod: { kind: 'fixed', mods: { intelligence: 2, dexterity: 1, constitution: -1 } },
    tamanho: 'Médio',
    deslocamento: 12,
    visaoNoEscuro: false,
    visaoNaPenumbra: true,
    abilities: [
      { name: 'Graça de Glórienn', summary: 'Deslocamento 12m em vez de 9m.' },
      { name: 'Sangue Mágico', summary: '+1 PM por nível.' },
      { name: 'Sentidos Élficos', summary: 'Visão na penumbra + +2 em Misticismo e Percepção.' },
    ],
    bookPage: 22,
  },
  {
    id: 'goblin',
    name: 'Goblin',
    tier: 'comum',
    atributoMod: { kind: 'fixed', mods: { dexterity: 2, intelligence: 1, charisma: -1 } },
    tamanho: 'Pequeno',
    deslocamento: 9,
    visaoNoEscuro: true,
    visaoNaPenumbra: false,
    abilities: [
      { name: 'Engenhoso', summary: 'Sem penalidade por não usar ferramenta; +2 se usar a correta.' },
      { name: 'Espelunqueiro', summary: 'Visão no escuro + deslocamento de escalada igual ao terrestre.' },
      { name: 'Peste Esguia', summary: 'Pequeno mas deslocamento permanece 9m.' },
      { name: 'Rato das Ruas', summary: '+2 em Fortitude; recuperação de PV/PM nunca inferior ao nível.' },
    ],
    bookPage: 23,
  },
  {
    id: 'lefou',
    name: 'Lefou',
    tier: 'comum',
    atributoMod: {
      kind: 'floating',
      count: 3,
      value: 1,
      exclude: 'charisma',
      penalty: { attribute: 'charisma', value: -1 },
    },
    tamanho: 'Médio',
    deslocamento: 9,
    visaoNoEscuro: false,
    visaoNaPenumbra: false,
    abilities: [
      { name: 'Cria da Tormenta', summary: 'Tipo monstro; +5 em testes de resistência contra efeitos de lefeu e Tormenta.' },
      { name: 'Deformidade', summary: '+2 em duas perícias livres (contam como poderes da Tormenta); pode trocar uma por um poder da Tormenta livre.' },
    ],
    bookPage: 24,
  },
  {
    id: 'minotauro',
    name: 'Minotauro',
    tier: 'comum',
    atributoMod: { kind: 'fixed', mods: { strength: 2, constitution: 1, wisdom: -1 } },
    tamanho: 'Médio',
    deslocamento: 9,
    visaoNoEscuro: false,
    visaoNaPenumbra: false,
    abilities: [
      { name: 'Chifres', summary: 'Arma natural 1d6/x2/perfuração; 1×/rodada na ação agredir, 1 PM dá ataque extra.' },
      { name: 'Couro Rígido', summary: '+1 na Defesa.' },
      { name: 'Faro', summary: 'Em alcance curto não percebido não fica desprevenido; camuflagem total dá só 20% de falha.' },
      { name: 'Medo de Altura', summary: 'Adjacente a queda de 3m+: fica abalado.' },
    ],
    bookPage: 25,
  },
  {
    id: 'qareen',
    name: 'Qareen',
    tier: 'comum',
    atributoMod: { kind: 'fixed', mods: { charisma: 2, intelligence: 1, wisdom: -1 } },
    tamanho: 'Médio',
    deslocamento: 9,
    visaoNoEscuro: false,
    visaoNaPenumbra: false,
    ascendencias: ['água', 'ar', 'fogo', 'terra', 'luz', 'trevas'],
    abilities: [
      { name: 'Desejos', summary: 'Ao lançar magia pedida desde último turno: −1 PM. Fazer pedido é ação livre.' },
      { name: 'Resistência Elemental', summary: 'Redução 10 a um tipo (frio/eletricidade/fogo/ácido/luz/trevas conforme ascendência).' },
      { name: 'Tatuagem Mística', summary: 'Lança uma magia de 1º círculo livre (atrib-chave Car); −1 PM se aprender de novo.' },
    ],
    bookPage: 26,
  },
  {
    id: 'golem',
    name: 'Golem',
    tier: 'extra',
    atributoMod: { kind: 'fixed', mods: { strength: 2, constitution: 1, charisma: -1 } },
    tamanho: 'Médio',
    deslocamento: 6,
    visaoNoEscuro: true,
    visaoNaPenumbra: false,
    ascendencias: ['água', 'ar', 'fogo', 'terra'],
    abilities: [
      { name: 'Chassi', summary: 'Desloc 6m sem redução; +2 Defesa, −2 penalidade; vestir/remover armadura leva 1 dia; armadura acoplada não conta no limite.' },
      { name: 'Criatura Artificial', summary: 'Construto; visão no escuro; imune a cansaço/metabólicos/veneno; não respira/come/dorme; recarrega 8h; Ofício (artesão) substitui Cura.' },
      { name: 'Fonte Elemental', summary: 'Escolhe água/ar/fogo/terra: imune a esse tipo; dano mágico desse tipo cura metade.' },
      { name: 'Propósito de Criação', summary: 'Sem origem; recebe um poder geral livre.' },
    ],
    bookPage: 27,
  },
  {
    id: 'hynne',
    name: 'Hynne',
    tier: 'extra',
    atributoMod: { kind: 'fixed', mods: { dexterity: 2, charisma: 1, strength: -1 } },
    tamanho: 'Pequeno',
    deslocamento: 6,
    visaoNoEscuro: false,
    visaoNaPenumbra: false,
    abilities: [
      { name: 'Arremessador', summary: 'Dano de funda/arma de arremesso aumenta um passo.' },
      { name: 'Pequeno e Rechonchudo', summary: 'Pequeno, desloc 6m; +2 em Enganação; pode usar Des em Atletismo.' },
      { name: 'Sorte Salvadora', summary: 'Em teste de resistência: gasta 1 PM para rolar de novo.' },
    ],
    bookPage: 27,
  },
  {
    id: 'kliren',
    name: 'Kliren',
    tier: 'extra',
    atributoMod: { kind: 'fixed', mods: { intelligence: 2, charisma: 1, strength: -1 } },
    tamanho: 'Médio',
    deslocamento: 9,
    visaoNoEscuro: false,
    visaoNaPenumbra: false,
    abilities: [
      { name: 'Híbrido', summary: 'Treinado em uma perícia livre.' },
      { name: 'Engenhosidade', summary: 'Gasta 2 PM para somar Int em teste de perícia; −1 PM se aprender de novo.' },
      { name: 'Ossos Frágeis', summary: '+1 ponto de dano por dado de dano de impacto sofrido.' },
      { name: 'Vanguardista', summary: 'Proficiência em armas de fogo + +2 em um Ofício livre.' },
    ],
    bookPage: 28,
  },
  {
    id: 'medusa',
    name: 'Medusa',
    tier: 'extra',
    atributoMod: { kind: 'fixed', mods: { dexterity: 2, charisma: 1 } },
    tamanho: 'Médio',
    deslocamento: 9,
    visaoNoEscuro: true,
    visaoNaPenumbra: false,
    abilities: [
      { name: 'Cria de Megalokk', summary: 'Tipo monstro; visão no escuro.' },
      { name: 'Natureza Venenosa', summary: 'Resistência a veneno +5; ação mov + 1 PM envenena arma (1d12 perda de PV, dura até acertar/fim da cena).' },
      { name: 'Olhar Atordoante', summary: 'Ação mov + 1 PM: alvo em curto faz Fortitude (CD Car); falha = atordoado 1 rodada (1×/cena).' },
    ],
    bookPage: 28,
  },
  {
    id: 'osteon',
    name: 'Osteon',
    tier: 'extra',
    atributoMod: {
      kind: 'floating',
      count: 3,
      value: 1,
      exclude: 'constitution',
      penalty: { attribute: 'constitution', value: -1 },
    },
    tamanho: 'Médio',
    deslocamento: 9,
    visaoNoEscuro: true,
    visaoNaPenumbra: false,
    abilities: [
      { name: 'Armadura Óssea', summary: 'Redução de corte/frio/perfuração 5.' },
      { name: 'Memória Póstuma', summary: 'Treinado em uma perícia OU um poder geral OU herda uma habilidade + tamanho de outra raça humanoide (exceto humano).' },
      { name: 'Natureza Esquelética', summary: 'Morto-vivo; visão no escuro; imune a cansaço/metabólicos/trevas/veneno; cura de luz fere; dano de trevas cura.' },
      { name: 'Preço da Não-Vida', summary: 'Descansa só 8h sob estrelas ou subterrâneo; senão sofre fome.' },
    ],
    bookPage: 29,
  },
  {
    id: 'sereia-tritao',
    name: 'Sereia/Tritão',
    tier: 'extra',
    atributoMod: { kind: 'floating', count: 3, value: 1 },
    tamanho: 'Médio',
    deslocamento: 9,
    visaoNoEscuro: false,
    visaoNaPenumbra: false,
    abilities: [
      { name: 'Canção dos Mares', summary: 'Lança 2 dentre Amedrontar/Comando/Despedaçar/Enfeitiçar/Hipnotismo/Sono (atrib-chave Car); −1 PM se aprender de novo.' },
      { name: 'Mestre do Tridente', summary: 'Tridente é arma simples; +2 em dano com azagaias/lanças/tridentes.' },
      { name: 'Transformação Anfíbia', summary: 'Respira embaixo d\'água; natação 12m com cauda; fora d\'água ganha pernas (9m); >1 dia sem água: não recupera PM.' },
    ],
    bookPage: 30,
  },
  {
    id: 'silfide',
    name: 'Sílfide',
    tier: 'extra',
    atributoMod: { kind: 'fixed', mods: { charisma: 2, dexterity: 1, strength: -2 } },
    tamanho: 'Minúsculo',
    deslocamento: 9,
    visaoNoEscuro: false,
    visaoNaPenumbra: true,
    abilities: [
      { name: 'Asas de Borboleta', summary: 'Minúsculo; paira a 1,5m; ignora terreno difícil; imune a dano por queda; 1 PM/rodada: voo 12m.' },
      { name: 'Espírito da Natureza', summary: 'Espírito; visão na penumbra; fala com animais livremente.' },
      { name: 'Magia das Fadas', summary: 'Lança 2 dentre Criar Ilusão/Enfeitiçar/Luz (arcana)/Sono (atrib-chave Car); −1 PM se aprender de novo.' },
    ],
    bookPage: 30,
  },
  {
    id: 'suraggel',
    name: 'Suraggel',
    tier: 'extra',
    atributoMod: {
      kind: 'subraca-gated',
      variants: {
        aggelus: { wisdom: 2, charisma: 1 },
        sulfure: { dexterity: 2, intelligence: 1 },
      },
    },
    tamanho: 'Médio',
    deslocamento: 9,
    visaoNoEscuro: true,
    visaoNaPenumbra: false,
    ascendencias: ['aggelus', 'sulfure'],
    abilities: [
      { name: 'Herança Divina', summary: 'Espírito; visão no escuro.' },
      { name: 'Luz Sagrada (Aggelus)', summary: '+2 em Diplomacia/Intuição; lança Luz (divina, atrib-chave Car); −1 PM se aprender de novo.' },
      { name: 'Sombras Profanas (Sulfure)', summary: '+2 em Enganação/Furtividade; lança Escuridão (divina, atrib-chave Int); −1 PM se aprender de novo.' },
    ],
    bookPage: 30,
  },
  {
    id: 'trog',
    name: 'Trog',
    tier: 'extra',
    atributoMod: { kind: 'fixed', mods: { constitution: 2, strength: 1, intelligence: -1 } },
    tamanho: 'Médio',
    deslocamento: 9,
    visaoNoEscuro: true,
    visaoNaPenumbra: false,
    abilities: [
      { name: 'Mau Cheiro', summary: 'Ação padrão + 2 PM: outras criaturas em curto fazem Fortitude vs veneno (CD Con); falha = enjoadas 1d6 rodadas.' },
      { name: 'Mordida', summary: 'Arma natural 1d6/x2/perfuração; 1×/rodada na ação agredir, 1 PM dá ataque corpo a corpo extra.' },
      { name: 'Reptiliano', summary: 'Monstro; visão no escuro; +1 Defesa; +5 em Furtividade sem armadura/roupas pesadas.' },
      { name: 'Sangue Frio', summary: '+1 ponto de dano adicional por dado de dano de frio sofrido.' },
    ],
    bookPage: 31,
  },
]

export const RACAS: Readonly<Record<string, Raca>> = Object.freeze(
  RACAS_LIST.reduce<Record<string, Raca>>((acc, r) => {
    acc[r.id] = r
    return acc
  }, {}),
)

export const RACA_IDS: readonly string[] = RACAS_LIST.map((r) => r.id)

export function racaById(id: string): Raca {
  const r = RACAS[id]
  if (!r) throw new Error(`racaById: unknown raça id "${id}"`)
  return r
}

export function racasByTier(tier: 'comum' | 'extra'): readonly Raca[] {
  return RACAS_LIST.filter((r) => r.tier === tier)
}

/**
 * Resolve a raça's atributo modifiers into a flat `Partial<Record<AttributeKey, number>>`.
 *
 * - `fixed`: returns the static map.
 * - `floating`: requires `floatingPicks` — exactly `count` distinct
 *   attributes, each given `value`. Validates exclusion.
 * - `subraca-gated`: requires `ascendencia` matching a variant key.
 */
export function resolveAtributoMod(
  raca: Raca,
  opts: {
    floatingPicks?: readonly AttributeKey[]
    ascendencia?: string
  } = {},
): Partial<Record<AttributeKey, number>> {
  const mod = raca.atributoMod
  if (mod.kind === 'fixed') return { ...mod.mods }

  if (mod.kind === 'floating') {
    const picks = opts.floatingPicks ?? []
    if (picks.length !== mod.count) {
      throw new Error(
        `resolveAtributoMod: ${raca.name} requires exactly ${mod.count} floating picks, got ${picks.length}`,
      )
    }
    if (new Set(picks).size !== picks.length) {
      throw new Error(`resolveAtributoMod: ${raca.name} floating picks must be distinct`)
    }
    if (mod.exclude && picks.includes(mod.exclude)) {
      throw new Error(
        `resolveAtributoMod: ${raca.name} cannot place +${mod.value} in ${mod.exclude}`,
      )
    }
    const result: Partial<Record<AttributeKey, number>> = {}
    for (const a of picks) result[a] = mod.value
    if (mod.penalty) result[mod.penalty.attribute] = mod.penalty.value
    return result
  }

  // subraca-gated
  if (!opts.ascendencia || !mod.variants[opts.ascendencia]) {
    const keys = Object.keys(mod.variants).join(', ')
    throw new Error(
      `resolveAtributoMod: ${raca.name} requires ascendência in [${keys}], got ${opts.ascendencia}`,
    )
  }
  return { ...mod.variants[opts.ascendencia] }
}
