import { RACES_CATALOG } from './abilities/races'

/**
 * First race (abilities-catalog name) in `raceNames` that owns the Deformidade
 * ability (Lefou, p23), or undefined. Split out of `./deformidade` — it's the
 * ONLY thing there that reads RACES_CATALOG, so isolating it lets the rest of
 * deformidade (`deformidadeAvailablePowers` et al.) tree-shake free of the
 * abilities catalog in the frontend bundle (project_front_decouple_catalog
 * B.3). The front reads the fetched catalog via `abilities-cache` instead.
 */
export function raceWithDeformidade(
  raceNames: readonly string[],
): string | undefined {
  const owners = new Set(
    RACES_CATALOG.filter((r) => r.hasDeformidade === true).map((r) => r.name),
  )
  return raceNames.find((n) => owners.has(n))
}
