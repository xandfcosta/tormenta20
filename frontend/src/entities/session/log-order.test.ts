import { describe, expect, it } from 'vitest'
import type { Session } from '@/shared/api/api'
import { orderSessionsForLog } from './log-order'

function fakeSession(overrides: Partial<Session> = {}): Session {
  return {
    id: overrides.sessionNumber ?? 1,
    campaignId: 1,
    title: null,
    sessionNumber: 1,
    notes: null,
    status: 'planned',
    startedAt: null,
    endedAt: null,
    createdAt: '',
    updatedAt: '',
    ...overrides,
  }
}

describe('orderSessionsForLog', () => {
  it('orders strictly newest-first by number (live is NOT hoisted)', () => {
    const list = [
      fakeSession({ sessionNumber: 1, status: 'ended' }),
      fakeSession({ sessionNumber: 3, status: 'planned' }),
      fakeSession({ sessionNumber: 2, status: 'active' }),
    ]
    expect(orderSessionsForLog(list).map((s) => s.sessionNumber)).toEqual([3, 2, 1])
  })

  it('does not mutate the input array', () => {
    const list = [
      fakeSession({ sessionNumber: 1 }),
      fakeSession({ sessionNumber: 2 }),
    ]
    orderSessionsForLog(list)
    expect(list.map((s) => s.sessionNumber)).toEqual([1, 2])
  })
})
