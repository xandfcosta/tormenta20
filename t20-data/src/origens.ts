/**
 * Origens — PDF book p85-95 (Tabela 1-19 on p87).
 *
 * Cada origem expõe:
 *  - Uma lista fixa de perícias e uma lista fixa de poderes.
 *  - Um **poder único** (sempre o último da lista de poderes) exclusivo
 *    de quem tem essa origem.
 *  - Um conjunto de itens iniciais (concedidos automaticamente, sem
 *    contar como benefício).
 *
 * O jogador escolhe **2 benefícios** quaisquer combinando perícias e
 * poderes da listagem. Os itens iniciais são sempre concedidos.
 *
 * Quirks:
 *  - Algumas origens listam um slot adicional "+ um poder de combate /
 *    Tormenta a sua escolha". Modelado como `poderChoiceCategory`.
 *  - Amnésico não tem listas fixas: as escolhas são feitas pelo mestre
 *    e o poder Lembranças Graduais é obrigatório (marcado com
 *    `gmDriven: true`).
 *  - Soldado / Fazendeiro / Membro de Guilda têm 4 perícias fixas, mas
 *    o jogador ainda escolhe apenas 2 benefícios.
 */

export const ORIGEM_BENEFICIOS_PER_CHARACTER = 2

export type OrigemPoderChoiceCategory = 'combate' | 'tormenta'

export type Origem = {
  id: string
  name: string
  pericias: readonly string[]
  poderes: readonly string[]
  /** Always === poderes[poderes.length - 1] for non-GM-driven origens. */
  poderUnico: string
  poderChoiceCategory?: OrigemPoderChoiceCategory
  itensIniciais: readonly string[]
  bookPage: number
  /** Set on Amnésico — perícias / poderes are picked by the GM. */
  gmDriven?: boolean
  /** Always-applied power (Amnésico's Lembranças Graduais). */
  poderObrigatorio?: string
}

