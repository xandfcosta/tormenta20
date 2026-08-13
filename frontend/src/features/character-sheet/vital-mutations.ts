import type { QueryClient } from '@tanstack/solid-query'
import { invalidateCharacterDependents } from '@/entities/character/character-cache'
import { characterQueryOptions } from '@/entities/character/queries'
import {
  applyDrainToEffects,
  applyPoolResult,
  drainPlan,
  isPoolSuperseded,
  reconcileDamageResult,
  routeDamage,
  tempHpPool,
} from '@/entities/character/temp-hp-pool'
import { type Character, type UpdateVitalsInput, api } from '@/shared/api/api'
import { toast } from '@/shared/ui/sonner'

/** PV/PM live in [0, max]; the server clamps too, this just avoids the round-trip. */
export function clampVital(value: number, max: number): number {
  return Math.max(0, Math.min(max, value))
}

/**
 * Optimistic mirror of the server's damage routing: drain the temp-PV pools
 * first, then take the overflow off PV, floored at 0 (dying is the server's
 * call — the sheet never paints negative PV).
 *
 * @example predictDamage(barbaro, 14).hpCurrent // 46, com 10 de pool drenado
 */
export function predictDamage(character: Character, amount: number): Character {
  const pool = tempHpPool(character)
  const { toPool, toHp } = routeDamage(amount, pool.total)
  return {
    ...character,
    hpCurrent: Math.max(0, character.hpCurrent - toHp),
    activeEffects: applyDrainToEffects(character.activeEffects, drainPlan(pool.slices, toPool)),
  }
}

export type VitalActions = {
  /** Atomic temp-first damage — ONE request, the server routes it. */
  applyDamage: (amount: number) => Promise<void>
  /** Paints immediately; the network send is debounced across a click burst. */
  setHp: (next: number) => void
  setMp: (next: number) => void
  /** The GM's ad-hoc temp-PV pool; 0 clears it. */
  setManualTempHp: (value: number) => Promise<void>
}

export type VitalActionsOptions = {
  /** Debounce window for the PV/PM burst, in ms. */
  wait?: number
}

/**
 * The HUD's vitals writes. `create…`, not `…Actions` like `itemActions`: this
 * one OWNS state between calls (the debounce timer and the pre-burst rollback
 * snapshot), so it must be created ONCE — in a component body, which Solid runs
 * a single time. Calling it per event would hand every click a fresh timer and
 * silently defeat the debounce.
 *
 * Takes the character as an accessor so the clamp reads the CURRENT maxes
 * (level-up changes them mid-session).
 *
 * @example
 * const vitals = createVitalActions(queryClient, () => props.character)
 * vitals.setHp(props.character.hpCurrent - 1)
 */
export function createVitalActions(
  queryClient: QueryClient,
  character: () => Character,
  options: VitalActionsOptions = {},
): VitalActions {
  const wait = options.wait ?? 350
  const queryKey = () => characterQueryOptions(character().id).queryKey
  const cached = () => queryClient.getQueryData<Character>(queryKey())

  /** The character as it was BEFORE the current burst — what a failure restores. */
  let rollback: Character | undefined
  let timer: ReturnType<typeof setTimeout> | undefined
  let pending: UpdateVitalsInput = {}

  const send = async () => {
    const input = pending
    pending = {}
    const key = queryKey()
    try {
      const delta = await api.characters.updateVitals(character().id, input)
      queryClient.setQueryData<Character>(key, (prev) =>
        prev ? { ...prev, hpCurrent: delta.hpCurrent, mpCurrent: delta.mpCurrent } : prev,
      )
      invalidateCharacterDependents(queryClient, character().id)
    } catch {
      if (rollback) queryClient.setQueryData(key, rollback)
      // A silent rollback would leave the player believing the damage saved.
      toast.error('Falha ao salvar PV/PM — valores revertidos', {
        description: 'Verifique a conexão e aplique de novo.',
      })
    } finally {
      rollback = undefined
    }
  }

  // Deliberately NOT `createCharacterWrite`: this is the only sheet write with a
  // debounced burst. Three clicks send ONE PUT, and a failure rolls back to the
  // value from before the whole burst — not to a snapshot per click, which is
  // exactly what the shared helper takes. Keep the two apart.
  const setVital = (field: 'hpCurrent' | 'mpCurrent', max: number, next: number) => {
    const clamped = clampVital(next, max)
    const current = cached() ?? character()
    if (clamped === current[field]) return

    queryClient.cancelQueries({ queryKey: queryKey() })
    rollback ??= cached()
    queryClient.setQueryData<Character>(queryKey(), (prev) =>
      prev ? { ...prev, [field]: clamped } : prev,
    )

    pending = { ...pending, [field]: clamped }
    if (timer) clearTimeout(timer)
    timer = setTimeout(send, wait)
  }

  return {
    setHp: (next) => setVital('hpCurrent', character().hpMax, next),
    setMp: (next) => setVital('mpCurrent', character().mpMax, next),

    applyDamage: async (amount) => {
      const key = queryKey()
      await queryClient.cancelQueries({ queryKey: key })
      const previous = cached()
      if (previous) queryClient.setQueryData<Character>(key, predictDamage(previous, amount))
      try {
        const result = await api.characters.applyDamage(character().id, amount)
        queryClient.setQueryData<Character>(key, (prev) =>
          prev ? reconcileDamageResult(prev, result) : prev,
        )
        invalidateCharacterDependents(queryClient, character().id)
      } catch (failure) {
        if (previous) queryClient.setQueryData(key, previous)
        toast.error('Falha ao aplicar dano — a ficha voltou ao valor anterior')
        throw failure
      }
    },

    setManualTempHp: async (value) => {
      const key = queryKey()
      try {
        const result = await api.characters.applyEffect(character().id, { manualTempHp: value })
        if (isPoolSuperseded(result)) {
          // Nothing was written: a bigger pool wins (vale-o-maior, p256).
          toast.info(`Pool maior já ativo (${result.keptAmount} PV) — vale o maior (p256)`)
          return
        }
        queryClient.setQueryData<Character>(key, (prev) =>
          prev ? applyPoolResult(prev, result) : prev,
        )
      } catch (failure) {
        toast.error('Falha ao definir PV temporários')
        throw failure
      }
    },
  }
}
