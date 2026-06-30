/**
 * Bestiário — PDF Cap 7 (Ameaças), livro p282-323.
 *
 * Subset de 20 monstros cobrindo NDs 1/4 → 20. Selecionados para
 * representar todos os tipos principais (humanoide, animal, monstro,
 * morto-vivo, construto) e tamanhos (pequeno → colossal).
 *
 * `treasureXp` segue a regra de PDF Cap 8 (Recompensas, p326):
 * `XP = ND × 1.000` (clamp dos ND fracionados em 250/500).
 *
 * NÃO incluídos (não constam no core T20): Lich, Cubo Gelatinoso,
 * Beholder, Minotauro (raça PJ apenas), Urso comum. "Dragão" aparece
 * em múltiplos NDs (filhote/jovem/adulto/rei) — selecionei 4 idades.
 */

export type MonsterTipo =
  | 'humanoide'
  | 'animal'
  | 'monstro'
  | 'morto-vivo'
  | 'construto'
  | 'espirito'
  | 'planar'

export type MonsterSize =
  | 'minusculo'
  | 'pequeno'
  | 'medio'
  | 'grande'
  | 'enorme'
  | 'colossal'

export type MonsterAttack = {
  name: string
  attackBonus: number
  damage: string
  special?: string
}

export type Monster = {
  id: string
  name: string
  nd: number
  tipo: MonsterTipo
  size: MonsterSize
  hp: number
  defesa: number
  forca: number
  destreza: number
  constituicao: number
  inteligencia: number
  sabedoria: number
  carisma: number
  fortitude: number
  reflexos: number
  vontade: number
  deslocamento: string
  attacks: readonly MonsterAttack[]
  specialAbilities: readonly string[]
  treasureXp: number
  bookPage: number
}

/** XP de tesouro derivado do ND (PDF Cap 8 p326). */
export function xpForNd(nd: number): number {
  return Math.round(nd * 1000)
}

