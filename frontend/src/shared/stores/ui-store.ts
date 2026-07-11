import { create } from 'zustand'
import { persist } from 'zustand/middleware'

export type Theme = 'light' | 'dark'

type UiStore = {
  theme: Theme
  toggleTheme: () => void
  setTheme: (theme: Theme) => void
}

export const THEME_STORAGE_KEY = 't20-ui'

export const useUiStore = create<UiStore>()(
  persist(
    (set) => ({
      theme: 'light',
      toggleTheme: () =>
        set((s) => ({ theme: s.theme === 'light' ? 'dark' : 'light' })),
      setTheme: (theme) => set({ theme }),
    }),
    { name: THEME_STORAGE_KEY },
  ),
)
