import { Show, createMemo } from 'solid-js'
import { allConditionals } from '@/entities/character/derived'
import type { Character } from '@/shared/api/api'
import { useConditionals } from '@/shared/stores/conditionals-context'
import { CountBadge } from '@/shared/ui/count-badge'
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
      <CountBadge count={total()} label={total() === 1 ? 'efeito ativo' : 'efeitos ativos'} tom={total() > 0 ? 'primary' : 'muted'} />
    </Show>
  )
}
