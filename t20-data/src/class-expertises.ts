import type { ExpertiseName } from './expertises'

/**
 * Class-granted perícia training (PDF Cap 1 — class entries p36-83). Each
 * base class auto-trains a fixed list of perícias plus an "à escolha"
 * pool from which the player picks N at character creation. A handful of
 * classes (Bucaneiro, Caçador, Guerreiro, Nobre) also have an `either/or`
 * slot — a single fixed slot the player resolves between two pre-set
 * options.
 *
 * The three saving perícias (Fortitude, Reflexos, Vontade) are *not*
 * separate from the perícia list — they sit alongside the others and a
 * class is "trained in Reflexos" precisely when Reflexos appears in its
 * fixed list (or is picked from the either/or slot or the choose pool).
 *
 * This module is data-only: PDF rules encoded as a typed table + a small
 * resolver. Backend / UI integration is a separate PR.
 */
export type ClassTrainedExpertises = {
  /** Always auto-trained. */
  fixed: ExpertiseName[]
  /** Pick ONE; the chosen perícia is trained. PDF refers to these as the
   *  "Luta ou Pontaria" / "Diplomacia ou Intimidação" lines. */
  eitherOr?: {
    options: [ExpertiseName, ExpertiseName]
  }
  /** Number of additional perícias the player picks from `choosePool`. */
  chooseCount: number
  /** Pool the player may draw from for the `chooseCount` slots. */
  choosePool: ExpertiseName[]
}

export const CLASS_EXPERTISES_TRAINED: Record<string, ClassTrainedExpertises> = {
  Arcanista: {
    fixed: ['Misticismo', 'Vontade'],
    chooseCount: 2,
    choosePool: [
      'Conhecimento',
      'Diplomacia',
      'Enganação',
      'Guerra',
      'Iniciativa',
      'Intimidação',
      'Intuição',
      'Investigação',
      'Nobreza',
      'Ofício',
      'Percepção',
    ],
  },
  'Bárbaro': {
    fixed: ['Fortitude', 'Luta'],
    chooseCount: 4,
    choosePool: [
      'Adestramento',
      'Atletismo',
      'Cavalgar',
      'Iniciativa',
      'Intimidação',
      'Ofício',
      'Percepção',
      'Pontaria',
      'Sobrevivência',
      'Vontade',
    ],
  },
  Bardo: {
    fixed: ['Atuação', 'Reflexos'],
    chooseCount: 6,
    choosePool: [
      'Acrobacia',
      'Cavalgar',
      'Conhecimento',
      'Diplomacia',
      'Enganação',
      'Furtividade',
      'Iniciativa',
      'Intuição',
      'Investigação',
      'Jogatina',
      'Ladinagem',
      'Luta',
      'Misticismo',
      'Nobreza',
      'Percepção',
      'Pontaria',
      'Vontade',
    ],
  },
  Bucaneiro: {
    fixed: ['Reflexos'],
    eitherOr: { options: ['Luta', 'Pontaria'] },
    chooseCount: 4,
    choosePool: [
      'Acrobacia',
      'Atletismo',
      'Atuação',
      'Enganação',
      'Fortitude',
      'Furtividade',
      'Iniciativa',
      'Intimidação',
      'Jogatina',
      'Ofício',
      'Percepção',
      'Pilotagem',
    ],
  },
  'Caçador': {
    fixed: ['Sobrevivência'],
    eitherOr: { options: ['Luta', 'Pontaria'] },
    chooseCount: 6,
    choosePool: [
      'Adestramento',
      'Atletismo',
      'Cavalgar',
      'Cura',
      'Fortitude',
      'Furtividade',
      'Iniciativa',
      'Investigação',
      'Ofício',
      'Percepção',
      'Reflexos',
    ],
  },
  Cavaleiro: {
    fixed: ['Fortitude', 'Luta'],
    chooseCount: 2,
    choosePool: [
      'Adestramento',
      'Atletismo',
      'Cavalgar',
      'Diplomacia',
      'Guerra',
      'Iniciativa',
      'Intimidação',
      'Nobreza',
      'Percepção',
      'Vontade',
    ],
  },
  'Clérigo': {
    fixed: ['Religião', 'Vontade'],
    chooseCount: 2,
    choosePool: [
      'Conhecimento',
      'Cura',
      'Diplomacia',
      'Fortitude',
      'Iniciativa',
      'Intuição',
      'Luta',
      'Misticismo',
      'Nobreza',
      'Ofício',
      'Percepção',
    ],
  },
  Druida: {
    fixed: ['Sobrevivência', 'Vontade'],
    chooseCount: 4,
    choosePool: [
      'Adestramento',
      'Atletismo',
      'Cavalgar',
      'Conhecimento',
      'Cura',
      'Fortitude',
      'Intuição',
      'Luta',
      'Misticismo',
      'Ofício',
      'Percepção',
      'Religião',
    ],
  },
  Guerreiro: {
    fixed: ['Fortitude'],
    eitherOr: { options: ['Luta', 'Pontaria'] },
    chooseCount: 2,
    choosePool: [
      'Adestramento',
      'Atletismo',
      'Cavalgar',
      'Guerra',
      'Iniciativa',
      'Intimidação',
      'Ofício',
      'Percepção',
      'Reflexos',
    ],
  },
  Inventor: {
    fixed: ['Ofício', 'Vontade'],
    chooseCount: 4,
    choosePool: [
      'Conhecimento',
      'Cura',
      'Diplomacia',
      'Fortitude',
      'Iniciativa',
      'Investigação',
      'Luta',
      'Misticismo',
      'Percepção',
      'Pilotagem',
      'Pontaria',
    ],
  },
  Ladino: {
    fixed: ['Ladinagem', 'Reflexos'],
    chooseCount: 8,
    choosePool: [
      'Acrobacia',
      'Atletismo',
      'Atuação',
      'Cavalgar',
      'Conhecimento',
      'Diplomacia',
      'Enganação',
      'Furtividade',
      'Iniciativa',
      'Intimidação',
      'Intuição',
      'Investigação',
      'Jogatina',
      'Luta',
      'Ofício',
      'Percepção',
      'Pilotagem',
      'Pontaria',
    ],
  },
  Lutador: {
    fixed: ['Fortitude', 'Luta'],
    chooseCount: 4,
    choosePool: [
      'Acrobacia',
      'Adestramento',
      'Atletismo',
      'Enganação',
      'Furtividade',
      'Iniciativa',
      'Intimidação',
      'Ofício',
      'Percepção',
      'Pontaria',
      'Reflexos',
    ],
  },
  Nobre: {
    fixed: ['Vontade'],
    eitherOr: { options: ['Diplomacia', 'Intimidação'] },
    chooseCount: 4,
    choosePool: [
      'Adestramento',
      'Atuação',
      'Cavalgar',
      'Conhecimento',
      'Enganação',
      'Fortitude',
      'Guerra',
      'Iniciativa',
      'Intuição',
      'Investigação',
      'Jogatina',
      'Luta',
      'Nobreza',
      'Ofício',
      'Percepção',
      'Pontaria',
    ],
  },
  Paladino: {
    fixed: ['Luta', 'Vontade'],
    chooseCount: 2,
    choosePool: [
      'Adestramento',
      'Atletismo',
      'Cavalgar',
      'Cura',
      'Diplomacia',
      'Fortitude',
      'Guerra',
      'Iniciativa',
      'Intuição',
      'Nobreza',
      'Percepção',
      'Religião',
    ],
  },
}

