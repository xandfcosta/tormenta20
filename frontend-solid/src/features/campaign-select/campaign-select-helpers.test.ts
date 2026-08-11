import { describe, expect, it } from 'vitest'
import type { Session } from '@/shared/api/api'
import { type CampaignSessions, activeSessionByCampaign } from './campaign-select-helpers'

function session(id: number, status: Session['status'], campaignId = 1): Session {
  return {
    id,
    campaignId,
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

describe('activeSessionByCampaign', () => {
  it('mapeia crônica → id da sessão ao vivo', () => {
    const lists: CampaignSessions[] = [{ campaignId: 5, sessions: [session(12, 'active')] }]
    expect(activeSessionByCampaign(lists)).toEqual({ 5: 12 })
  })

  it('crônica sem sessão ao vivo fica fora do mapa', () => {
    const lists: CampaignSessions[] = [
      { campaignId: 1, sessions: [session(1, 'ended'), session(2, 'planned')] },
    ]
    expect(activeSessionByCampaign(lists)).toEqual({})
  })

  // Lista undefined = ainda carregando; incluir mostraria brasa errada no rail.
  it('pula crônicas ainda carregando', () => {
    const lists: CampaignSessions[] = [
      { campaignId: 1, sessions: undefined },
      { campaignId: 2, sessions: [session(7, 'active')] },
    ]
    expect(activeSessionByCampaign(lists)).toEqual({ 2: 7 })
  })

  it('lida com várias crônicas ao vivo ao mesmo tempo', () => {
    const lists: CampaignSessions[] = [
      { campaignId: 1, sessions: [session(4, 'active')] },
      { campaignId: 2, sessions: [session(9, 'ended')] },
      { campaignId: 3, sessions: [session(11, 'active')] },
    ]
    expect(activeSessionByCampaign(lists)).toEqual({ 1: 4, 3: 11 })
  })

  it('mapa vazio sem crônica nenhuma', () => {
    expect(activeSessionByCampaign([])).toEqual({})
  })
})
