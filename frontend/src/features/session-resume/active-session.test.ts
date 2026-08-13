import { describe, expect, it } from 'vitest'
import type { Session } from '@/shared/api/api'
import { type CampaignSessions, firstActiveSession } from './active-session'

function session(id: number, status: Session['status']): Session {
  return {
    id,
    campaignId: 1,
    title: null,
    sessionNumber: id,
    notes: null,
    status,
    startedAt: null,
    endedAt: null,
    createdAt: '2026-01-01T00:00:00.000Z',
    updatedAt: '2026-01-01T00:00:00.000Z',
  }
}

const noLiveSession: CampaignSessions = {
  campaignId: 1,
  sessions: [session(1, 'ended'), session(2, 'planned')],
}

describe('firstActiveSession', () => {
  it('acha a sessão ao vivo', () => {
    const lists: CampaignSessions[] = [{ campaignId: 3, sessions: [session(9, 'active')] }]
    expect(firstActiveSession(lists)).toEqual({ campaignId: 3, sessionId: 9 })
  })

  it('null quando nenhuma está ao vivo', () => {
    expect(firstActiveSession([noLiveSession])).toBeNull()
  })

  it('null sem crônica nenhuma', () => {
    expect(firstActiveSession([])).toBeNull()
  })

  // Lista undefined = ainda carregando. Pular evita o "Continuar sessão"
  // piscar na tela enquanto as queries chegam.
  it('pula crônicas ainda carregando', () => {
    const lists: CampaignSessions[] = [
      { campaignId: 1, sessions: undefined },
      { campaignId: 2, sessions: [session(5, 'active')] },
    ]
    expect(firstActiveSession(lists)).toEqual({ campaignId: 2, sessionId: 5 })
  })

  it('a primeira crônica em ordem vence quando há mais de uma ao vivo', () => {
    const lists: CampaignSessions[] = [
      { campaignId: 1, sessions: [session(4, 'active')] },
      { campaignId: 2, sessions: [session(8, 'active')] },
    ]
    expect(firstActiveSession(lists)).toEqual({ campaignId: 1, sessionId: 4 })
  })

  it('ignora encerradas e planejadas dentro da mesma crônica', () => {
    const lists: CampaignSessions[] = [
      { campaignId: 1, sessions: [session(1, 'ended'), session(2, 'active'), session(3, 'planned')] },
    ]
    expect(firstActiveSession(lists)).toEqual({ campaignId: 1, sessionId: 2 })
  })
})
