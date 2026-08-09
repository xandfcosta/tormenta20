import { create } from 'zustand'
import { persist } from 'zustand/middleware'

export type Theme = 'light' | 'dark'

type UiStore = {
  theme: Theme
  toggleTheme: () => void
  setTheme: (theme: Theme) => void
  /** UI sound cues (hover/select/scene transition). Off by default so the app
   *  never surprises with sound — the player opts in from the Hub. */
  sfx: boolean
  toggleSfx: () => void
  setSfx: (on: boolean) => void
}

export const THEME_STORAGE_KEY = 't20-ui'

export const useUiStore = create<UiStore>()(
  persist(
    (set) => ({
      theme: 'light',
      toggleTheme: () =>
        set((s) => ({ theme: s.theme === 'light' ? 'dark' : 'light' })),
      setTheme: (theme) => set({ theme }),
      sfx: false,
      toggleSfx: () => set((s) => ({ sfx: !s.sfx })),
      setSfx: (on) => set({ sfx: on }),
    }),
    { name: THEME_STORAGE_KEY },
  ),
)
