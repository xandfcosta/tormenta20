import { type ParentProps, createContext, useContext } from 'solid-js'
import { type AuthStore, createAuthStore } from './auth-store'

const AuthContext = createContext<AuthStore>()

/**
 * Provides the app's single auth store. Takes an explicit `store` so tests can
 * mount a scene already logged in, instead of reaching for a global.
 *
 * @example <AuthProvider store={createAuthStore(user)}><Scene /></AuthProvider>
 */
export function AuthProvider(props: ParentProps<{ store?: AuthStore }>) {
  const store = props.store ?? createAuthStore()
  return <AuthContext.Provider value={store}>{props.children}</AuthContext.Provider>
}

export function useAuth(): AuthStore {
  const store = useContext(AuthContext)
  if (!store) {
    throw new Error('useAuth: sem <AuthProvider> acima na árvore (esperado um AuthStore no contexto)')
  }
  return store
}
