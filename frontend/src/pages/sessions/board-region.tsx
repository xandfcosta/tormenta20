import { Crosshair, LayoutGrid, Minus, Plus, Users, X } from 'lucide-solid'
import { Show, createSignal } from 'solid-js'
import { BoardView } from '@/features/battle-board/board-view'
import { type BoardViewport, SQUARE_METRES } from '@/features/battle-board/board-viewport'
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
 * aba, ir ao bestiário apagaria a seleção e o enquadramento — o mesmo defeito que
 * forçou o autosave das notas. Pela mesma razão a JANELA (origem e zoom) é dona
 * da página e chega por prop: ela precisa sobreviver à troca de região.
 */
export function BoardRegion(props: {
  rt: SessionRealtime
  isGm: boolean
  view: BoardViewport
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
    // `w-full flex-1`: no rail do jogador o cartão é filho de um flex, e sem
    // isso ele encolhe para o conteúdo — medido em 138px de 352 disponíveis,
    // ou seja, o tabuleiro virava uma tira estreita (ALE-124).
    <section class="@container flex w-full min-h-0 min-w-0 flex-1 flex-col rounded-sm border border-grimorio-iron bg-[var(--grimorio-panel)]">
      <Show when={board()} fallback={<EmptyBoard isGm={props.isGm} onOpen={props.rt.openBoard} />}>
        {(live) => (
          <>
            <header class="flex shrink-0 flex-wrap items-center gap-2 border-b border-grimorio-iron px-3 py-2">
              <h2 class="min-w-0 truncate font-heading text-sm uppercase tracking-wide text-grimorio-gold">
                {live().place}
              </h2>
              <span class="font-mono text-[11px] tabular-nums text-muted-foreground">
                {live().tokens.length} peças · 1 quadrado = {SQUARE_METRES.toFixed(1).replace('.', ',')}m
              </span>

              <div class="ml-auto flex items-center gap-1">
                <ViewControls view={props.view} onFit={() => props.view.fit(live().tokens)} />
                <Show when={props.isGm}>
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
                      <Button
                        size="sm"
                        variant="ghost"
                        aria-label="Encerrar o tabuleiro"
                        onClick={open}
                      >
                        <X aria-hidden="true" class="size-4" />
                      </Button>
                    )}
                  />
                </Show>
              </div>
            </header>

            <BoardView
              board={live()}
              view={props.view}
              activeEntryId={props.activeEntryId}
              selectedTokenId={selectedTokenId()}
              onSelectToken={props.isGm ? selectToken : undefined}
              onPlaceToken={props.isGm ? placeSelected : undefined}
            />

            {/* A dica só existe no momento em que ela é ACIONÁVEL: como linha
                permanente, ela custava 26px de altura em todo formato para dizer
                o óbvio, e no celular deitado isso é uma fileira inteira de
                quadrados a menos (ALE-124). */}
            <Show when={props.isGm && selectedTokenId()}>
              <p class="shrink-0 border-t border-grimorio-iron px-3 py-1 text-[11px] text-grimorio-gold">
                Clique num quadrado para pousar a peça.
              </p>
            </Show>
          </>
        )}
      </Show>
    </section>
  )
}

/**
 * Mover a vista e o zoom. Num plano INFINITO isto não é enfeite: sem uma forma
 * de andar com a janela, metade da cena fica inalcançável.
 *
 * "Centralizar" enquadra as PEÇAS, e não a origem: o centro de um plano infinito
 * não significa nada — o que o mestre quer é achar o grupo.
 */
function ViewControls(props: { view: BoardViewport; onFit: () => void }) {
  const step = () => Math.max(1, Math.floor(props.view.cols() / 3))

  return (
    <div class="flex items-center gap-0.5">
      <PanButton label="Mover a vista para a esquerda" onClick={() => props.view.pan(-step(), 0)}>
        ←
      </PanButton>
      <PanButton label="Mover a vista para cima" onClick={() => props.view.pan(0, -step())}>
        ↑
      </PanButton>
      <PanButton label="Mover a vista para baixo" onClick={() => props.view.pan(0, step())}>
        ↓
      </PanButton>
      <PanButton label="Mover a vista para a direita" onClick={() => props.view.pan(step(), 0)}>
        →
      </PanButton>
      <Button size="sm" variant="ghost" aria-label="Afastar" onClick={() => props.view.zoom(-8)}>
        <Minus aria-hidden="true" class="size-4" />
      </Button>
      <Button size="sm" variant="ghost" aria-label="Aproximar" onClick={() => props.view.zoom(8)}>
        <Plus aria-hidden="true" class="size-4" />
      </Button>
      <Button size="sm" variant="ghost" aria-label="Centralizar nas peças" onClick={props.onFit}>
        <Crosshair aria-hidden="true" class="size-4" />
      </Button>
    </div>
  )
}

function PanButton(props: { label: string; onClick: () => void; children: string }) {
  return (
    <Button
      size="sm"
      variant="ghost"
      class="h-8 w-8 font-mono"
      aria-label={props.label}
      onClick={props.onClick}
    >
      <span aria-hidden="true">{props.children}</span>
    </Button>
  )
}

/**
 * Sessão sem tabuleiro. O estado vazio é do MESTRE: o jogador não abre cena, e
 * dizer a ele "abra um tabuleiro" seria oferecer um botão que não existe.
 */
function EmptyBoard(props: { isGm: boolean; onOpen: (place: string, terrain: string) => void }) {
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
