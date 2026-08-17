import { createEffect, onCleanup } from 'solid-js'
import type { InitiativeEntry } from '@/shared/realtime/realtime'

/** Quanto esperar pelo broadcast antes de concluir que nada entrou. */
const ESPERA_MS = 1500

/**
 * O que "Adicionar grupo" trouxe, dito em palavras (ALE-135).
 *
 * O botão ficava cinza depois do clique e parecia desabilitado, mas continua
 * clicável — e clicar de novo é inofensivo, porque `populateParty` é
 * idempotente. Faltava a tela CONTAR isso.
 *
 * Por que comparar estado em vez de ler uma resposta: `populateParty` é um
 * `send` pelo socket, sem ack. Quem sabe o resultado é o broadcast seguinte, e
 * quando nada entra pode não haver broadcast nenhum — daí a espera, que é o
 * caso "já está tudo aqui".
 *
 * @example const anunciar = createPartyFeedback(() => rt.state().initiative, toast)
 */
export function createPartyFeedback(
  entries: () => readonly InitiativeEntry[],
  notify: (mensagem: string) => void,
): () => void {
  let esperando: Set<string> | null = null
  let timer: ReturnType<typeof setTimeout> | undefined

  const desistir = () => {
    clearTimeout(timer)
    timer = undefined
  }

  createEffect(() => {
    const agora = entries()
    if (!esperando) return
    const novos = agora.filter((entrada) => !esperando?.has(entrada.id)).length
    if (novos === 0) return
    desistir()
    esperando = null
    notify(
      novos === 1
        ? '1 personagem entrou na iniciativa'
        : `${novos} personagens entraram na iniciativa`,
    )
  })

  onCleanup(desistir)

  return () => {
    esperando = new Set(entries().map((entrada) => entrada.id))
    desistir()
    timer = setTimeout(() => {
      if (!esperando) return
      esperando = null
      notify('O grupo já está na iniciativa')
    }, ESPERA_MS)
  }
}
