import { useQueryClient } from '@tanstack/solid-query'
import type { CatalogSpell } from '@/shared/api/catalog-types'
import type { Modifier } from '@/shared/api/item-types'
import { Plus } from 'lucide-solid'
import { Show, createMemo, createSignal } from 'solid-js'
import { invalidateCharacterDependents } from '@/entities/character/character-cache'
import { characterQueryOptions } from '@/entities/character/queries'
import { applyPoolResult } from '@/entities/character/temp-hp-pool'
import { buffSpells } from '@/shared/lib/spell-cache'
import { type Character, api } from '@/shared/api/api'
import { Button } from '@/shared/ui/button'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/shared/ui/dialog'
import { DialogInlineError } from '@/shared/ui/dialog-inline-error'
import { Input } from '@/shared/ui/input'
import { VirtualList } from '@/shared/ui/virtual-list'
import { FactChips } from './fact-chips'
import { ITEM_DIALOG_CONTENT, ITEM_DIALOG_TITLE } from './item-dialog-kit'
import { ModifierList } from './modifier-list'
import { normalize } from './normalize'

/**
 * Manual self-apply of a spell buff as a scene/day ActiveEffect. A buff is
 * never auto-applied from another caster — the target (or the GM) picks the
 * source and applies it here. Scope comes from the buff's `defaultScope`.
 */
export function ApplyEffectDialog(props: { character: Character }) {
  const queryClient = useQueryClient()
  const [open, setOpen] = createSignal(false)
  const [query, setQuery] = createSignal('')
  const [pending, setPending] = createSignal(false)
  const [formError, setFormError] = createSignal<string | null>(null)

  // From the primed spell cache, not a module const: the cache is filled by the
  // loader gate at runtime, so a const read at import time would be empty.
  const matches = createMemo(() => {
    const search = normalize(query().trim())
    if (!search) return buffSpells()
    return buffSpells().filter((spell) => normalize(spell.name).includes(search))
  })

  const apply = async (spellId: string) => {
    setPending(true)
    setFormError(null)
    try {
      const result = await api.characters.applyEffect(props.character.id, { spellId })
      const queryKey = characterQueryOptions(props.character.id).queryKey
      // The endpoint answers a plain row for spell buffs, but it is shared with
      // the temp-PV pool paths — applyPoolResult covers every outcome it makes.
      queryClient.setQueryData<Character>(queryKey, (prev) =>
        prev ? applyPoolResult(prev, result) : prev,
      )
      invalidateCharacterDependents(queryClient, props.character.id)
      setOpen(false)
    } catch {
      // Inline, not a toast: the sonner region is a sibling of the open modal
      // and Kobalte marks it aria-hidden, so a toast here is never announced.
      setFormError('Não foi possível aplicar o efeito.')
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
        class="h-6 gap-1 px-2 text-[11px]"
        onClick={() => {
          setQuery('')
          setFormError(null)
          setOpen(true)
        }}
      >
        <Plus aria-hidden="true" class="size-3" />
        Aplicar efeito
      </Button>

      <Dialog open={open()} onOpenChange={setOpen}>
        <DialogContent class={ITEM_DIALOG_CONTENT}>
          <DialogHeader>
            <DialogTitle class={ITEM_DIALOG_TITLE}>Aplicar efeito de magia</DialogTitle>
          </DialogHeader>
          <div class="space-y-3">
            <Input
              value={query()}
              onInput={(event) => setQuery(event.currentTarget.value)}
              placeholder="Buscar magia…"
              aria-label="Buscar magia"
            />
            <DialogInlineError message={formError()} />
            <Show
              when={matches().length > 0}
              fallback={
                <p class="rounded-sm border border-border bg-muted px-3 py-6 text-center text-sm text-muted-foreground">
                  Nenhuma magia com efeito aplicável.
                </p>
              }
            >
              <VirtualList
                class="max-h-72 rounded-sm border border-border bg-muted"
                items={matches()}
                estimateSize={60}
                getKey={(spell) => spell.id}
                renderItem={(spell) => (
                  <BuffRow spell={spell} disabled={pending()} onPick={() => void apply(spell.id)} />
                )}
              />
            </Show>
          </div>
        </DialogContent>
      </Dialog>
    </>
  )
}

function BuffRow(props: { spell: CatalogSpell; disabled: boolean; onPick: () => void }) {
  const modifiers = (): readonly Modifier[] => props.spell.buff?.modifiers ?? []
  return (
    <button
      type="button"
      disabled={props.disabled}
      onClick={() => props.onPick()}
      class="flex w-full flex-col gap-1 rounded-sm p-2 text-left transition-colors hover:bg-accent disabled:opacity-50"
    >
      <div class="flex items-center justify-between gap-2">
        <span class="truncate text-sm font-medium text-foreground">{props.spell.name}</span>
        <span class="shrink-0 text-[10px] uppercase text-muted-foreground">
          {props.spell.buff?.defaultScope === 'day' ? 'dia' : 'cena'}
        </span>
      </div>
      <ModifierList modifiers={modifiers()} />
      <FactChips facts={props.spell.buff?.facts ?? []} />
    </button>
  )
}
