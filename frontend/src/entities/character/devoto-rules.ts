import type { Deus } from '@tormenta20/t20-data'
import { devotoTermsTable } from '@/shared/lib/rules-tables-cache'

/**
 * Verdadeiro quando o personagem pode ser devoto de `deus` pelo livro (p96): a
 * raça ou a classe tem de aparecer na linha "Devotos." do deus — exceto Humanos
 * e Clérigos, que podem ser devotos de qualquer um.
 *
 * A linha do livro é texto verbatim (plurais, sentinelas como "Quaisquer"), e a
 * tradução termo → nome do app vem do catálogo servido (ALE-102). Termo
 * desconhecido é IGNORADO, nunca concede acesso em silêncio.
 *
 * Os consumidores tratam a inelegibilidade como AVISO — mesas caseiras negociam.
 *
 * @example devotoEligible(khalmyr, ['Anão'], ['Guerreiro']) // true
 */
export function devotoEligible(
  deus: Deus,
  raceNames: readonly string[],
  classNames: readonly string[],
): boolean {
  if (raceNames.includes('Humano') || classNames.includes('Clérigo')) return true
  const { openTerms, termToNames } = devotoTermsTable()
  const open = new Set(openTerms)
  const owned = new Set([...raceNames, ...classNames])
  for (const term of deus.devotos ?? []) {
    if (open.has(term)) return true
    if ((termToNames[term] ?? []).some((n) => owned.has(n))) return true
  }
  return false
}
