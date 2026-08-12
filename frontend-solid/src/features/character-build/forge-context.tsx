import { type ParentProps, createContext, useContext } from 'solid-js'
import type { CharacterOptions } from '@/shared/api/api'
import type { CharacterDraftStore } from '@/shared/stores/character-draft-store'

export type Forge = {
  /** The in-progress character. A store proxy — read fields, don't destructure. */
  draft: CharacterDraftStore
  /** Races/classes/origins/gods/sizes the backend offers, already loaded. */
  options: CharacterOptions
}

const ForgeContext = createContext<Forge>()

/**
 * Shared state of the creation scene, provided once by the Forja shell and
 * consumed by every step rendered in the router Outlet. The shell never
 * unmounts across step navigation, so the draft survives the walk.
 *
 * Unlike the React wizard there is no form instance here: the draft store IS
 * the form, and it persists — a step reads and writes it directly.
 */
export function ForgeProvider(props: ParentProps<Forge>) {
  // Not destructured: `props.draft` must stay a getter for the store proxy to
  // keep tracking (gotcha #7 of the port).
  const forge: Forge = {
    get draft() {
      return props.draft
    },
    get options() {
      return props.options
    },
  }
  return <ForgeContext.Provider value={forge}>{props.children}</ForgeContext.Provider>
}

export function useForge(): Forge {
  const forge = useContext(ForgeContext)
  if (!forge) {
    throw new Error('useForge: sem <ForgeProvider> acima na árvore (esperado o shell da Forja)')
  }
  return forge
}
