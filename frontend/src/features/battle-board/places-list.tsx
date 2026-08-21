import { Hammer, Library, MonitorPlay, RotateCcw, Trash2 } from 'lucide-solid'
import { type JSX, For, Show, createSignal } from 'solid-js'
import type { BoardPlace } from '@/shared/realtime/realtime'
import { Button } from '@/shared/ui/button'
import { ConfirmDialog } from '@/shared/ui/confirm-dialog'
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from '@/shared/ui/dialog'
import { SectionLabel } from '@/shared/ui/section-label'

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
 * Com uma cena NA MESA (`onTable`), reabrir vira TROCAR: o botão diz "Mostrar à
 * mesa" e pergunta antes, porque a troca acontece na tela de todo mundo no
 * mesmo instante — e porque ela move a cena atual para o acervo (ALE-191).
 *
 * @example <PlacesList places={lugares()} onReopen={rt.reopenPlace} onRemove={apagar} />
 */
export function PlacesList(props: {
  places: readonly BoardPlace[]
  onReopen: (placeId: number) => void
  onRemove: (placeId: number) => void
  /** Montar a cena sem pôr nada na mesa (ALE-191, fatia 2). */
  onEdit?: (placeId: number) => void
  /** O nome da cena que está na mesa agora, quando há uma. */
  onTable?: string
}) {
  return (
    <Show when={props.places.length > 0}>
      <section class="w-full max-w-sm text-left">
        <SectionLabel as="h3" tom="gold" class="text-xs mb-1">
          Lugares da crônica
        </SectionLabel>
        <PlaceRows
          places={props.places}
          onReopen={props.onReopen}
          onRemove={props.onRemove}
          onEdit={props.onEdit}
          onTable={props.onTable}
        />
      </section>
    </Show>
  )
}

/** As linhas do acervo, sem título: dentro do diálogo quem nomeia é o cabeçalho
 *  dele, e repetir "Lugares da crônica" duas vezes na mesma caixa é ruído. */
function PlaceRows(props: {
  places: readonly BoardPlace[]
  onReopen: (placeId: number) => void
  onRemove: (placeId: number) => void
  /** Montar a cena sem pôr nada na mesa (ALE-191, fatia 2). */
  onEdit?: (placeId: number) => void
  onTable?: string
}) {
  return (
    <ul class="divide-y divide-grimorio-iron rounded-none border border-grimorio-iron">
      <For each={props.places}>
        {(place) => (
          <li class="flex items-center gap-2 px-2 py-1.5">
            <div class="min-w-0 flex-1">
              <p class="truncate text-sm text-foreground">{place.name}</p>
              <p class="text-2xs text-muted-foreground">
                {place.tokens === 1 ? '1 peça' : `${place.tokens} peças`}
              </p>
            </div>
            <PlaceAction place={place} onTable={props.onTable} onReopen={props.onReopen} />
            {/* Montar acontece FORA da mesa: é a preparação da próxima sala
                enquanto o grupo ainda está na taverna (ALE-191). */}
            <Show when={props.onEdit}>
              {(montar) => (
                <Button
                  size="sm"
                  variant="ghost"
                  aria-label={`Montar ${place.name}`}
                  onClick={() => montar()(place.id)}
                >
                  <Hammer aria-hidden="true" class="size-3.5" />
                </Button>
              )}
            </Show>
            {/* Apagar é o ÚNICO caminho que destrói uma cena montada — e por
                isso é o único que pergunta. */}
            <ConfirmDialog
              title={`Apagar ${place.name}?`}
              description="A cena guardada e as posições dela se perdem. Não dá para desfazer."
              confirmLabel="Apagar"
              destructive
              onConfirm={() => props.onRemove(place.id)}
              trigger={(open) => (
                <Button size="sm" variant="ghost" aria-label={`Apagar ${place.name}`} onClick={open}>
                  <Trash2 aria-hidden="true" class="size-3.5" />
                </Button>
              )}
            />
          </li>
        )}
      </For>
    </ul>
  )
}

