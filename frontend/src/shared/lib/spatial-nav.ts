/**
 * Pure geometry core for game-like on-screen navigation (ALE-55). No DOM, no
 * focus, no React — it takes element rectangles as data and answers "which id is
 * next in this direction?" so all the tricky math is unit-testable (jsdom has no
 * layout, so a pure core is the only way to test it). The React glue that reads
 * real rects and moves focus lives in `shared/ui/scene-nav`.
 */

export type Dir = 'up' | 'down' | 'left' | 'right'
export type RegionLayout = 'grid' | 'row' | 'column'

export interface NavRect {
  left: number
  top: number
  right: number
  bottom: number
}

export interface NavCandidate {
  id: string
  rect: NavRect
}

// How strongly perpendicular misalignment is punished vs primary-axis distance,
// so an aligned neighbour beats a nearer-but-diagonal one when moving in a grid.
const CROSS_WEIGHT = 2

const HORIZONTAL: Record<Dir, boolean> = {
  left: true,
  right: true,
  up: false,
  down: false,
}

function centerX(r: NavRect): number {
  return (r.left + r.right) / 2
}

function centerY(r: NavRect): number {
  return (r.top + r.bottom) / 2
}

/** Whether `dir` carries meaning for a region of this layout — a `row` only
 *  answers to ←/→, a `column` only to ↑/↓, a `grid` to all four. */
function dirActive(dir: Dir, layout: RegionLayout): boolean {
  if (layout === 'grid') return true
  return layout === 'row' ? HORIZONTAL[dir] : !HORIZONTAL[dir]
}

/** Is `cand` strictly beyond `from` along the primary axis of `dir`? */
function isAhead(from: NavRect, cand: NavRect, dir: Dir): boolean {
  switch (dir) {
    case 'right':
      return centerX(cand) > centerX(from) + 0.5
    case 'left':
      return centerX(cand) < centerX(from) - 0.5
    case 'down':
      return centerY(cand) > centerY(from) + 0.5
    case 'up':
      return centerY(cand) < centerY(from) - 0.5
  }
}

/** Cost of stepping from `from` to `cand` in `dir`: primary-axis distance plus
 *  weighted perpendicular misalignment. Lower wins. */
function moveCost(from: NavRect, cand: NavRect, dir: Dir): number {
  const dx = Math.abs(centerX(cand) - centerX(from))
  const dy = Math.abs(centerY(cand) - centerY(from))
  const primary = HORIZONTAL[dir] ? dx : dy
  const cross = HORIZONTAL[dir] ? dy : dx
  return primary + CROSS_WEIGHT * cross
}

/**
 * The next item id when moving `dir` from `fromId`, or `null` at the region's
 * edge (no wrap — the caller crosses to a neighbouring region there). Respects
 * the layout, so a list never drifts sideways.
 *
 * @example nextInDirection(cards, 'a', 'right', 'grid') // id of the card to the right
 */
export function nextInDirection(
  items: readonly NavCandidate[],
  fromId: string,
  dir: Dir,
  layout: RegionLayout,
): string | null {
  if (!dirActive(dir, layout)) return null
  const from = items.find((i) => i.id === fromId)
  if (!from) return null
  let best: string | null = null
  let bestCost = Number.POSITIVE_INFINITY
  for (const cand of items) {
    if (cand.id === fromId || !isAhead(from.rect, cand.rect, dir)) continue
    const cost = moveCost(from.rect, cand.rect, dir)
    if (cost < bestCost) {
      bestCost = cost
      best = cand.id
    }
  }
  return best
}

/**
 * The candidate nearest `fromRect` that lies in `dir` — like `nextInDirection`
 * but seeded by a free rectangle instead of a member id (no self to exclude, no
 * layout gate). Used to pick the neighbouring **region** to cross into when a
 * region's own cursor hits its edge. Returns `null` if nothing lies that way.
 */
export function nearestInDirection(
  items: readonly NavCandidate[],
  fromRect: NavRect,
  dir: Dir,
): string | null {
  let best: string | null = null
  let bestCost = Number.POSITIVE_INFINITY
  for (const cand of items) {
    if (!isAhead(fromRect, cand.rect, dir)) continue
    const cost = moveCost(fromRect, cand.rect, dir)
    if (cost < bestCost) {
      bestCost = cost
      best = cand.id
    }
  }
  return best
}

/**
 * The item whose centre is closest to `fromRect`'s centre — used when crossing
 * INTO a region that has no remembered cursor, so entry lands nearest to where
 * the cursor left the previous region. Returns `null` for an empty region.
 */
export function nearestTo(
  items: readonly NavCandidate[],
  fromRect: NavRect,
): string | null {
  let best: string | null = null
  let bestDist = Number.POSITIVE_INFINITY
  for (const cand of items) {
    const dx = centerX(cand.rect) - centerX(fromRect)
    const dy = centerY(cand.rect) - centerY(fromRect)
    const dist = dx * dx + dy * dy
    if (dist < bestDist) {
      bestDist = dist
      best = cand.id
    }
  }
  return best
}
