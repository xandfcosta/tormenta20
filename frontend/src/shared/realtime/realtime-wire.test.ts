import { existsSync, readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { createRoot } from 'solid-js'
import { describe, expect, it } from 'vitest'
import {
  type BoardState,
  createSessionSocket,
  type SessionRealtime,
  type SessionSocket,
} from './realtime'

/**
 * O FORMATO DE FIO (ALE-186, bloco 3).
 *
 * Doze ações do socket não tinham uma asserção sequer: todo componente para num
 * `FakeRealtime` e nunca chega ao nome do evento, então RENOMEAR
 * `board-move-propose` passava 100% do vitest e quebrava a mesa em produção.
 * Não há vermelho possível nas camadas de cima — socket.io simplesmente ignora
 * um evento que ninguém registrou, sem erro e sem ack.
 *
 * Duas metades, e as duas são necessárias:
 *
 * 1. A TABELA prova o que sai do cliente — nome exato e corpo exato.
 * 2. A PARIDADE compara essa tabela com o `sock.On` do gateway Go. Sem ela a
 *    tabela só congelaria o nome errado se as duas pontas já discordassem, e
 *    um evento novo no servidor sem cliente nenhum continuaria invisível.
 */

class WireSocket implements SessionSocket {
  readonly emitted: { event: string; body: unknown }[] = []
  connected = false

  on(): void {}

  emit(event: string, ...args: unknown[]): void {
    this.emitted.push({ event, body: args[0] })
  }

  connect(): void {
    this.connected = true
  }

  disconnect(): void {
    this.connected = false
  }
}

/** Executa UMA ação e devolve o que foi para o fio. */
function wireOf(act: (rt: SessionRealtime) => void): { event: string; body: unknown } {
  const socket = new WireSocket()
  createRoot((dispose) => {
    act(
      createSessionSocket(
        () => 1,
        () => 7,
        { connect: () => socket },
      ),
    )
    dispose()
  })
  const sent = socket.emitted.filter((e) => e.event !== 'leave-session')
  if (sent.length !== 1) {
    throw new Error(
      `a ação devia emitir exatamente 1 evento, emitiu ${sent.length}: ${JSON.stringify(sent)}`,
    )
  }
  return sent[0]
}

const SCOPE = { campaignId: 1, sessionId: 7 }

/** Cena mínima para o `savePlace` — o servidor a saneia, o fio só a carrega. */
const CENA: BoardState = {
  version: 4,
  place: 'Taverna',
  terrain: 'madeira',
  tokens: [{ id: 't1', label: 'Ogro', x: 2, y: 3, footprint: 2, kind: 'npc' }],
}

/** Toda ação do `SessionRealtime`, o evento que ela emite e o corpo completo. */
const WIRE: [string, string, (rt: SessionRealtime) => void, Record<string, unknown>][] = [
  [
    'entra na iniciativa',
    'initiative-add',
    (rt) => rt.addEntry({ label: 'Ogro', initiative: 14, type: 'npc' }),
    { ...SCOPE, entry: { label: 'Ogro', initiative: 14, type: 'npc' } },
  ],
  [
    'edita a linha',
    'initiative-update',
    (rt) => rt.updateEntry('e1', { initiative: 9 }),
    { ...SCOPE, entryId: 'e1', patch: { initiative: 9 } },
  ],
  ['tira a linha', 'initiative-remove', (rt) => rt.removeEntry('e1'), { ...SCOPE, entryId: 'e1' }],
  ['passa a vez', 'initiative-next-turn', (rt) => rt.nextTurn(), SCOPE],
  ['volta a vez', 'initiative-previous-turn', (rt) => rt.previousTurn(), SCOPE],
  ['zera a iniciativa', 'initiative-reset', (rt) => rt.resetInitiative(), SCOPE],
  ['traz o grupo', 'initiative-populate', (rt) => rt.populateParty(), SCOPE],
  ['inicia a cena', 'session-scene-start', (rt) => rt.startScene(), SCOPE],
  ['encerra a cena', 'session-scene-end', (rt) => rt.endScene(), SCOPE],
  [
    // O D20, nunca o total (ALE-213): quem soma o bônus da perícia é o Go.
    'o jogador registra a própria',
    'initiative-self',
    (rt) => rt.rollSelfInitiative(42, 17),
    { ...SCOPE, characterId: 42, d20: 17 },
  ],
  [
    'descansa',
    'session-rest',
    (rt) => rt.rest('day', 'confortavel'),
    { ...SCOPE, scope: 'day', condition: 'confortavel' },
  ],
  [
    'escreve os vitais',
    'vitals-patch',
    (rt) => rt.patchVitals('e1', { hpCurrent: 12 }),
    { ...SCOPE, entryId: 'e1', patch: { hpCurrent: 12 } },
  ],
  [
    'soma dano nos vitais',
    'vitals-delta',
    (rt) => rt.deltaVitals('e1', { hpDelta: -3 }),
    { ...SCOPE, entryId: 'e1', hpDelta: -3 },
  ],
  [
    'aplica um efeito',
    'apply-effect',
    (rt) => rt.applyEffect('e1', 'escudo-da-fe', 'scene'),
    { ...SCOPE, entryId: 'e1', spellId: 'escudo-da-fe', scope: 'scene' },
  ],
  [
    'abre o tabuleiro',
    'board-open',
    (rt) => rt.openBoard('Taverna', 'madeira'),
    { ...SCOPE, place: 'Taverna', terrain: 'madeira' },
  ],
  ['encerra o tabuleiro', 'board-close', (rt) => rt.closeBoard(), SCOPE],
  [
    'põe uma peça',
    'board-token-add',
    (rt) => rt.addToken({ label: 'Ogro', x: 2, y: 3, footprint: 2, kind: 'npc' }),
    { ...SCOPE, label: 'Ogro', x: 2, y: 3, footprint: 2, kind: 'npc' },
  ],
  ['tira a peça', 'board-token-remove', (rt) => rt.removeToken('t1'), { ...SCOPE, tokenId: 't1' }],
  [
    'move a peça',
    'board-token-update',
    (rt) => rt.updateToken('t1', { x: 5, y: 6 }),
    { ...SCOPE, tokenId: 't1', patch: { x: 5, y: 6 } },
  ],
  [
    'duplica a peça',
    'board-token-duplicate',
    (rt) => rt.duplicateToken('t1'),
    { ...SCOPE, tokenId: 't1' },
  ],
  [
    'marca o lugar',
    'board-marker-add',
    (rt) => rt.addMarker({ x: 4, y: 5, text: '1A', color: 'ouro', hidden: true }),
    { ...SCOPE, x: 4, y: 5, text: '1A', color: 'ouro', hidden: true },
  ],
  [
    'revela o marcador',
    'board-marker-update',
    (rt) => rt.updateMarker('m1', { hidden: false }),
    { ...SCOPE, markerId: 'm1', patch: { hidden: false } },
  ],
  [
    'apaga o marcador',
    'board-marker-remove',
    (rt) => rt.removeMarker('m1'),
    { ...SCOPE, markerId: 'm1' },
  ],
  ['traz a iniciativa', 'board-populate', (rt) => rt.populateBoard(), SCOPE],
  [
    'pinta o brejo',
    'board-terrain-paint',
    (rt) => rt.paintTerrain(3, -1, true),
    { ...SCOPE, x: 3, y: -1, difficult: true },
  ],
  ['lista os lugares', 'board-places', (rt) => void rt.listPlaces(), SCOPE],
  ['reabre o lugar', 'board-reopen', (rt) => rt.reopenPlace(3), { ...SCOPE, placeId: 3 }],
  [
    'apaga o lugar',
    'board-place-remove',
    (rt) => void rt.removePlace(3),
    { ...SCOPE, placeId: 3 },
  ],
  ['vê como jogador', 'board-as-player', (rt) => void rt.boardAsPlayer(), SCOPE],
  [
    'abre a cena guardada',
    'board-place-scene',
    (rt) => void rt.placeScene(3),
    { ...SCOPE, placeId: 3 },
  ],
  [
    'guarda a cena montada',
    'board-place-save',
    (rt) => void rt.savePlace(3, CENA),
    { ...SCOPE, placeId: 3, scene: CENA },
  ],
  [
    'propõe o movimento',
    'board-move-propose',
    (rt) => rt.proposeMove('t1', [{ x: 1, y: 1 }]),
    { ...SCOPE, tokenId: 't1', path: [{ x: 1, y: 1 }] },
  ],
  ['confirma o movimento', 'board-move-commit', (rt) => rt.commitMove(9), { ...SCOPE, version: 9 }],
  ['desfaz o movimento', 'board-move-cancel', (rt) => rt.cancelMove(), SCOPE],
]

describe('o formato de fio de toda ação', () => {
  it.each(WIRE)('%s → %s', (_nome, event, act, body) => {
    const sent = wireOf(act)

    expect(sent.event).toBe(event)
    // `toEqual` e não `toMatchObject`: um campo A MAIS também é defeito de fio
    // — o servidor lê o corpo inteiro, e lixo extra é o que esconde um rename
    // pela metade (o campo novo entra, o velho fica).
    expect(sent.body).toEqual(body)
  })
})

/** Sobe da pasta atual até achar o gateway: o vitest roda ora da raiz do
 *  monorepo, ora da pasta do pacote, e um caminho relativo fixo quebraria num
 *  dos dois. */
function gatewayPath(): string {
  let dir = process.cwd()
  for (let up = 0; up < 6; up++) {
    const candidate = resolve(dir, 'engine-go/api/realtime_gateway.go')
    if (existsSync(candidate)) return candidate
    dir = dirname(dir)
  }
  throw new Error(`não achei engine-go/api/realtime_gateway.go subindo de ${process.cwd()}`)
}

/** Os nomes que o gateway Go registra, lidos do fonte — a lista viva. */
function serverEvents(): Set<string> {
  const gateway = gatewayPath()
  const source = readFileSync(gateway, 'utf8')
  const names = [...source.matchAll(/sock\.On\("([a-z-]+)"/g)].map((m) => m[1])
  if (names.length === 0) {
    throw new Error(`nenhum sock.On reconhecido em ${gateway} — o registro mudou de forma`)
  }
  return new Set(names)
}

/** Do transporte ou do ciclo de vida da sala: não são ações do `SessionRealtime`. */
const LIFECYCLE = new Set([
  'join-session',
  'leave-session',
  'get-session-state',
  'get-board-state',
  'disconnect',
])

describe('cliente e servidor falam a mesma língua', () => {
  const clientEvents = new Set(WIRE.map(([, event]) => event))

  it('todo evento que o cliente emite existe no gateway', () => {
    const orphans = [...clientEvents].filter((event) => !serverEvents().has(event))

    // Um nome que o servidor não escuta some em silêncio: socket.io não
    // devolve erro nem ack para evento sem registro.
    expect(orphans).toEqual([])
  })

  it('todo evento do gateway tem quem o chame', () => {
    const unused = [...serverEvents()].filter(
      (event) => !LIFECYCLE.has(event) && !clientEvents.has(event),
    )

    // Handler no servidor sem cliente é regra escrita e nunca exercida — e é
    // assim que a tabela envelhece sem ninguém notar.
    expect(unused).toEqual([])
  })
})
