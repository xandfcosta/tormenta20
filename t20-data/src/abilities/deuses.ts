/**
 * Tormenta 20 panteão (PDF Cap 3, Religião — Tabela 1-20, p97). The 20
 * deuses maiores of Arton. The catalog is used by the classChoices picker
 * (clérigo/paladino/druida devoto) and by typed `classChoice` prerequisites
 * that gate powers on devotion.
 *
 * `paladinoEligible` / `druidaEligible` reflect the per-class devoto whitelist:
 *   Paladino (p82) — Azgher, Khalmyr, Lena, Lin-Wu, Marah, Tanna-Toh, Thyatis, Valkaria
 *   Druida   (p61) — Allihanna, Megalokk, Oceano
 *
 * Clérigos may pick any deus maior (p57). The boolean `major` separates
 * deuses maiores from deuses menores for the clérigo picker — currently all
 * entries are maiores; deuses menores arrive in future supplements (p105).
 */
/**
 * Energia canalizada — book uses Positiva / Negativa / Qualquer (book p97).
 * 'qualquer' = devoto picks one at character creation (irreversible).
 *
 * Note: T20 explicitly drops D&D-style 2-axis alinhamento. No alinhamento
 * field here. The clérigo's behavioural restrictions come from the deus's
 * doutrina prose, not a numeric grid.
 */
export type DeusEnergia = 'positiva' | 'negativa' | 'qualquer'

export type Deus = {
  id: string
  name: string
  major: boolean
  paladinoEligible: boolean
  druidaEligible: boolean
  /** PDF p96-105 enrichment fields — empty on Panteão / Paladino do Bem sentinels. */
  portfolio?: string
  energia?: DeusEnergia
  simbolo?: string
  /**
   * Arma preferida. `null` quando o deus proíbe explicitamente Arma
   * Espiritual (Lena, Marah). Nimb usa `'todas'` por sua natureza caótica.
   */
  armaPreferida?: string | null
  /** Sempre 4 poderes concedidos por deus maior (book Cap 2 — Poderes Concedidos). */
  poderesConcedidos?: readonly string[]
  /** Raças + classes elegíveis a devoção (verbatim do "Devotos." line). */
  devotos?: readonly string[]
  bookPage?: number
}

/**
 * Sentinel ids used in `ClassChoiceBlob.devoto` for the non-divindade
 * alternatives both classes allow:
 *  - Clérigo (p57): "cultuar o Panteão como um todo".
 *  - Paladino (p82): "paladino do bem [...] sem deus específico".
 *
 * Stored alongside deus ids in the same slot so the picker has a single
 * source of truth; prereq checks treat them as "not devoto of a divindade"
 * (Arma Sagrada explicitly forbids `paladino-do-bem`, for instance).
 */
export const CULTO_PANTEAO = 'panteao'
export const CULTO_PALADINO_DO_BEM = 'paladino-do-bem'

