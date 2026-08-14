import { useQuery } from '@tanstack/solid-query'
import { Dices } from 'lucide-solid'
import { Show } from 'solid-js'
import { computedSheetFor, expertiseFromSheet } from '@/entities/character/computed-sheet'
import { characterQueryOptions } from '@/entities/character/queries'
import type { Character } from '@/shared/api/api'
import type { SessionRealtime } from '@/shared/realtime/realtime'
import { useConditionals } from '@/shared/stores/conditionals-context'
import { Button } from '@/shared/ui/button'
import { rollD20 } from '@/shared/lib/dice'
import { Skeleton } from '@/shared/ui/skeleton'
import { toast } from '@/shared/ui/sonner'
import { settledQuery } from '@/shared/lib/settled-query'

/**
 * The player rolls their OWN initiative: d20 + the Iniciativa perícia total
 * (½ nível + atributo + treino + outros), read from the computed sheet so the
 * table sees the same number the sheet shows. Upserted by characterId
 * server-side, so re-rolling replaces instead of duplicating.
 */
export function InitiativeRollButton(props: { characterId: number; rt: SessionRealtime }) {
  const character = useQuery(() => characterQueryOptions(props.characterId))

  return (
    <Show when={settledQuery(character)} fallback={<Skeleton class="h-8 w-44" />}>
      {(data) => <RollButton character={data()} rt={props.rt} />}
    </Show>
  )
}

function RollButton(props: { character: Character; rt: SessionRealtime }) {
  const conditionals = useConditionals()
  const bonus = () => {
    const sheet = computedSheetFor(
      props.character,
      conditionals.active(props.character.id),
    )
    return expertiseFromSheet(sheet, 'Iniciativa')?.total ?? 0
  }

  const roll = () => {
    const d20 = rollD20()
    const total = d20 + bonus()
    props.rt.rollSelfInitiative(props.character.id, total)
    toast.success(`Iniciativa ${total}`, {
      description: `d20 ${d20} ${bonus() >= 0 ? '+' : ''}${bonus()} (Iniciativa)`,
    })
  }

  return (
    <Button
      type="button"
      size="sm"
      variant="secondary"
      class="gap-1.5"
      disabled={!props.rt.isConnected()}
      onClick={roll}
    >
      <Dices aria-hidden="true" class="size-4" />
      Rolar iniciativa ({bonus() >= 0 ? '+' : ''}
      {bonus()})
    </Button>
  )
}
