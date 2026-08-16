import { LayoutGrid, Users, X } from 'lucide-solid'
import { Show, createSignal } from 'solid-js'
import { BoardView } from '@/features/battle-board/board-view'
import { OpenBoardDialog } from '@/features/battle-board/open-board-dialog'
import type { SessionRealtime } from '@/shared/realtime/realtime'
import { Button } from '@/shared/ui/button'
import { ConfirmDialog } from '@/shared/ui/confirm-dialog'

/**
 * O tabuleiro dentro da cena da sessão (ALE-124).
 *
 * Composição mora em `pages/`: liga o socket, a seleção que a iniciativa já usa
 * e os controles do mestre. A feature `battle-board` só desenha e avisa — ela
 * não conhece `session-tracker`, e é isso que mantém as dependências apontando
 * para baixo.
 *
 * É REGIÃO e não aba do workspace porque `Tabs` desmonta o conteúdo inativo: numa
 * aba, ir ao bestiário apagaria a seleção e (quando existir) o enquadramento do
 * tabuleiro — o mesmo defeito que forçou o autosave das notas.
 */
export function BoardRegion(props: {
  rt: SessionRealtime
  isGm: boolean
  /** Linha da iniciativa na vez: a peça dela ganha o anel dourado. */
  activeEntryId?: string | null
}) {
  const [selectedTokenId, setSelectedTokenId] = createSignal<string | null>(null)
  const board = () => props.rt.board()

  // Selecionar de novo a mesma peça DESSELECIONA: sem isso não há como largar a
  // peça sem posicioná-la, e o próximo clique num quadrado a moveria sem querer.
  const selectToken = (tokenId: string) =>
    setSelectedTokenId((current) => (current === tokenId ? null : tokenId))

  const placeSelected = (x: number, y: number) => {
    const tokenId = selectedTokenId()
    if (!tokenId) return
    props.rt.updateToken(tokenId, { x, y })
    setSelectedTokenId(null)
  }

  return (
    <section class="@container flex min-h-0 min-w-0 flex-col rounded-sm border border-grimorio-iron bg-[var(--grimorio-panel)]">
      <Show
        when={board()}
        fallback={<EmptyBoard isGm={props.isGm} onOpen={props.rt.openBoard} />}
      >
        {(live) => (
          <>
            <header class="flex shrink-0 flex-wrap items-center gap-2 border-b border-grimorio-iron px-3 py-2">
              <h2 class="min-w-0 truncate font-heading text-sm uppercase tracking-wide text-grimorio-gold">
                {live().place}
              </h2>
              <span class="font-mono text-[11px] tabular-nums text-muted-foreground">
                {live().cols}×{live().rows} quadrados · {(live().cols * 1.5).toFixed(1)}m de frente
              </span>
              <Show when={props.isGm}>
                <div class="ml-auto flex items-center gap-2">
                  <Button
                    size="sm"
                    variant="secondary"
                    disabled={!props.rt.isConnected()}
                    onClick={props.rt.populateBoard}
                  >
                    <Users aria-hidden="true" class="size-4" />
                    Trazer a iniciativa
                  </Button>
                  <ConfirmDialog
                    title="Encerrar o tabuleiro?"
                    description="As peças e as posições desta cena se perdem. A iniciativa e os PV continuam como estão."
                    confirmLabel="Encerrar"
                    destructive
                    onConfirm={props.rt.closeBoard}
                    trigger={(open) => (
                      <Button size="sm" variant="ghost" aria-label="Encerrar o tabuleiro" onClick={open}>
                        <X aria-hidden="true" class="size-4" />
                      </Button>
                    )}
                  />
                </div>
              </Show>
            </header>

            <BoardView
              board={live()}
              activeEntryId={props.activeEntryId}
              selectedTokenId={selectedTokenId()}
              onSelectToken={props.isGm ? selectToken : undefined}
              onPlaceToken={props.isGm ? placeSelected : undefined}
            />

            <Show when={props.isGm}>
              <p class="shrink-0 border-t border-grimorio-iron px-3 py-1.5 text-[11px] text-muted-foreground">
                {selectedTokenId()
                  ? 'Clique num quadrado para pousar a peça.'
                  : 'Clique numa peça para movê-la.'}
              </p>
            </Show>
          </>
        )}
      </Show>
    </section>
  )
}

/**
 * Sessão sem tabuleiro. O estado vazio é do MESTRE: o jogador não abre cena, e
 * dizer a ele "abra um tabuleiro" seria oferecer um botão que não existe.
 */
function EmptyBoard(props: {
  isGm: boolean
  onOpen: (place: string, cols: number, rows: number, terrain: string) => void
}) {
  return (
    <div class="flex min-h-0 flex-1 flex-col items-center justify-center gap-3 p-6 text-center">
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
      </Show>
    </div>
  )
}