export const DEUSES: Deus[] = [
  {
    id: 'aharadak',
    name: 'Aharadak',
    major: true, paladinoEligible: false, druidaEligible: false,
    portfolio: 'Tormenta, escatologia, loucura',
    energia: 'negativa',
    simbolo: 'Olho macabro de pupila vertical cercado de espinhos',
    armaPreferida: 'Corrente de espinhos',
    poderesConcedidos: ['Afinidade com a Tormenta', 'Êxtase da Loucura', 'Percepção Temporal', 'Rejeição Divina'],
    devotos: ['Quaisquer'],
    bookPage: 96,
  },
  {
    id: 'allihanna',
    name: 'Allihanna',
    major: true, paladinoEligible: false, druidaEligible: true,
    portfolio: 'Natureza, vida selvagem',
    energia: 'positiva',
    simbolo: 'Pequena árvore (ou animal totêmico)',
    armaPreferida: 'Bordão',
    poderesConcedidos: ['Compreender os Ermos', 'Dedo Verde', 'Descanso Natural', 'Voz da Natureza'],
    devotos: ['Dahllan', 'Elfos', 'Sílfides', 'Bárbaros', 'Caçadores', 'Druidas'],
    bookPage: 97,
  },
  {
    id: 'arsenal',
    name: 'Arsenal',
    major: true, paladinoEligible: false, druidaEligible: false,
    portfolio: 'Guerra, conflito',
    energia: 'qualquer',
    simbolo: 'Martelo de guerra e espada longa cruzados sobre um escudo',
    armaPreferida: 'Martelo de guerra',
    poderesConcedidos: ['Conjurar Arma', 'Coragem Total', 'Fé Guerreira', 'Sangue de Ferro'],
    devotos: ['Anões', 'Minotauros', 'Bárbaros', 'Cavaleiros', 'Guerreiros', 'Lutadores'],
    bookPage: 98,
  },
  {
    id: 'azgher',
    name: 'Azgher',
    major: true, paladinoEligible: true, druidaEligible: false,
    portfolio: 'Sol, deserto, honestidade',
    energia: 'positiva',
    simbolo: 'Um sol dourado',
    armaPreferida: 'Cimitarra',
    poderesConcedidos: ['Espada Solar', 'Fulgor Solar', 'Habitante do Deserto', 'Inimigo de Tenebra'],
    devotos: ['Aggelus', 'Qareen', 'Arcanistas', 'Bárbaros', 'Caçadores', 'Cavaleiros', 'Guerreiros', 'Nobres', 'Paladinos'],
    bookPage: 98,
  },
  {
    id: 'hyninn',
    name: 'Hyninn',
    major: true, paladinoEligible: false, druidaEligible: false,
    portfolio: 'Trapaça, astúcia',
    energia: 'qualquer',
    simbolo: 'Adaga atravessando uma máscara/raposa',
    armaPreferida: 'Adaga',
    poderesConcedidos: ['Apostar com o Trapaceiro', 'Farsa do Fingidor', 'Forma de Macaco', 'Golpista Divino'],
    devotos: ['Hynne', 'Goblins', 'Sílfides', 'Bardos', 'Bucaneiros', 'Ladinos', 'Inventores', 'Nobres'],
    bookPage: 98,
  },
  {
    id: 'kallyadranoch',
    name: 'Kallyadranoch',
    major: true, paladinoEligible: false, druidaEligible: false,
    portfolio: 'Dragões, soberania, riqueza',
    energia: 'negativa',
    simbolo: 'Escamas de cinco cores diferentes',
    armaPreferida: 'Lança',
    poderesConcedidos: ['Aura de Medo', 'Escamas Dracônicas', 'Presas Primordiais', 'Servos do Dragão'],
    devotos: ['Elfos', 'Medusas', 'Sulfure', 'Arcanistas', 'Cavaleiros', 'Guerreiros', 'Lutadores', 'Nobres'],
    bookPage: 99,
  },
  {
    id: 'khalmyr',
    name: 'Khalmyr',
    major: true, paladinoEligible: true, druidaEligible: false,
    portfolio: 'Justiça, ordem, lei',
    energia: 'positiva',
    simbolo: 'Espada sobreposta a uma balança',
    armaPreferida: 'Espada longa',
    poderesConcedidos: ['Coragem Total', 'Dom da Verdade', 'Espada Justiceira', 'Reparar Injustiça'],
    devotos: ['Aggelus', 'Anões', 'Cavaleiros', 'Guerreiros', 'Nobres', 'Paladinos'],
    bookPage: 99,
  },
  {
    id: 'lena',
    name: 'Lena',
    major: true, paladinoEligible: true, druidaEligible: false,
    portfolio: 'Vida, cura, fertilidade',
    energia: 'positiva',
    simbolo: 'Lua crescente prateada',
    armaPreferida: null,
    poderesConcedidos: ['Ataque Piedoso', 'Aura Restauradora', 'Cura Gentil', 'Curandeira Perfeita'],
    devotos: ['Dahllan', 'Qareen', 'Nobres', 'Paladinos'],
    bookPage: 100,
  },
  {
    id: 'lin-wu',
    name: 'Lin-Wu',
    major: true, paladinoEligible: true, druidaEligible: false,
    portfolio: 'Honra, samurai, Império de Jade',
    energia: 'qualquer',
    simbolo: 'Placa de metal com dragão-serpente celestial',
    armaPreferida: 'Katana',
    poderesConcedidos: ['Coragem Total', 'Kiai Divino', 'Mente Vazia', 'Tradição de Lin-Wu'],
    devotos: ['Anões', 'Cavaleiros', 'Guerreiros', 'Nobres', 'Paladinos'],
    bookPage: 100,
  },
  {
    id: 'marah',
    name: 'Marah',
    major: true, paladinoEligible: true, druidaEligible: false,
    portfolio: 'Paz, amor, harmonia',
    energia: 'positiva',
    simbolo: 'Coração vermelho',
    armaPreferida: null,
    poderesConcedidos: ['Aura de Paz', 'Dom da Esperança', 'Palavras de Bondade', 'Talento Artístico'],
    devotos: ['Aggelus', 'Elfos', 'Hynne', 'Qareen', 'Bardos', 'Nobres', 'Paladinos'],
    bookPage: 101,
  },
  {
    id: 'megalokk',
    name: 'Megalokk',
    major: true, paladinoEligible: false, druidaEligible: true,
    portfolio: 'Monstros, ferocidade',
    energia: 'negativa',
    simbolo: 'Garra de um monstro',
    armaPreferida: 'Maça',
    poderesConcedidos: ['Olhar Amedrontador', 'Presas Primordiais', 'Urro Divino', 'Voz dos Monstros'],
    devotos: ['Goblins', 'Medusas', 'Minotauros', 'Sulfure', 'Trogs', 'Bárbaros', 'Caçadores', 'Druidas', 'Lutadores'],
    bookPage: 101,
  },
  {
    id: 'nimb',
    name: 'Nimb',
    major: true, paladinoEligible: false, druidaEligible: false,
    portfolio: 'Caos, sorte, loucura',
    energia: 'qualquer',
    simbolo: 'Dado de seis faces',
    armaPreferida: 'todas',
    poderesConcedidos: ['Êxtase da Loucura', 'Poder Oculto', 'Sorte dos Loucos', 'Transmissão da Loucura'],
    devotos: ['Goblins', 'Qareen', 'Sílfides', 'Arcanistas', 'Bárbaros', 'Bardos', 'Bucaneiros', 'Inventores', 'Ladinos'],
    bookPage: 102,
  },
  {
    id: 'oceano',
    name: 'Oceano',
    major: true, paladinoEligible: false, druidaEligible: true,
    portfolio: 'Mares, marinheiros',
    energia: 'qualquer',
    simbolo: 'Concha',
    armaPreferida: 'Tridente',
    poderesConcedidos: ['Anfíbio', 'Arsenal das Profundezas', 'Mestre dos Mares', 'Sopro do Mar'],
    devotos: ['Dahllan', 'Hynne', 'Minotauros', 'Sereias/Tritões', 'Bárbaros', 'Bucaneiros', 'Caçadores', 'Druidas'],
    bookPage: 102,
  },
  {
    id: 'sszzaas',
    name: 'Sszzaas',
    major: true, paladinoEligible: false, druidaEligible: false,
    portfolio: 'Traição, mentira',
    energia: 'negativa',
    simbolo: 'Naja vertendo veneno pelas presas',
    armaPreferida: 'Adaga',
    poderesConcedidos: ['Astúcia da Serpente', 'Familiar Ofídico', 'Presas Venenosas', 'Sangue Ofídico'],
    devotos: ['Medusas', 'Arcanistas', 'Bardos', 'Bucaneiros', 'Inventores', 'Ladinos', 'Nobres'],
    bookPage: 103,
  },
  {
    id: 'tanna-toh',
    name: 'Tanna-Toh',
    major: true, paladinoEligible: true, druidaEligible: false,
    portfolio: 'Conhecimento, civilização',
    energia: 'qualquer',
    simbolo: 'Pergaminho e pena',
    armaPreferida: 'Bordão',
    poderesConcedidos: ['Conhecimento Enciclopédico', 'Mente Analítica', 'Pesquisa Abençoada', 'Voz da Civilização'],
    devotos: ['Golens', 'Kliren', 'Arcanistas', 'Bardos', 'Inventores', 'Nobres', 'Paladinos'],
    bookPage: 103,
  },
  {
    id: 'tenebra',
    name: 'Tenebra',
    major: true, paladinoEligible: false, druidaEligible: false,
    portfolio: 'Noite, escuridão, mortos-vivos',
    energia: 'negativa',
    simbolo: 'Estrela negra de cinco pontas',
    armaPreferida: 'Adaga',
    poderesConcedidos: ['Carícia Sombria', 'Manto da Penumbra', 'Visão nas Trevas', 'Zumbificar'],
    devotos: ['Anões', 'Medusas', 'Qareen', 'Osteon', 'Sulfure', 'Trogs', 'Arcanistas', 'Bardos', 'Ladinos'],
    bookPage: 104,
  },
  {
    id: 'thwor',
    name: 'Thwor',
    major: true, paladinoEligible: false, druidaEligible: false,
    portfolio: 'Goblinoides, força',
    energia: 'qualquer',
    simbolo: 'Grande punho fechado',
    armaPreferida: 'Machado de guerra',
    poderesConcedidos: ['Almejar o Impossível', 'Fúria Divina', 'Olhar Amedrontador', 'Tropas Duyshidakk'],
    devotos: ['Qualquer duyshidakk'],
    bookPage: 104,
  },
  {
    id: 'thyatis',
    name: 'Thyatis',
    major: true, paladinoEligible: true, druidaEligible: false,
    portfolio: 'Ressurreição, profecia, perdão',
    energia: 'positiva',
    simbolo: 'Ave fênix',
    armaPreferida: 'Espada longa',
    poderesConcedidos: ['Ataque Piedoso', 'Dom da Imortalidade', 'Dom da Profecia', 'Dom da Ressurreição'],
    devotos: ['Aggelus', 'Cavaleiros', 'Guerreiros', 'Inventores', 'Lutadores', 'Paladinos'],
    bookPage: 104,
  },
  {
    id: 'valkaria',
    name: 'Valkaria',
    major: true, paladinoEligible: true, druidaEligible: false,
    portfolio: 'Ambição, humanidade, liberdade',
    energia: 'positiva',
    simbolo: 'Estátua de Valkaria ou seis faixas entrelaçadas',
    armaPreferida: 'Mangual',
    poderesConcedidos: ['Almejar o Impossível', 'Armas da Ambição', 'Coragem Total', 'Liberdade Divina'],
    devotos: ['Aventureiros (todas as classes)'],
    bookPage: 105,
  },
  {
    id: 'wynna',
    name: 'Wynna',
    major: true, paladinoEligible: false, druidaEligible: false,
    portfolio: 'Magia arcana',
    energia: 'qualquer',
    simbolo: 'Anel metálico',
    armaPreferida: 'Adaga',
    poderesConcedidos: ['Bênção do Mana', 'Centelha Mágica', 'Escudo Mágico', 'Teurgista Místico'],
    devotos: ['Elfos', 'Golens', 'Qareen', 'Sílfides', 'Arcanistas', 'Bardos'],
    bookPage: 105,
  },
]

