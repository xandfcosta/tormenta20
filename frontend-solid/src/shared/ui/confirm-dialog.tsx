import { type JSX, Show, createSignal } from 'solid-js'
import { Button } from './button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from './dialog'

export type ConfirmDialogProps = {
  /**
   * Render prop instead of the React kit's `trigger` node: Kobalte has no
   * `asChild`, and the callers here pass their own styled icon button with its
   * own pt-BR `aria-label`. Handing them the opener keeps that element native.
   */
  trigger: (open: () => void) => JSX.Element
  title: string
  description?: string
  confirmLabel?: string
  cancelLabel?: string
  destructive?: boolean
  onConfirm: () => void
}

/**
 * Themed stand-in for `window.confirm` on destructive actions. Owns its open
 * state, so the caller only says what to ask and what to do on yes.
 *
 * @example
 * <ConfirmDialog
 *   title='Remover "Espada longa"?'
 *   confirmLabel="Remover"
 *   onConfirm={() => actions().remove(item.id)}
 *   trigger={(open) => <button onClick={open} aria-label="Remover">…</button>}
 * />
 */
export function ConfirmDialog(props: ConfirmDialogProps) {
  const [open, setOpen] = createSignal(false)
  const confirm = () => {
    props.onConfirm()
    setOpen(false)
  }

  return (
    <>
      {props.trigger(() => setOpen(true))}
      <Dialog open={open()} onOpenChange={setOpen}>
        <DialogContent class="max-w-sm">
          <DialogHeader>
            <DialogTitle>{props.title}</DialogTitle>
            <Show when={props.description}>
              {(description) => <DialogDescription>{description()}</DialogDescription>}
            </Show>
          </DialogHeader>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => setOpen(false)}>
              {props.cancelLabel ?? 'Cancelar'}
            </Button>
            <Button
              type="button"
              variant={(props.destructive ?? true) ? 'destructive' : 'default'}
              onClick={confirm}
            >
              {props.confirmLabel ?? 'Confirmar'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}
