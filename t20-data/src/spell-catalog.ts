import type {
  AugmentKind,
  SpellCircle,
  SpellComponent,
  SpellDuration,
  SpellExecution,
  SpellRange,
  SpellResistance,
  SpellSaveType,
  SpellSchool,
} from './spells'

/**
 * Spell catalog seed — PDF book Cap 4 (p174-211).
 *
 * This is a curated **seed** (14 entries), not the full grimoire. The
 * goal is to cover every escola, save type, resistência type, and
 * range / duration / execução variant at least once, so downstream
 * code that consumes the catalog can be exercised end-to-end without
 * shipping all ~200 magias at once.
 *
 * Augments carry T20-specific gating: some require a higher círculo
 * (`requiresCircle`), some are class-list-only (`classOnly: 'arcanos'`
 * / `'divinos'` / `'druidas'`). PDF formatting: "Requer Xº círculo",
 * "Apenas Arcanos / Divinos / Druidas".
 */

export type SpellClassName = 'Arcanista' | 'Bardo' | 'Clérigo' | 'Druida' | 'Paladino'

export type CatalogAugment = {
  pmCost: number
  kind: AugmentKind
  description: string
  requiresCircle?: SpellCircle
  classOnly?: 'arcanos' | 'divinos' | 'druidas'
}

export type CatalogSpell = {
  id: string
  name: string
  circle: SpellCircle
  school: SpellSchool
  execution: SpellExecution
  range: SpellRange
  duration: SpellDuration
  /** Free-text duration when `duration === 'definida'`. */
  durationNote?: string
  saveType: SpellSaveType
  /** `null` when the magia offers no resistance roll at all. */
  resistance: SpellResistance | null
  components: SpellComponent[]
  classes: SpellClassName[]
  baseEffect: string
  augments: CatalogAugment[]
  bookPage: number
}

const VG: SpellComponent[] = ['verbal', 'gestual']

