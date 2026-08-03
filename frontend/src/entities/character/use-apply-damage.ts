import { useMutation, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { api, type Character } from '@/shared/api/api'
import { characterQueryOptions } from './queries'
import {
  applyDrainToEffects,
  drainPlan,
  reconcileDamageResult,
  routeDamage,
  tempHpPool,
} from './temp-hp-pool'

/**
 * Atomic damage via POST :id/damage (F2): ONE request routes the hit
 * temp-first server-side. The optimistic patch reuses the pure
 * `routeDamage`/`drainPlan` prediction; the response reconciles hpCurrent +
 * activeEffects (server wins). Healing stays on `useVitals.setHp`.
 *
 * @example const { applyDamage } = useApplyDamage(character); applyDamage(7)
 */
export function useApplyDamage(character: Character): {
  applyDamage: (amount: number) => void
} {
  const qc = useQueryClient()
  const queryKey = characterQueryOptions(character.id).queryKey
  const mutation = useMutation({
    mutationFn: (amount: number) => api.characters.applyDamage(character.id, amount),
    onMutate: async (amount) => {
      await qc.cancelQueries({ queryKey })
      qc.setQueryData<Character>(queryKey, (prev) =>
        prev ? predictDamage(prev, amount) : prev,
      )
    },
    onSuccess: (result) => {
      qc.setQueryData<Character>(queryKey, (prev) =>
        prev ? reconcileDamageResult(prev, result) : prev,
      )
    },
    onError: () => {
      toast.error('Falha ao aplicar dano — recarregando a ficha')
      qc.invalidateQueries({ queryKey })
    },
  })
  return { applyDamage: (amount: number) => mutation.mutate(amount) }
}

/** Optimistic mirror of the server routing: drain pools, floor hp at 0. */
function predictDamage(prev: Character, amount: number): Character {
  const pool = tempHpPool(prev)
  const { toPool, toHp } = routeDamage(amount, pool.total)
  return {
    ...prev,
    hpCurrent: Math.max(0, prev.hpCurrent - toHp),
    activeEffects: applyDrainToEffects(
      prev.activeEffects,
      drainPlan(pool.slices, toPool),
    ),
  }
}
