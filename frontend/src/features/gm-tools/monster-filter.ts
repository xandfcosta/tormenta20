import type { Monster, MonsterTipo } from '@tormenta20/t20-data'
import { createSignal } from 'solid-js'
import { matchesQuery } from '@/shared/lib/fuzzy-filter'
import { clampToRange } from '@/shared/ui/number-input'

/** ND is a 0–20 scale walked in quarter steps. */
export const ND_MIN = 0
export const ND_MAX = 20
export const ND_STEP = 0.25

export type MonsterCriteria = {
  name: string
  /** Empty means EVERY tipo, not none. */
  tipos: ReadonlySet<MonsterTipo>
  ndMin: number
  ndMax: number
}

export const EMPTY_MONSTER_CRITERIA: MonsterCriteria = {
  name: '',
  tipos: new Set<MonsterTipo>(),
  ndMin: ND_MIN,
  ndMax: ND_MAX,
}

/**
 * The bestiary narrowed by name, tipo and ND band, sorted by threat.
 *
 * A plain function rather than the React app's headless TanStack Table: the
 * Solid adapter is v9 with a rewritten API, and wiring it for one client-side
 * filter over 80 rows costs far more than it returns (gotcha #10 of the port).
 * The typo/accent tolerance — the part that actually mattered — is kept via
 * `matchesQuery`.
 *
 * Sorted by ND then name because a GM looks for a THREAT LEVEL first ("o que
 * eu jogo contra um grupo de nível 3?") and only then for a creature.
 *
 * @example filterMonsters(bestiary, { ...EMPTY_MONSTER_CRITERIA, ndMax: 3 })
 */
export function filterMonsters(
  monsters: readonly Monster[],
  criteria: MonsterCriteria,
): Monster[] {
  return monsters
    .filter(
      (m) =>
        matchesQuery([m.name], criteria.name) &&
        (criteria.tipos.size === 0 || criteria.tipos.has(m.tipo)) &&
        m.nd >= criteria.ndMin &&
        m.nd <= criteria.ndMax,
    )
    .sort((a, b) => a.nd - b.nd || a.name.localeCompare(b.name, 'pt-BR'))
}

export type MonsterFilterStore = {
  criteria: () => MonsterCriteria
  setName: (name: string) => void
  setNdMin: (nd: number) => void
  setNdMax: (nd: number) => void
  toggleTipo: (tipo: MonsterTipo) => void
  reset: () => void
}

/**
 * The filter's state, shared by the Bestiário and by the in-session monster
 * add so both narrow identically. `create*` holding state → born once in a
 * component body, never per event (gotcha #17).
 *
 * The ND inputs are CLAMPED on the way in: a typed 999 or a NaN reaching the
 * band would hide every monster and read as an empty bestiary.
 */
export function createMonsterFilter(): MonsterFilterStore {
  const [criteria, setCriteria] = createSignal<MonsterCriteria>(EMPTY_MONSTER_CRITERIA)
  const clampNd = (nd: number) => clampToRange(Number.isFinite(nd) ? nd : 0, ND_MIN, ND_MAX)

  return {
    criteria,
    setName: (name) => setCriteria((prev) => ({ ...prev, name })),
    setNdMin: (nd) => setCriteria((prev) => ({ ...prev, ndMin: clampNd(nd) })),
    setNdMax: (nd) => setCriteria((prev) => ({ ...prev, ndMax: clampNd(nd) })),
    toggleTipo: (tipo) =>
      setCriteria((prev) => {
        const tipos = new Set(prev.tipos)
        if (!tipos.delete(tipo)) tipos.add(tipo)
        return { ...prev, tipos }
      }),
    reset: () => setCriteria(EMPTY_MONSTER_CRITERIA),
  }
}
