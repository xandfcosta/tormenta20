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

  // ─── Lote +19 — abrir cobertura Cap 7 (p179-211) ────────────────
  {
    id: 'dissipar-magia',
    name: 'Dissipar Magia',
    circle: 2,
    school: 'abjuracao',
    execution: 'padrao',
    range: 'medio',
    duration: 'instantanea',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Arcanista', 'Bardo', 'Clérigo', 'Druida'],
    baseEffect:
      'Dissipa magias ativas em alvo ou área de 3m de raio. Faça teste de Misticismo; dissipa magias com CD igual ou menor que o resultado. Contra item mágico, transforma em mundano por 1d6 rodadas (Vontade anula).',
    augments: [
      {
        pmCost: 12,
        kind: 'muda',
        description:
          'Área esfera 9m; todas as magias automaticamente dissipadas e itens mágicos não carregados viram mundanos por uma cena (Vontade evita).',
        requiresCircle: 5,
      },
    ],
    bookPage: 191,
  },
  {
    id: 'escudo-da-fe',
    name: 'Escudo da Fé',
    circle: 1,
    school: 'abjuracao',
    execution: 'reacao',
    range: 'curto',
    duration: 'definida',
    durationNote: '1 turno',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Clérigo', 'Paladino'],
    baseEffect:
      'Escudo místico se manifesta momentaneamente. Alvo recebe +2 na Defesa.',
    augments: [
      {
        pmCost: 1,
        kind: 'muda',
        description:
          'Muda execução para padrão, alcance toque, duração cena.',
      },
      {
        pmCost: 1,
        kind: 'muda',
        description: 'Fornece também camuflagem leve contra ataques à distância.',
      },
      { pmCost: 2, kind: 'aumenta', description: 'Aumenta o bônus em +1.' },
      {
        pmCost: 2,
        kind: 'muda',
        description:
          'Padrão/toque/cena. Cria conexão mística — alvo sofre metade do dano por ataques; outra metade transfere para você. Magia dissipa se alvo sair do alcance curto.',
        requiresCircle: 2,
      },
      { pmCost: 3, kind: 'muda', description: 'Muda duração para um dia.' },
    ],
    bookPage: 192,
  },
  {
    id: 'invulnerabilidade',
    name: 'Invulnerabilidade',
    circle: 5,
    school: 'abjuracao',
    execution: 'padrao',
    range: 'pessoal',
    duration: 'cena',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Arcanista', 'Bardo', 'Clérigo', 'Druida'],
    baseEffect:
      'Cria barreira mágica impenetrável contra efeitos nocivos mentais ou físicos, à sua escolha. Mental: imune a abalado, alquebrado, apavorado, atordoado, confuso, esmorecido, fascinado, frustrado, pasmo, encantamento e ilusão. Físico: imune a atordoado, cego, debilitado, enjoado, envenenado, exausto, fatigado, fraco, lento, ofuscado, paralisado, acertos críticos, ataques furtivos e doenças.',
    augments: [
      {
        pmCost: 5,
        kind: 'muda',
        description: 'Muda alcance para curto e alvo para 1 criatura.',
      },
    ],
    bookPage: 195,
  },
  {
    id: 'detectar-ameacas',
    name: 'Detectar Ameaças',
    circle: 1,
    school: 'adivinhacao',
    execution: 'padrao',
    range: 'pessoal',
    duration: 'definida',
    durationNote: 'cena, até ser descarregada',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Clérigo', 'Druida', 'Paladino'],
    baseEffect:
      'Intuição aguçada sobre perigos. Esfera 18m. Quando criatura hostil ou armadilha entra, faça teste de Percepção (CD a critério do mestre). Sucesso revela origem, direção e distância; falha revela apenas que perigo existe.',
    augments: [
      {
        pmCost: 1,
        kind: 'aumenta',
        description:
          'Descobre também raça/espécie e poder da criatura via aura (tênue ≤6º, moderada 7-12, poderosa 13-20, avassaladora >20).',
      },
      {
        pmCost: 2,
        kind: 'aumenta',
        description:
          'Não fica surpreendido contra perigos detectados; +5 em testes de resistência contra armadilhas.',
        requiresCircle: 2,
      },
    ],
    bookPage: 190,
  },
  {
    id: 'voz-divina',
    name: 'Voz Divina',
    circle: 2,
    school: 'adivinhacao',
    execution: 'padrao',
    range: 'pessoal',
    duration: 'cena',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Clérigo', 'Druida'],
    baseEffect:
      'Você pode conversar com criaturas de qualquer raça/tipo (animal, construto, espírito, humanoide, monstro, morto-vivo). Pode fazer perguntas e entender respostas mesmo sem idioma comum, respeitando a Inteligência da criatura. Atitude não muda, mas pode usar Diplomacia.',
    augments: [
      {
        pmCost: 1,
        kind: 'aumenta',
        description:
          'Concede vida a um cadáver para responder perguntas. Conhecimento limitado ao do alvo em vida; respostas curtas e enigmáticas. Cabeça precisa estar intacta. Um corpo só pode ser alvo uma vez.',
      },
      {
        pmCost: 1,
        kind: 'aumenta',
        description:
          'Pode falar com plantas (normais ou monstruosas) e rochas. Percepção limitada; respostas simplórias.',
      },
    ],
    bookPage: 211,
  },
  {
    id: 'videncia',
    name: 'Vidência',
    circle: 3,
    school: 'adivinhacao',
    execution: 'completa',
    range: 'ilimitado',
    duration: 'sustentada',
    saveType: 'vontade',
    resistance: 'anula',
    components: ['verbal', 'gestual', 'foco'],
    classes: ['Arcanista', 'Bardo', 'Clérigo', 'Druida'],
    baseEffect:
      'Através de uma superfície reflexiva (bacia de água benta, lago, bola de cristal, espelho...) você vê e ouve uma criatura escolhida e seus arredores (6m). Vontade anula. Modificadores conforme conhecimento: não conhece +10; ouviu falar +5; outro plano +5; encontrou em pessoa +0; retrato/escultura -2; conhece bem -5; pertence pessoal -5; parte do corpo -10.',
    augments: [],
    bookPage: 211,
  },
  {
    id: 'caminhos-da-natureza',
    name: 'Caminhos da Natureza',
    circle: 1,
    school: 'convocacao',
    execution: 'padrao',
    range: 'curto',
    duration: 'definida',
    durationNote: '1 dia',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Clérigo', 'Druida'],
    baseEffect:
      'Invoca espíritos da natureza para abrir caminho. Alvos recebem deslocamento +3m e ignoram penalidades por terreno difícil em terrenos naturais.',
    augments: [
      {
        pmCost: 0,
        kind: 'muda',
        description:
          'Truque: muda alcance para pessoal/alvo para você. +5 em testes de Sobrevivência para se orientar.',
      },
      {
        pmCost: 1,
        kind: 'aumenta',
        description: 'CD para rastrear os alvos em terreno natural +10.',
      },
      {
        pmCost: 2,
        kind: 'aumenta',
        description: 'Aumenta o bônus de deslocamento em +3m.',
      },
    ],
    bookPage: 183,
  },
  {
    id: 'teletransporte',
    name: 'Teletransporte',
    circle: 3,
    school: 'convocacao',
    execution: 'padrao',
    range: 'toque',
    duration: 'instantanea',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Arcanista', 'Bardo'],
    baseEffect:
      'Transporta até 5 criaturas voluntárias para lugar à sua escolha em até 1.000 km. Teste de Misticismo: CD 20 familiar; CD 30 conhecido; CD 40 só por descrição. Falha = 1d10 × 10 km fora; falha por 5+ = lugar parecido mas errado; 1 natural = falha + atordoado 1d4 rodadas.',
    augments: [
      {
        pmCost: 2,
        kind: 'aumenta',
        description: 'Aumenta número de alvos em +5.',
      },
      {
        pmCost: 2,
        kind: 'muda',
        description:
          'Em vez do normal, teletransporta para seu santuário — local familiar pré-preparado. Sem teste, mas apenas dentro do mesmo plano. Preparar santuário: ritual de 1 dia + T$ 1.000. Apenas um santuário por vez.',
      },
    ],
    bookPage: 208,
  },
  {
    id: 'amedrontar',
    name: 'Amedrontar',
    circle: 1,
    school: 'necromancia',
    execution: 'padrao',
    range: 'curto',
    duration: 'cena',
    saveType: 'vontade',
    resistance: 'parcial',
    components: VG,
    classes: ['Arcanista', 'Bardo'],
    baseEffect:
      'O alvo (1 animal ou humanoide) é envolvido por energias sombrias. Falha: apavorado 1 rodada, depois abalado. Sucesso: abalado 1d4 rodadas.',
    augments: [
      {
        pmCost: 1,
        kind: 'aumenta',
        description:
          'Alvos que falhem ficam apavorados por 1d4+1 rodadas em vez de 1.',
      },
      { pmCost: 2, kind: 'muda', description: 'Muda alvo para 1 criatura.' },
      {
        pmCost: 5,
        kind: 'aumenta',
        description: 'Afeta todos os alvos válidos à sua escolha no alcance.',
      },
    ],
    bookPage: 179,
  },
  {
    id: 'marionete',
    name: 'Marionete',
    circle: 4,
    school: 'encantamento',
    execution: 'padrao',
    range: 'medio',
    duration: 'sustentada',
    saveType: 'fortitude',
    resistance: 'anula',
    components: VG,
    classes: ['Arcanista', 'Bardo'],
    baseEffect:
      'Manipula o sistema nervoso do alvo. Ao sofrer e no início de cada turno do alvo, Fortitude anula. Em falha, todas as ações físicas no turno são controladas pelo conjurador. Vítima fica consciente e pode falar e lançar magias com esforço. Você precisa de linha de efeito; sem ela, vítima fica paralisada até retomar controle ou magia terminar.',
    augments: [],
    bookPage: 198,
  },
  {
    id: 'explosao-de-chamas',
    name: 'Explosão de Chamas',
    circle: 1,
    school: 'evocacao',
    execution: 'padrao',
    range: 'pessoal',
    duration: 'instantanea',
    saveType: 'reflexos',
    resistance: 'metade',
    components: VG,
    classes: ['Arcanista'],
    baseEffect:
      'Leque de chamas de suas mãos causa 2d6 de fogo em criaturas na área (cone 6m).',
    augments: [
      {
        pmCost: 0,
        kind: 'muda',
        description:
          'Truque: muda alcance para curto, área para 1 objeto, resistência para Reflexos anula. Não causa dano, mas acende vela/tocha/fogueira ou inflama objeto com RD 0 (corda, pergaminho). Criatura em posse pode evitar com a resistência.',
      },
      { pmCost: 1, kind: 'aumenta', description: 'Aumenta o dano em +1d6.' },
      {
        pmCost: 1,
        kind: 'muda',
        description:
          'Muda resistência para Reflexos parcial. Sucesso = metade do dano; falha = em chamas.',
      },
    ],
    bookPage: 193,
  },
  {
    id: 'tempestade-divina',
    name: 'Tempestade Divina',
    circle: 2,
    school: 'evocacao',
    execution: 'padrao',
    range: 'longo',
    duration: 'sustentada',
    saveType: 'reflexos',
    resistance: 'metade',
    components: VG,
    classes: ['Clérigo', 'Druida'],
    baseEffect:
      'Apenas em ambientes abertos. Cilindro 15m raio × 15m altura. Vendaval: ataques à distância -5; chamas apagam; névoas dissipam. Pode gerar chuva (-5 Percepção), neve (chuva + terreno difícil) ou granizo (chuva + 1 dano impacto/rodada).',
    augments: [
      {
        pmCost: 1,
        kind: 'aumenta',
        description:
          'Uma vez por rodada, ação padrão para fazer raio cair em alvo na área: 3d8 eletricidade (Reflexos metade).',
      },
      {
        pmCost: 2,
        kind: 'aumenta',
        description: 'Aumenta o dano do raio em +1d8.',
      },
      {
        pmCost: 3,
        kind: 'aumenta',
        description:
          'Se chuva, fica muito grossa: revela silhueta de invisíveis; Médias ou menores ficam lentas; voadoras precisam Atletismo/rodada ou caem.',
      },
      {
        pmCost: 3,
        kind: 'aumenta',
        description: 'Se granizo, muda dano para 2d6/rodada.',
      },
      {
        pmCost: 3,
        kind: 'aumenta',
        description:
          'Se neve, criaturas na área sofrem 2d6 de frio no início de seus turnos.',
      },
      {
        pmCost: 3,
        kind: 'muda',
        description: 'Cilindro 90m raio × 90m altura.',
      },
    ],
    bookPage: 208,
  },
  {
    id: 'manto-de-sombras',
    name: 'Manto de Sombras',
    circle: 3,
    school: 'ilusao',
    execution: 'padrao',
    range: 'pessoal',
    duration: 'sustentada',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Arcanista', 'Bardo', 'Clérigo'],
    baseEffect:
      'Manto de energia sombria torna você incorpóreo (com equipamento). Só afetado por armas e habilidades mágicas ou criaturas incorpóreas; atravessa objetos sólidos mas não manipula. Vulnerável a luz direta (1 dano/rodada exposto). Ação de movimento + 1 PM: entra em sombra do seu tamanho ou maior e teletransporta para outra sombra em alcance médio.',
    augments: [
      {
        pmCost: 4,
        kind: 'muda',
        description: 'Muda alcance para toque e alvo para 1 criatura voluntária.',
        requiresCircle: 4,
      },
    ],
    bookPage: 197,
  },
  {
    id: 'imagem-espelhada',
    name: 'Imagem Espelhada',
    circle: 1,
    school: 'ilusao',
    execution: 'padrao',
    range: 'pessoal',
    duration: 'cena',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Arcanista', 'Bardo'],
    baseEffect:
      'Três cópias ilusórias suas aparecem e imitam suas ações. +6 na Defesa. Cada ataque que erra destrói uma cópia (1 cópia = +4 Defesa, 0 = +2). Oponente precisa ver as cópias para se confundir.',
    augments: [
      {
        pmCost: 2,
        kind: 'aumenta',
        description:
          'Aumenta número de cópias em +1 (e bônus na Defesa em +2).',
      },
      {
        pmCost: 2,
        kind: 'aumenta',
        description:
          'Cópia destruída emite clarão: criatura que a destruiu fica ofuscada por uma rodada.',
        requiresCircle: 2,
      },
    ],
    bookPage: 195,
  },
  {
    id: 'sopro-das-uivantes',
    name: 'Sopro das Uivantes',
    circle: 2,
    school: 'evocacao',
    execution: 'padrao',
    range: 'pessoal',
    duration: 'instantanea',
    saveType: 'fortitude',
    resistance: 'parcial',
    components: VG,
    classes: ['Arcanista'],
    baseEffect:
      'Sopra ar gélido em cone de 9m: 4d6 dano de frio (Fortitude metade). Criaturas Médias que falham ficam caídas e empurradas 6m. Parede no caminho para o movimento, mas vítima sofre +2d6 de impacto.',
    augments: [
      { pmCost: 2, kind: 'aumenta', description: 'Aumenta dano de frio em +2d6.' },
      {
        pmCost: 3,
        kind: 'aumenta',
        description: 'Aumenta tamanho máximo afetado em uma categoria.',
        requiresCircle: 3,
      },
    ],
    bookPage: 207,
  },
  {
    id: 'conjurar-mortos-vivos',
    name: 'Conjurar Mortos-Vivos',
    circle: 2,
    school: 'necromancia',
    execution: 'completa',
    range: 'curto',
    duration: 'sustentada',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Arcanista', 'Bardo', 'Clérigo', 'Druida'],
    baseEffect:
      'Conjura 6 esqueletos capangas Médios em espaços desocupados. Movimento (ação de mov): andam 9m. Ataque (ação padrão): mordida 1d6+2 trevas em criaturas adjacentes. Esqueletos têm For 2 / Des 2 / Defesa 18 / 1 PV; falham automaticamente em resistências; imunes a atordoamento, cansaço, dano não letal, doença, encantamento, frio, ilusão, paralisia, sono e veneno.',
    augments: [
      {
        pmCost: 1,
        kind: 'aumenta',
        description: 'Mortos-vivos na área sofrem -2 em testes e Defesa.',
      },
      {
        pmCost: 2,
        kind: 'aumenta',
        description: 'Aumenta número de mortos-vivos em +1.',
      },
      {
        pmCost: 3,
        kind: 'muda',
        description: 'Em vez de esqueletos, conjura carniçais.',
        requiresCircle: 3,
      },
      {
        pmCost: 7,
        kind: 'muda',
        description: 'Em vez de esqueletos, conjura sombras.',
        requiresCircle: 4,
      },
    ],
    bookPage: 186,
  },
  {
    id: 'tranca-arcana',
    name: 'Tranca Arcana',
    circle: 1,
    school: 'abjuracao',
    execution: 'padrao',
    range: 'toque',
    duration: 'definida',
    durationNote: 'permanente',
    saveType: 'none',
    resistance: null,
    components: ['verbal', 'gestual', 'material'],
    classes: ['Arcanista'],
    baseEffect:
      'Tranca porta ou item que possa ser aberto/fechado (baú, caixa). CD para abrir com Força ou Ladinagem +10. Você abre livremente sua própria tranca. Componente material: chave de bronze T$ 25.',
    augments: [
      {
        pmCost: 0,
        kind: 'muda',
        description:
          'Truque: alcance curto. Abre ou fecha objeto Grande ou menor (porta, baú). Não afeta objetos trancados.',
      },
      {
        pmCost: 1,
        kind: 'muda',
        description:
          'Alcance curto, duração instantânea. Abre portas, baús e janelas trancadas/presas/barradas ou protegidas por Tranca Arcana (dissipa); afrouxa grilhões e correntes.',
      },
      {
        pmCost: 5,
        kind: 'aumenta',
        description: 'Aumenta CD para abrir em +5.',
      },
      {
        pmCost: 5,
        kind: 'muda',
        description: 'Alvo passa a ser objeto de qualquer tamanho.',
        requiresCircle: 3,
      },
    ],
    bookPage: 210,
  },
  {
    id: 'velocidade',
    name: 'Velocidade',
    circle: 2,
    school: 'transmutacao',
    execution: 'padrao',
    range: 'curto',
    duration: 'sustentada',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Arcanista', 'Bardo'],
    baseEffect:
      'Alvo (1 criatura) pode realizar uma ação padrão ou de movimento adicional por turno. Não pode lançar magias nem ativar engenhocas.',
    augments: [
      {
        pmCost: 0,
        kind: 'muda',
        description:
          'Muda duração para cena. Ação adicional só pode ser de movimento. Uma criatura só pode receber uma ação adicional por turno por Velocidade.',
      },
      {
        pmCost: 7,
        kind: 'muda',
        description: 'Muda alvo para criaturas escolhidas no alcance.',
        requiresCircle: 4,
      },
      {
        pmCost: 7,
        kind: 'muda',
        description:
          'Muda alcance para pessoal e alvo para você. Ação adicional pode lançar magias e ativar engenhocas.',
        requiresCircle: 4,
      },
    ],
    bookPage: 210,
  },
  {
    id: 'voo',
    name: 'Voo',
    circle: 3,
    school: 'transmutacao',
    execution: 'padrao',
    range: 'pessoal',
    duration: 'cena',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Arcanista', 'Bardo'],
    baseEffect:
      'Recebe deslocamento de voo 12m. Voar é simples como andar — pode atacar e lançar magias normalmente. Quando termina, desce lentamente como Queda Suave.',
    augments: [
      { pmCost: 1, kind: 'muda', description: 'Muda alcance para toque e alvo para 1 criatura.' },
      {
        pmCost: 4,
        kind: 'muda',
        description: 'Muda duração para um dia.',
        requiresCircle: 4,
      },
      {
        pmCost: 4,
        kind: 'muda',
        description: 'Muda alcance para curto e alvo para até 10 criaturas.',
        requiresCircle: 4,
      },
    ],
    bookPage: 211,
  },
  {
    id: 'abencoar-alimentos',
    name: 'Abençoar Alimentos',
    circle: 1,
    school: 'transmutacao',
    execution: 'padrao',
    range: 'curto',
    duration: 'cena',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Clérigo', 'Druida'],
    baseEffect:
      'Purifica e abençoa alimento para 1 criatura, tornando comida suja/estragada/envenenada própria para consumo. Oferece 5 PV temporários ou 1 PM temporário. Bônus de alimentação duram um dia; cada personagem só pode receber um bônus de alimentação por dia.',
    augments: [
      {
        pmCost: 0,
        kind: 'muda',
        description:
          'Truque: o alimento é purificado (não causa efeito nocivo se estava estragado ou envenenado), mas não fornece bônus ao ser consumido.',
      },
      {
        pmCost: 1,
        kind: 'aumenta',
        description: 'Aumenta o número de alvos em +1.',
      },
      {
        pmCost: 1,
        kind: 'muda',
        description:
          'Muda a duração para permanente, o alvo passa a ser 1 frasco com água, adiciona componente material (pó de prata T$ 5). Em vez do normal, cria um frasco de água benta.',
      },
    ],
    bookPage: 178,
  },
  {
    id: 'acalmar-animal',
    name: 'Acalmar Animal',
    circle: 1,
    school: 'encantamento',
    execution: 'padrao',
    range: 'curto',
    duration: 'cena',
    saveType: 'vontade',
    resistance: 'anula',
    components: VG,
    classes: ['Clérigo', 'Druida'],
    baseEffect:
      'Animal (1 alvo) fica prestativo em relação ao conjurador — não sob controle, mas percebe palavras/ações favoravelmente. +10 em Adestramento e Diplomacia contra o animal. Alvo hostil em combate recebe +5 na resistência. Ação hostil dissipa a magia.',
    augments: [
      {
        pmCost: 1,
        kind: 'muda',
        description: 'Muda o alcance para médio.',
      },
      {
        pmCost: 1,
        kind: 'muda',
        description: 'Muda o alvo para 1 monstro ou espírito com Inteligência -5 ou -4.',
      },
      {
        pmCost: 5,
        kind: 'muda',
        description: 'Muda o alvo para 1 monstro ou espírito.',
        requiresCircle: 3,
      },
    ],
    bookPage: 178,
  },
  {
    id: 'adaga-mental',
    name: 'Adaga Mental',
    circle: 1,
    school: 'encantamento',
    execution: 'padrao',
    range: 'curto',
    duration: 'instantanea',
    saveType: 'vontade',
    resistance: 'parcial',
    components: VG,
    classes: ['Arcanista', 'Bardo'],
    baseEffect:
      'Adaga imaterial contra a mente de 1 criatura: 2d6 dano psíquico + atordoado 1 rodada. Resistência = metade do dano e evita a condição. Uma criatura só pode ficar atordoada por esta magia uma vez por cena.',
    augments: [
      {
        pmCost: 1,
        kind: 'muda',
        description:
          'Lança sem gesticular/pronunciar (permite lançar magia de armadura) e a adaga fica invisível. Alvo que falha na resistência não percebe a magia.',
      },
      {
        pmCost: 2,
        kind: 'muda',
        description:
          'Muda a duração para um dia. Além do normal, finca a adaga na mente do alvo — sabe direção/localização enquanto no mesmo mundo.',
      },
      {
        pmCost: 2,
        kind: 'aumenta',
        description: 'Aumenta o dano em +1d6.',
      },
    ],
    bookPage: 178,
  },
  {
    id: 'alarme',
    name: 'Alarme',
    circle: 1,
    school: 'abjuracao',
    execution: 'padrao',
    range: 'curto',
    duration: 'dia',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Arcanista', 'Bardo'],
    baseEffect:
      'Barreira invisível (esfera 9m de raio) detecta criaturas que tocarem/entrarem. Ao lançar, escolhe criaturas isentas. Aviso telepático (alerta o conjurador até 1km, acorda-o) ou sonoro (alerta todos em alcance longo).',
    augments: [
      {
        pmCost: 2,
        kind: 'muda',
        description: 'Muda o alcance para pessoal. A área é emanada a partir do conjurador.',
      },
      {
        pmCost: 5,
        kind: 'muda',
        description:
          'Além do normal, percebe efeitos de adivinhação que atravessem a área. Teste oposto de Misticismo revela vislumbre do rosto e localização aproximada.',
      },
      {
        pmCost: 9,
        kind: 'muda',
        description:
          'Muda a duração para um dia ou até descarregada e a resistência para Vontade anula. Descarregar quando intruso entra: falha na resistência = paralisado 1d4 rodadas. +10 em Sobrevivência para rastrear o intruso por 24h.',
      },
    ],
    bookPage: 178,
  },
  {
    id: 'aliado-animal',
    name: 'Aliado Animal',
    circle: 2,
    school: 'encantamento',
    execution: 'padrao',
    range: 'curto',
    duration: 'dia',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Clérigo', 'Druida'],
    baseEffect:
      'Vínculo mental com animal prestativo (1 alvo). Obedece no melhor de suas capacidades. Funciona como parceiro veterano de tipo à escolha entre animal, combatente, fortão, guardião, montaria ou perseguidor.',
    augments: [
      {
        pmCost: 1,
        kind: 'muda',
        description:
          'Muda o alvo para 1 animal Minúsculo e a duração para 1 semana. Animal se desloca até local designado (carta, item), espera até o fim da magia permitindo apenas criaturas escolhidas se aproximarem.',
      },
      {
        pmCost: 7,
        kind: 'muda',
        description: 'Muda o parceiro para mestre.',
        requiresCircle: 3,
      },
      {
        pmCost: 12,
        kind: 'muda',
        description:
          'Muda o alvo para 2 animais prestativos. Cada animal funciona como parceiro de tipo diferente (respeitando o limite de parceiros por nível).',
        requiresCircle: 4,
      },
    ],
    bookPage: 178,
  },
  {
    id: 'alterar-destino',
    name: 'Alterar Destino',
    circle: 5,
    school: 'adivinhacao',
    execution: 'reacao',
    range: 'pessoal',
    duration: 'instantanea',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Arcanista', 'Bardo'],
    baseEffect:
      'Visualiza todas as possibilidades de um evento: rola novamente um teste de resistência com +10, ou faz um inimigo repetir um ataque contra o conjurador com -10.',
    augments: [],
    bookPage: 179,
  },
  {
    id: 'alterar-memoria',
    name: 'Alterar Memória',
    circle: 4,
    school: 'encantamento',
    execution: 'padrao',
    range: 'toque',
    duration: 'instantanea',
    saveType: 'vontade',
    resistance: 'anula',
    components: VG,
    classes: ['Arcanista', 'Bardo'],
    baseEffect:
      'Invade a mente do alvo (1 criatura) e altera/apaga memórias da última hora.',
    augments: [
      {
        pmCost: 2,
        kind: 'muda',
        description: 'Muda o alcance para pessoal e o alvo para área cone de 4,5m.',
      },
      {
        pmCost: 5,
        kind: 'muda',
        description: 'Pode alterar ou apagar memórias das últimas 24 horas.',
      },
    ],
    bookPage: 179,
  },
  {
    id: 'alterar-tamanho',
    name: 'Alterar Tamanho',
    circle: 2,
    school: 'transmutacao',
    execution: 'padrao',
    range: 'curto',
    duration: 'dia',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Arcanista', 'Bardo'],
    baseEffect:
      'Aumenta ou diminui o tamanho de 1 item mundano em até três categorias (Enorme→Pequeno). Também muda consistência (rígido como pedra / flexível como seda) sem alterar RD/PV. Objeto de criatura involuntária: Vontade anula.',
    augments: [
      {
        pmCost: 1,
        kind: 'aumenta',
        description: 'Aumenta o número de alvos em +1.',
      },
      {
        pmCost: 2,
        kind: 'muda',
        description:
          'Muda o alcance para toque e o alvo para 1 criatura. Aumenta uma categoria de tamanho (equipamento se ajusta) e recebe Força +2. Alvo involuntário: Fortitude nega.',
      },
      {
        pmCost: 3,
        kind: 'muda',
        description:
          'Muda o alcance para toque e o alvo para 1 criatura. Diminui uma categoria de tamanho (equipamento se ajusta) e recebe Destreza +2. Alvo involuntário: Fortitude nega.',
        requiresCircle: 3,
      },
      {
        pmCost: 7,
        kind: 'muda',
        description:
          'Muda alcance para toque, alvo para 1 criatura, duração para permanente e resistência para Fortitude anula. Falha = tamanho Minúsculo, Força reduzida a -5, deslocamentos reduzidos a 3m.',
        requiresCircle: 4,
      },
    ],
    bookPage: 179,
  },
  {
    id: 'amarras-etereas',
    name: 'Amarras Etéreas',
    circle: 2,
    school: 'convocacao',
    execution: 'padrao',
    range: 'medio',
    duration: 'cena',
    saveType: 'reflexos',
    resistance: 'anula',
    components: VG,
    classes: ['Arcanista', 'Bardo'],
    baseEffect:
      'Três laços de energia agarram o alvo (1 criatura). Ação padrão + Atletismo destrói um laço (+1 laço por 5 pontos acima da CD). Cada laço: Defesa 10, 10 PV, RD 5, imune a dano mágico. Todos destruídos = magia dissipada. Afeta criaturas incorpóreas.',
    augments: [
      {
        pmCost: 2,
        kind: 'aumenta',
        description:
          'Aumenta o número de laços em +1 (limitado pelo círculo máximo que o conjurador lança).',
      },
      {
        pmCost: 2,
        kind: 'muda',
        description:
          'Muda o efeito: cria fio de energia esmeralda que prende o alvo a um ponto fixo no alcance (pode flutuar). Alvo não se afasta mais de 3m — nem fisicamente, nem por movimento planar. Fio: 20 PV, RD 10.',
      },
      {
        pmCost: 3,
        kind: 'muda',
        description:
          'Cada laço é destruído automaticamente com um ataque; laço destruído libera choque de 1d8+1 dano de essência na criatura amarrada.',
        requiresCircle: 3,
      },
      {
        pmCost: 4,
        kind: 'muda',
        description:
          'Como a variante do fio, mas em vez de fio cria uma corrente de energia com 20 PV e RD 40.',
      },
      {
        pmCost: 4,
        kind: 'muda',
        description:
          'Muda o alvo para cubo de 9m, duração para permanente, adiciona componente material (chave de esmeralda T$ 2.000). Nenhum movimento planar entra ou sai da área.',
      },
      {
        pmCost: 9,
        kind: 'muda',
        description:
          'Muda alcance para médio, área para esfera de 3m de raio, alvo para criaturas escolhidas. Fio de energia prende todos os alvos ao centro da área.',
      },
    ],
    bookPage: 179,
  },
  {
    id: 'ancora-dimensional',
    name: 'Âncora Dimensional',
    circle: 3,
    school: 'abjuracao',
    execution: 'padrao',
    range: 'curto',
    duration: 'cena',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Arcanista', 'Bardo'],
    baseEffect:
      'Alvo (1 criatura ou objeto) é envolvido por campo de força esmeralda que impede qualquer movimento planar — convocação (Salto Dimensional, Teletransporte), viagem astral, habilidade incorpóreo.',
    augments: [],
    bookPage: 179,
  },
  {
    id: 'animar-objetos',
    name: 'Animar Objetos',
    circle: 4,
    school: 'transmutacao',
    execution: 'padrao',
    range: 'medio',
    duration: 'cena',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Arcanista', 'Bardo'],
    baseEffect:
      'Concede vida a até 8 objetos Minúsculos ou Pequenos / 4 Médios / 2 Grandes / 1 Enorme. Cada objeto vira parceiro sob controle (não conta no limite). Comando mental por ação de movimento. Construtos: For/Des/PV por tamanho; sem Defesa/testes de resistência; falham em opostos. Podem sofrer ações hostis. Não afeta itens mágicos nem carregados. Estatísticas: Minúsculo For -3 Des 4 5PV; Pequeno For 0 Des 2 10PV; Médio For 0 Des 1 20PV; Grande For 2 Des 0 40PV; Enorme For 4 Des -2 80PV.',
    augments: [
      {
        pmCost: 5,
        kind: 'muda',
        description:
          'Muda a duração para permanente, adiciona componente material (prataria T$ 1.000). Máximo de objetos animados = metade do nível.',
      },
    ],
    bookPage: 179,
  },
  {
    id: 'anular-a-luz',
    name: 'Anular a Luz',
    circle: 3,
    school: 'necromancia',
    execution: 'padrao',
    range: 'pessoal',
    duration: 'definida',
    durationNote: 'ver texto (aura persistente até fim da cena; enjôo 1d4 rodadas)',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Clérigo', 'Druida'],
    baseEffect:
      'Onda de escuridão em esfera de 6m de raio. Dissipa magias até 3º círculo na área (Religião oposto por CD). Aliados na área: aura sombria, +4 Defesa até fim da cena. Inimigos ficam enjoados 1d4 rodadas (uma vez por cena). Anula Dispersar as Trevas (instantâneo).',
    augments: [
      {
        pmCost: 2,
        kind: 'aumenta',
        description: 'Aumenta o bônus na Defesa em +1.',
      },
      {
        pmCost: 4,
        kind: 'muda',
        description: 'Muda as magias dissipadas para até 4º círculo.',
        requiresCircle: 4,
      },
      {
        pmCost: 9,
        kind: 'muda',
        description: 'Muda as magias dissipadas para até 5º círculo.',
        requiresCircle: 5,
      },
    ],
    bookPage: 180,
  },
  {
    id: 'aparencia-perfeita',
    name: 'Aparência Perfeita',
    circle: 2,
    school: 'ilusao',
    execution: 'padrao',
    range: 'pessoal',
    duration: 'cena',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Arcanista', 'Bardo'],
    baseEffect:
      'Rosto idealizado, porte garboso, voz melodiosa. CAR 5+ recebe +2 no atributo; senão vira 5 (conta como bônus). +5 em Diplomacia e Enganação. Ao terminar, observadores suspeitam; pessoas que viram o alvo sob a magia sentem "algo errado" em condições normais. Ao fim da cena, gastar novamente os PM como ação livre mantém ativa. Não fornece PV/PM adicionais.',
    augments: [
      {
        pmCost: 1,
        kind: 'muda',
        description: 'Muda o alcance para toque e o alvo para 1 humanoide.',
      },
    ],
    bookPage: 180,
  },
  {
    id: 'aprisionamento',
    name: 'Aprisionamento',
    circle: 5,
    school: 'abjuracao',
    execution: 'completa',
    range: 'curto',
    duration: 'permanente',
    saveType: 'vontade',
    resistance: 'anula',
    components: ['verbal', 'gestual', 'material'],
    classes: ['Arcanista', 'Bardo'],
    baseEffect:
      'Prisão mágica para 1 criatura. Falha na resistência = sofre efeito; sucesso = imune por 1 semana. Preso: não respira, não come, não envelhece; adivinhações não localizam. Formas (componente material T$ 1.000 cada): Acorrentamento (paralisado, correntes de mitral); Contenção Mínima (2cm dentro de gema); Prisão Dimensional (semiplano protegido); Sepultamento (esfera enterrada); Sono Eterno (adormecido). Condição de libertação especificada ao lançar — não pode se basear em estatísticas intangíveis.',
    augments: [],
    bookPage: 180,
  },
  {
    id: 'area-escorregadia',
    name: 'Área Escorregadia',
    circle: 1,
    school: 'convocacao',
    execution: 'padrao',
    range: 'curto',
    duration: 'cena',
    saveType: 'reflexos',
    resistance: 'anula',
    components: VG,
    classes: ['Arcanista', 'Bardo'],
    baseEffect:
      'Superfície (quadrado de 3m ou 1 objeto) coberta por substância escorregadia. Criatura na área: Reflexos ou cai. Rodadas seguintes: movimento requer Acrobacia CD 10 (equilíbrio). Item afetado: portador testa resistência ou derruba/não usa.',
    augments: [
      {
        pmCost: 1,
        kind: 'aumenta',
        description: 'Aumenta a área em +1 quadrado de 1,5m.',
      },
      {
        pmCost: 2,
        kind: 'muda',
        description: 'Muda a CD dos testes de Acrobacia para 15.',
      },
      {
        pmCost: 5,
        kind: 'muda',
        description: 'Muda a CD dos testes de Acrobacia para 20.',
      },
    ],
    bookPage: 180,
  },
  {
    id: 'arma-espiritual',
    name: 'Arma Espiritual',
    circle: 1,
    school: 'convocacao',
    execution: 'padrao',
    range: 'pessoal',
    duration: 'cena',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Clérigo', 'Druida'],
    baseEffect:
      'Invoca a arma preferida da divindade flutuando ao lado. Uma vez por rodada, ao sofrer ataque corpo a corpo, reação faz a arma causar 2d6 dano do tipo da arma no atacante. Dissipa se conjurador morrer.',
    augments: [
      {
        pmCost: 1,
        kind: 'aumenta',
        description: 'Além do normal, a arma protege. +1 na Defesa.',
      },
      {
        pmCost: 2,
        kind: 'aumenta',
        description: 'Aumenta o bônus na Defesa em +1.',
      },
      {
        pmCost: 2,
        kind: 'muda',
        description:
          'Muda a duração para sustentada. Uma vez por rodada, ação livre faz a arma acertar automaticamente alvo adjacente. Se atacar, não contra-ataca até o próximo turno.',
      },
      {
        pmCost: 2,
        kind: 'muda',
        description: 'Muda o tipo do dano para essência.',
        requiresCircle: 2,
      },
      {
        pmCost: 2,
        kind: 'aumenta',
        description:
          'Aumenta o dano da arma em +1d6 (bônus máximo limitado pelo círculo máximo).',
      },
      {
        pmCost: 5,
        kind: 'aumenta',
        description:
          'Invoca duas armas — contra-ataca (ou ataca) duas vezes por rodada.',
        requiresCircle: 3,
      },
    ],
    bookPage: 180,
  },
  {
    id: 'arma-magica',
    name: 'Arma Mágica',
    circle: 1,
    school: 'transmutacao',
    execution: 'padrao',
    range: 'toque',
    duration: 'cena',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Arcanista', 'Bardo', 'Clérigo', 'Druida'],
    baseEffect:
      '1 arma empunhada vira mágica; +1 em ataque e dano (bônus de encanto). Se o conjurador empunhar, pode usar atributo-chave de magias em vez do original em ataque (não cumulativo).',
    augments: [
      {
        pmCost: 2,
        kind: 'aumenta',
        description:
          'Aumenta o bônus em +1 (bônus máximo limitado pelo círculo máximo).',
      },
      {
        pmCost: 2,
        kind: 'aumenta',
        description:
          'Arma causa +1d6 de dano de ácido, eletricidade, fogo ou frio (escolhido ao lançar). Uma vez apenas.',
      },
      {
        pmCost: 3,
        kind: 'muda',
        description: 'Muda o bônus de dano acima para +2d6.',
      },
    ],
    bookPage: 181,
  },
  {
    id: 'armamento-da-natureza',
    name: 'Armamento da Natureza',
    circle: 1,
    school: 'transmutacao',
    execution: 'padrao',
    range: 'toque',
    duration: 'cena',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Clérigo', 'Druida'],
    baseEffect:
      'Fortalece arma mundana primitiva (sem custo T$: bordão, clava, funda, tacape), arma natural ou ataque desarmado. Dano +1 passo, considerada mágica. Ao lançar, pode mudar tipo de dano (corte, impacto ou perfuração).',
    augments: [
      {
        pmCost: 1,
        kind: 'aumenta',
        description: 'Fornece +1 nos testes de ataque com a arma.',
      },
      {
        pmCost: 2,
        kind: 'muda',
        description: 'Muda a execução para ação de movimento.',
      },
      {
        pmCost: 3,
        kind: 'aumenta',
        description: 'Aumenta o bônus nos testes de ataque em +1.',
      },
      {
        pmCost: 5,
        kind: 'aumenta',
        description: 'Aumenta o dano da arma em mais um passo.',
      },
    ],
    bookPage: 181,
  },
  {
    id: 'assassino-fantasmagorico',
    name: 'Assassino Fantasmagórico',
    circle: 4,
    school: 'necromancia',
    execution: 'padrao',
    range: 'longo',
    duration: 'definida',
    durationNote: 'cena, até ser descarregada',
    saveType: 'vontade',
    resistance: 'anula',
    components: VG,
    classes: ['Arcanista', 'Bardo'],
    baseEffect:
      'Cria imagem do maior medo da vítima (1 criatura). Só ela vê; outros veem espectro sombrio. Espectro surge adjacente ao conjurador; Vontade da vítima: sucesso = percebe ilusão, dissipa. Falha: espectro flutua 18m/rodada em direção à vítima (fim do turno). Incorpóreo, imune a magias (exceto dissipadoras). Adjacente à vítima no fim do turno: Fortitude — sucesso = 6d6 dano de trevas (não reduz abaixo de 0 PV, não sangra); falha = colapsa a -1 PV sangrando. Desaparece se vítima inconsciente / fora do alcance longo / dissipado.',
    augments: [],
    bookPage: 181,
  },
  {
    id: 'augurio',
    name: 'Augúrio',
    circle: 2,
    school: 'adivinhacao',
    execution: 'completa',
    range: 'pessoal',
    duration: 'instantanea',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Clérigo', 'Druida'],
    baseEffect:
      'Prevê resultado de ação até 1h no futuro. Mestre rola 1d6 secreto: 2-6 magia funciona e retorna "felicidade" / "miséria" / "felicidade e miséria" / "nada". Resultado 1: falha e retorna "nada" (não distinguível de sucesso). Múltiplas lançadas sobre mesmo assunto sempre retornam o primeiro resultado.',
    augments: [
      {
        pmCost: 3,
        kind: 'muda',
        description:
          'Muda a execução para 1 minuto. Consulta divindade com pergunta sobre evento até 1 dia no futuro. 2-6: resposta (frase, profecia ou enigma) com pistas de caminho. Falha: sem resposta.',
        requiresCircle: 3,
      },
      {
        pmCost: 7,
        kind: 'muda',
        description:
          'Muda execução para 10 min e duração para 1 min. Consulta divindade — uma pergunta por rodada respondida por sim/não/não sei. Chance de falha por pergunta; falha = "não sei".',
        requiresCircle: 4,
      },
      {
        pmCost: 7,
        kind: 'muda',
        description: 'O mestre rola 1d12; a magia só falha em resultado 1.',
      },
      {
        pmCost: 12,
        kind: 'muda',
        description: 'O mestre rola 1d20; a magia só falha em resultado 1.',
      },
    ],
    bookPage: 181,
  },
  {
    id: 'aura-divina',
    name: 'Aura Divina',
    circle: 5,
    school: 'abjuracao',
    execution: 'padrao',
    range: 'pessoal',
    duration: 'cena',
    saveType: 'vontade',
    resistance: 'parcial',
    components: VG,
    classes: ['Clérigo'],
    baseEffect:
      'Emana aura brilhante em esfera de 9m de raio. Conjurador + aliados devotos da mesma divindade: imunes a encantamento, +10 Defesa e testes de resistência. Aliados não devotos: +5 Defesa e resistência. Inimigos que entrem: Vontade — falha = uma condição à escolha entre esmorecido/debilitado/lento até fim da cena. Teste refeito ao reentrar.',
    augments: [
      {
        pmCost: 2,
        kind: 'aumenta',
        description: 'Aumenta os bônus na Defesa e em testes de resistência em +1.',
      },
    ],
    bookPage: 182,
  },
  {
    id: 'aviso',
    name: 'Aviso',
    circle: 1,
    school: 'adivinhacao',
    execution: 'movimento',
    range: 'longo',
    duration: 'instantanea',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Arcanista', 'Bardo', 'Clérigo', 'Druida'],
    baseEffect:
      'Aviso telepático para 1 criatura (sem linha de efeito exigida). Escolha: Alerta (+5 no próximo Iniciativa e Percepção até fim da próxima cena); Mensagem (25 palavras, idioma comum); Localização (alvo sabe posição no momento — não atualiza).',
    augments: [
      {
        pmCost: 1,
        kind: 'aumenta',
        description: 'Aumenta o alcance em fator de 10 (90m→900m, 900m→9km etc).',
      },
      {
        pmCost: 1,
        kind: 'muda',
        description:
          'Se escolher mensagem, alvo envia resposta de até 25 palavras até fim do próximo turno.',
      },
      {
        pmCost: 2,
        kind: 'muda',
        description:
          'Se escolher localização, muda a duração para cena. Alvo sabe posição mesmo com movimento.',
      },
      {
        pmCost: 3,
        kind: 'aumenta',
        description: 'Aumenta o número de alvos em +1.',
      },
    ],
    bookPage: 182,
  },
  {
    id: 'banimento',
    name: 'Banimento',
    circle: 3,
    school: 'abjuracao',
    execution: 'completa',
    range: 'curto',
    duration: 'instantanea',
    saveType: 'vontade',
    resistance: 'parcial',
    components: VG,
    classes: ['Clérigo', 'Druida'],
    baseEffect:
      'Expulsa criatura não nativa de Arton. Execução: 1d3+1 rodadas (ritual). Nativa de outro mundo: teletransportada de volta. Morto-vivo: conexão com energia negativa rompida, reduzida a 0 PV. Sucesso na resistência = enjoada 1d4 rodadas. Itens que se oponham ao alvo (ex: água benta contra frios, tocha) aumentam CD em +2 por item — mestre decide validade.',
    augments: [
      {
        pmCost: 0,
        kind: 'muda',
        description:
          'Muda a resistência para nenhum. Devolve automaticamente uma criatura conjurada (por magia de convocação) para o plano de origem.',
      },
    ],
    bookPage: 182,
  },
  {
    id: 'barragem-elemental-de-vectorius',
    name: 'Barragem Elemental de Vectorius',
    circle: 5,
    school: 'evocacao',
    execution: 'padrao',
    range: 'longo',
    duration: 'instantanea',
    saveType: 'reflexos',
    resistance: 'parcial',
    components: VG,
    classes: ['Arcanista'],
    baseEffect:
      '4 esferas (ácido, eletricidade, fogo, frio) até ponto escolhido. Cada explode em área de 12m de raio: 6d6 dano do respectivo tipo. Reflexos = metade. Alvos podem ser diferentes por esfera; criatura em múltiplas resiste por esfera. Efeitos por falha: Ácido = vulnerável até fim da cena; Elétrica = atordoado 1 rodada (uma vez/cena); Fogo = em chamas; Frio = lento até fim da cena.',
    augments: [
      {
        pmCost: 5,
        kind: 'aumenta',
        description: 'Aumenta o dano de cada esfera em +2d6.',
      },
      {
        pmCost: 5,
        kind: 'muda',
        description:
          'Muda o tipo de dano de todas as esferas para essência (mas ainda causam os outros efeitos como se o tipo não mudasse).',
      },
    ],
    bookPage: 182,
  },
  {
    id: 'bencao',
    name: 'Bênção',
    circle: 1,
    school: 'encantamento',
    execution: 'padrao',
    range: 'curto',
    duration: 'cena',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Clérigo', 'Druida', 'Paladino'],
    baseEffect:
      'Aliados no alcance recebem +1 em testes de ataque e rolagens de dano. Anula Perdição.',
    augments: [
      {
        pmCost: 1,
        kind: 'muda',
        description:
          'Muda o alvo para 1 cadáver e a duração para 1 semana. Cadáver não se decompõe nem vira morto-vivo.',
      },
      {
        pmCost: 2,
        kind: 'aumenta',
        description:
          'Aumenta o bônus em +1 (bônus máximo limitado pelo círculo máximo).',
      },
    ],
    bookPage: 182,
  },
  {
    id: 'buraco-negro',
    name: 'Buraco Negro',
    circle: 5,
    school: 'convocacao',
    execution: 'completa',
    range: 'longo',
    duration: 'definida',
    durationNote: '3 rodadas',
    saveType: 'fortitude',
    resistance: 'parcial',
    components: VG,
    classes: ['Arcanista', 'Clérigo'],
    baseEffect:
      'Vácuo em espaço desocupado. Início de cada um dos 3 turnos seguintes: criaturas em alcance longo (incl. conjurador) fazem Fortitude. Falha = caído + puxado 30m. Objetos soltos também são puxados. Ação de movimento para se segurar em objeto fixo dá +2 na resistência. Criatura/objeto que inicia turno no espaço do buraco: gasta ação de movimento + Fortitude para se arrastar 1,5m. Falha = perde ação (pode gastar outra). Terminar turno no espaço = sugado, desaparece para sempre. Destino desconhecido (Sombria?).',
    augments: [
      {
        pmCost: 5,
        kind: 'muda',
        description: 'Muda o efeito para que o conjurador não seja afetado.',
      },
    ],
    bookPage: 182,
  },
  {
    id: 'campo-antimagia',
    name: 'Campo Antimagia',
    circle: 4,
    school: 'abjuracao',
    execution: 'padrao',
    range: 'pessoal',
    duration: 'sustentada',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Arcanista', 'Bardo'],
    baseEffect:
      'Barreira invisível 3m de raio acompanha o conjurador. Habilidades mágicas ou itens mágicos na área são suprimidos. Convocadas na área desaparecem — reaparecem quando termina o Campo (se a duração da magia convocadora ainda não terminou). Criaturas mágicas não são afetadas mas não podem usar magia dentro. Magias dissipadoras não dissipam o Campo; 2 Campos na mesma área não se neutralizam. Artefatos e deuses maiores imunes.',
    augments: [],
    bookPage: 183,
  },
  {
    id: 'campo-de-forca',
    name: 'Campo de Força',
    circle: 2,
    school: 'abjuracao',
    execution: 'padrao',
    range: 'pessoal',
    duration: 'cena',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Arcanista', 'Bardo'],
    baseEffect: 'Película protetora: 30 PV temporários.',
    augments: [
      {
        pmCost: 1,
        kind: 'muda',
        description:
          'Muda execução para reação e duração para instantânea. Em vez do normal, RD 30 contra o próximo dano.',
      },
      {
        pmCost: 3,
        kind: 'muda',
        description: 'Muda os PV temporários ou a RD para 50.',
        requiresCircle: 3,
      },
      {
        pmCost: 7,
        kind: 'muda',
        description: 'Muda os PV temporários ou a RD para 70.',
        requiresCircle: 4,
      },
      {
        pmCost: 7,
        kind: 'muda',
        description:
          'Muda alcance para curto, alvo para criatura/objeto sólido Enorme ou menor, duração para sustentada. Esfera imóvel e tremeluzente: nada passa (dano, criaturas, objetos); respiração normal. Criaturas na área: Reflexos para evitar aprisionamento (e sempre que o conjurador se concentrar).',
        requiresCircle: 4,
      },
      {
        pmCost: 9,
        kind: 'muda',
        description:
          'Como acima, mas conteúdo fica sem peso. Uma vez por rodada, ação livre flutua esfera 9m em uma direção.',
        requiresCircle: 4,
      },
    ],
    bookPage: 183,
  },
  {
    id: 'camuflagem-ilusoria',
    name: 'Camuflagem Ilusória',
    circle: 2,
    school: 'ilusao',
    execution: 'padrao',
    range: 'toque',
    duration: 'cena',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Arcanista', 'Bardo'],
    baseEffect:
      'Alvo (1 criatura) fica com imagem nublada — recebe efeitos de camuflagem leve.',
    augments: [
      {
        pmCost: 3,
        kind: 'muda',
        description:
          'Muda a duração para sustentada. Imagem mais distorcida: chance de falha da camuflagem leve sobe para 50%.',
      },
      {
        pmCost: 7,
        kind: 'muda',
        description: 'Muda o alcance para curto e o alvo para criaturas escolhidas.',
        requiresCircle: 4,
      },
    ],
    bookPage: 183,
  },
  {
    id: 'chuva-de-meteoros',
    name: 'Chuva de Meteoros',
    circle: 5,
    school: 'convocacao',
    execution: 'completa',
    range: 'longo',
    duration: 'instantanea',
    saveType: 'reflexos',
    resistance: 'parcial',
    components: VG,
    classes: ['Arcanista'],
    baseEffect:
      'Meteoros caem em quadrado 18m de lado. Criaturas: 15d6 impacto + 15d6 fogo + caídas + agarradas (escombros). Sucesso = metade do dano total, sem cair/agarrar. Escapar dos escombros: ação padrão + Atletismo. Área vira terreno difícil coberto de escombros + nuvem de poeira (camuflagem leve). Só a céu aberto.',
    augments: [
      {
        pmCost: 2,
        kind: 'aumenta',
        description:
          'Aumenta o número de meteoros: +2d6 impacto e +2d6 fogo.',
      },
    ],
    bookPage: 183,
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
