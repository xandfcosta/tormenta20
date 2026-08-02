import {
  AlertTriangle,
  Pencil,
  Sparkles,
  Trash2,
} from 'lucide-react'
import { useState } from 'react'
import { type AnyFieldApi, useForm } from '@tanstack/react-form'
import { getCatalogItem } from '@tormenta20/t20-data'
import { Button } from '@/shared/ui/button'
import { ConfirmDialog } from '@/shared/ui/confirm-dialog'
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
import type {
  CharacterItem,
  ConsumeItemInput,
  UpdateItemInput,
} from '@/shared/api/api'
import {
  accentStrong,
  dimText,
  hoverRow,
  selectClass,
  subtleText,
} from '@/shared/lib/sheet-theme'
import { cn } from '@/shared/lib/utils'
import {
  OverlayPickerDialog,
} from './catalog-picker-dialog'
import { type InstantRoll, rollValueSchema } from './consume-roll'
import { formatLoad } from './item-describe'
import { ItemFormDialog, ItemInfoDialog } from './item-form-dialog'

const EQUIP_OPTIONS: { value: '' | 'vested' | 'wielded' | 'wielded2'; label: string }[] = [
  { value: '', label: '—' },
  { value: 'vested', label: 'Vestido' },
  { value: 'wielded', label: '1 mão' },
  { value: 'wielded2', label: '2 mãos' },
]

/**
 * One inventory row — desktop `sm:` layout is a horizontal grid;
 * mobile stacks the metadata below the item name. Both variants
 * expose the same action cluster (info, overlay, use, edit, delete).
 */
export function InventoryRow({
  item,
  proficient,
  onUpdate,
  onDelete,
  onConsume,
}: {
  item: CharacterItem
  proficient: boolean
  onUpdate: (input: UpdateItemInput, onError: (e: Error) => void) => void
  onDelete: () => void
  onConsume: (input?: ConsumeItemInput) => void
}) {
  const total = item.quantity * item.slots
  const catalog = item.catalogId ? getCatalogItem(item.catalogId) : undefined
  const consumable = catalog?.consumable
  const equipped = item.equipped !== null
  const proficiencyWarning = equipped && !proficient ? (
    <Tooltip>
      <TooltipTrigger asChild>
        <span
          className="inline-flex size-5 items-center justify-center rounded-full bg-red-100 text-red-700 dark:bg-red-950/40 dark:text-red-400"
          aria-label="Sem proficiência"
        >
          <AlertTriangle className="size-3" />
        </span>
      </TooltipTrigger>
      <TooltipContent>
        {catalog?.category.startsWith('weapon-')
          ? 'Sem proficiência: -5 nos testes de ataque'
          : 'Sem proficiência: não aplica Des na Defesa'}
      </TooltipContent>
    </Tooltip>
  ) : null
  const useButton = consumable ? (
    <ConsumeAction
      consumable={consumable}
      itemName={item.name}
      onConsume={onConsume}
    />
  ) : null
  const editTrigger = (
    <Button
      type="button"
      variant="ghost"
      size="icon"
      className={cn(
        'size-7',
        subtleText,
        'hover:bg-muted hover:text-foreground dark:hover:bg-muted dark:hover:text-foreground',
      )}
      aria-label={`Editar ${item.name}`}
    >
      <Pencil className="size-3.5" />
    </Button>
  )
  const deleteButton = (
    <ConfirmDialog
      title={`Remover "${item.name}"?`}
      confirmLabel="Remover"
      onConfirm={onDelete}
      trigger={
        <Button
          type="button"
          variant="ghost"
          size="icon"
          className="size-7 text-foreground hover:bg-red-100 hover:text-red-700  dark:hover:bg-red-950/40 dark:hover:text-red-400"
          aria-label={`Remover ${item.name}`}
        >
          <Trash2 className="size-3.5" />
        </Button>
      }
    />
  )
  return (
    <>
      <div
        className={cn(
          'hidden items-center gap-2 rounded-md px-2 py-1 sm:flex',
          hoverRow,
        )}
      >
        <span className="flex flex-1 items-center gap-1.5 truncate text-sm text-foreground ">
          <span className="truncate">{item.name}</span>
          {item.equipped !== null && (
            <span className="shrink-0 rounded border border-primary/50 px-1 text-[9px] font-semibold uppercase text-primary">
              equipado
            </span>
          )}
          {proficiencyWarning}
        </span>
        <span className="w-12 text-center font-mono text-xs text-foreground ">
          {item.quantity}
        </span>
        <span className="w-14 text-center font-mono text-xs text-foreground ">
          {formatLoad(item.slots)}
        </span>
        <span
          className={cn(
            'w-14 text-right font-mono text-xs font-semibold',
            accentStrong,
          )}
        >
          {formatLoad(total)}
        </span>
        <EquipSelect
          value={item.equipped}
          onChange={(v) => onUpdate({ equipped: v }, () => {})}
          itemName={item.name}
          className="w-20"
        />
        <ItemInfoDialog item={item} />
        <OverlayPickerDialog item={item} onUpdate={onUpdate} />
        {useButton}
        <ItemFormDialog
          title={`Editar ${item.name}`}
          submitLabel="Salvar"
          trigger={editTrigger}
          initial={{ name: item.name, quantity: item.quantity, slots: item.slots }}
          onSubmit={(input, fail) => onUpdate(input, fail)}
        />
        {deleteButton}
      </div>
      <div
        className={cn(
          'flex items-center gap-2 rounded-md px-2 py-1.5 sm:hidden',
          hoverRow,
        )}
      >
        <div className="flex min-w-0 flex-1 flex-col leading-tight">
          <span className="flex items-center gap-1.5 truncate text-sm text-foreground ">
            <span className="truncate">{item.name}</span>
            {item.equipped !== null && (
            <span className="shrink-0 rounded border border-primary/50 px-1 text-[9px] font-semibold uppercase text-primary">
              equipado
            </span>
          )}
            {proficiencyWarning}
          </span>
          <span className={cn('truncate text-[10px]', dimText)}>
            {item.quantity} × {formatLoad(item.slots)} ={' '}
            <span className={cn('font-semibold', accentStrong)}>
              {formatLoad(total)}
            </span>
          </span>
        </div>
        <EquipSelect
          value={item.equipped}
          onChange={(v) => onUpdate({ equipped: v }, () => {})}
          itemName={item.name}
          className="w-20"
        />
        <ItemInfoDialog item={item} />
        <OverlayPickerDialog item={item} onUpdate={onUpdate} />
        {useButton}
        <ItemFormDialog
          title={`Editar ${item.name}`}
          submitLabel="Salvar"
          trigger={editTrigger}
          initial={{ name: item.name, quantity: item.quantity, slots: item.slots }}
          onSubmit={(input, fail) => onUpdate(input, fail)}
        />
        {deleteButton}
      </div>
    </>
  )
}

