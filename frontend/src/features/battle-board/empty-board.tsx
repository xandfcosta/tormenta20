import { LayoutGrid } from 'lucide-solid'
import { Show } from 'solid-js'
import type { BoardPlace } from '@/shared/realtime/realtime'
import { Button } from '@/shared/ui/button'
import { OpenBoardDialog } from './open-board-dialog'
import { PlacesList } from './places-list'

/**
 * Sessão sem tabuleiro. O estado vazio é do MESTRE: o jogador não abre cena, e
 * dizer a ele "abra um tabuleiro" seria oferecer um botão que não existe.
 */
export function EmptyBoard(props: {
  isGm: boolean
  onOpen: (place: string, terrain: string) => void
  places: readonly BoardPlace[]
  onReopen: (placeId: number) => void
  onRemovePlace: (placeId: number) => void
  /** Montar um lugar guardado sem abrir tabuleiro nenhum (ALE-191). */
  onEdit: (placeId: number) => void
}) {
  return (
    <div class="flex min-h-0 flex-1 flex-col items-center justify-center gap-3 overflow-y-auto p-6 text-center">
      <LayoutGrid aria-hidden="true" class="size-8 text-muted-foreground" />
      <p class="text-sm text-muted-foreground">
        {props.isGm
          ? 'Nenhum tabuleiro aberto. Vale para combate e para cena de interpretação.'
          : 'O mestre ainda não abriu um tabuleiro.'}
      </p>
      <Show when={props.isGm}>
        <OpenBoardDialog
          onOpen={props.onOpen}
          trigger={(open) => <Button onClick={open}>Abrir tabuleiro</Button>}
        />
        {/* O acervo vem DEPOIS do botão de abrir: montar uma cena nova é o que
            se faz na primeira noite, e reabrir é o que se faz nas outras. */}
        <PlacesList
          places={props.places}
          onReopen={props.onReopen}
          onRemove={props.onRemovePlace}
          onEdit={props.onEdit}
        />
      </Show>
    </div>
  )
}
