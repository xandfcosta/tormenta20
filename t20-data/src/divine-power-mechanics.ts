/**
 * Divine power mechanics — action-economy + PM metadata pra cada
 * poder concedido por deus maior.
 *
 * Complementa `abilities/deuses.ts` (que só carrega `poderesConcedidos`
 * como nomes). PDF Cap 2 Poderes p132-135 (Poderes Concedidos).
 *
 * Cada deus maior concede 4 poderes. Total: 20 deuses × 4 = 80.
 */

export type DivinePowerAction =
  | 'padrao'
  | 'movimento'
  | 'livre'
  | 'reacao'
  | 'gratuita'
  | 'completa'
  | 'passivo'
  | 'varia'

/** Limite de uso. null = ilimitado (só limitado por PM). */
export type DivinePowerUses = null | 'cena' | 'rodada' | number

export type DivinePower = {
  deusId: string
  name: string
  action: DivinePowerAction
  /** PM fixo, ou 'variavel' quando concede magia (custo = base spell PM). */
  pmCost: number | 'variavel'
  uses: DivinePowerUses
  bookPage: number
}

const DEFAULT_PAGE = 132

const RAW: readonly DivinePower[] = [
  // ─── Aharadak ────────────────────────────────────────────────────
  { deusId: 'aharadak', name: 'Afinidade com a Tormenta', action: 'passivo', pmCost: 0, uses: null, bookPage: DEFAULT_PAGE },
  { deusId: 'aharadak', name: 'Êxtase da Loucura', action: 'reacao', pmCost: 0, uses: null, bookPage: DEFAULT_PAGE },
  { deusId: 'aharadak', name: 'Percepção Temporal', action: 'livre', pmCost: 3, uses: 'cena', bookPage: DEFAULT_PAGE },
  { deusId: 'aharadak', name: 'Rejeição Divina', action: 'passivo', pmCost: 0, uses: null, bookPage: DEFAULT_PAGE },

  // ─── Allihanna ───────────────────────────────────────────────────
  { deusId: 'allihanna', name: 'Compreender os Ermos', action: 'passivo', pmCost: 0, uses: null, bookPage: DEFAULT_PAGE },
  { deusId: 'allihanna', name: 'Dedo Verde', action: 'varia', pmCost: 'variavel', uses: null, bookPage: DEFAULT_PAGE },
  { deusId: 'allihanna', name: 'Descanso Natural', action: 'passivo', pmCost: 0, uses: null, bookPage: DEFAULT_PAGE },
  { deusId: 'allihanna', name: 'Voz da Natureza', action: 'varia', pmCost: 'variavel', uses: null, bookPage: DEFAULT_PAGE },

  // ─── Arsenal ─────────────────────────────────────────────────────
  { deusId: 'arsenal', name: 'Conjurar Arma', action: 'padrao', pmCost: 1, uses: 'cena', bookPage: DEFAULT_PAGE },
  { deusId: 'arsenal', name: 'Coragem Total', action: 'passivo', pmCost: 0, uses: null, bookPage: DEFAULT_PAGE },
  { deusId: 'arsenal', name: 'Fé Guerreira', action: 'passivo', pmCost: 2, uses: null, bookPage: DEFAULT_PAGE },
  { deusId: 'arsenal', name: 'Sangue de Ferro', action: 'livre', pmCost: 3, uses: 'cena', bookPage: DEFAULT_PAGE },

  // ─── Azgher ──────────────────────────────────────────────────────
  { deusId: 'azgher', name: 'Espada Solar', action: 'livre', pmCost: 1, uses: 'cena', bookPage: DEFAULT_PAGE },
  { deusId: 'azgher', name: 'Fulgor Solar', action: 'reacao', pmCost: 1, uses: 'rodada', bookPage: DEFAULT_PAGE },
  { deusId: 'azgher', name: 'Habitante do Deserto', action: 'livre', pmCost: 1, uses: null, bookPage: DEFAULT_PAGE },
  { deusId: 'azgher', name: 'Inimigo de Tenebra', action: 'passivo', pmCost: 0, uses: null, bookPage: DEFAULT_PAGE },

  // ─── Hyninn ──────────────────────────────────────────────────────
  { deusId: 'hyninn', name: 'Apostar com o Trapaceiro', action: 'livre', pmCost: 1, uses: null, bookPage: 133 },
  { deusId: 'hyninn', name: 'Farsa do Fingidor', action: 'varia', pmCost: 'variavel', uses: null, bookPage: 133 },
  { deusId: 'hyninn', name: 'Forma de Macaco', action: 'completa', pmCost: 2, uses: 'cena', bookPage: 133 },
  { deusId: 'hyninn', name: 'Golpista Divino', action: 'passivo', pmCost: 0, uses: null, bookPage: 133 },

  // ─── Kallyadranoch ───────────────────────────────────────────────
  { deusId: 'kallyadranoch', name: 'Aura de Medo', action: 'livre', pmCost: 2, uses: 'cena', bookPage: 133 },
  { deusId: 'kallyadranoch', name: 'Escamas Dracônicas', action: 'passivo', pmCost: 0, uses: null, bookPage: 133 },
  { deusId: 'kallyadranoch', name: 'Presas Primordiais', action: 'livre', pmCost: 1, uses: 'cena', bookPage: 133 },
  { deusId: 'kallyadranoch', name: 'Servos do Dragão', action: 'completa', pmCost: 2, uses: 'cena', bookPage: 133 },

  // ─── Khalmyr ─────────────────────────────────────────────────────
  { deusId: 'khalmyr', name: 'Coragem Total', action: 'passivo', pmCost: 0, uses: null, bookPage: 133 },
  { deusId: 'khalmyr', name: 'Dom da Verdade', action: 'livre', pmCost: 2, uses: 'cena', bookPage: 133 },
  { deusId: 'khalmyr', name: 'Espada Justiceira', action: 'livre', pmCost: 1, uses: 'cena', bookPage: 133 },
  { deusId: 'khalmyr', name: 'Reparar Injustiça', action: 'reacao', pmCost: 2, uses: 'rodada', bookPage: 133 },

  // ─── Lena ────────────────────────────────────────────────────────
  { deusId: 'lena', name: 'Ataque Piedoso', action: 'passivo', pmCost: 0, uses: null, bookPage: 133 },
  { deusId: 'lena', name: 'Aura Restauradora', action: 'passivo', pmCost: 0, uses: null, bookPage: 133 },
  { deusId: 'lena', name: 'Cura Gentil', action: 'passivo', pmCost: 0, uses: null, bookPage: 133 },
  { deusId: 'lena', name: 'Curandeira Perfeita', action: 'passivo', pmCost: 0, uses: null, bookPage: 133 },

  // ─── Lin-Wu ──────────────────────────────────────────────────────
  { deusId: 'lin-wu', name: 'Coragem Total', action: 'passivo', pmCost: 0, uses: null, bookPage: 134 },
  { deusId: 'lin-wu', name: 'Kiai Divino', action: 'livre', pmCost: 3, uses: 'rodada', bookPage: 134 },
  { deusId: 'lin-wu', name: 'Mente Vazia', action: 'passivo', pmCost: 0, uses: null, bookPage: 134 },
  { deusId: 'lin-wu', name: 'Tradição de Lin-Wu', action: 'passivo', pmCost: 0, uses: null, bookPage: 134 },

  // ─── Marah ───────────────────────────────────────────────────────
  { deusId: 'marah', name: 'Aura de Paz', action: 'livre', pmCost: 2, uses: 'cena', bookPage: 134 },
  { deusId: 'marah', name: 'Dom da Esperança', action: 'passivo', pmCost: 0, uses: null, bookPage: 134 },
  { deusId: 'marah', name: 'Palavras de Bondade', action: 'varia', pmCost: 'variavel', uses: null, bookPage: 134 },
  { deusId: 'marah', name: 'Talento Artístico', action: 'passivo', pmCost: 0, uses: null, bookPage: 134 },

  // ─── Megalokk ────────────────────────────────────────────────────
  { deusId: 'megalokk', name: 'Olhar Amedrontador', action: 'varia', pmCost: 'variavel', uses: null, bookPage: 134 },
  { deusId: 'megalokk', name: 'Presas Primordiais', action: 'livre', pmCost: 1, uses: 'cena', bookPage: 134 },
  { deusId: 'megalokk', name: 'Urro Divino', action: 'livre', pmCost: 1, uses: null, bookPage: 134 },
  { deusId: 'megalokk', name: 'Voz dos Monstros', action: 'passivo', pmCost: 0, uses: null, bookPage: 134 },

  // ─── Nimb ────────────────────────────────────────────────────────
  { deusId: 'nimb', name: 'Êxtase da Loucura', action: 'reacao', pmCost: 0, uses: null, bookPage: 134 },
  { deusId: 'nimb', name: 'Poder Oculto', action: 'movimento', pmCost: 2, uses: 'cena', bookPage: 134 },
  { deusId: 'nimb', name: 'Sorte dos Loucos', action: 'livre', pmCost: 1, uses: null, bookPage: 134 },
  { deusId: 'nimb', name: 'Transmissão da Loucura', action: 'varia', pmCost: 'variavel', uses: null, bookPage: 134 },

  // ─── Oceano ──────────────────────────────────────────────────────
  { deusId: 'oceano', name: 'Anfíbio', action: 'passivo', pmCost: 0, uses: null, bookPage: 134 },
  { deusId: 'oceano', name: 'Arsenal das Profundezas', action: 'passivo', pmCost: 0, uses: null, bookPage: 134 },
  { deusId: 'oceano', name: 'Mestre dos Mares', action: 'varia', pmCost: 'variavel', uses: null, bookPage: 134 },
  { deusId: 'oceano', name: 'Sopro do Mar', action: 'padrao', pmCost: 1, uses: 'rodada', bookPage: 134 },

  // ─── Sszzaas ─────────────────────────────────────────────────────
  { deusId: 'sszzaas', name: 'Astúcia da Serpente', action: 'passivo', pmCost: 0, uses: null, bookPage: 134 },
  { deusId: 'sszzaas', name: 'Familiar Ofídico', action: 'passivo', pmCost: 0, uses: null, bookPage: 134 },
  { deusId: 'sszzaas', name: 'Presas Venenosas', action: 'movimento', pmCost: 1, uses: 'cena', bookPage: 134 },
  { deusId: 'sszzaas', name: 'Sangue Ofídico', action: 'passivo', pmCost: 0, uses: null, bookPage: 134 },

  // ─── Tanna-Toh ───────────────────────────────────────────────────
  { deusId: 'tanna-toh', name: 'Conhecimento Enciclopédico', action: 'passivo', pmCost: 0, uses: null, bookPage: 135 },
  { deusId: 'tanna-toh', name: 'Mente Analítica', action: 'passivo', pmCost: 0, uses: null, bookPage: 135 },
  { deusId: 'tanna-toh', name: 'Pesquisa Abençoada', action: 'passivo', pmCost: 0, uses: null, bookPage: 135 },
  { deusId: 'tanna-toh', name: 'Voz da Civilização', action: 'passivo', pmCost: 0, uses: null, bookPage: 135 },

  // ─── Tenebra ─────────────────────────────────────────────────────
  { deusId: 'tenebra', name: 'Carícia Sombria', action: 'padrao', pmCost: 1, uses: 'rodada', bookPage: 135 },
  { deusId: 'tenebra', name: 'Manto da Penumbra', action: 'varia', pmCost: 'variavel', uses: null, bookPage: 135 },
  { deusId: 'tenebra', name: 'Visão nas Trevas', action: 'passivo', pmCost: 0, uses: null, bookPage: 135 },
  { deusId: 'tenebra', name: 'Zumbificar', action: 'completa', pmCost: 3, uses: null, bookPage: 135 },

  // ─── Thwor ───────────────────────────────────────────────────────
  { deusId: 'thwor', name: 'Almejar o Impossível', action: 'passivo', pmCost: 0, uses: null, bookPage: 135 },
  { deusId: 'thwor', name: 'Fúria Divina', action: 'livre', pmCost: 2, uses: 'cena', bookPage: 135 },
  { deusId: 'thwor', name: 'Olhar Amedrontador', action: 'varia', pmCost: 'variavel', uses: null, bookPage: 135 },
  { deusId: 'thwor', name: 'Tropas Duyshidakk', action: 'completa', pmCost: 2, uses: 'cena', bookPage: 135 },

  // ─── Thyatis ─────────────────────────────────────────────────────
  { deusId: 'thyatis', name: 'Ataque Piedoso', action: 'passivo', pmCost: 0, uses: null, bookPage: 135 },
  { deusId: 'thyatis', name: 'Dom da Imortalidade', action: 'passivo', pmCost: 0, uses: null, bookPage: 135 },
  { deusId: 'thyatis', name: 'Dom da Profecia', action: 'varia', pmCost: 'variavel', uses: null, bookPage: 135 },
  { deusId: 'thyatis', name: 'Dom da Ressurreição', action: 'completa', pmCost: 'variavel', uses: 'cena', bookPage: 135 },

  // ─── Valkaria ────────────────────────────────────────────────────
  { deusId: 'valkaria', name: 'Almejar o Impossível', action: 'passivo', pmCost: 0, uses: null, bookPage: 135 },
  { deusId: 'valkaria', name: 'Armas da Ambição', action: 'passivo', pmCost: 0, uses: null, bookPage: 135 },
  { deusId: 'valkaria', name: 'Coragem Total', action: 'passivo', pmCost: 0, uses: null, bookPage: 135 },
  { deusId: 'valkaria', name: 'Liberdade Divina', action: 'livre', pmCost: 2, uses: 'rodada', bookPage: 135 },

  // ─── Wynna ───────────────────────────────────────────────────────
  { deusId: 'wynna', name: 'Bênção do Mana', action: 'passivo', pmCost: 0, uses: null, bookPage: 135 },
  { deusId: 'wynna', name: 'Centelha Mágica', action: 'varia', pmCost: 'variavel', uses: null, bookPage: 135 },
  { deusId: 'wynna', name: 'Escudo Mágico', action: 'reacao', pmCost: 0, uses: null, bookPage: 135 },
  { deusId: 'wynna', name: 'Teurgista Místico', action: 'passivo', pmCost: 0, uses: null, bookPage: 135 },
]

export const DIVINE_POWERS: readonly DivinePower[] = Object.freeze(RAW)

/** Poderes concedidos de um deus específico. */
export function divinePowersOf(deusId: string): readonly DivinePower[] {
  return DIVINE_POWERS.filter((p) => p.deusId === deusId)
}

/** Filtra só poderes ativos (não-passivos). */
export function activeDivinePowers(deusId: string): readonly DivinePower[] {
  return divinePowersOf(deusId).filter((p) => p.action !== 'passivo')
}

/** Filtra só poderes que concedem magia (pmCost variável). */
export function spellGrantingDivinePowers(): readonly DivinePower[] {
  return DIVINE_POWERS.filter((p) => p.pmCost === 'variavel')
}
