import { describe, expect, it } from 'vitest'
import { pickExclusive, toggleWithLimit } from './choice-lists'

describe('pickExclusive', () => {
  const variants = new Set(['versatil-pericia', 'versatil-poder'])

  it('escolher uma variante tira a irmã que estava marcada', () => {
    const next = pickExclusive(['versatil-pericia', 'outra'], variants, 'versatil-poder')

    expect(next).toEqual(['outra', 'versatil-poder'])
  })

  it('escolhas de outras habilidades ficam intactas', () => {
    expect(pickExclusive(['nada-a-ver'], variants, 'versatil-poder')).toEqual([
      'nada-a-ver',
      'versatil-poder',
    ])
  })

  // Reescolher a mesma variante não pode duplicá-la na lista.
  it('reescolher a mesma variante não duplica', () => {
    const next = pickExclusive(['versatil-poder'], variants, 'versatil-poder')

    expect(next).toEqual(['versatil-poder'])
  })
})

describe('toggleWithLimit', () => {
  it('marca enquanto houver vaga', () => {
    expect(toggleWithLimit([], 'a', 2)).toEqual(['a'])
    expect(toggleWithLimit(['a'], 'b', 2)).toEqual(['a', 'b'])
  })

  it('desmarcar sempre pode, mesmo no limite', () => {
    expect(toggleWithLimit(['a', 'b'], 'a', 2)).toEqual(['b'])
  })

  // No limite, marcar mais um é ignorado — a lista volta igual, e é isso que
  // deixa o chamador saber que nada mudou.
  it('no limite, marcar outro não muda nada', () => {
    const selected = ['a', 'b']

    expect(toggleWithLimit(selected, 'c', 2)).toEqual(selected)
  })
})
