import { For, Show } from 'solid-js'
import type { JSX } from 'solid-js'
import { Input } from './input'
import { Label } from './label'

export type TextFieldProps = {
  /** Doubles as the input's `id` and `name`, so the label associates by `for`. */
  name: string
  label: string
  type?: 'text' | 'email' | 'password'
  autocomplete?: JSX.InputHTMLAttributes<HTMLInputElement>['autocomplete']
  value: string
  onInput: (value: string) => void
  /** Validation messages; a non-empty list flips the field to the error style. */
  errors?: string[]
  hint?: string
}

/**
 * A labelled input with its validation messages — the shape every form field in
 * the app takes. Replaces the React kit's `Field`/`FieldLabel`/`FieldError`
 * trio, which existed to give shadcn's compound API somewhere to hang; one
 * component covers every use here.
 *
 * `aria-invalid` drives the error ring, so screen readers and the styling agree.
 *
 * @example <TextField name="email" label="E-mail" type="email" value={email()} onInput={setEmail} />
 */
export function TextField(props: TextFieldProps) {
  const invalid = () => (props.errors?.length ?? 0) > 0
  return (
    <div class="space-y-2">
      <Label for={props.name}>{props.label}</Label>
      <Input
        id={props.name}
        name={props.name}
        type={props.type ?? 'text'}
        autocomplete={props.autocomplete}
        value={props.value}
        onInput={(e) => props.onInput(e.currentTarget.value)}
        aria-invalid={invalid()}
      />
      <Show when={!invalid() && props.hint}>
        {(hint) => <p class="text-xs text-muted-foreground">{hint()}</p>}
      </Show>
      <For each={props.errors}>{(message) => <p class="text-sm text-destructive">{message}</p>}</For>
    </div>
  )
}
