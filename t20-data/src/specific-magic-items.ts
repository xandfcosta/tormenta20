/**
 * Itens Mágicos Específicos — Armas + Armaduras + Escudos (PDF Cap 6
 * — Tesouro):
 *
 *  - Tabela 8-9 (armas, p337; descriptions p336-338) — 18 entries.
 *  - Tabela 8-11 (armaduras + escudos, p340; descriptions p340) — 13
 *    entries.
 *
 * These are *named* standalone items, distinct from generic encantos
 * applied to a base item. Per PDF p334: "Todas as armas e armaduras
 * específicas deste livro são itens maiores." Every entry is tier
 * `maior`, with priceTibar 30.000-200.000 T$.
 *
 * `requiresClass` captures both hard restrictions (Vingadora Sagrada
 * only powers up for a paladino) and conditional bonus scaling
 * (Espada Baronial scales with código de conduta / devoção a Khalmyr
 * / treinado em Nobreza). Tabela 8-11 entries have no hard class
 * locks — conditional bonuses (devoto/código de conduta/uso do poder
 * Comandar) live in the `effect` text.
 *
 * No attunement system in T20 — every `requiresAttunement: false`.
 */
export type SpecificItemCategory =
  | 'arma'
  | 'armadura'
  | 'escudo'
  | 'anel'
  | 'maravilhoso'
  | 'varinha'
  | 'pocao'
  | 'pergaminho'
  | 'haste'
  | 'outro'

export type SpecificItemTier = 'menor' | 'medio' | 'maior' | 'artefato'

export type SpecificMagicItem = {
  id: string
  name: string
  category: SpecificItemCategory
  tier: SpecificItemTier
  priceTibar: number
  baseItem: string | null
  effect: string
  requiresClass: string | null
  requiresAttunement: false
  bookPage: number
}

