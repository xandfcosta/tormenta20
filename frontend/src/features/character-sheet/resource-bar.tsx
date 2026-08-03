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

export function ResourceAdjustDialog({
  label,
  current,
  max,
  onSetCurrent,
  onDamage,
  tempPool,
  triggerClassName,
}: {
  label: string
  current: number
  max: number
  onSetCurrent: (next: number) => void
  /** F2: when set, "Remover" routes through the atomic damage endpoint
   *  (temp-PV pool drains first) instead of a plain vitals write. */
  onDamage?: (amount: number) => void
  /** F3: pool-aware VIDA dialog — current temp-PV total + manual pool setter. */
  tempPool?: { total: number; onSetManual: (value: number) => void }
  /** Override the trigger button size — the HUD uses a more compact one. */
  triggerClassName?: string
}) {
  const [open, setOpen] = useState(false)
  const [mode, setMode] = useState<'add' | 'remove'>('remove')

  const form = useForm({
    defaultValues: { amount: 0 },
    validators: { onSubmit: resourceAdjustSchema },
    onSubmit: ({ value }) => {
      if (mode === 'remove' && onDamage) {
        // Unclamped: the server soaks the pool first, remainder floors at 0.
        onDamage(value.amount)
        close(false)
        return
      }
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
          className={cn('size-7', triggerClassName)}
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
                    aria-invalid={invalid}
                  />
                  {/* Quick amounts: most combat deltas land on these — no
                      keyboard needed at the table (audit: autoFocus keyboard
                      pop + 0-default friction). */}
                  <div className="flex gap-1.5">
                    {[5, 10, 20].map((quick) => (
                      <button
                        key={quick}
                        type="button"
                        onClick={() => f.handleChange(quick)}
                        className="rounded-md border border-border px-2.5 py-1 text-xs hover:bg-accent"
                      >
                        {quick}
                      </button>
                    ))}
                  </div>
                  {invalid && <FieldError errors={f.state.meta.errors} />}
                </Field>
              )
            }}
          </form.Field>

          <form.Subscribe selector={(s) => s.values.amount}>
            {(amount) => {
              // Removals soak the temp-PV pool first (F2 routing) — the
              // preview mirrors what the damage endpoint will do.
              const soak =
                mode === 'remove' ? Math.min(amount, tempPool?.total ?? 0) : 0
              const delta = mode === 'add' ? amount : -(amount - soak)
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
                      {soak > 0 && ` · PV temp. absorvem ${soak}`}
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
        {tempPool && (
          <ManualTempHpField
            pool={tempPool}
            onDone={() => close(false)}
          />
        )}
      </DialogContent>
    </Dialog>
  )
}

/**
 * F3 — GM-entered ad-hoc temp-PV pool inside the VIDA ✎ dialog. SETS the
 * manual pool to the typed value (0 remove); vale-o-maior (p256) is enforced
 * server-side, the helper text just reminds the table. Own state, not part
 * of the add/remove form — setting the pool is an independent action.
 */
function ManualTempHpField({
  pool,
  onDone,
}: {
  pool: { total: number; onSetManual: (value: number) => void }
  onDone: () => void
}) {
  const [value, setValue] = useState(pool.total)
  return (
    <div className="space-y-2 rounded-lg border border-border bg-muted p-3">
      <div className="flex items-baseline justify-between">
        <FieldLabel htmlFor="manual-temp-hp">PV temporários</FieldLabel>
        <span className={cn('font-mono text-sm font-bold', accentStrong)}>
          +{pool.total}
        </span>
      </div>
      <div className="flex items-center gap-2">
        <NumberInput
          id="manual-temp-hp"
          value={value}
          onChange={setValue}
          min={0}
          max={9999}
          className="flex-1"
        />
        <Button
          type="button"
          variant="outline"
          onClick={() => {
            pool.onSetManual(value)
            onDone()
          }}
        >
          Definir
        </Button>
      </div>
      <p className={cn('text-[10px]', dimText)}>
        vale o maior — não acumulam (p256) · 0 remove o valor manual
      </p>
    </div>
  )
}
