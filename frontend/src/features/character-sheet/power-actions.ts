import type { QueryClient } from '@tanstack/solid-query'
import { FLAG_ACTIVATIONS } from '@/shared/rules/flag-activations'
import type { ActivationSpec } from '@/shared/api/catalog-types'
import { allConditionals } from '@/entities/character/derived'
import {
  type PowerUseContext,
  type PowerUseDecision,
  enforcedScopeOf,
  grantPowersForFlag,
  powerUseDecision,
  stanceActivationDecision,
  stanceFlagOf,
  stanceTotalPm,
} from '@/entities/character/power-rules'
import { characterQueryOptions } from '@/entities/character/queries'
import { applyPoolResult, isPoolSuperseded } from '@/entities/character/temp-hp-pool'
import { type Character, api } from '@/shared/api/api'
import type { ConditionalsStore } from '@/shared/stores/conditionals-store'
import type { PowerUsesStore } from '@/shared/stores/power-uses-store'
import type { StanceActivationStore } from '@/shared/stores/stance-activation-store'
import { toast } from '@/shared/ui/sonner'

/**
 * The three local stores a power action touches. Passed in rather than read
 * here: they come from context, which only a component body may read — and
 * this file has to stay callable from an event handler.
 */
export type PowerStores = {
  conditionals: ConditionalsStore
  powerUses: PowerUsesStore
  stanceActivations: StanceActivationStore
}

export type PowerActions = {
  canUse: (spec: ActivationSpec) => PowerUseDecision
  /** One-tap "Usar" for an instant power: debits PM, burns the use. */
  use: (spec: ActivationSpec) => Promise<void>
  /** Enter a stance paying base + steps×stepPm; flips its flag group on. */
  activateStance: (spec: ActivationSpec, steps: number) => Promise<void>
  /** Flip the flag group off + clear the payment record. No refund. */
  deactivateStance: (flag: string) => Promise<void>
}

/**
 * Everything a power does when the player taps it. Stateless between calls —
 * no timer, no rollback snapshot spanning events — so it may be built per
 * event, unlike `createVitalActions`.
 *
 * @example
 * const stores = { conditionals: useConditionals(), ... }
 * await powerActions(queryClient, character, stores).activateStance(furia, 1)
 */
export function powerActions(
  queryClient: QueryClient,
  character: Character,
  stores: PowerStores,
): PowerActions {
  const queryKey = characterQueryOptions(character.id).queryKey
  const entries = () => allConditionals(character, stores.conditionals.active(character.id))

  const mergeCharacter = (transform: (prev: Character) => Character) =>
    queryClient.setQueryData<Character>(queryKey, (prev) => (prev ? transform(prev) : prev))

  /**
   * Optimistic PM debit. The React version had no rollback — a rejected write
   * left the sheet showing PM the character never actually spent.
   */
  const spendPm = async (total: number): Promise<boolean> => {
    const before = character.mpCurrent
    const next = before - total
    mergeCharacter((prev) => ({ ...prev, mpCurrent: next }))
    try {
      const vitals = await api.characters.updateVitals(character.id, { mpCurrent: next })
      mergeCharacter((prev) => ({ ...prev, mpCurrent: vitals.mpCurrent }))
      return true
    } catch {
      mergeCharacter((prev) => ({ ...prev, mpCurrent: before }))
      toast.error('Falha ao gastar PM — o poder não foi ativado')
      return false
    }
  }

  const contextFor = (spec: ActivationSpec): PowerUseContext => ({
    mpCurrent: character.mpCurrent,
    usedScene: stores.powerUses.used(character.id, spec.id).scene,
    usedDay: stores.powerUses.used(character.id, spec.id).day,
    activeFlags: new Set(
      entries().flatMap((e) => (e.active && e.effect.flag ? [e.effect.flag] : [])),
    ),
  })

  const grants = flagGrantEffects(queryClient, character)

  return {
    canUse: (spec) => powerUseDecision(spec, contextFor(spec)),

    use: async (spec) => {
      const decision = powerUseDecision(spec, contextFor(spec))
      if (!decision.ok || spec.pmCost === 'variavel') {
        if (decision.reason) toast.error(`${spec.name}: ${decision.reason}`)
        return
      }
      if (spec.pmCost > 0 && !(await spendPm(spec.pmCost))) return
      const scope = enforcedScopeOf(spec)
      if (scope) stores.powerUses.bump(character.id, spec.id, scope)
      toast.success(`${spec.name} — ${spec.pmCost} PM`)
    },

    activateStance: async (spec, steps) => {
      const flag = stanceFlagOf(spec)
      if (!flag) return
      const decision = stanceActivationDecision(spec, steps, character.mpCurrent)
      if (!decision.ok) {
        toast.error(`${spec.name}: ${decision.reason}`)
        return
      }
      const total = stanceTotalPm(spec, steps)
      if (total > 0 && !(await spendPm(total))) return
      // Same store writes the old plain toggle produced, so HUD badges that
      // read the flag keep working.
      stores.conditionals.setMany(character.id, flagEntryIds(entries(), flag), true)
      stores.stanceActivations.logActivation(character.id, flag, { steps, pmPaid: total })
      await grants.apply(flag)
      toast.success(`${spec.name} ativada — ${total} PM`)
    },

    deactivateStance: async (flag) => {
      stores.conditionals.setMany(character.id, flagEntryIds(entries(), flag), false)
      stores.stanceActivations.clearActivation(character.id, flag)
      await grants.remove(flag)
      toast.success(`${FLAG_ACTIVATIONS[flag]?.name ?? flag} encerrada`)
    },
  }
}