const SPELLS: readonly CatalogSpell[] = [
  {
    id: 'luz',
    name: 'Luz',
    circle: 1,
    school: 'evocacao',
    execution: 'padrao',
    range: 'curto',
    duration: 'cena',
    saveType: 'vontade',
    resistance: 'anula',
    components: VG,
    classes: ['Arcanista', 'Bardo', 'Clérigo', 'Druida'],
    baseEffect:
      'O alvo (1 objeto) emite luz, mas sem calor, em uma área de 6m de raio. Pode ser guardado para interromper a luz.',
    augments: [
      { pmCost: 1, kind: 'aumenta', description: 'Aumenta o raio iluminado em +3m.' },
      { pmCost: 2, kind: 'aumenta', description: 'Muda a duração para um dia.' },
      {
        pmCost: 0,
        kind: 'muda',
        description:
          'Alvo passa a ser uma criatura nos olhos; alvo fica ofuscado pela cena.',
        classOnly: 'arcanos',
      },
    ],
    bookPage: 197,
  },
  {
    id: 'armadura-arcana',
    name: 'Armadura Arcana',
    circle: 1,
    school: 'abjuracao',
    execution: 'padrao',
    range: 'pessoal',
    duration: 'cena',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Arcanista'],
    baseEffect:
      'Película invisível mas tangível confere +5 na Defesa. Cumulativa com outras magias, mas não com bônus de armadura.',
    augments: [
      {
        pmCost: 1,
        kind: 'muda',
        description:
          'Execução para reação; vira escudo mágico contra um único ataque, dando +5 na Defesa contra ele.',
      },
      { pmCost: 2, kind: 'aumenta', description: 'Aumenta o bônus de Defesa em +1.' },
      { pmCost: 2, kind: 'muda', description: 'Muda a duração para um dia.' },
    ],
    bookPage: 181,
  },
  {
    id: 'sono',
    name: 'Sono',
    circle: 1,
    school: 'encantamento',
    execution: 'padrao',
    range: 'curto',
    duration: 'cena',
    saveType: 'vontade',
    resistance: 'parcial',
    components: VG,
    classes: ['Arcanista', 'Bardo'],
    baseEffect:
      'Alvo cai em sono mágico (caso falhe, fica inconsciente e caído; em combate, exausto por 1 rodada e depois fatigado). Resistência reduz para fatigado 1d4 rodadas.',
    augments: [
      {
        pmCost: 1,
        kind: 'aumenta',
        description: 'Aumenta o número de alvos em +1 (compartilham um sonho).',
      },
    ],
    bookPage: 207,
  },
  {
    id: 'visao-mistica',
    name: 'Visão Mística',
    circle: 1,
    school: 'adivinhacao',
    execution: 'padrao',
    range: 'pessoal',
    duration: 'cena',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Arcanista', 'Bardo', 'Clérigo', 'Druida'],
    baseEffect:
      'Detecta auras mágicas em alcance médio. Ação de movimento descobre se uma criatura pode lançar magias e o círculo mais alto que ela pode lançar.',
    augments: [
      { pmCost: 1, kind: 'aumenta', description: 'Ganha visão no escuro.' },
      { pmCost: 2, kind: 'aumenta', description: 'Muda a duração para um dia.' },
      {
        pmCost: 2,
        kind: 'aumenta',
        description: 'Enxerga objetos e criaturas invisíveis (translúcidos).',
      },
    ],
    bookPage: 211,
  },
  {
    id: 'curar-ferimentos',
    name: 'Curar Ferimentos',
    circle: 1,
    school: 'evocacao',
    execution: 'padrao',
    range: 'toque',
    duration: 'instantanea',
    saveType: 'vontade',
    resistance: 'metade',
    components: VG,
    classes: ['Clérigo', 'Druida', 'Paladino'],
    baseEffect:
      'Canaliza luz que recupera 2d8+2 PV no alvo. Anula Infligir Ferimentos.',
    augments: [
      {
        pmCost: 1,
        kind: 'muda',
        description:
          'Alvo para um morto-vivo; em vez de curar, causa 1d8 dano de luz (Vontade metade).',
      },
      { pmCost: 1, kind: 'aumenta', description: 'Aumenta a cura em +1d8+1.' },
      {
        pmCost: 2,
        kind: 'aumenta',
        description: 'Também remove uma condição de cansaço do alvo.',
      },
      { pmCost: 2, kind: 'muda', description: 'Muda alcance para curto.' },
      {
        pmCost: 5,
        kind: 'muda',
        description: 'Alcance curto e alvo para criaturas escolhidas.',
      },
    ],
    bookPage: 189,
  },
  {
    id: 'enfeiticar',
    name: 'Enfeitiçar',
    circle: 1,
    school: 'encantamento',
    execution: 'padrao',
    range: 'curto',
    duration: 'cena',
    saveType: 'vontade',
    resistance: 'anula',
    components: VG,
    classes: ['Arcanista', 'Bardo'],
    baseEffect:
      'Alvo fica enfeitiçado. Em combate, recebe +5 na resistência. Ações hostis de aliados contra o alvo dissipam a magia.',
    augments: [
      {
        pmCost: 2,
        kind: 'muda',
        description:
          'Sugere uma ação razoável; alvo obedece. A magia termina ao concluir a ação.',
      },
      {
        pmCost: 3,
        kind: 'muda',
        description: 'Alvo para um espírito ou monstro.',
        requiresCircle: 3,
      },
      {
        pmCost: 5,
        kind: 'aumenta',
        description: 'Afeta todos os alvos válidos no alcance.',
      },
    ],
    bookPage: 191,
  },
  {
    id: 'conjurar-monstro',
    name: 'Conjurar Monstro',
    circle: 1,
    school: 'convocacao',
    execution: 'padrao',
    range: 'curto',
    duration: 'sustentada',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Arcanista'],
    baseEffect:
      'Convoca um monstro Pequeno controlável (For 2, Des 3, Defesa = do conjurador, 20 PV, imune a Fortitude/Vontade) que pode receber ordens de mover, atacar ou lançar magia.',
    augments: [
      {
        pmCost: 1,
        kind: 'aumenta',
        description: 'Ganha deslocamento de escalada e natação.',
      },
      { pmCost: 1, kind: 'aumenta', description: 'Aumenta o deslocamento em +3m.' },
      {
        pmCost: 1,
        kind: 'muda',
        description: 'Tipo de dano para ácido, fogo, frio ou eletricidade.',
      },
      {
        pmCost: 2,
        kind: 'aumenta',
        description: 'Aumenta o tamanho para Médio (For 4, 45 PV, 12m, 2d6+6).',
      },
      {
        pmCost: 2,
        kind: 'aumenta',
        description: 'Ganha redução de 5 contra dois tipos de dano.',
      },
      {
        pmCost: 5,
        kind: 'aumenta',
        description: 'Aumenta tamanho para Grande (For 7, 75 PV, 4d6+10).',
        requiresCircle: 2,
      },
    ],
    bookPage: 185,
  },
  {
    id: 'queda-suave',
    name: 'Queda Suave',
    circle: 1,
    school: 'transmutacao',
    execution: 'reacao',
    range: 'curto',
    duration: 'definida',
    durationNote: 'até chegar ao solo ou cena, o que ocorrer primeiro',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Arcanista', 'Bardo'],
    baseEffect:
      'Alvo (1 criatura ou objeto Grande ou menor) cai a 18m por rodada — não sofre dano. Pode ser usada contra projéteis (alvo sofre metade do dano).',
    augments: [
      {
        pmCost: 1,
        kind: 'muda',
        description: 'Alvo objeto Minúsculo; pode ser levitado 4,5m em qualquer direção como ação de movimento.',
      },
      {
        pmCost: 2,
        kind: 'aumenta',
        description: 'Alvo para até 10 criaturas ou objetos.',
      },
      {
        pmCost: 2,
        kind: 'aumenta',
        description: 'Aumenta a categoria de tamanho do alvo em uma.',
      },
    ],
    bookPage: 202,
  },
  {
    id: 'criar-ilusao',
    name: 'Criar Ilusão',
    circle: 1,
    school: 'ilusao',
    execution: 'padrao',
    range: 'medio',
    duration: 'cena',
    saveType: 'vontade',
    resistance: 'desacredita',
    components: VG,
    classes: ['Arcanista', 'Bardo'],
    baseEffect:
      'Cria uma ilusão sensorial (imagem, som ou aroma) num cubo de 3m de aresta. Interação dá novo teste de Vontade.',
    augments: [
      {
        pmCost: 1,
        kind: 'aumenta',
        description: 'Aumenta o tamanho do cubo em +3m de aresta.',
      },
      {
        pmCost: 2,
        kind: 'aumenta',
        description: 'Adiciona som, aroma OU textura à ilusão.',
      },
      {
        pmCost: 3,
        kind: 'muda',
        description: 'Cria uma ilusão de criatura ou objeto específico.',
        requiresCircle: 2,
      },
    ],
    bookPage: 189,
  },
  {
    id: 'bola-de-fogo',
    name: 'Bola de Fogo',
    circle: 2,
    school: 'evocacao',
    execution: 'padrao',
    range: 'medio',
    duration: 'instantanea',
    saveType: 'reflexos',
    resistance: 'metade',
    components: VG,
    classes: ['Arcanista'],
    baseEffect:
      'Explosão de 6d6 de dano de fogo em todas as criaturas e objetos numa esfera de 6m de raio.',
    augments: [
      { pmCost: 2, kind: 'aumenta', description: 'Aumenta o dano em +2d6.' },
      {
        pmCost: 2,
        kind: 'muda',
        description:
          'Cria esfera flamejante Média de 1,5m que causa 3d6 dano de fogo; pode voar 9m com ação de movimento.',
      },
      {
        pmCost: 3,
        kind: 'muda',
        description:
          'Vira pedra flamejante; pode ser arremessada (curto) e detonada como reação.',
      },
    ],
    bookPage: 182,
  },
  {
    id: 'invisibilidade',
    name: 'Invisibilidade',
    circle: 2,
    school: 'ilusao',
    execution: 'livre',
    range: 'pessoal',
    duration: 'definida',
    durationNote: '1 rodada',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Arcanista', 'Bardo'],
    baseEffect:
      'O alvo fica invisível: camuflagem total, +10 em Furtividade para ouvir. Acaba se o alvo faz uma ação hostil.',
    augments: [
      {
        pmCost: 1,
        kind: 'muda',
        description:
          'Execução padrão, alcance curto, alvo 1 criatura ou objeto Grande ou menor.',
      },
      {
        pmCost: 3,
        kind: 'muda',
        description: 'Muda a duração para cena.',
        requiresCircle: 3,
      },
      {
        pmCost: 7,
        kind: 'muda',
        description: 'Execução padrão, alcance toque; magia não dissipa por ação hostil.',
        requiresCircle: 4,
      },
    ],
    bookPage: 195,
  },
  {
    id: 'toque-vampirico',
    name: 'Toque Vampírico',
    circle: 2,
    school: 'necromancia',
    execution: 'padrao',
    range: 'toque',
    duration: 'instantanea',
    saveType: 'fortitude',
    resistance: 'metade',
    components: VG,
    classes: ['Arcanista'],
    baseEffect:
      'Causa 6d6 de dano de trevas no alvo. O conjurador recupera PV iguais à metade do dano causado.',
    augments: [
      {
        pmCost: 2,
        kind: 'muda',
        description:
          'Vira ataque corpo a corpo; soma dano do ataque + magia; cura metade do dano da magia.',
      },
      { pmCost: 2, kind: 'aumenta', description: 'Aumenta o dano em +2d6.' },
      {
        pmCost: 2,
        kind: 'muda',
        description:
          'Alcance pessoal, duração cena; uma vez por rodada toca 1 criatura para 3d6 trevas.',
        requiresCircle: 3,
      },
    ],
    bookPage: 209,
  },
  {
    id: 'relampago',
    name: 'Relâmpago',
    circle: 2,
    school: 'evocacao',
    execution: 'padrao',
    range: 'pessoal',
    duration: 'instantanea',
    saveType: 'reflexos',
    resistance: 'metade',
    components: VG,
    classes: ['Arcanista'],
    baseEffect:
      'Raio em linha de 30m causa 6d6 de dano de eletricidade em todas as criaturas e objetos.',
    augments: [
      { pmCost: 2, kind: 'aumenta', description: 'Aumenta o dano em +2d6.' },
      {
        pmCost: 3,
        kind: 'muda',
        description:
          'Alcance médio, alvo para criaturas escolhidas; dispara um relâmpago para cada alvo.',
        requiresCircle: 3,
      },
    ],
    bookPage: 203,
  },
  {
    id: 'metamorfose',
    name: 'Metamorfose',
    circle: 2,
    school: 'transmutacao',
    execution: 'padrao',
    range: 'pessoal',
    duration: 'cena',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Arcanista'],
    baseEffect:
      'Muda aparência e forma (inclusive equipamento) para outra criatura. +20 em Enganação para disfarce.',
    augments: [
      {
        pmCost: 1,
        kind: 'aumenta',
        description: 'Ganha faro, visão na penumbra OU visão no escuro.',
      },
      {
        pmCost: 3,
        kind: 'muda',
        description: 'Alcance toque, alvo 1 criatura inofensiva (Vontade anula).',
      },
      {
        pmCost: 5,
        kind: 'muda',
        description: 'Aceita formas não-humanoides (Forma Selvagem Aprimorada).',
        requiresCircle: 3,
      },
    ],
    bookPage: 198,
  },
]

export const SPELL_CATALOG: Readonly<Record<string, CatalogSpell>> =
  Object.freeze(
    SPELLS.reduce<Record<string, CatalogSpell>>((acc, spell) => {
      acc[spell.id] = spell
      return acc
    }, {}),
  )

export const SPELL_IDS: readonly string[] = SPELLS.map((s) => s.id)

export function spellById(id: string): CatalogSpell {
  const spell = SPELL_CATALOG[id]
  if (!spell) {
    throw new Error(`spellById: unknown spell id "${id}"`)
  }
  return spell
}

export function spellsBySchool(school: SpellSchool): readonly CatalogSpell[] {
  return SPELLS.filter((s) => s.school === school)
}

export function spellsByCircle(circle: SpellCircle): readonly CatalogSpell[] {
  return SPELLS.filter((s) => s.circle === circle)
}

export function spellsByClass(className: SpellClassName): readonly CatalogSpell[] {
  return SPELLS.filter((s) => s.classes.includes(className))
}
