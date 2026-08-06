import { afterEach, describe, expect, it, vi } from 'vitest'
import type { CatalogSpell } from '@tormenta20/t20-data'
import { api } from '@/shared/api/api'
import { buffSpells, spellList } from './use-spell-catalog'

function spell(over: Partial<CatalogSpell> & { id: string; name: string }): CatalogSpell {
  return { buff: undefined, ...(over as CatalogSpell) }
}

const CATALOG: Record<string, CatalogSpell> = {
  b: spell({ id: 'b', name: 'Bola de Fogo' }),
  a: spell({ id: 'a', name: 'Abençoar', buff: { facts: [] } as never }),
}

/** Named fake for the HTTP boundary (CLAUDE.md: no inline stubs). */
class FakeFetch {
  lastUrl = ''
  private readonly body: unknown
  constructor(body: unknown) {
    this.body = body
  }
  install() {
    vi.stubGlobal('fetch', (url: string) => {
      this.lastUrl = url
      return Promise.resolve({
        ok: true,
        status: 200,
        json: () => Promise.resolve(this.body),
      } as Response)
    })
  }
}

describe('spell catalog helpers', () => {
  afterEach(() => vi.unstubAllGlobals())

  it('spellList sorts by name and tolerates an unloaded catalog', () => {
    expect(spellList(undefined)).toEqual([])
    expect(spellList(CATALOG).map((s) => s.name)).toEqual([
      'Abençoar',
      'Bola de Fogo',
    ])
  })

  it('buffSpells keeps only buff spells', () => {
    expect(buffSpells(CATALOG).map((s) => s.id)).toEqual(['a'])
    expect(buffSpells(undefined)).toEqual([])
  })

  it('api.catalog.spells fetches the catalog endpoint', async () => {
    const fake = new FakeFetch(CATALOG)
    fake.install()
    const res = await api.catalog.spells()
    expect(fake.lastUrl).toBe('/api/catalog/spells')
    expect(Object.keys(res)).toEqual(['b', 'a'])
  })
})
