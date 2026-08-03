import { Pencil, Sparkles, Trash2 } from 'lucide-react'
import { getCatalogItem } from '@tormenta20/t20-data'
import { Button } from '@/shared/ui/button'
import { ConfirmDialog } from '@/shared/ui/confirm-dialog'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/shared/ui/dialog'
import type { CharacterItem, EquippedSlot } from '@/shared/api/api'
import { accentStrong, dimText } from '@/shared/lib/sheet-theme'
import { cn } from '@/shared/lib/utils'
import { CatalogInfoBody } from './catalog-info-body'
import { OverlayPickerDialog } from './catalog-picker-dialog'
import { ConsumeAction } from './consume-action'
import { equipOptionsFor } from './equip-options'
import {
  ITEM_DIALOG_CONTENT,
  ItemDialogFooter,
  ItemDialogMeta,
  ItemDialogSection,
  itemDialogTitleClass,
} from './item-dialog-kit'
import {
  formatLoad,
  itemOverlayCatalogs,
  itemOverlayNames,
  overlayNotesSummary,
} from './item-describe'
import { ItemFormDialog } from './item-form-dialog'
import type { ItemMutations } from './use-item-mutations'

/**
 * Action sheet for one bag item — the tap target behind every tile and
 * equipped card. Standard item-dialog structure (item-dialog-kit): title +
 * chips, meta line, then titled sections — Equipar, Usar, Ficha do item,
 * Melhorias & material — and the bordered action footer.
 */
export function BagItemSheet({
  item,
  proficient,
  open,
  onOpenChange,
  mutations,
}: {
  item: CharacterItem
  proficient: boolean
  open: boolean
  onOpenChange: (open: boolean) => void
  mutations: ItemMutations
}) {
  const catalog = item.catalogId ? getCatalogItem(item.catalogId) : undefined
  const consumable = catalog?.consumable
  const overlays = itemOverlayNames(item)

  const setEquipped = (value: '' | EquippedSlot) => {
    mutations.changeItem(item.id, { equipped: value || null }, () => {})
    onOpenChange(false)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className={cn(ITEM_DIALOG_CONTENT, 'space-y-3')}>
        <DialogHeader>
          <DialogTitle className={itemDialogTitleClass}>
            {item.name}
            {overlays.map((n) => (
              <span
                key={n}
                className="rounded border border-border bg-muted px-1 text-[10px] font-medium text-muted-foreground"
              >
                {n}
              </span>
            ))}
          </DialogTitle>
        </DialogHeader>

        <ItemDialogMeta>
          {item.quantity} × {formatLoad(item.slots)} espaço ={' '}
          <span className="font-semibold">
            {formatLoad(item.quantity * item.slots)}
          </span>
          {!proficient && item.equipped !== null && (
            <span className="ml-2 font-semibold text-red-700 dark:text-red-400">
              sem proficiência
            </span>
          )}
        </ItemDialogMeta>

        <EquipActions item={item} onPick={setEquipped} />

        {consumable && (
          <ItemDialogSection title="Usar">
            <ConsumeAction
              consumable={consumable}
              itemName={item.name}
              onConsume={(input) => {
                mutations.consumeItem(item, input)
                onOpenChange(false)
              }}
              trigger={(onClick) => (
                <Button
                  type="button"
                  variant="outline"
                  className="w-full gap-2 text-emerald-700 dark:text-emerald-400"
                  onClick={onClick}
                >
                  <Sparkles className="size-4" />
                  Usar
                </Button>
              )}
            />
          </ItemDialogSection>
        )}

        <ItemDialogSection title="Ficha do item">
          <div className="max-h-[34vh] space-y-3 overflow-y-auto rounded-md border border-border bg-muted/40 px-3 py-2">
            {catalog ? (
              <CatalogInfoBody catalog={catalog} />
            ) : (
              <p className={cn('text-xs', dimText)}>
                Item customizado, sem dados de catálogo.
              </p>
            )}
            <AppliedOverlays item={item} />
          </div>
        </ItemDialogSection>

        <ItemDialogFooter label="melhorias · editar · remover">
          <OverlayPickerDialog
            item={item}
            onUpdate={(input, fail) => mutations.changeItem(item.id, input, fail)}
          />
          <ItemFormDialog
            title={`Editar ${item.name}`}
            submitLabel="Salvar"
            trigger={
              <Button
                type="button"
                variant="ghost"
                size="icon"
                className="size-7"
                aria-label={`Editar ${item.name}`}
              >
                <Pencil className="size-3.5" />
              </Button>
            }
            initial={{ name: item.name, quantity: item.quantity, slots: item.slots }}
            onSubmit={(input, fail) => mutations.changeItem(item.id, input, fail)}
          />
          <ConfirmDialog
            title={`Remover "${item.name}"?`}
            confirmLabel="Remover"
            onConfirm={() => {
              mutations.removeItem(item.id)
              onOpenChange(false)
            }}
            trigger={
              <Button
                type="button"
                variant="ghost"
                size="icon"
                className="size-7 text-foreground hover:bg-red-100 hover:text-red-700 dark:hover:bg-red-950/40 dark:hover:text-red-400"
                aria-label={`Remover ${item.name}`}
              >
                <Trash2 className="size-3.5" />
              </Button>
            }
          />
        </ItemDialogFooter>
      </DialogContent>
    </Dialog>
  )
}

/** Applied melhorias + material with their effects — the base CatalogInfoBody
 *  only knows the host item, so overlays get their own block. */
function AppliedOverlays({ item }: { item: CharacterItem }) {
  const overlays = itemOverlayCatalogs(item)
  if (overlays.length === 0) return null
  return (
    <div className="space-y-1 text-xs">
      <p className={cn('text-[10px] uppercase tracking-widest', dimText)}>
        Melhorias & material
      </p>
      {overlays.map((o) => (
        <p key={o.id}>
          <span className={cn('font-semibold', accentStrong)}>{o.name}</span>
          <span className={cn('ml-2', dimText)}>
            {overlayNotesSummary(o.modifiers) || 'sem efeito mecânico'}
          </span>
        </p>
      ))}
    </div>
  )
}

const SLOT_ACTION_LABEL: Record<'' | EquippedSlot, string> = {
  '': 'Guardar',
  vested: 'Vestir',
  wielded: 'Empunhar (1 mão)',
  wielded2: 'Empunhar (2 mãos)',
}

/** One button per REACHABLE equip state — the current state is omitted (a
 *  stowed item is already in the bag, no "Guardar"; an empunhado item offers
 *  only Guardar/Vestir/other grip). Big targets replace the old select. */
function EquipActions({
  item,
  onPick,
}: {
  item: CharacterItem
  onPick: (value: '' | EquippedSlot) => void
}) {
  const current = item.equipped ?? ''
  const options = equipOptionsFor(item).filter((opt) => opt.value !== current)
  if (options.length === 0) return null
  return (
    <ItemDialogSection title="Equipar">
      <div className="grid grid-cols-2 gap-2">
        {options.map((opt) => (
          <Button
            key={opt.value}
            type="button"
            variant="outline"
            className="h-9 text-xs"
            onClick={() => onPick(opt.value)}
          >
            {SLOT_ACTION_LABEL[opt.value]}
          </Button>
        ))}
      </div>
    </ItemDialogSection>
  )
}
