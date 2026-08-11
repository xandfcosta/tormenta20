import { type ComponentProps, splitProps } from 'solid-js'
import { cn } from '@/shared/lib/utils'

/** `for` is required: a label with no control is the a11y bug this prevents. */
export type LabelProps = ComponentProps<'label'> & { for: string }

/**
 * Form label. The React kit wrapped Radix's Label; Kobalte has no standalone
 * equivalent (its labels belong to TextField), and Radix's only added
 * click-forwarding that a native `for` already gives us — so this is the plain
 * element, with the association made explicit in the type (biome's
 * `noLabelWithoutControl` also insists on seeing the attribute).
 *
 * @example <Label for="email">E-mail</Label>
 */
export function Label(props: LabelProps) {
  const [local, rest] = splitProps(props, ['class', 'for'])
  return (
    // biome-ignore lint/a11y/noLabelWithoutControl: this IS the label primitive, so it can't wrap the control; biome can't follow `for` through props, and LabelProps makes it required.
    <label
      data-slot="label"
      for={local.for}
      class={cn(
        'flex items-center gap-2 text-sm leading-none font-medium select-none group-data-[disabled=true]:pointer-events-none group-data-[disabled=true]:opacity-50 peer-disabled:cursor-not-allowed peer-disabled:opacity-50',
        local.class,
      )}
      {...rest}
    />
  )
}
