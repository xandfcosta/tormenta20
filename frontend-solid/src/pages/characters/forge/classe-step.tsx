import { X } from 'lucide-solid'
import { Index, Show, createSignal } from 'solid-js'
import {
  type AttributeSpread,
  type ClassEntry,
  addClassEntry,
  attributeSpreadOf,
  classPresetSpread,
  removeClassEntry,
  setClassLevel,
  totalClassLevel,
} from '@/features/character-build/class-entries'
import { ClassTileGrid } from '@/features/character-build/class-picker'
import { useForge } from '@/features/character-build/forge-context'
import { ClassGrantLines, GrantBox } from '@/features/character-build/grant-panels'
import { Button } from '@/shared/ui/button'
import { ConfirmDialog } from '@/shared/ui/confirm-dialog'
import { NumberInput } from '@/shared/ui/number-input'

/**
 * Second step: the ofício. Same grammar as the lineage step — a grid of tiles,
 * the chosen ones detailed beside it — because a player should learn one way of
 * choosing and use it twice.
 *
 * Two rules of the table live here. Multiclasse is not a level-1 thing, so a
 * SECOND class is confirmed before it lands; and picking the first class fills
 * the suggested attribute spread, which it announces with an undo instead of
 * quietly overwriting whatever the player had typed.
 */
export function ClasseStep() {
  const { draft, options } = useForge()
  const [pendingMulticlass, setPendingMulticlass] = createSignal<string | null>(null)
  const [presetUndo, setPresetUndo] = createSignal<{
    className: string
    previous: AttributeSpread
  } | null>(null)

  const entries = () => draft.values.classes
  const chosenNames = () => entries().map((entry) => entry.className)

  /** Fill the class's suggested spread, remembering what it replaced. */
  const applyPreset = (className: string) => {
    const preset = classPresetSpread(className)
    if (!preset) return
    setPresetUndo({ className, previous: attributeSpreadOf(draft.values) })
    draft.patchValues(preset)
  }

  const undoPreset = () => {
    const undo = presetUndo()
    if (!undo) return
    draft.patchValues(undo.previous)
    setPresetUndo(null)
  }

  const addClass = (className: string) => {
    draft.setValue('classes', addClassEntry(entries(), className))
  }

  const toggle = (className: string) => {
    if (chosenNames().includes(className)) {
      draft.setValue('classes', removeClassEntry(entries(), className))
      return
    }
    if (entries().length === 0) {
      addClass(className)
      applyPreset(className)
      return
    }
    // Second class onward: ask first. The preset stays with the primary — a
    // multiclasse pick must not rewrite attributes chosen for the main class.
    setPendingMulticlass(className)
  }

  const setLevel = (className: string, level: number) => {
    draft.setValue('classes', setClassLevel(entries(), className, level))
  }

  return (
    <section class="flex min-h-0 flex-1 flex-col gap-3" aria-labelledby="forge-step-classe">
      <div class="flex flex-wrap items-baseline gap-x-3 gap-y-1">
        <h2
          id="forge-step-classe"
          class="font-heading text-lg uppercase tracking-[0.16em] text-grimorio-gold"
        >
          Escolha o ofício
        </h2>
        <Show
          when={entries().length > 1}
          fallback={
            <p class="text-xs text-muted-foreground">
              A primeira classe é a principal — ela semeia os PV e a sugestão de atributos.
            </p>
          }
        >
          <p class="text-xs text-muted-foreground">
            Nível total {totalClassLevel(entries())} · multiclasse
          </p>
        </Show>
      </div>

      <Show when={presetUndo()}>
        {(undo) => (
          <p class="flex flex-wrap items-center gap-2 rounded-md border border-grimorio-iron bg-muted/20 px-3 py-2 text-xs text-muted-foreground">
            Atributos preenchidos com a sugestão de {undo().className}.
            <Button type="button" variant="outline" size="sm" onClick={undoPreset}>
              Desfazer
            </Button>
          </p>
        )}
      </Show>

      <div class="grid gap-4 lg:min-h-0 lg:flex-1 lg:grid-cols-[1.15fr_1fr]">
        <div class="p-1 lg:min-h-0 lg:overflow-y-auto">
          <ClassTileGrid options={options.classes} value={chosenNames()} onToggle={toggle} />
        </div>

        <section
          aria-label="Classes escolhidas"
          class="space-y-2 lg:min-h-0 lg:overflow-y-auto lg:pr-1"
        >
          <Show
            when={entries().length > 0}
            fallback={
              <p class="rounded-lg border border-dashed border-grimorio-iron p-4 text-center text-xs text-muted-foreground">
                Escolha um ofício para ver o que ele concede.
              </p>
            }
          >
            {/* `Index`, not `For`: every keystroke in a level field produces a
                NEW entry object, and a reference-keyed For would tear the row
                down mid-typing — the field would lose focus after one digit. */}
            <Index each={entries()}>
              {(entry, i) => (
                <ChosenClass
                  entry={entry()}
                  isPrimary={i === 0}
                  onLevel={(level) => setLevel(entry().className, level)}
                  onRemove={() =>
                    draft.setValue('classes', removeClassEntry(entries(), entry().className))
                  }
                />
              )}
            </Index>
          </Show>
        </section>
      </div>

      {/* The tile is what asks, so the dialog runs controlled: it opens when a
          class is pending and clears the pending one however it closes. */}
      <ConfirmDialog
        open={pendingMulticlass() !== null}
        onOpenChange={(next) => !next && setPendingMulticlass(null)}
        title="Adicionar multiclasse?"
        description="Multiclasse é adquirida em níveis mais altos, pelo Poder de Multiclasse — não é padrão no nível 1. A sugestão de atributos continua sendo a da classe principal."
        confirmLabel="Adicionar mesmo assim"
        destructive={false}
        onConfirm={() => {
          const className = pendingMulticlass()
          if (className) addClass(className)
        }}
      />
    </section>
  )
}

function ChosenClass(props: {
  entry: ClassEntry
  isPrimary: boolean
  onLevel: (level: number) => void
  onRemove: () => void
}) {
  const levelId = () => `class-level-${props.entry.className}`
  return (
    <GrantBox
      title={props.isPrimary ? `${props.entry.className} · principal` : props.entry.className}
    >
      <div class="flex items-end gap-2">
        <div class="w-24">
          <label
            for={levelId()}
            class="mb-1 block text-[11px] font-semibold uppercase tracking-wide text-muted-foreground"
          >
            Nível
          </label>
          <NumberInput
            id={levelId()}
            min={1}
            max={20}
            value={props.entry.level}
            onChange={props.onLevel}
            aria-label={`Nível de ${props.entry.className}`}
          />
        </div>
        <Show when={!props.isPrimary}>
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={() => props.onRemove()}
            aria-label={`Tirar ${props.entry.className}`}
          >
            <X aria-hidden="true" class="size-3.5" />
            Tirar
          </Button>
        </Show>
      </div>
      <ClassGrantLines className={props.entry.className} level={props.entry.level} />
    </GrantBox>
  )
}
