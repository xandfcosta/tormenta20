import type { ActivationSpec } from '@/shared/api/catalog-types'
import { resolveActivationSpec } from '@/entities/character/power-rules'
import type { SheetSearchEntry } from './sheet-search-index'

/**
 * Spec for an owned-ability entry (flat search results + the play list).
 * General/Tormenta powers are outside the activation registry (its id space
 * is class/race/origin/deus) — skip them so a name collision with a class
 * power can't paint a wrong Usar button.
 *
 * @example ownedPowerSpec({ name: 'Fúria', powerId: 'class.barbaro.furia', ... })
 */
export function ownedPowerSpec(
  entry: SheetSearchEntry,
): ActivationSpec | undefined {
  if (entry.source === 'Poder geral' || entry.source === 'Poder da Tormenta') {
    return undefined
  }
  return resolveActivationSpec(entry.name, entry.powerId)
}
