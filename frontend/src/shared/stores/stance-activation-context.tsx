import { type ParentProps, createContext, useContext } from 'solid-js'
import { api } from '@/shared/api/api'
import { toast } from '@/shared/ui/sonner'
import {
  type StanceActivationStore,
  createStanceActivationStore,
} from './stance-activation-store'

const StanceActivationContext = createContext<StanceActivationStore>()

/**
 * Provê o registro do que cada POSTURA custou. Vale para o app inteiro, e é
 * indexado por personagem — duas fichas abertas mantêm os próprios pagamentos.
 *
 * É aqui que a escrita ganha destino e o erro ganha voz: o store não importa
 * `api` nem `toast`, porque um store que fala com a tela não se testa sem ela.
 *
 * Recebe um `store` explícito nos testes.
 */
export function StanceActivationProvider(
  props: ParentProps<{ store?: StanceActivationStore }>,
) {
  const store =
    props.store ??
    createStanceActivationStore(
      {
        set: (characterId, flag, record) => api.characters.setStance(characterId, flag, record),
        clear: (characterId, flag) => api.characters.clearStance(characterId, flag),
      },
      // Perder o registro do pagamento é o pior dos três: sair da postura
      // passaria a DEVOLVER o PM, que é exatamente o que este store existe para
      // impedir.
      () => toast.error('Não consegui salvar a postura — o registro voltou ao que estava.'),
    )
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
