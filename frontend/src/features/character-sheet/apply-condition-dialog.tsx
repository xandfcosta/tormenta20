import { Plus } from 'lucide-solid'
import { For, Show, createMemo, createSignal } from 'solid-js'
import { conditionEffectSummary } from '@/shared/rules/condition-modifiers'
import type { ConditionId } from '@/shared/api/catalog-types'
import { Button } from '@/shared/ui/button'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/shared/ui/dialog'
import { DialogInlineError } from '@/shared/ui/dialog-inline-error'
import { Input } from '@/shared/ui/input'
import type { ConditionEditing } from './conditions-section'
import { ITEM_DIALOG_CONTENT, ITEM_DIALOG_TITLE } from './item-dialog-kit'
import { normalize } from './normalize'

/**
 * Aplicar uma condição do livro (p394-395) pelo MESMO gesto do "Aplicar efeito"
 * ao lado: um botão de adicionar que abre um diálogo com busca e lista. Antes
 * esta era a única das duas escolhas do bloco Efeitos que se tomava num campo
 * embutido, e uma escolha desenhada de dois jeitos obriga a aprender duas vezes
 * (ALE-216, o defeito que a ALE-169 consertou noutro lugar).
 *
 * A lista NÃO é virtualizada de propósito: são 35 verbetes fechados no livro, e
 * uma lista virtual mede zero em jsdom — o teste passaria verde sobre um
 * diálogo vazio.
 *
 * @example <ApplyConditionDialog conditions={createConditionEditing(char)} />
 */
export function ApplyConditionDialog(props: { conditions: ConditionEditing }) {
  const [open, setOpen] = createSignal(false)
  const [query, setQuery] = createSignal('')
  const [pending, setPending] = createSignal(false)
  const [formError, setFormError] = createSignal<string | null>(null)

  const matches = createMemo(() => {
    const search = normalize(query().trim())
    const options = props.conditions.options()
    if (!search) return options
    return options.filter((option) => normalize(option.label).includes(search))
  })

  const apply = async (id: string) => {
    setPending(true)
    setFormError(null)
    try {
      await props.conditions.apply(id)
      setOpen(false)
    } catch {
      // Inline, nunca toast: o modal marca os irmãos `aria-hidden` e a região do
      // sonner é irmã, então um toast daqui não é anunciado.
      setFormError('Não foi possível aplicar a condição — a ficha voltou ao valor anterior.')
    } finally {
      setPending(false)
    }
  }

  return (
    <>
      <Button
        type="button"
        variant="outline"
        size="sm"
        class="h-6 gap-1 px-2 text-2xs"
        onClick={() => {
          setQuery('')
          setFormError(null)
          setOpen(true)
        }}
      >
        <Plus aria-hidden="true" class="size-3" />
        Aplicar condição
      </Button>

      <Dialog open={open()} onOpenChange={setOpen}>
        <DialogContent class={ITEM_DIALOG_CONTENT}>
          <DialogHeader>
            <DialogTitle class={ITEM_DIALOG_TITLE}>Aplicar condição</DialogTitle>
          </DialogHeader>
          <div class="space-y-3">
            <Input
              value={query()}
              onInput={(event) => setQuery(event.currentTarget.value)}
              placeholder="Buscar condição…"
              aria-label="Buscar condição"
            />
            <DialogInlineError message={formError()} />
            <Show
              when={matches().length > 0}
              fallback={
                <p class="rounded-sm border border-border bg-muted px-3 py-6 text-center text-sm text-muted-foreground">
                  Nenhuma condição para aplicar.
                </p>
              }
            >
              <ul class="max-h-72 overflow-y-auto rounded-sm border border-border bg-muted p-1">
                <For each={matches()}>
                  {(option) => (
                    <li>
                      <ConditionRow
                        id={option.value as ConditionId}
                        label={option.label}
                        disabled={pending()}
                        onPick={() => void apply(option.value)}
                      />
                    </li>
                  )}
                </For>
              </ul>
            </Show>
          </div>
        </DialogContent>
      </Dialog>
    </>
  )
}

/** O que ela FAZ vem junto do nome — a mesma razão do chip ativo (ALE-28). */
function ConditionRow(props: {
  id: ConditionId
  label: string
  disabled: boolean
  onPick: () => void
}) {
  return (
    <button
      type="button"
      disabled={props.disabled}
      onClick={() => props.onPick()}
      class="flex w-full flex-col gap-0.5 rounded-sm p-2 text-left transition-colors hover:bg-accent disabled:opacity-50"
    >
      <span class="text-sm font-medium text-foreground">{props.label}</span>
      <span class="text-2xs text-muted-foreground">{conditionEffectSummary(props.id)}</span>
    </button>
  )
}
