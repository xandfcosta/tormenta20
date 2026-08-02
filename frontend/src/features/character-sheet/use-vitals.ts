import { toast } from 'sonner'
import { useQueryClient } from '@tanstack/react-query'
import { useRef } from 'react'
import { useDebouncedCallback } from '@tanstack/react-pacer'
import type { Character, UpdateVitalsInput } from '@/shared/api/api'
import { api } from '@/shared/api/api'
import { invalidateCharacterDependents } from '@/entities/character/character-cache'
import { characterQueryOptions } from '@/entities/character/queries'

/**
 * Optimistic PV/PM controls for the HUD. Snapshots the pre-burst state, lets
 * `setHp`/`setMp` apply the optimistic value, then debounces the network send;
 * on failure it rolls back to the snapshot. The server echoes only the clamped
 * hp/mp (a delta), merged on success.
 *
 * @example
 * const { setHp } = useVitals(character)
 * setHp(character.hpCurrent - 1)
 */
export function useVitals(character: Character) {
  const qc = useQueryClient()
  const queryKey = characterQueryOptions(character.id).queryKey
  const rollbackSnapshot = useRef<Character | undefined>(undefined)

  const sendVitals = useDebouncedCallback(
    async (input: UpdateVitalsInput) => {
      try {
        const delta = await api.characters.updateVitals(character.id, input)
        qc.setQueryData<Character>(queryKey, (prev) =>
          prev
            ? { ...prev, hpCurrent: delta.hpCurrent, mpCurrent: delta.mpCurrent }
            : prev,
        )
        invalidateCharacterDependents(qc, character.id)
      } catch {
        if (rollbackSnapshot.current) {
          qc.setQueryData(queryKey, rollbackSnapshot.current)
          // Silent rollback = the player believes the damage saved (audit).
          toast.error('Falha ao salvar PV/PM — valores revertidos', {
            description: 'Verifique a conexão e aplique de novo.',
          })
        }
      } finally {
        rollbackSnapshot.current = undefined
      }
    },
    { wait: 350 },
  )

  const setVital = (
    field: 'hpCurrent' | 'mpCurrent',
    max: number,
    next: number,
  ) => {
    const clamped = Math.max(0, Math.min(max, next))
    if (clamped === character[field]) return
    qc.cancelQueries({ queryKey })
    if (!rollbackSnapshot.current) {
      rollbackSnapshot.current = qc.getQueryData<Character>(queryKey)
    }
    qc.setQueryData<Character>(queryKey, (prev) =>
      prev ? { ...prev, [field]: clamped } : prev,
    )
    sendVitals({ [field]: clamped })
  }

  const setHp = (next: number) => setVital('hpCurrent', character.hpMax, next)
  const setMp = (next: number) => setVital('mpCurrent', character.mpMax, next)
  return { setHp, setMp }
}
