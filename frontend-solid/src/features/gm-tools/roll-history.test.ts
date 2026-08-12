import { createRoot } from 'solid-js'
import { describe, expect, it } from 'vitest'
import { createRollHistory } from './roll-history'

const withHistory = <T>(fn: (h: ReturnType<typeof createRollHistory<string>>) => T): T =>
  createRoot((dispose) => {
    const result = fn(createRollHistory<string>())
    dispose()
    return result
  })

describe('createRollHistory', () => {
  it('começa vazio', () => {
    withHistory((history) => {
      expect(history.entries()).toEqual([])
      expect(history.latest()).toBeUndefined()
    })
  })

  it('guarda a rolagem mais nova na frente', () => {
    withHistory((history) => {
      history.push(3, 'primeira')
      history.push(5, 'segunda')

      expect(history.latest()).toEqual({ roll: 5, result: 'segunda' })
      expect(history.entries().map((e) => e.result)).toEqual(['segunda', 'primeira'])
    })
  })

  it('lembra só as cinco últimas — o resto é ruído', () => {
    withHistory((history) => {
      for (let roll = 1; roll <= 8; roll++) history.push(roll, `r${roll}`)

      expect(history.entries()).toHaveLength(5)
      expect(history.entries().at(-1)?.roll).toBe(4)
    })
  })

  it('limpar esvazia o histórico', () => {
    withHistory((history) => {
      history.push(1, 'x')
      history.clear()

      expect(history.entries()).toEqual([])
    })
  })
})
