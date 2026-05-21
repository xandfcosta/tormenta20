import { create } from 'zustand'
import type { AuthUser } from '@/lib/api'

type AuthState = {
  user: AuthUser | null
  setUser: (user: AuthUser | null) => void
}

export const useAuthStore = create<AuthState>((set) => ({
  user: null,
  setUser: (user) => set({ user }),
}))
