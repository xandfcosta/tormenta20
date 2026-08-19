import { RotateCcw, Trash2 } from 'lucide-solid'
import { For, Show } from 'solid-js'
import type { BoardPlace } from '@/shared/realtime/realtime'
import { Button } from '@/shared/ui/button'
import { ConfirmDialog } from '@/shared/ui/confirm-dialog'

/**
 * Os Lugares da crônica: as cenas que o mestre já montou (ALE-124, fatia 5).
 *
 * Encerrar o tabuleiro ARQUIVA — a taverna com as nove peças onde ficaram volta
 * para cá e é reaberta na semana seguinte sem remontar nada. Antes desta fatia,
 * encerrar destruía a cena, e era a única promessa da épica que o código
 * contradizia.
 *
 * A lista mostra a contagem de peças e não a cena: é por ela que o mestre
 * reconhece "a taverna, aquela com os nove", e baixar o acervo inteiro para
 * desenhar um menu seria pagar caro por um número.
 *
 * @example <PlacesList places={lugares()} onReopen={rt.reopenPlace} onRemove={apagar} />
 */
export function PlacesList(props: {
  places: readonly BoardPlace[]
  onReopen: (placeId: number) => void
  onRemove: (placeId: number) => void
}) {
  return (
    <Show when={props.places.length > 0}>
      <section class="w-full max-w-sm text-left">
        <h3 class="mb-1 font-heading text-xs uppercase tracking-widest text-grimorio-gold">
          Lugares da crônica
        </h3>
        <ul class="divide-y divide-grimorio-iron rounded-sm border border-grimorio-iron">
          <For each={props.places}>
            {(place) => (
              <li class="flex items-center gap-2 px-2 py-1.5">
                <div class="min-w-0 flex-1">
                  <p class="truncate text-sm text-foreground">{place.name}</p>
                  <p class="text-[11px] text-muted-foreground">
                    {place.tokens === 1 ? '1 peça' : `${place.tokens} peças`}
                  </p>
                </div>
                <Button size="sm" variant="secondary" onClick={() => props.onReopen(place.id)}>
                  <RotateCcw aria-hidden="true" class="mr-1 size-3.5" />
                  Reabrir
                </Button>
                {/* Apagar é o ÚNICO caminho que destrói uma cena montada — e por
                    isso é o único que pergunta. */}
                <ConfirmDialog
                  title={`Apagar ${place.name}?`}
                  description="A cena guardada e as posições dela se perdem. Não dá para desfazer."
                  confirmLabel="Apagar"
                  destructive
                  onConfirm={() => props.onRemove(place.id)}
                  trigger={(open) => (
                    <Button
                      size="sm"
                      variant="ghost"
                      aria-label={`Apagar ${place.name}`}
                      onClick={open}
                    >
                      <Trash2 aria-hidden="true" class="size-3.5" />
                    </Button>
                  )}
                />
              </li>
            )}
          </For>
        </ul>
      </section>
    </Show>
  )
}
