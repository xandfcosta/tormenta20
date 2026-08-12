import { For, type ParentProps, Show } from 'solid-js'
import { Label } from './label'

export type FieldFrameProps = ParentProps<{
  /** Doubles as the control's `id`, so the label associates by `for`. */
  name: string
  label: string
  /** Validation messages; a non-empty list replaces the hint. */
  errors?: string[]
  hint?: string
}>

/** True when a field carries validation messages — drives `aria-invalid`. */
export function isInvalid(errors: string[] | undefined): boolean {
  return (errors?.length ?? 0) > 0
}

/**
 * The frame every form field wears: label above, control in the middle, hint or
 * validation messages below. It owns no control of its own, so `TextField` and
 * `TextAreaField` share one shape instead of copying it.
 *
 * @example
 * <FieldFrame name="nome" label="Nome" errors={errors().nome}>
 *   <Input id="nome" aria-invalid={isInvalid(errors().nome)} />
 * </FieldFrame>
 */
export function FieldFrame(props: FieldFrameProps) {
  return (
    <div class="space-y-2">
      <Label for={props.name}>{props.label}</Label>
      {props.children}
      <Show when={!isInvalid(props.errors) && props.hint}>
        {(hint) => <p class="text-xs text-muted-foreground">{hint()}</p>}
      </Show>
      <For each={props.errors}>{(message) => <p class="text-sm text-destructive">{message}</p>}</For>
    </div>
  )
}
