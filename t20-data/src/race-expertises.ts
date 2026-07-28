/**
 * Races that grant "treinado em N perícias livres" at character creation (PDF
 * Cap 1 — race entries). These picks are unrestricted — the book states they
 * "não precisam ser da sua classe" — so they belong to the same free category
 * as the Inteligência bonus, not the class list.
 *
 * Deliberately EXCLUDED:
 *  - Bônus-only grants (Lefou "Deformidade" +2 em duas, Kliren "Vanguardista"
 *    +2 em um Ofício) — those are situational bonuses, not perícia *training*.
 *  - Osteon "Memória Póstuma" — an either/or (uma perícia OU um poder geral OU
 *    herdar habilidade), so not an unconditional perícia grant; capturing it
 *    here would wrongly force a perícia on a player who took another mode.
 */
export const RACE_FREE_EXPERTISES: Record<string, number> = {
  Humano: 2, // Versátil (p26)
  Kliren: 1, // Híbrido (p28)
}

/** Total free (any-perícia) training slots granted by the chosen races. */
export function raceFreeExpertiseCount(raceNames: readonly string[]): number {
  return raceNames.reduce((n, r) => n + (RACE_FREE_EXPERTISES[r] ?? 0), 0)
}
