import { describe, expect, it } from 'vitest'
import type { Session, SessionStatus } from '@/shared/api/api'
import { firstActiveSession } from './use-active-session'

/** Minimal Session builder — only `status` matters to the selection rule. */
function session(id: number, status: SessionStatus): Session {
  return {
    id,
    campaignId: 0,
    title: null,
    sessionNumber: id,
    notes: null,
    status,
    startedAt: null,
    endedAt: null,
    createdAt: '',
    updatedAt: '',
  }
}

describe('firstActiveSession', () => {
  it('returns null when there are no campaigns', () => {
    expect(firstActiveSession([])).toBeNull()
  })

  it('returns null when no session is active', () => {
    const lists = [
      { campaignId: 1, sessions: [session(10, 'planned'), session(11, 'ended')] },
    ]
    expect(firstActiveSession(lists)).toBeNull()
  })

  it('returns the active session as a routable ref', () => {
    const lists = [
      { campaignId: 7, sessions: [session(20, 'ended'), session(21, 'active')] },
    ]
    expect(firstActiveSession(lists)).toEqual({ campaignId: 7, sessionId: 21 })
  })

  it('picks the first campaign with a live session, in order', () => {
    const lists = [
      { campaignId: 1, sessions: [session(30, 'ended')] },
      { campaignId: 2, sessions: [session(31, 'active')] },
      { campaignId: 3, sessions: [session(32, 'active')] },
    ]
    expect(firstActiveSession(lists)).toEqual({ campaignId: 2, sessionId: 31 })
  })

  it('skips campaigns whose sessions are still loading (undefined)', () => {
    const lists = [
      { campaignId: 1, sessions: undefined },
      { campaignId: 2, sessions: [session(40, 'active')] },
    ]
    expect(firstActiveSession(lists)).toEqual({ campaignId: 2, sessionId: 40 })
  })
})
