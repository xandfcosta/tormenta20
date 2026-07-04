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
  {
    id: 'circulo-da-justica',
    name: 'Círculo da Justiça',
    circle: 2,
    school: 'abjuracao',
    execution: 'completa',
    range: 'curto',
    duration: 'dia',
    saveType: 'vontade',
    resistance: 'parcial',
    components: VG,
    classes: ['Clérigo', 'Druida', 'Paladino'],
    baseEffect:
      'Também conhecida como Lágrimas de Hynnn. Esfera 9m. Criaturas: -10 em Acrobacia/Enganação/Furtividade/Ladinagem, não podem mentir deliberadamente (mas podem evadir/omitir). Sucesso na resistência: penalidades reduzidas a -5, pode mentir.',
    augments: [
      {
        pmCost: 1,
        kind: 'muda',
        description:
          'Muda execução para padrão, alcance para pessoal, alvo para conjurador, duração para cena, resistência para nenhuma. Criaturas/objetos invisíveis em alcance curto ficam visíveis (sem dissipar); ao sair do alcance voltam a ficar invisíveis.',
      },
      {
        pmCost: 3,
        kind: 'muda',
        description:
          'Muda as penalidades nas perícias para -10 (se passar na resistência) e -20 (se falhar).',
        requiresCircle: 4,
      },
      {
        pmCost: 7,
        kind: 'muda',
        description:
          'Muda a duração para permanente e adiciona componente material (balança de prata T$ 5.000).',
      },
    ],
    bookPage: 183,
  },
  {
    id: 'circulo-da-restauracao',
    name: 'Círculo da Restauração',
    circle: 4,
    school: 'evocacao',
    execution: 'padrao',
    range: 'curto',
    duration: 'definida',
    durationNote: '5 rodadas',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Clérigo', 'Druida', 'Paladino'],
    baseEffect:
      'Círculo de luz (esfera 3m). Criatura viva que termina turno dentro: recupera 3d8+3 PV e 1 PM. Mortos-vivos e criaturas vulneráveis a luz perdem PV/PM na mesma quantidade. Máximo 5 PM por dia recuperados por esta magia.',
    augments: [
      {
        pmCost: 2,
        kind: 'aumenta',
        description: 'Aumenta a regeneração de PV em 1d8+1.',
      },
    ],
    bookPage: 184,
  },
  {
    id: 'colera-de-azgher',
    name: 'Cólera de Azgher',
    circle: 4,
    school: 'evocacao',
    execution: 'padrao',
    range: 'medio',
    duration: 'instantanea',
    saveType: 'reflexos',
    resistance: 'parcial',
    components: VG,
    classes: ['Clérigo', 'Druida', 'Paladino'],
    baseEffect:
      'Fulgor dourado em esfera 6m. Criaturas: cegas 1d4 rodadas + em chamas + 10d6 dano de fogo (mortos-vivos 10d8). Sucesso: sem cegueira/chamas, metade do dano.',
    augments: [
      {
        pmCost: 2,
        kind: 'aumenta',
        description: 'Aumenta o dano em +2d6 (+2d8 contra mortos-vivos).',
      },
      {
        pmCost: 2,
        kind: 'aumenta',
        description: 'Aumenta a área em +6m de raio.',
      },
      {
        pmCost: 5,
        kind: 'muda',
        description:
          'Luz purificadora do Deus-Sol dissipa todas as magias de necromancia ativas na área.',
        requiresCircle: 5,
      },
    ],
    bookPage: 184,
  },
  {
    id: 'coluna-de-chamas',
    name: 'Coluna de Chamas',
    circle: 3,
    school: 'evocacao',
    execution: 'padrao',
    range: 'longo',
    duration: 'instantanea',
    saveType: 'reflexos',
    resistance: 'metade',
    components: VG,
    classes: ['Clérigo', 'Druida', 'Paladino'],
    baseEffect:
      'Pilar de fogo sagrado (cilindro 3m raio, 30m altura): 6d6 dano de fogo + 6d6 dano de luz em criaturas e objetos livres.',
    augments: [
      {
        pmCost: 1,
        kind: 'aumenta',
        description: 'Aumenta o dano de fogo em +1d6.',
      },
      {
        pmCost: 1,
        kind: 'aumenta',
        description: 'Aumenta o dano de luz em +1d6.',
      },
    ],
    bookPage: 184,
  },
  {
    id: 'comando',
    name: 'Comando',
    circle: 1,
    school: 'encantamento',
    execution: 'padrao',
    range: 'curto',
    duration: 'definida',
    durationNote: '1 rodada',
    saveType: 'vontade',
    resistance: 'anula',
    components: VG,
    classes: ['Clérigo', 'Druida', 'Paladino'],
    baseEffect:
      'Ordem irresistível (1 humanoide deve ouvir; não precisa entender). Falha na resistência = obedece no próprio turno. Escolha: Fuja (afasta usando todas ações), Largue (solta itens, ação livre, não pega até próximo turno), Pare (pasmo, uma vez/cena), Senta (senta ou desce ao chão, não levanta até próximo turno), Venha (aproxima usando todas ações).',
    augments: [
      {
        pmCost: 1,
        kind: 'muda',
        description: 'Muda o alvo para 1 criatura.',
      },
      {
        pmCost: 2,
        kind: 'aumenta',
        description: 'Aumenta a quantidade de alvos em +1.',
      },
    ],
    bookPage: 184,
  },
  {
    id: 'compreensao',
    name: 'Compreensão',
    circle: 1,
    school: 'adivinhacao',
    execution: 'padrao',
    range: 'toque',
    duration: 'cena',
    saveType: 'vontade',
    resistance: 'anula',
    components: VG,
    classes: ['Arcanista', 'Bardo', 'Clérigo', 'Druida', 'Paladino'],
    baseEffect:
      'Toque em texto: entende palavras sem conhecer idioma. Toque em criatura inteligente: comunica sem idioma comum. Toque em não-inteligente (animal): percebe sentimentos. Ação de movimento: ouve pensamentos (alvo involuntário faz Vontade para bloquear).',
    augments: [
      {
        pmCost: 1,
        kind: 'muda',
        description: 'Muda o alcance para curto.',
      },
      {
        pmCost: 2,
        kind: 'muda',
        description:
          'Muda alcance para curto, alvo para criaturas escolhidas. Entende todas; ouve pensamentos de uma por vez.',
      },
      {
        pmCost: 2,
        kind: 'muda',
        description:
          'Muda alvo para 1 criatura. Vasculha pensamentos para extrair informação. Vontade anula. Mestre decide se alvo sabe.',
        requiresCircle: 2,
      },
      {
        pmCost: 5,
        kind: 'muda',
        description:
          'Muda alcance para pessoal e alvo para conjurador. Fala, entende e escreve qualquer idioma.',
        requiresCircle: 3,
      },
    ],
    bookPage: 184,
  },
  {
    id: 'comunhao-com-a-natureza',
    name: 'Comunhão com a Natureza',
    circle: 3,
    school: 'adivinhacao',
    execution: 'completa',
    range: 'pessoal',
    duration: 'dia',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Clérigo', 'Druida', 'Paladino'],
    baseEffect:
      'União com natureza local revela informações da região (dia de viagem). 6d4 dados de auxílio. Em teste de perícia em área natural: gasta 2d4 (+2d4 por círculo acima do 3º disponível) e adiciona ao teste. Termina se acabarem os dados.',
    augments: [
      {
        pmCost: 1,
        kind: 'muda',
        description:
          'Muda execução para 1 min, duração para instantânea. Descobre 1d4+1 informações sobre: terreno, animais, vegetais, minerais, cursos d\'água, criaturas antinaturais em região natural.',
      },
      {
        pmCost: 3,
        kind: 'aumenta',
        description: 'Aumenta o número de dados de auxílio em +2.',
      },
      {
        pmCost: 4,
        kind: 'muda',
        description: 'Muda o tipo dos dados de auxílio para d6.',
      },
      {
        pmCost: 8,
        kind: 'muda',
        description: 'Muda o tipo dos dados de auxílio para d8.',
      },
    ],
    bookPage: 184,
  },
  {
    id: 'conceder-milagre',
    name: 'Conceder Milagre',
    circle: 4,
    school: 'encantamento',
    execution: 'padrao',
    range: 'toque',
    duration: 'definida',
    durationNote: 'permanente até ser descarregada',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Clérigo', 'Druida', 'Paladino'],
    baseEffect:
      'Transfere poder divino a 1 criatura. Escolha magia até 2º círc conhecida; alvo lança 1x sem pagar PM base (aprimoramentos usam PM do alvo). Conjurador sofre -3 PM até ser descarregada.',
    augments: [
      {
        pmCost: 4,
        kind: 'muda',
        description: 'Muda o círculo da magia concedida para 3º e a penalidade de PM para -6.',
      },
    ],
    bookPage: 184,
  },
  {
    id: 'concentracao-de-combate',
    name: 'Concentração de Combate',
    circle: 1,
    school: 'adivinhacao',
    execution: 'livre',
    range: 'pessoal',
    duration: 'definida',
    durationNote: '1 rodada',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Arcanista', 'Bardo'],
    baseEffect:
      'Amplia percepção — em teste de ataque rola 2 dados e usa o melhor.',
    augments: [
      {
        pmCost: 2,
        kind: 'muda',
        description: 'Muda a execução para padrão e a duração para cena.',
        requiresCircle: 2,
      },
      {
        pmCost: 5,
        kind: 'muda',
        description:
          'Além do normal, ao atacar o conjurador, inimigo rola 2 dados e usa o pior.',
        requiresCircle: 3,
      },
      {
        pmCost: 9,
        kind: 'muda',
        description:
          'Muda execução para padrão, alcance para curto, alvo para criaturas escolhidas, duração para cena.',
        requiresCircle: 4,
      },
      {
        pmCost: 14,
        kind: 'muda',
        description:
          'Muda execução para padrão e duração para 1 dia. Sexto sentido: imune a surpreendido/desprevenido, +10 Defesa e Reflexos.',
        requiresCircle: 5,
      },
    ],
    bookPage: 185,
  },
  {
    id: 'condicao',
    name: 'Condição',
    circle: 2,
    school: 'adivinhacao',
    execution: 'padrao',
    range: 'curto',
    duration: 'cena',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Clérigo', 'Druida', 'Paladino'],
    baseEffect:
      'Conhece posição e status (PV, condições, magias ativas) de até 5 alvos. Distância não importa; deixa de detectar apenas se morrer ou mudar de plano.',
    augments: [
      {
        pmCost: 1,
        kind: 'aumenta',
        description: 'Aumenta o número de alvos em +1.',
      },
      {
        pmCost: 1,
        kind: 'muda',
        description: 'Muda a duração para 1 dia.',
      },
    ],
    bookPage: 185,
  },
  {
    id: 'conjurar-elemental',
    name: 'Conjurar Elemental',
    circle: 4,
    school: 'convocacao',
    execution: 'completa',
    range: 'medio',
    duration: 'sustentada',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Arcanista', 'Bardo'],
    baseEffect:
      'Transforma porção de elemento inerte (fogueira, tocha, poça) em elemental Grande (ar/água/fogo/terra). Parceiro mestre: destruidor (habilidade 2 PM) + 1 tipo do elemento. Auxilia apenas conjurador, não conta no limite. Ar: assassino/perseguidor/vigilante, dano eletricidade. Água: ajudante/guardião/médico, dano frio. Fogo: atirador/combatente/fortão, dano fogo. Terra: combatente/guardião/montaria, dano impacto.',
    augments: [
      {
        pmCost: 5,
        kind: 'muda',
        description:
          'Elemental muda para Enorme e recebe dois tipos de parceiro do seu elemento.',
      },
      {
        pmCost: 5,
        kind: 'muda',
        description:
          'Convoca um elemental de cada tipo. Ao lançar, escolhe se cada auxilia conjurador ou aliado no alcance.',
        requiresCircle: 5,
      },
    ],
    bookPage: 185,
  },
  {
    id: 'consagrar',
    name: 'Consagrar',
    circle: 1,
    school: 'evocacao',
    execution: 'padrao',
    range: 'longo',
    duration: 'dia',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Clérigo', 'Druida', 'Paladino'],
    baseEffect:
      'Enche esfera de 9m com energia positiva. Cura por efeitos de luz é maximizada na área — inclui dano em mortos-vivos por esses efeitos (ex: Curar Ferimentos cura automaticamente 18 PV). Não pode ser lançada em área com símbolo dedicado a outra divindade. Anula Profanar.',
    augments: [
      {
        pmCost: 1,
        kind: 'aumenta',
        description: 'Além do normal, mortos-vivos na área sofrem -2 em testes e Defesa.',
      },
      {
        pmCost: 2,
        kind: 'aumenta',
        description:
          'Aumenta as penalidades para mortos-vivos em -1 (máximo limitado pelo círculo máximo).',
      },
      {
        pmCost: 9,
        kind: 'muda',
        description:
          'Muda execução para 1 hora, duração para permanente, adiciona componente material (incenso e óleos T$ 1.000).',
        requiresCircle: 4,
      },
    ],
    bookPage: 186,
  },
  {
    id: 'contato-extraplanar',
    name: 'Contato Extraplanar',
    circle: 2,
    school: 'adivinhacao',
    execution: 'completa',
    range: 'pessoal',
    duration: 'dia',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Arcanista', 'Bardo'],
    baseEffect:
      'Mente viaja a outro plano; contrata entidade (gênio/demônio) que se alimenta de mana. 6d6 dados de auxílio. Em teste de perícia: gasta 1d6 (+1d6 por círc acima do 3º) e adiciona ao teste. Cada "6" rolado suga 1 PM. Termina se gastar dados / ficar sem PM / fim do dia.',
    augments: [
      {
        pmCost: 2,
        kind: 'aumenta',
        description: 'Aumenta o número de dados de auxílio em +1.',
      },
      {
        pmCost: 8,
        kind: 'muda',
        description:
          'Muda os dados de auxílio para d12. Resultado 12 suga 2 PM.',
        requiresCircle: 4,
      },
    ],
    bookPage: 186,
  },
  {
    id: 'controlar-a-gravidade',
    name: 'Controlar a Gravidade',
    circle: 4,
    school: 'transmutacao',
    execution: 'padrao',
    range: 'medio',
    duration: 'sustentada',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Arcanista', 'Bardo'],
    baseEffect:
      'Controla gravidade em cubo de 12m. Ação padrão muda efeito. Aumentar: início do turno = Atletismo — sucesso fatigado, falha fatigado + caído. Inverter: criaturas/objetos "caem" 12m ao teto em 1 rodada; obstáculo = 1d6 impacto por 1,5m; senão flutuam; voadoras normais; Reflexos para agarrar. Reduzir: Médios ou menores voam 6m; +20 Atletismo escalar/saltar; -2 ataque instável.',
    augments: [],
    bookPage: 186,
  },
  {
    id: 'controlar-agua',
    name: 'Controlar Água',
    circle: 3,
    school: 'transmutacao',
    execution: 'padrao',
    range: 'longo',
    duration: 'cena',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Clérigo', 'Druida', 'Paladino'],
    baseEffect:
      'Controla água em esfera 30m. Congelar: água mundana congela, nadadores imóveis; escapar = padrão + Atletismo/Acrobacia. Derreter: gelo vira água, termina. Enchente: eleva nível 4,5m ou muda alvo p/ embarcação (+3m deslocamento). Evaporar: água/gelo evaporam; elementais/plantas monstruosas/imunes a frio 10d8 fogo (outros metade, Fortitude reduz metade). Partir: diminui nível 4,5m; caminho seco em raso; redemoinho em profundo; elementais lentos.',
    augments: [
      {
        pmCost: 2,
        kind: 'aumenta',
        description: 'Aumenta o dano em +2d8.',
      },
    ],
    bookPage: 186,
  },
  {
    id: 'controlar-fogo',
    name: 'Controlar Fogo',
    circle: 2,
    school: 'evocacao',
    execution: 'padrao',
    range: 'curto',
    duration: 'cena',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Clérigo', 'Druida', 'Paladino'],
    baseEffect:
      'Cria/molda/move/extingue chamas. Chamejar: armas +1d6 fogo (inclui naturais/desarmado). Esquentar: 1 objeto sofre 1d6 fogo/rodada + causa dano a quem segura/veste; pode pegar fogo; padrão + água resfria. Extinguir: 1 chama Grande ou menor apagada, nuvem 3m raio (camuflagem leve). Modelar: 1 chama Grande ou menor move 9m/rodada (ação livre); atravessa criatura = 2d6 fogo (1x/rodada por alvo).',
    augments: [
      {
        pmCost: 1,
        kind: 'muda',
        description:
          'Muda duração para sustentada, resistência para Reflexos metade. Labaredas: ação de movimento projeta labareda em alvo curto = 4d6 fogo (Reflexos metade).',
      },
      {
        pmCost: 2,
        kind: 'aumenta',
        description: 'Aumenta o dano em +1d6 (exceto do efeito chamejar).',
      },
      {
        pmCost: 3,
        kind: 'muda',
        description:
          'Muda alvo para 1 criatura composta de fogo/lava/magma (elemental do fogo), resistência Fortitude parcial. Falha = reduzida a 0 PV. Sucesso = 5d6 dano.',
      },
    ],
    bookPage: 187,
  },
  {
    id: 'controlar-madeira',
    name: 'Controlar Madeira',
    circle: 2,
    school: 'transmutacao',
    execution: 'padrao',
    range: 'medio',
    duration: 'cena',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Clérigo', 'Druida', 'Paladino'],
    baseEffect:
      '1 objeto de madeira Grande ou menor. Objeto de criatura involuntária: Vontade anula. Fortalecer: armas dano +1 passo, escudos +2 Defesa (cumulativo), itens +5 RD e dobra PV. Modelar: muda forma (galho→espada, tronco→caixa). Sem mecanismos complexos ou consumíveis. Repelir: ataques com arma repelida falham; portas se abrem; objetos desviam. Retorcer: porta emperra (For CD 25); armas/itens -5 perícia; escudos sem bônus (penalidades mantém); barco afunda ao fim da cena.',
    augments: [
      {
        pmCost: 1,
        kind: 'muda',
        description:
          'Muda alcance para pessoal, alvo para conjurador, duração para 1 dia. Vira árvore Grande — não fala nem age, percebe arredores. Ataque dissipa. Sobrevivência CD 30 revela.',
      },
      {
        pmCost: 3,
        kind: 'muda',
        description:
          'Muda alvo para quadrado 9m e duração para cena. Vegetação fica rígida/afiada: terreno difícil + 1d6 corte por 1,5m avançado.',
      },
      {
        pmCost: 7,
        kind: 'muda',
        description: 'Muda o tamanho do alvo para Enorme ou menor.',
        requiresCircle: 3,
      },
      {
        pmCost: 12,
        kind: 'muda',
        description: 'Muda o tamanho do alvo para Colossal ou menor.',
        requiresCircle: 4,
      },
    ],
    bookPage: 187,
  },
  {
    id: 'controlar-o-clima',
    name: 'Controlar o Clima',
    circle: 4,
    school: 'transmutacao',
    execution: 'completa',
    range: 'longo',
    duration: 'definida',
    durationNote: '4d12 horas',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Clérigo', 'Druida', 'Paladino'],
    baseEffect:
      'Muda clima em esfera de 2km: chuva, neve, ventos, névoa. Efeitos em Cap 6: O Mestre.',
    augments: [
      {
        pmCost: 1,
        kind: 'muda',
        description: 'Muda o raio para 3km e a duração para 1d4 dias.',
        classOnly: 'druidas',
      },
    ],
    bookPage: 187,
  },
  {
    id: 'controlar-o-tempo',
    name: 'Controlar o Tempo',
    circle: 5,
    school: 'transmutacao',
    execution: 'padrao',
    range: 'pessoal',
    duration: 'definida',
    durationNote: 'veja texto',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Arcanista', 'Bardo'],
    baseEffect:
      'Congelar o tempo: bolha do próprio tamanho — 3 rodadas locais (2 turnos extras) sem efeitos contínuos; efeitos que gerar não saem da área ocupada ao lançar; sem preparar ações. Saltar no tempo: até 5 criaturas voluntárias saltam 1-24h para o futuro; ressurgem no mesmo lugar; obstáculo = área vazia mais próxima. Voltar no tempo: desfaz rodada anterior (inclui PV/PM exceto custo desta magia); conjurador é o único que lembra; só 1x por rodada.',
    augments: [],
    bookPage: 187,
  },
  {
    id: 'controlar-plantas',
    name: 'Controlar Plantas',
    circle: 1,
    school: 'transmutacao',
    execution: 'padrao',
    range: 'curto',
    duration: 'cena',
    saveType: 'reflexos',
    resistance: 'anula',
    components: VG,
    classes: ['Clérigo', 'Druida', 'Paladino'],
    baseEffect:
      'Área com vegetação (quadrado 9m). Plantas se enroscam — falha = enredado. Libertar: padrão + Acrobacia ou Atletismo. Terreno difícil. Início do turno: novo Reflexos tenta enredar.',
    augments: [
      {
        pmCost: 0,
        kind: 'muda',
        description:
          'Truque: muda área para alvo (1 planta) e resistência para nenhuma. Planta se move como animada, sem causar dano ou atrapalhar concentração.',
      },
      {
        pmCost: 1,
        kind: 'muda',
        description:
          'Muda duração para instantânea. Plantas na área diminuem (podadas): terreno normal, sem camuflagem. Dissipa Controlar Plantas normal.',
      },
      {
        pmCost: 1,
        kind: 'aumenta',
        description:
          'Além do normal, criaturas que falhem na resistência também ficam imóveis.',
      },
      {
        pmCost: 2,
        kind: 'muda',
        description:
          'Muda alcance para pessoal, área para alvo (conjurador), resistência para nenhuma. Comunica com plantas (atitude prestativa), Diplomacia com plantas. Respostas simplórias.',
      },
    ],
    bookPage: 188,
  },
  {
    id: 'controlar-terra',
    name: 'Controlar Terra',
    circle: 3,
    school: 'transmutacao',
    execution: 'padrao',
    range: 'longo',
    duration: 'instantanea',
    saveType: 'reflexos',
    resistance: 'metade',
    components: VG,
    classes: ['Clérigo', 'Druida', 'Paladino'],
    baseEffect:
      '9 cubos 1,5m — manipula densidade/forma de terra/pedra/lama/argila/areia. Amolecer: teto/coluna/suporte = desabamento 10d6 impacto (Reflexos metade); piso vira terreno difícil. Solidificar: areia vira terra/pedra, criaturas com pés na superfície agarradas (padrão + Atletismo escapa).',
    augments: [
      {
        pmCost: 1,
        kind: 'aumenta',
        description: 'Aumenta o número de cubos de 1,5m em +2.',
      },
      {
        pmCost: 1,
        kind: 'muda',
        description:
          'Muda alcance para pessoal, alvo para conjurador, duração para 1 dia. Funde-se com superfície de pedra/lama/areia. Ação livre volta ao adjacente (dissipa). Sem falar/agir; percebe arredores. Objeto destruído dissipa + 10d6 impacto.',
      },
    ],
    bookPage: 188,
  },
  {
    id: 'convocacao-instantanea',
    name: 'Convocação Instantânea',
    circle: 3,
    school: 'convocacao',
    execution: 'padrao',
    range: 'ilimitado',
    duration: 'instantanea',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Arcanista', 'Bardo'],
    baseEffect:
      'Invoca 1 objeto de até 2 espaços de qualquer lugar para a mão. Objeto deve ter runa pessoal (T$ 5). Se estiver com outra criatura: sabe localização/carregador (ou descrição física).',
    augments: [
      {
        pmCost: 1,
        kind: 'aumenta',
        description:
          'Até 1h após lançar, ação de movimento envia o item de volta ao local anterior.',
      },
      {
        pmCost: 1,
        kind: 'muda',
        description:
          'Muda alvo para baú Médio, duração para permanente, adiciona sacrifício 1 PM. Baú no Éter Entre Mundos com 20 espaços; itens de qualquer tamanho cabem. Ação padrão invoca/envia. Componente material: baú de matéria-prima T$ 1.000 + miniatura T$ 100.',
      },
      {
        pmCost: 2,
        kind: 'aumenta',
        description: 'Aumenta o número de alvos em +1.',
      },
      {
        pmCost: 2,
        kind: 'muda',
        description:
          'Muda o alvo para 1 objeto de até 10 espaços. Muito grande/pesado surge em espaço adjacente à escolha.',
      },
    ],
    bookPage: 188,
  },
  {
    id: 'cranio-voador-de-vladislav',
    name: 'Crânio Voador de Vladislav',
    circle: 2,
    school: 'necromancia',
    execution: 'padrao',
    range: 'medio',
    duration: 'instantanea',
    saveType: 'fortitude',
    resistance: 'parcial',
    components: VG,
    classes: ['Arcanista', 'Bardo'],
    baseEffect:
      'Crânio envolto em energia negativa: 4d8+4 dano trevas + som horrendo (abalado no alvo e inimigos em 3m; já abalados = apavorados 1d4 rodadas). Sucesso: metade dano, sem condição.',
    augments: [
      {
        pmCost: 2,
        kind: 'aumenta',
        description: 'Aumenta o dano em +1d8+1.',
      },
      {
        pmCost: 2,
        kind: 'aumenta',
        description: 'Aumenta o número de alvos em +1.',
      },
    ],
    bookPage: 188,
  },
  {
    id: 'criar-elementos',
    name: 'Criar Elementos',
    circle: 1,
    school: 'convocacao',
    execution: 'padrao',
    range: 'curto',
    duration: 'instantanea',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Clérigo', 'Druida', 'Paladino'],
    baseEffect:
      'Cria pequena porção de elemento (real, não mágico). Água: enche recipiente Minúsculo ou cria cubo de gelo Minúsculo. Ar: vento fraco em 1,5m² purifica gás/fumaça ou remove névoa 1 rodada. Fogo: chama como tocha (segurar sem queimar) ou surge em 1,5m² (1d6 fogo, Reflexos evita em chamas). Terra: cubo Minúsculo terra/argila/pedra. Também pode criar objetos simples de gelo/terra/pedra.',
    augments: [
      {
        pmCost: 1,
        kind: 'aumenta',
        description:
          'Aumenta a quantidade em um passo (categoria de tamanho para água/terra, +1 quadrado 1,5m para ar/fogo).',
      },
      {
        pmCost: 1,
        kind: 'muda',
        description:
          'Muda para alvo 1 criatura ou objeto, resistência Reflexos metade. Água/terra: arremessa cubo/objeto = 2d4 impacto (+1 passo por categoria acima Minúsculo). Cubo se desfaz.',
      },
      {
        pmCost: 1,
        kind: 'aumenta',
        description: 'Se escolheu fogo, aumenta o dano inicial em +1d6.',
      },
    ],
    bookPage: 188,
  },
  {
    id: 'cupula-de-repulsao',
    name: 'Cúpula de Repulsão',
    circle: 4,
    school: 'abjuracao',
    execution: 'completa',
    range: 'pessoal',
    duration: 'sustentada',
    saveType: 'vontade',
    resistance: 'anula',
    components: VG,
    classes: ['Clérigo', 'Druida', 'Paladino'],
    baseEffect:
      'Cúpula invisível impede aproximação de tipo/raça escolhida. Criatura afetada que tente ficar adjacente (3m): Vontade — falha = gasta ação, tenta novamente na rodada seguinte. Impede corpo a corpo, não à distância. Se ultrapassar 3m: rompe cúpula, dissipa.',
    augments: [
      {
        pmCost: 2,
        kind: 'muda',
        description:
          'Cúpula impede aproximação a menos de 4,5m (2 quadrados entre conjurador e criaturas).',
      },
      {
        pmCost: 5,
        kind: 'aumenta',
        description:
          'Além do normal, criaturas afetadas também testam ao fazer ataque/efeito à distância. Falha = desviado pela cúpula.',
        requiresCircle: 5,
      },
    ],
    bookPage: 189,
  },
  {
    id: 'deflagracao-de-mana',
    name: 'Deflagração de Mana',
    circle: 5,
    school: 'evocacao',
    execution: 'completa',
    range: 'pessoal',
    duration: 'instantanea',
    saveType: 'fortitude',
    resistance: 'parcial',
    components: VG,
    classes: ['Arcanista', 'Bardo'],
    baseEffect:
      'Emana energia (esfera 15m). Criaturas: 150 dano essência + itens mágicos (exceto artefatos) viram mundanos. Conjurador não é afetado. Fortitude sucesso: metade dano, itens voltam após 1 dia.',
    augments: [
      {
        pmCost: 1,
        kind: 'aumenta',
        description: 'Aumenta o dano em +10.',
      },
      {
        pmCost: 5,
        kind: 'muda',
        description: 'Afeta apenas criaturas à escolha.',
      },
    ],
    bookPage: 189,
  },
  {
    id: 'desejo',
    name: 'Desejo',
    circle: 5,
    school: 'transmutacao',
    execution: 'completa',
    range: 'longo',
    duration: 'definida',
    durationNote: 'veja texto',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Arcanista', 'Bardo'],
    baseEffect:
      'Mais poderosa das arcanas — altera realidade. Opções base: dissipa magia até 4º; transporta 10 criaturas voluntárias em alcance longo para qualquer plano; desfaz ataque da rodada anterior de criatura no alcance. Com sacrifício de 2 PM: cria item mundano até T$ 30.000; duplica magia até 4º (mantém componentes materiais); +1 em atributo (uma vez por atributo). Efeitos maiores possíveis mas com riscos — mestre decide.',
    augments: [],
    bookPage: 190,
  },
  {
    id: 'desespero-esmagador',
    name: 'Desespero Esmagador',
    circle: 2,
    school: 'encantamento',
    execution: 'padrao',
    range: 'pessoal',
    duration: 'instantanea',
    saveType: 'vontade',
    resistance: 'parcial',
    components: VG,
    classes: ['Arcanista', 'Bardo'],
    baseEffect:
      'Humanoides em cone 6m: fracos + frustrados até fim da cena (ou 1 rodada com sucesso).',
    augments: [
      {
        pmCost: 2,
        kind: 'muda',
        description: 'Em vez do normal, as condições são debilitado e esmorecido.',
      },
      {
        pmCost: 3,
        kind: 'muda',
        description: 'Em vez do normal, afeta qualquer tipo de criatura.',
      },
      {
        pmCost: 3,
        kind: 'aumenta',
        description:
          'Além do normal, criaturas que falhem na resistência ficam aos prantos (pasmas) por 1 rodada (uma vez por cena).',
        requiresCircle: 3,
      },
    ],
    bookPage: 190,
  },
  {
    id: 'desintegrar',
    name: 'Desintegrar',
    circle: 4,
    school: 'transmutacao',
    execution: 'padrao',
    range: 'medio',
    duration: 'instantanea',
    saveType: 'fortitude',
    resistance: 'parcial',
    components: VG,
    classes: ['Arcanista', 'Bardo'],
    baseEffect:
      'Raio esverdeado: 10d12 dano essência em 1 criatura/objeto. Sucesso: 2d12. PV a 0 ou menos = completamente desintegrado (só pó).',
    augments: [
      {
        pmCost: 4,
        kind: 'aumenta',
        description:
          'Aumenta o dano total em +2d12 e o dano mínimo em +1d12.',
      },
    ],
    bookPage: 190,
  },
  {
    id: 'despedacar',
    name: 'Despedaçar',
    circle: 1,
    school: 'evocacao',
    execution: 'padrao',
    range: 'curto',
    duration: 'instantanea',
    saveType: 'fortitude',
    resistance: 'parcial',
    components: VG,
    classes: ['Clérigo', 'Druida', 'Paladino'],
    baseEffect:
      'Som agudo em 1 criatura ou objeto mundano Pequeno: 1d8+2 impacto (dobro + ignora RD contra construto/objeto mundano) + atordoado 1 rodada (uma vez por cena). Fortitude reduz metade + evita atordoamento. Anula Transmutar Objetos.',
    augments: [
      {
        pmCost: 2,
        kind: 'aumenta',
        description: 'Aumenta o dano em +1d8+2.',
      },
      {
        pmCost: 2,
        kind: 'muda',
        description: 'Muda o alvo para objeto mundano Médio.',
        requiresCircle: 2,
      },
      {
        pmCost: 5,
        kind: 'muda',
        description: 'Muda o alvo para objeto mundano Grande.',
        requiresCircle: 3,
      },
      {
        pmCost: 9,
        kind: 'muda',
        description: 'Muda o alvo para objeto mundano Enorme.',
        requiresCircle: 4,
      },
      {
        pmCost: 14,
        kind: 'muda',
        description: 'Muda o alvo para objeto mundano Colossal.',
        requiresCircle: 5,
      },
      {
        pmCost: 5,
        kind: 'muda',
        description:
          'Muda alcance para pessoal e alvo para esfera 6m raio. Afeta todas as criaturas e objetos mundanos na área.',
      },
    ],
    bookPage: 190,
  },
  {
    id: 'despertar-consciencia',
    name: 'Despertar Consciência',
    circle: 3,
    school: 'encantamento',
    execution: 'completa',
    range: 'toque',
    duration: 'dia',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Clérigo', 'Druida', 'Paladino'],
    baseEffect:
      'Desperta consciência de 1 animal ou planta. Vira parceiro veterano de tipo à escolha (ajudante/combatente/fortão/guardião/médico/perseguidor/vigilante). Em parceiro existente: aumenta um tipo em um passo (1x/parceiro). Já mestre: recebe iniciante extra. Vira racional, INT -1, fala.',
    augments: [
      {
        pmCost: 4,
        kind: 'muda',
        description:
          'Muda alvo para 1 escultura mundana inanimada. Além do normal, alvo tem características de construto.',
      },
      {
        pmCost: 4,
        kind: 'muda',
        description: 'Muda a duração para permanente e adiciona penalidade de -3 PM.',
      },
    ],
    bookPage: 190,
  },
  {
    id: 'dificultar-deteccao',
    name: 'Dificultar Detecção',
    circle: 3,
    school: 'abjuracao',
    execution: 'padrao',
    range: 'toque',
    duration: 'dia',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Arcanista', 'Bardo'],
    baseEffect:
      'Oculta 1 criatura ou objeto contra meios mágicos de detecção (inclui Detectar Magia). Conjurador que use adivinhação para detectar: Vontade — falha = magia não funciona, PM gastos. Em criatura, protege também equipamento.',
    augments: [
      {
        pmCost: 4,
        kind: 'muda',
        description:
          'Muda alvo para cubo 9m. Criaturas/objetos na área recebem o efeito enquanto dentro.',
      },
      {
        pmCost: 4,
        kind: 'muda',
        description: 'Muda a duração para 1 semana.',
      },
    ],
    bookPage: 190,
  },
  {
    id: 'dispersar-as-trevas',
    name: 'Dispersar as Trevas',
    circle: 3,
    school: 'evocacao',
    execution: 'padrao',
    range: 'pessoal',
    duration: 'definida',
    durationNote: 'veja texto',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Clérigo', 'Druida', 'Paladino'],
    baseEffect:
      'Brilho multicolorido/divino em esfera 6m. Dissipa magias até 3º círc na área (Religião oposto por CD). Aliados: +4 resistências + redução trevas 10 até fim da cena. Inimigos: cegos 1d4 rodadas (1x/cena). Anula Anular a Luz (instantâneo).',
    augments: [
      {
        pmCost: 2,
        kind: 'aumenta',
        description: 'Aumenta o bônus nas resistências em +1.',
      },
      {
        pmCost: 4,
        kind: 'muda',
        description:
          'Muda alcance para curto, área para alvo 1 criatura, duração para cena. Alvo imune a efeitos de trevas.',
      },
      {
        pmCost: 4,
        kind: 'muda',
        description: 'Muda o círculo máximo de magias dissipadas para 4º.',
        requiresCircle: 4,
      },
      {
        pmCost: 9,
        kind: 'muda',
        description: 'Muda o círculo máximo de magias dissipadas para 5º.',
        requiresCircle: 5,
      },
    ],
    bookPage: 191,
  },
  {
    id: 'disfarce-ilusorio',
    name: 'Disfarce Ilusório',
    circle: 1,
    school: 'ilusao',
    execution: 'padrao',
    range: 'pessoal',
    duration: 'cena',
    saveType: 'vontade',
    resistance: 'desacredita',
    components: VG,
    classes: ['Arcanista', 'Bardo'],
    baseEffect:
      'Muda aparência (incluindo equipamento): altura/peso/pele/cabelo/voz. +10 Enganação para disfarce. Sem novas habilidades; equipamento continua funcional (espada disfarçada de bordão causa dano de espada).',
    augments: [
      {
        pmCost: 0,
        kind: 'muda',
        description:
          'Truque: muda alcance para toque, alvo para 1 criatura, duração para 1 semana. Pequena alteração inofensiva mas persistente (nariz vermelho, gerânio na cabeça — arrancado renasce).',
      },
      {
        pmCost: 1,
        kind: 'muda',
        description:
          'Muda alcance para curto e alvo para 1 objeto (ferro→moedas de ouro). +10 Enganação para falsificação.',
      },
      {
        pmCost: 1,
        kind: 'muda',
        description:
          'Muda alcance para curto e alvo para 1 criatura. Involuntária anula por Vontade.',
      },
      {
        pmCost: 2,
        kind: 'aumenta',
        description:
          'Ilusão inclui odores e sensações. Bônus em Enganação para disfarce sobe para +20.',
      },
      {
        pmCost: 3,
        kind: 'muda',
        description:
          'Muda alcance para curto e alvo para criaturas escolhidas. Cada uma pode ter aparência diferente. Involuntárias anulam por Vontade.',
        requiresCircle: 2,
      },
    ],
    bookPage: 191,
  },
  {
    id: 'duplicata-ilusoria',
    name: 'Duplicata Ilusória',
    circle: 4,
    school: 'ilusao',
    execution: 'padrao',
    range: 'medio',
    duration: 'cena',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Arcanista', 'Bardo'],
    baseEffect:
      'Cópia ilusória semirreal do conjurador — idêntica em aparência/som/cheiro, intangível. A cada turno escolhe ver/ouvir pela cópia ou corpo. Cópia reproduz todas as ações (inclui fala). Magias de toque+ podem originar da cópia. Movimento diferente do corpo: ação de movimento. Interação: Vontade percebe ilusão. Magias originadas dela são reais. Some se sair do alcance.',
    augments: [
      {
        pmCost: 3,
        kind: 'aumenta',
        description: 'Cria uma cópia adicional.',
      },
    ],
    bookPage: 191,
  },
  {
    id: 'engenho-de-mana',
    name: 'Engenho de Mana',
    circle: 5,
    school: 'abjuracao',
    execution: 'padrao',
    range: 'medio',
    duration: 'sustentada',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Arcanista', 'Bardo'],
    baseEffect:
      'Disco de energia (1,5m diâmetro) imune a dano, imóvel. Contramagia automática contra magias em alcance médio (exceto do conjurador) usando Misticismo. Vitória: anula + absorve PM como temporários. No turno do conjurador, se em alcance: gasta PM do disco para lançar magias.',
    augments: [
      {
        pmCost: 1,
        kind: 'muda',
        description:
          'Em vez de flutuar no ponto conjurado, disco flutua atrás do conjurador (sempre adjacente).',
      },
      {
        pmCost: 4,
        kind: 'muda',
        description: 'Muda a duração para 1 dia.',
      },
    ],
    bookPage: 192,
  },
  {
    id: 'enxame-de-pestes',
    name: 'Enxame de Pestes',
    circle: 2,
    school: 'convocacao',
    execution: 'completa',
    range: 'medio',
    duration: 'sustentada',
    saveType: 'fortitude',
    resistance: 'metade',
    components: VG,
    classes: ['Clérigo', 'Druida', 'Paladino'],
    baseEffect:
      'Enxame Médio (1,5m²) — besouros/gafanhotos/ratos/morcegos/serpentes. Passa por espaços; permite outras criaturas. Fim do turno: 2d12 dano a criaturas no espaço (Fortitude metade). Ação de movimento: move 12m.',
    augments: [
      {
        pmCost: 2,
        kind: 'aumenta',
        description: 'Aumenta o dano em +1d12.',
      },
      {
        pmCost: 3,
        kind: 'muda',
        description:
          'Muda resistência para Reflexos metade e enxame para criaturas maiores (gatos/guaxinins/compsognatos/kobolds). 3d12 dano à escolha entre corte/impacto/perfuração.',
      },
      {
        pmCost: 5,
        kind: 'aumenta',
        description: 'Aumenta o número de enxames em +1 (não podem ocupar o mesmo espaço).',
        requiresCircle: 3,
      },
      {
        pmCost: 7,
        kind: 'muda',
        description:
          'Muda resistência para Reflexos metade e enxame para criaturas elementais. 5d12 dano à escolha entre ácido/eletricidade/fogo/frio.',
        requiresCircle: 4,
      },
    ],
    bookPage: 192,
  },
  {
    id: 'enxame-rubro-de-ichabod',
    name: 'Enxame Rubro de Ichabod',
    circle: 3,
    school: 'convocacao',
    execution: 'padrao',
    range: 'medio',
    duration: 'sustentada',
    saveType: 'reflexos',
    resistance: 'metade',
    components: VG,
    classes: ['Arcanista', 'Bardo'],
    baseEffect:
      'Enxame Grande (3m²) de criaturas da Tormenta. Passa por espaços; permite outras. Fim do turno: 4d12 dano ácido em criaturas no espaço (Reflexos metade). Ação de movimento: move 12m.',
    augments: [
      {
        pmCost: 1,
        kind: 'muda',
        description:
          'Além do normal, falha em Reflexos = agarrada (enxame cobre corpo). Escapar: padrão + Acrobacia/Atletismo. Mover o enxame libera.',
      },
      {
        pmCost: 2,
        kind: 'aumenta',
        description: 'Aumenta o dano em +1d12.',
      },
      {
        pmCost: 2,
        kind: 'muda',
        description: 'Muda o dano para trevas.',
      },
      {
        pmCost: 3,
        kind: 'muda',
        description: 'O enxame vira Enorme (quadrado 6m).',
      },
      {
        pmCost: 3,
        kind: 'muda',
        description:
          'Enxame ganha voo 18m e ocupa cubo em vez de quadrado.',
      },
      {
        pmCost: 4,
        kind: 'muda',
        description:
          'Parasitas explodem em novos enxames. Início do turno: rola 1d6 — 5-6 surge novo adjacente. Move todos com uma ação (sem sobrepor).',
        requiresCircle: 4,
      },
    ],
    bookPage: 192,
  },
  {
    id: 'erupcao-glacial',
    name: 'Erupção Glacial',
    circle: 3,
    school: 'evocacao',
    execution: 'padrao',
    range: 'medio',
    duration: 'instantanea',
    saveType: 'reflexos',
    resistance: 'parcial',
    components: VG,
    classes: ['Arcanista', 'Bardo'],
    baseEffect:
      'Estacas de gelo (quadrado 6m): 4d6 corte + 4d6 frio + caído. Sucesso Reflexos = evita corte e queda. Estacas duram cena — área terreno difícil + cobertura leve. Destruídas por qualquer dano de fogo mágico.',
    augments: [
      {
        pmCost: 3,
        kind: 'aumenta',
        description: 'Aumenta o dano de frio em +2d6 e o dano de corte em +2d6.',
      },
      {
        pmCost: 4,
        kind: 'muda',
        description:
          'Muda área para cilindro 6m raio × 6m altura, duração para sustentada. Tempestade de granizo: 3d6 impacto + 3d6 frio em todas as criaturas (sem resistência) + camuflagem leve + piso escorregadio (terreno difícil, Acrobacia equilíbrio).',
        requiresCircle: 4,
      },
    ],
    bookPage: 192,
  },
  {
    id: 'esculpir-sons',
    name: 'Esculpir Sons',
    circle: 2,
    school: 'ilusao',
    execution: 'padrao',
    range: 'medio',
    duration: 'cena',
    saveType: 'vontade',
    resistance: 'anula',
    components: VG,
    classes: ['Arcanista', 'Bardo'],
    baseEffect:
      'Altera sons do alvo (1 criatura ou objeto). Omite (carroça silenciosa) ou transforma (fala vira canto de pássaro). Não cria sons; não fala idioma desconhecido. Escolha fixa. Voz drasticamente modificada impede lançar magia.',
    augments: [
      {
        pmCost: 2,
        kind: 'aumenta',
        description:
          'Aumenta o número de alvos em +1. Todas afetadas da mesma forma.',
      },
    ],
    bookPage: 192,
  },
  {
    id: 'escuridao',
    name: 'Escuridão',
    circle: 1,
    school: 'necromancia',
    execution: 'padrao',
    range: 'curto',
    duration: 'cena',
    saveType: 'vontade',
    resistance: 'anula',
    components: VG,
    classes: ['Arcanista', 'Bardo', 'Clérigo', 'Druida'],
    baseEffect:
      '1 objeto emana sombras (esfera 6m raio). Criaturas na área: camuflagem leve por escuridão leve. Luz natural não ilumina. Objeto guardado interrompe (revelado volta). Objeto de criatura involuntária: Vontade anula. Anula Luz.',
    augments: [
      {
        pmCost: 1,
        kind: 'aumenta',
        description: 'Aumenta a área da escuridão em +1,5m de raio.',
      },
      {
        pmCost: 2,
        kind: 'muda',
        description:
          'Muda o efeito para camuflagem total por escuridão total. Bloqueia visão na área e através dela.',
      },
      {
        pmCost: 2,
        kind: 'muda',
        description:
          'Muda alvo para 1 criatura e resistência para Fortitude parcial. Lançada nos olhos: cego pela cena; sucesso = cego 1 rodada.',
        requiresCircle: 2,
      },
      {
        pmCost: 3,
        kind: 'muda',
        description: 'Muda a duração para 1 dia.',
      },
      {
        pmCost: 5,
        kind: 'muda',
        description:
          'Muda alcance para pessoal e alvo para conjurador. Coberto por sombras: +10 Furtividade + camuflagem leve.',
        requiresCircle: 2,
      },
    ],
    bookPage: 193,
  },
  {
    id: 'explosao-caleidoscopica',
    name: 'Explosão Caleidoscópica',
    circle: 4,
    school: 'ilusao',
    execution: 'padrao',
    range: 'curto',
    duration: 'instantanea',
    saveType: 'fortitude',
    resistance: 'parcial',
    components: VG,
    classes: ['Arcanista'],
    baseEffect:
      'Luzes estroboscópicas + sons cacofônicos em esfera 6m. Efeito por nível/ND: ≤4 falha=inconsciente, sucesso=atordoado 1d4 + enjoado cena; 5-9 falha=atordoado 1d4 + enjoado cena, sucesso=atordoado 1 + enjoado 1d4; ≥10 falha=atordoado 1 + enjoado 1d4, sucesso=desprevenido + enjoado 1 rodada. Atordoamento 1x/cena por criatura.',
    augments: [],
    bookPage: 193,
  },
  {
    id: 'ferver-sangue',
    name: 'Ferver Sangue',
    circle: 3,
    school: 'necromancia',
    execution: 'padrao',
    range: 'curto',
    duration: 'sustentada',
    saveType: 'fortitude',
    resistance: 'parcial',
    components: VG,
    classes: ['Arcanista', 'Bardo'],
    baseEffect:
      'Sangue do alvo (1 criatura) ferve. No lançamento e no início de cada turno do conjurador: 4d8 fogo + enjoado 1 rodada (Fortitude metade + evita condição). 2 Fortitude seguidos passados = dissipa. Reduzido a 0 PV = corpo explode, 6d6 fogo em 3m (Reflexos metade). Não afeta sem sangue (construtos/mortos-vivos).',
    augments: [
      {
        pmCost: 2,
        kind: 'aumenta',
        description: 'Aumenta o dano em +1d8.',
      },
      {
        pmCost: 9,
        kind: 'muda',
        description: 'Muda alvo para criaturas escolhidas.',
        requiresCircle: 5,
      },
    ],
    bookPage: 193,
  },
  {
    id: 'fisico-divino',
    name: 'Físico Divino',
    circle: 2,
    school: 'transmutacao',
    execution: 'padrao',
    range: 'toque',
    duration: 'cena',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Clérigo', 'Druida', 'Paladino'],
    baseEffect:
      'Alvo (1 criatura) recebe +2 em Força, Destreza ou Constituição à escolha. Sem PV/PM adicionais.',
    augments: [
      {
        pmCost: 3,
        kind: 'muda',
        description: 'Muda alcance para curto e alvo para criaturas escolhidas.',
      },
      {
        pmCost: 3,
        kind: 'muda',
        description: '+2 nos três atributos físicos.',
        requiresCircle: 3,
      },
      {
        pmCost: 7,
        kind: 'muda',
        description: '+4 no atributo escolhido.',
        requiresCircle: 4,
      },
      {
        pmCost: 12,
        kind: 'muda',
        description: '+4 nos três atributos físicos.',
        requiresCircle: 5,
      },
    ],
    bookPage: 193,
  },
  {
    id: 'flecha-acida',
    name: 'Flecha Ácida',
    circle: 2,
    school: 'evocacao',
    execution: 'padrao',
    range: 'medio',
    duration: 'instantanea',
    saveType: 'reflexos',
    resistance: 'parcial',
    components: VG,
    classes: ['Arcanista', 'Bardo'],
    baseEffect:
      'Projétil: 4d6 ácido em 1 criatura ou objeto. Falha = muco corrosivo + 2d6 ácido no início dos 2 próximos turnos. Objeto sem posse: dano dobrado + ignora RD.',
    augments: [
      {
        pmCost: 1,
        kind: 'aumenta',
        description:
          'Além do normal, armadura/escudo do alvo corroídos: -1 Defesa permanente do item (conserto restaura, ver Ofício p121).',
      },
      {
        pmCost: 2,
        kind: 'aumenta',
        description: 'Aumenta a redução na Defesa em +1.',
      },
      {
        pmCost: 2,
        kind: 'aumenta',
        description: 'Aumenta o dano inicial e o dano por rodada em +1d6.',
      },
    ],
    bookPage: 193,
  },
  {
    id: 'forma-eterea',
    name: 'Forma Etérea',
    circle: 4,
    school: 'transmutacao',
    execution: 'completa',
    range: 'pessoal',
    duration: 'sustentada',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Arcanista'],
    baseEffect:
      'Transporta conjurador + equipamento para o plano etéreo. Vira fantasma (mas viva): invisível (alterna ação livre), incorpórea, move em qualquer direção. Enxerga material em cinza; visão/audição reduzidas a 18m. Só abjuração/essência afetam etéreas. Etérea não ataca material. Etéreas se afetam. Materializar: ação de movimento encerra magia. Materializar em espaço ocupado = joga para o não-ocupado mais próximo + 1d6 impacto por 1,5m.',
    augments: [
      {
        pmCost: 5,
        kind: 'muda',
        description:
          'Muda alcance para toque e alvo para até 5 criaturas voluntárias de mãos dadas (podem soltar depois).',
        requiresCircle: 5,
      },
    ],
    bookPage: 193,
  },
  {
    id: 'furia-do-panteao',
    name: 'Fúria do Panteão',
    circle: 5,
    school: 'evocacao',
    execution: 'completa',
    range: 'longo',
    duration: 'sustentada',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Clérigo', 'Druida'],
    baseEffect:
      'Tempestade em cubo 90m. Ventos impedem ataques à distância + condição terrível para lançar magias na área. Inimigos: visibilidade reduzida (como Névoa). Ação de movimento por turno: Nevasca (10d6 frio, Fortitude metade, terreno difícil até siroco/fim cena); Raios (10d8 eletricidade, Reflexos metade); Siroco (10d6 metade corte + metade fogo + sangrando, Fortitude metade + evita); Trovões (10d6 impacto + desprevenido 1 rodada, Fortitude metade + evita).',
    augments: [],
    bookPage: 194,
  },
  {
    id: 'globo-da-verdade-de-gwen',
    name: 'Globo da Verdade de Gwen',
    circle: 2,
    school: 'adivinhacao',
    execution: 'padrao',
    range: 'curto',
    duration: 'cena',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Clérigo', 'Druida', 'Paladino'],
    baseEffect:
      'Globo flutuante 50cm diâmetro mostra cena vista até 1 semana atrás pelo conjurador ou criatura tocada (Vontade anula pergunta indesejada).',
    augments: [
      {
        pmCost: 1,
        kind: 'muda',
        description: 'Globo mostra cena vista até 1 mês atrás.',
      },
      {
        pmCost: 2,
        kind: 'muda',
        description: 'Como acima, até 1 ano atrás.',
      },
      {
        pmCost: 2,
        kind: 'muda',
        description:
          'Ao lançar, toca cadáver — globo mostra a última cena vista pela criatura.',
      },
      {
        pmCost: 4,
        kind: 'muda',
        description:
          'Muda alcance para longo e efeito para 10 globos. Todos mostram a mesma cena.',
      },
    ],
    bookPage: 194,
  },
  {
    id: 'globo-de-invulnerabilidade',
    name: 'Globo de Invulnerabilidade',
    circle: 3,
    school: 'abjuracao',
    execution: 'padrao',
    range: 'pessoal',
    duration: 'sustentada',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Arcanista', 'Bardo'],
    baseEffect:
      'Esfera 3m raio detém magias até 2º círculo. Nenhuma magia contra alvo dentro; área ineficaz. De dentro para fora, magias funcionam. Dissipação só afeta se usada diretamente contra conjurador (área não). Magias entrando são suprimidas (voltam ao sair se duração continua). Imóvel; não afeta criaturas/objetos. Entrada/saída livre após lançar.',
    augments: [
      {
        pmCost: 4,
        kind: 'muda',
        description: 'Muda o efeito para afetar magias até 3º círculo.',
        requiresCircle: 4,
      },
      {
        pmCost: 9,
        kind: 'muda',
        description: 'Muda o efeito para afetar magias até 4º círculo.',
        requiresCircle: 5,
      },
    ],
    bookPage: 194,
  },
  {
    id: 'guardiao-divino',
    name: 'Guardião Divino',
    circle: 4,
    school: 'convocacao',
    execution: 'padrao',
    range: 'curto',
    duration: 'definida',
    durationNote: 'cena ou até ser descarregado',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Clérigo', 'Druida'],
    baseEffect:
      'Elemental Pequeno em forma de orbe de luz divina. Incorpóreo, imune a dano, ilumina como tocha. Tem 100 pontos de luz. Uma vez por rodada no turno do conjurador: move (voo 18m) + gasta pontos para curar em alcance curto — 1 PV por 1 ponto ou 1 condição por 3 pontos (abalado/apavorado/alquebrado/atordoado/cego/confuso/debilitado/enjoado/esmorecido/exausto/fascinado/fatigado/fraco/frustrado/ofuscado/pasmo/sangrando/surdo/vulnerável). Encerra sem pontos.',
    augments: [],
    bookPage: 194,
  },
  {
    id: 'ilusao-lacerante',
    name: 'Ilusão Lacerante',
    circle: 3,
    school: 'ilusao',
    execution: 'padrao',
    range: 'medio',
    duration: 'sustentada',
    saveType: 'vontade',
    resistance: 'anula',
    components: VG,
    classes: ['Arcanista', 'Bardo'],
    baseEffect:
      'Ilusão de perigo mortal em cubo 9m. No lançamento: criaturas na área testam Vontade — falha = crê real + 3d6 psíquico não letal. Início do turno na área repete Vontade; falha = repete dano. Só quem falha vê a ilusão e racionaliza (mesmo teto pode "cair" várias vezes).',
    augments: [
      {
        pmCost: 3,
        kind: 'aumenta',
        description: 'Aumenta o dano em +2d6.',
      },
      {
        pmCost: 4,
        kind: 'muda',
        description: 'Muda a área para um cubo de 90m.',
        requiresCircle: 4,
      },
    ],
    bookPage: 195,
  },
  {
    id: 'imobilizar',
    name: 'Imobilizar',
    circle: 3,
    school: 'encantamento',
    execution: 'padrao',
    range: 'curto',
    duration: 'cena',
    saveType: 'vontade',
    resistance: 'parcial',
    components: VG,
    classes: ['Arcanista', 'Bardo', 'Clérigo', 'Druida'],
    baseEffect:
      '1 humanoide ou animal fica paralisado; sucesso = lento. Cada rodada: ação completa faz novo Vontade — sucesso liberta.',
    augments: [
      {
        pmCost: 1,
        kind: 'muda',
        description: 'Muda o alvo para 1 espírito.',
      },
      {
        pmCost: 2,
        kind: 'aumenta',
        description: 'Aumenta o número de alvos em +1.',
      },
      {
        pmCost: 3,
        kind: 'muda',
        description: 'Muda o alvo para 1 criatura.',
        requiresCircle: 4,
      },
    ],
    bookPage: 195,
  },
  {
    id: 'infligir-ferimentos',
    name: 'Infligir Ferimentos',
    circle: 1,
    school: 'necromancia',
    execution: 'padrao',
    range: 'toque',
    duration: 'instantanea',
    saveType: 'fortitude',
    resistance: 'metade',
    components: VG,
    classes: ['Clérigo', 'Druida', 'Paladino'],
    baseEffect:
      'Energia negativa em 1 criatura: 2d8+2 trevas (ou cura 2d8+2 PV em morto-vivo). Anula Curar Ferimentos.',
    augments: [
      {
        pmCost: 1,
        kind: 'aumenta',
        description: 'Além do normal, alvo fica fraco pela cena (resistência evita).',
      },
      {
        pmCost: 2,
        kind: 'aumenta',
        description: 'Aumenta o dano em +1d8+1.',
      },
      {
        pmCost: 2,
        kind: 'muda',
        description:
          'Muda resistência para nenhuma. Como parte da execução: ataque corpo a corpo contra o alvo. Acerto = dano do ataque + efeito da magia.',
      },
      {
        pmCost: 5,
        kind: 'muda',
        description: 'Muda o alcance para curto e o alvo para criaturas escolhidas.',
      },
    ],
    bookPage: 195,
  },
  {
    id: 'intervencao-divina',
    name: 'Intervenção Divina',
    circle: 5,
    school: 'convocacao',
    execution: 'completa',
    range: 'ilimitado',
    duration: 'definida',
    durationNote: 'veja texto',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Clérigo', 'Druida'],
    baseEffect:
      'Divindade intercede. Base: cura PV + condições de até 10 criaturas em alcance longo (dano em mortos-vivos); dissipa magias até 4º círc. Com sacrifício 2 PM: item mundano T$ 30.000; duplica magia até 4º (materiais mantidos); proteger cidade de desastre; ressuscitar criatura em alcance longo morta há 1 rodada (1 PV); outras ações à discrição do mestre e da divindade.',
    augments: [],
    bookPage: 195,
  },
  {
    id: 'lagrimas-de-wynna',
    name: 'Lágrimas de Wynna',
    circle: 5,
    school: 'abjuracao',
    execution: 'padrao',
    range: 'curto',
    duration: 'instantanea',
    saveType: 'vontade',
    resistance: 'parcial',
    components: VG,
    classes: ['Clérigo', 'Druida'],
    baseEffect:
      '1 criatura falha na resistência = perde habilidade de lançar magias arcanas até fim da cena. Sucesso = perde por 1 rodada.',
    augments: [
      {
        pmCost: 2,
        kind: 'muda',
        description:
          'Muda área para esfera 6m raio e alvo para criaturas escolhidas.',
      },
      {
        pmCost: 5,
        kind: 'muda',
        description:
          'Muda execução para 1 dia + sacrifício 1 PM. Alvo em alcance curto durante toda execução. Ao término: Vontade — falha = perde arcanas permanentemente (irreversível). Sucesso = resiste mas pode ser alvo no dia seguinte.',
      },
    ],
    bookPage: 196,
  },
  {
    id: 'lanca-ignea-de-aleph',
    name: 'Lança Ígnea de Aleph',
    circle: 3,
    school: 'evocacao',
    execution: 'padrao',
    range: 'medio',
    duration: 'instantanea',
    saveType: 'reflexos',
    resistance: 'parcial',
    components: VG,
    classes: ['Arcanista', 'Bardo'],
    baseEffect:
      'Projétil: 4d6 fogo + 4d6 perfuração em 1 criatura + em chamas (2d6/rodada). Sucesso = metade + sem chamas. Respingos de rocha atingem adjacentes ao alvo (Reflexos evita ficarem em chamas).',
    augments: [
      {
        pmCost: 3,
        kind: 'aumenta',
        description:
          'Aumenta o dano inicial em +2d6 e o dano em chamas em +1d6.',
      },
      {
        pmCost: 4,
        kind: 'muda',
        description:
          'Muda duração para cena ou até descarregada. Cria 4 dardos de lava flutuantes. Uma vez por rodada, ação livre dispara um dardo (efeito normal da magia).',
        requiresCircle: 4,
      },
    ],
    bookPage: 196,
  },
  {
    id: 'legiao',
    name: 'Legião',
    circle: 5,
    school: 'encantamento',
    execution: 'padrao',
    range: 'medio',
    duration: 'sustentada',
    saveType: 'vontade',
    resistance: 'parcial',
    components: VG,
    classes: ['Arcanista'],
    baseEffect:
      'Domina mente de até 10 criaturas na área. Obedecem cegamente exceto ordens claramente suicidas. Fim do turno = novo teste; sucesso = abalado 1 rodada.',
    augments: [
      {
        pmCost: 1,
        kind: 'aumenta',
        description: 'Aumenta o número de alvos em +1.',
      },
    ],
    bookPage: 196,
  },
  {
    id: 'lendas-e-historias',
    name: 'Lendas e Histórias',
    circle: 3,
    school: 'adivinhacao',
    execution: 'padrao',
    range: 'toque',
    duration: 'sustentada',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Arcanista', 'Bardo', 'Clérigo', 'Druida'],
    baseEffect:
      'Descobre informações sobre criatura/objeto/local tocado — mestre modula detalhe/pistas. Por rodada mantida revela: tudo que testes de Conhecimento revelariam (raça/classe/nível/atributos/magias/resistências/fraquezas para criatura); efeito/funcionamento de item mágico; magias ativas.',
    augments: [
      {
        pmCost: 4,
        kind: 'muda',
        description:
          'Muda execução para 1 dia, alcance para ilimitado, adiciona componente material (cuba de ouro + água + ingredientes T$ 1.000). Precisa alguma informação sobre o alvo (nome/descrição/localização).',
      },
    ],
    bookPage: 196,
  },
  {
    id: 'leque-cromatico',
    name: 'Leque Cromático',
    circle: 1,
    school: 'ilusao',
    execution: 'padrao',
    range: 'pessoal',
    duration: 'instantanea',
    saveType: 'vontade',
    resistance: 'parcial',
    components: VG,
    classes: ['Arcanista', 'Bardo'],
    baseEffect:
      'Cone 4,5m de luzes. Animais e humanoides na área: atordoados 1 rodada (1x/cena, Vontade anula) + ofuscados pela cena. Não afeta cegas.',
    augments: [
      {
        pmCost: 2,
        kind: 'aumenta',
        description: 'Além do normal, criaturas afetadas ficam vulneráveis pela cena.',
      },
      {
        pmCost: 2,
        kind: 'aumenta',
        description: 'Também afeta espíritos e monstros na área.',
        requiresCircle: 2,
      },
      {
        pmCost: 5,
        kind: 'aumenta',
        description:
          'Também afeta constructos, espíritos, monstros e mortos-vivos na área.',
        requiresCircle: 3,
      },
    ],
    bookPage: 196,
  },
  {
    id: 'libertacao',
    name: 'Libertação',
    circle: 4,
    school: 'abjuracao',
    execution: 'padrao',
    range: 'curto',
    duration: 'cena',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Arcanista', 'Bardo', 'Clérigo', 'Druida'],
    baseEffect:
      'Alvo (1 criatura) imune a efeitos de movimento; ignora restrições de deslocamento. Usa habilidades de liberdade de movimentos mesmo com armadura/escudo.',
    augments: [
      {
        pmCost: 2,
        kind: 'aumenta',
        description:
          'Além do normal, anda sobre água/líquidos com deslocamento normal (não protege contra efeitos do líquido: lava causa dano).',
      },
      {
        pmCost: 2,
        kind: 'aumenta',
        description: 'Além do normal, alvo pode escolher 20 em Atletismo.',
      },
      {
        pmCost: 2,
        kind: 'aumenta',
        description:
          'Além do normal, alvo pode escolher 20 em Acrobacia + faz todas as manobras sem treinamento.',
      },
      {
        pmCost: 5,
        kind: 'muda',
        description: 'Muda o alvo para até 5 criaturas.',
      },
      {
        pmCost: 5,
        kind: 'muda',
        description: 'Pode dissipar Aprisionamento.',
      },
    ],
    bookPage: 196,
  },
  {
    id: 'ligacao-sombria',
    name: 'Ligação Sombria',
    circle: 4,
    school: 'necromancia',
    execution: 'padrao',
    range: 'longo',
    duration: 'dia',
    saveType: 'fortitude',
    resistance: 'anula',
    components: VG,
    classes: ['Clérigo', 'Druida', 'Paladino'],
    baseEffect:
      'Conexão corpo-a-corpo com 1 criatura (marca idêntica na pele de ambos). Quando o conjurador sofrer dano ou condição: alvo faz Fortitude — falha = mesmo dano/condição. Termina se alvo chegar a 0 PV.',
    augments: [
      {
        pmCost: 5,
        kind: 'aumenta',
        description:
          'A magia não termina se o alvo chegar a 0 PV (dano por esta magia pode matá-lo).',
      },
    ],
    bookPage: 196,
  },
  {
    id: 'ligacao-telepatica',
    name: 'Ligação Telepática',
    circle: 2,
    school: 'adivinhacao',
    execution: 'padrao',
    range: 'toque',
    duration: 'dia',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Arcanista', 'Bardo'],
    baseEffect:
      'Elo mental entre 2 criaturas voluntárias (INT -3+; conjurador pode ser uma). Comunicam sem idioma comum, sem limite de distância; falha em mundos diferentes.',
    augments: [
      {
        pmCost: 2,
        kind: 'aumenta',
        description: 'Aumenta o número de alvos em +1.',
      },
      {
        pmCost: 3,
        kind: 'muda',
        description:
          'Muda alvo para 1 criatura. Cria elo mental — ação de movimento vê/ouve pelos sentidos. Involuntária faz Vontade para suprimir por 1 hora.',
        requiresCircle: 3,
      },
    ],
    bookPage: 197,
  },
  {
    id: 'localizacao',
    name: 'Localização',
    circle: 2,
    school: 'adivinhacao',
    execution: 'padrao',
    range: 'pessoal',
    duration: 'cena',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Arcanista', 'Bardo'],
    baseEffect:
      'Encontra 1 criatura ou objeto (geral ou específico). Indica direção e distância do mais próximo em esfera 90m. Específico exige imagem precisa; imprecisa = falha (PM gastos). Bloqueada por fina camada de chumbo.',
    augments: [
      {
        pmCost: 0,
        kind: 'muda',
        description:
          'Truque: muda alvo para conjurador. Sabe onde fica o norte + +5 Sobrevivência para se orientar.',
      },
      {
        pmCost: 5,
        kind: 'aumenta',
        description:
          'Aumenta a área em fator de 10 (90m→900m, 900m→9km etc).',
      },
    ],
    bookPage: 197,
  },
  {
    id: 'manto-do-cruzado',
    name: 'Manto do Cruzado',
    circle: 4,
    school: 'evocacao',
    execution: 'padrao',
    range: 'pessoal',
    duration: 'sustentada',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Clérigo', 'Druida', 'Paladino'],
    baseEffect:
      'Manto de energia divina. Escolha versão ao aprender (fixa). Manto de Luz: início do turno, conjurador + aliados em alcance curto recuperam 2d8 PV; imune a trevas; ataques causam +2d8 dano de luz. Manto de Trevas: início do turno, inimigos em alcance curto sofrem 4d8 trevas; conjurador cura metade do dano causado.',
    augments: [],
    bookPage: 197,
  },
  {
    id: 'mao-poderosa-de-talude',
    name: 'Mão Poderosa de Talude',
    circle: 4,
    school: 'convocacao',
    execution: 'padrao',
    range: 'medio',
    duration: 'sustentada',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Arcanista', 'Bardo'],
    baseEffect:
      'Mão Grande flutuante entre conjurador e oponente escolhido. Cobertura leve (+5 Defesa) contra o oponente; imune a escuridão/invisibilidade/metamorfose/disfarce. Defesa 20, PV e resistências do conjurador. Ação de movimento: muda oponente ou executa. Agarrar: manobra usando Misticismo +10 (mantém sem dano). Esmagar: 2d6+10 impacto em agarrado. Empurrar: manobra usando Misticismo +10 (mão acompanha).',
    augments: [
      {
        pmCost: 2,
        kind: 'aumenta',
        description: 'Aumenta o dano em +1d6+5.',
      },
      {
        pmCost: 5,
        kind: 'muda',
        description: 'Muda o bônus em Misticismo para +20.',
        requiresCircle: 5,
      },
    ],
    bookPage: 197,
  },
  {
    id: 'mapear',
    name: 'Mapear',
    circle: 2,
    school: 'adivinhacao',
    execution: 'padrao',
    range: 'toque',
    duration: 'cena',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Arcanista', 'Bardo'],
    baseEffect:
      'Fagulha esboça mapa em superfície/objeto plano. Lugar conhecido = mapa completo; desconhecido = esboço geral + ponto de referência + lugar de interesse (mestre). Máximo 10km². Dentro de construção = mostra o andar atual.',
    augments: [
      {
        pmCost: 3,
        kind: 'muda',
        description:
          'Muda alvo para 1 criatura e duração para 1 hora. Criatura tocada descobre caminho mais direto para entrar/sair de um lugar (não localiza criaturas ou objetos). Passar >1h da magia perde conhecimento.',
      },
    ],
    bookPage: 198,
  },
  {
    id: 'marca-da-obediencia',
    name: 'Marca da Obediência',
    circle: 2,
    school: 'encantamento',
    execution: 'padrao',
    range: 'toque',
    duration: 'cena',
    saveType: 'vontade',
    resistance: 'anula',
    components: VG,
    classes: ['Arcanista', 'Bardo', 'Clérigo', 'Druida', 'Paladino'],
    baseEffect:
      'Grava marca mística em 1 criatura + ordem verbal ("não me ataque", "siga-me", "não saia"). Alvo gasta todas as ações do turno para obedecer. Sem ordens genéricas ("ajude-me") ou suicidas. Cada rodada: Vontade — sucesso dissipa.',
    augments: [
      {
        pmCost: 3,
        kind: 'muda',
        description:
          'Muda a duração para 1 dia. Fora de combate, alvo testa Vontade só a cada hora.',
        requiresCircle: 3,
      },
      {
        pmCost: 3,
        kind: 'aumenta',
        description:
          'Falha em Vontade = marca causa 3d6 dano psíquico.',
        requiresCircle: 3,
      },
    ],
    bookPage: 198,
  },
  {
    id: 'mata-dragao',
    name: 'Mata-Dragão',
    circle: 5,
    school: 'evocacao',
    execution: 'completa',
    range: 'pessoal',
    duration: 'instantanea',
    saveType: 'reflexos',
    resistance: 'metade',
    components: VG,
    classes: ['Arcanista', 'Bardo'],
    baseEffect:
      'Execução ritual de 2 rodadas (2 cânticos). Carga de energia em cone 30m: 20d12 essência em criaturas/construções/objetos livres. Cada resultado 12 rolado = +1d12. Lenta, difícil em combate.',
    augments: [
      {
        pmCost: 1,
        kind: 'aumenta',
        description: 'Aumenta o dano em +1d12.',
      },
    ],
    bookPage: 198,
  },
  {
    id: 'mente-divina',
    name: 'Mente Divina',
    circle: 2,
    school: 'adivinhacao',
    execution: 'padrao',
    range: 'toque',
    duration: 'cena',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Clérigo', 'Druida', 'Paladino'],
    baseEffect:
      '1 criatura recebe +2 em Inteligência, Sabedoria ou Carisma. Sem PV/PM/perícias adicionais.',
    augments: [
      {
        pmCost: 3,
        kind: 'muda',
        description: 'Muda alcance para curto e alvo para criaturas escolhidas.',
      },
      {
        pmCost: 3,
        kind: 'muda',
        description: '+2 nos três atributos mentais.',
        requiresCircle: 3,
      },
      {
        pmCost: 7,
        kind: 'muda',
        description: '+4 no atributo escolhido.',
        requiresCircle: 4,
      },
      {
        pmCost: 12,
        kind: 'muda',
        description: '+4 nos três atributos mentais.',
        requiresCircle: 5,
      },
    ],
    bookPage: 198,
  },
  {
    id: 'miasma-mefitico',
    name: 'Miasma Mefítico',
    circle: 2,
    school: 'necromancia',
    execution: 'padrao',
    range: 'medio',
    duration: 'instantanea',
    saveType: 'fortitude',
    resistance: 'metade',
    components: VG,
    classes: ['Clérigo', 'Druida', 'Paladino'],
    baseEffect:
      'Nuvem 6m raio: 5d6 ácido + enjoados 1 rodada. Sucesso = metade + sem condição.',
    augments: [
      {
        pmCost: 0,
        kind: 'muda',
        description:
          'Truque: muda alcance para toque, área para alvo (1 criatura com 0 PV ou menos), duração instantânea, resistência para Fortitude anula, adiciona componente material (pó de ônix T$ 10). Canaliza Miasma contra vítima. Falha = morre, +2 CD das magias por 1 dia. Sucesso = imune ao truque por 1 dia.',
      },
      {
        pmCost: 2,
        kind: 'aumenta',
        description: 'Aumenta o dano em +1d6.',
      },
      {
        pmCost: 3,
        kind: 'muda',
        description: 'Muda o tipo do dano para trevas.',
      },
    ],
    bookPage: 199,
  },
  {
    id: 'miragem',
    name: 'Miragem',
    circle: 3,
    school: 'ilusao',
    execution: 'padrao',
    range: 'longo',
    duration: 'dia',
    saveType: 'vontade',
    resistance: 'desacredita',
    components: VG,
    classes: ['Arcanista', 'Bardo'],
    baseEffect:
      'Cubo 90m parece outro terreno (sons e cheiros). Planície vira pântano, floresta vira montanha. Cria armadilhas (areia movediça vira terra firme, precipício vira lago). Altera/inclui/esconde estruturas; não cria criaturas (podem se esconder nas estruturas ilusórias).',
    augments: [
      {
        pmCost: 4,
        kind: 'aumenta',
        description:
          'Além do normal, altera aparência de criaturas escolhidas na área (como Disfarce Ilusório).',
      },
      {
        pmCost: 9,
        kind: 'muda',
        description:
          'Muda duração para permanente + componente material (pó de diamante T$ 1.000).',
        requiresCircle: 4,
      },
    ],
    bookPage: 199,
  },
  {
    id: 'missao-divina',
    name: 'Missão Divina',
    circle: 3,
    school: 'encantamento',
    execution: 'padrao',
    range: 'curto',
    duration: 'definida',
    durationNote: '1 semana ou até ser descarregada',
    saveType: 'vontade',
    resistance: 'anula',
    components: VG,
    classes: ['Clérigo', 'Druida', 'Paladino'],
    baseEffect:
      'Alvo (1 criatura) deve cumprir tarefa escolhida. Dura 1 semana ou até cumprir. Pode recusar — fim do dia sem esforço = Vontade; falha = -2 cumulativo em testes/rolagens. Não força ato suicida ou missão impossível.',
    augments: [
      {
        pmCost: 2,
        kind: 'muda',
        description:
          'Muda alcance para toque, duração para permanente, penalidade -1 PM. Inscreve marca (tatuagem) + ação-gatilho (crime, ato contra divindade). Ativação = -2 cumulativo em testes. Dissipação suprime marca por 1 dia; só conjurador original ou Purificação remove.',
      },
      {
        pmCost: 4,
        kind: 'muda',
        description: 'Muda a duração para 1 ano ou até ser descarregada.',
      },
    ],
    bookPage: 199,
  },
  {
    id: 'montaria-arcana',
    name: 'Montaria Arcana',
    circle: 2,
    school: 'convocacao',
    execution: 'padrao',
    range: 'curto',
    duration: 'dia',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Arcanista', 'Bardo'],
    baseEffect:
      'Convoca parceiro cavalo (ou pônei) de guerra veterano. Animal negro, crina cinzenta, cascos de fumaça (aparência editável). Atravessa terreno difícil sem redução. Misticismo substitui Cavalgar (inclui treinado).',
    augments: [
      {
        pmCost: 1,
        kind: 'aumenta',
        description:
          'Além do normal, animais em alcance curto testam Vontade — sucesso = abalados pela cena; falha = apavorados 1d4 rodadas depois abalados pela cena.',
      },
      {
        pmCost: 3,
        kind: 'muda',
        description: 'Muda a duração para permanente e adiciona penalidade -3 PM.',
      },
      {
        pmCost: 3,
        kind: 'aumenta',
        description:
          'Aumenta o tamanho em uma categoria. Enorme carrega 2, Colossal carrega 6. Uma controla, outras são deslocadas.',
      },
      {
        pmCost: 3,
        kind: 'muda',
        description: 'Muda o nível do parceiro para mestre.',
        requiresCircle: 3,
      },
    ],
    bookPage: 199,
  },
  {
    id: 'muralha-de-ossos',
    name: 'Muralha de Ossos',
    circle: 4,
    school: 'necromancia',
    execution: 'padrao',
    range: 'medio',
    duration: 'cena',
    saveType: 'reflexos',
    resistance: 'anula',
    components: VG,
    classes: ['Arcanista', 'Bardo', 'Clérigo', 'Druida', 'Paladino'],
    baseEffect:
      'Parede de ossos (15m×9m×1,5m, qualquer forma tocando o solo). Criaturas na área/adjacentes: 4d8 corte + Reflexos ou agarradas. Agarrada: ação padrão + Atletismo — sucesso sai adjacente, falha = 4d8 corte. Trecho 3m: Defesa 8, 40 PV, RD 10 corte/frio/perfuração. Escalar: Atletismo + 4d8 corte por 3m.',
    augments: [
      {
        pmCost: 3,
        kind: 'aumenta',
        description: 'Aumenta o comprimento em +15m e a altura em +3m.',
      },
      {
        pmCost: 5,
        kind: 'muda',
        description:
          'Muro de esqueletos animados. Início do turno adjacente/escalando: Reflexos — falha = agarrada (efeitos normais).',
      },
    ],
    bookPage: 199,
  },
  {
    id: 'muralha-elemental',
    name: 'Muralha Elemental',
    circle: 3,
    school: 'evocacao',
    execution: 'padrao',
    range: 'medio',
    duration: 'cena',
    saveType: 'reflexos',
    resistance: 'metade',
    components: VG,
    classes: ['Arcanista', 'Bardo'],
    baseEffect:
      'Muro 30m×3m ou cúpula 3m raio. Fogo: lado quente à escolha — 2d6 fogo em 6m ao lançar e no início do turno; atravessar = 8d6 fogo; presas testam Reflexos, escape ao lado quente sofre +2d6. Gelo: parede 15cm; cúpula prende criaturas (Reflexos escapa antes). Trecho 3m: Defesa 8, 40 PV, RD 5. Fogo causa dano dobrado. Atravessar trecho rompido = 4d8 frio.',
    augments: [
      {
        pmCost: 2,
        kind: 'aumenta',
        description: 'Aumenta o dano por atravessar em +2d6.',
      },
      {
        pmCost: 2,
        kind: 'aumenta',
        description:
          'Aumenta o comprimento em +15m e altura em +3m (até 60m×9m).',
      },
      {
        pmCost: 4,
        kind: 'muda',
        description:
          'Muda duração para sustentada + adiciona escolha Força. Muralha invisível e indestrutível — imune a dano/magia. Não atravessada por incorpóreos. Teletransporte (Salto Dimensional) atravessa. Magia de área não passa; magia lançada diretamente em alvo do outro lado (Sono) tem linha de efeito.',
        requiresCircle: 4,
      },
    ],
    bookPage: 200,
  },
  {
    id: 'nevoa',
    name: 'Névoa',
    circle: 1,
    school: 'convocacao',
    execution: 'padrao',
    range: 'curto',
    duration: 'cena',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Arcanista', 'Bardo', 'Clérigo', 'Druida', 'Paladino'],
    baseEffect:
      'Névoa espessa (nuvem 6m raio × 6m altura). Camuflagem leve até 1,5m, camuflagem total a partir 3m. Vento forte dispersa em 4 rodadas, vendaval em 1. Não funciona sob a água.',
    augments: [
      {
        pmCost: 1,
        kind: 'aumenta',
        description: 'Funciona sob a água, criando cortina de tinta.',
      },
      {
        pmCost: 2,
        kind: 'aumenta',
        description:
          'Escolhe criaturas ao lançar; elas enxergam através do efeito.',
        requiresCircle: 2,
      },
      {
        pmCost: 2,
        kind: 'aumenta',
        description:
          'Nuvem tem cheiro horrível. Início do turno dentro ou criaturas com faro em alcance curto: Fortitude — falha = enjoada 1 rodada.',
      },
      {
        pmCost: 2,
        kind: 'aumenta',
        description:
          'Nuvem esverdeada e cáustica. Início do turno dentro: 2d4 ácido.',
      },
      {
        pmCost: 3,
        kind: 'aumenta',
        description: 'Aumenta o dano de ácido em +2d4.',
      },
      {
        pmCost: 5,
        kind: 'aumenta',
        description:
          'Nuvem espessa/quase sólida. Deslocamento reduzido a 3m + -2 em ataque e dano.',
      },
    ],
    bookPage: 200,
  },
  {
    id: 'oracao',
    name: 'Oração',
    circle: 2,
    school: 'encantamento',
    execution: 'padrao',
    range: 'curto',
    duration: 'sustentada',
    saveType: 'none',
    resistance: null,
    components: ['verbal', 'gestual', 'material'],
    classes: ['Clérigo', 'Druida', 'Paladino'],
    baseEffect:
      'Aliados no alcance +2 em perícia/dano; inimigos no alcance -2 em perícia/dano. Cumulativo com outras magias. Componente material: T$ 20 por PM em incensos/oferendas.',
    augments: [
      {
        pmCost: 1,
        kind: 'aumenta',
        description:
          'Aumenta os bônus em +1 (máximo limitado pelo círculo máximo).',
      },
      {
        pmCost: 2,
        kind: 'aumenta',
        description:
          'Aumenta as penalidades em -1 (máximo limitado pelo círculo máximo).',
      },
      {
        pmCost: 7,
        kind: 'muda',
        description: 'Muda o alcance para médio.',
        requiresCircle: 3,
      },
      {
        pmCost: 12,
        kind: 'muda',
        description: 'Muda a duração para cena.',
        requiresCircle: 4,
      },
    ],
    bookPage: 200,
  },
  {
    id: 'orientacao',
    name: 'Orientação',
    circle: 1,
    school: 'adivinhacao',
    execution: 'padrao',
    range: 'curto',
    duration: 'definida',
    durationNote: '1 rodada',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Clérigo', 'Druida', 'Paladino'],
    baseEffect:
      '1 criatura rola 2 dados e usa o melhor no próximo teste de perícia.',
    augments: [
      {
        pmCost: 2,
        kind: 'muda',
        description:
          'Muda duração para cena. Testes de perícia baseados no atributo escolhido rolam 2 dados e usam o melhor. Não afeta ataque/resistência.',
        requiresCircle: 2,
      },
      {
        pmCost: 5,
        kind: 'muda',
        description:
          'Como acima, mas escolhe entre atributos físicos (For/Des/Con) ou mentais (Int/Sab/Car).',
        requiresCircle: 3,
      },
      {
        pmCost: 5,
        kind: 'muda',
        description: 'Muda o alvo para criaturas escolhidas.',
        requiresCircle: 3,
      },
    ],
    bookPage: 200,
  },
  {
    id: 'palavra-primordial',
    name: 'Palavra Primordial',
    circle: 5,
    school: 'encantamento',
    execution: 'padrao',
    range: 'curto',
    duration: 'instantanea',
    saveType: 'vontade',
    resistance: 'parcial',
    components: VG,
    classes: ['Arcanista', 'Bardo', 'Clérigo', 'Druida', 'Paladino'],
    baseEffect:
      'Palavra do idioma primordial da Criação em 1 criatura com menos níveis que o conjurador. Escolha: Atordoar (1d4+1 rodadas, 1x/cena; sucesso ou já atordoada = desprevenida 1d4). Cegar (cega; sucesso = ofuscada 1d4). Matar (morre; se >metade PV também testa Fortitude; sucesso em qualquer = perde 10d8 PV + sangrando).',
    augments: [
      {
        pmCost: 2,
        kind: 'muda',
        description:
          'Muda alcance para toque e alvo para 1 criatura. Falha se conjurador e alvo não forem devotos da mesma divindade.',
      },
    ],
    bookPage: 200,
  },
  {
    id: 'pele-de-pedra',
    name: 'Pele de Pedra',
    circle: 3,
    school: 'transmutacao',
    execution: 'padrao',
    range: 'pessoal',
    duration: 'cena',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Arcanista', 'Bardo', 'Clérigo', 'Druida', 'Paladino'],
    baseEffect:
      'Pele de rocha: RD 5.',
    augments: [
      {
        pmCost: 1,
        kind: 'muda',
        description: 'Muda o alcance para toque e o alvo para 1 criatura.',
      },
      {
        pmCost: 4,
        kind: 'muda',
        description: 'Muda a duração para 1 dia.',
      },
      {
        pmCost: 4,
        kind: 'aumenta',
        description: 'Pele de aço: RD 10.',
        requiresCircle: 4,
      },
      {
        pmCost: 4,
        kind: 'muda',
        description:
          'Muda alcance para toque, alvo para 1 criatura, duração para 1d4 rodadas + Resistência Fortitude anula. Transforma alvo + equipamento em estátua inerte sem consciência: mesmos PV + RD 8. Quebrada = morre.',
        requiresCircle: 4,
      },
      {
        pmCost: 9,
        kind: 'muda',
        description: 'Como acima, mas duração permanente.',
        requiresCircle: 5,
      },
    ],
    bookPage: 201,
  },
  {
    id: 'perdicao',
    name: 'Perdição',
    circle: 1,
    school: 'necromancia',
    execution: 'padrao',
    range: 'curto',
    duration: 'cena',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Clérigo', 'Paladino'],
    baseEffect:
      'Amaldiçoa criaturas escolhidas: -1 em ataque e dano. Anula Bênção.',
    augments: [
      {
        pmCost: 2,
        kind: 'aumenta',
        description:
          'Aumenta as penalidades em -1 (máximo limitado pelo círculo máximo).',
      },
    ],
    bookPage: 201,
  },
  {
    id: 'poeira-da-podridao',
    name: 'Poeira da Podridão',
    circle: 3,
    school: 'necromancia',
    execution: 'padrao',
    range: 'medio',
    duration: 'cena',
    saveType: 'fortitude',
    resistance: 'metade',
    components: VG,
    classes: ['Clérigo', 'Paladino'],
    baseEffect:
      'Nuvem 6m raio de energia negativa. Ao lançar e no início do turno do conjurador: criaturas na área sofrem 2d8+8 trevas (Fortitude metade). Falha = sem cura de PV por 1 rodada.',
    augments: [
      {
        pmCost: 2,
        kind: 'aumenta',
        description: 'Aumenta o dano em +1d8+4.',
      },
    ],
    bookPage: 201,
  },
  {
    id: 'possessao',
    name: 'Possessão',
    circle: 5,
    school: 'encantamento',
    execution: 'padrao',
    range: 'longo',
    duration: 'dia',
    saveType: 'vontade',
    resistance: 'anula',
    components: VG,
    classes: ['Arcanista', 'Bardo'],
    baseEffect:
      'Projeta consciência no corpo do alvo (1 criatura). Controle total; próprio corpo inconsciente; alvo inerte. Usa ficha própria com atributos físicos/deslocamento do alvo. Sucesso: alvo sabe da tentativa, imune por 1 dia. Corpo alvo morre com conjurador dentro: alvo morre; Vontade contra CD da própria magia — sucesso volta ao corpo (se em alcance); falha morre. Voltar voluntariamente = ação livre.',
    augments: [
      {
        pmCost: 5,
        kind: 'aumenta',
        description: 'Recebe acesso às habilidades de raça e classe da criatura.',
      },
      {
        pmCost: 5,
        kind: 'aumenta',
        description:
          'Enquanto durar e em alcance do próprio corpo, "salta" de possuída para nova (Vontade — falha = assume nova, anterior recobra consciência).',
      },
      {
        pmCost: 5,
        kind: 'muda',
        description:
          'Muda duração para permanente, destrói corpo original. Possuída faz Vontade no início do dia para retomar; sucesso recobra (conjurador fica inerte). Corpo morre + nova em alcance = pode possuir. Enquanto houver corpos, é imortal.',
      },
    ],
    bookPage: 201,
  },
  {
    id: 'potencia-divina',
    name: 'Potência Divina',
    circle: 3,
    school: 'transmutacao',
    execution: 'padrao',
    range: 'pessoal',
    duration: 'sustentada',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Clérigo', 'Paladino'],
    baseEffect:
      'Canaliza divindade: +1 categoria de tamanho (equipamento ajusta), Força +4, RD 10. Não pode lançar magias enquanto ativa.',
    augments: [
      {
        pmCost: 2,
        kind: 'aumenta',
        description: 'Aumenta o bônus de Força em +1.',
      },
      {
        pmCost: 5,
        kind: 'aumenta',
        description: 'Aumenta a RD em +5.',
      },
    ],
    bookPage: 201,
  },
  {
    id: 'premonicao',
    name: 'Premonição',
    circle: 4,
    school: 'adivinhacao',
    execution: 'padrao',
    range: 'pessoal',
    duration: 'cena',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Clérigo', 'Paladino'],
    baseEffect:
      'Vislumbres do futuro. Uma vez por rodada rola novamente um teste recém realizado; aceita o novo resultado.',
    augments: [
      {
        pmCost: 3,
        kind: 'muda',
        description:
          'Muda execução para reação, alcance para toque, alvo para 1 criatura, duração para instantânea. Contra criatura que acabou de testar — obriga novo teste e aceitar novo resultado. Involuntária faz Vontade para negar.',
      },
      {
        pmCost: 10,
        kind: 'muda',
        description: 'Muda a duração para 1 dia.',
      },
    ],
    bookPage: 201,
  },
  {
    id: 'primor-atletico',
    name: 'Primor Atlético',
    circle: 1,
    school: 'transmutacao',
    execution: 'padrao',
    range: 'toque',
    duration: 'cena',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Arcanista', 'Bardo'],
    baseEffect:
      '1 criatura: +9m deslocamento + +10 Atletismo.',
    augments: [
      {
        pmCost: 1,
        kind: 'aumenta',
        description:
          'Além do normal, +20 Atletismo para saltar (total +30).',
      },
      {
        pmCost: 1,
        kind: 'aumenta',
        description:
          'Além do normal, escala paredes/tetos sem Atletismo. Mãos livres (uma pode segurar). Não fica desprevenido.',
      },
      {
        pmCost: 1,
        kind: 'muda',
        description:
          'Muda execução para ação de movimento, alcance para pessoal, alvo para conjurador, duração para instantânea. Salta em corpo a corpo de criatura em alcance — se atacar corpo a corpo neste turno, ignora benefícios/penalidades de investida + arma causa 1 dado extra do mesmo tipo.',
      },
      {
        pmCost: 3,
        kind: 'aumenta',
        description:
          'Além do normal, em perícias baseadas em For/Des/Con rola 2 dados e usa o melhor. Não afeta ataque ou resistência.',
        requiresCircle: 2,
      },
    ],
    bookPage: 201,
  },
  {
    id: 'profanar',
    name: 'Profanar',
    circle: 1,
    school: 'necromancia',
    execution: 'padrao',
    range: 'longo',
    duration: 'dia',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Clérigo', 'Paladino'],
    baseEffect:
      'Enche esfera 9m com energia negativa. Dano de trevas maximizado na área. Não pode ser lançada em área com símbolo dedicado a outra divindade. Anula Consagrar.',
    augments: [
      {
        pmCost: 1,
        kind: 'aumenta',
        description: 'Além do normal, mortos-vivos na área recebem +2 Defesa e +2 em testes.',
      },
      {
        pmCost: 2,
        kind: 'aumenta',
        description:
          'Aumenta os bônus para mortos-vivos em +1 (máximo limitado pelo círculo máximo).',
      },
      {
        pmCost: 9,
        kind: 'muda',
        description:
          'Muda execução para 1 hora, duração para permanente, adiciona componente material (incenso e óleos T$ 1.000).',
        requiresCircle: 4,
      },
    ],
    bookPage: 202,
  },
  {
    id: 'projetar-consciencia',
    name: 'Projetar Consciência',
    circle: 5,
    school: 'adivinhacao',
    execution: 'padrao',
    range: 'ilimitado',
    duration: 'sustentada',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Arcanista', 'Bardo', 'Clérigo', 'Druida', 'Paladino'],
    baseEffect:
      'Consciência sai do corpo e vai a local conhecido ou perto de criatura (mesmo plano). Forma fantasmagórica invisível — ação de movimento para se mostrar. Move 18m, atravessa objetos sólidos. Restrita ao local escolhido ou alcance curto da criatura alvo. Vê e ouve como presente. Fala mentalmente com criaturas visíveis com idioma comum.',
    augments: [
      {
        pmCost: 10,
        kind: 'aumenta',
        description:
          'Além do normal, projeção lança magias sem componentes materiais e sem duração sustentada. Forma fantasmagórica igual a Forma Etérea (afetada por abjuração/essência), magias lançadas afetam corpóreas.',
      },
    ],
    bookPage: 202,
  },
  {
    id: 'protecao-contra-magia',
    name: 'Proteção Contra Magia',
    circle: 3,
    school: 'abjuracao',
    execution: 'padrao',
    range: 'toque',
    duration: 'cena',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Clérigo', 'Paladino'],
    baseEffect:
      '1 criatura: +5 em testes de resistência contra magias.',
    augments: [
      {
        pmCost: 4,
        kind: 'muda',
        description: 'Muda o bônus para +10.',
        requiresCircle: 4,
      },
      {
        pmCost: 4,
        kind: 'muda',
        description: 'Alvo fica imune a uma escola de magia à escolha.',
        requiresCircle: 4,
      },
      {
        pmCost: 9,
        kind: 'muda',
        description: 'Alvo fica imune a duas escolas de magia à escolha.',
        requiresCircle: 5,
      },
    ],
    bookPage: 202,
  },
  {
    id: 'protecao-divina',
    name: 'Proteção Divina',
    circle: 1,
    school: 'abjuracao',
    execution: 'padrao',
    range: 'toque',
    duration: 'cena',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Clérigo', 'Druida', 'Paladino'],
    baseEffect: 'Barreira mística: 1 criatura recebe +2 em testes de resistência.',
    augments: [
      {
        pmCost: 2,
        kind: 'aumenta',
        description: 'Aumenta o bônus concedido em +1.',
      },
      {
        pmCost: 2,
        kind: 'muda',
        description:
          'Muda execução para reação, alcance para curto, duração para 1 rodada. Alvo recebe +5 no próximo teste de resistência (cumulativo com o efeito básico).',
      },
      {
        pmCost: 2,
        kind: 'muda',
        description:
          'Muda alvo para esfera 3m raio. Todos os aliados dentro recebem o bônus.',
      },
      {
        pmCost: 5,
        kind: 'aumenta',
        description: 'Torna o alvo imune a efeitos mentais e de medo.',
        requiresCircle: 3,
      },
    ],
    bookPage: 202,
  },
  {
    id: 'purificacao',
    name: 'Purificação',
    circle: 2,
    school: 'evocacao',
    execution: 'padrao',
    range: 'toque',
    duration: 'instantanea',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Clérigo', 'Druida', 'Paladino'],
    baseEffect:
      'Remove uma condição da criatura tocada: abalado/apavorado/alquebrado/atordoado/cego/confuso/debilitado/enjoado/envenenado/esmorecido/exausto/fascinado/fatigado/fraco/frustrado/lento/ofuscado/paralisado.',
    augments: [
      {
        pmCost: 2,
        kind: 'aumenta',
        description: 'Também recupera todos os PV perdidos por veneno.',
      },
      {
        pmCost: 2,
        kind: 'muda',
        description: 'Em vez de uma, remove todas as condições listadas.',
      },
      {
        pmCost: 3,
        kind: 'aumenta',
        description:
          'Alvo pode soltar qualquer item amaldiçoado que segure (não remove a maldição do item).',
      },
      {
        pmCost: 7,
        kind: 'aumenta',
        description:
          'Dissipa magias e efeitos prejudiciais de encantamento, necromancia e transmutação afetando o alvo.',
        requiresCircle: 3,
      },
    ],
    bookPage: 202,
  },
  {
    id: 'raio-do-enfraquecimento',
    name: 'Raio do Enfraquecimento',
    circle: 1,
    school: 'necromancia',
    execution: 'padrao',
    range: 'curto',
    duration: 'cena',
    saveType: 'fortitude',
    resistance: 'parcial',
    components: VG,
    classes: ['Arcanista', 'Bardo'],
    baseEffect:
      'Raio púrpura em 1 criatura. Falha = fatigado. Sucesso = vulnerável. Não acumula (2 castes contra mesmo alvo não deixa exausto).',
    augments: [
      {
        pmCost: 0,
        kind: 'muda',
        description:
          'Truque: muda alcance para toque e resistência para Fortitude anula. Mão emana brilho púrpura; toque = fatigado.',
      },
      {
        pmCost: 2,
        kind: 'muda',
        description:
          'Falha = exausto. Sucesso = fatigado.',
        requiresCircle: 2,
      },
      {
        pmCost: 5,
        kind: 'muda',
        description: 'Como acima, mas alvo para criaturas escolhidas.',
        requiresCircle: 3,
      },
    ],
    bookPage: 202,
  },
  {
    id: 'raio-polar',
    name: 'Raio Polar',
    circle: 4,
    school: 'evocacao',
    execution: 'padrao',
    range: 'medio',
    duration: 'instantanea',
    saveType: 'fortitude',
    resistance: 'parcial',
    components: VG,
    classes: ['Arcanista', 'Bardo'],
    baseEffect:
      'Raio azul-esbranquiçado em 1 criatura: 10d8 frio + preso em bloco de gelo (paralisado). Sucesso = metade dano + lento 1 rodada. Bloco: 20 PV, RD 10, vulnerável a fogo. Presa: ação completa + Atletismo — sucesso causa 10 dano ao bloco (ignora RD).',
    augments: [
      {
        pmCost: 3,
        kind: 'aumenta',
        description: 'Aumenta o dano em +2d8.',
      },
      {
        pmCost: 5,
        kind: 'muda',
        description:
          'Muda alvo para esfera 6m raio. Bola de gelo explode causando efeito em todos.',
      },
    ],
    bookPage: 203,
  },
  {
    id: 'raio-solar',
    name: 'Raio Solar',
    circle: 2,
    school: 'evocacao',
    execution: 'padrao',
    range: 'pessoal',
    duration: 'instantanea',
    saveType: 'reflexos',
    resistance: 'metade',
    components: VG,
    classes: ['Clérigo', 'Druida', 'Paladino'],
    baseEffect:
      'Rajada de energia positiva em linha 30m. Criaturas: 4d8 luz (4d12 em mortos-vivos) + ofuscadas 1 rodada. Sucesso = metade dano + sem ofuscada.',
    augments: [
      {
        pmCost: 0,
        kind: 'muda',
        description:
          'Truque: muda duração para cena, resistência para nenhuma. Facho de luz ilumina a área. Ação livre por rodada muda direção.',
      },
      {
        pmCost: 2,
        kind: 'aumenta',
        description: 'Aumenta dano ou cura em +1d8 (+1d12 em mortos-vivos).',
      },
      {
        pmCost: 3,
        kind: 'muda',
        description:
          'Criaturas vivas na área curam 4d8 PV; restante sofre dano normalmente.',
      },
      {
        pmCost: 3,
        kind: 'aumenta',
        description: 'Falha na resistência = cegas 1d4 rodadas.',
      },
    ],
    bookPage: 203,
  },
  {
    id: 'reanimacao-impura',
    name: 'Reanimação Impura',
    circle: 5,
    school: 'necromancia',
    execution: 'completa',
    range: 'toque',
    duration: 'cena',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Clérigo', 'Druida', 'Paladino'],
    baseEffect:
      'Reanima criatura morta na mesma cena. Tipo vira morto-vivo; retém memórias e habilidades (inclui magia). Pensa e fala livre; obedece cegamente. Fim da cena volta morta. Destruída = não pode ser reanimada de novo.',
    augments: [],
    bookPage: 203,
  },
  {
    id: 'refugio',
    name: 'Refúgio',
    circle: 2,
    school: 'abjuracao',
    execution: 'completa',
    range: 'curto',
    duration: 'dia',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Arcanista', 'Bardo'],
    baseEffect:
      'Domo 6m raio imóvel, opaco por fora / transparente por dentro. Protege calor/frio/forças pequenas (neve, vento) — não dano (flecha, Bola de Fogo). Camuflagem total contra ataques externos. Entrada/saída livre. Descanso normal (PV+PM).',
    augments: [
      {
        pmCost: 1,
        kind: 'aumenta',
        description:
          'Além do normal, fumaça escura envolve limites — de fora não enxerga/ouve (de dentro sim). Bloqueia adivinhação.',
      },
      {
        pmCost: 3,
        kind: 'muda',
        description:
          'Cria cabana para 10 Médios. Descanso confortável (dobro do nível PV/PM). Cabana real: paredes madeira, telhado, porta, 2 janelas, mobília. Porta/janelas: 15 PV RD 5 + Tranca Arcana. Paredes: 200 PV RD 5.',
      },
      {
        pmCost: 3,
        kind: 'muda',
        description:
          'Cria espaço extradimensional (caverna vazia escura) para 10 Médios. Entrada embebida em objeto fixo (pedra/árvore). Efeitos do mundo real não afetam o espaço e vice-versa. Vê o mundo por janela na entrada. Ao terminar, conteúdo surge na área vazia mais próxima.',
        requiresCircle: 3,
      },
      {
        pmCost: 9,
        kind: 'muda',
        description:
          'Cria mansão extradimensional para 100 Médios: quartos luxuosos + comida/bebida + servos fantasmagóricos (Servos Invisíveis). Descanso luxuoso (triplo do nível PV/PM). Entrada única (porta de luz), visível/invisível por ação livre; só criaturas escolhidas passam.',
        requiresCircle: 4,
      },
    ],
    bookPage: 203,
  },
  {
    id: 'relampago-flamejante-de-reynard',
    name: 'Relâmpago Flamejante de Reynard',
    circle: 4,
    school: 'evocacao',
    execution: 'completa',
    range: 'curto',
    duration: 'sustentada',
    saveType: 'reflexos',
    resistance: 'metade',
    components: VG,
    classes: ['Arcanista', 'Bardo'],
    baseEffect:
      'Uma mão em chamas + outra eletrificada. Ação de movimento por turno: bola de fogo (10d6 esfera 6m) OU relâmpago (10d6 linha). Ação padrão duas mãos: energia mista (20d12, metade fogo/metade eletricidade, esfera 9m) — encerra a magia. Precisa das duas mãos livres. Tempo de execução não pode ser reduzido.',
    augments: [
      {
        pmCost: 2,
        kind: 'aumenta',
        description:
          'Aumenta o dano das rajadas em +1d6 e o da rajada mista em +2d12.',
      },
    ],
    bookPage: 203,
  },
  {
    id: 'requiem',
    name: 'Réquiem',
    circle: 5,
    school: 'ilusao',
    execution: 'completa',
    range: 'curto',
    duration: 'sustentada',
    saveType: 'vontade',
    resistance: 'anula',
    components: VG,
    classes: ['Arcanista', 'Bardo'],
    baseEffect:
      'Ilusão particular para cada criatura escolhida. Início do turno: Vontade — falha = acha que não tomou ações do turno anterior, obrigada a repeti-las com -5 cumulativo em testes (a penalidade não afeta o Vontade). Ação repetida consome PM/recursos; qualquer alvo faz teste de resistência com bônus = à penalidade.',
    augments: [],
    bookPage: 204,
  },
  {
    id: 'resistencia-a-energia',
    name: 'Resistência a Energia',
    circle: 1,
    school: 'abjuracao',
    execution: 'padrao',
    range: 'toque',
    duration: 'cena',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Arcanista', 'Bardo', 'Clérigo', 'Druida', 'Paladino'],
    baseEffect:
      'Ao lançar, escolha ácido/eletricidade/fogo/frio/luz/trevas. Alvo (1 criatura) recebe RD 10 contra o tipo escolhido.',
    augments: [
      {
        pmCost: 2,
        kind: 'aumenta',
        description: 'Aumenta a redução de dano em +5.',
      },
    ],
    bookPage: 204,
  },
  {
    id: 'rogar-maldicao',
    name: 'Rogar Maldição',
    circle: 2,
    school: 'necromancia',
    execution: 'padrao',
    range: 'curto',
    duration: 'sustentada',
    saveType: 'fortitude',
    resistance: 'anula',
    components: VG,
    classes: ['Clérigo', 'Paladino'],
    baseEffect:
      'Amaldiçoa 1 criatura. Escolha: Debilidade (esmorecido, não se comunica/lança magias, ainda reconhece aliados). Doença (muda duração para instantânea, alvo contrai doença sem incubação). Fraqueza (debilitado + lento). Isolamento (perde 1 sentido: visão=cego / audição=surdo / olfato/paladar=sem faro / tato=caído sem levantar). Mestre aprova maldições customizadas.',
    augments: [
      {
        pmCost: 3,
        kind: 'aumenta',
        description: 'Aumenta o número de efeitos que pode escolher em +1.',
        requiresCircle: 3,
      },
      {
        pmCost: 7,
        kind: 'muda',
        description:
          'Muda duração para permanente + adiciona Fortitude parcial. Sucesso = sofre por 1 rodada.',
        requiresCircle: 4,
      },
    ],
    bookPage: 204,
  },
  {
    id: 'roubar-a-alma',
    name: 'Roubar a Alma',
    circle: 5,
    school: 'necromancia',
    execution: 'padrao',
    range: 'toque',
    duration: 'permanente',
    saveType: 'vontade',
    resistance: 'anula',
    components: VG,
    classes: ['Arcanista', 'Bardo', 'Clérigo', 'Druida', 'Paladino'],
    baseEffect:
      'Rouba alma da vítima e armazena em objeto. Sucesso = abalado 1 rodada. Falha = corpo caído inconsciente inerte, alma no objeto. Corpo em estase (não envelhece/decompõe); pode ser atacado/destruído. Objeto: T$ 1.000 por nível/ND, sem alma prévia. Objeto destruído = magia esvai (alma retorna ao corpo se disponível; senão vai para os Mundos dos Deuses). Custo adicional: sacrifício 1 PM.',
    augments: [
      {
        pmCost: 5,
        kind: 'aumenta',
        description:
          'Objeto detém mesmos PM totais que o alvo. Empunhando: usa esses PM para lançar magias. Recupera PM/dia como descanso normal.',
      },
      {
        pmCost: 10,
        kind: 'aumenta',
        description:
          'Como reação ao lançar, possui o corpo sem alma como em Possessão (mesmo que não substitua a magia).',
      },
    ],
    bookPage: 204,
  },
  {
    id: 'runa-de-protecao',
    name: 'Runa de Proteção',
    circle: 2,
    school: 'abjuracao',
    execution: 'completa',
    range: 'toque',
    duration: 'definida',
    durationNote: 'permanente até ser descarregada',
    saveType: 'reflexos',
    resistance: 'metade',
    components: ['verbal', 'gestual', 'material'],
    classes: ['Arcanista', 'Bardo', 'Clérigo', 'Druida', 'Paladino'],
    baseEffect:
      'Runa em superfície fixa (parede/chão) — ativação por criatura entrando na área causa 6d6 em todos até 6m (tipo: ácido/eletricidade/fogo/frio/luz/trevas, escolhido ao lançar). Ativadora não testa Reflexos; outras metade. Ativação condicional (só goblins, só mortos-vivos) + palavra mágica desativa. Encontrar: Investigação; desarmar: Ladinagem. Componente material: pó de diamante T$ 200.',
    augments: [
      {
        pmCost: 1,
        kind: 'aumenta',
        description: 'Aumenta o dano em +2d6.',
      },
      {
        pmCost: 1,
        kind: 'muda',
        description:
          'Muda alcance para pessoal. Escolhe magia de 1º círc conhecida (ação padrão ou menos). Runa no próprio corpo + condição de ativação ("quando eu for alvo de ataque", "quando for alvo de magia"). Reação lança a magia. Uma runa no corpo por vez.',
      },
      {
        pmCost: 3,
        kind: 'muda',
        description: 'Como o aprimoramento anterior, mas magias de 2º círculo.',
        requiresCircle: 3,
      },
    ],
    bookPage: 204,
  },
  {
    id: 'salto-dimensional',
    name: 'Salto Dimensional',
    circle: 2,
    school: 'convocacao',
    execution: 'padrao',
    range: 'curto',
    duration: 'instantanea',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Arcanista', 'Bardo'],
    baseEffect:
      'Teletransporta conjurador para lugar no alcance (sem linha de efeito exigida; basta imaginar). Transportadas não agem até a próxima rodada. Não aparece dentro de corpo sólido; sem espaço livre = área vazia mais próxima.',
    augments: [
      {
        pmCost: 1,
        kind: 'muda',
        description: 'Muda o alcance para médio.',
      },
      {
        pmCost: 1,
        kind: 'muda',
        description:
          'Muda alvo para conjurador e 1 criatura voluntária. Pode ser escolhido múltiplas vezes (+1 alvo por vez), mas deve tocar todos.',
      },
      {
        pmCost: 2,
        kind: 'muda',
        description:
          'Muda execução para reação. +5 na Defesa e Reflexos contra ataque/efeito iminente. Após resolução, salta 1,5m para espaço adjacente.',
      },
      {
        pmCost: 3,
        kind: 'muda',
        description: 'Muda o alcance para longo.',
      },
    ],
    bookPage: 205,
  },
  {
    id: 'santuario',
    name: 'Santuário',
    circle: 1,
    school: 'abjuracao',
    execution: 'padrao',
    range: 'toque',
    duration: 'cena',
    saveType: 'vontade',
    resistance: 'anula',
    components: VG,
    classes: ['Clérigo', 'Paladino'],
    baseEffect:
      'Criatura que tente ação hostil contra 1 alvo: Vontade — falha = não consegue, perde ação, não tenta de novo enquanto ativa. Alvo não pode fazer ações hostis ou dissipa (habilidades/magias de cura e suporte OK).',
    augments: [
      {
        pmCost: 1,
        kind: 'aumenta',
        description:
          'Além do normal, escolha tipo (animal/construto/morto-vivo). Não pode ser percebido por criaturas mais inteligentes (INT -4 ou menor) do tipo.',
      },
      {
        pmCost: 9,
        kind: 'aumenta',
        description:
          'Também protege contra efeitos de área. Criatura que ataque área com alvo dentro: Vontade — falha = não consegue, perde ação. Só tenta de novo se o alvo sair da área.',
      },
    ],
    bookPage: 205,
  },
  {
    id: 'segunda-chance',
    name: 'Segunda Chance',
    circle: 5,
    school: 'evocacao',
    execution: 'padrao',
    range: 'toque',
    duration: 'instantanea',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Clérigo', 'Paladino'],
    baseEffect:
      'Brilho em asas de fênix: alvo recupera 200 PV + cura 1 condição (abalado/apavorado/alquebrado/atordoado/cego/confuso/debilitado/enjoado/envenenado/esmorecido/exausto/fascinado/fatigado/fraco/frustrado/lento/ofuscado/paralisado/pasmo/surdo).',
    augments: [
      {
        pmCost: 1,
        kind: 'aumenta',
        description: 'Aumenta a cura em +20 PV.',
      },
      {
        pmCost: 2,
        kind: 'muda',
        description: 'Muda alcance para curto e alvo para até 5 criaturas.',
      },
      {
        pmCost: 5,
        kind: 'muda',
        description:
          'Muda alvo para 1 criatura morta há até 1 rodada. Pode curá-la.',
      },
    ],
    bookPage: 205,
  },
  {
    id: 'selo-de-mana',
    name: 'Selo de Mana',
    circle: 3,
    school: 'encantamento',
    execution: 'padrao',
    range: 'toque',
    duration: 'cena',
    saveType: 'vontade',
    resistance: 'anula',
    components: VG,
    classes: ['Arcanista', 'Bardo', 'Clérigo', 'Druida', 'Paladino'],
    baseEffect:
      'Selo mágico na pele do alvo (1 criatura). Ação que gasta PM: Vontade — sucesso = ação normal; falha = ação sem efeito (PM gastos).',
    augments: [
      {
        pmCost: 4,
        kind: 'muda',
        description:
          'Muda alcance para curto e alvo para criaturas escolhidas no alcance.',
        requiresCircle: 4,
      },
    ],
    bookPage: 205,
  },
  {
    id: 'semiplano',
    name: 'Semiplano',
    circle: 5,
    school: 'convocacao',
    execution: 'completa',
    range: 'curto',
    duration: 'dia',
    saveType: 'none',
    resistance: null,
    components: VG,
    classes: ['Arcanista', 'Bardo'],
    baseEffect:
      'Dimensão particular 30m lado. Entrar: ação padrão + 10 PM. Leva voluntárias (1 PM extra por criatura) e objetos (1 PM Médio-, 2 PM Grande, 5 PM Enorme, 10 PM Colossal). Ação completa volta ao plano material no mesmo local. Viagem Planar volta em outro local. Escolhe forma/aparência; ar/luz/calor; itens levados persistem.',
    augments: [
      {
        pmCost: 2,
        kind: 'aumenta',
        description:
          'Adiciona alvo (1 criatura) expulsa para semiplano labiríntico. A cada rodada: Investigação ou Sobrevivência com +1 cumulativo por teste. Escapa = magia termina, reaparece no plano material. Salto Dimensional/Teletransporte não ajudam; Viagem Planar sim.',
      },
      {
        pmCost: 5,
        kind: 'muda',
        description:
          'Muda duração para permanente + componente material (maquete T$ 5.000). Múltiplas castes aumentam semiplano em +30m de lado cada.',
      },
    ],
    bookPage: 205,
  },
  {
    id: 'servo-divino',
    name: 'Servo Divino',
    circle: 3,
    school: 'convocacao',
    execution: 'padrao',
    range: 'curto',
    duration: 'definida',
    durationNote: 'cena ou até ser descarregada',
    saveType: 'none',
    resistance: null,
    components: ['verbal', 'gestual', 'material'],
    classes: ['Clérigo', 'Paladino'],
    baseEffect:
      'Divindade envia espírito para tarefa até 1h ("nos leve ao topo da montanha", "escolte esses camponeses"). Descarregada ao cumprir. Tipo escolhido pelo mestre. Componente material: T$ 100 pagamento (doação/item mágico/ouro).',
    augments: [
      {
        pmCost: 4,
        kind: 'muda',
        description:
          'Muda duração para 1 dia ou até ser descarregada. Tarefa até 1 dia. Pagamento sobe para T$ 500.',
      },
      {
        pmCost: 9,
        kind: 'muda',
        description:
          'Muda duração para 1 semana ou até ser descarregada. Tarefa até 1 semana. Pagamento sobe para T$ 1.000.',
      },
    ],
    bookPage: 205,
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
