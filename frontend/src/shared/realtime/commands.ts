/**
 * O CONTRATO DE FIO da mesa ao vivo (ALE-253).
 *
 * Cada comando é uma ROTA. Antes isto era um nome de evento numa string, e um
 * `board-move-propose` renomeado de um lado passava 100% do vitest e quebrava a
 * mesa em silêncio — socket.io ignora evento sem registro. Agora o mesmo erro é
 * um 404, que o servidor conta.
 *
 * A tabela mora sozinha porque é ela que o `realtime-wire.test.ts` compara com
 * o `mountLiveRoutes` do Go: é um documento de fronteira, e muda quando o
 * SERVIDOR muda — não quando a tela muda.
 */
export type CommandSpec = { method: string; path: (ids: SessionIds, body: CommandBody) => string }
type SessionIds = { campaignId: number; sessionId: number }
type CommandBody = Record<string, unknown>

const base = ({ campaignId, sessionId }: SessionIds) =>
  `/api/campaigns/${campaignId}/sessions/${sessionId}`

export const COMMANDS: Record<string, CommandSpec> = {
  'initiative-add': { method: 'POST', path: (i) => `${base(i)}/initiative` },
  'initiative-self': { method: 'POST', path: (i) => `${base(i)}/initiative/self` },
  'initiative-populate': { method: 'POST', path: (i) => `${base(i)}/initiative/populate` },
  'initiative-next-turn': { method: 'POST', path: (i) => `${base(i)}/initiative/next-turn` },
  'initiative-previous-turn': { method: 'POST', path: (i) => `${base(i)}/initiative/previous-turn` },
  'initiative-reset': { method: 'DELETE', path: (i) => `${base(i)}/initiative` },
  'initiative-update': { method: 'PATCH', path: (i, b) => `${base(i)}/initiative/${b.entryId}` },
  'initiative-remove': { method: 'DELETE', path: (i, b) => `${base(i)}/initiative/${b.entryId}` },
  'vitals-patch': { method: 'PATCH', path: (i, b) => `${base(i)}/initiative/${b.entryId}/vitals` },
  'vitals-delta': { method: 'POST', path: (i, b) => `${base(i)}/initiative/${b.entryId}/vitals/delta` },
  'apply-effect': { method: 'POST', path: (i, b) => `${base(i)}/initiative/${b.entryId}/effects` },
  'session-scene-start': { method: 'POST', path: (i) => `${base(i)}/scene/start` },
  'session-scene-end': { method: 'POST', path: (i) => `${base(i)}/scene/end` },
  'session-rest': { method: 'POST', path: (i) => `${base(i)}/rest` },

  'board-open': { method: 'POST', path: (i) => `${base(i)}/board` },
  'board-close': { method: 'DELETE', path: (i) => `${base(i)}/board` },
  'get-board-state': { method: 'GET', path: (i) => `${base(i)}/board` },
  'board-as-player': { method: 'GET', path: (i) => `${base(i)}/board/as-player` },
  'board-populate': { method: 'POST', path: (i) => `${base(i)}/board/populate` },
  'board-terrain-paint': { method: 'POST', path: (i) => `${base(i)}/board/terrain` },
  'board-curtain': { method: 'POST', path: (i) => `${base(i)}/board/curtain` },
  'board-token-add': { method: 'POST', path: (i) => `${base(i)}/board/tokens` },
  'board-token-update': { method: 'PATCH', path: (i, b) => `${base(i)}/board/tokens/${b.tokenId}` },
  'board-token-remove': { method: 'DELETE', path: (i, b) => `${base(i)}/board/tokens/${b.tokenId}` },
  'board-token-duplicate': { method: 'POST', path: (i, b) => `${base(i)}/board/tokens/${b.tokenId}/duplicate` },
  'board-move-propose': { method: 'POST', path: (i, b) => `${base(i)}/board/tokens/${b.tokenId}/move` },
  'board-move-commit': { method: 'POST', path: (i) => `${base(i)}/board/move/commit` },
  'board-move-cancel': { method: 'POST', path: (i) => `${base(i)}/board/move/cancel` },
  'board-marker-add': { method: 'POST', path: (i) => `${base(i)}/board/markers` },
  'board-marker-update': { method: 'PATCH', path: (i, b) => `${base(i)}/board/markers/${b.markerId}` },
  'board-marker-remove': { method: 'DELETE', path: (i, b) => `${base(i)}/board/markers/${b.markerId}` },
  'board-places': { method: 'GET', path: (i) => `${base(i)}/board/places` },
  'board-reopen': { method: 'POST', path: (i, b) => `${base(i)}/board/places/${b.placeId}/reopen` },
  'board-place-scene': { method: 'GET', path: (i, b) => `${base(i)}/board/places/${b.placeId}/scene` },
  'board-place-save': { method: 'PUT', path: (i, b) => `${base(i)}/board/places/${b.placeId}/scene` },
  'board-place-remove': { method: 'DELETE', path: (i, b) => `${base(i)}/board/places/${b.placeId}` },
  'get-session-state': { method: 'GET', path: (i) => `${base(i)}/state` },
}

/** Manda um comando. O padrão é `fetch`; o teste injeta para observar o fio. */
export async function sendCommand(
  event: string, ids: SessionIds, body: CommandBody,
): Promise<unknown> {
  const spec = COMMANDS[event]
  if (!spec) throw new Error(`comando desconhecido no fio: ${event}`)
  const temCorpo = spec.method !== 'GET' && spec.method !== 'DELETE'
  const resp = await fetch(spec.path(ids, body), {
    method: spec.method,
    credentials: 'include',
    headers: temCorpo ? { 'Content-Type': 'application/json' } : undefined,
    body: temCorpo ? JSON.stringify(body) : undefined,
  })
  if (!resp.ok) throw new Error(await resp.text())
  return resp.status === 204 ? null : resp.json()
}
