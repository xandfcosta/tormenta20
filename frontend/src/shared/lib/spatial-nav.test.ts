import { describe, expect, it } from 'vitest'
import {
  type NavCandidate,
  nearestInDirection,
  nearestTo,
  nextInDirection,
} from './spatial-nav'

// A 2-column × 3-row grid (gap 20). Columns at x[0..100] / [120..220];
// rows at y[0..50] / [70..120] / [140..190].
//   a b
//   c d
//   e f
const grid: NavCandidate[] = [
  { id: 'a', rect: { left: 0, top: 0, right: 100, bottom: 50 } },
  { id: 'b', rect: { left: 120, top: 0, right: 220, bottom: 50 } },
  { id: 'c', rect: { left: 0, top: 70, right: 100, bottom: 120 } },
  { id: 'd', rect: { left: 120, top: 70, right: 220, bottom: 120 } },
  { id: 'e', rect: { left: 0, top: 140, right: 100, bottom: 190 } },
  { id: 'f', rect: { left: 120, top: 140, right: 220, bottom: 190 } },
]

describe('nextInDirection — grid', () => {
  it('moves right/down/left/up to the aligned neighbour', () => {
    expect(nextInDirection(grid, 'a', 'right', 'grid')).toBe('b')
    expect(nextInDirection(grid, 'a', 'down', 'grid')).toBe('c')
    expect(nextInDirection(grid, 'd', 'left', 'grid')).toBe('c')
    expect(nextInDirection(grid, 'd', 'up', 'grid')).toBe('b')
  })

  it('clamps (returns null) at every edge — no wrap', () => {
    expect(nextInDirection(grid, 'a', 'up', 'grid')).toBeNull()
    expect(nextInDirection(grid, 'a', 'left', 'grid')).toBeNull()
    expect(nextInDirection(grid, 'f', 'down', 'grid')).toBeNull()
    expect(nextInDirection(grid, 'f', 'right', 'grid')).toBeNull()
  })

  it('prefers the aligned neighbour over a nearer diagonal', () => {
    // from a, right: b (aligned) beats d (same distance but a row off)
    expect(nextInDirection(grid, 'a', 'right', 'grid')).toBe('b')
  })

  it('follows a ragged last row (2→1 column reflow)', () => {
    const ragged = grid.slice(0, 5) // a b c d e (no f)
    expect(nextInDirection(ragged, 'd', 'down', 'grid')).toBe('e')
  })
})

describe('nextInDirection — layout gating', () => {
  it('row ignores the vertical axis', () => {
    expect(nextInDirection(grid, 'a', 'down', 'row')).toBeNull()
    expect(nextInDirection(grid, 'a', 'up', 'row')).toBeNull()
    expect(nextInDirection(grid, 'a', 'right', 'row')).toBe('b')
  })

  it('column ignores the horizontal axis', () => {
    expect(nextInDirection(grid, 'a', 'right', 'column')).toBeNull()
    expect(nextInDirection(grid, 'a', 'left', 'column')).toBeNull()
    expect(nextInDirection(grid, 'a', 'down', 'column')).toBe('c')
  })
})

describe('nextInDirection — guards', () => {
  it('returns null for an unknown fromId', () => {
    expect(nextInDirection(grid, 'zzz', 'down', 'grid')).toBeNull()
  })

  it('returns null for an empty item set', () => {
    expect(nextInDirection([], 'a', 'down', 'grid')).toBeNull()
  })
})

describe('nearestInDirection', () => {
  // Three regions laid out like the tome: rail (left), header (top), content.
  const regions: NavCandidate[] = [
    { id: 'rail', rect: { left: 0, top: 60, right: 80, bottom: 400 } },
    { id: 'header', rect: { left: 100, top: 0, right: 900, bottom: 50 } },
    { id: 'content', rect: { left: 100, top: 60, right: 900, bottom: 400 } },
  ]

  it('crosses left from content to the rail', () => {
    const fromCard = { left: 120, top: 100, right: 200, bottom: 140 }
    expect(nearestInDirection(regions, fromCard, 'left')).toBe('rail')
  })

  it('crosses up from content to the header', () => {
    const fromCard = { left: 300, top: 70, right: 380, bottom: 110 }
    expect(nearestInDirection(regions, fromCard, 'up')).toBe('header')
  })

  it('returns null when nothing lies that way', () => {
    const belowEverything = { left: 300, top: 420, right: 380, bottom: 460 }
    expect(nearestInDirection(regions, belowEverything, 'down')).toBeNull()
  })
})

describe('nearestTo', () => {
  it('picks the item whose centre is closest to a probe rect', () => {
    const probe = { left: 118, top: 138, right: 122, bottom: 142 } // near f
    expect(nearestTo(grid, probe)).toBe('f')
  })

  it('returns null for an empty region', () => {
    expect(nearestTo([], { left: 0, top: 0, right: 1, bottom: 1 })).toBeNull()
  })
})
