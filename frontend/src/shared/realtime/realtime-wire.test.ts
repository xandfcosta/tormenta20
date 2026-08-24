import { existsSync, readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { createRoot } from 'solid-js'
import { describe, expect, it } from 'vitest'
import {
  type BoardState,
  COMMANDS,
  createSessionSocket,
  type SessionRealtime,
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

/** Grava o comando que sobe. Substitui o `WireSocket`: o fio de subida agora é
 *  HTTP, e o que interessa é o par (comando, corpo) — o CAMINHO é comparado
 *  contra o roteador do Go mais abaixo. */
class WireCommands {
  readonly sent: { event: string; body: Record<string, unknown> }[] = []
  readonly run = (
    event: string,
    _ids: { campaignId: number; sessionId: number },
    body: Record<string, unknown>,
  ): Promise<unknown> => {
    this.sent.push({ event, body })
    return Promise.resolve(null)
  }
}

function wireOf(act: (rt: SessionRealtime) => void) {
  const comandos = new WireCommands()
  createRoot((dispose) => {
    const rt = createSessionSocket(
      () => 1,
      () => 7,
      { connect: () => ({ on: () => {}, close: () => {} }), command: comandos.run },
    )
    act(rt)
    dispose()
  })
  if (comandos.sent.length !== 1) {
    throw new Error(
      `a ação devia mandar exatamente 1 comando, mandou ${comandos.sent.length}: ${JSON.stringify(comandos.sent)}`,
    )
  }
  return comandos.sent[0]
}

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
    { entry: { label: 'Ogro', initiative: 14, type: 'npc' } },
  ],
  [
    'edita a linha',
    'initiative-update',
    (rt) => rt.updateEntry('e1', { initiative: 9 }),
    { entryId: 'e1', patch: { initiative: 9 } },
  ],
  ['tira a linha', 'initiative-remove', (rt) => rt.removeEntry('e1'), { entryId: 'e1' }],
  ['passa a vez', 'initiative-next-turn', (rt) => rt.nextTurn(), {}],
  ['volta a vez', 'initiative-previous-turn', (rt) => rt.previousTurn(), {}],
  ['zera a iniciativa', 'initiative-reset', (rt) => rt.resetInitiative(), {}],
  ['traz o grupo', 'initiative-populate', (rt) => rt.populateParty(), {}],
  ['inicia a cena', 'session-scene-start', (rt) => rt.startScene(), {}],
  ['encerra a cena', 'session-scene-end', (rt) => rt.endScene(), {}],
  [
    // O D20, nunca o total (ALE-213): quem soma o bônus da perícia é o Go.
    'o jogador registra a própria',
    'initiative-self',
    (rt) => rt.rollSelfInitiative(42, 17),
    { characterId: 42, d20: 17 },
  ],
  [
    'descansa',
    'session-rest',
    (rt) => rt.rest('day', 'confortavel'),
    { scope: 'day', condition: 'confortavel' },
  ],
  [
    'escreve os vitais',
    'vitals-patch',
    (rt) => rt.patchVitals('e1', { hpCurrent: 12 }),
    { entryId: 'e1', patch: { hpCurrent: 12 } },
  ],
  [
    'soma dano nos vitais',
    'vitals-delta',
    (rt) => rt.deltaVitals('e1', { hpDelta: -3 }),
    { entryId: 'e1', hpDelta: -3 },
  ],
  [
    'aplica um efeito',
    'apply-effect',
    (rt) => rt.applyEffect('e1', 'escudo-da-fe', 'scene'),
    { entryId: 'e1', spellId: 'escudo-da-fe', scope: 'scene' },
  ],
  [
    'abre o tabuleiro',
    'board-open',
    (rt) => rt.openBoard('Taverna', 'madeira'),
    { place: 'Taverna', terrain: 'madeira' },
  ],
  ['encerra o tabuleiro', 'board-close', (rt) => rt.closeBoard(), {}],
  [
    'põe uma peça',
    'board-token-add',
    (rt) => rt.addToken({ label: 'Ogro', x: 2, y: 3, footprint: 2, kind: 'npc' }),
    { label: 'Ogro', x: 2, y: 3, footprint: 2, kind: 'npc' },
  ],
  ['tira a peça', 'board-token-remove', (rt) => rt.removeToken('t1'), { tokenId: 't1' }],
  [
    'move a peça',
    'board-token-update',
    (rt) => rt.updateToken('t1', { x: 5, y: 6 }),
    { tokenId: 't1', patch: { x: 5, y: 6 } },
  ],
  [
    'duplica a peça',
    'board-token-duplicate',
    (rt) => rt.duplicateToken('t1'),
    { tokenId: 't1' },
  ],
  [
    'marca o lugar',
    'board-marker-add',
    (rt) => rt.addMarker({ x: 4, y: 5, text: '1A', color: 'ouro', hidden: true }),
    { x: 4, y: 5, text: '1A', color: 'ouro', hidden: true },
  ],
  [
    'revela o marcador',
    'board-marker-update',
    (rt) => rt.updateMarker('m1', { hidden: false }),
    { markerId: 'm1', patch: { hidden: false } },
  ],
  [
    'apaga o marcador',
    'board-marker-remove',
    (rt) => rt.removeMarker('m1'),
    { markerId: 'm1' },
  ],
  // Quem vem para o mapa viaja por ID (ALE-204): rótulo faria a tela decidir o
  // nome da peça, e é o servidor que numera os repetidos (ALE-192).
  [
    'traz quem o mestre escolheu',
    'board-populate',
    (rt) => rt.populateBoard(['e1', 'e3']),
    { entryIds: ['e1', 'e3'] },
  ],
  [
    'pinta o brejo',
    'board-terrain-paint',
    (rt) => rt.paintTerrain(3, -1, true),
    { x: 3, y: -1, difficult: true },
  ],
  ['lista os lugares', 'board-places', (rt) => void rt.listPlaces(), {}],
  ['reabre o lugar', 'board-reopen', (rt) => rt.reopenPlace(3), { placeId: 3 }],
  [
    'apaga o lugar',
    'board-place-remove',
    (rt) => void rt.removePlace(3),
    { placeId: 3 },
  ],
  ['vê como jogador', 'board-as-player', (rt) => void rt.boardAsPlayer(), {}],
  [
    'abre a cena guardada',
    'board-place-scene',
    (rt) => void rt.placeScene(3),
    { placeId: 3 },
  ],
  [
    'guarda a cena montada',
    'board-place-save',
    (rt) => void rt.savePlace(3, CENA),
    { placeId: 3, scene: CENA },
  ],
  [
    'propõe o movimento',
    'board-move-propose',
    (rt) => rt.proposeMove('t1', [{ x: 1, y: 1 }]),
    { tokenId: 't1', path: [{ x: 1, y: 1 }] },
  ],
  ['confirma o movimento', 'board-move-commit', (rt) => rt.commitMove(9), { version: 9 }],
  ['desfaz o movimento', 'board-move-cancel', (rt) => rt.cancelMove(), {}],
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

/** Sobe da pasta atual até achar o arquivo: o vitest roda ora da raiz do
 *  monorepo, ora da pasta do pacote. */
function goFile(rel: string): string {
  let dir = process.cwd()
  for (let up = 0; up < 6; up++) {
    const candidate = resolve(dir, rel)
    if (existsSync(candidate)) return candidate
    dir = dirname(dir)
  }
  throw new Error(`não achei ${rel} subindo de ${process.cwd()}`)
}

/**
 * As rotas que o Go registra em `mountLiveRoutes`, lidas do fonte — a lista
 * viva. Reconstrói o caminho inteiro juntando os prefixos dos `r.Route`.
 */
function serverRoutes(): Set<string> {
  const source = readFileSync(goFile('engine-go/api/session_commands.go'), 'utf8')
  const corpo = source.slice(source.indexOf('func (s *Server) mountLiveRoutes'))
  const rotas = new Set<string>()
  const prefixos: string[] = []
  for (const linha of corpo.split('\n')) {
    const abre = linha.match(/r\.Route\("([^"]+)"/)
    if (abre) {
      prefixos.push(abre[1])
      continue
    }
    if (linha.trim().startsWith('})')) prefixos.pop()
    const verbo = linha.match(/r\.(Get|Post|Put|Patch|Delete)\("([^"]+)"/)
    if (verbo) {
      const caminho = (prefixos.join('') + (verbo[2] === '/' ? '' : verbo[2])) || '/'
      rotas.add(`${verbo[1].toUpperCase()} ${caminho}`)
    }
  }
  if (rotas.size === 0) throw new Error('nenhuma rota reconhecida — o registro mudou de forma')
  return rotas
}

/** O que o cliente pede, na mesma forma: método + caminho SEM a base. */
function clientRoutes(): Set<string> {
  const marcadores = { entryId: '{entryId}', tokenId: '{tokenId}', markerId: '{markerId}', placeId: '{placeId}' }
  const base = '/api/campaigns/1/sessions/7'
  const out = new Set<string>()
  for (const [event, spec] of Object.entries(COMMANDS)) {
    if (event === 'get-session-state') continue // hidratação, não comando
    const caminho = spec.path({ campaignId: 1, sessionId: 7 }, marcadores).replace(base, '')
    out.add(`${spec.method} ${caminho.replace('/{id}', '')}`)
  }
  return out
}

describe('cliente e servidor falam a mesma língua', () => {
  // A paridade mudou de EIXO com a ALE-253: antes comparava nomes de evento,
  // agora compara ROTAS. E ficou mais forte — um nome de evento errado sumia em
  // silêncio (socket.io ignora evento sem registro), enquanto uma rota errada é
  // um 404 que o servidor conta. O guarda existe para o erro chegar antes.
  const doServidor = () => new Set([...serverRoutes()].map((r) => r.replace('/{id}', '')))

  it('toda rota que o cliente chama existe no roteador', () => {
    const orfas = [...clientRoutes()].filter((r) => !doServidor().has(r))

    expect(orfas).toEqual([])
  })

  it('toda rota do roteador tem quem a chame', () => {
    const semUso = [...doServidor()].filter((r) => !clientRoutes().has(r))

    // Rota no servidor sem cliente é regra escrita e nunca exercida — e é assim
    // que a tabela envelhece sem ninguém notar.
    expect(semUso).toEqual([])
  })
})
