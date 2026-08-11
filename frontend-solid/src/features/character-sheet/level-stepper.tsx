import { useQueryClient } from '@tanstack/solid-query'
import { ChevronDown, ChevronUp } from 'lucide-solid'
import { For, Show, createSignal } from 'solid-js'
import type { Character } from '@/shared/api/api'
import { Button } from '@/shared/ui/button'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/shared/ui/dialog'
import { MAX_LEVEL, eligibleClasses, levelActions } from './level-mutations'

/**
 * Total level with its stepper. A single-class character steps straight; a
 * multiclass one is ASKED which class takes the level — guessing would silently
 * put a Bardo level on the Guerreiro.
 */
export function LevelStepper(props: { character: Character }) {
  const queryClient = useQueryClient()
  const actions = () => levelActions(queryClient, props.character.id)
  const [picking, setPicking] = createSignal<'up' | 'down' | null>(null)
  const [pending, setPending] = createSignal(false)

  const atMax = () => props.character.level >= MAX_LEVEL
  const atMin = () => props.character.level <= 1

  const bump = async (className: string, direction: 'up' | 'down') => {
    setPending(true)
    try {
      await actions().bump(className, direction === 'up' ? 1 : -1)
    } catch {
      // levelActions already rolled back and told the player.
    } finally {
      setPending(false)
    }
  }

  const step = (direction: 'up' | 'down') => {
    const eligible = eligibleClasses(props.character, direction)
    if (eligible.length === 1) {
      void bump(eligible[0].className, direction)
      return
    }
    setPicking(direction)
  }

  return (
    <>
      {/* The NAME lives on the group, not on the number: a <p> has role
          `paragraph`, which takes no accessible name — the label was being
          dropped silently. */}
      <fieldset
        aria-label="Nível"
        class="flex items-center gap-0.5 rounded-sm border border-grimorio-iron bg-[var(--grimorio-panel-raised)] px-1 py-0.5 text-center sm:gap-1 sm:px-2 sm:py-1"
      >
        <StepButton
          direction="down"
          disabled={atMin() || pending()}
          onClick={() => step('down')}
        />
        <div class="flex flex-col items-center leading-none">
          <p class="text-[8px] uppercase tracking-widest text-muted-foreground sm:text-[9px]">Nv</p>
          <p class="w-5 text-center text-lg font-bold leading-none text-grimorio-gold sm:w-7 sm:text-2xl">
            {props.character.level}
          </p>
        </div>
        <StepButton direction="up" disabled={atMax() || pending()} onClick={() => step('up')} />
      </fieldset>

      <Show when={picking()}>
        {(direction) => (
          <ClassLevelPicker
            character={props.character}
            direction={direction()}
            onPick={(className) => {
              setPicking(null)
              void bump(className, direction())
            }}
            onClose={() => setPicking(null)}
          />
        )}
      </Show>
    </>
  )
}

function StepButton(props: { direction: 'up' | 'down'; disabled: boolean; onClick: () => void }) {
  return (
    <button
      type="button"
      disabled={props.disabled}
      onClick={() => props.onClick()}
      aria-label={props.direction === 'up' ? 'Aumentar nível' : 'Diminuir nível'}
      class="flex size-7 items-center justify-center text-foreground transition-colors hover:text-grimorio-gold disabled:opacity-30 sm:size-6"
    >
      {props.direction === 'up' ? (
        <ChevronUp aria-hidden="true" class="size-3.5 sm:size-4" />
      ) : (
        <ChevronDown aria-hidden="true" class="size-3.5 sm:size-4" />
      )}
    </button>
  )
}

function ClassLevelPicker(props: {
  character: Character
  direction: 'up' | 'down'
  onPick: (className: string) => void
  onClose: () => void
}) {
  const eligible = () => eligibleClasses(props.character, props.direction)
  const verb = () => (props.direction === 'up' ? 'Subir' : 'Descer')

  return (
    <Dialog open onOpenChange={(open) => !open && props.onClose()}>
      <DialogContent class="max-w-sm">
        <DialogHeader>
          <DialogTitle class="font-heading uppercase tracking-wide text-grimorio-gold">
            {verb()} nível — escolha a classe
          </DialogTitle>
        </DialogHeader>
        <Show
          when={eligible().length > 0}
          fallback={
            <p class="text-xs italic text-muted-foreground">
              Nenhuma classe pode {verb().toLowerCase()} de nível.
            </p>
          }
        >
          <div class="flex flex-col gap-2">
            <For each={eligible()}>
              {(entry) => (
                <Button
                  type="button"
                  variant="outline"
                  class="justify-between"
                  onClick={() => props.onPick(entry.className)}
                >
                  <span>{entry.className}</span>
                  <span class="font-mono text-muted-foreground">
                    {entry.level} → {entry.level + (props.direction === 'up' ? 1 : -1)}
                  </span>
                </Button>
              )}
            </For>
          </div>
        </Show>
      </DialogContent>
    </Dialog>
  )
}
