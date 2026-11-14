/**
 * Roda a FAXINA de um caso que mexeu na semente, sem deixar que ela fale mais
 * alto que o defeito.
 *
 * A semente é uma só para a corrida inteira, e os specs a dividem em fila: o que
 * um caso deixa para trás é o ponto de partida do próximo. Por isso a faxina mora
 * num `finally`, e não no fim do corpo — um caso que falha no meio é justamente o
 * que mais suja.
 *
 * E ela ganha `catch`: um `finally` que estoura troca, no relatório, o erro do
 * teste pelo da limpeza (ver "O INSTRUMENTO MENTE" em `docs/testing.md`). A falha
 * da faxina vira um aviso que diz o que ficou sujo.
 *
 * @example
 * try { … } finally {
 *   await tidyUp('o Caído do personagem 1', () => takeOff(page, 'Caído'))
 * }
 */
export async function tidyUp(leftover: string, steps: () => Promise<void>): Promise<void> {
  try {
    await steps()
  } catch (error) {
    console.warn(`a faxina falhou e deixou ${leftover} na semente:`, error)
  }
}
