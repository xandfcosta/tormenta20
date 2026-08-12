import { type ParentProps, createComputed, createContext, useContext } from 'solid-js'
import type { CharacterOptions } from '@/shared/api/api'
import type { CharacterDraftStore } from '@/shared/stores/character-draft-store'
import { devotoSyncPatch } from './devocao-sync'
import { deriveDraftVitals } from './draft-vitals'
import { vitalsSyncPatch } from './vitals-sync'

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
  syncDerivedFields(() => props.draft)
  return <ForgeContext.Provider value={forge}>{props.children}</ForgeContext.Provider>
}

/**
 * Fields the draft holds but nobody types: the PV/PM pools the build derives,
 * and the per-class devoto slot the chosen god drives (p96 — one devotion per
 * character).
 *
 * They live in the forge and not in a step because a player can reach the
 * Resumo without ever opening Identidade — the numbers that get SAVED cannot
 * depend on which screens happened to mount.
 *
 * `createComputed`, not `createEffect`: derived state that has to settle in the
 * same update, before anything renders the stale value (gotcha #8).
 */
function syncDerivedFields(draft: () => CharacterDraftStore): void {
  createComputed(() => {
    const store = draft()
    const patch = vitalsSyncPatch(store.values, deriveDraftVitals(store.values, store.raceChoices))
    if (patch) store.patchValues(patch)
  })

  createComputed(() => {
    const store = draft()
    const classChoices = devotoSyncPatch(store.values)
    if (classChoices) store.setValue('classChoices', classChoices)
  })
}

export function useForge(): Forge {
  const forge = useContext(ForgeContext)
  if (!forge) {
    throw new Error('useForge: sem <ForgeProvider> acima na árvore (esperado o shell da Forja)')
  }
  return forge
}
