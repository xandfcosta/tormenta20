import { useQueryClient } from '@tanstack/solid-query'
import { Show, createSignal } from 'solid-js'
import { campaignSessionQueryOptions, campaignSessionsQueryOptions } from '@/entities/session/queries'
import { sessionStatusMeta } from '@/entities/session/status'
import { ApiError, type Session, type SessionStatus, api } from '@/shared/api/api'
import { Button } from '@/shared/ui/button'
import { DialogInlineError } from '@/shared/ui/dialog-inline-error'
import { Input } from '@/shared/ui/input'
import { cn } from '@/shared/lib/utils'

/**
 * Session identity and lifecycle: number, title, status, and the start/end
 * controls. GM-only for the writes — a player sees the same card read-only,
 * because knowing whether the session is live is not privileged information.
 */
export function HeaderCard(props: { campaignId: number; session: Session; isGm: boolean }) {
  const queryClient = useQueryClient()
  const [editing, setEditing] = createSignal(false)
  const [title, setTitle] = createSignal(props.session.title ?? '')
  const [pending, setPending] = createSignal(false)
  const [error, setError] = createSignal<string | null>(null)

  const refresh = () => {
    queryClient.invalidateQueries({
      queryKey: campaignSessionQueryOptions(props.campaignId, props.session.id).queryKey,
    })
    queryClient.invalidateQueries({
      queryKey: campaignSessionsQueryOptions(props.campaignId).queryKey,
    })
  }

  const run = async (write: () => Promise<unknown>) => {
    setPending(true)
    setError(null)
    try {
      await write()
      refresh()
      setEditing(false)
    } catch (failure) {
      setError(failure instanceof ApiError ? failure.message : 'Erro ao salvar')
    } finally {
      setPending(false)
    }
  }

  const status = () => sessionStatusMeta(props.session.status as SessionStatus)

  return (
    <section class="space-y-3 rounded-sm border border-grimorio-iron bg-[var(--grimorio-panel)] p-3">
      <div class="flex flex-row items-center justify-between gap-2">
        <h2 class="font-heading text-sm uppercase tracking-wide text-grimorio-gold">
          Sessão {props.session.sessionNumber}
        </h2>
        <div class="flex items-center gap-2">
          <span
            class={cn(
              'rounded-sm px-1.5 py-0.5 text-[10px] uppercase tracking-widest',
              status().tone === 'live' && 'bg-primary text-primary-foreground',
              status().tone === 'planned' && 'bg-muted text-muted-foreground',
              status().tone === 'ended' && 'border border-border text-muted-foreground',
            )}
          >
            {status().label}
          </span>
          <Show when={props.isGm && !editing()}>
            <Button variant="outline" size="sm" onClick={() => setEditing(true)}>
              Editar
            </Button>
          </Show>
        </div>
      </div>

      <Show
        when={editing()}
        fallback={
          <p class="text-sm text-muted-foreground">
            {props.session.title || 'Sem título.'}
          </p>
        }
      >
        <div class="space-y-2">
          <Input
            value={title()}
            onInput={(event) => setTitle(event.currentTarget.value)}
            placeholder="Título da sessão…"
            aria-label="Título da sessão"
          />
          <DialogInlineError message={error()} />
          <div class="flex justify-end gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => {
                setEditing(false)
                setTitle(props.session.title ?? '')
                setError(null)
              }}
            >
              Cancelar
            </Button>
            <Button
              size="sm"
              disabled={pending()}
              onClick={() =>
                void run(() =>
                  api.sessions.update(props.campaignId, props.session.id, { title: title() }),
                )
              }
            >
              {pending() ? 'Salvando…' : 'Salvar'}
            </Button>
          </div>
        </div>
      </Show>

      <Show when={props.isGm}>
        <div class="flex flex-wrap gap-2">
          <Show when={props.session.status === 'planned'}>
            <Button
              size="sm"
              disabled={pending()}
              onClick={() => void run(() => api.sessions.start(props.campaignId, props.session.id))}
            >
              Iniciar sessão
            </Button>
          </Show>
          <Show when={props.session.status === 'active'}>
            <Button
              size="sm"
              variant="outline"
              disabled={pending()}
              onClick={() => void run(() => api.sessions.end(props.campaignId, props.session.id))}
            >
              Encerrar sessão
            </Button>
          </Show>
        </div>
      </Show>
    </section>
  )
}
