import { Combobox as KCombobox } from '@kobalte/core/combobox'
import { ChevronDownIcon } from 'lucide-solid'
import { createSignal } from 'solid-js'
import { matchesQuery } from '@/shared/lib/fuzzy-filter'
import { useSceneContainer } from '@/shared/lib/scene-container'
import { cn } from '@/shared/lib/utils'

export type PickerOption = { value: string; label: string }

export type PickerComboboxProps = {
  options: PickerOption[]
  /** Fired with the picked value; the field clears itself right after. */
  onPick: (value: string) => void
  placeholder?: string
  emptyMessage?: string
  disabled?: boolean
  class?: string
  'aria-label': string
  /** Portal target. Defaults to the enclosing grimório scene, else body. */
  mount?: Node
}

/**
 * Searchable picker that ACTS instead of holding a value: pick "Cego" and it
 * reports the id and empties itself, ready for the next pick. Bound-field
 * comboboxes would keep the last choice selected, which for an "aplicar
 * condição" control means the player can only ever apply one.
 *
 * Kobalte filters nothing on its own — the options are narrowed here with the
 * app's own `matchesQuery`, same as the other ported lists.
 *
 * @example
 * <PickerCombobox
 *   options={conditions()}
 *   aria-label="Aplicar condição"
 *   onPick={(id) => apply(id)}
 * />
 */
export function PickerCombobox(props: PickerComboboxProps) {
  const scene = useSceneContainer()
  const [query, setQuery] = createSignal('')

  const matches = () =>
    props.options.filter((option) => matchesQuery([option.label], query()))

  return (
    <KCombobox<PickerOption>
      options={matches()}
      value={null}
      onChange={(option) => {
        if (!option) return
        props.onPick(option.value)
        setQuery('')
      }}
      onInputChange={setQuery}
      optionValue="value"
      optionTextValue="label"
      optionLabel="label"
      disabled={props.disabled}
      placeholder={props.placeholder}
      // Kobalte's default ('input') only opens the list once the player types,
      // so a click on the field would answer with nothing — this control exists
      // precisely to BROWSE what can be applied.
      triggerMode="focus"
      // Kobalte ships English strings that OVERRIDE any sr-only text of ours.
      translations={{
        focusAnnouncement: (optionText, isSelected) =>
          `${optionText}${isSelected ? ', selecionado' : ''}`,
        // Kobalte types this one by its DEFAULT's return literal ("one option
        // available"), so no other language can satisfy it — upstream typing
        // bug, not a real constraint. The cast is on this line alone.
        countAnnouncement: ((count: number) =>
          count === 1 ? '1 opção disponível' : `${count} opções disponíveis`) as () =>
          | 'one option available'
          | undefined,
        selectedAnnouncement: (optionText) => `${optionText}, selecionado`,
        triggerLabel: 'Ver opções',
        listboxLabel: 'Opções',
      }}
      // Without this the listbox hides on an empty result and the player gets
      // no answer at all to a search that found nothing.
      allowsEmptyCollection
      itemComponent={(itemProps) => (
        <KCombobox.Item
          item={itemProps.item}
          class="flex w-full cursor-default items-center rounded-none px-2 py-1.5 text-sm outline-hidden select-none data-[disabled]:pointer-events-none data-[disabled]:opacity-50 data-[highlighted]:bg-accent data-[highlighted]:text-accent-foreground"
        >
          <KCombobox.ItemLabel>{itemProps.item.rawValue.label}</KCombobox.ItemLabel>
        </KCombobox.Item>
      )}
    >
      <KCombobox.Control class={cn('relative flex w-full items-center', props.class)}>
        <KCombobox.Input
          aria-label={props['aria-label']}
          class="h-9 w-full rounded-sm border border-input bg-transparent px-3 pr-9 text-sm outline-none transition-[color,box-shadow] focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-ring/50 disabled:cursor-not-allowed disabled:opacity-50 dark:bg-input/30"
        />
        <KCombobox.Trigger class="absolute right-1 flex size-7 items-center justify-center rounded-none text-muted-foreground hover:text-foreground">
          <KCombobox.Icon>
            <ChevronDownIcon class="size-4" />
          </KCombobox.Icon>
        </KCombobox.Trigger>
      </KCombobox.Control>
      <KCombobox.Portal mount={props.mount ?? scene() ?? undefined}>
        <KCombobox.Content class="z-50 min-w-32 overflow-hidden rounded-sm border bg-popover text-popover-foreground shadow-md data-[closed]:animate-out data-[closed]:fade-out-0 data-[expanded]:animate-in data-[expanded]:fade-in-0">
          {matches().length === 0 ? (
            <p class="px-3 py-4 text-center text-sm text-muted-foreground">
              {props.emptyMessage ?? 'Nada encontrado.'}
            </p>
          ) : (
            <KCombobox.Listbox class="max-h-60 overflow-y-auto p-1" />
          )}
        </KCombobox.Content>
      </KCombobox.Portal>
    </KCombobox>
  )
}
