import { useQueryClient } from '@tanstack/solid-query'
import { Show, createSignal } from 'solid-js'
import { campaignSessionQueryOptions } from '@/entities/session/queries'
import { ApiError, type Session, api } from '@/shared/api/api'
import { Button } from '@/shared/ui/button'
import { DialogInlineError } from '@/shared/ui/dialog-inline-error'
import { Textarea } from '@/shared/ui/textarea'

/** The GM's running notes for the session — what happened, XP, treasure. */
export function NotesCard(props: { campaignId: number; session: Session }) {
  const queryClient = useQueryClient()
  const [editing, setEditing] = createSignal(false)
  const [notes, setNotes] = createSignal(props.session.notes ?? '')
  const [pending, setPending] = createSignal(false)
  const [error, setError] = createSignal<string | null>(null)

  const save = async () => {
    setPending(true)
    setError(null)
    try {
      await api.sessions.update(props.campaignId, props.session.id, { notes: notes() })
      queryClient.invalidateQueries({
        queryKey: campaignSessionQueryOptions(props.campaignId, props.session.id).queryKey,
      })
      setEditing(false)
    } catch (failure) {
      setError(failure instanceof ApiError ? failure.message : 'Erro ao salvar')
    } finally {
      setPending(false)
    }
  }

  return (
    <section class="space-y-3 rounded-sm border border-grimorio-iron bg-[var(--grimorio-panel)] p-3 text-sm">
      <div class="flex flex-row items-center justify-between">
        <h2 class="font-heading text-sm uppercase tracking-wide text-grimorio-gold">Notas</h2>
        <Show when={!editing()}>
          <Button variant="outline" size="sm" onClick={() => setEditing(true)}>
            Editar
          </Button>
        </Show>
      </div>

      <Show
        when={editing()}
        fallback={
          <p class="whitespace-pre-line text-muted-foreground">
            {props.session.notes || 'Nenhuma nota ainda.'}
          </p>
        }
      >
        <Textarea
          value={notes()}
          onInput={(event) => setNotes(event.currentTarget.value)}
          rows={10}
          aria-label="Notas da sessão"
          placeholder="Anote acontecimentos, decisões, XP, tesouro…"
        />
        <DialogInlineError message={error()} />
        <div class="flex justify-end gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={() => {
              setEditing(false)
              setNotes(props.session.notes ?? '')
              setError(null)
            }}
          >
            Cancelar
          </Button>
          <Button size="sm" disabled={pending()} onClick={() => void save()}>
            {pending() ? 'Salvando…' : 'Salvar'}
          </Button>
        </div>
      </Show>
    </section>
  )
}
