import { createEffect, onCleanup } from 'solid-js'
import {
  type Dir,
  type NavCandidate,
  type NavRect,
  type RegionLayout,
  nearestInDirection,
  nearestTo,
  nextInDirection,
} from '@/shared/lib/spatial-nav'
import type { SfxName } from '@/shared/lib/sfx-player'
import { createMediaQuery } from '@/shared/lib/media-query'

/** Input-agnostic navigation intent — a keyboard keydown or (later) a gamepad
 *  button both resolve to one of these before the scene reacts. */
export type NavCommand =
  | { type: 'move'; dir: Dir }
  | { type: 'activate' }
  | { type: 'back' }
  | { type: 'bumper'; dir: 'prev' | 'next' }
  | { type: 'edge'; to: 'first' | 'last' }

export interface SceneNavOptions {
  /** The scene root to scope regions to (e.g. `[data-tome-root]`). */
  root: () => HTMLElement | null
  /** Leave the scene — Esc from the rail / top level. */
  onEscape: () => void
  /** Section switch from anywhere (PageUp/PageDown). */
  bumpers?: { prev: () => void; next: () => void }
  /** Scenes with a bespoke cursor (selection-driven, e.g. the roster) set
   *  `delegated` and map the standard grammar here; return true when handled.
   *  This is the gamepad-ready seam. */
  onCommand?: (cmd: NavCommand) => boolean
  /** Delegated-scene escape hatch for keyboard-only accelerators (D, /, O) and
   *  typing-aware Esc — runs before the command mapping, even while typing. The
   *  scene calls preventDefault as needed and returns true when it consumed the
   *  key. */
  onKey?: (e: KeyboardEvent) => boolean
  delegated?: boolean
  sfx: (name: SfxName) => void
  /** Turn nav off while the scene is loading/errored (media gate still applies). */
  active?: () => boolean
}

const DESKTOP = '(min-width: 1280px) and (pointer: fine)'
const FOCUSABLE =
  'a[href], button:not([disabled]), [tabindex]:not([tabindex="-1"]), [data-nav-item]'

/**
 * One window-level keyboard driver for a grimório scene (ALE-55). Scenes declare
 * **regions** in the DOM (`data-nav-region`, `data-nav-layout="grid|row|column"`)
 * and this moves real focus by geometry inside a region, crosses to the
 * neighbour at an edge, runs PageUp/Down bumpers, Enter (native), and Esc
 * (up one level). Keyboard is additive — every item stays a real clickable
 * control. Enabled only on laptop/desktop with a fine pointer.
 *
 * Solid port (ALE-66): everything below the shell is plain DOM code and came
 * over unchanged. What went away is the React bookkeeping — `useRef(opts)`
 * existed only so the effect closure wouldn't read stale props across
 * re-renders, and Solid has no re-render, so `opts` is read directly.
 *
 * Still listens in the **capture** phase: Kobalte's tab list does roving arrow
 * focus just like Radix did, so the driver has to pre-empt it (the issue
 * assumed this could go away — it can't).
 *
 * @example
 * createSceneNav({ root: () => el(), onEscape: toHub, sfx })
 */
export function createSceneNav(opts: SceneNavOptions): void {
  // Region → last-focused item, so re-entering a region restores its cursor.
  const memory: Memory = new WeakMap<HTMLElement, HTMLElement>()
  const desktop = createMediaQuery(DESKTOP)

  createEffect(() => {
    if (opts.active?.() === false || !desktop()) return
    const onKey = (e: KeyboardEvent) => handleKey(e, opts, memory)
    window.addEventListener('keydown', onKey, true)
    onCleanup(() => window.removeEventListener('keydown', onKey, true))
  })
}

type Memory = WeakMap<HTMLElement, HTMLElement>

function handleKey(e: KeyboardEvent, opts: SceneNavOptions, memory: Memory): void {
  if (e.ctrlKey || e.metaKey || e.altKey) return
  if (hasOpenOverlay()) return
  const root = opts.root()
  if (!root) return
  const active = document.activeElement as HTMLElement | null
  const region = active?.closest<HTMLElement>('[data-nav-region]') ?? null
  const delegated = opts.delegated || region?.dataset.navMode === 'delegated'

  // Delegated (selection-model) scenes: bespoke keys first — even while typing,
  // so search Esc-to-clear works — then the standard grammar via onCommand.
  if (delegated) {
    if (opts.onKey?.(e)) return
    if (isTypingTarget()) return
    const cmd = toCommand(e)
    if (!cmd) return
    if (opts.onCommand?.(cmd)) {
      stop(e)
      return
    }
    // Unhandled Esc still backs out one level (leave the scene).
    if (cmd.type === 'back') {
      stop(e)
      opts.onEscape()
    }
    return
  }
  if (isTypingTarget()) return
  const cmd = toCommand(e)
  if (!cmd) return
  route(e, cmd, opts, region, root, memory)
}

