import { QueryClient, QueryClientProvider } from '@tanstack/solid-query'
import { render } from '@solidjs/testing-library'
import { createSignal } from 'solid-js'
import { describe, expect, it, vi } from 'vitest'
import type { SessionRealtime, SessionRuntimeState } from '@/shared/realtime/realtime'
import { createCharacterVitalsSync } from './character-vitals-sync'

/** Um estado de rastreador com um personagem e um NPC. */
function state(hp: number, mp = 30): SessionRuntimeState {
  return {
    initiative: [
      { id: 'a', label: 'Herói', initiative: 12, type: 'pc', characterId: 7, hpCurrent: hp, hpMax: 95, mpCurrent: mp, mpMax: 30 },
      { id: 'b', label: 'Ogro', initiative: 8, type: 'npc', hpCurrent: 130, hpMax: 130 },
    ],
    round: 1,
    turnIndex: 0,
  } as unknown as SessionRuntimeState
}

/** Monta o hook com um socket falso cujo estado o teste controla. */
function setup() {
  const [current, setCurrent] = createSignal(state(95))
  const queryClient = new QueryClient()
  const invalidate = vi.spyOn(queryClient, 'invalidateQueries').mockResolvedValue(undefined)
  const rt = { state: current } as unknown as SessionRealtime

  render(() => (
    <QueryClientProvider client={queryClient}>
      {(() => {
        createCharacterVitalsSync(rt, () => 3)
        return null
      })()}
    </QueryClientProvider>
  ))
  const keys = () => invalidate.mock.calls.map(([arg]) => JSON.stringify(arg?.queryKey))
  return { setCurrent, keys, invalidate }
}

describe('createCharacterVitalsSync', () => {
  // O card "Grupo" NÃO lê a ficha: os PV dele vêm do payload de MEMBROS da
  // campanha. Invalidar só `characters` deixava 52 na iniciativa e 57 no card —
  // exatamente a queixa que originou a ALE-122, e o erro que eu cometi no
  // primeiro conserto.
  it('atualiza a ficha E os membros da campanha quando o PV muda', () => {
    const { setCurrent, keys } = setup()

    setCurrent(state(90))

    expect(keys()).toContain(JSON.stringify(['characters', 7]))
    expect(keys()).toContain(JSON.stringify(['campaigns', 3, 'members']))
  })

  it('segue o PM também', () => {
    const { setCurrent, keys } = setup()

    setCurrent(state(95, 20))

    expect(keys()).toContain(JSON.stringify(['characters', 7]))
  })

  // Passar o turno reemite o estado inteiro; refazer as fichas da mesa a cada
  // turno seria uma tempestade de requisições no meio do combate.
  it('não invalida nada quando os vitais não mudaram', () => {
    const { setCurrent, invalidate } = setup()
    invalidate.mockClear()

    setCurrent(state(95))

    expect(invalidate).not.toHaveBeenCalled()
  })
})
