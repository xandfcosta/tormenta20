import { type ParentProps, createContext, useContext } from 'solid-js'
import { type PowerUsesStore, createPowerUsesStore } from './power-uses-store'

const PowerUsesContext = createContext<PowerUsesStore>()

/**
 * Provides the limited-power-use counters. App-wide, like the zustand it
 * replaces — the store is keyed by character, so two heroes open in the same
 * session keep their own counts.
 *
 * Takes an explicit `store` for tests.
 */
export function PowerUsesProvider(props: ParentProps<{ store?: PowerUsesStore }>) {
  const store = props.store ?? createPowerUsesStore()
  return <PowerUsesContext.Provider value={store}>{props.children}</PowerUsesContext.Provider>
}

export function usePowerUses(): PowerUsesStore {
  const store = useContext(PowerUsesContext)
  if (!store) {
    throw new Error(
      'usePowerUses: sem <PowerUsesProvider> acima na árvore (esperado um PowerUsesStore no contexto)',
    )
  }
  return store
}
