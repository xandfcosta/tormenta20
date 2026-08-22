import { useQueryClient } from '@tanstack/solid-query'
import { SPELLCASTER_CLASSES } from '@/shared/rules/class-spellcasting'
import { spellPmCostWithMods, validateCast } from '@/shared/rules/rules-spells'
import { firstErrorMessage } from '@/shared/rules/rules-types'
import { SPELL_BASE_PM_COST } from '@/shared/rules/spells'
import type { CatalogSpell } from '@/shared/api/catalog-types'
import { Sparkles, Zap } from 'lucide-solid'
import { For, type JSX, Show, createMemo, createSignal } from 'solid-js'
import {
  castableClassesFor,
  highestCastableCircle,
  spellPmLimitFor,
} from '@/entities/character/spell-rules'
import { ApiError, type Character } from '@/shared/api/api'
import { Button } from '@/shared/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/shared/ui/dialog'
import { DialogInlineError } from '@/shared/ui/dialog-inline-error'
import { NumberInput } from '@/shared/ui/number-input'
import { cn } from '@/shared/lib/utils'
import { augmentPicksFrom, augmentPmFor, isAugmentLocked } from './spell-augments'
import { computedSheetFor } from '@/entities/character/computed-sheet'
import { spellActions } from './spell-mutations'
import { FieldLabel } from '@/shared/ui/section-label'

/**
 * Cast dialog — the player picks stacks per aprimoramento (0 = not taken;
 * `muda` is a checkbox because it cannot stack), over a live PM total checked
 * against the per-spell limit. The server is authoritative; this preview only
 * spares a round-trip on the obvious refusals.
 *
 * `compact` renders the trigger icon-only below `sm` (row headers at phone
 * width) while keeping the labeled button on larger screens; the aria-label
 * carries the spell name either way.
 */
