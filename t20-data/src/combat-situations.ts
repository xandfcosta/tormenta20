/**
 * Tabela 5-3 (Situações Especiais) — PDF Cap 5, livro p239.
 *
 * Numeric and qualitative modifiers that apply when an attack is
 * resolved under a tabled "situação especial". Cover (cobertura) and
 * concealment (camuflagem) rows are owned by `combat-cover.ts`; this
 * module covers the rest of the table.
 *
 * `appliesTo` indicates which side of the resolution math the modifier
 * sits on (ataque bonus vs alvo's Defesa). `meleeOnly` / `rangedOnly`
 * gate the modifier by attack range.
 *
 * "Caído" appears on the alvo side as TWO rows because the same
 * condition gives opposite modifiers depending on the attack range
 * (-5 vs corpo a corpo, +5 vs distância). They are not collapsible.
 */
export type SituacaoSide = 'atacante' | 'alvo'
export type SituacaoAppliesTo = 'ataque' | 'defesa' | 'misc'

export type SituacaoEspecial = {
  id: string
  name: string
  side: SituacaoSide
  condition: string
  modifier: number
  appliesTo: SituacaoAppliesTo
  meleeOnly: boolean
  rangedOnly: boolean
  qualitativeNote: string | null
  bookPage: 239
}

export const SITUACOES_ESPECIAIS: readonly SituacaoEspecial[] = Object.freeze([
  // ─── O atacante está... (Modificador no Ataque) ─────────────────
  {
    id: 'atacante-caido',
    name: 'Caído',
    side: 'atacante',
    condition: 'atacante caído, atacando em corpo a corpo',
    modifier: -5,
    appliesTo: 'ataque',
    meleeOnly: true,
    rangedOnly: false,
    qualitativeNote: null,
    bookPage: 239,
  },
  {
    id: 'atacante-cego',
    name: 'Cego',
    side: 'atacante',
    condition: 'atacante cego',
    modifier: 0,
    appliesTo: 'ataque',
    meleeOnly: false,
    rangedOnly: false,
    qualitativeNote: '50% de chance de falha',
    bookPage: 239,
  },
  {
    id: 'atacante-em-posicao-elevada',
    name: 'Em posição elevada',
    side: 'atacante',
    condition: 'atacante em posição elevada em relação ao alvo',
    modifier: 2,
    appliesTo: 'ataque',
    meleeOnly: false,
    rangedOnly: false,
    qualitativeNote: null,
    bookPage: 239,
  },
  {
    id: 'atacante-flanqueando',
    name: 'Flanqueando o alvo',
    side: 'atacante',
    condition: 'atacante flanqueando o alvo (corpo a corpo)',
    modifier: 2,
    appliesTo: 'ataque',
    meleeOnly: true,
    rangedOnly: false,
    qualitativeNote: null,
    bookPage: 239,
  },
  {
    id: 'atacante-invisivel',
    name: 'Invisível',
    side: 'atacante',
    condition: 'atacante invisível — alvo sofre -5 na Defesa',
    modifier: -5,
    appliesTo: 'defesa',
    meleeOnly: false,
    rangedOnly: false,
    qualitativeNote: 'O alvo sofre -5 na Defesa',
    bookPage: 239,
  },
  {
    id: 'atacante-ofuscado',
    name: 'Ofuscado',
    side: 'atacante',
    condition: 'atacante ofuscado',
    modifier: -2,
    appliesTo: 'ataque',
    meleeOnly: false,
    rangedOnly: false,
    qualitativeNote: null,
    bookPage: 239,
  },

  // ─── O alvo está... (Modificador na Defesa) ─────────────────────
  {
    id: 'alvo-caido-cac',
    name: 'Caído',
    side: 'alvo',
    condition: 'alvo caído, sendo atacado em corpo a corpo',
    modifier: -5,
    appliesTo: 'defesa',
    meleeOnly: true,
    rangedOnly: false,
    qualitativeNote: null,
    bookPage: 239,
  },
  {
    id: 'alvo-caido-distancia',
    name: 'Caído',
    side: 'alvo',
    condition: 'alvo caído, sendo atacado à distância',
    modifier: 5,
    appliesTo: 'defesa',
    meleeOnly: false,
    rangedOnly: true,
    qualitativeNote: null,
    bookPage: 239,
  },
  {
    id: 'alvo-cego',
    name: 'Cego',
    side: 'alvo',
    condition: 'alvo cego',
    modifier: -5,
    appliesTo: 'defesa',
    meleeOnly: false,
    rangedOnly: false,
    qualitativeNote: null,
    bookPage: 239,
  },
  {
    id: 'alvo-desprevenido',
    name: 'Desprevenido',
    side: 'alvo',
    condition: 'alvo desprevenido',
    modifier: -5,
    appliesTo: 'defesa',
    meleeOnly: false,
    rangedOnly: false,
    qualitativeNote: null,
    bookPage: 239,
  },
])

const byId = new Map(SITUACOES_ESPECIAIS.map((s) => [s.id, s]))

export function situacaoById(id: string): SituacaoEspecial | undefined {
  return byId.get(id)
}

export function situacoesBySide(
  side: SituacaoSide,
): readonly SituacaoEspecial[] {
  return SITUACOES_ESPECIAIS.filter((s) => s.side === side)
}

export type AttackRange = 'melee' | 'ranged'

/**
 * Returns the subset of `active` ids that *apply* given the attack
 * range. Drops entries gated by `meleeOnly` when range is ranged, and
 * `rangedOnly` when range is melee. Unknown ids are ignored.
 */
export function applicableSituacoes(
  active: readonly string[],
  range: AttackRange,
): readonly SituacaoEspecial[] {
  const out: SituacaoEspecial[] = []
  for (const id of active) {
    const s = byId.get(id)
    if (!s) continue
    if (s.meleeOnly && range !== 'melee') continue
    if (s.rangedOnly && range !== 'ranged') continue
    out.push(s)
  }
  return out
}

/**
 * Summed numeric modifier from a set of active situações, partitioned
 * by where the modifier lands. Qualitative effects (miss chance) are
 * NOT summed — callers handle those separately. Filters by range first.
 */
export function aggregateSituacaoModifiers(
  active: readonly string[],
  range: AttackRange,
): { ataque: number; defesa: number } {
  let ataque = 0
  let defesa = 0
  for (const s of applicableSituacoes(active, range)) {
    if (s.appliesTo === 'ataque') ataque += s.modifier
    else if (s.appliesTo === 'defesa') defesa += s.modifier
  }
  return { ataque, defesa }
}

/**
 * Qualitative notes from active situações (e.g. "50% de chance de
 * falha" do atacante cego). Returns one note per matching row, in the
 * order they appear in `active`. Filters by range first.
 */
export function qualitativeSituacaoNotes(
  active: readonly string[],
  range: AttackRange,
): readonly string[] {
  const out: string[] = []
  for (const s of applicableSituacoes(active, range)) {
    if (s.qualitativeNote) out.push(s.qualitativeNote)
  }
  return out
}
