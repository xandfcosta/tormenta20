import { describe, expect, it } from 'vitest'
import { makeUsageByKind } from '../skill-usage-resolver'

type DemoKind = 'alpha' | 'beta' | 'gamma'
type DemoUsage = { kind: DemoKind; name: string; value: number }

const DEMO_USAGES: readonly DemoUsage[] = [
  { kind: 'alpha', name: 'Alpha', value: 1 },
  { kind: 'beta', name: 'Beta', value: 2 },
  { kind: 'gamma', name: 'Gamma', value: 3 },
]

describe('makeUsageByKind — fábrica genérica', () => {
  const resolve = makeUsageByKind<DemoKind, DemoUsage>(DEMO_USAGES, 'demoUsageByKind')

  it('resolve kind válido para o usage correto', () => {
    expect(resolve('alpha').value).toBe(1)
    expect(resolve('beta').name).toBe('Beta')
    expect(resolve('gamma').kind).toBe('gamma')
  })

  it('lança em kind desconhecido com mensagem esperada', () => {
    expect(() =>
      // @ts-expect-error — inválido de propósito
      resolve('delta'),
    ).toThrow(/demoUsageByKind: unknown kind delta/)
  })

  it('preserva prefixo do resolver na mensagem', () => {
    const other = makeUsageByKind<DemoKind, DemoUsage>(DEMO_USAGES, 'outroNome')
    expect(() =>
      // @ts-expect-error — inválido de propósito
      other('inexistente'),
    ).toThrow(/outroNome: unknown kind inexistente/)
  })
})