const CULTO_PANTEAO_OPTION: Deus = {
  id: CULTO_PANTEAO,
  name: 'Panteão',
  major: false,
  paladinoEligible: false,
  druidaEligible: false,
}

const CULTO_PALADINO_DO_BEM_OPTION: Deus = {
  id: CULTO_PALADINO_DO_BEM,
  name: 'Paladino do Bem',
  major: false,
  paladinoEligible: false,
  druidaEligible: false,
}

export const DEUS_BY_ID: Record<string, Deus> = Object.fromEntries(
  DEUSES.map((d) => [d.id, d]),
)

/** Caminhos per class — subpath choices unlocked by class abilities. */
export type CaminhoOption = { id: string; name: string }

export const CAMINHOS: Record<string, CaminhoOption[]> = {
  Arcanista: [
    { id: 'bruxo', name: 'Bruxo' },
    { id: 'feiticeiro', name: 'Feiticeiro' },
    { id: 'mago', name: 'Mago' },
  ],
  Paladino: [
    { id: 'egide-sagrada', name: 'Égide Sagrada' },
    { id: 'montaria-sagrada', name: 'Montaria Sagrada' },
  ],
  Cavaleiro: [
    { id: 'bastiao', name: 'Bastião' },
    { id: 'montaria', name: 'Montaria' },
  ],
}

