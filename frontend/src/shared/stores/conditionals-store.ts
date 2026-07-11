import { create } from 'zustand'
import { persist } from 'zustand/middleware'

type CharacterId = number

type ConditionalsState = {
  active: Record<CharacterId, string[]>
  toggle: (characterId: CharacterId, id: string) => void
  /** Batch toggle: set every id in `ids` to `value` for one character. */
  setMany: (characterId: CharacterId, ids: string[], value: boolean) => void
  clear: (characterId: CharacterId) => void
}

export const CONDITIONALS_STORAGE_KEY = 't20-conditionals'

export const useConditionalsStore = create<ConditionalsState>()(
  persist(
    (set) => ({
      active: {},
      toggle: (characterId, id) =>
        set((s) => {
          const current = s.active[characterId] ?? []
          const next = current.includes(id)
            ? current.filter((x) => x !== id)
            : [...current, id]
          return { active: { ...s.active, [characterId]: next } }
        }),
      setMany: (characterId, ids, value) =>
        set((s) => {
          const current = new Set(s.active[characterId] ?? [])
          for (const id of ids) {
            if (value) current.add(id)
            else current.delete(id)
          }
          return {
            active: { ...s.active, [characterId]: Array.from(current) },
          }
        }),
      clear: (characterId) =>
        set((s) => {
          const next = { ...s.active }
          delete next[characterId]
          return { active: next }
        }),
    }),
    { name: CONDITIONALS_STORAGE_KEY },
  ),
)

export function useActiveConditionals(characterId: CharacterId): Set<string> {
  const ids = useConditionalsStore((s) => s.active[characterId] ?? EMPTY)
  return new Set(ids)
}

const EMPTY: string[] = []