export const BESTIARY: readonly Monster[] = Object.freeze([
  {
    id: 'goblin-salteador',
    name: 'Goblin Salteador',
    nd: 0.25,
    tipo: 'humanoide',
    size: 'pequeno',
    hp: 4,
    defesa: 13,
    forca: 0, destreza: 3, constituicao: 0,
    inteligencia: 0, sabedoria: -1, carisma: -1,
    fortitude: 2, reflexos: 3, vontade: -1,
    deslocamento: '9m, escalada 9m',
    attacks: [{ name: 'Duas adagas', attackBonus: 7, damage: '1d4', special: 'crítico 19' }],
    specialAbilities: [
      'Frenesi: +2 em ataque e dano por cada outro goblin salteador adjacente ao alvo.',
      'Visão no escuro.',
    ],
    treasureXp: 250,
    bookPage: 300,
  },
  {
    id: 'zumbi',
    name: 'Zumbi',
    nd: 0.25,
    tipo: 'morto-vivo',
    size: 'medio',
    hp: 20,
    defesa: 11,
    forca: 3, destreza: -1, constituicao: 2,
    inteligencia: 0, sabedoria: -1, carisma: 0,
    fortitude: 3, reflexos: -1, vontade: -1,
    deslocamento: '6m',
    attacks: [{ name: 'Mordida', attackBonus: 7, damage: '1d6+6' }],
    specialAbilities: [
      'Fraqueza Zumbi: dobro de dano em críticos ou ataques contra cérebro (Defesa 21).',
      'Visão no escuro; imunidades de morto-vivo (cansaço/metabólicos/trevas/veneno).',
    ],
    treasureXp: 250,
    bookPage: 297,
  },
  {
    id: 'orc-combatente',
    name: 'Orc Combatente',
    nd: 0.5,
    tipo: 'humanoide',
    size: 'medio',
    hp: 8,
    defesa: 14,
    forca: 4, destreza: 1, constituicao: 2,
    inteligencia: -1, sabedoria: -1, carisma: -1,
    fortitude: 5, reflexos: 3, vontade: 0,
    deslocamento: '9m',
    attacks: [{ name: 'Maça', attackBonus: 9, damage: '1d8+7' }],
    specialAbilities: ['Sensibilidade a Luz: ofuscado sob luz solar.', 'Visão no escuro.'],
    treasureXp: 500,
    bookPage: 286,
  },
  {
    id: 'lobo',
    name: 'Lobo',
    nd: 0.5,
    tipo: 'animal',
    size: 'medio',
    hp: 14,
    defesa: 14,
    forca: 3, destreza: 3, constituicao: 3,
    inteligencia: -4, sabedoria: 2, carisma: -2,
    fortitude: 6, reflexos: 3, vontade: 1,
    deslocamento: '15m',
    attacks: [
      { name: 'Mordida', attackBonus: 7, damage: '1d6+5', special: 'derrubar (teste +7) em acerto' },
    ],
    specialAbilities: [
      'Táticas de Alcateia: +2 ataque/+2 dano ao flanquear (total +4/+2).',
      'Faro, visão na penumbra.',
    ],
    treasureXp: 500,
    bookPage: 289,
  },
  {
    id: 'esqueleto',
    name: 'Esqueleto',
    nd: 2,
    tipo: 'morto-vivo',
    size: 'medio',
    hp: 14,
    defesa: 19,
    forca: 5, destreza: 3, constituicao: 0,
    inteligencia: 0, sabedoria: 0, carisma: -5,
    fortitude: 3, reflexos: 7, vontade: 12,
    deslocamento: '9m',
    attacks: [{ name: 'Espada longa', attackBonus: 14, damage: '2d8+12', special: 'crítico 19' }],
    specialAbilities: [
      'Redução de corte, frio e perfuração 5.',
      'Visão no escuro; imunidades de morto-vivo.',
      'Sem inteligência: segue ordens do criador.',
    ],
    treasureXp: 2000,
    bookPage: 297,
  },
  {
    id: 'aranha-gigante',
    name: 'Aranha Gigante',
    nd: 2,
    tipo: 'monstro',
    size: 'grande',
    hp: 77,
    defesa: 19,
    forca: 5, destreza: 4, constituicao: 1,
    inteligencia: -5, sabedoria: 0, carisma: -4,
    fortitude: 8, reflexos: 11, vontade: 3,
    deslocamento: '12m, escalar 12m',
    attacks: [
      {
        name: 'Mordida',
        attackBonus: 12,
        damage: '2d6+8 mais veneno',
        special: 'Veneno: fraco (Fort CD 18 evita)',
      },
    ],
    specialAbilities: [
      'Teia (padrão): teia em quadrado 3m a curto, enredando (Ref CD 18 evita). Soltar = ação completa + Força/Acrobacia CD 20.',
      'Visão no escuro.',
    ],
    treasureXp: 2000,
    bookPage: 286,
  },
  {
    id: 'gargula',
    name: 'Gárgula',
    nd: 2,
    tipo: 'construto',
    size: 'medio',
    hp: 65,
    defesa: 19,
    forca: 6, destreza: 2, constituicao: 4,
    inteligencia: -2, sabedoria: 1, carisma: -2,
    fortitude: 13, reflexos: 7, vontade: 2,
    deslocamento: '12m, voo 18m',
    attacks: [{ name: 'Duas garras', attackBonus: 12, damage: '1d6+6' }],
    specialAbilities: [
      'Imobilidade: estática parece estátua (Percepção CD 35).',
      'Redução de dano 5; imune a condição petrificado.',
      'Visão no escuro; imunidades de construto.',
    ],
    treasureXp: 2000,
    bookPage: 286,
  },
  {
    id: 'dragao-filhote',
    name: 'Dragão Filhote',
    nd: 3,
    tipo: 'monstro',
    size: 'medio',
    hp: 140,
    defesa: 22,
    forca: 4, destreza: 3, constituicao: 3,
    inteligencia: 0, sabedoria: 0, carisma: 0,
    fortitude: 15, reflexos: 3, vontade: 9,
    deslocamento: '12m, voo 18m',
    attacks: [
      { name: 'Mordida', attackBonus: 15, damage: '2d6+5' },
      { name: 'Duas garras', attackBonus: 15, damage: '1d6+5' },
    ],
    specialAbilities: [
      'Sopro (padrão, recarga mov): cone 6m, 2d12 fogo + em chamas (Ref CD 18 reduz à metade e evita).',
      'Imune a fogo; vulnerável a frio; resistência a magia +1.',
      'Visão no escuro, percepção às cegas.',
    ],
    treasureXp: 3000,
    bookPage: 310,
  },
  {
    id: 'grifo',
    name: 'Grifo',
    nd: 3,
    tipo: 'monstro',
    size: 'grande',
    hp: 110,
    defesa: 19,
    forca: 5, destreza: 4, constituicao: 3,
    inteligencia: -4, sabedoria: 2, carisma: -1,
    fortitude: 9, reflexos: 15, vontade: 4,
    deslocamento: '12m, voo 24m',
    attacks: [
      { name: 'Mordida', attackBonus: 14, damage: '2d6+5' },
      { name: 'Duas garras', attackBonus: 14, damage: '1d6+5' },
    ],
    specialAbilities: [
      'Bote (completa): investida + mordida + duas garras contra mesmo alvo (+2 da investida).',
      'Imune a medo; visão no escuro.',
    ],
    treasureXp: 3000,
    bookPage: 292,
  },
  {
    id: 'basilisco',
    name: 'Basilisco',
    nd: 4,
    tipo: 'monstro',
    size: 'medio',
    hp: 145,
    defesa: 23,
    forca: 4, destreza: 2, constituicao: 4,
    inteligencia: -4, sabedoria: 1, carisma: 0,
    fortitude: 10, reflexos: 9, vontade: 9,
    deslocamento: '9m, natação 9m',
    attacks: [
      {
        name: 'Mordida',
        attackBonus: 16,
        damage: '2d8+12 mais veneno',
        special: 'Peçonha Concentrada (1d12 PV/rod por 3 rod; Fort CD 18 reduz para 1 rod).',
      },
    ],
    specialAbilities: [
      'Olhar Petrificante: criaturas em curto fazem Ref CD 18; falha = lento; já lento = petrificado permanente. Fechar olhos como reação imuniza por uma rodada (mas cego nessa rodada).',
      'Redução de dano 5; visão no escuro.',
    ],
    treasureXp: 4000,
    bookPage: 293,
  },
  {
    id: 'ogro',
    name: 'Ogro',
    nd: 4,
    tipo: 'humanoide',
    size: 'grande',
    hp: 130,
    defesa: 23,
    forca: 7, destreza: 0, constituicao: 4,
    inteligencia: -3, sabedoria: -2, carisma: -2,
    fortitude: 16, reflexos: 10, vontade: 0,
    deslocamento: '9m',
    attacks: [{ name: 'Tacape', attackBonus: 16, damage: '1d12+18' }],
    specialAbilities: [
      'Burro Demais...: -5 em Intuição e Vontade (já contabilizado).',
      '...Para Morrer!: dano de corte, impacto e perfuração reduzido à metade.',
      'Visão na penumbra.',
    ],
    treasureXp: 4000,
    bookPage: 293,
  },
  {
    id: 'urso-coruja',
    name: 'Urso-Coruja',
    nd: 4,
    tipo: 'monstro',
    size: 'grande',
    hp: 145,
    defesa: 23,
    forca: 7, destreza: 3, constituicao: 5,
    inteligencia: -4, sabedoria: 1, carisma: -2,
    fortitude: 16, reflexos: 10, vontade: 5,
    deslocamento: '12m',
    attacks: [
      { name: 'Mordida', attackBonus: 16, damage: '1d8+5' },
      { name: 'Duas garras', attackBonus: 15, damage: '1d6+5' },
    ],
    specialAbilities: [
      'Agarrar Aprimorado (livre): em acerto de garra, manobra agarrar (teste +18).',
      'Faro, visão no escuro.',
    ],
    treasureXp: 4000,
    bookPage: 293,
  },
  {
    id: 'troll',
    name: 'Troll',
    nd: 5,
    tipo: 'monstro',
    size: 'grande',
    hp: 165,
    defesa: 23,
    forca: 6, destreza: 2, constituicao: 6,
    inteligencia: -2, sabedoria: -1, carisma: -2,
    fortitude: 14, reflexos: 10, vontade: 6,
    deslocamento: '9m',
    attacks: [
      { name: 'Mordida', attackBonus: 17, damage: '1d8+6' },
      { name: 'Duas garras', attackBonus: 17, damage: '1d6+6' },
    ],
    specialAbilities: [
      'Cura acelerada 15 (anulada por ácido ou fogo).',
      'Dilacerar: duas garras no mesmo alvo na mesma rodada = +2d6+6 de dano.',
      'Visão no escuro.',
    ],
    treasureXp: 5000,
    bookPage: 309,
  },
  {
    id: 'manticora',
    name: 'Mantícora',
    nd: 6,
    tipo: 'monstro',
    size: 'grande',
    hp: 240,
    defesa: 26,
    forca: 7, destreza: 2, constituicao: 5,
    inteligencia: -2, sabedoria: 1, carisma: -1,
    fortitude: 18, reflexos: 7, vontade: 12,
    deslocamento: '9m, voo 15m',
    attacks: [
      { name: 'Mordida', attackBonus: 18, damage: '1d10+12' },
      { name: 'Duas garras', attackBonus: 18, damage: '1d8+12' },
    ],
    specialAbilities: [
      'Espinhos (movimento, recarga mov): 1d4 espinhos a médio, cada 1d8+7 perfuração (Ref CD 22 reduz à metade).',
      'Faro, visão no escuro.',
    ],
    treasureXp: 6000,
    bookPage: 287,
  },
  {
    id: 'dragao-jovem',
    name: 'Dragão Jovem',
    nd: 7,
    tipo: 'monstro',
    size: 'grande',
    hp: 320,
    defesa: 32,
    forca: 7, destreza: 2, constituicao: 6,
    inteligencia: 2, sabedoria: 2, carisma: 2,
    fortitude: 20, reflexos: 9, vontade: 12,
    deslocamento: '12m, voo 18m',
    attacks: [
      { name: 'Mordida', attackBonus: 25, damage: '2d6+14', special: 'crítico 19' },
      { name: 'Duas garras', attackBonus: 25, damage: '1d8+14', special: 'crítico 19' },
    ],
    specialAbilities: [
      'Sopro (padrão, recarga mov): cone 9m, 6d12 fogo + em chamas (Ref CD 25 reduz à metade e evita).',
      'Varrer (livre): 1×/rod, reduzir alvo a 0 PV concede ataque adicional contra adjacente.',
      'Redução de dano 5; imune fogo; vulnerável frio; resistência a magia +2.',
      'Metamorfose Dracônica; percepção às cegas.',
    ],
    treasureXp: 7000,
    bookPage: 311,
  },
  {
    id: 'golem-de-ferro',
    name: 'Golem de Ferro',
    nd: 10,
    tipo: 'construto',
    size: 'grande',
    hp: 400,
    defesa: 36,
    forca: 12, destreza: -1, constituicao: 10,
    inteligencia: 0, sabedoria: 0, carisma: -5,
    fortitude: 24, reflexos: 14, vontade: 11,
    deslocamento: '9m',
    attacks: [{ name: 'Duas pancadas', attackBonus: 30, damage: '2d10+25' }],
    specialAbilities: [
      'Imunidade a Magia (exceções: eletricidade deixa lento 1d6 rod; fogo remove lento e cura 1 PV por 3 de dano).',
      'Sopro (movimento, recarga, veneno): cubo 3m, 6d12 PV + enjoado (Fort CD 30 reduz à metade e evita).',
      'Redução de dano 10; imunidades de construto.',
    ],
    treasureXp: 10000,
    bookPage: 288,
  },
  {
    id: 'dragao-adulto',
    name: 'Dragão Adulto',
    nd: 11,
    tipo: 'monstro',
    size: 'enorme',
    hp: 600,
    defesa: 42,
    forca: 11, destreza: 1, constituicao: 8,
    inteligencia: 4, sabedoria: 4, carisma: 4,
    fortitude: 24, reflexos: 11, vontade: 18,
    deslocamento: '12m, voo 24m',
    attacks: [
      { name: 'Mordida', attackBonus: 35, damage: '4d10+25', special: 'crítico 18' },
      { name: 'Duas garras', attackBonus: 35, damage: '3d10+25', special: 'crítico 18' },
    ],
    specialAbilities: [
      'Sopro (padrão, recarga mov): cone 12m, 12d12 fogo + em chamas (Ref CD 32 reduz à metade e evita).',
      'Aura Aterradora (raio 90m, Von CD 31).',
      'Magia arcano 11º (CD 32): Campo de Força, Curar Ferimentos, Dissipar Magia, Enfeitiçar, Globo de Invulnerabilidade, Velocidade.',
      'Varrer, Metamorfose Dracônica, percepção às cegas, RD 10, imune fogo, vulnerável frio, resistência a magia +3.',
    ],
    treasureXp: 11000,
    bookPage: 311,
  },
  {
    id: 'hidra',
    name: 'Hidra',
    nd: 11,
    tipo: 'monstro',
    size: 'enorme',
    hp: 550,
    defesa: 35,
    forca: 10, destreza: 2, constituicao: 8,
    inteligencia: -4, sabedoria: 1, carisma: -2,
    fortitude: 24, reflexos: 18, vontade: 9,
    deslocamento: '9m, natação 9m',
    attacks: [{ name: 'Cinco mordidas', attackBonus: 34, damage: '3d6+16' }],
    specialAbilities: [
      'Cura acelerada 100.',
      'Cortar Cabeças: corte que cause dano à hidra (CD 24) nasce duas cabeças no lugar.',
      'Faro, visão no escuro.',
    ],
    treasureXp: 11000,
    bookPage: 306,
  },
  {
    id: 'vampiro',
    name: 'Vampiro',
    nd: 12,
    tipo: 'morto-vivo',
    size: 'medio',
    hp: 550,
    defesa: 45,
    forca: 6, destreza: 5, constituicao: 5,
    inteligencia: 3, sabedoria: 3, carisma: 6,
    fortitude: 12, reflexos: 26, vontade: 20,
    deslocamento: '18m, escalar 18m',
    attacks: [
      {
        name: 'Espada longa x2',
        attackBonus: 36,
        damage: '2d8+25 mais 2d10 trevas',
        special: 'crítico 17',
      },
      { name: 'Garra', attackBonus: 36, damage: '2d6+25 mais 2d10 trevas' },
    ],
    specialAbilities: [
      'Dominação Vampírica (padrão): humanoide a curto fica confuso/enfeitiçado/fascinado ou perde memórias (Von CD 29 evita); 1×/cena por alvo.',
      'Drenar Sangue: criatura agarrada perde 6d6 perfuração; vampiro cura igual.',
      'Forma de Morcego; Presença Majestosa (Von CD 29).',
      'Cura acelerada 10; RD 10/luz; sensibilidade ao sol (ofuscado, perde 6d6 PV/rod sob luz solar).',
    ],
    treasureXp: 12000,
    bookPage: 299,
  },
  {
    id: 'dragao-rei',
    name: 'Dragão-Rei',
    nd: 20,
    tipo: 'monstro',
    size: 'colossal',
    hp: 1400,
    defesa: 62,
    forca: 15, destreza: 1, constituicao: 12,
    inteligencia: 8, sabedoria: 8, carisma: 8,
    fortitude: 34, reflexos: 20, vontade: 28,
    deslocamento: '12m, voo 36m',
    attacks: [
      { name: 'Mordida', attackBonus: 55, damage: '6d20+50', special: 'crítico 16' },
      { name: 'Duas garras', attackBonus: 50, damage: '6d20+50', special: 'crítico 16' },
    ],
    specialAbilities: [
      'Sopro (padrão, recarga mov): cone 18m, 20d12 fogo + em chamas (Ref CD 50 reduz à metade e evita).',
      'Escamas Supremas: dano de fontes mundanas reduzido à metade.',
      'Aura Aterradora (Von CD 50).',
      'Magia arcano 20º (CD 50): Campo de Força, Controlar a Gravidade, Controlar o Tempo, Dissipar Magia, Enfeitiçar, Globo de Invulnerabilidade, Segunda Chance, Velocidade.',
      'Magia Acelerada, Fluxo de Mana, percepção às cegas, RD 20, imune fogo, vulnerável frio, resistência a magia +5.',
    ],
    treasureXp: 20000,
    bookPage: 312,
  },
])

const byId = new Map(BESTIARY.map((m) => [m.id, m]))

export function monsterById(id: string): Monster | undefined {
  return byId.get(id)
}

export function monstersByTipo(tipo: MonsterTipo): readonly Monster[] {
  return BESTIARY.filter((m) => m.tipo === tipo)
}

export function monstersByNdRange(min: number, max: number): readonly Monster[] {
  return BESTIARY.filter((m) => m.nd >= min && m.nd <= max)
}

export function monstersBySize(size: MonsterSize): readonly Monster[] {
  return BESTIARY.filter((m) => m.size === size)
}