/**
 * O que se faz com um lugar guardado, e são três situações diferentes.
 *
 * Sem cena na mesa é REABRIR, e não pergunta nada: não há o que perder. Com
 * outra cena na mesa é MOSTRAR À MESA, e pergunta — o clique troca o que a mesa
 * inteira está olhando, e leva a cena atual para o acervo. E a cena que já está
 * na mesa não oferece botão nenhum: mandar para a mesa o que já está nela seria
 * um caminho que só pode confundir.
 */
function PlaceAction(props: {
  place: BoardPlace
  onTable?: string
  onReopen: (placeId: number) => void
}) {
  return (
    <Show
      when={props.place.name !== props.onTable}
      fallback={<span class="text-2xs text-grimorio-gold">Na mesa</span>}
    >
      <Show
        when={props.onTable}
        fallback={
          <Button size="sm" variant="secondary" onClick={() => props.onReopen(props.place.id)}>
            <RotateCcw aria-hidden="true" class="mr-1 size-3.5" />
            Reabrir
          </Button>
        }
      >
        {(naMesa) => (
          <ConfirmDialog
            title={`Mostrar ${props.place.name} à mesa?`}
            description={`A cena que está na mesa, ${naMesa()}, vai para os Lugares da crônica com as peças onde estão — você a reabre quando quiser.`}
            confirmLabel="Mostrar à mesa"
            onConfirm={() => props.onReopen(props.place.id)}
            trigger={(open) => (
              <Button size="sm" variant="secondary" onClick={open}>
                <MonitorPlay aria-hidden="true" class="mr-1 size-3.5" />
                Mostrar à mesa
              </Button>
            )}
          />
        )}
      </Show>
    </Show>
  )
}

/**
 * O acervo alcançável COM uma cena na mesa (ALE-191).
 *
 * Até aqui os Lugares só existiam na tela vazia, então trocar de cena obrigava a
 * ENCERRAR o tabuleiro primeiro — a mesa via a grade sumir e voltar. É um
 * diálogo e não uma lista fixa no cabeçalho porque o acervo é do preparo: ele é
 * consultado uma vez por cena e não merece largura permanente numa cena que já
 * disputa espaço com a iniciativa.
 *
 * @example <PlacesDialog places={lugares()} onOpenList={refresh} … />
 */
export function PlacesDialog(props: {
  places: readonly BoardPlace[]
  onReopen: (placeId: number) => void
  onRemove: (placeId: number) => void
  onEdit: (placeId: number) => void
  onTable: string
  /** Chamado ao ABRIR: o acervo chega por pergunta, e ele pode ter mudado desde
   *  a última vez (a cena que acabou de sair da mesa está nele). */
  onOpenList: () => void
  trigger: (open: () => void) => JSX.Element
}) {
  const [open, setOpen] = createSignal(false)
  const abrir = () => {
    props.onOpenList()
    setOpen(true)
  }

  return (
    <>
      {props.trigger(abrir)}
      <Dialog open={open()} onOpenChange={setOpen}>
        <DialogContent class="max-w-sm">
          <DialogHeader>
            <DialogTitle>
              <Library aria-hidden="true" class="mr-1 inline size-4" />
              Lugares da crônica
            </DialogTitle>
            <DialogDescription>
              As cenas guardadas desta crônica. Mostrar uma à mesa guarda a que está lá.
            </DialogDescription>
          </DialogHeader>
          <Show
            when={props.places.length > 0}
            fallback={
              <p class="text-sm text-muted-foreground">
                Nenhuma cena guardada ainda. Encerrar um tabuleiro guarda a cena aqui.
              </p>
            }
          >
            <PlaceRows
              places={props.places}
              onReopen={(placeId) => {
                props.onReopen(placeId)
                setOpen(false)
              }}
              onRemove={props.onRemove}
              onEdit={(placeId) => {
                props.onEdit(placeId)
                setOpen(false)
              }}
              onTable={props.onTable}
            />
          </Show>
        </DialogContent>
      </Dialog>
    </>
  )
}
