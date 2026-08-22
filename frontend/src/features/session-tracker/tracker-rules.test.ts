import { describe, expect, it } from 'vitest'
import type { InitiativeEntry } from '@/shared/realtime/realtime'
import {
  connectionStatus,
  myCharacterIdsOf,
  nextTurnTarget,
  palcoBaixo,
  reservaOOlho,
  turnCounterLabel,
} from './tracker-rules'

const entry = (overrides: Partial<InitiativeEntry> = {}): InitiativeEntry => ({
  id: 'a',
  label: 'Goblin',
  initiative: 12,
  type: 'npc',
  ...overrides,
})

describe('connectionStatus', () => {
  it('conectado é conectado', () => {
    expect(connectionStatus(true, null)).toBe('connected')
  })

  // Sem conexão e SEM erro é tentativa em curso — mostrar "offline" no
  // intervalo entre tentativas mentiria sobre um socket que vai voltar.
  it('sem conexão e sem erro é reconectando', () => {
    expect(connectionStatus(false, null)).toBe('reconnecting')
  })

  it('erro do servidor é offline de verdade', () => {
    expect(connectionStatus(false, 'Sem permissão')).toBe('offline')
  })
})

describe('myCharacterIdsOf', () => {
  const members = [
    { characterId: 1, character: { ownerId: 10 } },
    { characterId: 2, character: { ownerId: 20 } },
    { characterId: 3, character: null },
  ]

  // O personagem do membro é o SNAPSHOT da campanha, que não aparece na lista
  // /characters do jogador — casar por dono é o único jeito de saber o que é
  // meu (ALE-33).
  it('reconhece o personagem pelo dono, não pela lista do jogador', () => {
    expect([...myCharacterIdsOf(members, 10)]).toEqual([1])
  })

  it('sem usuário identificado, nada é meu', () => {
    expect(myCharacterIdsOf(members, undefined).size).toBe(0)
  })

  it('membro sem personagem não entra', () => {
    expect([...myCharacterIdsOf(members, 30)]).toEqual([])
  })
})

/**
 * A coluna de ações (ALE-141), reduzida pela ALE-213.
 *
 * A coluna inteira passou a ser do MESTRE — a fila do jogador virou leitura —,
 * então curar/ferir/editar/remover existem em TODAS as linhas dele e não têm o
 * que reservar. O que continua condicional é o OLHO: esconder PV só faz sentido
 * em linha com vida, e sem a reserva a fileira encolhia na linha sem vida — o
 * `+` de uma caía onde estava o lápis de outra, 36px de deslocamento medidos.
 */
describe('reservaOOlho', () => {
  it('reserva quando alguma linha tem vida', () => {
    expect(reservaOOlho([entry({ hpMax: 30 }), entry({ id: 'b' })], true)).toBe(true)
  })

  // Esconder PV do que não tem PV não significa nada.
  it('sem ninguém com vida, o olho não ocupa lugar', () => {
    expect(reservaOOlho([entry(), entry({ id: 'b' })], true)).toBe(false)
  })

  it('para o jogador não há coluna nenhuma a reservar', () => {
    expect(reservaOOlho([entry({ hpMax: 30 })], false)).toBe(false)
  })
})

describe('nextTurnTarget', () => {
  const arwen = entry({ id: 'a', label: 'Arwen' })
  const ogro = entry({ id: 'b', label: 'Ogro' })
  const zumbi = entry({ id: 'c', label: 'Zumbi 1' })

  it('anuncia QUEM entra, não o que o botão faz', () => {
    expect(nextTurnTarget([arwen, ogro, zumbi], 0).label).toBe('Próximo: Ogro')
  })

  it('no último da lista, volta ao primeiro', () => {
    // Circular como a tira do jogador (ALE-179): é no último turno da rodada
    // que "quem vem depois" mais importa, e cortar ali deixaria o botão mudo.
    expect(nextTurnTarget([arwen, ogro, zumbi], 2).label).toBe('Próximo: Arwen')
  })

  it('fora de combate o verbo é COMEÇAR, não "próximo"', () => {
    // "Próximo: Arwen" mentiria sobre uma rodada que não começou — quem clica
    // ali está começando o combate, e o primeiro da lista é quem entra.
    expect(nextTurnTarget([arwen, ogro], -1).label).toBe('Começar: Arwen')
  })

  it('sem ninguém na lista, não inventa um nome', () => {
    expect(nextTurnTarget([], -1)).toEqual({ label: 'Ninguém na fila', entry: null })
  })
})

/**
 * A frase que diz ONDE a sessão está (ALE-210).
 *
 * São quatro estados exclusivos e a ordem entre eles é a regra: a cena existe
 * antes da fila, e a fila existe antes do turno. É unitário porque é
 * exatamente isso que carrega decisão — a composição na tela se prova em
 * `pages/sessions/turn-bar.test.tsx`, onde o botão que acompanha a frase muda
 * junto.
 */
describe('turnCounterLabel', () => {
  const fila = [entry({ id: 'a' }), entry({ id: 'b' })]

  it('sem cena, a rodada nem é mencionada', () => {
    // "Rodada 0" antes de o mestre iniciar dizia que algo estava em curso.
    expect(turnCounterLabel({ sceneActive: false, round: 0, turnIndex: -1, initiative: [] })).toBe(
      'Fora de cena',
    )
  })

  it('iniciada e ainda vazia, diz o que falta', () => {
    // O primeiro instante do fluxo: iniciar abre a gaveta para montar a ordem.
    expect(turnCounterLabel({ sceneActive: true, round: 0, turnIndex: -1, initiative: [] })).toBe(
      'Em cena · ninguém na fila',
    )
  })

  it('com fila e sem turno, conta quem está na fila', () => {
    expect(
      turnCounterLabel({ sceneActive: true, round: 0, turnIndex: -1, initiative: fila }),
    ).toBe('Rodada 0 · 2 na fila')
  })

  it('em combate, volta a ser a posição na rodada', () => {
    expect(turnCounterLabel({ sceneActive: true, round: 3, turnIndex: 1, initiative: fila })).toBe(
      'Rodada 3 · Turno 2/2',
    )
  })
})

/**
 * O limiar que decide se a faixa de turno cabe em duas fileiras (ALE-146).
 *
 * É regra e não pintura: a mesma resposta governa o que fica na fileira E o
 * que aparece dentro do menu da sessão, que nasce num portal fora do palco.
 */
describe('palcoBaixo', () => {
  it('diz sim no palco do celular deitado', () => {
    expect(palcoBaixo(325)).toBe(true)
  })

  it('diz não no palco do tablet em pé', () => {
    expect(palcoBaixo(950)).toBe(false)
  })

  /** Antes da primeira medição a resposta tem de ser o arranjo de sempre. */
  it('diz não enquanto não mediu', () => {
    expect(palcoBaixo(0)).toBe(false)
  })
})