function route(
  e: KeyboardEvent,
  cmd: NavCommand,
  opts: SceneNavOptions,
  region: HTMLElement | null,
  root: HTMLElement,
  memory: Memory,
): void {
  switch (cmd.type) {
    case 'back':
      handleBack(e, opts, region, root, memory)
      return
    case 'bumper':
      handleBumper(e, opts, cmd.dir)
      return
    case 'edge':
      handleEdge(e, opts, region, cmd.to, memory)
      return
    case 'move':
      handleMove(e, opts, region, cmd.dir, root, memory)
      return
    case 'activate':
      return // real a/button handle Enter natively
  }
}

function handleMove(
  e: KeyboardEvent,
  opts: SceneNavOptions,
  region: HTMLElement | null,
  dir: Dir,
  root: HTMLElement,
  memory: Memory,
): void {
  if (!region) {
    // Enter the scene only from a blank focus. If the user Tabbed onto a control
    // that lives outside any nav region (a form field, a HUD button), leave the
    // arrows to native behaviour instead of yanking the cursor to the spine.
    const el = document.activeElement
    if (!el || el === document.body) enterScene(e, opts, root, memory)
    return
  }
  const els = itemsOf(region)
  const active = document.activeElement as HTMLElement
  const from = els.indexOf(active)
  const next =
    from === -1 ? null : nextInDirection(candidates(els), String(from), dir, layoutOf(region))
  if (next !== null) {
    focusItem(els[Number(next)], region, opts, memory, 'hover')
    stop(e)
    return
  }
  const neighbour = neighbourRegion(region, dir, rectOf(active), root)
  if (neighbour) {
    enterRegion(neighbour, rectOf(active), opts, memory, 'select')
    stop(e)
  }
}

function handleEdge(
  e: KeyboardEvent,
  opts: SceneNavOptions,
  region: HTMLElement | null,
  to: 'first' | 'last',
  memory: Memory,
): void {
  if (!region) return
  const els = itemsOf(region)
  if (els.length === 0) return
  focusItem(to === 'first' ? els[0] : els[els.length - 1], region, opts, memory, 'hover')
  stop(e)
}

function handleBack(
  e: KeyboardEvent,
  opts: SceneNavOptions,
  region: HTMLElement | null,
  root: HTMLElement,
  memory: Memory,
): void {
  stop(e)
  const rail = root.querySelector<HTMLElement>('[data-nav-region="rail"]')
  if (region && rail && region !== rail) {
    enterRegion(rail, rectOf(region), opts, memory, 'back')
    return
  }
  opts.onEscape()
}

function handleBumper(e: KeyboardEvent, opts: SceneNavOptions, dir: 'prev' | 'next'): void {
  if (!opts.bumpers) return
  stop(e)
  if (dir === 'next') opts.bumpers.next()
  else opts.bumpers.prev()
  opts.sfx('select')
  // After the section re-renders, land on the new active marker so ↓ dives in.
  requestAnimationFrame(() => {
    const rail = opts.root()?.querySelector<HTMLElement>('[data-nav-region="rail"]')
    rail?.querySelector<HTMLElement>(ACTIVE_TAB)?.focus()
  })
}

/** First arrow with focus outside any region: enter the scene at its spine. */
function enterScene(
  e: KeyboardEvent,
  opts: SceneNavOptions,
  root: HTMLElement,
  memory: Memory,
): void {
  const start =
    root.querySelector<HTMLElement>('[data-nav-region="rail"]') ??
    root.querySelector<HTMLElement>('[data-nav-region]')
  if (!start) return
  enterRegion(start, ZERO_RECT, opts, memory, 'select')
  stop(e)
}

/** Move the cursor into `region`: its remembered item, else the active tab (a
 *  rail), else the item nearest to where the cursor left. */
