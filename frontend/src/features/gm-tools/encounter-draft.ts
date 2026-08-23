import { createSignal } from 'solid-js'
import type { EncounterEntry } from './encounter'
import type { PartyDefaults } from './party-defaults'

export type EncounterDraft = {
  entries: () => EncounterEntry[]
  partyLevel: () => number
  partySize: () => number
  setPartyLevel: (level: number) => void
  setPartySize: (size: number) => void
  /** Adds one of a creature, or bumps the count if it is already in. */
  add: (monsterId: string) => void
  setQuantity: (monsterId: string, quantity: number) => void
  remove: (monsterId: string) => void
  clear: () => void
}

/** Sem campanha por perto (a Mesa do Mestre monta encontro no vazio), o grupo
 *  padrão é o do livro: quatro personagens de 1º (p282). */
const DEFAULT_PARTY_LEVEL = 1
const DEFAULT_PARTY_SIZE = 4

/**
 * The encounter being composed. `create*` holding state → born once in a
 * component body (gotcha #17).
 *
 * Adding a creature already in the encounter BUMPS its count instead of adding
 * a second row: two rows of the same monster would each compute their own group
 * ND, and the p282 doubling rule only means anything over one group.
 *
 * O `party` é o grupo DERIVADO da campanha (ALE-209), e é um acessor porque os
 * membros chegam por prop e podem trocar depois da montagem. A regra é: o campo
 * mostra o derivado até o mestre ENCOSTAR nele, e a partir daí o que ele
 * escreveu vence para sempre — inclusive se ele digitar exatamente o número que
 * já estava. Sem isso, montar para meio grupo seria desfeito pelo próprio app
 * assim que a lista de membros se atualizasse.
 *
 * @example const draft = createEncounterDraft(() => partyFromMembers(props.members))
 */
export function createEncounterDraft(party?: () => PartyDefaults | null): EncounterDraft {
  const [entries, setEntries] = createSignal<EncounterEntry[]>([])
  const [levelOverride, setLevelOverride] = createSignal<number | null>(null)
  const [sizeOverride, setSizeOverride] = createSignal<number | null>(null)

  const partyLevel = () => levelOverride() ?? party?.()?.level ?? DEFAULT_PARTY_LEVEL
  const partySize = () => sizeOverride() ?? party?.()?.size ?? DEFAULT_PARTY_SIZE

  return {
    entries,
    partyLevel,
    partySize,
    setPartyLevel: (level) => setLevelOverride(clamp(level, 1, 20)),
    setPartySize: (size) => setSizeOverride(clamp(size, 1, 8)),
    add: (monsterId) =>
      setEntries((prev) =>
        prev.some((entry) => entry.monsterId === monsterId)
          ? prev.map((entry) =>
              entry.monsterId === monsterId
                ? { ...entry, quantity: entry.quantity + 1 }
                : entry,
            )
          : [...prev, { monsterId, quantity: 1 }],
      ),
    setQuantity: (monsterId, quantity) =>
      setEntries((prev) =>
        prev.map((entry) =>
          entry.monsterId === monsterId
            ? { ...entry, quantity: Math.max(1, quantity) }
            : entry,
        ),
      ),
    remove: (monsterId) =>
      setEntries((prev) => prev.filter((entry) => entry.monsterId !== monsterId)),
    clear: () => setEntries([]),
  }
}

function clamp(value: number, min: number, max: number): number {
  if (!Number.isFinite(value)) return min
  return Math.min(max, Math.max(min, Math.round(value)))
}
