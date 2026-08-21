import { ChevronDown } from 'lucide-solid'
import { type JSX, Show, createEffect, createSignal } from 'solid-js'
import { cn } from '@/shared/lib/utils'

/**
 * A focus request from the Pendências callout: which card to open, plus a
 * nonce so clicking the same pendência twice re-triggers the open + scroll.
 */
export type CardFocus = { id: string; nonce: number } | null

/**
 * Collapsible section shell for the Habilidades sub-tabs: title, optional
 * count, and a "pendente" badge while the section still owes choices. When
 * `focus` targets this card's id it opens and scrolls into view.
 *
 * @example
 * <CollapsibleAbilityCard id="raca:humano" title="Humano" focus={focus()}>
 *   …
 * </CollapsibleAbilityCard>
 */
export function CollapsibleAbilityCard(props: {
  id: string
  title: string
  count?: JSX.Element
  pending?: number
  defaultOpen?: boolean
  focus: CardFocus
  children: JSX.Element
}) {
  const [open, setOpen] = createSignal(props.defaultOpen ?? true)
  let card: HTMLDivElement | undefined

  // `focus` is a fresh object per jump (it carries a nonce), so this re-fires
  // even when the player re-targets the same card after collapsing it.
  createEffect(() => {
    if (props.focus?.id !== props.id) return
    props.focus.nonce
    setOpen(true)
    card?.scrollIntoView({ behavior: 'smooth', block: 'start' })
  })

  const pending = () => props.pending ?? 0

  return (
    <div ref={card} class="overflow-hidden rounded-none border border-grimorio-iron bg-card">
      <button
        type="button"
        onClick={() => setOpen(!open())}
        aria-expanded={open()}
        class="flex w-full items-center gap-2 px-3 py-2 text-left hover:bg-accent"
      >
        <ChevronDown
          aria-hidden="true"
          class={cn(
            'size-4 shrink-0 text-muted-foreground transition-transform',
            !open() && '-rotate-90',
          )}
        />
        <span class="text-sm font-semibold">{props.title}</span>
        <Show when={props.count != null}>
          <span class="text-xs text-muted-foreground">{props.count}</span>
        </Show>
        <Show when={pending() > 0}>
          <span class="ml-auto rounded-full bg-destructive px-2 py-0.5 text-3xs font-bold text-white">
            {pending()} pendente{pending() > 1 ? 's' : ''}
          </span>
        </Show>
      </button>
      <Show when={open()}>
        <div class="border-t border-grimorio-iron px-3 py-2">{props.children}</div>
      </Show>
    </div>
  )
}
