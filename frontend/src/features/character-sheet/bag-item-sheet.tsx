import { Pencil, Sparkles, Trash2 } from 'lucide-solid'
import { For, Show, createMemo, createSignal } from 'solid-js'
import type { CharacterItem, EquippedSlot } from '@/shared/api/api'
import { getCatalogItem } from '@/shared/lib/catalog-cache'
import { Button } from '@/shared/ui/button'
import { ConfirmDialog } from '@/shared/ui/confirm-dialog'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/shared/ui/dialog'
import { DialogInlineError } from '@/shared/ui/dialog-inline-error'
import { cn } from '@/shared/lib/utils'
import { CatalogInfoBody } from './catalog-info-body'
import { ConsumeAction } from './consume-action'
import { equipOptionsFor } from './equip-options'
import {
  ITEM_DIALOG_CONTENT,
  ITEM_DIALOG_TITLE,
  ItemDialogFooter,
  ItemDialogMeta,
  ItemDialogSection,
} from './item-dialog-kit'
import {
  formatLoad,
  itemOverlayCatalogs,
  itemOverlayNames,
  overlayNotesSummary,
} from './item-describe'
import { ItemFormDialog } from './item-form-dialog'
import { type ItemActions, itemWriteMessage } from './item-mutations'
import { OverlayPickerDialog, acceptsOverlays } from './overlay-picker-dialog'

export type BagItemSheetProps = {
  item: CharacterItem
  proficient: boolean
  open: boolean
  onOpenChange: (open: boolean) => void
  actions: ItemActions
}

/**
 * Action sheet for one bag item — the target behind every tile and every
 * equipped card. Standard item-dialog structure: title + chips, meta line, then
 * the titled sections (Equipar, Usar, Ficha do item) and the action footer.
 */
export function BagItemSheet(props: BagItemSheetProps) {
  const catalog = createMemo(() =>
    props.item.catalogId ? getCatalogItem(props.item.catalogId) : undefined,
  )
  const overlays = createMemo(() => itemOverlayNames(props.item))
  const overlayable = createMemo(() => {
    const entry = catalog()
    return Boolean(entry && acceptsOverlays(entry))
  })

  const [refusal, setRefusal] = createSignal<string | null>(null)

  /**
   * The sheet only closes once the write went through, and a refusal ("Limite
   * de 2 mãos atingido") lands INSIDE it. Not a toast: Kobalte marks every
   * sibling of an open modal `aria-hidden`, so a toast fired from here is
   * invisible to a screen reader and easy to miss with the eyes on the dialog.
   */
  const runAndClose = async (write: Promise<void>, fallback: string) => {
    setRefusal(null)
    try {
      await write
      props.onOpenChange(false)
    } catch (failure) {
      setRefusal(itemWriteMessage(failure, fallback))
    }
  }

  const pickEquip = (value: '' | EquippedSlot) =>
    runAndClose(
      props.actions.change(props.item.id, { equipped: value || null }),
      'Não foi possível equipar o item.',
    )

  return (
    <Dialog open={props.open} onOpenChange={props.onOpenChange}>
      <DialogContent class={cn(ITEM_DIALOG_CONTENT, 'space-y-3')}>
        <DialogHeader>
          <DialogTitle class={ITEM_DIALOG_TITLE}>
            {props.item.name}
            <For each={overlays()}>
              {(name) => (
                <span class="rounded-md border border-border bg-muted px-1 text-[10px] font-medium text-muted-foreground">
                  {name}
                </span>
              )}
            </For>
          </DialogTitle>
        </DialogHeader>

        <ItemDialogMeta>
          {props.item.quantity} × {formatLoad(props.item.slots)} espaço ={' '}
          <span class="font-semibold">{formatLoad(props.item.quantity * props.item.slots)}</span>
          <Show when={!props.proficient && props.item.equipped !== null}>
            <span class="ml-2 font-semibold text-destructive">sem proficiência</span>
          </Show>
        </ItemDialogMeta>

        <DialogInlineError message={refusal()} />

        <EquipActions item={props.item} onPick={pickEquip} />

        <Show when={catalog()?.consumable}>
          {(consumable) => (
            <ItemDialogSection title="Usar">
              <ConsumeAction
                consumable={consumable()}
                itemName={props.item.name}
                onConsume={(input) =>
                  runAndClose(
                    props.actions.consume(props.item, input),
                    'Não foi possível usar o item.',
                  )
                }
                trigger={(open) => (
                  <Button
                    type="button"
                    variant="outline"
                    class="w-full gap-2 text-emerald-400"
                    onClick={open}
                  >
                    <Sparkles aria-hidden="true" class="size-4" />
                    Usar
                  </Button>
                )}
              />
            </ItemDialogSection>
          )}
        </Show>

        <ItemDialogSection title="Ficha do item">
          <div class="max-h-[34vh] space-y-3 overflow-y-auto rounded-sm border border-border bg-muted px-3 py-2">
            <Show
              when={catalog()}
              fallback={
                <p class="text-xs text-muted-foreground">
                  Item customizado, sem dados de catálogo.
                </p>
              }
            >
              {(entry) => <CatalogInfoBody catalog={entry()} />}
            </Show>
            <AppliedOverlays item={props.item} />
          </div>
        </ItemDialogSection>

        {/* The label names the buttons that are actually there: a poção shows
            no Gem button, and announcing "melhorias" over an absent control
            sends the player looking for it. */}
        <ItemDialogFooter
          label={overlayable() ? 'melhorias · editar · remover' : 'editar · remover'}
        >
          <OverlayPickerDialog
            item={props.item}
            onApply={(input) => props.actions.change(props.item.id, input)}
          />
          <ItemFormDialog
            title={`Editar ${props.item.name}`}
            submitLabel="Salvar"
            initial={{
              name: props.item.name,
              quantity: props.item.quantity,
              slots: props.item.slots,
            }}
            onSubmit={(input) => props.actions.change(props.item.id, input)}
            trigger={(open) => (
              <Button
                type="button"
                variant="ghost"
                size="icon"
                class="size-7"
                aria-label={`Editar ${props.item.name}`}
                onClick={open}
              >
                <Pencil aria-hidden="true" class="size-3.5" />
              </Button>
            )}
          />
          <ConfirmDialog
            title={`Remover "${props.item.name}"?`}
            confirmLabel="Remover"
            onConfirm={() =>
              runAndClose(props.actions.remove(props.item.id), 'Não foi possível remover o item.')
            }
            trigger={(open) => (
              <Button
                type="button"
                variant="ghost"
                size="icon"
                class="size-7 text-foreground hover:bg-destructive/20 hover:text-destructive"
                aria-label={`Remover ${props.item.name}`}
                onClick={open}
              >
                <Trash2 aria-hidden="true" class="size-3.5" />
              </Button>
            )}
          />
        </ItemDialogFooter>
      </DialogContent>
    </Dialog>
  )
}

