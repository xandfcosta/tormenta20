import { FLAG_ACTIVATIONS } from '@tormenta20/t20-data'
import type { ConditionalEntry } from '@/entities/character/derived'

/** One row of the "Situação" list: a lone conditional, or every conditional
 *  sharing a `flag` folded into a single toggle. */
export type ConditionalGroup =
  | { kind: 'single'; entry: ConditionalEntry }
  | {
      kind: 'flag'
      flag: string
      label: string
      source: string
      entries: ConditionalEntry[]
    }

/**
 * Folds conditional entries that share a `flag` into one toggle row, so a
 * homebrew item's multi-modifier group activates together — as separate rows
 * the player could leave half the effect switched on.
 *
 * @example groupConditionals(allConditionals(character, active))
 */
export function groupConditionals(entries: ConditionalEntry[]): ConditionalGroup[] {
  const byFlag = new Map<string, ConditionalEntry[]>()
  const groups: ConditionalGroup[] = []
  for (const entry of entries) {
    const flag = entry.effect.flag
    if (!flag) {
      groups.push({ kind: 'single', entry })
      continue
    }
    byFlag.set(flag, [...(byFlag.get(flag) ?? []), entry])
  }
  for (const [flag, list] of byFlag) {
    groups.push({
      kind: 'flag',
      flag,
      label: list[0].effect.note,
      source: list[0].effect.source,
      entries: list,
    })
  }
  return groups
}

/**
 * What "Situação" shows: everything except the power stances registered in
 * FLAG_ACTIVATIONS, whose on-switch lives in the Poderes block (they cost PM
 * and end through their own Encerrar). Item flag groups (homebrew-*) and every
 * non-flag conditional keep their toggle here.
 */
export function situationalGroups(entries: ConditionalEntry[]): ConditionalGroup[] {
  return groupConditionals(entries).filter(
    (group) => group.kind === 'single' || !FLAG_ACTIVATIONS[group.flag],
  )
}
