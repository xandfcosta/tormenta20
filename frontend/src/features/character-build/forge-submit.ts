import type { QueryClient } from '@tanstack/solid-query'
import { createSignal } from 'solid-js'
import { charactersQueryOptions } from '@/entities/character/queries'
import { ApiError } from '@/shared/api/api'
import type { CreateCharacterInput } from '@/shared/api/types'
import type { CharacterDraftStore } from '@/shared/stores/character-draft-store'
import { createCharacterPayload } from './create-payload'

export type ForgeSubmit = {
  /** Forges the character and walks to its sheet. Safe to call twice — the
   *  second call is ignored while the first is in flight. */
  create: () => Promise<void>
  isPending: boolean
  /** Message to show inline; null once a retry starts. */
  error: string | null
}

export type ForgeSubmitDeps = {
  draft: CharacterDraftStore
  queryClient: QueryClient
  /** The write itself, injected — the page owns the api client. */
  createCharacter: (input: CreateCharacterInput) => Promise<{ id: number }>
  onCreated: (id: number) => void | Promise<void>
}

/**
 * The Forja's one write. Holds the in-flight/error state, so it is a `create*`
 * factory born ONCE in the component body — calling it per click would reset
 * the guard that keeps a double-tap from forging two characters (gotcha #17).
 *
 * The draft is only discarded AFTER the server accepted it: a failed create
 * that threw away the wizard would cost the player the whole build.
 *
 * @example const forge = createForgeSubmit({ draft, queryClient, onCreated })
 */
export function createForgeSubmit(deps: ForgeSubmitDeps): ForgeSubmit {
  const [pending, setPending] = createSignal(false)
  const [error, setError] = createSignal<string | null>(null)

  const create = async () => {
    if (pending()) return
    setPending(true)
    setError(null)
    try {
      const created = await deps.createCharacter(
        createCharacterPayload(deps.draft.values, deps.draft.raceChoices),
      )
      deps.draft.reset()
      await deps.queryClient.invalidateQueries({ queryKey: charactersQueryOptions.queryKey })
      await deps.onCreated(created.id)
    } catch (cause) {
      setError(
        cause instanceof ApiError
          ? cause.message
          : 'Não foi possível forjar o personagem. Tente de novo.',
      )
    } finally {
      setPending(false)
    }
  }

  return {
    create,
    get isPending() {
      return pending()
    },
    get error() {
      return error()
    },
  }
}
