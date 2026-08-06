import { useMutation, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { FLAG_ACTIVATIONS } from '@tormenta20/t20-data'
import type { ActivationSpec } from '@tormenta20/t20-data'
import {
  activationSpecs,
  findActivationByName,
  getActivation,
} from '@/shared/lib/activation-cache'
import { api, type Character } from '@/shared/api/api'
import { useConditionalsStore } from '@/shared/stores/conditionals-store'
import {
  usePowerUsesStore,
  type PowerUseScope,
} from '@/shared/stores/power-uses-store'
import { useStanceActivationStore } from '@/shared/stores/stance-activation-store'
import { parseChoiceSet, useAllConditionals, type ConditionalEntry } from './derived'
import { characterQueryOptions } from './queries'
import { applyPoolResult, isPoolSuperseded } from './temp-hp-pool'

export type PowerUseDecision = { ok: boolean; reason?: string }

export type PowerUseContext = {
  mpCurrent: number
  usedScene: number
  usedDay: number
  activeFlags: ReadonlySet<string>
}

/**
 * Can this INSTANT power fire right now? Pure so the button/disabled logic is
 * unit-testable without the hook. 'rodada' and numeric limits ship unenforced
 * (badge-only) — the table tracks rounds, the sheet doesn't.
 *
 * @example powerUseDecision(spec, { mpCurrent: 0, usedScene: 0, usedDay: 0, activeFlags: new Set() })
 *          // { ok: false, reason: 'PM insuficiente' }
 */
export function powerUseDecision(
  spec: ActivationSpec,
  ctx: PowerUseContext,
): PowerUseDecision {
  if (spec.pmCost === 'variavel') return { ok: false, reason: 'custo variável' }
  if (spec.requiresFlag && !ctx.activeFlags.has(spec.requiresFlag)) {
    return { ok: false, reason: `requer ${spec.requiresFlag}` }
  }
  if (spec.uses === 'cena' && ctx.usedScene >= 1) {
    return { ok: false, reason: 'limite por cena atingido' }
  }
  if (spec.uses === 'dia' && ctx.usedDay >= 1) {
    return { ok: false, reason: 'limite por dia atingido' }
  }
  if (spec.pmCost > ctx.mpCurrent) return { ok: false, reason: 'PM insuficiente' }
  return { ok: true }
}

/** Store scope for a limited spec, or undefined when the limit is unenforced. */
export function enforcedScopeOf(
  spec: ActivationSpec,
): PowerUseScope | undefined {
  if (spec.uses === 'cena') return 'scene'
  if (spec.uses === 'dia') return 'day'
  return undefined
}

/**
 * Registry lookup for an ability row: exact id first (class powers already use
 * the `class.<x>.<slug>` convention), then name fallback for surfaces whose
 * local ids don't follow it (race `humano-versatil`, origin `poder-sortudo`).
 *
 * @example resolveActivationSpec('Golpe Poderoso', 'class.barbaro.golpe-poderoso')
 */
export function resolveActivationSpec(
  name: string,
  id?: string,
): ActivationSpec | undefined {
  const byId = id ? getActivation(id) : undefined
  if (byId) return byId
  const byName = findActivationByName(name)
  if (byName) return byName
  // Tier autos ("Inspiração +1", "Fúria +3") share one stance spec named
  // without the tier suffix — retry stripped so every tier row resolves to
  // the same activation instead of falling through as a silent passive.
  const tierless = name.replace(/\s\+\d+$/, '')
  return tierless !== name ? findActivationByName(tierless) : undefined
}

/**
 * The flag a stance spec raises when activated. Stance specs don't carry their
 * flag (only `requiresFlag` for triggered powers), but every stance is born
 * from a FLAG_ACTIVATIONS entry — match back by name.
 *
 * @example stanceFlagOf(getActivation('class.barbaro.furia')!) // 'furia'
 */
export function stanceFlagOf(spec: ActivationSpec): string | undefined {
  return Object.values(FLAG_ACTIVATIONS).find((a) => a.name === spec.name)?.flag
}

/** Accent/case-insensitive slug so 'Bárbaro' matches the id segment 'barbaro'. */
function classNameSlug(name: string): string {
  return name
    .normalize('NFD')
    .replace(/[\u0300-\u036f]/g, '')
    .toLowerCase()
    .replace(/\s+/g, '-')
}

/**
 * Level in the class that OWNS the stance (p40: multiclass counts only the
 * Bárbaro levels for Fúria's scaling). 0 for non-class specs / class not taken.
 *
 * @example stanceClassLevel(furiaSpec, [{ className: 'Bárbaro', level: 6 }]) // 6
 */
export function stanceClassLevel(
  spec: ActivationSpec,
  classes: readonly { className: string; level: number }[],
): number {
  const [scope, owner] = spec.id.split('.')
  if (scope !== 'class') return 0
  return classes.find((c) => classNameSlug(c.className) === owner)?.level ?? 0
}

/**
 * Stepper ceiling for a scaling stance — 0 when the spec doesn't scale.
 *
 * @example stanceMaxSteps(furiaSpec, [{ className: 'Bárbaro', level: 6 }]) // 1
 */
export function stanceMaxSteps(
  spec: ActivationSpec,
  classes: readonly { className: string; level: number }[],
): number {
  if (!spec.scaling) return 0
  return spec.scaling.maxStepsForLevel(stanceClassLevel(spec, classes))
}

/**
 * Total PM to enter the stance with N extra steps. Non-scaling stances cost
 * their flat pmCost; a 'variavel' cost yields NaN so the decision refuses it.
 *
 * @example stanceTotalPm(furiaSpec, 1) // 3 (base 2 + 1×1)
 */
export function stanceTotalPm(spec: ActivationSpec, steps: number): number {
  if (spec.scaling) return spec.scaling.basePm + steps * spec.scaling.stepPm
  return spec.pmCost === 'variavel' ? Number.NaN : spec.pmCost
}

/**
 * Can the stance be entered with these steps and this pool? Pure so the
 * dialog's disabled state and the hook share one gate.
 *
 * @example stanceActivationDecision(furiaSpec, 1, 2) // { ok: false, reason: 'PM insuficiente' }
 */
export function stanceActivationDecision(
  spec: ActivationSpec,
  steps: number,
  mpCurrent: number,
): PowerUseDecision {
  const total = stanceTotalPm(spec, steps)
  if (Number.isNaN(total)) return { ok: false, reason: 'custo variável' }
  if (total > mpCurrent) return { ok: false, reason: 'PM insuficiente' }
  return { ok: true }
}

/**
 * OWNED powers whose grant fires with this stance flag (Fase 4): Alma de
 * Bronze materializes its temp-PV pool when Fúria switches on. Ownership
 * checks classPowers, tolerating bare slugs and full convention ids.
 *
 * @example grantPowersForFlag(barbaro, 'furia').map((s) => s.id)
 *          // ['class.barbaro.alma-de-bronze']
 */
export function grantPowersForFlag(
  character: Character,
  flag: string,
): ActivationSpec[] {
  const chosen = parseChoiceSet(character.classPowers)
  return activationSpecs().filter((spec) => {
    if (spec.requiresFlag !== flag || !spec.grant) return false
    const tail = spec.id.split('.').pop() ?? spec.id
    if (chosen.has(spec.id) || chosen.has(tail)) return true
    return [...chosen].some((id) => id.endsWith(`.${tail}`))
  })
}

export type PowerAction = {
  use: (spec: ActivationSpec) => void
  canUse: (spec: ActivationSpec) => PowerUseDecision
  /** Enter a stance paying base + steps×stepPm; flips its flag group on. */
  activateStance: (spec: ActivationSpec, steps: number) => void
  /** Flip the flag group off + clear the payment record. No refund. */
  deactivateStance: (flag: string) => void
}

/**
 * One-tap "Usar" for instant powers: validates via `powerUseDecision`, debits
 * PM optimistically (same pattern as the Efeitos tab's stance activation),
 * bumps the local use counter when the power is scene/day-limited.
 *
 * @example const { use, canUse } = usePowerAction(character)
 */
export function usePowerAction(character: Character): PowerAction {
  const qc = useQueryClient()
  const queryKey = characterQueryOptions(character.id).queryKey
  const entries = useAllConditionals(character)
  const bump = usePowerUsesStore((s) => s.bump)
  const counts = usePowerUsesStore((s) => s.uses[character.id])

  const spendPm = useMutation({
    mutationFn: (mpCurrent: number) =>
      api.characters.updateVitals(character.id, { mpCurrent }),
    onMutate: async (mpCurrent) => {
      await qc.cancelQueries({ queryKey })
      qc.setQueryData<Character>(queryKey, (prev) =>
        prev ? { ...prev, mpCurrent } : prev,
      )
    },
  })

  const contextFor = (spec: ActivationSpec): PowerUseContext => ({
    mpCurrent: character.mpCurrent,
    usedScene: counts?.scene[spec.id] ?? 0,
    usedDay: counts?.day[spec.id] ?? 0,
    activeFlags: new Set(
      entries.flatMap((e) => (e.active && e.effect.flag ? [e.effect.flag] : [])),
    ),
  })

  const use = (spec: ActivationSpec) => {
    const decision = powerUseDecision(spec, contextFor(spec))
    if (!decision.ok || spec.pmCost === 'variavel') return
    if (spec.pmCost > 0) spendPm.mutate(character.mpCurrent - spec.pmCost)
    const scope = enforcedScopeOf(spec)
    if (scope) bump(character.id, spec.id, scope)
    toast.success(`${spec.name} — ${spec.pmCost} PM`)
  }

  const stance = useStanceActions(character, entries, (mp) => spendPm.mutate(mp))
  return {
    use,
    canUse: (spec) => powerUseDecision(spec, contextFor(spec)),
    ...stance,
  }
}

/**
 * Stance enter/exit — owns the PM debit that used to live in the Efeitos
 * tab's `setFlagGroup`: pay total (base + steps), flip every conditional
 * sharing the stance flag (same store writes the old toggle produced, so
 * HUD badges like Alma de Bronze keep working), log {steps, pmPaid} for the
 * Posturas ativas display. Exit is free — the cost bought the stance.
 */
function useStanceActions(
  character: Character,
  entries: ConditionalEntry[],
  debitPm: (mpCurrent: number) => void,
): Pick<PowerAction, 'activateStance' | 'deactivateStance'> {
  const setMany = useConditionalsStore((s) => s.setMany)
  const logActivation = useStanceActivationStore((s) => s.logActivation)
  const clearActivation = useStanceActivationStore((s) => s.clearActivation)
  const { applyFlagGrants, removeFlagGrants } = useStanceGrantEffects(character)
  const flagEntryIds = (flag: string) =>
    entries.filter((e) => e.effect.flag === flag).map((e) => e.id)

  const activateStance = (spec: ActivationSpec, steps: number) => {
    const flag = stanceFlagOf(spec)
    if (!flag) return
    const decision = stanceActivationDecision(spec, steps, character.mpCurrent)
    if (!decision.ok) {
      toast.error(`${spec.name}: ${decision.reason}`)
      return
    }
    const total = stanceTotalPm(spec, steps)
    if (total > 0) debitPm(character.mpCurrent - total)
    setMany(character.id, flagEntryIds(flag), true)
    logActivation(character.id, flag, { steps, pmPaid: total })
    applyFlagGrants(flag)
    toast.success(`${spec.name} ativada — ${total} PM`)
  }

  const deactivateStance = (flag: string) => {
    setMany(character.id, flagEntryIds(flag), false)
    clearActivation(character.id, flag)
    removeFlagGrants(flag)
    toast.success(`${FLAG_ACTIVATIONS[flag]?.name ?? flag} encerrada`)
  }

  return { activateStance, deactivateStance }
}

/**
 * Fase 4: stance-triggered grants become REAL ActiveEffects. Entering the
 * stance applies every owned grant power of its flag (Alma de Bronze → the
 * server computes nível + Força and persists the temp-PV pool); leaving it
 * deletes those effects. The pool the HUD drains lives in activeEffects, so
 * both mutations patch the cached character with the returned deltas.
 */
function useStanceGrantEffects(character: Character): {
  applyFlagGrants: (flag: string) => void
  removeFlagGrants: (flag: string) => void
} {
  const qc = useQueryClient()
  const queryKey = characterQueryOptions(character.id).queryKey
  const apply = useMutation({
    mutationFn: (powerId: string) =>
      api.characters.applyEffect(character.id, { powerId }),
    onSuccess: (result) => {
      // Vale-o-maior (F1, p256): a bigger pool already covers this grant —
      // nothing was written, just tell the player which pool won.
      if (isPoolSuperseded(result)) {
        toast.info(
          `PV temporários mantidos (${result.keptAmount}) — vale o maior (p256)`,
        )
        return
      }
      qc.setQueryData<Character>(queryKey, (prev) =>
        prev ? applyPoolResult(prev, result) : prev,
      )
    },
  })
  const remove = useMutation({
    mutationFn: (effectId: number) =>
      api.characters.removeActiveEffect(character.id, effectId),
    onMutate: (effectId) => {
      qc.setQueryData<Character>(queryKey, (prev) =>
        prev
          ? {
              ...prev,
              activeEffects: prev.activeEffects.filter((e) => e.id !== effectId),
            }
          : prev,
      )
    },
  })
  const applyFlagGrants = (flag: string) => {
    for (const spec of grantPowersForFlag(character, flag)) apply.mutate(spec.id)
  }
  const removeFlagGrants = (flag: string) => {
    const grantIds = new Set(grantPowersForFlag(character, flag).map((s) => s.id))
    for (const effect of character.activeEffects ?? []) {
      if (grantIds.has(effect.catalogId)) remove.mutate(effect.id)
    }
  }
  return { applyFlagGrants, removeFlagGrants }
}
