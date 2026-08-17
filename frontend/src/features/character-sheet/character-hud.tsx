import { createMemo } from 'solid-js'
import type { Character } from '@/shared/api/api'
import { useConditionals } from '@/shared/stores/conditionals-context'
import { CharacterPortrait } from '@/shared/ui/character-portrait'
import { cn } from '@/shared/lib/utils'
import { AttributesGrid } from './attributes-grid'
import { CombatStats, SavesStats } from './combat-stats'
import { ConditionPips } from './condition-pips'
import { ContextualStatBlocks } from './contextual-stat-blocks'
import { DefenseChip } from './defense-chip'
import { LevelStepper } from './level-stepper'
import { ClassBadges, SheetIdentityText } from './sheet-identity'
import { VitalRows } from './vital-rows'

/**
 * The HUD pinned to the bottom of the sheet on both layouts: a player
 * "nameplate" — square portrait beside the identity stacked over the PV/PM bars
 * — plus the attribute boxes from tablet up.
 *
 * Este é o HUD do JOGADOR na ficha dele. O painel do combatente do mestre já
 * NÃO o usa: ele tem a `CombatantBand`, que é uma faixa pequena e deliberada
 * (ALE-145). Foi essa mistura que trouxe o `dense`, um segundo layout escondido
 * aqui dentro para caber numa coluna estreita.
 *
 * The React version wrapped this in `memo()`; its comment explains that doing so
 * is what collapsed the ~160ms-per-block-switch floor that survived memoizing
 * the panels. There is no wrapper here: a switch touches only what reads the
 * signal that changed, and `e2e/bench-tabs.mjs` is what settles whether the
 * floor came back (ALE-90).
 */
export function CharacterHud(props: { character: Character; class?: string }) {
  const conditionals = useConditionals()
  const active = createMemo(() => conditionals.active(props.character.id))

  return (
    <div class={cn('border-t border-grimorio-iron bg-[var(--grimorio-panel)] px-3 py-2 sm:px-4', props.class)}>
      {/* Below lg the attribute cluster stacks BELOW the nameplate: side by
          side it stretched the square portrait, which squeezed the PV/PM row
          until its −/+ buttons slid under the cluster and stopped being
          tappable. */}
      <div class="flex flex-col gap-3 lg:flex-row lg:items-stretch lg:gap-4">
        <div class="flex min-w-0 items-stretch gap-3 lg:w-[34rem] lg:flex-none">
          {/* `h-auto w-24 self-stretch`, not `aspect-square`: aspect-ratio
              cannot derive a width from a flex-stretched height, so the
              portrait collapsed to the width of its initials. */}
          <CharacterPortrait
            name={props.character.name}
            size="sm"
            class="h-auto w-24 self-stretch rounded-sm border-2 border-grimorio-iron text-4xl"
          />
          <div class="flex min-w-0 flex-1 flex-col gap-1.5">
            <div class="flex items-start justify-between gap-2">
              <SheetIdentityText character={props.character} activeConditionals={active()} />
              <div class="flex shrink-0 items-center gap-1.5">
                <DefenseChip
                  character={props.character}
                  activeConditionals={active()}
                  class="md:hidden"
                />
                <LevelStepper character={props.character} />
              </div>
            </div>
            {/* Class badges and condition pips share ONE row — a dedicated
                conditions row doubled the nameplate height. */}
            <div class="flex flex-wrap items-center gap-1">
              <ClassBadges character={props.character} />
              <ConditionPips character={props.character} mini />
            </div>
            <VitalRows character={props.character} class="mt-auto" />
          </div>
        </div>

        {/* From md the attribute boxes ride along — this IS the Vitais content,
            which is why the phone layout keeps its own Vitais section and these
            widths do not. Os mesmos números moram na aba Combate desde a
            ALE-145, que é como o telefone (onde este bloco é `hidden`) passou a
            alcançá-los. */}
        <div class="hidden min-w-0 flex-1 flex-col justify-center gap-1.5 md:flex">
          {/* Row A is the reactive numbers (defesa/ataques + as três
              resistências); row B is contextual — fórmulas de arma e o triplo
              de magia — com os atributos ao lado. */}
          <div class="grid gap-2 grid-cols-1 md:grid-cols-2 lg:grid-cols-1 xl:grid-cols-2">
            <CombatStats character={props.character} activeConditionals={active()} />
            <SavesStats character={props.character} activeConditionals={active()} />
          </div>
          <div class="grid gap-2 grid-cols-1 md:grid-cols-2 lg:grid-cols-1 xl:grid-cols-2">
            <AttributesGrid
              character={props.character}
              activeConditionals={active()}
              class="grid-cols-6"
            />
            <ContextualStatBlocks
              character={props.character}
              activeConditionals={active()}
            />
          </div>
        </div>
      </div>
    </div>
  )
}
