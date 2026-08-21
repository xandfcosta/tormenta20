/**
 * O bloco de criatura do livro — a forma com que o Tormenta 20 descreve TANTO
 * monstro quanto NPC humano. "BANDIDO — ND 1/4 — Humanoide (humano) Médio"
 * (p289) tem a mesma estrutura do Ogro (p293), com Perícias, Equipamento e
 * Tesouro; conjurador ganha Pontos de Mana (Centauro Xamã, 20 PM, p290).
 *
 * Espelha `CreatureBlock` do Go. Os dois lados são escritos à mão porque o
 * gerador de tipos cobre a fronteira do WASM, não a da API (ALE-108).
 */
export type CreatureBlock = {
  nd: number
  tipo: CreatureTipo
  size: CreatureSize
  iniciativa: number
  percepcao: number
  defesa: number
  fortitude: number
  reflexos: number
  vontade: number
  hp: number
  /**
   * Ausente na maioria: a linha "Pontos de Mana" só existe em conjurador, e um
   * zero diria "tem mana e está sem", que é outro estado.
   */
  pm?: number
  deslocamento: string
  forca: number
  destreza: number
  constituicao: number
  inteligencia: number
  sabedoria: number
  carisma: number
  attacks: CreatureAttack[]
  skills: CreatureSkill[]
  equipment: string
  treasure: string
  specialAbilities: string[]
  /** De qual verbete este bloco foi copiado; vazio quando escrito do zero. */
  sourceMonsterId?: string
}

/** Uma linha de ataque: "Corpo a Corpo Clava +7 (1d6+3)". */
export type CreatureAttack = {
  name: string
  attackBonus: number
  damage: string
  /** "À Distância" é linha separada de "Corpo a Corpo" no bloco impresso. */
  ranged?: boolean
  /** A nota entre parênteses da linha de ataque ("mais agarrar"). */
  special?: string
}

/**
 * Uma perícia da linha "Perícias Furtividade +5".
 *
 * `nota` é o bônus CONDICIONAL entre parênteses DEPOIS do número — a Hidra tem
 * "Furtividade +4 (+14 em pântanos)" (p306). Irmã do `special` do ataque, e
 * pela mesma razão: prosa que o livro grudou numa linha estruturada.
 *
 * Cuidado com o outro parêntese, que quer o oposto: `Ofício (armeiro) +2` vem
 * ANTES do número e faz parte do NOME da perícia (ALE-151).
 */
export type CreatureSkill = { name: string; bonus: number; nota?: string }

/** Os tipos de criatura do livro — fechados porque o livro os fecha. */
export type CreatureTipo =
  | 'humanoide'
  | 'animal'
  | 'monstro'
  | 'morto-vivo'
  | 'construto'
  | 'espirito'
  | 'planar'

export type CreatureSize =
  | 'minusculo'
  | 'pequeno'
  | 'medio'
  | 'grande'
  | 'enorme'
  | 'colossal'

/** Uma criatura da campanha: identidade fora, bloco dentro. */
export type CampaignCreature = {
  id: number
  campaignId: number
  name: string
  block: CreatureBlock
  createdAt: string
  updatedAt: string
}

export type CreatureInput = { name: string; block: CreatureBlock }

/**
 * O bloco de quem ainda não tem nada: um capanga genérico de ND 1, que é o que
 * o mestre mais improvisa. Serve de ponto de partida para "NPC completo" criado
 * do zero — começar com tudo em zero obrigaria a preencher dezoito campos antes
 * de a criatura existir.
 */
export function blankCreatureBlock(): CreatureBlock {
  return {
    nd: 1,
    tipo: 'humanoide',
    size: 'medio',
    iniciativa: 0,
    percepcao: 0,
    defesa: 10,
    fortitude: 0,
    reflexos: 0,
    vontade: 0,
    hp: 10,
    deslocamento: '9m (6q)',
    forca: 0,
    destreza: 0,
    constituicao: 0,
    inteligencia: 0,
    sabedoria: 0,
    carisma: 0,
    attacks: [],
    skills: [],
    equipment: '',
    treasure: '',
    specialAbilities: [],
  }
}
