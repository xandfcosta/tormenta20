import { describe, expect, it } from 'vitest'
import { settledQuery } from './settled-query'

type Sheet = { defense: { total: number } }

/** Stands in for a solid-query result without dragging the library in — the
 *  helper only ever needs these two fields (ALE-95). */
class FakeQueryResult<T> {
  constructor(
    readonly isPending: boolean,
    readonly data: T | undefined,
  ) {}
}

describe('settledQuery', () => {
  it('devolve null enquanto a query está pendente', () => {
    const pending = new FakeQueryResult<Sheet>(true, undefined)

    expect(settledQuery(pending)).toBeNull()
  })

  it('devolve os dados quando a query assentou', () => {
    const settled = new FakeQueryResult<Sheet>(false, { defense: { total: 22 } })

    expect(settledQuery(settled)).toEqual({ defense: { total: 22 } })
  })

  it('devolve null quando assentou sem dados (erro, ou query desabilitada)', () => {
    const failed = new FakeQueryResult<Sheet>(false, undefined)

    expect(settledQuery(failed)).toBeNull()
  })

  // A regressão da ALE-95: a chave mudou, os dados VELHOS ainda estão no
  // objeto e a query voltou a pendente. Devolver `data` aqui mostraria a DEF do
  // personagem ANTERIOR — um número real e errado, pior que o travessão.
  it('não vaza dados do alvo anterior quando a chave muda e volta a pendente', () => {
    const refetching = new FakeQueryResult<Sheet>(true, { defense: { total: 22 } })

    expect(settledQuery(refetching)).toBeNull()
  })
})
