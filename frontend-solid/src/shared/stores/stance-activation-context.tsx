import { type ParentProps, createContext, useContext } from 'solid-js'
import {
  type StanceActivationStore,
  createStanceActivationStore,
} from './stance-activation-store'

const StanceActivationContext = createContext<StanceActivationStore>()

/**
 * Provides the record of what each active stance cost. App-wide, like the
 * zustand it replaces — keyed by character, so two heroes open in the same
 * session keep their own payments.
 *
 * Takes an explicit `store` for tests.
 */
export function StanceActivationProvider(
  props: ParentProps<{ store?: StanceActivationStore }>,
) {
  const store = props.store ?? createStanceActivationStore()
  return (
    <StanceActivationContext.Provider value={store}>
      {props.children}
    </StanceActivationContext.Provider>
  )
}

export function useStanceActivations(): StanceActivationStore {
  const store = useContext(StanceActivationContext)
  if (!store) {
    throw new Error(
      'useStanceActivations: sem <StanceActivationProvider> acima na árvore (esperado um StanceActivationStore no contexto)',
    )
  }
  return store
}
