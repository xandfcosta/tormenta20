import { Show, createMemo } from 'solid-js'
import { allConditionals } from '@/entities/character/derived'
import type { Character } from '@/shared/api/api'
import { useConditionals } from '@/shared/stores/conditionals-context'
import { cn } from '@/shared/lib/utils'
import { effectsShownCount } from './effects-count'

/**
 * Numeric pill next to the "Efeitos" tab. Reads the conditionals store here so
 * `effectsShownCount` stays a pure function — the counting rule is tested
 * without rendering anything.
 */
export function EffectsCountBadge(props: { character: Character }) {
  const conditionals = useConditionals()
  const entries = createMemo(() =>
    allConditionals(props.character, conditionals.active(props.character.id)),
  )
  const total = createMemo(() => effectsShownCount(props.character, entries()))

  return (
    <Show when={total() > 0 || entries().length > 0}>
      {/* A bare `aria-label` on a <span> is ignored — the element has no role to
          carry it, so the pill announced only "3". The number is hidden from
          assistive tech and the meaning spelled out beside it instead. */}
      <span
        class={cn(
          'ml-0.5 inline-flex h-4 min-w-4 items-center justify-center rounded-full px-1 text-[10px] font-bold',
          total() > 0 ? 'bg-primary text-primary-foreground' : 'bg-muted text-muted-foreground',
        )}
        aria-hidden="true"
      >
        {total()}
      </span>
      <span class="sr-only">{total()} efeitos ativos</span>
    </Show>
  )
}
