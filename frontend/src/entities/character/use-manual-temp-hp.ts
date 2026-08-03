import { useMutation, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { api, type Character } from '@/shared/api/api'
import { characterQueryOptions } from './queries'
import { applyPoolResult, isPoolSuperseded } from './temp-hp-pool'

/**
 * Set (or clear, with 0) the GM-entered manual temp-PV pool (F3) via
 * POST :id/active-effects `{ manualTempHp }`. Vale-o-maior lives server-side;
 * a superseded set only toasts (nothing changed), other outcomes patch the
 * cached activeEffects from the returned delta.
 *
 * @example const { setManualTempHp } = useManualTempHp(character); setManualTempHp(12)
 */
export function useManualTempHp(character: Character): {
  setManualTempHp: (value: number) => void
} {
  const qc = useQueryClient()
  const queryKey = characterQueryOptions(character.id).queryKey
  const mutation = useMutation({
    mutationFn: (value: number) =>
      api.characters.applyEffect(character.id, { manualTempHp: value }),
    onSuccess: (result) => {
      if (isPoolSuperseded(result)) {
        toast.info(
          `Pool maior já ativo (${result.keptAmount} PV) — vale o maior (p256)`,
        )
        return
      }
      qc.setQueryData<Character>(queryKey, (prev) =>
        prev ? applyPoolResult(prev, result) : prev,
      )
    },
    onError: () => {
      toast.error('Falha ao definir PV temporários')
    },
  })
  return { setManualTempHp: (value: number) => mutation.mutate(value) }
}
