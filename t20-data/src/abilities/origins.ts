import type { Modifier } from '../items/types'
import type { OriginBenefit, OriginDefinition } from './types'

/**
 * T20 origens (PDF Cap. 1, p85+ / Tabela 1-19 p87). Each origin has a benefits
 * pool the player picks 2 from (mix of perícias and poderes). Each also has a
 * `poderUnico` exclusive to it that may be picked as one of the two slots.
 *
 * - `kind`: 'pericia' grants treinamento in the listed expertise; 'poder'
 *   grants the named poder geral. Modifiers are only embedded for poderes
 *   with clearly always-on numeric effects (no per-scene actions, no
 *   conditional terrain bonuses, no spell-economy effects); everything else
 *   ships as descriptive text so the player tracks it manually.
 * - Some entries reference free-pick poderes ("um poder de combate a sua
 *   escolha", "um poder da Tormenta a sua escolha"). These are modeled as
 *   regular benefits with no numeric mods — the player picks the specific
 *   power off-catalog.
 */

const sortudo: OriginBenefit = {
  id: 'poder-sortudo',
  name: 'Sortudo',
  kind: 'poder',
  description: 'Quando faz um teste, pode gastar 3 PM para rolá-lo novamente.',
}

const vontadeDeFerro: OriginBenefit = {
  id: 'poder-vontade-de-ferro',
  name: 'Vontade de Ferro',
  kind: 'poder',
  description: 'Você recebe +2 em testes de Vontade.',
  modifiers: [
    {
      target: { k: 'expertise', name: 'Vontade' },
      amount: 2,
      bonusType: 'untyped',
    },
  ],
}

const atraente: OriginBenefit = {
  id: 'poder-atraente',
  name: 'Atraente',
  kind: 'poder',
  description: 'Você recebe +2 em Diplomacia, Enganação e Intimidação.',
  modifiers: [
    { target: { k: 'expertise', name: 'Diplomacia' }, amount: 2, bonusType: 'untyped' },
    { target: { k: 'expertise', name: 'Enganação' }, amount: 2, bonusType: 'untyped' },
    { target: { k: 'expertise', name: 'Intimidação' }, amount: 2, bonusType: 'untyped' },
  ],
}

const torcida: OriginBenefit = {
  id: 'poder-torcida',
  name: 'Torcida',
  kind: 'poder',
  description:
    'Você recebe +2 em testes de perícia e Defesa quando tem a torcida a seu favor (qualquer número de criaturas inteligentes torcendo).',
}

const acrobatico: OriginBenefit = {
  id: 'poder-acrobatico',
  name: 'Acrobático',
  kind: 'poder',
  description: 'Você recebe +2 em testes de Acrobacia e Furtividade.',
  modifiers: [
    { target: { k: 'expertise', name: 'Acrobacia' }, amount: 2, bonusType: 'untyped' },
    { target: { k: 'expertise', name: 'Furtividade' }, amount: 2, bonusType: 'untyped' },
  ],
}

const sentidosAgucados: OriginBenefit = {
  id: 'poder-sentidos-agucados',
  name: 'Sentidos Aguçados',
  kind: 'poder',
  description: 'Você recebe +2 em testes de Percepção e Sobrevivência.',
  modifiers: [
    { target: { k: 'expertise', name: 'Percepção' }, amount: 2, bonusType: 'untyped' },
    { target: { k: 'expertise', name: 'Sobrevivência' }, amount: 2, bonusType: 'untyped' },
  ],
}

const loboSolitario: OriginBenefit = {
  id: 'poder-lobo-solitario',
  name: 'Lobo Solitário',
  kind: 'poder',
  description:
    'Quando está sozinho (sem aliados em alcance curto), recebe +2 em testes de perícia, ataque e Defesa.',
}

const vitalidade: OriginBenefit = {
  id: 'poder-vitalidade',
  name: 'Vitalidade',
  kind: 'poder',
  description: 'Você recebe +1 PV por nível de personagem e +2 em Fortitude.',
  modifiers: [
    { target: { k: 'expertise', name: 'Fortitude' }, amount: 2, bonusType: 'untyped' },
  ],
}