export function CastSpellDialog(props: {
  spell: CatalogSpell
  character: Character
  disabled?: boolean
  compact?: boolean
}) {
  const queryClient = useQueryClient()
  const [open, setOpen] = createSignal(false)
  const [stacks, setStacks] = createSignal<ReadonlyMap<number, number>>(new Map())
  const [pending, setPending] = createSignal(false)
  const [error, setError] = createSignal<string | null>(null)

  /**
   * Highest circle this character can CAST — gates aprimoramentos carrying
   * `requiresCircle`. Never below the spell's own circle: a power-granted spell
   * on a non-caster is castable at that circle, and only that one.
   */
  const castableCircle = createMemo(() =>
    Math.max(
      // Every caster class the character has, not just this spell's list: the
      // gate is "what circles can you reach at all".
      highestCastableCircle(props.character, castableClassesFor(props.character, SPELLCASTER_CLASSES)),
      props.spell.circle,
    ),
  )

  const picks = createMemo(() => augmentPicksFrom(stacks()))
  const basePm = () => SPELL_BASE_PM_COST[props.spell.circle]
  // p226 pelo mesmo caminho do servidor: a redução de custo entra aqui, não
  // acumula entre fontes e nunca leva abaixo de 1 PM. Sem isto o diálogo
  // prometia um preço e a API cobrava outro (ALE-110).
  const pmCostContribs = () => computedSheetFor(props.character).pmCostMod.contributions
  const totalPm = createMemo(() =>
    props.spell.circle === 0
      ? 0
      : spellPmCostWithMods(basePm(), augmentPmFor(props.spell.augments, picks()), pmCostContribs()),
  )
  // The PER-SPELL ceiling (p224), from the same Go function the cast handler
  // runs. It deliberately does NOT match the HUD's "Limite PM" tile: that tile
  // is a per-character summary ("best caster level"), and gating on it offered a
  // Bardo 7 / Arcanista 1 seven PM on an Arcanista spell that the server then
  // refused above 1 (ALE-92). When they differ, the spell's own class wins.
  const perSpellLimit = createMemo(() => spellPmLimitFor(props.character, props.spell.classes))
  const blocked = createMemo(() =>
    firstErrorMessage(
      validateCast({
        circle: props.spell.circle,
        basePm: basePm(),
        totalPm: totalPm(),
        pmLimit: perSpellLimit(),
        mpCurrent: props.character.mpCurrent,
        // Preparation stays server-enforced: the Conjurar button only shows for
        // learned spells, and predicting the prep rule here buys nothing.
        needsPrep: false,
        prepared: true,
      }),
    ),
  )

  const pickStacks = (index: number, next: number) => {
    const map = new Map(stacks())
    if (next <= 0) map.delete(index)
    else map.set(index, next)
    setStacks(map)
  }

  const close = (next: boolean) => {
    setOpen(next)
    if (!next) {
      setStacks(new Map())
      setError(null)
    }
  }

  const cast = async () => {
    setPending(true)
    setError(null)
    try {
      await spellActions(queryClient, props.character.id).cast(props.spell.id, picks())
      close(false)
    } catch (failure) {
      // Inline, not a toast: the sonner region is a sibling of the open modal
      // and Kobalte marks it aria-hidden.
      setError(failure instanceof ApiError ? failure.message : 'Erro ao conjurar')
    } finally {
      setPending(false)
    }
  }

  return (
    <>
      <Button
        type="button"
        size="sm"
        class={cn('h-7 gap-1 text-xs', props.compact && 'shrink-0 px-2 sm:px-3')}
        disabled={props.disabled}
        aria-label={`Conjurar ${props.spell.name}`}
        onClick={() => setOpen(true)}
      >
        <Sparkles aria-hidden="true" class="size-3.5" />
        <span class={props.compact ? 'hidden sm:inline' : undefined}>Conjurar</span>
      </Button>

      <Dialog open={open()} onOpenChange={close}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle class="flex items-center gap-2 font-heading tracking-wide">
              <Zap aria-hidden="true" class="size-5 text-grimorio-crimson-bright" />
              {props.spell.name}
            </DialogTitle>
            <DialogDescription>
              Base {basePm()} PM • Limite por magia {perSpellLimit()} PM • PM atual{' '}
              {props.character.mpCurrent} / {props.character.mpMax}
            </DialogDescription>
          </DialogHeader>

          <Show
            when={props.spell.augments.length > 0 && props.spell.circle > 0}
            fallback={
              <p class="text-xs italic text-muted-foreground">
                {props.spell.circle === 0
                  ? 'Truques não aceitam aprimoramentos.'
                  : 'Esta magia não possui aprimoramentos.'}
              </p>
            }
          >
            <div class="space-y-2">
              <FieldLabel as="p">
                Aprimoramentos
              </FieldLabel>
              <ul class="space-y-2">
                <For each={props.spell.augments}>
                  {(augment, index) => (
                    <AugmentRow
                      augment={augment}
                      index={index()}
                      stacks={stacks().get(index()) ?? 0}
                      locked={isAugmentLocked(augment, castableCircle())}
                      onPick={(next) => pickStacks(index(), next)}
                    />
                  )}
                </For>
              </ul>
            </div>
          </Show>

          <div class="flex items-center justify-between rounded-none border border-border bg-muted px-3 py-2">
            <FieldLabel class="text-xs">
              Custo total
            </FieldLabel>
            <span
              class={cn(
                'font-mono text-lg font-bold',
                blocked() ? 'text-penalty-ink' : 'text-grimorio-gold',
              )}
            >
              {totalPm()} PM
            </span>
          </div>

          <Show when={blocked()}>
            {(reason) => <p class="text-xs text-penalty-ink">{reason()}</p>}
          </Show>
          <DialogInlineError message={error()} />

          <DialogFooter>
            <Button variant="outline" onClick={() => close(false)}>
              Cancelar
            </Button>
            <Button disabled={pending() || Boolean(blocked())} onClick={() => void cast()}>
              <Sparkles aria-hidden="true" class="mr-1 size-4" />
              {pending() ? 'Conjurando…' : 'Conjurar'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}

function AugmentRow(props: {
  augment: CatalogSpell['augments'][number]
  index: number
  stacks: number
  locked: boolean
  onPick: (stacks: number) => void
}): JSX.Element {
  return (
    <li
      class={cn(
        'flex flex-wrap items-start gap-2 rounded-none border border-border p-2',
        props.locked && 'opacity-50',
      )}
    >
      <div class="flex-1 space-y-0.5">
        <p class="text-xs">
          <span
            class={cn(
              'mr-2 font-mono text-3xs uppercase tracking-widest',
              props.augment.kind === 'muda' ? 'text-arcane-ink' : 'text-bonus-ink',
            )}
          >
            {props.augment.kind}
          </span>
          +{props.augment.pmCost} PM {props.augment.kind === 'aumenta' ? 'cada' : ''}
          <Show when={props.locked}>
            <span class="ml-2 font-semibold text-penalty-ink">
              requer {props.augment.requiresCircle}º círculo
            </span>
          </Show>
        </p>
        <p class="text-xs text-foreground">{props.augment.description}</p>
      </div>
      <Show
        when={props.augment.kind === 'muda'}
        fallback={
          <NumberInput
            value={props.stacks}
            onChange={(value) => props.onPick(Math.max(0, value))}
            min={0}
            max={20}
            disabled={props.locked}
            class="w-20"
            aria-label={`Aprimoramento ${props.index + 1} — degraus`}
          />
        }
      >
        {/* 'muda' is single by nature — a checkbox, not a stepper. */}
        <input
          type="checkbox"
          checked={props.stacks > 0}
          disabled={props.locked}
          onChange={(event) => props.onPick(event.currentTarget.checked ? 1 : 0)}
          class="mt-1 size-5 accent-violet-600"
          aria-label={`Aprimoramento: ${props.augment.description.slice(0, 40)}`}
        />
      </Show>
    </li>
  )
}
