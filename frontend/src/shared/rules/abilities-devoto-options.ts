import type { Deus } from '@/shared/api/catalog-types'
/**
 * Devoto picker logic — pure filter over a DEUSES list PASSED IN, NO catalog
 * data of its own. Split out of `./deuses` so the frontend can run the same
 * per-class whitelist against its fetched-and-cached deuses without anchoring
 * the DEUSES catalog into the bundle (project_front_decouple_catalog B.3).
 * `./deuses` re-exports the culto sentinels + wraps `devotoOptionsIn` with the
 * real DEUSES.
 *
 * NOTE: the `CULTO_*` string sentinels live HERE (not in `./deuses`) so this
 * module has no runtime import back into `./deuses` — only the erased `Deus`
 * type — keeping the two modules free of an import cycle.
 */
export const CULTO_PANTEAO = 'panteao'
export const CULTO_PALADINO_DO_BEM = 'paladino-do-bem'

const CULTO_PANTEAO_OPTION: Deus = {
  id: CULTO_PANTEAO,
  name: 'Panteão',
  major: false,
  paladinoEligible: false,
  druidaEligible: false,
}
const CULTO_PALADINO_DO_BEM_OPTION: Deus = {
  id: CULTO_PALADINO_DO_BEM,
  name: 'Paladino do Bem',
  major: false,
  paladinoEligible: false,
  druidaEligible: false,
}

/**
 * Filter the given deus list for a class's devoto picker, or null when the
 * class has no devoto slot. Mirrors the per-class lists in PDF Cap 3
 * (Religião): Clérigo — any deus maior + 'Panteão' (p57); Paladino — the
 * 8-deus whitelist + 'Paladino do Bem' (p82); Druida — Allihanna, Megalokk,
 * Oceano (p61), no non-divindade alternative.
 */
export function devotoOptionsIn(
  deuses: readonly Deus[],
  className: string,
): Deus[] | null {
  switch (className) {
    case 'Clérigo':
      return [...deuses.filter((d) => d.major), CULTO_PANTEAO_OPTION]
    case 'Paladino':
      return [
        ...deuses.filter((d) => d.paladinoEligible),
        CULTO_PALADINO_DO_BEM_OPTION,
      ]
    case 'Druida':
      return deuses.filter((d) => d.druidaEligible)
    default:
      return null
  }
}