const proficiencia: OriginBenefit = {
  id: 'poder-proficiencia',
  name: 'Proficiência',
  kind: 'poder',
  description:
    'Escolha uma das proficiências a seguir: armas marciais, armas exóticas, armas de fogo, armaduras pesadas ou escudos. Você se torna proficiente nelas.',
}

const focoEmPericia: OriginBenefit = {
  id: 'poder-foco-em-pericia',
  name: 'Foco em Perícia',
  kind: 'poder',
  description:
    'Escolha uma perícia. Você recebe +2 em testes desta perícia. Você pode escolher este poder várias vezes, para perícias diferentes.',
}

const comandar: OriginBenefit = {
  id: 'poder-comandar',
  name: 'Comandar',
  kind: 'poder',
  description:
    'Pode gastar uma ação padrão e 2 PM para ordenar uma ação a um aliado em alcance curto. Se o aliado obedecer, ganha +2 em testes para realizar essa ação.',
}

const medicina: OriginBenefit = {
  id: 'poder-medicina',
  name: 'Medicina',
  kind: 'poder',
  description:
    'Você recebe +2 em testes de Cura. Pode estabilizar uma criatura em 0 PV com ação livre.',
  modifiers: [
    { target: { k: 'expertise', name: 'Cura' }, amount: 2, bonusType: 'untyped' },
  ],
}

const venefico: OriginBenefit = {
  id: 'poder-venefico',
  name: 'Venefício',
  kind: 'poder',
  description:
    'Você é treinado em fabricar e aplicar venenos. Não corre risco de envenenar a si mesmo ao aplicar veneno em uma arma.',
}

const aparenciaInofensiva: OriginBenefit = {
  id: 'poder-aparencia-inofensiva',
  name: 'Aparência Inofensiva',
  kind: 'poder',
  description:
    'Sua aparência faz com que ninguém o considere uma ameaça. Recebe +5 em Enganação para enganar inteligências e fica imune a Intuição direcionada a você.',
}

const atletico: OriginBenefit = {
  id: 'poder-atletico',
  name: 'Atlético',
  kind: 'poder',
  description: 'Você recebe +2 em testes de Atletismo e Fortitude.',
  modifiers: [
    { target: { k: 'expertise', name: 'Atletismo' }, amount: 2, bonusType: 'untyped' },
    { target: { k: 'expertise', name: 'Fortitude' }, amount: 2, bonusType: 'untyped' },
  ],
}

const surtoHeroico: OriginBenefit = {
  id: 'poder-surto-heroico',
  name: 'Surto Heroico',
  kind: 'poder',
  description:
    'Uma vez por dia, pode gastar uma ação livre para receber +5 em um teste, ataque ou Defesa contra um único ataque.',
}

// Helper builders to keep entries terse.
function pericia(
  expertise: import('../expertises').ExpertiseName,
  origin: string,
): OriginBenefit {
  return {
    id: `origin-${origin}-pericia-${expertise}`,
    name: expertise,
    kind: 'pericia',
    description: `Você se torna treinado em ${expertise}.`,
    expertise,
  }
}

function uniquePoder(
  origin: string,
  name: string,
  description: string,
  modifiers?: Modifier[],
): OriginBenefit {
  return {
    id: `origin-${origin}-unique`,
    name,
    kind: 'poder',
    description,
    modifiers,
  }
}

function placeholderPoder(
  origin: string,
  name: string,
  description: string,
): OriginBenefit {
  return { id: `origin-${origin}-poder-${slug(name)}`, name, kind: 'poder', description }
}

function slug(s: string): string {
  return s
    .toLowerCase()
    .normalize('NFD')
    .replace(/[̀-ͯ]/g, '')
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-|-$/g, '')
}

