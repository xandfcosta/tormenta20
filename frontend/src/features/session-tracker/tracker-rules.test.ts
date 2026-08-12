import { describe, expect, it } from 'vitest'
import type { InitiativeEntry } from '@/shared/realtime/realtime'
import { connectionStatus, entryPermissions, myCharacterIdsOf } from './tracker-rules'

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
