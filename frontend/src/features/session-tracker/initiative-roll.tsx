import { useQuery } from '@tanstack/react-query'
import { Dices } from 'lucide-react'
import { toast } from 'sonner'
import { Button } from '@/shared/ui/button'
import { rollD20 } from '@/shared/lib/dice'
import {
  expertiseTotalWithItems,
  useCharacterEffects,
} from '@/entities/character/derived'
import {
  EXPERTISES,
  expertiseStateFor,
} from '@/entities/character/expertise'
import { characterQueryOptions } from '@/entities/character/queries'
import type { Character } from '@/shared/api/api'
import type { useSessionSocket } from '@/shared/realtime/realtime'

// Iniciativa is a DEX perícia; the initiative roll is d20 + its total
// (½ nível + atributo + treino + outros) — computed client-side, same math
// the sheet shows, then sent to the session.
const INICIATIVA = EXPERTISES.find((e) => e.name === 'Iniciativa')!

/** Loads the player's own character so its Iniciativa bonus can be rolled. */
export function InitiativeRollButton({
  characterId,
  rt,
}: {
  characterId: number
  rt: ReturnType<typeof useSessionSocket>
}) {
  const character = useQuery(characterQueryOptions(characterId))
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
  const effects = useCharacterEffects(character)
  const bonus = expertiseTotalWithItems(
    character,
    expertiseStateFor(character, INICIATIVA),
    effects,
  ).total

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
