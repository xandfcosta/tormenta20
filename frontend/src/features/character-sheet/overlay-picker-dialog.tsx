import { familyFor } from '@/shared/api/item-classify'
import type { CatalogItem } from '@/shared/api/item-types'
import { Gem } from 'lucide-solid'
import { For, Show, createMemo, createSignal } from 'solid-js'
import { parseImprovementIds } from '@/entities/character/derived'
import type { CharacterItem, UpdateItemInput } from '@/shared/api/api'
import { catalogImprovements, catalogMaterials, getCatalogItem } from '@/shared/lib/catalog-cache'
import { Button } from '@/shared/ui/button'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/shared/ui/dialog'
import { DialogInlineError } from '@/shared/ui/dialog-inline-error'
import {
  ITEM_DIALOG_CONTENT,
  ITEM_DIALOG_TITLE,
  ItemDialogFooter,
  ItemDialogSection,
} from './item-dialog-kit'
import { overlayNotesSummary } from './item-describe'
import { itemWriteMessage } from './item-mutations'

/**
 * Categories that never take a melhoria or a special material: you do not
 * forge a potion out of aço-rubi. Returning false here is what hides the
 * whole control, instead of opening a dialog with two empty lists.
 *
 * @example acceptsOverlays(getCatalogItem('balsamo-restaurador')!) // false
 */
export function acceptsOverlays(catalog: CatalogItem): boolean {
  const closed = ['consumable', 'meal', 'catalyst', 'improvement', 'material', 'animal', 'vehicle']
  return !closed.includes(catalog.category)
}

export type OverlayPickerDialogProps = {
  item: CharacterItem
  onApply: (input: UpdateItemInput) => Promise<void>
}

/**
 * Melhorias + material of an already-owned item. Both lists are filtered by
 * the item's own family (`appliesTo`), so a shield is never offered a bowstring
 * upgrade. Renders nothing at all for an item whose category takes no overlays.
 */
export function OverlayPickerDialog(props: OverlayPickerDialogProps) {
  const catalog = createMemo(() =>
    props.item.catalogId ? getCatalogItem(props.item.catalogId) : undefined,
  )
  const eligible = createMemo(() => {
    const entry = catalog()
    return entry && acceptsOverlays(entry) ? entry : undefined
  })

  return <Show when={eligible()}>{(entry) => <OverlayPicker {...props} catalog={entry()} />}</Show>
}

function OverlayPicker(props: OverlayPickerDialogProps & { catalog: CatalogItem }) {
  const [open, setOpen] = createSignal(false)
  const [improvements, setImprovements] = createSignal<string[]>([])
  const [material, setMaterial] = createSignal<string | null>(null)
  const [formError, setFormError] = createSignal<string | null>(null)
  const [pending, setPending] = createSignal(false)

  const family = createMemo(() => familyFor(props.catalog))
  const availableImprovements = createMemo(() =>
    catalogImprovements().filter((imp) => imp.appliesTo?.includes(family())),
  )
  const availableMaterials = createMemo(() =>
    catalogMaterials().filter((mat) => mat.appliesTo?.includes(family())),
  )

  /** Opening always starts from what the item currently wears — a cancelled
   *  edit must not survive into the next open. */
  const reset = () => {
    setImprovements(parseImprovementIds(props.item.improvements))
    setMaterial(props.item.material)
    setFormError(null)
  }

  const toggleImprovement = (id: string) => {
    setImprovements((previous) =>
      previous.includes(id) ? previous.filter((x) => x !== id) : [...previous, id],
    )
  }

  const apply = async () => {
    setFormError(null)
    setPending(true)
    try {
      await props.onApply({ improvements: improvements(), material: material() })
      setOpen(false)
    } catch (failure) {
      setFormError(itemWriteMessage(failure, 'Não foi possível aplicar as melhorias.'))
    } finally {
      setPending(false)
    }
  }

  return (
    <>
      <Button
        type="button"
        variant="ghost"
        size="icon"
        class="size-7 text-muted-foreground hover:text-foreground"
        aria-label={`Melhorias e material de ${props.item.name}`}
        onClick={() => {
          reset()
          setOpen(true)
        }}
      >
        <Gem aria-hidden="true" class="size-3.5" />
      </Button>

      <Dialog open={open()} onOpenChange={setOpen}>
        <DialogContent class={ITEM_DIALOG_CONTENT}>
          <DialogHeader>
            <DialogTitle class={ITEM_DIALOG_TITLE}>
              Melhorias & Material — {props.item.name}
            </DialogTitle>
          </DialogHeader>
          <div class="space-y-4">
            <ItemDialogSection title="Melhorias">
              <Show
                when={availableImprovements().length > 0}
                fallback={<EmptyList text="Nenhuma melhoria compatível." />}
              >
                <ul class="mt-2 space-y-1">
                  <For each={availableImprovements()}>
                    {(improvement) => (
                      <OverlayOption
                        kind="checkbox"
                        id={`improvement-${improvement.id}`}
                        name={`improvements-${props.item.id}`}
                        label={improvement.name}
                        note={overlayNotesSummary(improvement.modifiers)}
                        checked={improvements().includes(improvement.id)}
                        onSelect={() => toggleImprovement(improvement.id)}
                      />
                    )}
                  </For>
                </ul>
              </Show>
            </ItemDialogSection>

            <ItemDialogSection title="Material">
              <Show
                when={availableMaterials().length > 0}
                fallback={<EmptyList text="Nenhum material compatível." />}
              >
                <ul class="mt-2 space-y-1">
                  <OverlayOption
                    kind="radio"
                    id={`material-none-${props.item.id}`}
                    name={`material-${props.item.id}`}
                    label="nenhum"
                    note=""
                    checked={material() === null}
                    onSelect={() => setMaterial(null)}
                  />
                  <For each={availableMaterials()}>
                    {(candidate) => (
                      <OverlayOption
                        kind="radio"
                        id={`material-${candidate.id}-${props.item.id}`}
                        name={`material-${props.item.id}`}
                        label={candidate.name}
                        note={overlayNotesSummary(candidate.modifiers)}
                        checked={material() === candidate.id}
                        onSelect={() => setMaterial(candidate.id)}
                      />
                    )}
                  </For>
                </ul>
              </Show>
            </ItemDialogSection>

            <DialogInlineError message={formError()} />
            <ItemDialogFooter>
              <Button type="button" variant="ghost" onClick={() => setOpen(false)}>
                Cancelar
              </Button>
              <Button type="button" onClick={apply} disabled={pending()}>
                {pending() ? 'Aplicando…' : 'Aplicar'}
              </Button>
            </ItemDialogFooter>
          </div>
        </DialogContent>
      </Dialog>
    </>
  )
}

function OverlayOption(props: {
  kind: 'checkbox' | 'radio'
  id: string
  name: string
  label: string
  note: string
  checked: boolean
  onSelect: () => void
}) {
  return (
    <li>
      <div class="flex items-start gap-2 rounded-md px-2 py-1 text-xs hover:bg-accent">
        <input
          type={props.kind}
          id={props.id}
          name={props.name}
          checked={props.checked}
          onChange={() => props.onSelect()}
          class="mt-0.5 cursor-pointer"
        />
        <label for={props.id} class="flex-1 cursor-pointer">
          <span class="font-semibold">{props.label}</span>
          <Show when={props.note}>
            <span class="ml-2 text-muted-foreground">{props.note}</span>
          </Show>
        </label>
      </div>
    </li>
  )
}

function EmptyList(props: { text: string }) {
  return <p class="mt-1 text-xs italic text-muted-foreground">{props.text}</p>
}
