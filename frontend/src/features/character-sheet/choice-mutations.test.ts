import { QueryClient } from '@tanstack/solid-query'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { makeCharacter } from '@/entities/character/__fixtures__/character'
import { characterQueryOptions } from '@/entities/character/queries'
import type { Character } from '@/shared/api/api'
import { choiceActions } from './choice-mutations'

const CHARACTER_ID = 1

function seeded(char: Character = makeCharacter()) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  client.setQueryData(characterQueryOptions(CHARACTER_ID).queryKey, char)
  return client
}

const cached = (client: QueryClient) =>
  client.getQueryData<Character>(characterQueryOptions(CHARACTER_ID).queryKey)

afterEach(() => vi.restoreAllMocks())

describe('choiceActions', () => {
  // A resposta DIVERGE do palpite (o servidor devolve o blob completo, com um
  // poder que a ficha já tinha): igual, o teste não sabia dizer quem venceu.
  it('pinta o poder escolhido antes da resposta e assenta pelo blob do SERVIDOR', async () => {
    const client = seeded()
    const api = await import('@/shared/api/api')
    let responder = (): void => {}
    const resposta = new Promise<{ classPowers: string }>((resolve) => {
      responder = () =>
        resolve({ classPowers: '["class.barbaro.alma-de-bronze","class.barbaro.furia"]' })
    })
    vi.spyOn(api.api.characters, 'updateAbilityChoices').mockReturnValue(resposta)

    const emVoo = choiceActions(client, CHARACTER_ID).setClassPowers([
      'class.barbaro.alma-de-bronze',
    ])
    await new Promise((resolve) => setTimeout(resolve, 0))
    expect(cached(client)?.classPowers).toBe('["class.barbaro.alma-de-bronze"]')

    responder()
    await emVoo
    expect(cached(client)?.classPowers).toBe(
      '["class.barbaro.alma-de-bronze","class.barbaro.furia"]',
    )
  })

  it('manda só o campo que mudou — o endpoint remenda um subconjunto', async () => {
    const client = seeded()
    const api = await import('@/shared/api/api')
    const update = vi
      .spyOn(api.api.characters, 'updateAbilityChoices')
      .mockResolvedValue({ originChoices: '["poder-sortudo"]' })

    await choiceActions(client, CHARACTER_ID).setOriginChoices(['poder-sortudo'])

    expect(update).toHaveBeenCalledWith(CHARACTER_ID, { originChoices: ['poder-sortudo'] })
  })

  it('falha devolve o blob anterior', async () => {
    const client = seeded(makeCharacter({ classPowers: '["antigo"]' }))
    const api = await import('@/shared/api/api')
    vi.spyOn(api.api.characters, 'updateAbilityChoices').mockRejectedValue(new Error('500'))

    await expect(choiceActions(client, CHARACTER_ID).setClassPowers([])).rejects.toThrow('500')

    expect(cached(client)?.classPowers).toBe('["antigo"]')
  })

  it('caminho de classe vai como objeto e volta como blob', async () => {
    const client = seeded()
    const api = await import('@/shared/api/api')
    const update = vi
      .spyOn(api.api.characters, 'updateAbilityChoices')
      .mockResolvedValue({ classChoices: '{"Bardo":{"caminho":"menestrel"}}' })

    await choiceActions(client, CHARACTER_ID).setClassChoices({
      Bardo: { caminho: 'menestrel' },
    })

    expect(update).toHaveBeenCalledWith(CHARACTER_ID, {
      classChoices: { Bardo: { caminho: 'menestrel' } },
    })
    expect(cached(client)?.classChoices).toBe('{"Bardo":{"caminho":"menestrel"}}')
  })

  // O servidor ecoa SÓ o que escreveu; mesclar o eco inteiro apagaria os outros
  // blobs do personagem em cache.
  it('o eco do servidor não apaga os outros blobs', async () => {
    const client = seeded(makeCharacter({ originChoices: '["intocado"]' }))
    const api = await import('@/shared/api/api')
    vi.spyOn(api.api.characters, 'updateAbilityChoices').mockResolvedValue({
      classPowers: '["novo"]',
    })

    await choiceActions(client, CHARACTER_ID).setClassPowers(['novo'])

    expect(cached(client)?.originChoices).toBe('["intocado"]')
    expect(cached(client)?.classPowers).toBe('["novo"]')
  })
})