type Consumable = NonNullable<
  NonNullable<ReturnType<typeof getCatalogItem>>['consumable']
>
const SCOPE_LABEL: Record<Consumable['scope'], string> = {
  instant: 'imediato',
  scene: '1 cena',
  day: '1 dia',
}

/**
 * "Usar" action for a consumable. When the item's instant gain rolls a die
 * (e.g. Bálsamo restaurador = 2d4 PV) it opens a dialog explaining the roll
 * and taking the player's result; fixed-gain / effect consumables apply
 * straight away.
 */
function ConsumeAction({
  consumable,
  itemName,
  onConsume,
}: {
  consumable: Consumable
  itemName: string
  onConsume: (input?: ConsumeItemInput) => void
}) {
  const instant =
    consumable.scope === 'instant' ? consumable.instant : undefined
  const hp = rollable(instant?.hp)
  const mp = rollable(instant?.mp)

  const button = (onClick?: () => void) => (
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
  )

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

function EquipSelect({
  value,
  onChange,
  itemName,
  className,
}: {
  value: CharacterItem['equipped']
  onChange: (next: CharacterItem['equipped']) => void
  itemName: string
  className?: string
}) {
  return (
    <select
      value={value ?? ''}
      onChange={(e) => onChange((e.target.value || null) as CharacterItem['equipped'])}
      className={cn(selectClass, 'h-7 px-1 font-mono text-[11px]', className)}
      aria-label={`Equipar ${itemName}`}
    >
      {EQUIP_OPTIONS.map((opt) => (
        <option key={opt.value} value={opt.value}>
          {opt.label}
        </option>
      ))}
    </select>
  )
}