function enterRegion(
  region: HTMLElement,
  fromRect: NavRect,
  opts: SceneNavOptions,
  memory: Memory,
  cue: SfxName,
): void {
  const remembered = memory.get(region)
  if (remembered && region.contains(remembered) && hasArea(remembered)) {
    focusItem(remembered, region, opts, memory, cue)
    return
  }
  const activeTab = region.querySelector<HTMLElement>(ACTIVE_TAB)
  if (activeTab) {
    focusItem(activeTab, region, opts, memory, cue)
    return
  }
  const els = itemsOf(region)
  if (els.length === 0) return
  const id = nearestTo(candidates(els), fromRect)
  if (id !== null) focusItem(els[Number(id)], region, opts, memory, cue)
}

function neighbourRegion(
  region: HTMLElement,
  dir: Dir,
  fromRect: NavRect,
  root: HTMLElement,
): HTMLElement | null {
  const others = Array.from(root.querySelectorAll<HTMLElement>('[data-nav-region]')).filter(
    (r) => r !== region && hasArea(r),
  )
  const override = region.getAttribute(`data-nav-edge-${dir}`)
  if (override) return others.find((r) => r.dataset.navRegion === override) ?? null
  const cands = others.map((r, i) => ({ id: String(i), rect: rectOf(r) }))
  const id = nearestInDirection(cands, fromRect, dir)
  return id === null ? null : others[Number(id)]
}

function focusItem(
  el: HTMLElement,
  region: HTMLElement,
  opts: SceneNavOptions,
  memory: Memory,
  cue: SfxName,
): void {
  el.focus()
  memory.set(region, el)
  opts.sfx(cue)
  el.scrollIntoView({ block: 'nearest', inline: 'nearest' })
}

// --- DOM helpers --------------------------------------------------------------

const ZERO_RECT: NavRect = { left: 0, top: 0, right: 0, bottom: 0 }

// Kobalte marks the selected tab `data-selected`; Radix used
// `data-state="active"`. Both are matched so a scene still mid-port works.
const ACTIVE_TAB = '[role="tab"][data-selected], [role="tab"][data-state="active"]'

/** Focusable items that belong to THIS region (not a nested one), are visible,
 *  and aren't the tabpanel wrapper (a `tabindex=0` container, not a stop). */
function itemsOf(region: HTMLElement): HTMLElement[] {
  return Array.from(region.querySelectorAll<HTMLElement>(FOCUSABLE)).filter(
    (el) =>
      el.closest('[data-nav-region]') === region &&
      el.getAttribute('role') !== 'tabpanel' &&
      !el.hasAttribute('data-nav-skip') &&
      hasArea(el),
  )
}

function candidates(els: readonly HTMLElement[]): NavCandidate[] {
  return els.map((el, i) => ({ id: String(i), rect: rectOf(el) }))
}

function layoutOf(region: HTMLElement): RegionLayout {
  const l = region.dataset.navLayout
  return l === 'grid' || l === 'row' ? l : 'column'
}

function rectOf(el: HTMLElement): NavRect {
  const r = el.getBoundingClientRect()
  return { left: r.left, top: r.top, right: r.right, bottom: r.bottom }
}

function hasArea(el: HTMLElement): boolean {
  const r = el.getBoundingClientRect()
  return r.width > 0 && r.height > 0
}

function isTypingTarget(): boolean {
  const el = document.activeElement
  if (!(el instanceof HTMLElement)) return false
  return (
    el.tagName === 'INPUT' ||
    el.tagName === 'TEXTAREA' ||
    el.tagName === 'SELECT' ||
    el.isContentEditable
  )
}

/** A modal surface owns the keyboard while open — stand down. Kobalte flags an
 *  open overlay with `data-expanded` (Radix used `data-state="open"`). */
function hasOpenOverlay(): boolean {
  return !!document.querySelector(
    '[role="dialog"][data-expanded], [role="menu"][data-expanded], [role="listbox"][data-expanded]',
  )
}

function toCommand(e: KeyboardEvent): NavCommand | null {
  switch (e.key) {
    case 'ArrowUp':
      return { type: 'move', dir: 'up' }
    case 'ArrowDown':
      return { type: 'move', dir: 'down' }
    case 'ArrowLeft':
      return { type: 'move', dir: 'left' }
    case 'ArrowRight':
      return { type: 'move', dir: 'right' }
    case 'Enter':
      return { type: 'activate' }
    case 'Escape':
      return { type: 'back' }
    case 'PageUp':
      return { type: 'bumper', dir: 'prev' }
    case 'PageDown':
      return { type: 'bumper', dir: 'next' }
    case 'Home':
      return { type: 'edge', to: 'first' }
    case 'End':
      return { type: 'edge', to: 'last' }
    default:
      return null
  }
}

function stop(e: KeyboardEvent): void {
  e.preventDefault()
  e.stopPropagation()
}
