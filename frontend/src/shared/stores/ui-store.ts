import { type Accessor, createSignal } from 'solid-js'

export type Theme = 'light' | 'dark'

/**
 * Persisted under the SAME key/shape the React app's zustand store used
 * (`t20-ui` → `{ state: { theme } }`) — index.html reads it before mount to
 * paint the right theme, and keeping the shape means a user switching between
 * the two apps during the migration doesn't lose their choice.
 */
const STORAGE_KEY = 't20-ui'

/** Cheio por padrão: os cues foram afinados um a um nesse ganho, e o slider só
 *  atenua a partir daí (ALE-180). */
const FULL_VOLUME = 100

export type UiStore = {
  theme: Accessor<Theme>
  setTheme: (theme: Theme) => void
  toggleTheme: () => void
  /** UI sound cues (hover/select/scene transition). Off by default so the app
   *  never surprises with sound — the player opts in from the Hub. */
  sfx: Accessor<boolean>
  setSfx: (on: boolean) => void
  toggleSfx: () => void
  /** 0–100. Alerta que só liga e desliga é alerta que se desliga: quem acha o
   *  sino do "Sua vez" alto abaixa em vez de calar a mesa inteira (ALE-180). */
  volume: Accessor<number>
  setVolume: (percent: number) => void
}

type PersistedUi = { state?: { theme?: unknown; sfx?: unknown; volume?: unknown } }

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

/** Qualquer coisa que não seja um número de 0 a 100 vira volume cheio: um
 *  storage corrompido não pode emudecer a mesa nem estourar o ganho. */
export function readStoredVolume(raw: string | null): number {
  return clampVolume(parseStored(raw)?.volume)
}

function clampVolume(value: unknown): number {
  if (typeof value !== 'number' || Number.isNaN(value)) return FULL_VOLUME
  return Math.min(100, Math.max(0, Math.round(value)))
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
  const [volume, setVolumeSignal] = createSignal(readStoredVolume(stored))

  const persist = () => {
    const state = { theme: theme(), sfx: sfx(), volume: volume() }
    storage?.setItem(STORAGE_KEY, JSON.stringify({ state }))
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

  const setVolume = (percent: number) => {
    setVolumeSignal(clampVolume(percent))
    persist()
  }

  return {
    theme,
    setTheme,
    toggleTheme: () => setTheme(theme() === 'dark' ? 'light' : 'dark'),
    sfx,
    setSfx,
    toggleSfx: () => setSfx(!sfx()),
    volume,
    setVolume,
  }
}
