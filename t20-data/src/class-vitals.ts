/**
 * Class PV/PM seeds (PDF Cap 1 — class entries p36-83).
 *
 * Per-class entry text format: "Você começa com X pontos de vida +
 * Constituição" and "ganha Y pontos de vida + Constituição por nível
 * seguinte". PM is uniform per level: "ganha Z PM por nível".
 *
 * These values are the **base** — Constituição is folded in separately
 * by the derivation helpers since it may change mid-campaign (race
 * picks, items). Attribute/level PM+PV grants (Paladino Devoção +Carisma,
 * Clérigo Magia Divina +Sabedoria, Anão Duro como Pedra…) are modeled as
 * `maxPv`/`maxPm` power modifiers folded in by `vital-grants.ts`.
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
  // Paladino Cha→PM (L1) is modeled via the Devoção power's maxPm modifier.
  Paladino: { pvInicial: 20, pvPerLevel: 5, mpPerLevel: 3 },
}

export type CharacterClassEntry = {
  className: string
  level: number
}

/**
 * PV pool with Constituição folded in, honouring the p34 floor: "você sempre
 * ganha pelo menos 1 PV ao subir de nível". Each level past the 1st grants
 * max(1, pvPerLevel + con); the 1st grants pvInicial + con (the floor talks
 * about LEVELING, so L1 has none — the sheet's global 0-floor covers it).
 * With pvPerLevel + con ≥ 1 this equals the plain linear sum.
 *
 * @example pvPoolWithCon(CLASS_VITALS.Arcanista, 5, -2) // → 10, not 6
 */
export function pvPoolWithCon(
  vitals: Pick<ClassVitals, 'pvInicial' | 'pvPerLevel'>,
  level: number,
  con: number,
): number {
  const perLevel = Math.max(1, vitals.pvPerLevel + con)
  return vitals.pvInicial + con + (level - 1) * perLevel
}

/**
 * Multiclass PV pool with Constituição + the p34 min-1 floor. p34-35: only
 * the FIRST class contributes its PV inicial ("Zaled ganha 5 PV pelo
 * primeiro nível de paladino, não 20"); every other level — of any class —
 * grants max(1, pvPerLevel + con). Single-class degenerates to
 * `pvPoolWithCon`.
 */
export function multiclassPvPool(
  classes: readonly CharacterClassEntry[],
  con: number,
): number {
  const seed = classes[0] && CLASS_VITALS[classes[0].className]
  if (!seed) return 0
  let pv = pvPoolWithCon(seed, classes[0]!.level, con)
  for (const c of classes.slice(1)) {
    const entry = CLASS_VITALS[c.className]
    if (!entry) continue
    pv += c.level * Math.max(1, entry.pvPerLevel + con)
  }
  return pv
}

/**
 * Multiclass PM pool — p35: "some os PM fornecidos por cada classe". No
 * attribute riders here; Paladino +Car / caster key-attribute→PM live as
 * `maxPm` power modifiers resolved by vital-grants (single source, no
 * double count).
 */
export function multiclassMpPool(
  classes: readonly CharacterClassEntry[],
): number {
  let mp = 0
  for (const c of classes) {
    const entry = CLASS_VITALS[c.className]
    if (entry) mp += entry.mpPerLevel * c.level
  }
  return mp
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
    // Paladino Cha→PM (L1). Authoritative copy for the sheet is the Abençoado
    // power modifier; this legacy multiclass helper keeps it for parity.
    if (c.className === 'Paladino') hasPaladino = true
  }
  if (hasPaladino) mp += charisma
  return mp
}