/**
 * Per-class persisted choices keyed by className. Stored as JSON on
 * Character.classChoices and merged into prerequisite evaluation.
 */
export type ClassChoiceBlob = {
  /** Deus id chosen as devoto (clérigo/paladino/druida). */
  devoto?: string
  /** Caminho/subpath id (arcanista, paladino L5, cavaleiro L5, ...). */
  caminho?: string
}

export type ClassChoices = Partial<Record<string, ClassChoiceBlob>>

/**
 * Returns the deus catalog filtered for the given class's devoto picker,
 * or null when the class has no devoto slot. Mirrors the per-class lists
 * in PDF Cap 3 (Religião):
 *  - Clérigo: any deus maior + 'Panteão' option (p57).
 *  - Paladino: 8 deuses whitelist + 'Paladino do Bem' option (p82).
 *  - Druida: Allihanna, Megalokk, Oceano (p61) — no non-divindade alt.
 */
export function devotoOptionsFor(className: string): Deus[] | null {
  switch (className) {
    case 'Clérigo':
      return [...DEUSES.filter((d) => d.major), CULTO_PANTEAO_OPTION]
    case 'Paladino':
      return [
        ...DEUSES.filter((d) => d.paladinoEligible),
        CULTO_PALADINO_DO_BEM_OPTION,
      ]
    case 'Druida':
      return DEUSES.filter((d) => d.druidaEligible)
    default:
      return null
  }
}

/**
 * Per-class caminho/subpath slot. Returns options + the class level at
 * which the caminho choice unlocks (the class's "Caminho" auto-power), or
 * null when the class has no caminho slot.
 */
export function caminhoSlotFor(
  className: string,
): { options: CaminhoOption[]; minLevel: number } | null {
  const options = CAMINHOS[className]
  if (!options) return null
  const minLevel = className === 'Arcanista' ? 1 : 5
  return { options, minLevel }
}
