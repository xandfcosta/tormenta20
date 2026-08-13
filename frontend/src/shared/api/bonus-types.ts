/**
 * Os tipos de bônus da p226, escritos À MÃO pelo mesmo motivo do
 * [AttributeKey](./attribute-keys.ts): no Go isto é `string`, e uma struct não
 * expressa união de literais.
 *
 * Aqui a união vale mais que em outros lugares, porque é ela que decide o
 * EMPILHAMENTO: modificadores do mesmo `bonusType` competem entre si (fica o de
 * maior módulo) enquanto `untyped` soma livremente. Um `bonusType` escrito
 * errado não daria erro nenhum com `string` — viraria um balde novo, empilhando
 * em silêncio, que foi exatamente o modo de falha da regra das condições.
 *
 * O gerador de tipos da fronteira importa isto em vez de emitir `string`.
 */
export type BonusType =
  | 'armor'
  | 'shield'
  | 'item'
  | 'training'
  | 'morale'
  | 'enhancement'
  | 'condition'
  | 'untyped'

/**
 * Onde um item é usado. Mais largo que o `*string` do Go de propósito: existe um
 * SEGUNDO slot de mão, e o app precisa distingui-lo.
 */
export type EquippedSlot = 'vested' | 'wielded' | 'wielded2'
