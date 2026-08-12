import { type JSX, Show } from 'solid-js'

/**
 * Failure line for an action taken INSIDE a dialog, announced as it appears.
 *
 * Not a toast on purpose: a Kobalte modal marks everything that is its sibling
 * with `aria-hidden`, and the sonner region is one of those siblings — a toast
 * fired from an open dialog is never announced. Errors raised inside a dialog
 * go inline; toasts are for actions fired outside one.
 *
 * @example <DialogInlineError message={formError()} />
 */
export function DialogInlineError(props: { message: string | null }): JSX.Element {
  return (
    <Show when={props.message}>
      {(message) => (
        <p class="text-xs text-destructive" role="alert">
          {message()}
        </p>
      )}
    </Show>
  )
}
