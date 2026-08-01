/**
 * Devoto eligibility (book p96): to be devoto of a god, your race or class
 * must appear in the god's "Devotos." line — except Humanos and Clérigos,
 * who may be devotos of any god.
 *
 * The `Deus.devotos` entries are verbatim book strings (plurals, sentinels
 * like "Quaisquer"), so this module owns the term → app-name mapping.
 * Consumers treat ineligibility as ADVISORY (homebrew tables negotiate).
 */
import type { Deus } from './abilities/deuses'

/** Sentinel devotos terms that admit everyone. */
const OPEN_TERMS: ReadonlySet<string> = new Set([
  'Quaisquer',
  'Aventureiros (todas as classes)',
])

/** Verbatim devotos term → race/class names as the app stores them. */
const TERM_TO_NAMES: Readonly<Record<string, readonly string[]>> = {
  Aggelus: ['Suraggel'],
  Sulfure: ['Suraggel'],
  'Qualquer duyshidakk': ['Goblin'],
  Anões: ['Anão'],
  Arcanistas: ['Arcanista'],
  Bárbaros: ['Bárbaro'],
  Bardos: ['Bardo'],
  Bucaneiros: ['Bucaneiro'],
  Caçadores: ['Caçador'],
  Cavaleiros: ['Cavaleiro'],
  Dahllan: ['Dahllan'],
  Druidas: ['Druida'],
  Elfos: ['Elfo'],
  Goblins: ['Goblin'],
  Golens: ['Golem'],
  Guerreiros: ['Guerreiro'],
  Hynne: ['Hynne'],
  Inventores: ['Inventor'],
  Kliren: ['Kliren'],
  Ladinos: ['Ladino'],
  Lutadores: ['Lutador'],
  Medusas: ['Medusa'],
  Minotauros: ['Minotauro'],
  Nobres: ['Nobre'],
  Osteon: ['Osteon'],
  Paladinos: ['Paladino'],
  Qareen: ['Qareen'],
  'Sereias/Tritões': ['Sereia/Tritão'],
  Sílfides: ['Sílfide'],
  Trogs: ['Trog'],
}

/**
 * True when the character may be devoto of `deus` by the book. Humano race or
 * Clérigo class always qualify (p96 exception). Unknown devotos terms are
 * skipped (never silently grant access).
 */
export function devotoEligible(
  deus: Deus,
  raceNames: readonly string[],
  classNames: readonly string[],
): boolean {
  if (raceNames.includes('Humano') || classNames.includes('Clérigo')) return true
  const owned = new Set([...raceNames, ...classNames])
  for (const term of deus.devotos ?? []) {
    if (OPEN_TERMS.has(term)) return true
    if ((TERM_TO_NAMES[term] ?? []).some((n) => owned.has(n))) return true
  }
  return false
}
