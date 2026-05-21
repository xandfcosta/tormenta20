import type { RaceDefinition } from './types'

/**
 * Seed catalog for races. Each `id` matches the localized RACES constant in
 * the backend so the stored `Character.races[].race` string is a valid lookup
 * key. 'Kiiren' here mirrors the backend constant (PDF spells it "Kliren").
 */
export const RACES_CATALOG: RaceDefinition[] = [
  {
    id: 'Humano',
    name: 'Humano',
    attributeBonuses: {},
    abilities: [
      {
        id: 'humano-tres-atributos',
        raceId: 'Humano',
        name: '+1 em Três Atributos Diferentes',
        description:
          'Filhos de Valkaria, humanos podem se destacar em qualquer caminho. Distribua +1 em três atributos diferentes (já aplicado na ficha).',
      },
      {
        id: 'humano-versatil',
        raceId: 'Humano',
        name: 'Versátil',
        description:
          'Você se torna treinado em duas perícias a sua escolha (não precisam ser da sua classe). Pode trocar uma dessas perícias por um poder geral a sua escolha.',
      },
    ],
  },
  {
    id: 'Anão',
    name: 'Anão',
    attributeBonuses: { constitution: 2, wisdom: 1, dexterity: -1 },
    abilities: [
      {
        id: 'anao-conhecimento-das-rochas',
        raceId: 'Anão',
        name: 'Conhecimento das Rochas',
        description:
          'Recebe visão no escuro e +2 em testes de Percepção e Sobrevivência realizados no subterrâneo.',
      },
      {
        id: 'anao-devagar-e-sempre',
        raceId: 'Anão',
        name: 'Devagar e Sempre',
        description:
          'Seu deslocamento é 6m (em vez de 9m). Porém, seu deslocamento não é reduzido por uso de armadura ou excesso de carga.',
      },
      {
        id: 'anao-duro-como-pedra',
        raceId: 'Anão',
        name: 'Duro como Pedra',
        description: 'Você recebe +3 pontos de vida no 1º nível e +1 por nível seguinte.',
      },
      {
        id: 'anao-tradicao-de-heredrimm',
        raceId: 'Anão',
        name: 'Tradição de Heredrimm',
        description:
          'Você é perito nas armas tradicionais anãs. Para você, todos os machados, martelos, marretas e picaretas são armas simples. Recebe +2 em ataques com essas armas.',
      },
    ],
  },
  {
    id: 'Dahllan',
    name: 'Dahllan',
    attributeBonuses: { wisdom: 2, dexterity: 1, intelligence: -1 },
    abilities: [
      {
        id: 'dahllan-amiga-das-plantas',
        raceId: 'Dahllan',
        name: 'Amiga das Plantas',
        description:
          'Você pode lançar a magia Controlar Plantas (atributo-chave Sabedoria). Caso aprenda novamente essa magia, seu custo diminui em –1 PM.',
      },
      {
        id: 'dahllan-armadura-de-allihanna',
        raceId: 'Dahllan',
        name: 'Armadura de Allihanna',
        description:
          'Você pode gastar uma ação de movimento e 1 PM para transformar sua pele em casca de árvore, recebendo +2 na Defesa até o fim da cena.',
      },
      {
        id: 'dahllan-empatia-selvagem',
        raceId: 'Dahllan',
        name: 'Empatia Selvagem',
        description:
          'Você pode se comunicar com animais por meio de linguagem corporal e vocalizações. Pode usar Adestramento para mudar atitude e persuasão com animais. Caso receba esta habilidade novamente, recebe +2 em Adestramento.',
      },
    ],
  },
  {
    id: 'Elfo',
    name: 'Elfo',
    attributeBonuses: { intelligence: 2, dexterity: 1, constitution: -1 },
    abilities: [
      {
        id: 'elfo-graca-de-glorienn',
        raceId: 'Elfo',
        name: 'Graça de Glórienn',
        description: 'Seu deslocamento é 12m (em vez de 9m).',
      },
      {
        id: 'elfo-sangue-magico',
        raceId: 'Elfo',
        name: 'Sangue Mágico',
        description: 'Você recebe +1 ponto de mana por nível.',
      },
      {
        id: 'elfo-sentidos-elficos',
        raceId: 'Elfo',
        name: 'Sentidos Élficos',
        description: 'Você recebe visão na penumbra e +2 em Misticismo e Percepção.',
        modifiers: [
          {
            target: { k: 'expertise', name: 'Misticismo' },
            amount: 2,
            bonusType: 'untyped',
          },
          {
            target: { k: 'expertise', name: 'Percepção' },
            amount: 2,
            bonusType: 'untyped',
          },
        ],
      },
    ],
  },
  {
    id: 'Goblin',
    name: 'Goblin',
    attributeBonuses: { dexterity: 2, intelligence: 1, charisma: -1 },
    abilities: [
      {
        id: 'goblin-engenhoso',
        raceId: 'Goblin',
        name: 'Engenhoso',
        description:
          'Você não sofre penalidades em testes de perícia por não usar ferramentas. Se usar a ferramenta necessária, recebe +2 no teste de perícia.',
      },
      {
        id: 'goblin-espelunqueiro',
        raceId: 'Goblin',
        name: 'Espelunqueiro',
        description:
          'Você recebe visão no escuro e deslocamento de escalada igual ao seu deslocamento terrestre.',
      },
      {
        id: 'goblin-peste-esguia',
        raceId: 'Goblin',
        name: 'Peste Esguia',
        description: 'Seu tamanho é Pequeno, mas seu deslocamento se mantém 9m.',
      },
      {
        id: 'goblin-rato-das-ruas',
        raceId: 'Goblin',
        name: 'Rato das Ruas',
        description:
          'Você recebe +2 em Fortitude e sua recuperação de PV e PM nunca é inferior ao seu nível.',
        modifiers: [
          {
            target: { k: 'expertise', name: 'Fortitude' },
            amount: 2,
            bonusType: 'untyped',
          },
        ],
      },
    ],
  },
  {
    id: 'Lefou',
    name: 'Lefou',
    attributeBonuses: { charisma: -1 },
    abilities: [
      {
        id: 'lefou-tres-atributos',
        raceId: 'Lefou',
        name: '+1 em Três Atributos Diferentes (exceto Carisma)',
        description:
          'Distribua +1 em três atributos diferentes, exceto Carisma (já aplicado na ficha).',
      },
      {
        id: 'lefou-cria-da-tormenta',
        raceId: 'Lefou',
        name: 'Cria da Tormenta',
        description:
          'Você é uma criatura do tipo monstro e recebe +5 em testes de resistência contra efeitos causados por lefeu e pela Tormenta.',
      },
      {
        id: 'lefou-deformidade',
        raceId: 'Lefou',
        name: 'Deformidade',
        description:
          'Recebe +2 em duas perícias a sua escolha (contam como poder da Tormenta, exceto para perda de Carisma). Pode trocar um desses bônus por um poder da Tormenta a sua escolha.',
      },
    ],
  },
  {
    id: 'Minotauro',
    name: 'Minotauro',
    attributeBonuses: { strength: 2, constitution: 1, wisdom: -1 },
    abilities: [
      {
        id: 'minotauro-chifres',
        raceId: 'Minotauro',
        name: 'Chifres',
        description:
          'Arma natural de chifres (dano 1d6, crítico x2, perfuração). Uma vez por rodada, ao usar agredir com outra arma, pode gastar 1 PM para fazer um ataque corpo a corpo extra com os chifres.',
      },
      {
        id: 'minotauro-couro-rigido',
        raceId: 'Minotauro',
        name: 'Couro Rígido',
        description: 'Sua pele é dura como a de um touro. Você recebe +1 na Defesa.',
        modifiers: [
          {
            target: { k: 'defense' },
            amount: 1,
            bonusType: 'untyped',
            note: 'Couro Rígido',
          },
        ],
      },
      {
        id: 'minotauro-faro',
        raceId: 'Minotauro',
        name: 'Faro',
        description:
          'Você tem olfato apurado. Contra inimigos em alcance curto que não possa perceber, não fica desprevenido e camuflagem total lhe causa apenas 20% de chance de falha.',
      },
      {
        id: 'minotauro-medo-de-altura',
        raceId: 'Minotauro',
        name: 'Medo de Altura',
        description:
          'Se estiver adjacente a uma queda de 3m ou mais de altura (como um buraco ou penhasco), você fica abalado.',
      },
    ],
  },
  {
    id: 'Qareen',
    name: 'Qareen',
    attributeBonuses: { charisma: 2, intelligence: 1, wisdom: -1 },
    abilities: [
      {
        id: 'qareen-desejos',
        raceId: 'Qareen',
        name: 'Desejos',
        description:
          'Se lançar uma magia que alguém tenha pedido desde seu último turno, o custo da magia diminui em –1 PM. Fazer um desejo ao qareen é uma ação livre.',
      },
      {
        id: 'qareen-resistencia-elemental',
        raceId: 'Qareen',
        name: 'Resistência Elemental',
        description:
          'Conforme sua ascendência, recebe redução 10 a um tipo de dano. Escolha uma das variantes abaixo.',
        variants: [
          { id: 'qareen-res-frio', name: 'Frio (qareen da água)', description: 'Redução 10 a frio.' },
          { id: 'qareen-res-eletricidade', name: 'Eletricidade (do ar)', description: 'Redução 10 a eletricidade.' },
          { id: 'qareen-res-fogo', name: 'Fogo (do fogo)', description: 'Redução 10 a fogo.' },
          { id: 'qareen-res-acido', name: 'Ácido (da terra)', description: 'Redução 10 a ácido.' },
          { id: 'qareen-res-luz', name: 'Luz (da luz)', description: 'Redução 10 a luz.' },
          { id: 'qareen-res-trevas', name: 'Trevas (qareen das trevas)', description: 'Redução 10 a trevas.' },
        ],
      },
      {
        id: 'qareen-tatuagem-mistica',
        raceId: 'Qareen',
        name: 'Tatuagem Mística',
        description:
          'Você pode lançar uma magia de 1º círculo a sua escolha (atributo-chave Carisma). Caso aprenda novamente essa magia, seu custo diminui em –1 PM.',
      },
    ],
  },
  {
    id: 'Suraggel',
    name: 'Suraggel',
    attributeBonuses: {},
    abilities: [
      {
        id: 'suraggel-heranca',
        raceId: 'Suraggel',
        name: 'Herança Divina',
        description:
          'Escolha sua ascendência: aggelus (celestial) ou sulfure (abissal). A escolha define seus bônus de atributo e habilidades adicionais.',
        variants: [
          {
            id: 'suraggel-aggelus',
            name: 'Aggelus',
            description:
              'Sabedoria +2, Carisma +1. Luz Sagrada: +2 em Diplomacia e Intuição; pode lançar Luz (magia divina, atributo-chave Carisma).',
            modifiers: [
              { target: { k: 'attribute', name: 'wisdom' }, amount: 2, bonusType: 'untyped' },
              { target: { k: 'attribute', name: 'charisma' }, amount: 1, bonusType: 'untyped' },
              { target: { k: 'expertise', name: 'Diplomacia' }, amount: 2, bonusType: 'untyped' },
              { target: { k: 'expertise', name: 'Intuição' }, amount: 2, bonusType: 'untyped' },
            ],
          },
          {
            id: 'suraggel-sulfure',
            name: 'Sulfure',
            description:
              'Destreza +2, Inteligência +1. Sombras Profanas: +2 em Enganação e Furtividade; pode lançar Escuridão (magia divina, atributo-chave Inteligência).',
            modifiers: [
              { target: { k: 'attribute', name: 'dexterity' }, amount: 2, bonusType: 'untyped' },
              { target: { k: 'attribute', name: 'intelligence' }, amount: 1, bonusType: 'untyped' },
              { target: { k: 'expertise', name: 'Enganação' }, amount: 2, bonusType: 'untyped' },
              { target: { k: 'expertise', name: 'Furtividade' }, amount: 2, bonusType: 'untyped' },
            ],
          },
        ],
      },
    ],
  },
  {
    id: 'Sílfide',
    name: 'Sílfide',
    attributeBonuses: { charisma: 2, dexterity: 1, strength: -2 },
    abilities: [
      {
        id: 'silfide-asas-de-borboleta',
        raceId: 'Sílfide',
        name: 'Asas de Borboleta',
        description:
          'Seu tamanho é Minúsculo. Pode pairar a 1,5m do chão com deslocamento 9m, ignorando terreno difícil e imune a dano por queda. Gastando 1 PM por rodada, voa com deslocamento 12m.',
      },
      {
        id: 'silfide-espirito-da-natureza',
        raceId: 'Sílfide',
        name: 'Espírito da Natureza',
        description:
          'Você é uma criatura do tipo espírito, recebe visão na penumbra e pode falar com animais livremente.',
      },
      {
        id: 'silfide-magia-das-fadas',
        raceId: 'Sílfide',
        name: 'Magia das Fadas',
        description:
          'Pode lançar duas das magias a seguir (atributo-chave Carisma): Criar Ilusão, Enfeitiçar, Luz (como magia arcana) e Sono. Caso aprenda novamente uma dessas magias, seu custo diminui em –1 PM.',
      },
    ],
  },
  {
    id: 'Sereia/Tritão',
    name: 'Sereia/Tritão',
    attributeBonuses: {},
    abilities: [
      {
        id: 'sereia-tres-atributos',
        raceId: 'Sereia/Tritão',
        name: '+1 em Três Atributos Diferentes',
        description: 'Distribua +1 em três atributos diferentes (já aplicado na ficha).',
      },
      {
        id: 'sereia-cancao-dos-mares',
        raceId: 'Sereia/Tritão',
        name: 'Canção dos Mares',
        description:
          'Pode lançar duas das magias a seguir: Amedrontar, Comando, Despedaçar, Enfeitiçar, Hipnotismo ou Sono (atributo-chave Carisma). Caso aprenda novamente uma dessas magias, seu custo diminui em –1 PM.',
      },
      {
        id: 'sereia-mestre-do-tridente',
        raceId: 'Sereia/Tritão',
        name: 'Mestre do Tridente',
        description:
          'Para você, o tridente é uma arma simples. Você recebe +2 em rolagens de dano com azagaias, lanças e tridentes.',
      },
      {
        id: 'sereia-transformacao-anfibia',
        raceId: 'Sereia/Tritão',
        name: 'Transformação Anfíbia',
        description:
          'Pode respirar debaixo d’água e possui cauda com deslocamento de natação 12m. Fora d’água, a cauda dá lugar a pernas (deslocamento 9m). Mais de um dia sem contato com água impede recuperação de PM por descanso.',
      },
    ],
  },
  {
    id: 'Osteon',
    name: 'Osteon',
    attributeBonuses: { constitution: -1 },
    abilities: [
      {
        id: 'osteon-tres-atributos',
        raceId: 'Osteon',
        name: '+1 em Três Atributos Diferentes (exceto Constituição)',
        description:
          'Distribua +1 em três atributos diferentes, exceto Constituição (já aplicado na ficha).',
      },
      {
        id: 'osteon-armadura-ossea',
        raceId: 'Osteon',
        name: 'Armadura Óssea',
        description: 'Você recebe redução de corte, frio e perfuração 5.',
      },
      {
        id: 'osteon-memoria-postuma',
        raceId: 'Osteon',
        name: 'Memória Póstuma',
        description:
          'Você se torna treinado em uma perícia (não precisa ser da sua classe) ou recebe um poder geral a sua escolha. Alternativamente, escolha uma habilidade de outra raça humanoide (que não humano); herda também o tamanho dessa raça se diferente de Médio.',
      },
      {
        id: 'osteon-natureza-esqueletica',
        raceId: 'Osteon',
        name: 'Natureza Esquelética',
        description:
          'Criatura do tipo morto-vivo. Visão no escuro e imunidade a cansaço, metabólicos, trevas e veneno. Não precisa respirar, comer ou dormir. Efeitos mágicos de cura de luz causam dano; dano de trevas recupera PV.',
      },
      {
        id: 'osteon-preco-da-nao-vida',
        raceId: 'Osteon',
        name: 'Preço da Não Vida',
        description:
          'Precisa passar oito horas sob luz de estrelas ou no subterrâneo. Se fizer isso, recupera PV e PM por descanso normalmente. Caso contrário, sofre efeitos de fome.',
      },
    ],
  },
  {
    id: 'Medusa',
    name: 'Medusa',
    attributeBonuses: { dexterity: 2, charisma: 1 },
    abilities: [
      {
        id: 'medusa-cria-de-megalokk',
        raceId: 'Medusa',
        name: 'Cria de Megalokk',
        description: 'Você é uma criatura do tipo monstro e recebe visão no escuro.',
      },
      {
        id: 'medusa-natureza-venenosa',
        raceId: 'Medusa',
        name: 'Natureza Venenosa',
        description:
          'Recebe resistência a veneno +5 e pode gastar uma ação de movimento e 1 PM para envenenar uma arma. A arma causa perda de 1d12 PV adicional. Veneno dura até acertar um ataque ou fim da cena.',
      },
      {
        id: 'medusa-olhar-atordoante',
        raceId: 'Medusa',
        name: 'Olhar Atordoante',
        description:
          'Pode gastar uma ação de movimento e 1 PM para forçar uma criatura em alcance curto a fazer Fortitude (CD Car). Se falhar, fica atordoada por uma rodada (apenas uma vez por cena).',
      },
    ],
  },
  {
    id: 'Kiiren',
    name: 'Kiiren',
    attributeBonuses: { intelligence: 2, charisma: 1, strength: -1 },
    abilities: [
      {
        id: 'kiiren-hibrido',
        raceId: 'Kiiren',
        name: 'Híbrido',
        description:
          'Sua natureza multifacetada o fez aprender conhecimentos variados. Você se torna treinado em uma perícia a sua escolha (não precisa ser da sua classe).',
      },
      {
        id: 'kiiren-engenhosidade',
        raceId: 'Kiiren',
        name: 'Engenhosidade',
        description:
          'Quando faz um teste de perícia, pode gastar 2 PM para somar sua Inteligência no teste. Não pode usar em testes de ataque. Caso receba esta habilidade novamente, seu custo é reduzido em –1 PM.',
      },
      {
        id: 'kiiren-ossos-frageis',
        raceId: 'Kiiren',
        name: 'Ossos Frágeis',
        description:
          'Você sofre 1 ponto de dano adicional por dado de dano de impacto.',
      },
      {
        id: 'kiiren-vanguardista',
        raceId: 'Kiiren',
        name: 'Vanguardista',
        description: 'Recebe proficiência em armas de fogo e +2 em Ofício (um qualquer, a sua escolha).',
      },
    ],
  },
  {
    id: 'Hynne',
    name: 'Hynne',
    attributeBonuses: { dexterity: 2, charisma: 1, strength: -1 },
    abilities: [
      {
        id: 'hynne-arremessador',
        raceId: 'Hynne',
        name: 'Arremessador',
        description:
          'Quando faz um ataque à distância com uma funda ou uma arma de arremesso, seu dano aumenta em um passo.',
      },
      {
        id: 'hynne-pequeno-e-rechonchudo',
        raceId: 'Hynne',
        name: 'Pequeno e Rechonchudo',
        description:
          'Seu tamanho é Pequeno e seu deslocamento é 6m. Recebe +2 em Enganação e pode usar Destreza como atributo-chave de Atletismo (em vez de Força).',
        modifiers: [
          {
            target: { k: 'expertise', name: 'Enganação' },
            amount: 2,
            bonusType: 'untyped',
          },
        ],
      },
      {
        id: 'hynne-sorte-salvadora',
        raceId: 'Hynne',
        name: 'Sorte Salvadora',
        description:
          'Quando faz um teste de resistência, você pode gastar 1 PM para rolar este teste novamente.',
      },
    ],
  },
  {
    id: 'Golem',
    name: 'Golem',
    attributeBonuses: { strength: 2, constitution: 1, charisma: -1 },
    abilities: [
      {
        id: 'golem-chassi',
        raceId: 'Golem',
        name: 'Chassi',
        description:
          'Seu corpo artificial é resistente, mas rígido. Deslocamento 6m, não reduzido por armadura ou excesso de carga. Recebe +2 na Defesa, mas possui penalidade de armadura –2. Sua armadura precisa de um dia para vestir/remover e não conta no limite de itens em uso.',
        modifiers: [
          {
            target: { k: 'defense' },
            amount: 2,
            bonusType: 'untyped',
            note: 'Chassi',
          },
          {
            target: { k: 'armorPenalty' },
            amount: -2,
            bonusType: 'untyped',
            note: 'Chassi',
          },
        ],
      },
      {
        id: 'golem-criatura-artificial',
        raceId: 'Golem',
        name: 'Criatura Artificial',
        description:
          'Você é uma criatura do tipo construto. Visão no escuro e imunidade a cansaço, metabólicos e veneno. Não respira, alimenta-se ou dorme, mas não se beneficia de cura mundana ou itens de alimentação. Precisa ficar inerte 8h por dia para recarregar (descansa em condições normais sempre).',
      },
      {
        id: 'golem-fonte-elemental',
        raceId: 'Golem',
        name: 'Fonte Elemental',
        description:
          'Você possui um espírito elemental preso em seu corpo. Imune ao tipo de dano associado; se sofreria dano mágico desse tipo, em vez disso cura PV em metade do dano. Escolha sua origem elemental:',
        variants: [
          { id: 'golem-fonte-agua', name: 'Água (frio)', description: 'Imune a frio.' },
          { id: 'golem-fonte-ar', name: 'Ar (eletricidade)', description: 'Imune a eletricidade.' },
          { id: 'golem-fonte-fogo', name: 'Fogo (fogo)', description: 'Imune a fogo.' },
          { id: 'golem-fonte-terra', name: 'Terra (ácido)', description: 'Imune a ácido.' },
        ],
      },
      {
        id: 'golem-proposito-de-criacao',
        raceId: 'Golem',
        name: 'Propósito de Criação',
        description:
          'Você foi construído pronto para um propósito específico e não teve infância. Não tem direito a escolher uma origem, mas recebe um poder geral a sua escolha.',
      },
    ],
  },
  {
    id: 'Trog',
    name: 'Trog',
    attributeBonuses: { constitution: 2, strength: 1, intelligence: -1 },
    abilities: [
      {
        id: 'trog-mau-cheiro',
        raceId: 'Trog',
        name: 'Mau Cheiro',
        description:
          'Pode gastar uma ação padrão e 2 PM para expelir um gás fétido. Criaturas (exceto trogs) em alcance curto devem passar em Fortitude contra veneno (CD Con) ou ficam enjoadas por 1d6 rodadas. Quem passar fica imune por um dia.',
      },
      {
        id: 'trog-mordida',
        raceId: 'Trog',
        name: 'Mordida',
        description:
          'Arma natural de mordida (dano 1d6, crítico x2, perfuração). Uma vez por rodada, ao usar agredir com outra arma, pode gastar 1 PM para fazer ataque corpo a corpo extra com a mordida.',
      },
      {
        id: 'trog-reptiliano',
        raceId: 'Trog',
        name: 'Reptiliano',
        description:
          'Criatura do tipo monstro, visão no escuro, +1 na Defesa e, se estiver sem armadura ou roupas pesadas, +5 em Furtividade.',
        modifiers: [
          {
            target: { k: 'defense' },
            amount: 1,
            bonusType: 'untyped',
            note: 'Reptiliano',
          },
        ],
      },
      {
        id: 'trog-sangue-frio',
        raceId: 'Trog',
        name: 'Sangue Frio',
        description: 'Você sofre 1 ponto de dano adicional por dado de dano de frio.',
      },
    ],
  },
]
