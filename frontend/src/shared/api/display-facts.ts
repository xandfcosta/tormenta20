/**
 * Fatos de exibição — as linhas de "o que este item/criatura faz" que a ficha
 * mostra como texto, sem virar número. Movido do `t20-data` (ALE-109).
 */
export type FactCategory = 'dr' | 'immunity' | 'sense' | 'movement' | 'action' | 'other'

export type DisplayFact = {
  category: FactCategory
  text: string
}

/** Rótulos legíveis, para cabeçalho de grupo e tom do chip. */
export const FACT_CATEGORY_LABEL: Record<FactCategory, string> = {
  dr: 'Redução de dano',
  immunity: 'Imunidade',
  sense: 'Sentido',
  movement: 'Movimento',
  action: 'Ação',
  other: 'Outro',
}
