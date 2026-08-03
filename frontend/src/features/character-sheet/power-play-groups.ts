import { FLAG_ACTIVATIONS } from '@tormenta20/t20-data'
import type { ActivationSpec } from '@tormenta20/t20-data'
import { stanceFlagOf } from '@/entities/character/use-power-action'
import type { SheetSearchEntry } from './sheet-search-index'

export type PlayPower = {
  entry: SheetSearchEntry
  spec: ActivationSpec | undefined
}

export type PlayPowerGroups = { acoes: PlayPower[]; passivas: PlayPower[] }

type AcaoPower = PlayPower & { spec: ActivationSpec }

function isAcao(power: PlayPower): power is AcaoPower {
  return power.spec?.kind === 'instant' || power.spec?.kind === 'stance'
}

/** 'variavel' sorts after every numeric cost without a special-case rank. */
function numericPm(spec: ActivationSpec): number {
  return spec.pmCost === 'variavel' ? Number.POSITIVE_INFINITY : spec.pmCost
}

function acaoRank(
  spec: ActivationSpec,
  activeFlags: ReadonlySet<string>,
): number {
  const flag = spec.kind === 'stance' ? stanceFlagOf(spec) : undefined
  return flag && activeFlags.has(flag) ? 0 : 1
}

/**
 * Partition owned powers into the play-mode groups and order AÇÕES for the
 * table: active stances first, then quick-bar favorites (in star order), then
 * everything usable by ascending PM cost with 'variavel' costs last. PASSIVAS
 * (passive, triggered-passive, no spec) keeps catalog order.
 *
 * @example groupPlayPowers(powers, new Set(['furia']))
 */
export function groupPlayPowers(
  powers: readonly PlayPower[],
  activeFlags: ReadonlySet<string>,
): PlayPowerGroups {
  // Tier autos collapse onto one spec (Inspiração +1..+5 → the stance) —
  // keep the first row per spec so the list shows one action, not five.
  const seen = new Set<string>()
  const deduped = powers.filter((p) => {
    if (!isAcao(p)) return true
    if (seen.has(p.spec.id)) return false
    seen.add(p.spec.id)
    return true
  })
  const acoes = deduped.filter(isAcao).sort((a, b) => {
    const rankA = acaoRank(a.spec, activeFlags)
    const rankB = acaoRank(b.spec, activeFlags)
    if (rankA !== rankB) return rankA - rankB
    return numericPm(a.spec) - numericPm(b.spec)
  })
  return { acoes, passivas: deduped.filter((p) => !isAcao(p)) }
}

/**
 * Triggered passives whose gatilho flag is currently up — the live line the
 * collapsed PASSIVAS disclosure surfaces so an active Fúria rider stays
 * visible without expanding the group.
 *
 * @example activeTriggeredPassives(passivas, new Set(['furia']))
 */
export function activeTriggeredPassives(
  passivas: readonly PlayPower[],
  activeFlags: ReadonlySet<string>,
): AcaoPower[] {
  return passivas.filter(
    (p): p is AcaoPower =>
      p.spec?.kind === 'triggered-passive' &&
      p.spec.requiresFlag !== undefined &&
      activeFlags.has(p.spec.requiresFlag),
  )
}

/** 'Alma de Bronze (Fúria)' — flag display name from FLAG_ACTIVATIONS. */
export function gatilhoLabel(spec: ActivationSpec): string {
  const flag = spec.requiresFlag ?? ''
  return FLAG_ACTIVATIONS[flag]?.name ?? flag
}

/** 'Classe · Bárbaro' → 'Bárbaro'; other sources compress to one word. */
export function shortSourceLabel(source: string): string {
  if (source.startsWith('Classe · ')) return source.slice('Classe · '.length)
  if (source.startsWith('Raça')) return 'Raça'
  if (source.startsWith('Origem')) return 'Origem'
  if (source.startsWith('Deus')) return 'Deus'
  if (source === 'Poder da Tormenta') return 'Tormenta'
  return 'Geral'
}

