import type { QueryClient } from '@tanstack/solid-query'
import { characterQueryOptions } from '@/entities/character/queries'
import type { Character } from '@/shared/api/api'

/**
 * The optimistic write choreography every sheet mutation runs: cancel in-flight
 * refetches, snapshot the cache, paint the guess, send, and roll back to the
 * exact snapshot if the server refuses.
 *
 * One definition on purpose. It was written out longhand in SEVEN modules
 * (items, expertises, spells, levels, proficiencies, choices, effects) — and two
 * of them had already extracted an identical local `optimistic` without finding
 * each other, which is how the copies drift.
 *
 * It deliberately does NOT decide how a failure reaches the user: it rethrows,
 * and the caller renders. A toast fired from inside a modal is not announced
 * (the modal marks its siblings `aria-hidden`), so a write triggered from a
 * dialog needs `DialogInlineError` while the same write from a panel wants a
 * toast — see `frontend/CLAUDE.md`. Baking a toast in here would have made that
 * choice for six screens at once.
 *
 * @example
 * const write = createCharacterWrite(queryClient, characterId)
 * await write(prev => addItem(prev, optimisticItem(input, tempId)), async () => {
 *   const created = await api.characters.addItem(characterId, input)
 *   // settle with the server's own answer
 * })
 */
export function createCharacterWrite(queryClient: QueryClient, characterId: number) {
  const queryKey = characterQueryOptions(characterId).queryKey

  return async function write(
    paint: (previous: Character) => Character,
    send: () => Promise<void>,
  ): Promise<void> {
    await queryClient.cancelQueries({ queryKey })
    const previous = queryClient.getQueryData<Character>(queryKey)
    if (previous) queryClient.setQueryData<Character>(queryKey, paint(previous))
    try {
      await send()
    } catch (failure) {
      // Back to the exact snapshot this write took — not a refetch, which would
      // race the next optimistic paint.
      if (previous) queryClient.setQueryData<Character>(queryKey, previous)
      throw failure
    }
  }
}
