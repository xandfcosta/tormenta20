import { createEffect } from 'solid-js'
import type { PlayCue } from '@/shared/lib/sfx'
import type { SessionRuntimeState } from '@/shared/realtime/realtime'

/** O que a vez do jogador dispara: a fala na tela e o sino. */
export type TurnAlert = {
  notify: (label: string) => void
  sfx: PlayCue
}

/**
 * Avisa no instante em que o combatente ativo passa a ser um personagem de
 * quem está olhando. O realce da linha cobre o estado permanente; isto é o
 * alerta transitório, e ele nasce de comparar SNAPSHOTS consecutivos — o
 * realtime entrega o estado inteiro a cada mutação e não tem stream de
 * eventos, então não há `socket.on` para escutar (ALE-180).
 *
 * Dispara na BORDA: enquanto continuar sendo a minha vez, nada se repete.
 *
 * Por que também som (ALE-180): é o único evento do app em que a atenção do
 * jogador está legitimamente longe da tela — ele está olhando o dado, o
 * mestre, o próprio celular. E é som como TERCEIRO canal, nunca único: o
 * toast, o crachá "Sua vez" e o anel na ficha já dizem a mesma coisa.
 *
 * @example createTurnCue(rt.state, myCharacterIds, { notify, sfx })
 */
export function createTurnCue(
  state: () => SessionRuntimeState,
  myCharacterIds: () => ReadonlySet<number>,
  alert: TurnAlert,
): void {
  let wasMyTurn = false
  createEffect(() => {
    const active = activeEntry(state())
    const isMyTurn = active?.characterId !== undefined && myCharacterIds().has(active.characterId)
    if (isMyTurn && !wasMyTurn && active) {
      alert.notify(active.label)
      alert.sfx('turn')
    }
    wasMyTurn = isMyTurn
  })
}

/** `turnIndex` é −1 fora de combate, e aí não há vez de ninguém. */
function activeEntry(state: SessionRuntimeState) {
  return state.turnIndex >= 0 ? state.initiative[state.turnIndex] : undefined
}