export const ORIGINS_CATALOG: OriginDefinition[] = [
  {
    id: 'Acólito',
    name: 'Acólito',
    benefits: [
      pericia('Cura', 'acolito'),
      pericia('Religião', 'acolito'),
      pericia('Vontade', 'acolito'),
      medicina,
      vontadeDeFerro,
    ],
    poderUnico: uniquePoder(
      'acolito',
      'Membro da Igreja',
      'Você consegue hospedagem confortável e informação em qualquer templo de sua divindade, para você e seus aliados.',
    ),
  },
  {
    id: 'Amigo dos Animais',
    name: 'Amigo dos Animais',
    benefits: [
      pericia('Adestramento', 'amigo-animais'),
      pericia('Cavalgar', 'amigo-animais'),
    ],
    poderUnico: uniquePoder(
      'amigo-animais',
      'Amigo Especial',
      'Você recebe +5 em testes de Adestramento com animais. Possui um animal de estimação que fornece +2 em uma perícia a sua escolha (exceto Luta ou Pontaria) e não conta em seu limite de parceiros.',
    ),
  },
  {
    id: 'Amnésico',
    name: 'Amnésico',
    benefits: [],
    poderUnico: uniquePoder(
      'amnesico',
      'Lembranças Graduais',
      'Em vez de dois benefícios, recebe uma perícia e um poder escolhidos pelo mestre, mais Lembranças Graduais. Em momentos a critério do mestre, pode fazer um teste de Sabedoria (CD 10) para reconhecer pessoas, criaturas ou lugares do passado esquecido.',
    ),
  },
  {
    id: 'Aristocrata',
    name: 'Aristocrata',
    benefits: [
      pericia('Diplomacia', 'aristocrata'),
      pericia('Enganação', 'aristocrata'),
      pericia('Nobreza', 'aristocrata'),
      comandar,
    ],
    poderUnico: uniquePoder(
      'aristocrata',
      'Sangue Azul',
      'Você tem influência política suficiente para ser tratado com leniência pela guarda e conseguir audiência com o nobre local.',
    ),
  },
  {
    id: 'Artesão',
    name: 'Artesão',
    benefits: [
      pericia('Ofício', 'artesao'),
      pericia('Vontade', 'artesao'),
      sortudo,
    ],
    poderUnico: uniquePoder(
      'artesao',
      'Frutos do Trabalho',
      'No início de cada aventura, recebe até 5 itens gerais que possa fabricar num valor total de até T$ 50 (T$ 100 veterano, T$ 300 campeão, T$ 500 lenda).',
    ),
  },
  {
    id: 'Artista',
    name: 'Artista',
    benefits: [
      pericia('Atuação', 'artista'),
      pericia('Enganação', 'artista'),
      atraente,
      sortudo,
      torcida,
    ],
    poderUnico: uniquePoder(
      'artista',
      'Dom Artístico',
      'Você recebe +2 em testes de Atuação e recebe o dobro de tibares em apresentações.',
      [
        { target: { k: 'expertise', name: 'Atuação' }, amount: 2, bonusType: 'untyped' },
      ],
    ),
  },
  {
    id: 'Assistente de Laboratório',
    name: 'Assistente de Laboratório',
    benefits: [
      pericia('Ofício', 'assistente'),
      pericia('Misticismo', 'assistente'),
      venefico,
      placeholderPoder(
        'assistente',
        'Poder da Tormenta (escolha)',
        'Você recebe um poder da Tormenta a sua escolha (anote separadamente). Pré-requisitos do poder se aplicam.',
      ),
    ],
    poderUnico: uniquePoder(
      'assistente',
      'Esse Cheiro...',
      'Você recebe +2 em Fortitude e detecta automaticamente a presença (mas não a localização ou natureza) de itens alquímicos em alcance curto.',
      [
        { target: { k: 'expertise', name: 'Fortitude' }, amount: 2, bonusType: 'untyped' },
      ],
    ),
  },
  {
    id: 'Batedor',
    name: 'Batedor',
    benefits: [
      pericia('Furtividade', 'batedor'),
      pericia('Percepção', 'batedor'),
      pericia('Sobrevivência', 'batedor'),
      placeholderPoder(
        'batedor',
        'À Prova de Tudo',
        'Você recebe +2 em testes de resistência contra fadiga e efeitos de terreno (frio, calor, sede, fome).',
      ),
      sentidosAgucados,
    ],
    poderUnico: uniquePoder(
      'batedor',
      'Estilo de Disparo',
      'Escolha um tipo de arma à distância (arcos, bestas, armas de arremesso ou armas de fogo). Você não sofre penalidade ao atacar à distância contra alvos engajados em combate corpo a corpo.',
    ),
  },
  {
    id: 'Capanga',
    name: 'Capanga',
    benefits: [
      pericia('Luta', 'capanga'),
      pericia('Intimidação', 'capanga'),
      placeholderPoder(
        'capanga',
        'Poder de Combate (escolha)',
        'Você recebe um poder de combate a sua escolha (anote separadamente). Pré-requisitos do poder se aplicam.',
      ),
    ],
    poderUnico: uniquePoder(
      'capanga',
      'Confissão',
      'Pode gastar uma ação completa e 1 PM para forçar uma criatura indefesa a fazer um teste de Vontade contra Intimidação. Se falhar, revela informações ao alcance do conhecimento dela.',
    ),
  },
  {
    id: 'Charlatão',
    name: 'Charlatão',
    benefits: [
      pericia('Enganação', 'charlatao'),
      pericia('Jogatina', 'charlatao'),
      aparenciaInofensiva,
      sortudo,
    ],
    poderUnico: uniquePoder(
      'charlatao',
      'Alpinista Social',
      'Você consegue forjar credenciais e se passar por membros de classes sociais altas. Recebe +5 em Enganação para se passar por outra pessoa de status social diferente.',
    ),
  },
  {
    id: 'Circense',
    name: 'Circense',
    benefits: [
      pericia('Acrobacia', 'circense'),
      pericia('Atuação', 'circense'),
      pericia('Reflexos', 'circense'),
      acrobatico,
      torcida,
    ],
    poderUnico: uniquePoder(
      'circense',
      'Truque de Mágica',
      'Você conhece truques de prestidigitação. Pode realizar pequenos efeitos mágicos cosméticos (acender velas, mudar a cor de um objeto pequeno) com ação livre.',
    ),
  },
  {
    id: 'Criminoso',
    name: 'Criminoso',
    benefits: [
      pericia('Enganação', 'criminoso'),
      pericia('Furtividade', 'criminoso'),
      pericia('Ladinagem', 'criminoso'),
      venefico,
    ],
    poderUnico: uniquePoder(
      'criminoso',
      'Punguista',
      'Você recebe +2 em testes de Ladinagem para furtar e em testes de Furtividade. Pode esconder um objeto pequeno na manga ou bolso interno como ação livre.',
      [
        { target: { k: 'expertise', name: 'Ladinagem' }, amount: 2, bonusType: 'untyped' },
        { target: { k: 'expertise', name: 'Furtividade' }, amount: 2, bonusType: 'untyped' },
      ],
    ),
  },
  {
    id: 'Curandeiro',
    name: 'Curandeiro',
    benefits: [
      pericia('Cura', 'curandeiro'),
      pericia('Vontade', 'curandeiro'),
      medicina,
      venefico,
    ],
    poderUnico: uniquePoder(
      'curandeiro',
      'Médico de Campo',
      'Pode gastar uma ação padrão e 1 PM para fazer um teste de Cura (CD 20) curando 1d8+Sabedoria PV. Cada criatura só pode ser alvo deste efeito uma vez por dia.',
    ),
  },
  {
    id: 'Eremita',
    name: 'Eremita',
    benefits: [
      pericia('Misticismo', 'eremita'),
      pericia('Religião', 'eremita'),
      pericia('Sobrevivência', 'eremita'),
      loboSolitario,
    ],
    poderUnico: uniquePoder(
      'eremita',
      'Busca Interior',
      'Você pode gastar 1 minuto meditando para receber +5 no próximo teste de perícia mental que fizer (Conhecimento, Misticismo, Religião, etc.). Limite uma vez por descanso.',
    ),
  },
  {
    id: 'Escravo',
    name: 'Escravo',
    benefits: [
      pericia('Atletismo', 'escravo'),
      pericia('Fortitude', 'escravo'),
      pericia('Furtividade', 'escravo'),
      vitalidade,
    ],
    poderUnico: uniquePoder(
      'escravo',
      'Desejo de Liberdade',
      'Você recebe +2 em testes de resistência contra efeitos que o agarrem, prendam, paralizem ou imobilizem.',
    ),
  },
  {
    id: 'Estudioso',
    name: 'Estudioso',
    benefits: [
      pericia('Conhecimento', 'estudioso'),
      pericia('Guerra', 'estudioso'),
      pericia('Misticismo', 'estudioso'),
      aparenciaInofensiva,
    ],
    poderUnico: uniquePoder(
      'estudioso',
      'Palpite Fundamentado',
      'Quando faz um teste de perícia baseado em Inteligência, pode gastar 1 PM para somar +5 no resultado.',
    ),
  },
  {
    id: 'Fazendeiro',
    name: 'Fazendeiro',
    benefits: [
      pericia('Adestramento', 'fazendeiro'),
      pericia('Cavalgar', 'fazendeiro'),
      pericia('Ofício', 'fazendeiro'),
      pericia('Sobrevivência', 'fazendeiro'),
      placeholderPoder(
        'fazendeiro',
        'Ginete',
        'Quando montado, você ignora a penalidade de armadura em testes de Cavalgar e recebe +2 em testes de Cavalgar.',
      ),
    ],
    poderUnico: uniquePoder(
      'fazendeiro',
      'Água no Feijão',
      'Você sabe esticar mantimentos. Cada ração que consome conta como dois dias de comida.',
    ),
  },
  {
    id: 'Forasteiro',
    name: 'Forasteiro',
    benefits: [
      pericia('Cavalgar', 'forasteiro'),
      pericia('Pilotagem', 'forasteiro'),
      pericia('Sobrevivência', 'forasteiro'),
      loboSolitario,
    ],
    poderUnico: uniquePoder(
      'forasteiro',
      'Cultura Exótica',
      'Você é versado em uma cultura distante. Recebe +2 em testes de Conhecimento e Nobreza relacionados a essa cultura, e aprende um idioma exótico.',
    ),
  },
  {
    id: 'Gladiador',
    name: 'Gladiador',
    benefits: [
      pericia('Atuação', 'gladiador'),
      pericia('Luta', 'gladiador'),
      atraente,
      torcida,
      placeholderPoder(
        'gladiador',
        'Poder de Combate (escolha)',
        'Você recebe um poder de combate a sua escolha (anote separadamente). Pré-requisitos do poder se aplicam.',
      ),
    ],
    poderUnico: uniquePoder(
      'gladiador',
      'Pão e Circo',
      'Quando entra em combate na presença de uma plateia (10 ou mais criaturas), recebe +1 em ataque e Defesa enquanto a plateia o assistir.',
    ),
  },
  {
    id: 'Guarda',
    name: 'Guarda',
    benefits: [
      pericia('Investigação', 'guarda'),
      pericia('Luta', 'guarda'),
      pericia('Percepção', 'guarda'),
      placeholderPoder(
        'guarda',
        'Investigador',
        'Você recebe +2 em Investigação e Percepção. Pode dobrar o tempo de uma investigação para receber +5 no teste final.',
        // Note: numeric mods deliberately omitted because the +5 portion is
        // gated on doubling time — the base +2 is what we surface.
      ),
      placeholderPoder(
        'guarda',
        'Poder de Combate (escolha)',
        'Você recebe um poder de combate a sua escolha (anote separadamente). Pré-requisitos do poder se aplicam.',
      ),
    ],
    poderUnico: uniquePoder(
      'guarda',
      'Detetive',
      'Você recebe +2 em testes de Intuição e Investigação. Sabe perceber rapidamente se uma criatura está mentindo ou escondendo informações.',
      [
        { target: { k: 'expertise', name: 'Intuição' }, amount: 2, bonusType: 'untyped' },
        { target: { k: 'expertise', name: 'Investigação' }, amount: 2, bonusType: 'untyped' },
      ],
    ),
  },
  {
    id: 'Herdeiro',
    name: 'Herdeiro',
    benefits: [
      pericia('Misticismo', 'herdeiro'),
      pericia('Nobreza', 'herdeiro'),
      pericia('Ofício', 'herdeiro'),
      comandar,
    ],
    poderUnico: uniquePoder(
      'herdeiro',
      'Herança',
      'No início de cada aventura, recebe um item mágico ou heirloom de até T$ 500 emprestado da família (T$ 1.500 veterano, T$ 5.000 campeão, T$ 15.000 lenda). O item retorna no fim da aventura.',
    ),
  },
  {
    id: 'Herói Camponês',
    name: 'Herói Camponês',
    benefits: [
      pericia('Adestramento', 'heroi-camponel'),
      pericia('Ofício', 'heroi-camponel'),
      sortudo,
      surtoHeroico,
      torcida,
    ],
    poderUnico: uniquePoder(
      'heroi-camponel',
      'Coração Heroico',
      'Pode gastar 2 PM como ação livre para receber pontos de vida temporários iguais ao seu nível + sua Carisma até o fim da cena.',
    ),
  },
  {
    id: 'Marujo',
    name: 'Marujo',
    benefits: [
      pericia('Atletismo', 'marujo'),
      pericia('Jogatina', 'marujo'),
      pericia('Pilotagem', 'marujo'),
      acrobatico,
    ],
    poderUnico: uniquePoder(
      'marujo',
      'Passagem de Navio',
      'Você pode conseguir passagem grátis em qualquer navio mercante (geralmente em troca de pequenos serviços).',
    ),
  },
  {
    id: 'Mateiro',
    name: 'Mateiro',
    benefits: [
      pericia('Atletismo', 'mateiro'),
      pericia('Furtividade', 'mateiro'),
      pericia('Sobrevivência', 'mateiro'),
      loboSolitario,
      sentidosAgucados,
    ],
    poderUnico: uniquePoder(
      'mateiro',
      'Vendedor de Carcaças',
      'Você sabe extrair couro, presas e ingredientes alquímicos de criaturas abatidas. Cada carcaça apropriada gera até T$ 5 × ND da criatura em mercadorias quando vendida.',
    ),
  },
  {
    id: 'Membro de Guilda',
    name: 'Membro de Guilda',
    benefits: [
      pericia('Diplomacia', 'guilda'),
      pericia('Enganação', 'guilda'),
      pericia('Misticismo', 'guilda'),
      pericia('Ofício', 'guilda'),
      focoEmPericia,
    ],
    poderUnico: uniquePoder(
      'guilda',
      'Rede de Contatos',
      'Você tem contatos em sua guilda. Em uma cidade grande, pode encontrar um contato útil (mercador, informante, especialista) com um teste de Diplomacia CD 15.',
    ),
  },
  {
    id: 'Mercador',
    name: 'Mercador',
    benefits: [
      pericia('Diplomacia', 'mercador'),
      pericia('Intuição', 'mercador'),
      pericia('Ofício', 'mercador'),
      proficiencia,
      sortudo,
    ],
    poderUnico: uniquePoder(
      'mercador',
      'Negociação',
      'Você compra itens por 90% do preço e vende por 60% (em vez de 50%). Recebe +2 em testes de Diplomacia para barganhar.',
    ),
  },
  {
    id: 'Minerador',
    name: 'Minerador',
    benefits: [
      pericia('Atletismo', 'minerador'),
      pericia('Fortitude', 'minerador'),
      pericia('Ofício', 'minerador'),
      placeholderPoder(
        'minerador',
        'Ataque Poderoso',
        'Quando faz um ataque corpo a corpo, pode reduzir sua chance de acertar para causar mais dano (–5 ataque, +10 dano).',
      ),
      sentidosAgucados,
    ],
    poderUnico: uniquePoder(
      'minerador',
      'Escavador',
      'Em terreno apropriado, pode cavar a 1,5m por minuto com ferramentas, ou 3m por minuto com magia ou Força elevada. Detecta automaticamente passagens secretas em paredes de pedra a alcance curto.',
    ),
  },
  {
    id: 'Nômade',
    name: 'Nômade',
    benefits: [
      pericia('Cavalgar', 'nomade'),
      pericia('Pilotagem', 'nomade'),
      pericia('Sobrevivência', 'nomade'),
      loboSolitario,
      sentidosAgucados,
    ],
    poderUnico: uniquePoder(
      'nomade',
      'Mochileiro',
      'Você recebe +2 espaços em seu limite de itens equipados e leves. Adicionalmente, não sofre penalidade de armadura por excesso de carga até o limite máximo.',
      [
        { target: { k: 'inventorySlots' }, amount: 2, bonusType: 'untyped' },
      ],
    ),
  },
  {
    id: 'Pivete',
    name: 'Pivete',
    benefits: [
      pericia('Furtividade', 'pivete'),
      pericia('Iniciativa', 'pivete'),
      pericia('Ladinagem', 'pivete'),
      acrobatico,
      aparenciaInofensiva,
    ],
    poderUnico: uniquePoder(
      'pivete',
      'Quebra-Galho',
      'Você consegue se adaptar a qualquer perícia. Quando faz um teste de perícia em que não é treinado, recebe metade do seu bônus de treino (em vez de nada).',
    ),
  },
  {
    id: 'Refugiado',
    name: 'Refugiado',
    benefits: [
      pericia('Fortitude', 'refugiado'),
      pericia('Reflexos', 'refugiado'),
      pericia('Vontade', 'refugiado'),
      vontadeDeFerro,
    ],
    poderUnico: uniquePoder(
      'refugiado',
      'Estoico',
      'Quando sofre uma condição negativa (abalado, agarrado, atordoado, etc.), pode gastar 1 PM como reação para ignorar a condição até o fim do próximo turno.',
    ),
  },
  {
    id: 'Seguidor',
    name: 'Seguidor',
    benefits: [
      pericia('Adestramento', 'seguidor'),
      pericia('Ofício', 'seguidor'),
      proficiencia,
      surtoHeroico,
    ],
    poderUnico: uniquePoder(
      'seguidor',
      'Antigo Mestre',
      'Você aprendeu segredos com um mentor poderoso. Recebe um poder de classe a sua escolha (cumprindo pré-requisitos) cujo atributo-chave é Carisma.',
    ),
  },
  {
    id: 'Selvagem',
    name: 'Selvagem',
    benefits: [
      pericia('Percepção', 'selvagem'),
      pericia('Reflexos', 'selvagem'),
      pericia('Sobrevivência', 'selvagem'),
      loboSolitario,
      vitalidade,
    ],
    poderUnico: uniquePoder(
      'selvagem',
      'Vida Rústica',
      'Você é imune a doenças mundanas e sofre apenas metade do dano de exposição (frio, calor, fome, sede).',
    ),
  },
  {
    id: 'Soldado',
    name: 'Soldado',
    benefits: [
      pericia('Fortitude', 'soldado'),
      pericia('Guerra', 'soldado'),
      pericia('Luta', 'soldado'),
      pericia('Pontaria', 'soldado'),
      placeholderPoder(
        'soldado',
        'Poder de Combate (escolha)',
        'Você recebe um poder de combate a sua escolha (anote separadamente). Pré-requisitos do poder se aplicam.',
      ),
    ],
    poderUnico: uniquePoder(
      'soldado',
      'Influência Militar',
      'Você pode conseguir audiência com oficiais militares e obter informações sobre tropas, fortes e estradas militares.',
    ),
  },
  {
    id: 'Taverneiro',
    name: 'Taverneiro',
    benefits: [
      pericia('Diplomacia', 'taverneiro'),
      pericia('Jogatina', 'taverneiro'),
      pericia('Ofício', 'taverneiro'),
      proficiencia,
      vitalidade,
    ],
    poderUnico: uniquePoder(
      'taverneiro',
      'Gororoba',
      'Você sabe preparar pratos simples que aguentam o frio e a fome. Durante o descanso, sua refeição concede +1 PV temporário por nível a cada aliado que comer.',
    ),
  },
  {
    id: 'Trabalhador',
    name: 'Trabalhador',
    benefits: [
      pericia('Atletismo', 'trabalhador'),
      pericia('Fortitude', 'trabalhador'),
      atletico,
    ],
    poderUnico: uniquePoder(
      'trabalhador',
      'Esforçado',
      'Você é resistente ao esforço. Pode trabalhar pesado por +4 horas além do normal sem sofrer penalidades de fadiga, e recebe +2 em testes de Atletismo para tarefas repetitivas.',
    ),
  },
]

/**
 * Modifiers contributed by an origin's chosen benefits. `choiceSet` is the
 * set of benefit ids the player has selected (including the poderUnico id
 * if picked). Only benefits with embedded `modifiers` contribute — perícia
 * benefits don't auto-train the expertise via the engine; the player toggles
 * trained state on the expertise panel.
 */
export function originModifiers(
  origin: OriginDefinition,
  choiceSet: ReadonlySet<string>,
): Modifier[] {
  const out: Modifier[] = []
  const all: OriginBenefit[] = [...origin.benefits, origin.poderUnico]
  for (const benefit of all) {
    if (!choiceSet.has(benefit.id)) continue
    if (benefit.modifiers) out.push(...benefit.modifiers)
  }
  return out
}
