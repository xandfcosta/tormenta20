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
}

export function readStoredTheme(raw: string | null): Theme {
  if (!raw) return 'light'
  try {
    const parsed = JSON.parse(raw) as { state?: { theme?: unknown } }
    return parsed.state?.theme === 'dark' ? 'dark' : 'light'
  } catch {
    return 'light'
  }
}

/**
 * Theme state. Applies the `dark` class to <html> on every change, so the
 * Tailwind `dark:` variant and the token overrides follow.
 *
 * @example const ui = createUiStore(); ui.toggleTheme()
 */
export function createUiStore(storage: Storage | undefined = globalThis.localStorage): UiStore {
  const [theme, set] = createSignal<Theme>(readStoredTheme(storage?.getItem(STORAGE_KEY) ?? null))

  const setTheme = (next: Theme) => {
    set(next)
    document.documentElement.classList.toggle('dark', next === 'dark')
    storage?.setItem(STORAGE_KEY, JSON.stringify({ state: { theme: next } }))
  }

  return { theme, setTheme, toggleTheme: () => setTheme(theme() === 'dark' ? 'light' : 'dark') }
}
