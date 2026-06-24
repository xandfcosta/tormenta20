/**
 * Class PV/PM seeds (PDF Cap 1 — class entries p36-83).
 *
 * Per-class entry text format: "Você começa com X pontos de vida +
 * Constituição" and "ganha Y pontos de vida + Constituição por nível
 * seguinte". PM is uniform per level: "ganha Z PM por nível".
 *
 * These values are the **base** — Constituição is folded in separately
 * by the derivation helpers since it may change mid-campaign (race
 * picks, items). Paladino has a one-time `+ Carisma` to PM at L1 — read
 * the `paladinoMpAtL1Bonus` flag at the call site and add it once.
 *
 * Defesa is NOT per-class in T20: PDF p106 gives `10 + Destreza + armor
 * + shield`. Class-flavored Defesa bonuses are situational powers
 * (Cavaleiro Baluarte, Bucaneiro Insolência, Lutador Braços Calejados)
 * and live in the powers catalog — not in this table.
 */
export type ClassVitals = {
  /** Pontos de Vida at level 1 (before adding Constituição). */
  pvInicial: number
  /** Pontos de Vida gained per level after 1st (before Constituição). */
  pvPerLevel: number
  /** Pontos de Mana gained per level (uniform L1..L20). */
  mpPerLevel: number
  /** Optional one-time L1 bonus marker — only Paladino sets this. */
  paladinoMpAtL1Bonus?: 'charisma'
}

export const CLASS_VITALS: Record<string, ClassVitals> = {
  Arcanista: { pvInicial: 8, pvPerLevel: 2, mpPerLevel: 6 },
  'Bárbaro': { pvInicial: 24, pvPerLevel: 6, mpPerLevel: 3 },
  Bardo: { pvInicial: 12, pvPerLevel: 3, mpPerLevel: 4 },
  Bucaneiro: { pvInicial: 16, pvPerLevel: 4, mpPerLevel: 3 },
  'Caçador': { pvInicial: 16, pvPerLevel: 4, mpPerLevel: 4 },
  Cavaleiro: { pvInicial: 20, pvPerLevel: 5, mpPerLevel: 3 },
  'Clérigo': { pvInicial: 16, pvPerLevel: 4, mpPerLevel: 5 },
  Druida: { pvInicial: 16, pvPerLevel: 4, mpPerLevel: 4 },
  Guerreiro: { pvInicial: 20, pvPerLevel: 5, mpPerLevel: 3 },
  Inventor: { pvInicial: 12, pvPerLevel: 3, mpPerLevel: 4 },
  Ladino: { pvInicial: 12, pvPerLevel: 3, mpPerLevel: 4 },
  Lutador: { pvInicial: 20, pvPerLevel: 5, mpPerLevel: 3 },
  Nobre: { pvInicial: 16, pvPerLevel: 4, mpPerLevel: 4 },
  Paladino: {
    pvInicial: 20,
    pvPerLevel: 5,
    mpPerLevel: 3,
    paladinoMpAtL1Bonus: 'charisma',
  },
}

export type CharacterClassEntry = {
  className: string
  level: number
}

/**
 * Total Pontos de Vida **before** Constituição. Splits the seed (PV
 * inicial of the L1 class) from the per-level grants summed across all
 * classes. Multiclass uses the *first* class's PV inicial as the seed —
 * matches PDF p33 sidebar: "PV inicial usa o valor da classe do 1º nível".
 *
 * Caller adds `Constituição * totalLevel` to get final PV max.
 */
export function classPvBase(classes: readonly CharacterClassEntry[]): number {
  if (classes.length === 0) return 0
  const seedClass = classes[0]!
  const seedEntry = CLASS_VITALS[seedClass.className]
  if (!seedEntry) return 0
  let pv = seedEntry.pvInicial
  pv += seedEntry.pvPerLevel * (seedClass.level - 1)
  for (let i = 1; i < classes.length; i++) {
    const c = classes[i]!
    const entry = CLASS_VITALS[c.className]
    if (!entry) continue
    pv += entry.pvPerLevel * c.level
  }
  return pv
}

/**
 * Total Pontos de Mana from class grants. Paladino additionally gets
 * `+ Carisma` once at L1 — the caller supplies the character's Carisma
 * value so the value is correct regardless of mid-campaign Cha changes.
 *
 * Multiclass: sums each class's `mpPerLevel * level`. Paladino bonus
 * applies once if *any* Paladino entry is present (multiclassing into
 * Paladino doesn't unlock a second Carisma bonus).
 */
export function classMpBase(
  classes: readonly CharacterClassEntry[],
  charisma: number,
): number {
  let mp = 0
  let hasPaladino = false
  for (const c of classes) {
    const entry = CLASS_VITALS[c.className]
    if (!entry) continue
    mp += entry.mpPerLevel * c.level
    if (entry.paladinoMpAtL1Bonus === 'charisma') hasPaladino = true
  }
  if (hasPaladino) mp += charisma
  return mp
}
