import { type ParentProps, createContext, useContext } from 'solid-js'
import { api } from '@/shared/api/api'
import { toast } from '@/shared/ui/sonner'
import { type PowerUsesStore, createPowerUsesStore } from './power-uses-store'

const PowerUsesContext = createContext<PowerUsesStore>()

/**
 * Provê os contadores de USO dos poderes limitados. Vale para o app inteiro, e
 * o store é indexado por personagem — duas fichas abertas mantêm as próprias
 * contas.
 *
 * É aqui que a escrita ganha destino e o erro ganha voz: o store não importa
 * `api` nem `toast`, porque um store que fala com a tela não se testa sem ela.
 *
 * Recebe um `store` explícito nos testes.
 */
export function PowerUsesProvider(props: ParentProps<{ store?: PowerUsesStore }>) {
  const store =
    props.store ??
    createPowerUsesStore(
      (characterId, powerId, scope) => api.characters.bumpPowerUse(characterId, powerId, scope),
      // Aqui a voz importa mais que no situacional: o contador voltando sozinho
      // faria o jogador gastar o mesmo poder duas vezes achando que não gastou.
      () => toast.error('Não consegui registrar o uso — a conta voltou ao que estava.'),
    )
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
