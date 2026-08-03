import { Sparkles } from 'lucide-react'
import { useState } from 'react'
import { type AnyFieldApi, useForm } from '@tanstack/react-form'
import { getCatalogItem } from '@tormenta20/t20-data'
import { Button } from '@/shared/ui/button'
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/shared/ui/dialog'
import { Field, FieldError, FieldLabel } from '@/shared/ui/field'
import { NumberInput } from '@/shared/ui/number-input'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/shared/ui/tooltip'
import type { ConsumeItemInput } from '@/shared/api/api'
import { subtleText } from '@/shared/lib/sheet-theme'
import { cn } from '@/shared/lib/utils'
import { type InstantRoll, rollValueSchema } from './consume-roll'

type Consumable = NonNullable<
  NonNullable<ReturnType<typeof getCatalogItem>>['consumable']
>
const SCOPE_LABEL: Record<Consumable['scope'], string> = {
  instant: 'imediato',
  scene: '1 cena',
  day: '1 dia',
}

/**
 * "Usar" action for a consumable (moved out of the old InventoryRow so the
 * Mochila bag reuses it). When the item's instant gain rolls a die (e.g.
 * Bálsamo restaurador = 2d4 PV) it opens a dialog explaining the roll and
 * taking the player's result; fixed-gain / effect consumables apply straight
 * away. Pass `trigger` to replace the default icon button (the bag's action
 * sheet uses a full-width button).
 */
export function ConsumeAction({
  consumable,
  itemName,
  onConsume,
  trigger,
}: {
  consumable: Consumable
  itemName: string
  onConsume: (input?: ConsumeItemInput) => void
  trigger?: (onClick?: () => void) => React.ReactNode
}) {
  const instant =
    consumable.scope === 'instant' ? consumable.instant : undefined
  const hp = rollable(instant?.hp)
  const mp = rollable(instant?.mp)

  const button =
    trigger ??
    ((onClick?: () => void) => (
      <Button
        type="button"
        variant="ghost"
        size="icon"
        className="size-7 text-emerald-700 hover:bg-emerald-100 hover:text-emerald-800 dark:text-emerald-400 dark:hover:bg-emerald-950/40"
        aria-label={`Usar ${itemName}`}
        onClick={onClick}
      >
        <Sparkles className="size-3.5" />
      </Button>
    ))

  if (!hp && !mp) {
    return (
      <Tooltip>
        <TooltipTrigger asChild>{button(() => onConsume())}</TooltipTrigger>
        <TooltipContent>Usar ({SCOPE_LABEL[consumable.scope]})</TooltipContent>
      </Tooltip>
    )
  }

  return (
    <ConsumeRollDialog
      itemName={itemName}
      hp={hp}
      mp={mp}
      trigger={button()}
      onConsume={onConsume}
    />
  )
}

/** A roll is needed only when the die string isn't the fixed "0". */
function rollable(roll: InstantRoll | undefined): InstantRoll | undefined {
  return roll && roll.dice !== '0' ? roll : undefined
}

function ConsumeRollDialog({
  itemName,
  hp,
  mp,
  trigger,
  onConsume,
}: {
  itemName: string
  hp?: InstantRoll
  mp?: InstantRoll
  trigger: React.ReactNode
  onConsume: (input?: ConsumeItemInput) => void
}) {
  const [open, setOpen] = useState(false)

  // Headless TanStack Form + per-field zod validators: each present roll is
  // validated against its die's range (rollValueSchema) on change, and Apply
  // stays disabled until every field holds a value the die can produce.
  const form = useForm({
    defaultValues: { hp: '', mp: '' },
    onSubmit: ({ value }) => {
      const input: ConsumeItemInput = {}
      if (hp) input.hpRolled = Number(value.hp) + (hp.bonus ?? 0)
      if (mp) input.mpRolled = Number(value.mp) + (mp.bonus ?? 0)
      onConsume(input)
      setOpen(false)
    },
  })

  const close = (next: boolean) => {
    setOpen(next)
    if (!next) form.reset()
  }

  return (
    <Dialog open={open} onOpenChange={close}>
      <DialogTrigger asChild>{trigger}</DialogTrigger>
      <DialogContent className="w-[calc(100vw-1.5rem)] max-w-sm">
        <DialogHeader>
          <DialogTitle>Usar {itemName}</DialogTitle>
        </DialogHeader>
        <form
          className="space-y-3"
          onSubmit={(e) => {
            e.preventDefault()
            e.stopPropagation()
            form.handleSubmit()
          }}
        >
          {hp && (
            <form.Field name="hp" validators={{ onChange: rollValueSchema(hp) }}>
              {(f) => <RollField field={f} label="PV" roll={hp} />}
            </form.Field>
          )}
          {mp && (
            <form.Field name="mp" validators={{ onChange: rollValueSchema(mp) }}>
              {(f) => <RollField field={f} label="PM" roll={mp} />}
            </form.Field>
          )}
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => close(false)}>
              Cancelar
            </Button>
            <form.Subscribe
              selector={(s) => [s.canSubmit, s.isDirty] as const}
              children={([canSubmit, isDirty]) => (
                <Button type="submit" disabled={!canSubmit || !isDirty}>
                  Aplicar
                </Button>
              )}
            />
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

function RollField({
  field,
  label,
  roll,
}: {
  field: AnyFieldApi
  label: string
  roll: InstantRoll
}) {
  const value = field.state.value as string
  const invalid = !field.state.meta.isValid
  const total = (Number(value) || 0) + (roll.bonus ?? 0)
  return (
    <Field data-invalid={invalid}>
      <FieldLabel htmlFor={`roll-${label}`}>
        Role {roll.dice}
        {roll.bonus ? ` + ${roll.bonus}` : ''} de {label} e informe o resultado
        do dado
      </FieldLabel>
      <NumberInput
        id={`roll-${label}`}
        min={0}
        max={999}
        value={value}
        onChange={(v) => field.handleChange(String(v))}
        onBlur={field.handleBlur}
        aria-invalid={invalid}
      />
      {invalid ? (
        <FieldError errors={field.state.meta.errors} />
      ) : (
        <p className={cn('text-xs', subtleText)}>
          {label} recuperado: <span className="font-semibold">{total}</span>
        </p>
      )}
    </Field>
  )
}
