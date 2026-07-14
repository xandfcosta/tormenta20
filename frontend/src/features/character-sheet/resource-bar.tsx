import { useState } from 'react'
import { useForm } from '@tanstack/react-form'
import { z } from 'zod'
import { Minus, Pencil, Plus } from 'lucide-react'
import { Button } from '@/shared/ui/button'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/shared/ui/dialog'
import { Field, FieldError, FieldLabel } from '@/shared/ui/field'
import { NumberInput } from '@/shared/ui/number-input'
import { accentStrong, dimText } from '@/shared/lib/sheet-theme'
import { cn } from '@/shared/lib/utils'

/** How much PV/PM to add or remove — a whole number of at least 1. Clamping to
 *  the resource's [0, max] happens at apply time, not here. */
const resourceAdjustSchema = z.object({
  amount: z
    .number()
    .int('Quantidade deve ser inteiro.')
    .min(1, 'Informe uma quantidade ≥ 1.')
    .max(9999, 'Máximo 9999.'),
})

/** Clamp a projected resource total into [0, max]; returns the value + whether
 *  it hit a bound (so the UI can hint "limitado"). */
function clampResource(raw: number, max: number) {
  const value = Math.max(0, Math.min(max, raw))
  return { value, clamped: value !== raw }
}

export function ResourceBar({
  label,
  current,
  max,
  fromColor,
  toColor,
  accent,
  className,
  onSetCurrent,
}: {
  label: string
  current: number
  max: number
  fromColor: string
  toColor: string
  accent: string
  className?: string
  onSetCurrent?: (next: number) => void
}) {
  const pct = max > 0 ? Math.max(0, Math.min(100, (current / max) * 100)) : 0
  return (
    <div
      className={cn(
        'rounded-lg border p-2.5',
        'border-border bg-muted  ',
        className,
      )}
    >
      <div className="flex items-baseline justify-between">
        <p className={cn('text-[10px] uppercase tracking-[0.3em]', accent)}>
          {label}
        </p>
        <p className="font-mono text-base">
          <span className="font-bold">{current}</span>
          <span className={dimText}> / {max}</span>
        </p>
      </div>
      <div className="mt-1.5 h-2.5 overflow-hidden rounded-full border border-border bg-muted  ">
        <div
          className={`h-full bg-gradient-to-r ${fromColor} ${toColor} transition-all`}
          style={{ width: `${pct}%` }}
        />
      </div>
      {onSetCurrent && (
        <ResourceControls
          label={label}
          current={current}
          max={max}
          onSetCurrent={onSetCurrent}
        />
      )}
    </div>
  )
}

function ResourceControls({
  label,
  current,
  max,
  onSetCurrent,
}: {
  label: string
  current: number
  max: number
  onSetCurrent: (next: number) => void
}) {
  return (
    <div className="mt-2 flex items-center gap-1.5">
      <Button
        type="button"
        variant="outline"
        size="icon"
        className="size-7"
        disabled={current <= 0}
        onClick={() => onSetCurrent(current - 1)}
        aria-label={`Reduzir ${label} em 1`}
      >
        <Minus className="size-3.5" />
      </Button>
      <Button
        type="button"
        variant="outline"
        size="icon"
        className="size-7"
        disabled={current >= max}
        onClick={() => onSetCurrent(current + 1)}
        aria-label={`Aumentar ${label} em 1`}
      >
        <Plus className="size-3.5" />
      </Button>
      <ResourceAdjustDialog
        label={label}
        current={current}
        max={max}
        onSetCurrent={onSetCurrent}
      />
    </div>
  )
}

function ResourceAdjustDialog({
  label,
  current,
  max,
  onSetCurrent,
}: {
  label: string
  current: number
  max: number
  onSetCurrent: (next: number) => void
}) {
  const [open, setOpen] = useState(false)
  const [mode, setMode] = useState<'add' | 'remove'>('remove')

  const form = useForm({
    defaultValues: { amount: 0 },
    validators: { onSubmit: resourceAdjustSchema },
    onSubmit: ({ value }) => {
      const delta = mode === 'add' ? value.amount : -value.amount
      onSetCurrent(clampResource(current + delta, max).value)
      close(false)
    },
  })

  const close = (next: boolean) => {
    setOpen(next)
    if (!next) {
      form.reset()
      setMode('remove')
    }
  }

  return (
    <Dialog open={open} onOpenChange={close}>
      <DialogTrigger asChild>
        <Button
          type="button"
          variant="outline"
          size="icon"
          className="size-7"
          aria-label={`Editar ${label}`}
        >
          <Pencil className="size-3.5" />
        </Button>
      </DialogTrigger>
      <DialogContent
        className={cn(
          'w-[calc(100vw-1.5rem)] max-w-[calc(100vw-1.5rem)] p-4 sm:w-full sm:max-w-sm sm:p-6',
          'border-border bg-muted text-foreground   ',
        )}
      >
        <DialogHeader>
          <DialogTitle className={cn(accentStrong)}>
            Ajustar {label}
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
          <div className="grid grid-cols-2 gap-2">
            <Button
              type="button"
              variant={mode === 'remove' ? 'default' : 'outline'}
              onClick={() => setMode('remove')}
              className="gap-1"
            >
              <Minus className="size-4" /> Remover
            </Button>
            <Button
              type="button"
              variant={mode === 'add' ? 'default' : 'outline'}
              onClick={() => setMode('add')}
              className="gap-1"
            >
              <Plus className="size-4" /> Adicionar
            </Button>
          </div>

          <form.Field
            name="amount"
            validators={{ onChange: resourceAdjustSchema.shape.amount }}
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
                    min={0}
                    max={9999}
                    autoFocus
                    aria-invalid={invalid}
                  />
                  {invalid && <FieldError errors={f.state.meta.errors} />}
                </Field>
              )
            }}
          </form.Field>

          <form.Subscribe selector={(s) => s.values.amount}>
            {(amount) => {
              const delta = mode === 'add' ? amount : -amount
              const { value: preview, clamped } = clampResource(
                current + delta,
                max,
              )
              return (
                <div
                  className={cn(
                    'flex items-center justify-between rounded-lg border px-4 py-2',
                    'border-border bg-muted  ',
                  )}
                >
                  <div className="flex flex-col">
                    <span
                      className={cn(
                        'text-[10px] uppercase tracking-widest',
                        dimText,
                      )}
                    >
                      novo total
                    </span>
                    <span className={cn('text-[10px]', dimText)}>
                      {current} {delta >= 0 ? '+' : '−'} {Math.abs(delta)}
                      {clamped && ' (limitado)'}
                    </span>
                  </div>
                  <span
                    className={cn('font-mono text-2xl font-bold', accentStrong)}
                  >
                    {preview}
                    <span className={cn('ml-1 text-sm font-normal', dimText)}>
                      / {max}
                    </span>
                  </span>
                </div>
              )
            }}
          </form.Subscribe>

          <div className="flex justify-end gap-2">
            <Button type="button" variant="outline" onClick={() => close(false)}>
              Cancelar
            </Button>
            <form.Subscribe
              selector={(s) => s.canSubmit}
              children={(canSubmit) => (
                <Button type="submit" disabled={!canSubmit}>
                  Aplicar
                </Button>
              )}
            />
          </div>
        </form>
      </DialogContent>
    </Dialog>
  )
}
