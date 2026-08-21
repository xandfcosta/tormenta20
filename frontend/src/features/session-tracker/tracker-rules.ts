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

/** Quem entra no próximo clique de avançar, e como o botão o anuncia. */
export type NextTurnTarget = { label: string; entry: InitiativeEntry | null }

/**
 * O rótulo do botão mais clicado da sessão (ALE-184).
 *
 * Ele diz PARA ONDE vai, não o que faz: o mestre lia "▶" e contava a lista
 * para saber quem entrava. Fora de combate o verbo muda — "Próximo: Arwen"
 * mentiria sobre uma rodada que ainda não começou, e quem clica ali está
 * COMEÇANDO o combate.
 *
 * A volta é circular como em {@link upcomingTurns}: depois do último vem o
 * primeiro, com a rodada seguinte.
 *
 * @example nextTurnTarget([arwen, ogro], 0) // { label: 'Próximo: Ogro', … }
 */
export function nextTurnTarget(
  initiative: readonly InitiativeEntry[],
  turnIndex: number,
): NextTurnTarget {
  // Lista vazia não tem para onde ir, e prometer um nome seria inventá-lo.
  if (initiative.length === 0) return { label: 'Próximo turno', entry: null }
  const emCombate = turnIndex >= 0
  const entry = initiative[emCombate ? (turnIndex + 1) % initiative.length : 0]
  if (!entry) throw new Error(`iniciativa sem entrada após o turno ${turnIndex}`)
  return { label: `${emCombate ? 'Próximo' : 'Começar'}: ${entry.label}`, entry }
}

/** Abaixo disto o palco da sessão não comporta duas fileiras de cromo. */
const PALCO_BAIXO_PX = 416

/**
 * Se o palco da sessão está BAIXO demais para a faixa de turno em duas
 * fileiras (ALE-146).
 *
 * Medido a 844×390 com uma ficha aberta: o cromo comia 252 dos 390px, 65% da
 * tela, e a faixa de turno sozinha era 90px porque enrolava em duas fileiras —
 * mais que as duas barras de navegação somadas. Sobravam 138px para o
 * conteúdo, dos quais a faixa do combatente já usava 89.
 *
 * Altura e não largura, e é por isso que não é `@media`: 844×390 (celular
 * deitado) e 768×1024 (tablet em pé) são larguras vizinhas com alturas
 * opostas, e o tablet não tem problema nenhum. A regra da casa proíbe consulta
 * de altura em `@media` porque o teclado virtual mexe na altura da JANELA
 * (ALE-176) — aqui a altura medida é a do PALCO, e no único formato onde o
 * teclado abriria ele já está muito abaixo do limiar, então não há chaveamento
 * para o dedo sentir.
 *
 * Altura zero significa "ainda não medi" e responde NÃO, que é o arranjo de
 * sempre.
 *
 * @example palcoBaixo(325) // true — celular deitado
 * @example palcoBaixo(950) // false — tablet em pé
 */
export function palcoBaixo(alturaDoPalco: number): boolean {
  return alturaDoPalco > 0 && alturaDoPalco < PALCO_BAIXO_PX
}
