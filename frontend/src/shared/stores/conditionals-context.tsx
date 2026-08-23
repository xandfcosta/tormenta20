import { type ParentProps, createContext, useContext } from 'solid-js'
import { api } from '@/shared/api/api'
import { toast } from '@/shared/ui/sonner'
import { type ConditionalsStore, createConditionalsStore } from './conditionals-store'

const ConditionalsContext = createContext<ConditionalsStore>()

/**
 * Provê o store dos SITUACIONAIS (Fúria & cia.). Vale para o app inteiro, e o
 * store é indexado por personagem — duas fichas abertas na mesma sessão mantêm
 * os próprios toggles.
 *
 * É AQUI que a escrita ganha destino e o erro ganha voz: o store não importa
 * `api` nem `toast`, porque um store que fala com a tela não se testa sem ela.
 *
 * Recebe um `store` explícito nos testes.
 */
export function ConditionalsProvider(props: ParentProps<{ store?: ConditionalsStore }>) {
  const store =
    props.store ??
    createConditionalsStore(
      (characterId, conditionals) => api.characters.updateConditionals(characterId, conditionals),
      // A recusa PRECISA de voz: o store já desfez a pintura otimista, e sem o
      // aviso o toggle voltaria sozinho e o jogador acharia que errou o clique.
      () => toast.error('Não consegui salvar o situacional — ele voltou ao que estava.'),
    )
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