const ORIGENS_LIST: readonly Origem[] = [
  {
    id: 'acolito',
    name: 'Acólito',
    pericias: ['Cura', 'Religião', 'Vontade'],
    poderes: ['Medicina', 'Membro da Igreja', 'Vontade de Ferro'],
    poderUnico: 'Membro da Igreja',
    itensIniciais: ['Símbolo sagrado', 'Traje de sacerdote'],
    bookPage: 85,
  },
  {
    id: 'amigo-dos-animais',
    name: 'Amigo dos Animais',
    pericias: ['Adestramento', 'Cavalgar'],
    poderes: ['Amigo Especial'],
    poderUnico: 'Amigo Especial',
    itensIniciais: ['Cão de caça, cavalo, pônei ou trobo (escolha)'],
    bookPage: 85,
  },
  {
    id: 'amnesico',
    name: 'Amnésico',
    pericias: [],
    poderes: ['Lembranças Graduais'],
    poderUnico: 'Lembranças Graduais',
    poderObrigatorio: 'Lembranças Graduais',
    gmDriven: true,
    itensIniciais: ['Itens variados (até T$ 500, aprovados pelo mestre)'],
    bookPage: 86,
  },
  {
    id: 'aristocrata',
    name: 'Aristocrata',
    pericias: ['Diplomacia', 'Enganação', 'Nobreza'],
    poderes: ['Comandar', 'Sangue Azul'],
    poderUnico: 'Sangue Azul',
    itensIniciais: ['Joia de família (T$ 300)', 'Traje da corte'],
    bookPage: 86,
  },
  {
    id: 'artesao',
    name: 'Artesão',
    pericias: ['Ofício', 'Vontade'],
    poderes: ['Frutos do Trabalho', 'Sortudo'],
    poderUnico: 'Frutos do Trabalho',
    itensIniciais: ['Instrumentos de ofício', 'Um item fabricado (até T$ 50)'],
    bookPage: 86,
  },
  {
    id: 'artista',
    name: 'Artista',
    pericias: ['Atuação', 'Enganação'],
    poderes: ['Atraente', 'Dom Artístico', 'Sortudo', 'Torcida'],
    poderUnico: 'Dom Artístico',
    itensIniciais: ['Estojo de disfarces OU instrumento musical'],
    bookPage: 87,
  },
  {
    id: 'assistente-de-laboratorio',
    name: 'Assistente de Laboratório',
    pericias: ['Ofício (alquimista)', 'Misticismo'],
    poderes: ['Esse Cheiro...', 'Venefício'],
    poderUnico: 'Esse Cheiro...',
    poderChoiceCategory: 'tormenta',
    itensIniciais: ['Instrumentos de Ofício (alquimista)'],
    bookPage: 87,
  },
  {
    id: 'batedor',
    name: 'Batedor',
    pericias: ['Furtividade', 'Percepção', 'Sobrevivência'],
    poderes: ['À Prova de Tudo', 'Estilo de Disparo', 'Sentidos Aguçados'],
    poderUnico: 'À Prova de Tudo',
    itensIniciais: [
      'Barraca',
      'Equipamento de viagem',
      'Arma simples ou marcial de ataque à distância',
    ],
    bookPage: 88,
  },
  {
    id: 'capanga',
    name: 'Capanga',
    pericias: ['Luta', 'Intimidação'],
    poderes: ['Confissão'],
    poderUnico: 'Confissão',
    poderChoiceCategory: 'combate',
    itensIniciais: [
      'Tatuagem/adereço de gangue (+1 em Intimidação)',
      'Arma simples corpo a corpo',
    ],
    bookPage: 88,
  },
  {
    id: 'charlatao',
    name: 'Charlatão',
    pericias: ['Enganação', 'Jogatina'],
    poderes: ['Alpinista Social', 'Aparência Inofensiva', 'Sortudo'],
    poderUnico: 'Alpinista Social',
    itensIniciais: [
      'Estojo de disfarces',
      'Joia falsificada (T$ 100 aparente, sem valor real)',
    ],
    bookPage: 88,
  },
  {
    id: 'circense',
    name: 'Circense',
    pericias: ['Acrobacia', 'Atuação', 'Reflexos'],
    poderes: ['Acrobático', 'Torcida', 'Truque de Mágica'],
    poderUnico: 'Truque de Mágica',
    itensIniciais: ['Três bolas coloridas para malabarismo (+1 em Atuação)'],
    bookPage: 89,
  },
  {
    id: 'criminoso',
    name: 'Criminoso',
    pericias: ['Enganação', 'Furtividade', 'Ladinagem'],
    poderes: ['Punguista', 'Venefício'],
    poderUnico: 'Punguista',
    itensIniciais: ['Estojo de disfarces OU gazua'],
    bookPage: 89,
  },
  {
    id: 'curandeiro',
    name: 'Curandeiro',
    pericias: ['Cura', 'Vontade'],
    poderes: ['Medicina', 'Médico de Campo', 'Venefício'],
    poderUnico: 'Médico de Campo',
    itensIniciais: ['Bálsamo restaurador ×2', 'Maleta de medicamentos'],
    bookPage: 89,
  },
  {
    id: 'eremita',
    name: 'Eremita',
    pericias: ['Misticismo', 'Religião', 'Sobrevivência'],
    poderes: ['Busca Interior', 'Lobo Solitário'],
    poderUnico: 'Busca Interior',
    itensIniciais: ['Barraca', 'Equipamento de viagem'],
    bookPage: 89,
  },
  {
    id: 'escravo',
    name: 'Escravo',
    pericias: ['Atletismo', 'Fortitude', 'Furtividade'],
    poderes: ['Desejo de Liberdade', 'Vitalidade'],
    poderUnico: 'Desejo de Liberdade',
    itensIniciais: ['Algemas', 'Ferramenta pesada (stats de maça)'],
    bookPage: 89,
  },
  {
    id: 'estudioso',
    name: 'Estudioso',
    pericias: ['Conhecimento', 'Guerra', 'Misticismo'],
    poderes: ['Aparência Inofensiva', 'Palpite Fundamentado'],
    poderUnico: 'Palpite Fundamentado',
    itensIniciais: [
      'Coleção de livros (+1 em Conhecimento, Guerra, Misticismo ou Nobreza à escolha)',
    ],
    bookPage: 90,
  },
  {
    id: 'fazendeiro',
    name: 'Fazendeiro',
    pericias: ['Adestramento', 'Cavalgar', 'Ofício (fazendeiro)', 'Sobrevivência'],
    poderes: ['Água no Feijão', 'Ginete'],
    poderUnico: 'Água no Feijão',
    itensIniciais: [
      'Carroça',
      'Ferramenta agrícola (stats de lança)',
      '10 rações de viagem',
      'Animal não combativo',
    ],
    bookPage: 90,
  },
  {
    id: 'forasteiro',
    name: 'Forasteiro',
    pericias: ['Cavalgar', 'Pilotagem', 'Sobrevivência'],
    poderes: ['Cultura Exótica', 'Lobo Solitário'],
    poderUnico: 'Cultura Exótica',
    itensIniciais: [
      'Equipamento de viagem',
      'Instrumento musical exótico (+1 em perícia de Carisma)',
      'Traje estrangeiro',
    ],
    bookPage: 90,
  },
  {
    id: 'gladiador',
    name: 'Gladiador',
    pericias: ['Atuação', 'Luta'],
    poderes: ['Atraente', 'Pão e Circo', 'Torcida'],
    poderUnico: 'Pão e Circo',
    poderChoiceCategory: 'combate',
    itensIniciais: ['Arma marcial ou exótica', 'Item sem valor de admirador'],
    bookPage: 90,
  },
  {
    id: 'guarda',
    name: 'Guarda',
    pericias: ['Investigação', 'Luta', 'Percepção'],
    poderes: ['Detetive', 'Investigador'],
    poderUnico: 'Detetive',
    poderChoiceCategory: 'combate',
    itensIniciais: ['Apito', 'Insígnia da milícia', 'Arma marcial'],
    bookPage: 91,
  },
  {
    id: 'herdeiro',
    name: 'Herdeiro',
    pericias: ['Misticismo', 'Nobreza', 'Ofício'],
    poderes: ['Comandar', 'Herança'],
    poderUnico: 'Herança',
    itensIniciais: ['Símbolo da herança (anel de sinete, manto cerimonial)'],
    bookPage: 91,
  },
  {
    id: 'heroi-camponês',
    name: 'Herói Camponês',
    pericias: ['Adestramento', 'Ofício'],
    poderes: ['Coração Heroico', 'Sortudo', 'Surto Heroico', 'Torcida'],
    poderUnico: 'Coração Heroico',
    itensIniciais: ['Instrumentos de ofício OU arma simples', 'Traje de plebeu'],
    bookPage: 91,
  },
  {
    id: 'marujo',
    name: 'Marujo',
    pericias: ['Atletismo', 'Jogatina', 'Pilotagem'],
    poderes: ['Acrobático', 'Passagem de Navio'],
    poderUnico: 'Passagem de Navio',
    itensIniciais: ['T$ 2d6 (último salário)', 'Corda'],
    bookPage: 92,
  },
  {
    id: 'mateiro',
    name: 'Mateiro',
    pericias: ['Atletismo', 'Furtividade', 'Sobrevivência'],
    poderes: ['Lobo Solitário', 'Sentidos Aguçados', 'Vendedor de Carcaças'],
    poderUnico: 'Vendedor de Carcaças',
    itensIniciais: ['Arco curto', 'Barraca', 'Equipamento de viagem', '20 flechas'],
    bookPage: 92,
  },
  {
    id: 'membro-de-guilda',
    name: 'Membro de Guilda',
    pericias: ['Diplomacia', 'Enganação', 'Misticismo', 'Ofício'],
    poderes: ['Foco em Perícia', 'Rede de Contatos'],
    poderUnico: 'Rede de Contatos',
    itensIniciais: ['Gazua OU instrumentos de ofício'],
    bookPage: 92,
  },
  {
    id: 'mercador',
    name: 'Mercador',
    pericias: ['Diplomacia', 'Intuição', 'Ofício'],
    poderes: ['Negociação', 'Proficiência', 'Sortudo'],
    poderUnico: 'Negociação',
    itensIniciais: ['Carroça', 'Trobo', 'Mercadorias para vender (T$ 100)'],
    bookPage: 93,
  },
  {
    id: 'minerador',
    name: 'Minerador',
    pericias: ['Atletismo', 'Fortitude', 'Ofício (minerador)'],
    poderes: ['Ataque Poderoso', 'Escavador', 'Sentidos Aguçados'],
    poderUnico: 'Escavador',
    itensIniciais: ['Gemas preciosas (T$ 100)', 'Picareta'],
    bookPage: 93,
  },
  {
    id: 'nomade',
    name: 'Nômade',
    pericias: ['Cavalgar', 'Pilotagem', 'Sobrevivência'],
    poderes: ['Lobo Solitário', 'Mochileiro', 'Sentidos Aguçados'],
    poderUnico: 'Mochileiro',
    itensIniciais: ['Bordão', 'Equipamento de viagem'],
    bookPage: 93,
  },
  {
    id: 'pivete',
    name: 'Pivete',
    pericias: ['Furtividade', 'Iniciativa', 'Ladinagem'],
    poderes: ['Acrobático', 'Aparência Inofensiva', 'Quebra-Galho'],
    poderUnico: 'Quebra-Galho',
    itensIniciais: [
      'Gazua',
      'Traje de plebeu',
      'Animal urbano (cão, gato, rato ou pombo)',
    ],
    bookPage: 93,
  },
  {
    id: 'refugiado',
    name: 'Refugiado',
    pericias: ['Fortitude', 'Reflexos', 'Vontade'],
    poderes: ['Estoico', 'Vontade de Ferro'],
    poderUnico: 'Estoico',
    itensIniciais: ['Um item estrangeiro (até T$ 100)'],
    bookPage: 93,
  },
  {
    id: 'seguidor',
    name: 'Seguidor',
    pericias: ['Adestramento', 'Ofício'],
    poderes: ['Antigo Mestre', 'Proficiência', 'Surto Heroico'],
    poderUnico: 'Antigo Mestre',
    itensIniciais: ['Item recebido do mestre (até T$ 100)'],
    bookPage: 94,
  },
  {
    id: 'selvagem',
    name: 'Selvagem',
    pericias: ['Percepção', 'Reflexos', 'Sobrevivência'],
    poderes: ['Lobo Solitário', 'Vida Rústica', 'Vitalidade'],
    poderUnico: 'Vida Rústica',
    itensIniciais: ['Arma simples', 'Pequeno animal de estimação'],
    bookPage: 94,
  },
  {
    id: 'soldado',
    name: 'Soldado',
    pericias: ['Fortitude', 'Guerra', 'Luta', 'Pontaria'],
    poderes: ['Influência Militar'],
    poderUnico: 'Influência Militar',
    poderChoiceCategory: 'combate',
    itensIniciais: ['Arma marcial', 'Uniforme militar', 'Insígnia de seu exército'],
    bookPage: 94,
  },
  {
    id: 'taverneiro',
    name: 'Taverneiro',
    pericias: ['Diplomacia', 'Jogatina', 'Ofício (cozinheiro)'],
    poderes: ['Gororoba', 'Proficiência', 'Vitalidade'],
    poderUnico: 'Gororoba',
    itensIniciais: [
      'Rolo de macarrão ou martelo de carne (stats de clava)',
      'Panela',
      'Avental',
      'Caneca',
      'Pano sujo',
    ],
    bookPage: 94,
  },
  {
    id: 'trabalhador',
    name: 'Trabalhador',
    pericias: ['Atletismo', 'Fortitude'],
    poderes: ['Atlético', 'Esforçado'],
    poderUnico: 'Atlético',
    itensIniciais: ['Ferramenta pesada (stats de maça ou lança)'],
    bookPage: 95,
  },
]

