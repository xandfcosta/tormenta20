import { describe, expect, it } from 'vitest'
import type { InitiativeEntry } from '@/shared/realtime/realtime'
import {
  connectionStatus,
  entryPermissions,
  myCharacterIdsOf,
  nextTurnTarget,
  reservedVerbs,
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

describe('entryPermissions', () => {
  const mine = new Set([7])

  it('o mestre edita e remove qualquer combatente', () => {
    const can = entryPermissions(entry(), { isGm: true, myCharacterIds: mine })

    expect(can.editVitals).toBe(true)
    expect(can.remove).toBe(true)
    expect(can.applyEffect).toBe(false)
  })

  it('o mestre aplica efeito só em quem tem ficha', () => {
    const can = entryPermissions(entry({ characterId: 7 }), {
      isGm: true,
      myCharacterIds: mine,
    })

    expect(can.applyEffect).toBe(true)
  })

  // O jogador mexe nos PV do PRÓPRIO personagem — a mesma regra que o servidor
  // aplica; a UI só evita oferecer o que seria recusado.
  it('o jogador edita o próprio e mais nada', () => {
    const meu = entryPermissions(entry({ characterId: 7, type: 'character' }), {
      isGm: false,
      myCharacterIds: mine,
    })
    const alheio = entryPermissions(entry({ characterId: 8, type: 'character' }), {
      isGm: false,
      myCharacterIds: mine,
    })

    expect(meu.editVitals).toBe(true)
    expect(meu.remove).toBe(false)
    expect(alheio.editVitals).toBe(false)
  })

  it('NPC não é de ninguém, então o jogador não mexe', () => {
    const can = entryPermissions(entry(), { isGm: false, myCharacterIds: mine })

    expect(can.editVitals).toBe(false)
  })
})

/**
 * A coluna de ações (ALE-141). O olho só existe em linha com vida e remover só
 * para o mestre, então cada linha tinha um conjunto diferente e a fileira
 * encolhia — o `+` de uma caía onde estava o lápis de outra.
 *
 * A regra é reservar por LISTA, não por linha: o lugar de cada verbo é o mesmo
 * em todas, e quem não o tem deixa o espaço vazio. E é a UNIÃO do que a lista
 * oferece, não os três sempre — no rail do jogador ninguém remove nem esconde
 * PV, e reservar aquilo ali seria buraco permanente.
 */
describe('reservedVerbs', () => {
  const mine = new Set([7])

  it('o mestre reserva os três quando há linha com vida', () => {
    const lista = [entry({ hpMax: 30 }), entry({ id: 'b' })]

    expect(reservedVerbs(lista, { isGm: true, myCharacterIds: mine })).toEqual([
      'vitals',
      'hide',
      'remove',
    ])
  })

  // Esconder PV do que não tem PV não significa nada: sem nenhuma linha com
  // vida, o lugar do olho não se reserva.
  it('sem ninguém com vida, o olho não ocupa lugar', () => {
    const lista = [entry(), entry({ id: 'b' })]

    expect(reservedVerbs(lista, { isGm: true, myCharacterIds: mine })).toEqual([
      'vitals',
      'remove',
    ])
  })

  it('o jogador com personagem na mesa reserva só os vitais', () => {
    const lista = [entry({ characterId: 7, type: 'character', hpMax: 20 }), entry({ id: 'b' })]

    expect(reservedVerbs(lista, { isGm: false, myCharacterIds: mine })).toEqual(['vitals'])
  })

  it('o jogador sem personagem na mesa não reserva nada', () => {
    const lista = [entry({ characterId: 8, type: 'character', hpMax: 20 })]

    expect(reservedVerbs(lista, { isGm: false, myCharacterIds: mine })).toEqual([])
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
    expect(nextTurnTarget([], -1)).toEqual({ label: 'Próximo turno', entry: null })
  })
})
