import { QueryClient } from '@tanstack/solid-query'
import { describe, expect, it } from 'vitest'
import { makeCharacter } from '@/entities/character/__fixtures__/character'
import { characterQueryOptions } from '@/entities/character/queries'
import type { Character } from '@/shared/api/api'
import { createCharacterWrite } from './character-write'

function seeded(character: Character) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  const queryKey = characterQueryOptions(character.id).queryKey
  queryClient.setQueryData<Character>(queryKey, character)
  return {
    queryClient,
    cached: () => queryClient.getQueryData<Character>(queryKey),
    write: createCharacterWrite(queryClient, character.id),
  }
}

describe('createCharacterWrite', () => {
  it('pinta o palpite antes de o servidor responder', async () => {
    const { cached, write } = seeded(makeCharacter({ hpCurrent: 20 }))
    let pintadoDurante = 0

    await write(
      (previous) => ({ ...previous, hpCurrent: 5 }),
      async () => {
        pintadoDurante = cached()?.hpCurrent ?? -1
      },
    )

    expect(pintadoDurante, 'a tela deve mostrar 5 enquanto o servidor pensa').toBe(5)
    expect(cached()?.hpCurrent).toBe(5)
  })

  /**
   * O ponto do rollback é voltar à foto que ESTE write tirou, não refazer o
   * fetch: um refetch correria com a próxima pintura otimista.
   */
  it('volta à foto exata quando o servidor recusa, e repropaga o erro', async () => {
    const { cached, write } = seeded(makeCharacter({ hpCurrent: 20 }))
    const recusa = new Error('servidor recusou')

    await expect(
      write(
        (previous) => ({ ...previous, hpCurrent: 5 }),
        async () => {
          throw recusa
        },
      ),
    ).rejects.toBe(recusa)

    expect(cached()?.hpCurrent, 'a ficha tem de voltar a 20').toBe(20)
  })

  // Quem decide COMO o erro aparece é o chamador (toast num painel, inline num
  // diálogo — um toast disparado dentro de modal não é anunciado). O helper só
  // repropaga.
  it('não engole o erro nem decide como mostrá-lo', async () => {
    const { write } = seeded(makeCharacter())

    const resultado = await write(
      (previous) => previous,
      async () => {
        throw new Error('qualquer')
      },
    ).then(
      () => 'resolveu',
      (e: Error) => e.message,
    )

    expect(resultado).toBe('qualquer')
  })

  it('sem personagem em cache, não inventa um', async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const write = createCharacterWrite(queryClient, 999)
    let enviou = false

    await write(
      (previous) => previous,
      async () => {
        enviou = true
      },
    )

    expect(enviou, 'o envio acontece mesmo sem cache — só a pintura é pulada').toBe(true)
    expect(queryClient.getQueryData(characterQueryOptions(999).queryKey)).toBeUndefined()
  })
})
