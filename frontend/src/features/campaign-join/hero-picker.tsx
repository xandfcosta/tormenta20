import { For, Show } from 'solid-js'
import { classLevelLine } from '@/entities/character/class-line'
import type { Character } from '@/shared/api/api'
import { hueFromName } from '@/shared/lib/hue-from-name'
import { cn } from '@/shared/lib/utils'
import { CharacterPortrait } from '@/shared/ui/character-portrait'
import { SectionLabel } from '@/shared/ui/section-label'

export type HeroPickerProps = {
  characters: readonly Character[]
  selectedId: number | null
  onSelect: (id: number) => void
}

/**
 * Which hero walks to the table: the player's roster as portrait plates, the
 * chosen one lit in gold. A picker rather than a dropdown because the choice is
 * about WHO — the face and the class line are the information, and a plate is
 * a tap target on a phone.
 *
 * @example <HeroPicker characters={list()} selectedId={pick()} onSelect={setPick} />
 */
export function HeroPicker(props: HeroPickerProps) {
  return (
    <fieldset class="space-y-3">
      <SectionLabel as="legend" class="mb-3 font-semibold">
        Qual herói entra na mesa?
      </SectionLabel>
      {/* A 2-D region for the shared scene-nav grammar: arrows walk the plates
          by layout, Enter picks (the plates are real buttons). */}
      <div
        data-nav-region="content"
        data-nav-layout="grid"
        class="grid gap-3 sm:grid-cols-2 xl:grid-cols-3"
      >
        <For each={props.characters}>
          {(character) => (
            <HeroPlate
              character={character}
              selected={props.selectedId === character.id}
              onSelect={() => props.onSelect(character.id)}
            />
          )}
        </For>
      </div>
    </fieldset>
  )
}

/** One hero as a selectable plate; the gilt border marks the current pick. */
function HeroPlate(props: { character: Character; selected: boolean; onSelect: () => void }) {
  const name = () => props.character.name
  const subtitle = () =>
    classLevelLine(props.character.classes) || `Nv ${props.character.level ?? 1}`

  return (
    <button
      type="button"
      aria-pressed={props.selected}
      onClick={() => props.onSelect()}
      class={cn(
        'flex items-center gap-3 rounded-none border bg-grimorio-panel p-3 text-left transition-colors',
        props.selected
          ? 'border-grimorio-gold bg-grimorio-panel-raised'
          : 'border-grimorio-iron hover:border-grimorio-gold/60',
      )}
    >
      <CharacterPortrait name={name()} size="sm" hue={hueFromName(name())} />
      <span class="min-w-0 flex-1">
        <span class="block truncate font-medium text-foreground">{name()}</span>
        <span class="block truncate text-xs text-muted-foreground">{subtitle()}</span>
      </span>
      <Show when={props.selected}>
        <span aria-hidden="true" class="shrink-0 text-grimorio-gold">
          ✦
        </span>
      </Show>
    </button>
  )
}
