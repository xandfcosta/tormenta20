import { Select as KSelect } from '@kobalte/core/select'
import { CheckIcon, ChevronDownIcon } from 'lucide-solid'
import { useSceneContainer } from '@/shared/lib/scene-container'
import { cn } from '@/shared/lib/utils'

export type SelectOption<T> = {
  value: T
  label: string
  disabled?: boolean
}

export type SelectProps<T> = {
  options: SelectOption<T>[]
  value: SelectOption<T> | null
  onChange: (option: SelectOption<T> | null) => void
  placeholder?: string
  size?: 'sm' | 'default'
  disabled?: boolean
  class?: string
  /** Required when there's no visible <Label> pointing at the trigger. */
  'aria-label'?: string
  /** Portal target. Defaults to the enclosing grimório scene, else body. */
  mount?: Node
}

/**
 * Single-choice select.
 *
 * This is the one primitive whose SHAPE changed: Radix took `<SelectItem>`
 * children, Kobalte is data-driven (`options` + `itemComponent`). Rather than
 * leak that, the kit owns a small interface — pass `options`, get a value
 * back — so scenes don't couple to either library.
 *
 * @example
 * <Select options={races()} value={race()} onChange={setRace} placeholder="Raça" />
 */
export function Select<T>(props: SelectProps<T>) {
  const scene = useSceneContainer()
  return (
    <KSelect<SelectOption<T>>
      options={props.options}
      value={props.value ?? undefined}
      onChange={(option) => props.onChange(option ?? null)}
      optionValue="value"
      optionTextValue="label"
      optionDisabled="disabled"
      disabled={props.disabled}
      placeholder={props.placeholder}
      itemComponent={(itemProps) => (
        <KSelect.Item
          item={itemProps.item}
          class="relative flex w-full cursor-default items-center gap-2 rounded-none py-1.5 pr-8 pl-2 text-sm outline-hidden select-none data-[disabled]:pointer-events-none data-[disabled]:opacity-50 data-[highlighted]:bg-accent data-[highlighted]:text-accent-foreground"
        >
          <KSelect.ItemLabel>{itemProps.item.rawValue.label}</KSelect.ItemLabel>
          <KSelect.ItemIndicator class="absolute right-2 flex size-3.5 items-center justify-center">
            <CheckIcon class="size-4" />
          </KSelect.ItemIndicator>
        </KSelect.Item>
      )}
    >
      <KSelect.Trigger
        data-slot="select-trigger"
        data-size={props.size ?? 'default'}
        aria-label={props['aria-label']}
        class={cn(
          "flex w-fit items-center justify-between gap-2 rounded-sm border border-input bg-transparent px-3 py-2 text-sm whitespace-nowrap shadow-xs transition-[color,box-shadow] outline-none focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-ring/50 disabled:cursor-not-allowed disabled:opacity-50 aria-invalid:border-destructive aria-invalid:ring-destructive/20 data-[size=default]:h-9 data-[size=sm]:h-8 dark:bg-input/30 dark:hover:bg-input/50 [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*='size-'])]:size-4",
          props.class,
        )}
      >
        <KSelect.Value<SelectOption<T>> data-slot="select-value" class="line-clamp-1">
          {(state) => state.selectedOption().label}
        </KSelect.Value>
        <KSelect.Icon>
          <ChevronDownIcon class="size-4 opacity-50" />
        </KSelect.Icon>
      </KSelect.Trigger>
      <KSelect.Portal mount={props.mount ?? scene() ?? undefined}>
        <KSelect.Content
          data-slot="select-content"
          class="z-50 min-w-32 overflow-hidden rounded-sm border bg-popover text-popover-foreground shadow-md data-[closed]:animate-out data-[closed]:fade-out-0 data-[expanded]:animate-in data-[expanded]:fade-in-0"
        >
          <KSelect.Listbox class="p-1" />
        </KSelect.Content>
      </KSelect.Portal>
    </KSelect>
  )
}
