import type { QueryClient } from '@tanstack/solid-query'
import type { Character } from '@/shared/api/api'
import { useConditionals } from '@/shared/stores/conditionals-context'
import { usePowerUses } from '@/shared/stores/power-uses-context'
import { useStanceActivations } from '@/shared/stores/stance-activation-context'
import { type PowerActions, powerActions } from './power-actions'

/**
 * Reads the three stores a power action needs — they come from context, which
 * only a component body may read — and hands back a builder the event handlers
 * can call with the CURRENT character.
 *
 * Call it in the component body; call the result per event. Passing the
 * character at call time (rather than closing over it here) is what keeps a
 * handler from paying PM against a stale pool.
 *
 * @example
 * const actions = usePowerActions()
 * <button onClick={() => actions(queryClient, props.character).use(spec)} />
 */
export function usePowerActions(): (
  queryClient: QueryClient,
  character: Character,
) => PowerActions {
  const stores = {
    conditionals: useConditionals(),
    powerUses: usePowerUses(),
    stanceActivations: useStanceActivations(),
  }
  return (queryClient, character) => powerActions(queryClient, character, stores)
}
