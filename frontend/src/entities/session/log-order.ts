import type { Session } from '@/shared/api/api'

/**
 * Orders sessions for the chronicle log: strictly newest-first by session
 * number, so the timeline reads as a monotonic sequence (…5, 4, 3…) instead of
 * jumping. The live session isn't hoisted — it's already highlighted in place,
 * and hoisting it broke the visible order (e.g. 4, 5, 3). Pure and non-mutating
 * — returns a new array so the query cache's list is untouched.
 *
 * @example orderSessionsForLog([{sessionNumber:1},{sessionNumber:3},{sessionNumber:2}])
 *   // #3, #2, #1
 */
export function orderSessionsForLog(sessions: readonly Session[]): Session[] {
  return [...sessions].sort((a, b) => b.sessionNumber - a.sessionNumber)
}
