import { describe, expect, it } from 'vitest'
import type { Session } from '@/shared/api/api'
import {
  activeSessionByCampaign,
  campaignInitials,
  roleLabel,
} from './campaign-select-helpers'

/** Named fake session — only the fields the selection rule reads matter. */
function fakeSession(id: number, status: Session['status']): Session {
  return {
    id,
    campaignId: 1,
    title: null,
    sessionNumber: id,
    notes: null,
    status,
    startedAt: null,
    endedAt: null,
    createdAt: '2026-08-10T00:00:00.000Z',
    updatedAt: '2026-08-10T00:00:00.000Z',
  }
}

describe('campaignInitials', () => {
  it('takes up to two words, uppercased', () => {
    expect(campaignInitials('A Lâmina de Arton')).toBe('AL')
    expect(campaignInitials('Tauron')).toBe('T')
    expect(campaignInitials('   ')).toBe('?')
  })
})

describe('roleLabel', () => {
  it('maps gm to Mestrando and everything else to Jogando', () => {
    expect(roleLabel('gm')).toBe('Mestrando')
    expect(roleLabel('player')).toBe('Jogando')
    expect(roleLabel(undefined)).toBe('Jogando')
  })
})

describe('activeSessionByCampaign', () => {
  it('maps each campaign with a live session to that session id', () => {
    const map = activeSessionByCampaign([
      { campaignId: 5, sessions: [fakeSession(12, 'active')] },
      { campaignId: 6, sessions: [fakeSession(20, 'ended')] },
    ])
    expect(map).toEqual({ 5: 12 })
  })

  it('skips loading (undefined) lists and campaigns with no live session', () => {
    const map = activeSessionByCampaign([
      { campaignId: 5, sessions: undefined },
      { campaignId: 6, sessions: [] },
    ])
    expect(map).toEqual({})
  })
})
