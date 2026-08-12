import type { GeneralPower } from '@tormenta20/t20-data'
import { Show, createMemo, createSignal } from 'solid-js'
import { matchesQuery } from '@/shared/lib/fuzzy-filter'
import { Button } from '@/shared/ui/button'
import { Input } from '@/shared/ui/input'
import { VirtualList } from '@/shared/ui/virtual-list'
import { GeneralPowerRow } from './power-rows'

/**
 * The general-power browse pool: a search box and an "owned only" toggle over a
 * virtualized list. A high-level character can have hundreds of eligible
 * powers, so the list is virtualized and its host card keeps it collapsed.
 */
export function GeneralPowersPool(props: {
  powers: GeneralPower[]
  isOwned: (id: string) => boolean
  isLocked: (power: GeneralPower) => boolean
  disabled: boolean
  onToggle: (id: string) => void
}) {
  const [query, setQuery] = createSignal('')
  const [ownedOnly, setOwnedOnly] = createSignal(false)

  const filtered = createMemo(() =>
    props.powers.filter((power) => {
      if (ownedOnly() && !props.isOwned(power.id)) return false
      return matchesQuery([power.name], query())
    }),
  )

  return (
    <div class="space-y-2">
      <div class="flex items-center gap-2">
        <Input
          value={query()}
          onInput={(event) => setQuery(event.currentTarget.value)}
          placeholder="Buscar poder…"
          aria-label="Buscar poder geral"
          class="h-8 text-xs"
        />
        <Button
          type="button"
          variant={ownedOnly() ? 'default' : 'outline'}
          size="sm"
          class="h-8 shrink-0 text-xs"
          aria-pressed={ownedOnly()}
          onClick={() => setOwnedOnly(!ownedOnly())}
        >
          Meus
        </Button>
      </div>
      <Show
        when={filtered().length > 0}
        fallback={<p class="text-xs italic text-muted-foreground">Nenhum poder.</p>}
      >
        <VirtualList
          class="max-h-80"
          items={filtered()}
          estimateSize={64}
          getKey={(power) => power.id}
          renderItem={(power) => (
            <GeneralPowerRow
              power={power}
              owned={props.isOwned(power.id)}
              locked={props.isLocked(power)}
              disabled={props.disabled}
              onToggle={() => props.onToggle(power.id)}
            />
          )}
        />
      </Show>
    </div>
  )
}
