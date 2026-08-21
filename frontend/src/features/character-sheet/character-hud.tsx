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
    <div class={cn('border-t border-grimorio-iron bg-grimorio-panel px-3 py-2 sm:px-4', props.class)}>
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
            class="h-auto w-24 self-stretch rounded-none border-2 border-grimorio-iron text-4xl"
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
              </div>
            </div>
            {/* Class badges and condition pips share ONE row — a dedicated
                conditions row doubled the nameplate height. O STEPPER DE NÍVEL
                desceu para cá (ALE-183): na faixa do nome ele disputava a
                largura com o chip de Defesa e truncava o herói para "Tanqu…" a
                390px. Aqui há folga, e nenhuma fileira nova é criada.

                Ele NÃO podia simplesmente sair no telefone, que era a proposta
                da issue: este stepper é o ÚNICO lugar do app que muda o nível
                de um personagem — não existe tela de edição, e a Forja só cria.
                Tirá-lo seria tirar a capacidade, não mover um controle. */}
            <div class="flex flex-wrap items-center gap-1">
              <ClassBadges character={props.character} />
              <ConditionPips character={props.character} mini />
              <LevelStepper character={props.character} />
            </div>
            <VitalRows character={props.character} class="mt-auto" />
          </div>
        </div>

        {/* From md the attribute boxes ride along — this IS the Vitais content,
            which is why the phone layout keeps its own Vitais section and these
            widths do not. Os mesmos números moram na aba Combate desde a
            ALE-145, que é como o telefone (onde este bloco é `hidden`) passou a
            alcançá-los. */}
        {/* `max-lg:landscape:hidden` (ALE-162): o gate era só de LARGURA, e o
            celular deitado tem largura de tablet com altura de telefone — a
            844×390 este bloco media 134px e, somado ao crachá de 152, deixava
            a barra de abas em y=401 numa tela de 390. A navegação inteira da
            ficha ficava inalcançável, e a página não rola.

            Esconder aqui não esconde o dado: os mesmos números moram na aba
            Combate desde a ALE-145, que é exatamente como o telefone em pé já
            os alcança — este bloco sempre foi `hidden` lá.

            A chave é largura + ORIENTAÇÃO, nunca altura: media query de altura
            é proibida na casa porque o teclado virtual muda a altura e
            reconstrói o componente no meio da digitação. */}
        <div class="hidden min-w-0 flex-1 flex-col justify-center gap-1.5 md:flex max-lg:landscape:hidden">
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
