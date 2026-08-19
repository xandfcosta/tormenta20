import { For, Show, createMemo } from 'solid-js'
import { upcomingTurns } from '@/features/session-tracker/tracker-rules'
import type { InitiativeEntry, SessionRealtime } from '@/shared/realtime/realtime'
import { cn } from '@/shared/lib/utils'

/**
 * Mora em `pages/` e não em `features/battle-board/`: ela lê a REGRA da fila do
 * `session-tracker`, e feature não importa feature — uma tela que precisa de
 * duas delas lado a lado é composição (guia do front).
 */

/**
 * Quem está na vez e quem vem depois, sem sair do mapa (ALE-179).
 *
 * Só para o JOGADOR: o mestre tem a iniciativa inteira numa coluna ao lado, e
 * repetir três nomes ali seria ruído. Na tela do jogador o tabuleiro ocupa a
 * superfície toda, e para saber se ele é o próximo ele precisava trocar de aba
 * — no meio do turno de outra pessoa, que é quando se decide o que fazer.
 *
 * Fora de combate a tira some: não há vez de ninguém para anunciar.
 */
export function TurnStrip(props: { rt: SessionRealtime; hidden: boolean }) {
  const fila = createMemo(() =>
    props.hidden ? [] : upcomingTurns(props.rt.state().initiative, props.rt.state().turnIndex, 3),
  )

  return (
    <Show when={fila().length > 0}>
      <div class="flex shrink-0 items-center gap-2 overflow-hidden border-b border-grimorio-iron px-3 py-1.5 text-[11px]">
        <span class="shrink-0 font-heading uppercase tracking-wide text-grimorio-gold">
          Na vez
        </span>
        <For each={fila()}>
          {(entrada: InitiativeEntry, posicao: () => number) => (
            <>
              <Show when={posicao() > 0}>
                <span aria-hidden="true" class="shrink-0 text-muted-foreground">
                  ›
                </span>
              </Show>
              <span
                class={cn(
                  'min-w-0 truncate',
                  posicao() === 0 ? 'font-semibold text-foreground' : 'text-muted-foreground',
                )}
              >
                {entrada.label}
              </span>
            </>
          )}
        </For>
      </div>
    </Show>
  )
}
