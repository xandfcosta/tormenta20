import { type JSX, createSignal } from 'solid-js'
import { z } from 'zod'
import type { CreateItemInput } from '@/shared/api/api'
import type { FieldErrors } from '@/shared/lib/form-errors'
import { Button } from '@/shared/ui/button'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/shared/ui/dialog'
import { FieldFrame, isInvalid } from '@/shared/ui/field-frame'
import { NumberInput } from '@/shared/ui/number-input'
import { TextField } from '@/shared/ui/text-field'
import {
  ITEM_DIALOG_CONTENT,
  ITEM_DIALOG_TITLE,
  ItemDialogError,
  ItemDialogFooter,
} from './item-dialog-kit'
import { itemWriteMessage } from './item-mutations'

/** A custom inventory item: a name plus a whole-number quantity and a load in
 *  half-slot increments (T20 tracks encumbrance in 0,5-espaço steps). */
const itemFormSchema = z.object({
  name: z.string().trim().min(1, 'Informe um nome.').max(80, 'Máximo 80 caracteres.'),
  quantity: z
    .number()
    .int('Quantidade deve ser inteiro ≥ 1.')
    .min(1, 'Quantidade deve ser inteiro ≥ 1.')
    .max(9999, 'Máximo 9999.'),
  slots: z
    .number()
    .min(0.5, 'Espaços deve ser múltiplo de 0,5 (mínimo 0,5).')
    .max(9999, 'Máximo 9999.')
    .refine((v) => Number.isInteger(v * 2), 'Espaços deve ser múltiplo de 0,5.'),
})

export type ItemFormValues = { name: string; quantity: number; slots: number }

export type ItemFormDialogProps = {
  title: string
  submitLabel: string
  /** Receives the opener; Kobalte has no `asChild`, and every caller brings its
   *  own labelled button. */
  trigger: (open: () => void) => JSX.Element
  /** Seeds the form when editing; leave undefined to create. */
  initial?: ItemFormValues
  onSubmit: (input: CreateItemInput) => Promise<void>
}

/**
 * The form behind both "novo item custom" and per-item edit. Catalog-based
 * items go through the catálogo dialog instead — this one only ever carries
 * name/quantity/slots.
 *
 * @example
 * <ItemFormDialog title="Novo item" submitLabel="Adicionar"
 *   trigger={(open) => <Button onClick={open}>Custom</Button>}
 *   onSubmit={(input) => actions().add(input)} />
 */
export function ItemFormDialog(props: ItemFormDialogProps) {
  const [open, setOpen] = createSignal(false)
  const [name, setName] = createSignal(props.initial?.name ?? '')
  const [quantity, setQuantity] = createSignal(props.initial?.quantity ?? 1)
  const [slots, setSlots] = createSignal(props.initial?.slots ?? 1)
  const [fieldErrors, setFieldErrors] = createSignal<FieldErrors>({})
  const [formError, setFormError] = createSignal<string | null>(null)
  const [pending, setPending] = createSignal(false)

  /** Reopening must not show the last attempt's leftovers. */
  const reset = () => {
    setName(props.initial?.name ?? '')
    setQuantity(props.initial?.quantity ?? 1)
    setSlots(props.initial?.slots ?? 1)
    setFieldErrors({})
    setFormError(null)
  }

  const submit = async (event: SubmitEvent) => {
    event.preventDefault()
    setFormError(null)
    const parsed = itemFormSchema.safeParse({
      name: name(),
      quantity: quantity(),
      slots: slots(),
    })
    if (!parsed.success) {
      setFieldErrors(z.flattenError(parsed.error).fieldErrors as FieldErrors)
      return
    }
    setFieldErrors({})
    setPending(true)
    try {
      await props.onSubmit(parsed.data)
      setOpen(false)
      reset()
    } catch (failure) {
      setFormError(itemWriteMessage(failure, 'Não foi possível salvar o item.'))
    } finally {
      setPending(false)
    }
  }

  return (
    <>
      {props.trigger(() => {
        reset()
        setOpen(true)
      })}
      <Dialog open={open()} onOpenChange={setOpen}>
        <DialogContent class={ITEM_DIALOG_CONTENT}>
          <DialogHeader>
            <DialogTitle class={ITEM_DIALOG_TITLE}>{props.title}</DialogTitle>
          </DialogHeader>
          <form class="space-y-4" onSubmit={submit} noValidate>
            <TextField
              name="item-name"
              label="Nome"
              value={name()}
              onInput={setName}
              errors={fieldErrors().name}
            />
            <div class="grid grid-cols-2 gap-3">
              <FieldFrame name="item-quantity" label="Quantidade" errors={fieldErrors().quantity}>
                <NumberInput
                  id="item-quantity"
                  value={quantity()}
                  onChange={setQuantity}
                  min={1}
                  max={9999}
                  step={1}
                  aria-invalid={isInvalid(fieldErrors().quantity)}
                />
              </FieldFrame>
              <FieldFrame
                name="item-slots"
                label="Espaços"
                hint="Múltiplo de 0,5 (ex.: 0,5 / 1 / 1,5)."
                errors={fieldErrors().slots}
              >
                <NumberInput
                  id="item-slots"
                  value={slots()}
                  onChange={setSlots}
                  min={0.5}
                  max={9999}
                  step={0.5}
                  aria-invalid={isInvalid(fieldErrors().slots)}
                />
              </FieldFrame>
            </div>
            <ItemDialogError message={formError()} />
            <ItemDialogFooter>
              <Button type="button" variant="outline" onClick={() => setOpen(false)}>
                Cancelar
              </Button>
              <Button type="submit" disabled={pending()}>
                {pending() ? 'Salvando…' : props.submitLabel}
              </Button>
            </ItemDialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </>
  )
}
