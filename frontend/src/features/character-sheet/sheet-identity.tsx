import { For, Show, createMemo } from 'solid-js'
import { computedSheetFor } from '@/entities/character/computed-sheet'
import { characterEffectsWith } from '@/entities/character/derived'
import type { Character } from '@/shared/api/api'
import type { ValueBreakdown } from '@/shared/lib/computed-sheet-v2'
import { Badge } from '@/shared/ui/badge'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/shared/ui/tooltip'
import { cn } from '@/shared/lib/utils'
import { signed } from './signed'

export type SheetIdentityProps = {
  character: Character
  activeConditionals: ReadonlySet<string>
}

/**
 * Who the character is, in one line under the name: races, origin, devotion,
 * size, displacement (with its breakdown when something changed it), flight and
 * the heavy-armor fatigue warning.
 */
export function SheetIdentityText(props: SheetIdentityProps) {
  const sheet = createMemo(() => computedSheetFor(props.character, props.activeConditionals))
  const races = () => props.character.races.map((r) => r.race).join(' / ')
  // Not a breakdown — read the raw flag off the effects (heavy-armor rest
  // penalty); everything numeric comes from the sheet.
  const fatigue = () =>
    characterEffectsWith(props.character, props.activeConditionals).flags.has('fatigue-on-sleep')

  return (
    <div class="min-w-0">
      {/* `line-clamp-2` e não `truncate` (ALE-183): a faixa do nome tem 201px a
          390px — o retrato de 96px e o chip de Defesa ficam com o resto —, e um
          nome de 22 caracteres ("Necromante Nv12 Magias") precisa de 241. O
          stepper de nível já desceu uma fileira e devolveu o que tinha para
          devolver; os 40px que faltavam só sairiam do retrato ou da Defesa, e
          nenhum dos dois vale menos que o nome.
          Mesma decisão que o dono tomou na ALE-167 para o nome do combatente, e
          pelo mesmo motivo: nome é o que se lê em voz alta na mesa. A segunda
          linha só aparece quando ela é necessária. */}
      <h1 class="line-clamp-2 font-heading text-lg font-bold leading-tight tracking-tight sm:text-xl">
        {props.character.name}
      </h1>
      <p class="line-clamp-1 text-xs leading-tight text-muted-foreground sm:line-clamp-2">
        {races()} • {props.character.origin}
        {' • '}
        <span class="text-foreground">
          {props.character.god ?? 'Sem devoção'}
          {props.character.god && props.character.godPower
            ? ` (${props.character.godPower})`
            : ''}
        </span>
        {' • '}
        {props.character.size} • <DisplacementBadge displacement={sheet().displacement} />
        <Show when={sheet().flySpeed > 0}>
          {' • '}
          <span class="text-foreground">voo {sheet().flySpeed}m</span>
        </Show>
        <Show when={fatigue()}>
          {' • '}
          <FatigueWarning />
        </Show>
      </p>
    </div>
  )
}

/** Displacement, underlined when an item changed it — the tooltip audits it. */
function DisplacementBadge(props: { displacement: ValueBreakdown }) {
  return (
    <Show
      when={props.displacement.itemBonus !== 0}
      fallback={<span>{props.displacement.total}m</span>}
    >
      <Tooltip>
        <TooltipTrigger
          as="button"
          type="button"
          class={cn(
            'cursor-help underline decoration-dotted underline-offset-2',
            props.displacement.itemBonus < 0 ? 'text-destructive' : 'text-bonus-ink',
          )}
        >
          {props.displacement.total}m
        </TooltipTrigger>
        <TooltipContent>
          <div class="text-xs">
            <div>Base {props.displacement.base}m</div>
            <For each={props.displacement.contributions}>
              {(contribution) => (
                <div>
                  {contribution.source} {signed(contribution.amount)}m
                </div>
              )}
            </For>
          </div>
        </TooltipContent>
      </Tooltip>
    </Show>
  )
}

function FatigueWarning() {
  return (
    <Tooltip>
      <TooltipTrigger
        as="button"
        type="button"
        class="cursor-help font-semibold text-foreground underline decoration-dotted underline-offset-2"
      >
        Fadiga ao dormir
      </TooltipTrigger>
      <TooltipContent>
        <div class="max-w-[260px] text-xs">
          Dormir vestindo armadura pesada causa Fadiga (1 condição). Remova a armadura antes de
          descansar.
        </div>
      </TooltipContent>
    </Tooltip>
  )
}

/** The character's per-class level badges (e.g. "Bardo 3"). */
export function ClassBadges(props: { character: Character; class?: string }) {
  return (
    <div class={cn('flex gap-1', props.class)}>
      <For each={props.character.classes}>
        {(entry) => (
          <Badge class="px-1.5 py-0 text-3xs leading-tight sm:px-2 sm:py-0.5 sm:text-xs">
            {entry.className} {entry.level}
          </Badge>
        )}
      </For>
    </div>
  )
}