export const ORIGENS: Readonly<Record<string, Origem>> = Object.freeze(
  ORIGENS_LIST.reduce<Record<string, Origem>>((acc, o) => {
    acc[o.id] = o
    return acc
  }, {}),
)

export const ORIGEM_IDS: readonly string[] = ORIGENS_LIST.map((o) => o.id)

export function origemById(id: string): Origem {
  const o = ORIGENS[id]
  if (!o) throw new Error(`origemById: unknown origem id "${id}"`)
  return o
}

export type OrigemBenefit =
  | { kind: 'pericia'; name: string }
  | { kind: 'poder'; name: string }

/**
 * Validate the player's selection of 2 benefícios from an origem's
 * combined perícia + poder pool. Throws if invalid. Returns nothing on
 * success.
 *
 * Amnésico is skipped here (the GM resolves benefícios manually); use
 * `validateBenefits` only for non-`gmDriven` origens.
 */
export function validateBenefits(
  origemId: string,
  picks: readonly OrigemBenefit[],
): void {
  const origem = origemById(origemId)
  if (origem.gmDriven) {
    throw new Error(
      `validateBenefits: origem "${origemId}" is GM-driven; cannot validate picks`,
    )
  }
  if (picks.length !== ORIGEM_BENEFICIOS_PER_CHARACTER) {
    throw new Error(
      `validateBenefits: must pick exactly ${ORIGEM_BENEFICIOS_PER_CHARACTER}, got ${picks.length}`,
    )
  }
  const seen = new Set<string>()
  for (const pick of picks) {
    const key = `${pick.kind}:${pick.name}`
    if (seen.has(key)) {
      throw new Error(`validateBenefits: duplicate pick ${key}`)
    }
    seen.add(key)
    const pool = pick.kind === 'pericia' ? origem.pericias : origem.poderes
    if (!pool.includes(pick.name)) {
      throw new Error(
        `validateBenefits: ${pick.kind} "${pick.name}" not in origem "${origemId}"`,
      )
    }
  }
}
