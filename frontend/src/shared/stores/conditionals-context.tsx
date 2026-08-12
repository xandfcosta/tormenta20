import { type ParentProps, createContext, useContext } from 'solid-js'
import { type ConditionalsStore, createConditionalsStore } from './conditionals-store'

const ConditionalsContext = createContext<ConditionalsStore>()

/**
 * Provides the opt-in conditionals (Fúria & co.) store. App-wide, like the
 * React zustand it replaces — the store is keyed by character, so two heroes
 * open in the same session keep their own toggles.
 *
 * Takes an explicit `store` for tests.
 */
export function ConditionalsProvider(props: ParentProps<{ store?: ConditionalsStore }>) {
  const store = props.store ?? createConditionalsStore()
  return (
    <ConditionalsContext.Provider value={store}>{props.children}</ConditionalsContext.Provider>
  )
}

export function useConditionals(): ConditionalsStore {
  const store = useContext(ConditionalsContext)
  if (!store) {
    throw new Error(
      'useConditionals: sem <ConditionalsProvider> acima na árvore (esperado um ConditionalsStore no contexto)',
    )
  }
  return store
}
