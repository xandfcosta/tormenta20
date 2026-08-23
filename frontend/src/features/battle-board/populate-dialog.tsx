import { For, type JSX, Show, createMemo, createSignal } from 'solid-js'
import type { InitiativeEntry } from '@/shared/realtime/realtime'
import { cn } from '@/shared/lib/utils'
import { Button } from '@/shared/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/shared/ui/dialog'
import { SectionLabel } from '@/shared/ui/section-label'

/** Quem é do lado dos jogadores. É o MESMO predicado que o servidor usa para
 *  escolher o lado do mapa (`entry.Type == "character"`), e não uma segunda
 *  regra: o atalho põe as peças exatamente onde a fileira do grupo nasce. */
export const isPlayerEntry = (entry: InitiativeEntry) => entry.type === 'character'

/**
 * Quem entra na cena (ALE-204).
 *
 * "Trazer a iniciativa" trazia a fila INTEIRA, e a fila inclui o assassino que o
 * mestre montou para aparecer no terceiro turno: num clique a emboscada virava
 * peça na tela da mesa, e desfazer era peça por peça. Aqui ele escolhe.
 *
 * O diálogo nasce com os JOGADORES marcados, e isso não é conveniência — é o
 * que faz o atalho do clique direito não ser um caminho exclusivo (regra da
 * casa: gesto nunca é o único caminho). Abrir e confirmar faz exatamente o que
 * o clique direito faz, sem gesto nenhum.
 *
 * Quem já tem peça aparece marcado e travado: trazer de novo não faria nada (o
 * servidor é idempotente), e esconder a linha faria o mestre procurar um nome
 * que ele acabou de ver na fila.
 *
 * @example <PopulateDialog entries={rt.state().initiative} placedEntryIds={jaNoMapa()}
 *            onPopulate={rt.populateBoard} trigger={(abrir) => <Button onClick={abrir}/>} />
 */
export function PopulateDialog(props: {
  entries: readonly InitiativeEntry[]
  /** As linhas que já têm peça no tabuleiro. */
  placedEntryIds: ReadonlySet<string>
  onPopulate: (entryIds: string[]) => void
  trigger: (open: () => void) => JSX.Element
}) {
  const [open, setOpen] = createSignal(false)
  const [chosen, setChosen] = createSignal<ReadonlySet<string>>(new Set())

  const pending = createMemo(() => props.entries.filter((e) => !props.placedEntryIds.has(e.id)))
  const players = createMemo(() => props.entries.filter(isPlayerEntry))
  const others = createMemo(() => props.entries.filter((entry) => !isPlayerEntry(entry)))

  // Cada abertura recomeça no padrão SEGURO: sem isto o diálogo mostraria o
  // rascunho da vez anterior, e o rascunho da vez anterior pode ter o vilão
  // marcado.
  const start = () => {
    setChosen(new Set(pending().filter(isPlayerEntry).map((entry) => entry.id)))
    setOpen(true)
  }

  const toggle = (entryId: string) => {
    const next = new Set(chosen())
    if (!next.delete(entryId)) next.add(entryId)
    setChosen(next)
  }

  const confirm = () => {
    props.onPopulate([...chosen()])
    setOpen(false)
  }

  return (
    <Dialog open={open()} onOpenChange={(next) => (next ? start() : setOpen(false))}>
      {props.trigger(start)}
      <DialogContent class="max-w-sm">
        <DialogHeader>
          <DialogTitle>Trazer para o tabuleiro</DialogTitle>
          <DialogDescription>
            Os jogadores já vêm marcados — eles sabem onde estão. O resto entra quando você
            quiser.
          </DialogDescription>
        </DialogHeader>

        <div class="flex items-center gap-1">
          <Button
            type="button"
            size="sm"
            variant="ghost"
            onClick={() => setChosen(new Set(pending().map((entry) => entry.id)))}
          >
            Todos
          </Button>
          <Button type="button" size="sm" variant="ghost" onClick={() => setChosen(new Set())}>
            Nenhum
          </Button>
        </div>

        <div class="max-h-[45vh] space-y-3 overflow-y-auto">
          <EntryGroup
            title="Jogadores"
            entries={players()}
            chosen={chosen()}
            placed={props.placedEntryIds}
            onToggle={toggle}
          />
          <EntryGroup
            title="NPCs e monstros"
            entries={others()}
            chosen={chosen()}
            placed={props.placedEntryIds}
            onToggle={toggle}
          />
        </div>

        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => setOpen(false)}>
            Cancelar
          </Button>
          <Button type="button" disabled={chosen().size === 0} onClick={confirm}>
            {`Trazer ${chosen().size} ${chosen().size === 1 ? 'peça' : 'peças'}`}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

/** Um lado da fila. Some inteiro quando não há ninguém: um cabeçalho sobre nada
 *  faria o mestre procurar a linha que falta. */
function EntryGroup(props: {
  title: string
  entries: readonly InitiativeEntry[]
  chosen: ReadonlySet<string>
  placed: ReadonlySet<string>
  onToggle: (entryId: string) => void
}) {
  return (
    <Show when={props.entries.length > 0}>
      <section class="space-y-1">
        <SectionLabel>{props.title}</SectionLabel>
        <ul>
          <For each={props.entries}>
            {(entry) => (
              <li>
                <EntryRow
                  entry={entry}
                  placed={props.placed.has(entry.id)}
                  chosen={props.chosen.has(entry.id)}
                  onToggle={() => props.onToggle(entry.id)}
                />
              </li>
            )}
          </For>
        </ul>
      </section>
    </Show>
  )
}

/** `aria-pressed` num `button` e não `role="checkbox"`: é o padrão da casa para
 *  alternador rico, e o único que o biome aceita (guia do front). */
function EntryRow(props: {
  entry: InitiativeEntry
  placed: boolean
  chosen: boolean
  onToggle: () => void
}) {
  return (
    <button
      type="button"
      aria-pressed={props.placed || props.chosen}
      disabled={props.placed}
      onClick={props.onToggle}
      class={cn(
        'flex w-full items-center gap-2 rounded-sm px-2 py-1 text-left text-sm transition-colors',
        props.placed
          ? 'text-muted-foreground'
          : props.chosen
            ? 'bg-accent text-grimorio-gold'
            : 'text-foreground hover:bg-accent',
      )}
    >
      <span class="min-w-0 flex-1 truncate">{props.entry.label}</span>
      <Show when={props.placed}>
        <span class="shrink-0 text-2xs text-muted-foreground">já no mapa</span>
      </Show>
    </button>
  )
}