/** Applied melhorias + material with their effects — `CatalogInfoBody` only
 *  knows the host item, so the overlays get their own block. */
function AppliedOverlays(props: { item: CharacterItem }) {
  const overlays = createMemo(() => itemOverlayCatalogs(props.item))
  return (
    <Show when={overlays().length > 0}>
      <div class="space-y-1 text-xs">
        <p class="text-[10px] uppercase tracking-widest text-muted-foreground">
          Melhorias & material
        </p>
        <For each={overlays()}>
          {(overlay) => (
            <p>
              <span class="font-semibold text-grimorio-gold">{overlay.name}</span>
              <span class="ml-2 text-muted-foreground">
                {overlayNotesSummary(overlay.modifiers) || 'sem efeito mecânico'}
              </span>
            </p>
          )}
        </For>
      </div>
    </Show>
  )
}

const SLOT_ACTION_LABEL: Record<'' | EquippedSlot, string> = {
  '': 'Guardar',
  vested: 'Vestir',
  wielded: 'Empunhar (1 mão)',
  wielded2: 'Empunhar (2 mãos)',
}

/**
 * One button per REACHABLE equip state — the current one is omitted (a stowed
 * item is already in the bag, so no "Guardar"; an empunhado item offers only
 * Guardar/Vestir/the other grip).
 */
function EquipActions(props: {
  item: CharacterItem
  onPick: (value: '' | EquippedSlot) => void
}) {
  const options = createMemo(() => {
    const current = props.item.equipped ?? ''
    return equipOptionsFor(props.item).filter((option) => option.value !== current)
  })

  return (
    <Show when={options().length > 0}>
      <ItemDialogSection title="Equipar">
        <div class="grid grid-cols-2 gap-2">
          <For each={options()}>
            {(option) => (
              <Button
                type="button"
                variant="outline"
                class="h-9 text-xs"
                onClick={() => props.onPick(option.value)}
              >
                {SLOT_ACTION_LABEL[option.value]}
              </Button>
            )}
          </For>
        </div>
      </ItemDialogSection>
    </Show>
  )
}
