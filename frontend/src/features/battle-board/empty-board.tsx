import { LayoutGrid, Library } from 'lucide-solid'
import { Show } from 'solid-js'
import type { BoardPlace } from '@/shared/realtime/realtime'
import { Button } from '@/shared/ui/button'
import { OpenBoardDialog } from './open-board-dialog'
import { PlacesDialog } from './places-list'

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
  /** Rebuscar o acervo ao ABRIR o diálogo: ele chega por pergunta e pode ter
   *  mudado desde a última vez. */
  onRefreshPlaces: () => void
}) {
  return (
    // `justify-center` NÃO entra aqui, e isto é conserto: com o acervo de Lugares
    // dentro da cena vazia (ALE-124, fatia 5), o conteúdo passa da altura da
    // caixa assim que a crônica junta algumas cenas — e centrar conteúdo que
    // transborda empurra o TOPO para fora da área rolável, onde o navegador não
    // deixa chegar. O primeiro item do topo é justamente "Abrir tabuleiro": o
    // mestre com dez lugares guardados perdia o botão de abrir o próximo.
    //
    // O idioma que centra SEM clipar é `m-auto` no miolo: sobrando espaço ele
    // centraliza, faltando espaço ele começa do topo e a rolagem alcança tudo.
    // É a mesma família do ✕ inalcançável da ALE-178.
    <div class="flex min-h-0 flex-1 flex-col overflow-y-auto p-6 text-center">
      {/* `w-full max-w-md` e não só `m-auto`: num flex-column a margem
          automática DESLIGA o esticamento e a largura passa a ser `fit-content`,
          que é o `max-content` do filho mais largo — foi por aí que o acervo
          transbordou a coluna de 374px e levou a lixeira para fora da tela
          (ALE-198). Com a largura declarada, o miolo nunca é mais largo que a
          região, e a medida de leitura continua sendo respeitada. */}
      <div class="m-auto flex w-full max-w-md flex-col items-center gap-3">
        <LayoutGrid aria-hidden="true" class="size-8 text-muted-foreground" />
        <p class="text-sm text-muted-foreground">
          {props.isGm
            ? 'Nenhum tabuleiro aberto. Vale para combate e para cena de interpretação.'
            : 'O mestre ainda não abriu um tabuleiro.'}
        </p>
        <Show when={props.isGm}>
          <div class="flex flex-wrap items-center justify-center gap-2">
            <OpenBoardDialog
              onOpen={props.onOpen}
              trigger={(open) => <Button onClick={open}>Abrir tabuleiro</Button>}
            />
            {/* O acervo vem DEPOIS do botão de abrir: montar uma cena nova é o
                que se faz na primeira noite, e reabrir é o que se faz nas
                outras. Ele é um BOTÃO e não uma lista porque esta região é a
                mais estreita da cena — e porque o mesmo acervo já tinha um
                diálogo, aberto com uma cena na mesa (ALE-191). Dois desenhos
                para o mesmo widget é o defeito que a ALE-169 consertou noutro
                lugar; aqui sobra um só. */}
            <Show when={props.places.length > 0}>
              <PlacesDialog
                places={props.places}
                onReopen={props.onReopen}
                onRemove={props.onRemovePlace}
                onEdit={props.onEdit}
                onOpenList={props.onRefreshPlaces}
                trigger={(open) => (
                  <Button variant="secondary" onClick={open}>
                    <Library aria-hidden="true" class="mr-1.5 size-4" />
                    Lugares da crônica · {props.places.length}
                  </Button>
                )}
              />
            </Show>
          </div>
        </Show>
      </div>
    </div>
  )
}
