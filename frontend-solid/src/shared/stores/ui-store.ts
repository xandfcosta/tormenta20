import { type Accessor, createSignal } from 'solid-js'

export type Theme = 'light' | 'dark'

/**
 * Persisted under the SAME key/shape the React app's zustand store used
 * (`t20-ui` → `{ state: { theme } }`) — index.html reads it before mount to
 * paint the right theme, and keeping the shape means a user switching between
 * the two apps during the migration doesn't lose their choice.
 */
const STORAGE_KEY = 't20-ui'

export type UiStore = {
  theme: Accessor<Theme>
  setTheme: (theme: Theme) => void
  toggleTheme: () => void
  /** UI sound cues (hover/select/scene transition). Off by default so the app
   *  never surprises with sound — the player opts in from the Hub. */
  sfx: Accessor<boolean>
  setSfx: (on: boolean) => void
  toggleSfx: () => void
}

type PersistedUi = { state?: { theme?: unknown; sfx?: unknown } }

function parseStored(raw: string | null): PersistedUi['state'] {
  if (!raw) return undefined
  try {
    return (JSON.parse(raw) as PersistedUi).state
  } catch {
    return undefined
  }
}

export function readStoredTheme(raw: string | null): Theme {
  return parseStored(raw)?.theme === 'dark' ? 'dark' : 'light'
}

export function readStoredSfx(raw: string | null): boolean {
  return parseStored(raw)?.sfx === true
}

/**
 * Theme state. Applies the `dark` class to <html> on every change, so the
 * Tailwind `dark:` variant and the token overrides follow.
 *
 * @example const ui = createUiStore(); ui.toggleTheme()
 */
export function createUiStore(storage: Storage | undefined = globalThis.localStorage): UiStore {
  const stored = storage?.getItem(STORAGE_KEY) ?? null
  const [theme, setThemeSignal] = createSignal<Theme>(readStoredTheme(stored))
  const [sfx, setSfxSignal] = createSignal(readStoredSfx(stored))

  const persist = () => {
    storage?.setItem(STORAGE_KEY, JSON.stringify({ state: { theme: theme(), sfx: sfx() } }))
  }

  const setTheme = (next: Theme) => {
    setThemeSignal(next)
    document.documentElement.classList.toggle('dark', next === 'dark')
    persist()
  }

  const setSfx = (on: boolean) => {
    setSfxSignal(on)
    persist()
  }

  return {
    theme,
    setTheme,
    toggleTheme: () => setTheme(theme() === 'dark' ? 'light' : 'dark'),
    sfx,
    setSfx,
    toggleSfx: () => setSfx(!sfx()),
  }
}
