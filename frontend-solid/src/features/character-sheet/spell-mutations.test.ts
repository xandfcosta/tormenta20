import { QueryClient } from '@tanstack/solid-query'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { makeCharacter } from '@/entities/character/__fixtures__/character'
import { characterQueryOptions } from '@/entities/character/queries'
import type { Character, CharacterSpell } from '@/shared/api/api'
import { spellActions } from './spell-mutations'

const CHARACTER_ID = 1
const BOLA_DE_FOGO = 'bola-de-fogo'

const learnedRow = (overrides: Partial<CharacterSpell> = {}): CharacterSpell => ({
  id: 7,
  catalogSpellId: BOLA_DE_FOGO,
  prepared: false,
  learnedAt: '2026-01-01T00:00:00.000Z',
  ...overrides,
})

function seeded(char: Character = makeCharacter()) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  client.setQueryData(characterQueryOptions(CHARACTER_ID).queryKey, char)
  return client
}

const cached = (client: QueryClient) =>
  client.getQueryData<Character>(characterQueryOptions(CHARACTER_ID).queryKey)

afterEach(() => vi.restoreAllMocks())

describe('spellActions.learn', () => {
  it('a magia entra na lista antes da resposta e assenta pela linha do servidor', async () => {
    const client = seeded()
    const api = await import('@/shared/api/api')
    vi.spyOn(api.api.characters, 'learnSpell').mockResolvedValue(learnedRow())

    await spellActions(client, CHARACTER_ID).learn(BOLA_DE_FOGO)

    expect(cached(client)?.spells).toEqual([learnedRow()])
  })

  // O id otimista é temporário; deixar ele no cache faria a linha seguinte
  // (preparar/esquecer) mirar uma magia que o servidor não conhece.
  it('o id temporário é trocado pelo real', async () => {
    const client = seeded()
    const api = await import('@/shared/api/api')
    vi.spyOn(api.api.characters, 'learnSpell').mockResolvedValue(learnedRow({ id: 99 }))

    await spellActions(client, CHARACTER_ID).learn(BOLA_DE_FOGO)

    expect(cached(client)?.spells.map((s) => s.id)).toEqual([99])
  })

  it('falha tira a magia que tinha sido pintada', async () => {
    const client = seeded()
    const api = await import('@/shared/api/api')
    vi.spyOn(api.api.characters, 'learnSpell').mockRejectedValue(new Error('409'))

    await expect(spellActions(client, CHARACTER_ID).learn(BOLA_DE_FOGO)).rejects.toThrow('409')

    expect(cached(client)?.spells).toEqual([])
  })
})

describe('spellActions.unlearn', () => {
  it('some com a magia do grimório', async () => {
    const client = seeded(makeCharacter({ spells: [learnedRow()] }))
    const api = await import('@/shared/api/api')
    vi.spyOn(api.api.characters, 'unlearnSpell').mockResolvedValue({
      catalogSpellId: BOLA_DE_FOGO,
      removed: 1,
    })

    await spellActions(client, CHARACTER_ID).unlearn(BOLA_DE_FOGO)

    expect(cached(client)?.spells).toEqual([])
  })

  it('falha devolve a magia', async () => {
    const client = seeded(makeCharacter({ spells: [learnedRow()] }))
    const api = await import('@/shared/api/api')
    vi.spyOn(api.api.characters, 'unlearnSpell').mockRejectedValue(new Error('500'))

    await expect(spellActions(client, CHARACTER_ID).unlearn(BOLA_DE_FOGO)).rejects.toThrow('500')

    expect(cached(client)?.spells).toHaveLength(1)
  })
})

describe('spellActions.setPrepared', () => {
  it('marca como preparada sem tocar nas outras magias', async () => {
    const outra = learnedRow({ id: 8, catalogSpellId: 'luz' })
    const client = seeded(makeCharacter({ spells: [learnedRow(), outra] }))
    const api = await import('@/shared/api/api')
    vi.spyOn(api.api.characters, 'setSpellPrepared').mockResolvedValue(
      learnedRow({ prepared: true }),
    )

    await spellActions(client, CHARACTER_ID).setPrepared(BOLA_DE_FOGO, true)

    const spells = cached(client)?.spells ?? []
    expect(spells.find((s) => s.catalogSpellId === BOLA_DE_FOGO)?.prepared).toBe(true)
    expect(spells.find((s) => s.catalogSpellId === 'luz')?.prepared).toBe(false)
  })

  it('falha volta ao estado anterior', async () => {
    const client = seeded(makeCharacter({ spells: [learnedRow({ prepared: true })] }))
    const api = await import('@/shared/api/api')
    vi.spyOn(api.api.characters, 'setSpellPrepared').mockRejectedValue(new Error('404'))

    await expect(
      spellActions(client, CHARACTER_ID).setPrepared(BOLA_DE_FOGO, false),
    ).rejects.toThrow('404')

    expect(cached(client)?.spells[0].prepared).toBe(true)
  })
})

describe('spellActions.cast', () => {
  // Conjurar NÃO é otimista: o servidor revalida preparada, aprimoramentos e o
  // limite de PM por magia, e recusa com 400 — pintar o PM antes mostraria um
  // gasto que não aconteceu.
  it('assenta o PM pela resposta do servidor', async () => {
    const client = seeded(makeCharacter({ mpCurrent: 20, spells: [learnedRow()] }))
    const api = await import('@/shared/api/api')
    vi.spyOn(api.api.characters, 'castSpell').mockResolvedValue({
      mpCurrent: 14,
      removedEffectIds: [],
    })

    await spellActions(client, CHARACTER_ID).cast(BOLA_DE_FOGO, [])

    expect(cached(client)?.mpCurrent).toBe(14)
  })

  it('recusa do servidor não mexe no PM', async () => {
    const client = seeded(makeCharacter({ mpCurrent: 20, spells: [learnedRow()] }))
    const api = await import('@/shared/api/api')
    vi.spyOn(api.api.characters, 'castSpell').mockRejectedValue(new Error('400'))

    await expect(spellActions(client, CHARACTER_ID).cast(BOLA_DE_FOGO, [])).rejects.toThrow('400')

    expect(cached(client)?.mpCurrent).toBe(20)
  })

  it('o catalisador consumido sai dos efeitos ativos', async () => {
    const client = seeded(
      makeCharacter({
        mpCurrent: 20,
        activeEffects: [
          { id: 5, catalogId: 'cajado', scope: 'scene', modifiers: '[]', createdAt: '' },
        ],
      }),
    )
    const api = await import('@/shared/api/api')
    vi.spyOn(api.api.characters, 'castSpell').mockResolvedValue({
      mpCurrent: 14,
      removedEffectIds: [5],
    })

    await spellActions(client, CHARACTER_ID).cast(BOLA_DE_FOGO, [])

    expect(cached(client)?.activeEffects).toEqual([])
  })
})
