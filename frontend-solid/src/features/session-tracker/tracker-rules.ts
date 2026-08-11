import type { ConnectionStatus } from '@/shared/ui/connection-chip'
import type { InitiativeEntry } from '@/shared/realtime/realtime'

/**
 * The socket reports only `isConnected` + `error`, but the chip has three
 * states. "Not connected AND no fatal error" is a retry in flight — showing
 * "offline" between attempts would lie about a socket that is coming back.
 *
 * @example connectionStatus(false, null) // 'reconnecting'
 */
export function connectionStatus(isConnected: boolean, error: string | null): ConnectionStatus {
  if (isConnected) return 'connected'
  return error ? 'offline' : 'reconnecting'
}

type MemberLike = { characterId: number; character?: { ownerId: number } | null }

/**
 * Which combatants belong to the viewer. A member's character is the campaign
 * SNAPSHOT (ALE-33) and never appears in the player's own `/characters` list,
 * so ownership is matched through the roster's `ownerId` — the only link back
 * to the person looking at the screen.
 *
 * @example myCharacterIdsOf(members, me.id)
 */
export function myCharacterIdsOf(
  members: readonly MemberLike[],
  userId: number | undefined,
): Set<number> {
  if (userId === undefined) return new Set()
  return new Set(
    members.filter((m) => m.character?.ownerId === userId).map((m) => m.characterId),
  )
}

export type EntryPermissions = {
  editVitals: boolean
  remove: boolean
  applyEffect: boolean
}

/**
 * What the viewer may do to one row. Mirrors the server's rule — the UI only
 * avoids offering what would be refused, it does not decide anything: the GM
 * edits everyone, a player edits their OWN character, and only the GM pushes
 * a buff (onto a row that actually has a sheet).
 *
 * @example entryPermissions(entry, { isGm: false, myCharacterIds })
 */
export function entryPermissions(
  entry: InitiativeEntry,
  viewer: { isGm: boolean; myCharacterIds: ReadonlySet<number> },
): EntryPermissions {
  const isMine =
    entry.characterId !== undefined && viewer.myCharacterIds.has(entry.characterId)
  return {
    editVitals: viewer.isGm || isMine,
    remove: viewer.isGm,
    applyEffect: viewer.isGm && entry.characterId !== undefined,
  }
}
