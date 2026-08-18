import { createRoot, createSignal } from 'solid-js'
import { describe, expect, it } from 'vitest'
import type { SfxName } from '@/shared/lib/sfx-player'
import type { InitiativeEntry, SessionRuntimeState } from '@/shared/realtime/realtime'
import { createTurnCue } from './turn-cue'

const linha = (id: string, characterId?: number) =>
  ({ id, label: id, characterId }) as unknown as InitiativeEntry

const combate = (turnIndex: number): SessionRuntimeState => ({
  initiative: [linha('Ogro'), linha('Paladino Sagrado', 7)],
  round: 1,
  turnIndex,
})

/**
 * `createEffect` roda DEPOIS do corpo, então o teste cede um microtique entre
 * mexer no sinal e conferir — medir na mesma linha mede o estado anterior.
 */
async function naRaiz(corpo: () => Promise<void>): Promise<void> {
  let descartar = () => {}
  const feito = createRoot((dispose) => {
    descartar = dispose
    return corpo()
  })
  await feito
  descartar()
}

function mesa(inicio: SessionRuntimeState) {
  const [state, setState] = createSignal(inicio)
  const ditos: string[] = []
  const tocados: SfxName[] = []
  createTurnCue(state, () => new Set([7]), {
    notify: (label) => ditos.push(label),
    sfx: (name) => tocados.push(name),
  })
  return { setState, ditos, tocados }
}

/**
 * O "Sua vez" é o único evento do app em que a atenção do jogador está longe da
 * tela, e por isso é o único que ganha som (ALE-180). O alerta nasce de comparar
 * SNAPSHOTS consecutivos: o realtime entrega o estado inteiro a cada mutação e
 * não tem stream de eventos para escutar.
 */
describe('createTurnCue', () => {
  it('toca o sino no instante em que a vez vira minha', async () => {
    await naRaiz(async () => {
      const { setState, ditos, tocados } = mesa(combate(0))

      await Promise.resolve()
      setState(combate(1))

      await Promise.resolve()
      expect(ditos).toEqual(['Paladino Sagrado'])
      expect(tocados).toEqual(['turn'])
    })
  })

  // Borda, não estado: o realtime reenvia o snapshot inteiro a cada mutação da
  // mesa (um PV ajustado, alguém que conectou), e cada um desses reenvios
  // tocaria o sino de novo no meio do MEU turno.
  it('não repete enquanto continua sendo a minha vez', async () => {
    await naRaiz(async () => {
      const { setState, tocados } = mesa(combate(1))

      await Promise.resolve()
      setState({ ...combate(1), round: 2 })

      await Promise.resolve()
      expect(tocados).toEqual(['turn'])
    })
  })

  it('cala na vez dos outros, e volta a tocar quando a minha chega de novo', async () => {
    await naRaiz(async () => {
      const { setState, tocados } = mesa(combate(1))

      await Promise.resolve()
      setState(combate(0))
      await Promise.resolve()
      expect(tocados).toEqual(['turn'])

      setState(combate(1))
      await Promise.resolve()
      expect(tocados).toEqual(['turn', 'turn'])
    })
  })

  // `turnIndex` −1 é "fora de combate": não é a vez de ninguém.
  it('não toca fora de combate', async () => {
    await naRaiz(async () => {
      const { setState, ditos, tocados } = mesa(combate(-1))

      await Promise.resolve()
      setState({ ...combate(-1), round: 2 })

      await Promise.resolve()
      expect(ditos).toEqual([])
      expect(tocados).toEqual([])
    })
  })
})
