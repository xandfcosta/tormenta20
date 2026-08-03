/**
 * Activation costs for flag-grouped stances — abilities the player switches
 * on in the Efeitos tab whose BOOK text charges PM to enter. The toggle UI
 * consults this registry to debit the cost on activation (switching off is
 * free; the PM was spent entering the stance).
 */
export type FlagActivation = {
  flag: string
  name: string
  pmCost: number
  bookPage: number
}

/**
 * Keyed by the `flagOn` condition flag the stance's modifiers share.
 *
 * @example FLAG_ACTIVATIONS['furia'].pmCost // 2 (Bárbaro, p40)
 */
export const FLAG_ACTIVATIONS: Record<string, FlagActivation> = {
  furia: { flag: 'furia', name: 'Fúria', pmCost: 2, bookPage: 40 },
  // Bardo p44: "gastar uma ação padrão e 2 PM ... +1 em testes de perícia até
  // o fim da cena". Electives (Golpe Elemental etc.) already gate on this flag
  // via requiresInspiracao.
  inspiracao: { flag: 'inspiracao', name: 'Inspiração', pmCost: 2, bookPage: 44 },
}
