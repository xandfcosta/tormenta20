import { QueryClient } from '@tanstack/solid-query'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { characterQueryOptions } from '@/entities/character/queries'
import type { Character, CharacterExpertise } from '@/shared/api/api'
import {
  addCustomExpertise,
  expertiseActions,
  patchExpertise,
  removeExpertise,
  settleExpertise,
} from './expertise-mutations'

const CHARACTER_ID = 1

function expertise(name: string, trained = false): CharacterExpertise {
  return { name, attribute: 'intelligence', trained, custom: false }
}

function character(expertises: CharacterExpertise[]): Character {
  return { id: CHARACTER_ID, name: 'Tanque', level: 10, expertises } as Character
}

afterEach(() => vi.restoreAllMocks())

describe('patchExpertise', () => {
  it('treina só a perícia nomeada', () => {
    const next = patchExpertise(character([expertise('Atletismo'), expertise('Furtividade')]), 'Atletismo', {
      trained: true,
    })
    expect(next.expertises.find((e) => e.name === 'Atletismo')?.trained).toBe(true)
    expect(next.expertises.find((e) => e.name === 'Furtividade')?.trained).toBe(false)
  })

  it('não muta o personagem que recebeu', () => {
    const original = character([expertise('Atletismo')])
    patchExpertise(original, 'Atletismo', { trained: true })
    expect(original.expertises[0].trained).toBe(false)
  })

  it('rechaveia o atributo sem tocar no treino', () => {
    const next = patchExpertise(character([expertise('Atletismo', true)]), 'Atletismo', {
      attribute: 'strength',
    })
    expect(next.expertises[0]).toMatchObject({ attribute: 'strength', trained: true })
  })
})

describe('addCustomExpertise', () => {
  // Inventar um ofício só faz sentido treinado — é o motivo de existir.
  it('entra treinado e marcado como custom, com o nome aparado', () => {
    const next = addCustomExpertise(character([]), { name: '  Ferraria ', attribute: 'strength' })
    expect(next.expertises).toEqual([
      { name: 'Ferraria', attribute: 'strength', trained: true, custom: true },
    ])
  })
})

describe('removeExpertise / settleExpertise', () => {
  it('remove pelo nome', () => {
    const next = removeExpertise(character([expertise('Ferraria'), expertise('Atletismo')]), 'Ferraria')
    expect(next.expertises.map((e) => e.name)).toEqual(['Atletismo'])
  })

  it('troca a entrada pela linha que o servidor confirmou', () => {
    const server: CharacterExpertise = {
      name: 'Atletismo',
      attribute: 'strength',
      trained: true,
      custom: false,
    }
    const next = settleExpertise(character([expertise('Atletismo')]), server)
    expect(next.expertises[0]).toEqual(server)
  })
})

/** Seeds the cache the way the sheet does, so the actions have something to patch. */
function seededClient(expertises: CharacterExpertise[]) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  client.setQueryData(characterQueryOptions(CHARACTER_ID).queryKey, character(expertises))
  return client
}

const cached = (client: QueryClient) =>
  client.getQueryData<Character>(characterQueryOptions(CHARACTER_ID).queryKey)

describe('expertiseActions', () => {
  it('pinta o treino antes do servidor responder', async () => {
    const client = seededClient([expertise('Atletismo')])
    const api = await import('@/shared/api/api')
    let resolve: (v: CharacterExpertise) => void = () => {}
    vi.spyOn(api.api.characters, 'updateExpertise').mockReturnValue(
      new Promise((r) => {
        resolve = r
      }),
    )

    const pending = expertiseActions(client, CHARACTER_ID).update('Atletismo', { trained: true })

    // Ainda SEM resposta do backend (a promise segue pendente), a ficha já
    // mostra treinado. A escrita otimista acontece um microtask depois da
    // chamada, porque `cancelQueries` é aguardado antes dela — sub-frame, mas
    // não síncrono, e o teste diz isso em vez de fingir que é.
    await vi.waitFor(() => expect(cached(client)?.expertises[0].trained).toBe(true))

    resolve({ name: 'Atletismo', attribute: 'intelligence', trained: true, custom: false })
    await pending
  })

  // O otimismo só se sustenta se o desfazer for real: sem isso a ficha fica
  // mentindo que treinou algo que o servidor recusou.
  it('desfaz ao falhar, voltando ao estado exato de antes', async () => {
    const client = seededClient([expertise('Atletismo')])
    const api = await import('@/shared/api/api')
    vi.spyOn(api.api.characters, 'updateExpertise').mockRejectedValue(new Error('403'))

    await expect(
      expertiseActions(client, CHARACTER_ID).update('Atletismo', { trained: true }),
    ).rejects.toThrow('403')

    expect(cached(client)?.expertises[0].trained).toBe(false)
  })

  it('remover também desfaz quando o backend recusa', async () => {
    const client = seededClient([expertise('Ferraria')])
    const api = await import('@/shared/api/api')
    vi.spyOn(api.api.characters, 'deleteExpertise').mockRejectedValue(new Error('500'))

    await expect(expertiseActions(client, CHARACTER_ID).remove('Ferraria')).rejects.toThrow('500')

    expect(cached(client)?.expertises.map((e) => e.name)).toEqual(['Ferraria'])
  })
})
