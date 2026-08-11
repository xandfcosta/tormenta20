import { describe, expect, it } from 'vitest'
import { sessionStatusMeta } from './status'

describe('sessionStatusMeta', () => {
  it('marks an active session as live', () => {
    expect(sessionStatusMeta('active')).toEqual({ label: 'Ao vivo', tone: 'live' })
  })

  it('marks an ended session as ended', () => {
    expect(sessionStatusMeta('ended')).toEqual({
      label: 'Encerrada',
      tone: 'ended',
    })
  })

  it('marks a planned session as planned', () => {
    expect(sessionStatusMeta('planned')).toEqual({
      label: 'Planejada',
      tone: 'planned',
    })
  })
})
