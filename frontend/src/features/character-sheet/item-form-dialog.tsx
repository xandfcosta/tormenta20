import { type ReactNode, useState } from 'react'
import { Info } from 'lucide-react'
import { useForm } from '@tanstack/react-form'
import { z } from 'zod'
import { getCatalogItem } from '@tormenta20/t20-data'
import { Button } from '@/shared/ui/button'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/shared/ui/dialog'
import { Field, FieldError, FieldLabel } from '@/shared/ui/field'
import { Input } from '@/shared/ui/input'
import { NumberInput } from '@/shared/ui/number-input'
import type {
  CharacterItem,
  CreateItemInput,
} from '@/shared/api/api'
import { accentStrong, dimText, subtleText } from '@/shared/lib/sheet-theme'
import { cn } from '@/shared/lib/utils'
import { CatalogInfoBody } from './catalog-info-body'
import { formatLoad } from './item-describe'

/** A custom inventory item: a name plus a whole-number quantity and a load in
 *  half-slot increments (T20 tracks encumbrance in 0,5-espaço steps). */
const itemFormSchema = z.object({
  name: z
    .string()
    .trim()
    .min(1, 'Informe um nome.')
    .max(80, 'Máximo 80 caracteres.'),
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

/**
 * Read-only info dialog for a single inventory row. Falls back to a
 * "custom item" message when the row has no catalog link.
 */
export function ItemInfoDialog({ item }: { item: CharacterItem }) {
  const catalog = item.catalogId ? getCatalogItem(item.catalogId) : undefined
  return (
    <Dialog>
      <DialogTrigger asChild>
        <Button
          type="button"
          variant="ghost"
          size="icon"
          className={cn(
            'size-7',
            subtleText,
            'hover:bg-muted hover:text-foreground dark:hover:bg-muted dark:hover:text-foreground',
          )}
          aria-label={`Informações de ${item.name}`}
        >
          <Info className="size-3.5" />
        </Button>
      </DialogTrigger>
      <DialogContent
        className={cn(
          'w-[calc(100vw-1.5rem)] max-w-[calc(100vw-1.5rem)] p-4 sm:w-full sm:max-w-md sm:p-6',
          'border-border bg-muted text-foreground   ',
        )}
      >
        <DialogHeader>
          <DialogTitle className={cn(accentStrong)}>
            {item.name}
          </DialogTitle>
        </DialogHeader>
        <div className="space-y-3">
          <div
            className={cn(
              'rounded-md border px-3 py-2 text-xs',
              'border-border bg-muted  ',
            )}
          >
            <p>
              quantidade <span className="font-mono">{item.quantity}</span> •
              espaços <span className="font-mono">{formatLoad(item.slots)}</span>{' '}
              • total{' '}
              <span className={cn('font-mono font-semibold', accentStrong)}>
                {formatLoad(item.quantity * item.slots)}
              </span>
            </p>
            <p className={dimText}>
              equipado:{' '}
              {item.equipped
                ? item.equipped === 'wielded'
                  ? '1 mão'
                  : item.equipped === 'wielded2'
                    ? '2 mãos'
                    : 'vestido'
                : '—'}
            </p>
          </div>
          {catalog ? (
            <CatalogInfoBody catalog={catalog} />
          ) : (
            <p className={cn('text-xs', dimText)}>
              Item customizado, sem dados de catálogo.
            </p>
          )}
        </div>
      </DialogContent>
    </Dialog>
  )
}

/**
 * Form dialog used for both "novo item custom" (create) and per-item
 * edit. `initial` seeds the form; leave undefined for create. The
 * shape passes name/quantity/slots — catalog-based items go through
 * `AddCatalogItemDialog` instead.
 */
export function ItemFormDialog({
  title,
  submitLabel,
  trigger,
  initial,
  onSubmit,
}: {
  title: string
  submitLabel: string
  trigger: ReactNode
  initial?: Partial<CreateItemInput>
  onSubmit: (input: CreateItemInput, onError: (e: Error) => void) => void
}) {
  const [open, setOpen] = useState(false)
  const [formError, setFormError] = useState<string | null>(null)

  const form = useForm({
    defaultValues: {
      name: initial?.name ?? '',
      quantity: initial?.quantity ?? 1,
      slots: initial?.slots ?? 1,
    },
    validators: { onSubmit: itemFormSchema },
    onSubmit: ({ value }) => {
      setFormError(null)
      onSubmit(
        { name: value.name.trim(), quantity: value.quantity, slots: value.slots },
        (e) => setFormError(e.message),
      )
      setOpen(false)
      form.reset()
    },
  })

  const close = (next: boolean) => {
    setOpen(next)
    if (!next) {
      form.reset()
      setFormError(null)
    }
  }

  return (
    <Dialog open={open} onOpenChange={close}>
      <DialogTrigger asChild>{trigger}</DialogTrigger>
      <DialogContent
        className={cn(
          'w-[calc(100vw-1.5rem)] max-w-[calc(100vw-1.5rem)] p-4 sm:w-full sm:max-w-sm sm:p-6',
          'border-border bg-muted text-foreground   ',
        )}
      >
        <DialogHeader>
          <DialogTitle className={cn(accentStrong)}>
            {title}
          </DialogTitle>
        </DialogHeader>
        <form
          className="space-y-4"
          onSubmit={(e) => {
            e.preventDefault()
            e.stopPropagation()
            form.handleSubmit()
          }}
        >
          <form.Field name="name" validators={{ onChange: itemFormSchema.shape.name }}>
            {(f) => {
              const invalid = f.state.meta.isTouched && !f.state.meta.isValid
              return (
                <Field data-invalid={invalid}>
                  <FieldLabel htmlFor={f.name}>Nome</FieldLabel>
                  <Input
                    id={f.name}
                    value={f.state.value}
                    onChange={(e) => f.handleChange(e.target.value)}
                    onBlur={f.handleBlur}
                    placeholder="Ex: Espada longa"
                    autoFocus
                    maxLength={80}
                    aria-invalid={invalid}
                  />
                  {invalid && <FieldError errors={f.state.meta.errors} />}
                </Field>
              )
            }}
          </form.Field>
          <div className="grid grid-cols-2 gap-3">
            <form.Field
              name="quantity"
              validators={{ onChange: itemFormSchema.shape.quantity }}
            >
              {(f) => {
                const invalid = f.state.meta.isTouched && !f.state.meta.isValid
                return (
                  <Field data-invalid={invalid}>
                    <FieldLabel htmlFor={f.name}>Quantidade</FieldLabel>
                    <NumberInput
                      id={f.name}
                      value={f.state.value}
                      onChange={(v) => f.handleChange(v)}
                      onBlur={f.handleBlur}
                      min={1}
                      max={9999}
                      step={1}
                      aria-invalid={invalid}
                    />
                    {invalid && <FieldError errors={f.state.meta.errors} />}
                  </Field>
                )
              }}
            </form.Field>
            <form.Field
              name="slots"
              validators={{ onChange: itemFormSchema.shape.slots }}
            >
              {(f) => {
                const invalid = f.state.meta.isTouched && !f.state.meta.isValid
                return (
                  <Field data-invalid={invalid}>
                    <FieldLabel htmlFor={f.name}>Espaços</FieldLabel>
                    <NumberInput
                      id={f.name}
                      value={f.state.value}
                      onChange={(v) => f.handleChange(v)}
                      onBlur={f.handleBlur}
                      min={0.5}
                      max={9999}
                      step={0.5}
                      aria-invalid={invalid}
                    />
                    {invalid && <FieldError errors={f.state.meta.errors} />}
                  </Field>
                )
              }}
            </form.Field>
          </div>
          <p className={cn('text-[11px]', dimText)}>
            Espaços é múltiplo de 0,5 (ex.: 0,5 / 1 / 1,5).
          </p>
          {formError && (
            <p className="text-xs text-destructive" role="alert">
              {formError}
            </p>
          )}
          <div className="flex justify-end gap-2">
            <Button type="button" variant="outline" onClick={() => close(false)}>
              Cancelar
            </Button>
            <form.Subscribe
              selector={(s) => s.canSubmit}
              children={(canSubmit) => (
                <Button type="submit" disabled={!canSubmit}>
                  {submitLabel}
                </Button>
              )}
            />
          </div>
        </form>
      </DialogContent>
    </Dialog>
  )
}
