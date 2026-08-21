import { caminhoSlotFor } from '@/shared/rules/abilities-caminhos'
import { resolveAtributoMod } from '@/shared/rules/racas-attr'
import { racasList } from '@/shared/lib/racas-cache'
import { slotsForClassLevel } from '@/shared/rules/abilities-classes-slots'
import {
  allGeneralPowers,
  classPowersFor,
  devotoOptionsFor,
  getOrigin,
  getRace,
} from '@/shared/lib/abilities-cache'
import type { AttributeKey } from '@/shared/api/attribute-keys'
import type { Character } from '@/shared/api/api'
import { parseClassChoices } from '@/entities/character/derived'
import { parseChoices } from './parse-choices'

/** Which sub-tab a pendência belongs to — matches the source tab values. */
export type PendenciaSource = 'raca' | 'origem' | 'classe'

/**
 * One still-unmade required choice on the Habilidades tab. `cardId` targets the
 * collapsible card that owns the choice, so clicking a pendência can jump to
 * the source tab and open + scroll to that card.
 */
export type Pendencia = {
  source: PendenciaSource
  label: string
  cardId: string
}

const ORIGIN_BENEFIT_LIMIT = 2

/**
 * Derives the list of unmade choices from query data alone (no server call).
 * A source that isn't in the catalog emits no pendência — we can't validate
 * choices for data we don't have.
 *
 * @example
 * const pendencias = computePendencias(character)
 * // [{ source: 'origem', label: 'Origem: faltam 2 benefícios', cardId: 'origem' }]
 */
export function computePendencias(character: Character): Pendencia[] {
  return [
    ...racePendencias(character),
    ...originPendencias(character),
    ...classPendencias(character),
  ]
}

function racePendencias(character: Character): Pendencia[] {
  const choices = parseChoices(character.raceAbilityChoices)
  const out: Pendencia[] = []
  for (const { race: raceId } of character.races) {
    const race = getRace(raceId)
    if (!race) continue
    if (atributoDeRacaPendente(character, race.name)) {
      out.push({
        source: 'raca',
        label: `Raça: distribuir o bônus de atributo de ${race.name}`,
        cardId: `raca:${race.id}`,
      })
    }
    for (const ability of race.abilities) {
      if (!ability.variants) continue
      const picked = ability.variants.some((v) => choices.includes(v.id))
      if (picked) continue
      out.push({
        source: 'raca',
        label: `Raça: escolher variante de ${ability.name}`,
        cardId: `raca:${race.id}`,
      })
    }
  }
  return out
}

function originPendencias(character: Character): Pendencia[] {
  const origin = getOrigin(character.origin)
  if (!origin) return []
  const benefitIds = new Set(
    [...origin.benefits, origin.poderUnico].map((b) => b.id),
  )
  const chosen = parseChoices(character.originChoices).filter((id) =>
    benefitIds.has(id),
  )
  const remaining = ORIGIN_BENEFIT_LIMIT - chosen.length
  if (remaining <= 0) return []
  const noun = remaining === 1 ? 'benefício' : 'benefícios'
  return [
    {
      source: 'origem',
      label: `Origem: faltam ${remaining} ${noun}`,
      cardId: 'origem',
    },
  ]
}

function classPendencias(character: Character): Pendencia[] {
  const allChosen = parseChoices(character.classPowers)
  const classChoices = parseClassChoices(character.classChoices)
  const out: Pendencia[] = []
  for (const entry of character.classes) {
    const { className, level } = entry
    const pool = classPowersFor(className)
    if (pool.length === 0) continue
    const cardId = `classe:${className}`

    const remaining = slotsRemainingFor(className, level, allChosen)
    if (remaining > 0) {
      const noun = remaining === 1 ? 'poder' : 'poderes'
      out.push({
        source: 'classe',
        label: `Classe ${className}: ${remaining} ${noun} por escolher`,
        cardId,
      })
    }

    const blob = classChoices[className] ?? {}
    if (devotoOptionsFor(className) !== null && !blob.devoto) {
      out.push({
        source: 'classe',
        label: `Classe ${className}: escolher devoto`,
        cardId,
      })
    }
    const caminhoSlot = caminhoSlotFor(className)
    if (caminhoSlot !== null && level >= caminhoSlot.minLevel && !blob.caminho) {
      out.push({
        source: 'classe',
        label: `Classe ${className}: escolher caminho`,
        cardId,
      })
    }
  }
  return out
}

/** Open power slots for a class = granted slots minus elective/general picks. */
function slotsRemainingFor(
  className: string,
  level: number,
  allChosen: string[],
): number {
  const pool = classPowersFor(className)
  const electiveIds = new Set(
    pool.filter((p) => p.grantedAtLevel === undefined).map((p) => p.id),
  )
  const generalIds = new Set(allGeneralPowers().map((p) => p.id))
  const picks = allChosen.filter(
    (id) => electiveIds.has(id) || generalIds.has(id),
  ).length
  return Math.max(0, slotsForClassLevel(className, level).length - picks)
}

/**
 * "1 escolha pendente" / "3 escolhas pendentes".
 *
 * O plural só apareceu quando o número passou a ser LIDO junto do rótulo: a
 * pílula era `aria-hidden` sem nada ao lado, então ninguém ouvia "1 escolhas"
 * (ALE-173, P6).
 */
export function escolhasPendentes(total: number): string {
  return total === 1 ? '1 escolha pendente' : `${total} escolhas pendentes`
}

/**
 * Se a raça ainda deve uma escolha de atributo — o `+1 ×3` do Humano, a
 * ascendência do Suraggel (ALE-169).
 *
 * PERGUNTA à autoridade em vez de repetir a condição dela: o
 * `resolveAtributoMod` já sabe quantas escolhas cada raça pede, que elas têm de
 * ser distintas e qual atributo é proibido, e LANÇA quando a conta não fecha.
 * Reescrever essas três regras aqui seria a asserção que se re-deriva da
 * implementação, com a garantia de divergir no dia em que uma raça nova tiver
 * uma quarta condição.
 *
 * Existe porque o Resumo da forja promete, por escrito, "dá para criar assim e
 * terminar na ficha" — e a ficha não carregava esta pendência. O personagem
 * ficava ilegal pelo livro ("Sua raça modifica seus atributos", p18) sem outro
 * conserto além de refazer a forja.
 */
function atributoDeRacaPendente(character: Character, raceName: string): boolean {
  const raca = racasList().find((r) => r.name === raceName)
  if (!raca) return false
  const escolha = lerEscolhaDeAtributo(character.raceAttributeChoices)
  try {
    resolveAtributoMod(raca, escolha)
    return false
  } catch {
    return true
  }
}

function lerEscolhaDeAtributo(raw: string): {
  floatingPicks?: AttributeKey[]
  ascendencia?: string
} {
  try {
    const p = JSON.parse(raw) as { floatingPicks?: unknown; ascendencia?: unknown }
    return {
      floatingPicks: Array.isArray(p.floatingPicks)
        ? (p.floatingPicks.filter((x) => typeof x === 'string') as AttributeKey[])
        : [],
      ascendencia: typeof p.ascendencia === 'string' ? p.ascendencia : undefined,
    }
  } catch {
    return { floatingPicks: [] }
  }
}
