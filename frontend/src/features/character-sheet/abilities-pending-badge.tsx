import { Show, createMemo } from 'solid-js'
import type { Character } from '@/shared/api/api'
import { CountBadge } from '@/shared/ui/count-badge'
import { computePendencias } from './pendencias'

/**
 * Numeric pill on the "Poderes" tab trigger — counts unmade required choices
 * (race variants, origin benefits, class powers/paths). Renders nothing when
 * there's nothing outstanding, like the Efeitos badge.
 */
export function AbilitiesPendingBadge(props: { character: Character }) {
  const count = createMemo(() => computePendencias(props.character).length)

  return (
    <Show when={count() > 0}>
      <CountBadge count={count()} label={count() === 1 ? 'escolha pendente' : 'escolhas pendentes'} />
    </Show>
  )
}