/** Every conditional id raised by one stance flag — they flip together. */
function flagEntryIds(
  entries: readonly { id: string; effect: { flag?: string } }[],
  flag: string,
): string[] {
  return entries.filter((entry) => entry.effect.flag === flag).map((entry) => entry.id)
}

/**
 * Stance-triggered grants are REAL ActiveEffects: entering Fúria applies every
 * owned grant power of that flag (Alma de Bronze → the server computes nível +
 * Força and persists the temp-PV pool), leaving it deletes them. The pool the
 * HUD drains lives in `activeEffects`, so both paths patch the cached
 * character with the returned deltas.
 */
function flagGrantEffects(queryClient: QueryClient, character: Character) {
  const queryKey = characterQueryOptions(character.id).queryKey

  const applyOne = async (powerId: string) => {
    const result = await api.characters.applyEffect(character.id, { powerId })
    // Vale-o-maior (p256): a bigger pool already covers this grant — nothing
    // was written, just tell the player which pool won.
    if (isPoolSuperseded(result)) {
      toast.info(`PV temporários mantidos (${result.keptAmount}) — vale o maior (p256)`)
      return
    }
    queryClient.setQueryData<Character>(queryKey, (prev) =>
      prev ? applyPoolResult(prev, result) : prev,
    )
  }

  const removeOne = async (effectId: number) => {
    await api.characters.removeActiveEffect(character.id, effectId)
    queryClient.setQueryData<Character>(queryKey, (prev) =>
      prev
        ? { ...prev, activeEffects: prev.activeEffects.filter((e) => e.id !== effectId) }
        : prev,
    )
  }

  return {
    apply: async (flag: string) => {
      for (const spec of grantPowersForFlag(character, flag)) {
        // One failed grant must not swallow the stance the player just paid for.
        try {
          await applyOne(spec.id)
        } catch {
          toast.error(`Não foi possível aplicar ${spec.name}`)
        }
      }
    },

    remove: async (flag: string) => {
      const grantIds = new Set(grantPowersForFlag(character, flag).map((s) => s.id))
      for (const effect of character.activeEffects ?? []) {
        if (!grantIds.has(effect.catalogId)) continue
        try {
          await removeOne(effect.id)
        } catch {
          toast.error('Não foi possível remover o efeito da postura')
        }
      }
    },
  }
}
