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
 * Os verbos que a LISTA reserva na coluna de ações — a união do que cada linha
 * oferece (ALE-141).
 *
 * O olho só existe em linha com vida e remover só para o mestre, então cada
 * linha tinha um conjunto diferente e a fileira encolhia: o `+` de uma caía
 * onde estava o lápis de outra. Reservando por LISTA, o lugar de cada verbo é
 * o mesmo em todas as linhas e quem não o tem deixa o espaço vazio.
 *
 * É a união, e não "sempre os cinco", porque no rail do jogador ninguém remove
 * nem esconde PV — reservar aquilo ali seria buraco permanente.
 *
 * @example reservedVerbs(entries, { isGm: true, myCharacterIds }) // ['vitals','hide','remove']
 */
export function reservedVerbs(
  entries: readonly InitiativeEntry[],
  viewer: { isGm: boolean; myCharacterIds: ReadonlySet<number> },
): ActionVerb[] {
  const permissoes = entries.map((entry) => entryPermissions(entry, viewer))
  const reservados: ActionVerb[] = []
  if (permissoes.some((can) => can.editVitals)) reservados.push('vitals')
  if (viewer.isGm && entries.some((entry) => entry.hpMax !== undefined)) reservados.push('hide')
  if (permissoes.some((can) => can.remove)) reservados.push('remove')
  return reservados
}

/** Um lugar na coluna de ações. `vitals` são os três de PV, que andam juntos. */
export type ActionVerb = 'vitals' | 'hide' | 'remove'

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

/**
 * Quem está na vez e quem vem depois, na ORDEM DA MESA (ALE-179).
 *
 * A lista é circular: depois do último vem o primeiro, com a rodada seguinte —
 * cortar no fim deixaria a tira vazia justamente no turno em que saber "quem
 * vem depois" mais importa, o último antes de virar a rodada.
 *
 * Fora de combate (`turnIndex` −1) não há vez de ninguém e não há fila.
 *
 * @example upcomingTurns(initiative, 7, 3) // [atual, próximo, seguinte]
 */
export function upcomingTurns(
  initiative: readonly InitiativeEntry[],
  turnIndex: number,
  count: number,
): InitiativeEntry[] {
  if (turnIndex < 0 || initiative.length === 0) return []
  return Array.from({ length: Math.min(count, initiative.length) }, (_, step) => {
    const entry = initiative[(turnIndex + step) % initiative.length]
    if (!entry) throw new Error(`iniciativa sem entrada no passo ${step} (turno ${turnIndex})`)
    return entry
  })
}