export const SPECIFIC_MAGIC_ITEMS: readonly SpecificMagicItem[] = Object.freeze([
  {
    id: 'azagaia-dos-relampagos',
    name: 'Azagaia dos Relâmpagos',
    category: 'arma',
    tier: 'maior',
    priceTibar: 30000,
    baseItem: 'Azagaia',
    effect:
      'Quando arremessada, esta azagaia se transforma em um Relâmpago (8d6 de dano de eletricidade numa linha com alcance médio; CD For ou Des à sua escolha). Quando atinge o fim do alcance ela volta a ser uma azagaia e volta para você no fim do turno.',
    requiresClass: null,
    requiresAttunement: false,
    bookPage: 337,
  },
  {
    id: 'espada-baronial',
    name: 'Espada Baronial',
    category: 'arma',
    tier: 'maior',
    priceTibar: 30000,
    baseItem: 'Espada Longa',
    effect:
      'Esta espada longa de guarda reta fornece +1 em testes de ataque e rolagens de dano. Este bônus aumenta em +1 se você possuir um código de conduta (de honra, do herói...), for devoto de Khalmyr ou for treinado em Nobreza. Os bônus são cumulativos — um personagem com código de conduta, devoto de Khalmyr e treinado em Nobreza recebe +4 em ataque e dano.',
    requiresClass: null,
    requiresAttunement: false,
    bookPage: 337,
  },
  {
    id: 'lamina-da-luz',
    name: 'Lâmina da Luz',
    category: 'arma',
    tier: 'maior',
    priceTibar: 45000,
    baseItem: 'Espada Bastarda',
    effect:
      'De lâmina prateada e reluzente, esta espada bastarda formidável é concedida a cavaleiros da Luz de honra e virtude comprovadas. Pode gastar uma ação de movimento e 2 PM para erguer a lâmina acima da cabeça; se fizer isso, ela irradia luz brilhante em alcance médio até o fim da cena. Todos os inimigos dentro da luz ficam ofuscados.',
    requiresClass: 'Cavaleiro da Luz (concedida)',
    requiresAttunement: false,
    bookPage: 338,
  },
  {
    id: 'lanca-animalesca',
    name: 'Lança Animalesca',
    category: 'arma',
    tier: 'maior',
    priceTibar: 45000,
    baseItem: 'Lança',
    effect:
      'Espinhos e folhas vivas brotam desta lança formidável. Se você usar a habilidade de Forma Selvagem, aplica o bônus de +2 em ataque e dano da lança formidável em suas armas naturais.',
    requiresClass: 'Druida (sinergia com Forma Selvagem)',
    requiresAttunement: false,
    bookPage: 338,
  },
  {
    id: 'maca-do-terror',
    name: 'Maça do Terror',
    category: 'arma',
    tier: 'maior',
    priceTibar: 45000,
    baseItem: 'Maça',
    effect:
      'Esta maça formidável é feita com um osso e um crânio e permite que você lance a magia Amedrontar (CD For ou Car à sua escolha). Caso já conheça a magia, o custo para lançá-la diminui em -1 PM.',
    requiresClass: null,
    requiresAttunement: false,
    bookPage: 338,
  },
  {
    id: 'florete-fugaz',
    name: 'Florete Fugaz',
    category: 'arma',
    tier: 'maior',
    priceTibar: 50000,
    baseItem: 'Florete',
    effect:
      'Este florete formidável tem o cabo e a guarda trabalhados com prata e pedrarias. Quando usa a ação agredir, pode gastar 1 PM; se acertar um crítico no turno, pode fazer um ataque adicional contra a mesma criatura.',
    requiresClass: null,
    requiresAttunement: false,
    bookPage: 337,
  },
  {
    id: 'cajado-da-destruicao',
    name: 'Cajado da Destruição',
    category: 'arma',
    tier: 'maior',
    priceTibar: 60000,
    baseItem: 'Bordão',
    effect:
      'Este bordão formidável escuro e reforçado com ponteiras de metal é procurado por conjuradores de batalha. Conta como um cajado arcano. Além dos benefícios desse esotérico, quando você lança uma magia de dano, ela causa +1 ponto de dano por dado.',
    requiresClass: 'Conjurador arcano (uso pleno)',
    requiresAttunement: false,
    bookPage: 337,
  },
  {
    id: 'cajado-da-vida',
    name: 'Cajado da Vida',
    category: 'arma',
    tier: 'maior',
    priceTibar: 60000,
    baseItem: 'Bordão',
    effect:
      'Este bordão formidável branco com runas prateadas é valorizado por curandeiros. Conta como um cajado arcano, mas afeta magias divinas. Quando lança uma magia de cura, ela cura +2 pontos de vida por dado.',
    requiresClass: 'Conjurador divino (uso pleno)',
    requiresAttunement: false,
    bookPage: 337,
  },
  {
    id: 'machado-silvestre',
    name: 'Machado Silvestre',
    category: 'arma',
    tier: 'maior',
    priceTibar: 70000,
    baseItem: 'Machado de Batalha',
    effect:
      'O cabo e a lâmina deste machado de batalha formidável são cobertos de gravuras representando plantas e animais selvagens. Quando usado em ambiente ermo e ao ar livre, causa +1d8 de dano e concede o poder Trespassar. Caso já possua este poder, pode utilizá-lo sem gastar PM.',
    requiresClass: null,
    requiresAttunement: false,
    bookPage: 338,
  },
  {
    id: 'martelo-de-doherimm',
    name: 'Martelo de Doherimm',
    category: 'arma',
    tier: 'maior',
    priceTibar: 70000,
    baseItem: 'Martelo de Guerra',
    effect:
      'Este martelo de guerra formidável é feito de pedra e aço. Quando empunhado por um anão, adquire o encanto arremesso e aumenta seu dano em +1d8 (ou +2d8 se usado contra criaturas Grandes ou maiores).',
    requiresClass: 'Anão (benefícios completos)',
    requiresAttunement: false,
    bookPage: 338,
  },
  {
    id: 'arco-do-poder',
    name: 'Arco do Poder',
    category: 'arma',
    tier: 'maior',
    priceTibar: 90000,
    baseItem: 'Arco Longo',
    effect:
      'O arco do poder conta como um arco longo formidável, mas parece apenas o corpo de um arco — sem corda nem flechas. Ao empunhar e fazer o gesto de puxar a corda inexistente, o arco cria uma corda e uma flecha de energia dourada. Lê suas intenções e produz tipos diferentes a sua escolha: Flecha Normal (3d8 essência), Flecha Piedosa (4d8 essência não letal), Flecha Explosiva (3d6 fogo em área 6m, Reflexos CD Des reduz à metade) e Flecha-Rede (sem dano, prende em rede de energia, Força/Acrobacia CD 25 para escapar).',
    requiresClass: null,
    requiresAttunement: false,
    bookPage: 336,
  },
  {
    id: 'lingua-do-deserto',
    name: 'Língua do Deserto',
    category: 'arma',
    tier: 'maior',
    priceTibar: 90000,
    baseItem: 'Cimitarra',
    effect:
      'Esta cimitarra formidável é originária do Deserto da Perdição. Pode gastar uma ação de movimento e 1 PM para transformar a lâmina em chamas até o fim da cena; o dano da arma aumenta em um passo e passa a ser fogo. Pode gastar uma ação de movimento e 2 PM para fazer as chamas brilharem com muita força, deixando inimigos em alcance curto desprevenidos por uma rodada.',
    requiresClass: null,
    requiresAttunement: false,
    bookPage: 338,
  },
  {
    id: 'besta-explosiva',
    name: 'Besta Explosiva',
    category: 'arma',
    tier: 'maior',
    priceTibar: 100000,
    baseItem: 'Besta Pesada',
    effect:
      'Esta besta pesada formidável é feita de madeira escurecida, similar a carvão. Pode gastar 3 PM para transformar o virote em uma Bola de Fogo, mirada em criatura ou ponto em alcance médio. Contra criatura: ataque normal; se acertar, dano do disparo + 6d6 fogo, mais 6d6 fogo em criaturas a até 6m (Ref CD Des reduz à metade); se errar, o virote se desfaz em cinzas. Contra ponto: funciona como a magia Bola de Fogo.',
    requiresClass: null,
    requiresAttunement: false,
    bookPage: 337,
  },
  {
    id: 'punhal-sszzaazita',
    name: 'Punhal Sszzaazita',
    category: 'arma',
    tier: 'maior',
    priceTibar: 100000,
    baseItem: 'Adaga',
    effect:
      'Esta adaga assassina formidável venenosa tem lâmina negra e ondulada. Pode gastar uma ação de movimento e 2 PM para transformar o punhal em um objeto inofensivo de tamanho similar (colher, pena...). Nenhuma magia detecta a transformação. Voltar à forma de arma é ação livre.',
    requiresClass: null,
    requiresAttunement: false,
    bookPage: 338,
  },
  {
    id: 'espada-sortuda',
    name: 'Espada Sortuda',
    category: 'arma',
    tier: 'maior',
    priceTibar: 110000,
    baseItem: 'Espada Curta',
    effect:
      'Esta espada curta formidável é cravejada de brilhantes. Você recebe +2 em testes de resistência e, ao fazer um teste, pode gastar 3 PM para rolá-lo novamente. Se possuir o poder Sortudo, em vez disso o custo diminui em -1 PM.',
    requiresClass: null,
    requiresAttunement: false,
    bookPage: 337,
  },
  {
    id: 'avalanche',
    name: 'Avalanche',
    category: 'arma',
    tier: 'maior',
    priceTibar: 140000,
    baseItem: 'Martelo de Guerra',
    effect:
      'Este martelo de guerra de gelo eterno congelante formidável fornece redução de dano 10. Pode gastar uma ação padrão e 6 PM para brandi-lo e invocar tempestade de gelo em alcance curto. Criaturas na área recebem camuflagem leve e sofrem 3d6 de impacto + 3d6 de frio por rodada. Você não sofre os efeitos nocivos e pode gastar 1 PM no início de cada turno para mantê-la.',
    requiresClass: null,
    requiresAttunement: false,
    bookPage: 337,
  },
  {
    id: 'cajado-do-poder',
    name: 'Cajado do Poder',
    category: 'arma',
    tier: 'maior',
    priceTibar: 180000,
    baseItem: 'Bordão',
    effect:
      'Este bordão defensor magnífico tem cabo reto e liso, com uma joia cintilante na ponta. Conta como cajado arcano. Além dos benefícios, o custo das suas magias arcanas diminui em -1 PM (cumulativo com Mestre em Escola) e a CD para resistir a elas aumenta em +2 (aumento total de +3).',
    requiresClass: 'Conjurador arcano (uso pleno)',
    requiresAttunement: false,
    bookPage: 337,
  },
  {
    id: 'vingadora-sagrada',
    name: 'Vingadora Sagrada',
    category: 'arma',
    tier: 'maior',
    priceTibar: 200000,
    baseItem: 'Espada Longa',
    effect:
      'Esta espada longa formidável revela todo o seu poder apenas quando empunhada por um paladino. Se você for paladino, recebe +5 em testes de ataque e rolagens de dano, o custo de seu Golpe Divino é reduzido em -1 PM, e você e seus aliados em alcance curto recebem resistência a magia +5.',
    requiresClass: 'Paladino',
    requiresAttunement: false,
    bookPage: 338,
  },

  // ─── Tabela 8-11: Armaduras + Escudos Específicos (p340) ─────────
  {
    id: 'cota-elfica',
    name: 'Cota Élfica',
    category: 'armadura',
    tier: 'maior',
    priceTibar: 30000,
    baseItem: 'Cota de Malha',
    effect:
      'Composta de anéis finíssimos, esta cota de malha defensora de mitral parece ser feita de seda. Permite aplicar sua Destreza na Defesa como se fosse uma armadura leve.',
    requiresClass: null,
    requiresAttunement: false,
    bookPage: 340,
  },
  {
    id: 'couro-de-monstro',
    name: 'Couro de Monstro',
    category: 'armadura',
    tier: 'maior',
    priceTibar: 36000,
    baseItem: 'Gibão de Peles',
    effect:
      'Usado por chefes bárbaros das Montanhas Sanguinárias, este gibão de peles defensor é feito do couro de monstros (basiliscos, serpes). Se usar o poder Ataque Poderoso ou fizer uma investida, recebe +2d6 nas rolagens de dano.',
    requiresClass: null,
    requiresAttunement: false,
    bookPage: 340,
  },
  {
    id: 'escudo-do-conjurador',
    name: 'Escudo do Conjurador',
    category: 'escudo',
    tier: 'maior',
    priceTibar: 45000,
    baseItem: 'Escudo Leve',
    effect:
      'Este escudo leve defensor tem uma pequena tira de couro na parte interna, sobre a qual um conjurador pode lançar uma magia. A magia não surte efeito na hora; fica inscrita na tira. A tira pode então ser lida como pergaminho, descarregando a magia em seus alvos/área. Uma vez descarregada, outra pode ser armazenada.',
    requiresClass: null,
    requiresAttunement: false,
    bookPage: 340,
  },
  {
    id: 'loriga-do-centuriao',
    name: 'Loriga do Centurião',
    category: 'armadura',
    tier: 'maior',
    priceTibar: 45000,
    baseItem: 'Loriga Segmentada',
    effect:
      'Esta loriga segmentada defensora é dourada com detalhes em vermelho e tem o símbolo de Tauron gravado no peitoral. Se estiver liderando uma ou mais criaturas (usando Comandar ou similar), seus ataques corpo a corpo causam +2d6 de fogo.',
    requiresClass: null,
    requiresAttunement: false,
    bookPage: 340,
  },
  {
    id: 'manto-da-noite',
    name: 'Manto da Noite',
    category: 'armadura',
    tier: 'maior',
    priceTibar: 45000,
    baseItem: 'Couro Batido',
    effect:
      'Este couro batido ajustado defensor sombrio é negro com partes metálicas foscas. Não sofre penalidade em testes de Furtividade por se mover em seu deslocamento normal e a penalidade por atacar diminui para -10.',
    requiresClass: null,
    requiresAttunement: false,
    bookPage: 340,
  },
  {
    id: 'couraca-do-comando',
    name: 'Couraça do Comando',
    category: 'armadura',
    tier: 'maior',
    priceTibar: 45000,
    baseItem: 'Couraça',
    effect:
      'Esta couraça banhada a ouro sob medida defensora irradia uma aura de autoridade. Você recebe +1 em Carisma. Se usar o poder Comandar, o bônus fornecido aumenta para +2.',
    requiresClass: null,
    requiresAttunement: false,
    bookPage: 340,
  },
  {
    id: 'baluarte-anao',
    name: 'Baluarte Anão',
    category: 'armadura',
    tier: 'maior',
    priceTibar: 50000,
    baseItem: 'Armadura Completa',
    effect:
      'Esta armadura completa reforçada defensora de adamante fornece proteção sem igual. Se não se deslocar em seu turno, a RD que ela fornece aumenta para 10 até seu próximo turno.',
    requiresClass: null,
    requiresAttunement: false,
    bookPage: 340,
  },
  {
    id: 'escudo-espinhoso',
    name: 'Escudo Espinhoso',
    category: 'escudo',
    tier: 'maior',
    priceTibar: 50000,
    baseItem: 'Escudo Pesado',
    effect:
      'Este escudo pesado defensor é coberto de espinhos. Pode gastar uma ação de movimento e 2 PM para disparar um espinho em alvo em alcance curto. O espinho acerta automaticamente e causa 1d10+2 pontos de dano de perfuração.',
    requiresClass: null,
    requiresAttunement: false,
    bookPage: 340,
  },
  {
    id: 'escudo-do-leao',
    name: 'Escudo do Leão',
    category: 'escudo',
    tier: 'maior',
    priceTibar: 50000,
    baseItem: 'Escudo Pesado',
    effect:
      'Este escudo pesado defensor é forjado como cabeça de leão rugindo. Uma vez por rodada, pode gastar 2 PM para fazer a cabeça criar vida e morder criatura adjacente. A mordida acerta automaticamente e causa 2d6+2 pontos de dano de perfuração.',
    requiresClass: null,
    requiresAttunement: false,
    bookPage: 340,
  },
  {
    id: 'carapaca-demoniaca',
    name: 'Carapaça Demoníaca',
    category: 'armadura',
    tier: 'maior',
    priceTibar: 63000,
    baseItem: 'Armadura Completa',
    effect:
      'Esta armadura completa macabra reforçada guardiã é forjada para fazer o usuário parecer um demônio — elmo com chifres, visão pela boca dentada. Se devoto de divindade que canaliza apenas energia negativa, seus ataques corpo a corpo causam +1d8 de dano de trevas.',
    requiresClass: null,
    requiresAttunement: false,
    bookPage: 340,
  },
  {
    id: 'escudo-do-eclipse',
    name: 'Escudo do Eclipse',
    category: 'escudo',
    tier: 'maior',
    priceTibar: 70000,
    baseItem: 'Escudo Pesado',
    effect:
      'Este escudo pesado defensor é completamente negro e parece absorver a luz. Fornece redução de trevas 10 e causa +1d8 de dano de trevas num ataque. Pode gastar uma ação de movimento e 2 PM para lançar Escuridão.',
    requiresClass: null,
    requiresAttunement: false,
    bookPage: 340,
  },
  {
    id: 'escudo-de-azgher',
    name: 'Escudo de Azgher',
    category: 'escudo',
    tier: 'maior',
    priceTibar: 140000,
    baseItem: 'Escudo Pesado',
    effect:
      'Este escudo pesado guardião é forjado na forma de sol estilizado. Pode gastar uma ação padrão e 10 PM para emitir luz brilhante e quente num cone de alcance curto. A luz gera os efeitos de Visão da Verdade e causa 6d6 de fogo em todos os inimigos (mortos-vivos e criaturas vulneráveis a luz solar sofrem 6d8). Pode gastar 1 PM no início de cada turno para mantê-la.',
    requiresClass: null,
    requiresAttunement: false,
    bookPage: 340,
  },
  {
    id: 'armadura-da-luz',
    name: 'Armadura da Luz',
    category: 'armadura',
    tier: 'maior',
    priceTibar: 150000,
    baseItem: 'Armadura Completa',
    effect:
      'Esta armadura completa banhada a ouro reforçada guardiã zelosa tem o símbolo de Khalmyr gravado no peitoral. Se possuir código de conduta (de honra, do herói...) ou for devoto de divindade que canaliza apenas energia positiva, recebe redução de dano igual ao seu Carisma.',
    requiresClass: null,
    requiresAttunement: false,
    bookPage: 340,
  },
])

const byId = new Map(SPECIFIC_MAGIC_ITEMS.map((i) => [i.id, i]))

export function specificItemById(
  id: string,
): SpecificMagicItem | undefined {
  return byId.get(id)
}

export function specificItemsByCategory(
  category: SpecificItemCategory,
): readonly SpecificMagicItem[] {
  return SPECIFIC_MAGIC_ITEMS.filter((i) => i.category === category)
}

export function specificItemsByTier(
  tier: SpecificItemTier,
): readonly SpecificMagicItem[] {
  return SPECIFIC_MAGIC_ITEMS.filter((i) => i.tier === tier)
}

/** Items with a hard or conditional class restriction. */
export function classRestrictedItems(): readonly SpecificMagicItem[] {
  return SPECIFIC_MAGIC_ITEMS.filter((i) => i.requiresClass !== null)
}
