import { type ParentProps, createContext, useContext } from 'solid-js'
import { type UiStore, createUiStore } from './ui-store'

const UiContext = createContext<UiStore>()

/** Provides the app's UI store (theme). Takes an explicit `store` for tests. */
export function UiProvider(props: ParentProps<{ store?: UiStore }>) {
  const store = props.store ?? createUiStore()
  return <UiContext.Provider value={store}>{props.children}</UiContext.Provider>
}

export function useUi(): UiStore {
  const store = useContext(UiContext)
  if (!store) {
    throw new Error('useUi: sem <UiProvider> acima na árvore (esperado um UiStore no contexto)')
  }
  return store
}
