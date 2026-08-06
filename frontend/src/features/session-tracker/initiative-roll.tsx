import { useQuery } from '@tanstack/react-query'
import { Dices } from 'lucide-react'
import { toast } from 'sonner'
import { Button } from '@/shared/ui/button'
import { Skeleton } from '@/shared/ui/skeleton'
import { rollD20 } from '@/shared/lib/dice'
import {
  expertiseFromSheet,
  useComputedSheet,
} from '@/entities/character/computed-sheet'
import { characterQueryOptions } from '@/entities/character/queries'
import type { Character } from '@/shared/api/api'
import type { useSessionSocket } from '@/shared/realtime/realtime'

// Iniciativa is a DEX perícia; the initiative roll is d20 + its total
// (½ nível + atributo + treino + outros) — read from the computed sheet, same
// math the sheet shows, then sent to the session.

/** Loads the player's own character so its Iniciativa bonus can be rolled. */
export function InitiativeRollButton({
  characterId,
  rt,
}: {
  characterId: number
  rt: ReturnType<typeof useSessionSocket>
}) {
  const character = useQuery(characterQueryOptions(characterId))
  if (character.isLoading) return <Skeleton className="h-8 w-44" />
  if (!character.data) return null
  return <RollButton character={character.data} rt={rt} />
}

function RollButton({
  character,
  rt,
}: {
  character: Character
  rt: ReturnType<typeof useSessionSocket>
}) {
  const sheet = useComputedSheet(character)
  const bonus = expertiseFromSheet(sheet, 'Iniciativa')?.total ?? 0

  const roll = () => {
    const d20 = rollD20()
    const total = d20 + bonus
    rt.rollSelfInitiative(character.id, total)
    toast(`Iniciativa ${total}`, {
      description: `d20 ${d20} ${bonus >= 0 ? '+' : ''}${bonus} (Iniciativa)`,
    })
  }

  return (
    <Button
      size="sm"
      onClick={roll}
      disabled={!rt.isConnected}
      className="gap-1.5"
    >
      <Dices className="size-4" /> Rolar iniciativa ({bonus >= 0 ? '+' : ''}
      {bonus})
    </Button>
  )
}
