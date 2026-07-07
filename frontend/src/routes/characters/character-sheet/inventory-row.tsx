import {
  AlertTriangle,
  Pencil,
  Sparkles,
  Trash2,
} from 'lucide-react'
import { getCatalogItem } from '@tormenta20/t20-data'
import { Button } from '@/components/ui/button'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import type {
  CharacterItem,
  UpdateItemInput,
} from '@/lib/api'
import {
  accentStrong,
  dimText,
  hoverRow,
  selectClass,
  subtleText,
} from '@/lib/sheet-theme'
import { cn } from '@/lib/utils'
import {
  OverlayPickerDialog,
} from './catalog-picker-dialog'
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
  onConsume: () => void
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
    <Tooltip>
      <TooltipTrigger asChild>
        <Button
          type="button"
          variant="ghost"
          size="icon"
          className="size-7 text-emerald-700 hover:bg-emerald-100 hover:text-emerald-800 dark:text-emerald-400 dark:hover:bg-emerald-950/40"
          onClick={onConsume}
          aria-label={`Usar ${item.name}`}
        >
          <Sparkles className="size-3.5" />
        </Button>
      </TooltipTrigger>
      <TooltipContent>
        Usar ({consumable.scope === 'instant'
          ? 'imediato'
          : consumable.scope === 'scene'
            ? '1 cena'
            : '1 dia'}
        )
      </TooltipContent>
    </Tooltip>
  ) : null
  const editTrigger = (
    <Button
      type="button"
      variant="ghost"
      size="icon"
      className={cn(
        'size-7',
        subtleText,
        'hover:bg-amber-200/60 hover:text-amber-900 dark:hover:bg-zinc-800/60 dark:hover:text-amber-200',
      )}
      aria-label={`Editar ${item.name}`}
    >
      <Pencil className="size-3.5" />
    </Button>
  )
  const deleteButton = (
    <Tooltip>
      <TooltipTrigger asChild>
        <Button
          type="button"
          variant="ghost"
          size="icon"
          className="size-7 text-zinc-500 hover:bg-red-100 hover:text-red-700 dark:text-zinc-400 dark:hover:bg-red-950/40 dark:hover:text-red-400"
          onClick={() => {
            if (confirm(`Remover "${item.name}"?`)) onDelete()
          }}
          aria-label={`Remover ${item.name}`}
        >
          <Trash2 className="size-3.5" />
        </Button>
      </TooltipTrigger>
      <TooltipContent>Remover item</TooltipContent>
    </Tooltip>
  )
  return (
    <>
      <div
        className={cn(
          'hidden items-center gap-2 rounded-md px-2 py-1 sm:flex',
          hoverRow,
        )}
      >
        <span className="flex flex-1 items-center gap-1.5 truncate text-sm text-zinc-800 dark:text-zinc-200">
          <span className="truncate">{item.name}</span>
          {proficiencyWarning}
        </span>
        <span className="w-12 text-center font-mono text-xs text-zinc-700 dark:text-zinc-300">
          {item.quantity}
        </span>
        <span className="w-14 text-center font-mono text-xs text-zinc-700 dark:text-zinc-300">
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
          <span className="flex items-center gap-1.5 truncate text-sm text-zinc-800 dark:text-zinc-200">
            <span className="truncate">{item.name}</span>
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
