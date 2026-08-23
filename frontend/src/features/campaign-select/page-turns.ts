import { type Accessor, createComputed, createSignal, untrack } from 'solid-js'
import type { Campaign } from '@/shared/api/api'
import { createMediaQuery, createPrefersReducedMotion } from '@/shared/lib/media-query'

export type Page = { campaign: Campaign; isLive: boolean }

/**
 * `fast`: more picks are still queued behind this turn (rapid navigation) —
 * play it quickly so the sequence drains; the settling turn (empty queue) runs
 * at normal speed.
 */
export type Turn = { from: Page; to: Page; dir: 1 | -1; fast: boolean }

export type PageTurns = {
  shown: Accessor<Page>
  turn: Accessor<Turn | null>
  /** Call when the leaf animation ends, to let the next queued turn start. */
  finishTurn: () => void
}

/**
 * Page-turn state machine with an unbounded queue. `target` is the current
 * pick; `shown` is what's landed. Every new target is appended to the queue
 * (deduped against the last known destination); the book turns to them one at
 * a time — so picks made during an animation all play, in order, never cutting
 * or restarting. Direction comes from the markers' order (`orderIds`): a later
 * marker turns forward, an earlier one backward. Disabled (instant, no leaf) on
 * phones and under reduced motion.
 *
 * Solid port note: the React version had to drive this machine **during
 * render** — a guarded block of `setState` calls in the component body, all
 * using functional updates so an enqueue and a dequeue in the same render
 * composed instead of clobbering, plus a `useRef` to remember the last target
 * across renders. None of that survives here: there is no render pass to
 * piggyback on, so "when the target changes, enqueue it" is literally an effect
 * on the target, and "when idle with work queued, start the next turn" is
 * another. The ref is gone.
 *
 * @example const { shown, turn, finishTurn } = createPageTurns(target, orderIds)
 */
export function createPageTurns(
  target: Accessor<Page>,
  orderIds: Accessor<number[]>,
): PageTurns {
  const wide = createMediaQuery('(min-width: 640px)')
  const reduced = createPrefersReducedMotion()
  const canTurn = () => wide() && !reduced()

  const [shown, setShown] = createSignal<Page>(target())
  const [turn, setTurn] = createSignal<Turn | null>(null)
  const [queue, setQueue] = createSignal<Page[]>([])

  const directionTo = (fromId: number, toId: number): 1 | -1 =>
    orderIds().indexOf(toId) >= orderIds().indexOf(fromId) ? 1 : -1

  // `createComputed`, not `createEffect`: this is derived state, and it has to
  // settle in the SAME update as the pick — an effect runs after the cycle, so
  // the book would lag a tick behind the rail. (React got this for free by
  // driving the machine during render.)
  //
  // Reacts to the PICK only — the reads below are untracked so landing a turn
  // (which writes shown/queue/turn) can't feed back in as a new enqueue.
  createComputed(() => {
    const next = target()
    untrack(() => {
      if (!canTurn()) {
        setShown(next)
        return
      }
      const lastDestination = queue().at(-1) ?? turn()?.to ?? shown()
      if (lastDestination.campaign.id !== next.campaign.id) {
        setQueue((q) => [...q, next])
        return
      }
      // Same chronicle, fresher data — a session just went live, say. Refresh
      // in place: turning a page to itself would be nonsense, but dropping the
      // update leaves the book claiming "Abrir campanha" while the rail already
      // shows the live ember (the React version does exactly that — ALE-78).
      if (queue().length === 0 && !turn()) setShown(next)
    })
  })

  // Idle with work queued → start the next turn.
  createComputed(() => {
    if (turn() !== null) return
    const [next, ...rest] = queue()
    if (!next) return
    untrack(() => {
      setTurn({
        from: shown(),
        to: next,
        dir: directionTo(shown().campaign.id, next.campaign.id),
        fast: rest.length > 0,
      })
      setShown(next)
      setQueue(rest)
    })
  })

  return { shown, turn, finishTurn: () => setTurn(null) }
}
