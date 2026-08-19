import { useQueryClient } from '@tanstack/solid-query'
import { type JSX, Show, createSignal } from 'solid-js'
import { campaignSessionQueryOptions, campaignSessionsQueryOptions } from '@/entities/session/queries'
import { sessionStatusMeta } from '@/entities/session/status'
import { ApiError, type Session, type SessionStatus, api } from '@/shared/api/api'
import { Button } from '@/shared/ui/button'
import { DialogInlineError } from '@/shared/ui/dialog-inline-error'
import { Input } from '@/shared/ui/input'
import { cn } from '@/shared/lib/utils'

/**
 * Identidade e ciclo de vida da sessão: número, título, estado, e as ações de
 * começar e encerrar. Só o mestre escreve — o jogador vê o mesmo em leitura,
 * porque saber se a sessão está ao vivo não é informação privilegiada.
 *
 * Redesenhado como FAIXAS DE AJUSTE (pedido do dono). O que estava errado:
 *
 * 1. Moldura dentro de moldura — um cartão com borda dentro de um diálogo com
 *    borda são dois quadros para uma coisa só.
 * 2. Hierarquia invertida: "Excluir sessão", a ação mais rara e a única
 *    irreversível, era o elemento mais forte da tela (grande e vermelho),
 *    enquanto "Encerrar", que é o fim normal de toda noite, era um botão
 *    pequeno solto no meio do texto. Agora o vermelho fica onde a decisão
 *    acontece, que é o diálogo de confirmação.
 * 3. Estado disputando com ação: o crachá "AO VIVO" (informação) dividia o
 *    canto com "Editar" (ação), com o mesmo peso.
 *
 * A faixa é o padrão de "uma linha, uma decisão": nome, o que ela faz em uma
 * frase, e o botão à direita.
 */
export function HeaderCard(props: {
  campaignId: number
  session: Session
  isGm: boolean
  /** A zona de risco entra por baixo, como última faixa — quem a monta é a
   *  cena, porque excluir NAVEGA para fora da sessão. */
  danger?: JSX.Element
}) {
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
    <section class="divide-y divide-grimorio-iron">
      <header class="flex flex-wrap items-baseline justify-between gap-x-3 gap-y-1 pb-3">
        <h2 class="font-heading text-sm uppercase tracking-wide text-grimorio-gold">
          Sessão {props.session.sessionNumber}
        </h2>
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
      </header>

      <Show
        when={editing()}
        fallback={
          <SettingRow
            label="Título"
            hint={props.session.title || 'Esta sessão ainda não tem nome.'}
            action={
              <Show when={props.isGm}>
                <Button variant="outline" size="sm" onClick={() => setEditing(true)}>
                  Editar
                </Button>
              </Show>
            }
          />
        }
      >
        <div class="space-y-2 py-3">
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

      <Show when={props.isGm && props.session.status === 'planned'}>
        <SettingRow
          label="Iniciar sessão"
          hint="A mesa passa a ver a cena ao vivo."
          action={
            <Button
              size="sm"
              disabled={pending()}
              onClick={() => void run(() => api.sessions.start(props.campaignId, props.session.id))}
            >
              Iniciar
            </Button>
          }
        />
      </Show>

      <Show when={props.isGm && props.session.status === 'active'}>
        <SettingRow
          label="Encerrar sessão"
          hint="A mesa para de receber a cena. A crônica e as fichas ficam como estão."
          action={
            <Button
              size="sm"
              variant="outline"
              disabled={pending()}
              onClick={() => void run(() => api.sessions.end(props.campaignId, props.session.id))}
            >
              Encerrar
            </Button>
          }
        />
      </Show>

      <Show when={props.danger}>
        <SettingRow
          label="Excluir sessão"
          hint="Apaga a sessão e o histórico dela. Não dá para desfazer."
          action={props.danger}
        />
      </Show>
    </section>
  )
}

/**
 * Uma decisão por linha: o nome, o que ela faz numa frase, e o botão à direita.
 * A frase não é enfeite — é ela que deixa o botão ser curto ("Encerrar" em vez
 * de "Encerrar sessão") sem virar adivinhação.
 */
function SettingRow(props: { label: string; hint: string; action: JSX.Element }) {
  return (
    <div class="flex flex-wrap items-center justify-between gap-x-4 gap-y-2 py-3">
      <div class="min-w-0 flex-1">
        <p class="text-sm text-foreground">{props.label}</p>
        <p class="text-xs text-muted-foreground">{props.hint}</p>
      </div>
      {props.action}
    </div>
  )
}
