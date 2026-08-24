import { describe, expect, it } from 'vitest'
import { hueFromName } from './hue-from-name'

describe('hueFromName', () => {
  /**
   * Valores FIXADOS, e é o que faltava: o teste antigo afirmava
   * `hueFromName('Thorvald')` igual a `hueFromName('Thorvald')` — a
   * implementação comparada consigo mesma, que só falharia se a função virasse
   * aleatória. Nenhum dos quatro casos prendia um número, e foi por isso que o
   * `@example` do docstring pôde dizer 214 (o certo é 186) sem ninguém notar.
   *
   * Agora eles valem o dobro: existe uma SEGUNDA implementação em Go (a peça do
   * tabuleiro no piloto Datastar), e estes números são o contrato entre as duas
   * linguagens. Foram rodados em node com a própria implementação e transcritos
   * — nunca derivados por um hash reescrito na asserção, que é comparar a
   * função consigo mesma outra vez.
   */
  it('fixa o matiz de nomes conhecidos — é o contrato com o motor Go', () => {
    expect(hueFromName('Thorvald')).toBe(186)
    expect(hueFromName('Akira')).toBe(278)
    // Acentuado de propósito: o laço é por RUNA (`for...of`), então "í" entra
    // como um code point e não como dois bytes. Um port que iterasse bytes
    // divergiria exatamente aqui, e em nenhum dos nomes ASCII acima.
    expect(hueFromName('Míriel')).toBe(22)
  })

  it('sempre devolve um matiz dentro de [0, 359]', () => {
    for (const name of ['A', 'Thorvald', 'Míriel Ávila', '', '日本語']) {
      const hue = hueFromName(name)
      expect(hue).toBeGreaterThanOrEqual(0)
      expect(hue).toBeLessThan(360)
    }
  })

  it('espalha nomes diferentes por matizes diferentes', () => {
    const names = ['Thorvald', 'Akira', 'Míriel', 'Bardo', 'Zenith']
    const hues = new Set(names.map(hueFromName))
    // Não é garantia estrita, mas colisão entre os cinco denunciaria um hash
    // degenerado. Espera-se ao menos 4 baldes distintos.
    expect(hues.size).toBeGreaterThanOrEqual(4)
  })

  it('string vazia é estável (matiz 0)', () => {
    expect(hueFromName('')).toBe(0)
  })
})