/**
 * Resolve which perícias a character is auto-trained in given their
 * classes + the player's per-class either/or + chosen picks. Returns a
 * Set so order is irrelevant downstream.
 *
 * Multiclass: a perícia is trained if ANY class trains it — duplicate
 * grants do nothing extra (no benefit "stacking" rule applies to
 * training; you either have it or you don't).
 *
 * `picks` is keyed by className so a Guerreiro/Bardo multiclass can pin
 * independent picks per side.
 */
export type ClassExpertisePicks = Partial<
  Record<
    string,
    {
      /** Which option of the eitherOr slot was chosen, if any. */
      eitherOr?: ExpertiseName
      /** Which entries from `choosePool` the player picked. */
      chosen?: readonly ExpertiseName[]
    }
  >
>

export function classTrainedExpertises(
  classNames: readonly string[],
  picks: ClassExpertisePicks = {},
): Set<ExpertiseName> {
  const trained = new Set<ExpertiseName>()
  for (const className of classNames) {
    const entry = CLASS_EXPERTISES_TRAINED[className]
    if (!entry) continue
    for (const p of entry.fixed) trained.add(p)
    const pick = picks[className]
    const eitherOrChoice = pick?.eitherOr
    if (entry.eitherOr && eitherOrChoice) {
      const allowed = entry.eitherOr.options
      if (allowed.includes(eitherOrChoice)) trained.add(eitherOrChoice)
    }
    for (const c of pick?.chosen ?? []) {
      if (entry.choosePool.includes(c)) trained.add(c)
    }
  }
  return trained
}
