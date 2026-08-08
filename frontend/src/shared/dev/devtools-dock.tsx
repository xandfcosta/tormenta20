import { ReactQueryDevtools } from '@tanstack/react-query-devtools'
import { TanStackRouterDevtools } from '@tanstack/react-router-devtools'
import { type PointerEvent as ReactPointerEvent, useCallback, useRef, useState } from 'react'

// Draggable dock that hosts both TanStack devtools launchers inline. The stock
// launchers pin themselves to screen corners, where they overlapped the app's
// own bottom-corner UI (brand, theme/user controls, the sheet's bottom tab bar)
// and got in the way of browser/automated UI testing. Docking them behind a
// drag handle lets you shove them anywhere; the position persists across
// reloads. Rendering is gated by VITE_DEVTOOLS in __root (prod is always off).
const STORAGE_KEY = 'devtools-dock-pos'

type Pos = { left: number; top: number }

const clamp = (v: number, min: number, max: number): number =>
  Math.max(min, Math.min(max, v))

function readPos(): Pos {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (raw) return JSON.parse(raw) as Pos
  } catch {
    // malformed/denied storage — fall through to the default corner
  }
  return { left: 12, top: Math.max(12, window.innerHeight - 56) }
}

export function DevtoolsDock() {
  const [pos, setPos] = useState<Pos>(readPos)
  const grab = useRef<{ dx: number; dy: number } | null>(null)

  const onPointerDown = useCallback(
    (e: ReactPointerEvent<HTMLButtonElement>) => {
      e.currentTarget.setPointerCapture(e.pointerId)
      grab.current = { dx: e.clientX - pos.left, dy: e.clientY - pos.top }
    },
    [pos],
  )

  const onPointerMove = useCallback((e: ReactPointerEvent<HTMLButtonElement>) => {
    if (!grab.current) return
    setPos({
      left: clamp(e.clientX - grab.current.dx, 0, window.innerWidth - 32),
      top: clamp(e.clientY - grab.current.dy, 0, window.innerHeight - 32),
    })
  }, [])

  const onPointerUp = useCallback(
    (e: ReactPointerEvent<HTMLButtonElement>) => {
      grab.current = null
      e.currentTarget.releasePointerCapture(e.pointerId)
      persist(pos)
    },
    [pos],
  )

  return (
    <div style={{ ...dockStyle, left: pos.left, top: pos.top }}>
      <button
        type="button"
        aria-label="Arrastar devtools"
        title="Arraste para mover as devtools"
        onPointerDown={onPointerDown}
        onPointerMove={onPointerMove}
        onPointerUp={onPointerUp}
        style={handleStyle}
      >
        ⠿
      </button>
      <ReactQueryDevtools buttonPosition="relative" />
      <TanStackRouterDevtools toggleButtonProps={{ style: unpinStyle }} />
    </div>
  )
}

function persist(pos: Pos): void {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(pos))
  } catch {
    // storage denied/quota — position simply won't survive a reload
  }
}

const dockStyle: React.CSSProperties = {
  position: 'fixed',
  zIndex: 99999,
  display: 'flex',
  alignItems: 'center',
  gap: 6,
}

const handleStyle: React.CSSProperties = {
  cursor: 'move',
  touchAction: 'none',
  width: 20,
  height: 20,
  borderRadius: 4,
  border: '1px solid var(--border)',
  background: 'var(--card)',
  color: 'var(--muted-foreground)',
  fontSize: 12,
  lineHeight: '18px',
}

// Neutralise the router devtools' fixed corner pinning so its launcher flows
// inline inside the dock (the RQ launcher already supports buttonPosition).
const unpinStyle: React.CSSProperties = {
  position: 'relative',
  inset: 'auto',
  margin: 0,
}
