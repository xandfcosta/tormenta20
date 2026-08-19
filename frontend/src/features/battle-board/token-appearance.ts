import { hueGradient } from '@/shared/lib/hue-from-name'

/** O sufixo numérico que a mesa usa para contar iguais: "Zumbi 3". */
const INSTANCE = /^(.*\S)\s+(\d{1,3})$/

export type TokenAppearance = {
  /** Duas letras da ESPÉCIE, não da instância. */
  monogram: string
  /** O número da instância, quando existe. */
  instance?: string
  /** Gradiente derivado da espécie: os três zumbis saem iguais. */
  background: string
}

/**
 * Como uma peça se parece (ALE-179).
 *
 * A regra que isto carrega: **a cor é da ESPÉCIE e o número é da INSTÂNCIA**.
 * Antes, o matiz vinha do rótulo inteiro, então "Zumbi 1" e "Zumbi 2" — a mesma
 * criatura — saíam em cores sem relação nenhuma, e "Zumbi 3" podia calhar na
 * cor do paladino: a cor dizia "coisas diferentes" sobre coisas iguais. Pior, o
 * monograma come as duas primeiras palavras, então "Zumbi Putrefato 2" virava
 * "ZP" e o número — a única coisa que distingue as três peças na mesa — era
 * justamente o que se perdia.
 *
 * Agora a espécie manda na cor e nas letras, e o número vira selo. "Eu ataco o
 * Zumbi 3" é a frase mais dita da noite, e ela passa a ter resposta num relance
 * — inclusive para quem não distingue matiz, porque o selo é texto.
 *
 * @example tokenAppearance('Zumbi 3') // { monogram: 'ZU', instance: '3', … }
 */
export function tokenAppearance(label: string): TokenAppearance {
  const match = INSTANCE.exec(label.trim())
  const species = match?.[1] ?? label
  return {
    monogram: monogramOf(species),
    instance: match?.[2],
    background: hueGradient(species, 0.55, 0.15),
  }
}

/**
 * DUAS letras, sempre — e por isso não é o `initials` da casa, que devolve uma
 * letra para nome de uma palavra ("Ogro" → "O"). No retrato do herói uma letra
 * grande funciona; no tabuleiro a peça é um disco de 44px cheio de vizinhos, e
 * um "O" solto tem metade da massa que ela precisa para ser achada num relance.
 */
function monogramOf(species: string): string {
  const words = species.trim().split(/\s+/).filter(Boolean)
  if (words.length === 0) return '?'
  const first = words[0] ?? ''
  if (words.length === 1) return first.slice(0, 2).toUpperCase()
  return `${first[0] ?? ''}${words[1]?.[0] ?? ''}`.toUpperCase()
}
