import { describe, expect, it } from 'vitest'
import { tokenAppearance } from './token-appearance'

/**
 * A regra: a cor é da ESPÉCIE e o número é da INSTÂNCIA (ALE-179).
 *
 * Unitário porque é regra de aparência pura, e porque o que ela protege é
 * exatamente o que um teste de tela não veria: dois zumbis com a mesma cor
 * continuam sendo dois botões diferentes no DOM.
 */
describe('tokenAppearance', () => {
  it('os iguais saem iguais: o número não entra na cor nem nas letras', () => {
    const um = tokenAppearance('Zumbi 1')
    const tres = tokenAppearance('Zumbi 3')

    expect(um.background).toBe(tres.background)
    expect(um.monogram).toBe(tres.monogram)
    expect(um.instance).toBe('1')
    expect(tres.instance).toBe('3')
  })

  // O monograma comia as duas primeiras palavras, então o número — a única
  // coisa que distinguia as três peças — era justamente o que se perdia.
  it('o monograma vem da espécie, e o número vira selo', () => {
    const peca = tokenAppearance('Zumbi Putrefato 2')

    expect(peca.monogram).toBe('ZP')
    expect(peca.instance).toBe('2')
  })

  // DUAS letras mesmo em nome de uma palavra: no tabuleiro um "O" solto tem
  // metade da massa que a peça precisa para ser achada entre vinte vizinhas.
  it('sem número, não há selo — e o monograma ainda tem duas letras', () => {
    const peca = tokenAppearance('Ogro')

    expect(peca.instance).toBeUndefined()
    expect(peca.monogram).toBe('OG')
  })

  // "Nv1" e "Nv10" estão no MEIO do nome e não são instância: separar por
  // qualquer dígito transformaria "Recruta Nv1 Simples" em outra espécie.
  it('número no meio do nome não é instância', () => {
    const peca = tokenAppearance('Recruta Nv1 Simples')

    expect(peca.instance).toBeUndefined()
    expect(peca.background).toBe(tokenAppearance('Recruta Nv1 Simples').background)
  })

  // Espécies diferentes continuam com cores diferentes — senão a cor deixaria
  // de dizer qualquer coisa.
  it('espécies diferentes continuam distintas', () => {
    expect(tokenAppearance('Zumbi 1').background).not.toBe(
      tokenAppearance('Goblin 1').background,
    )
  })
})
