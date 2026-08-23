import { KeyRound, Trash2 } from 'lucide-solid'
import { For, Show } from 'solid-js'
import type { AdminUser } from '@/shared/api/api'
import { Button } from '@/shared/ui/button'
import { ConfirmDialog } from '@/shared/ui/confirm-dialog'
import { FramedPanel } from '@/shared/ui/framed-panel'

export type PlayersPanelProps = {
  users: AdminUser[]
  /** Quem está olhando: a própria linha não oferece apagar. */
  currentUserId: number
  onResetPassword: (user: AdminUser) => void
  onDelete: (user: AdminUser) => Promise<void>
}

/**
 * Quem está na mesa (ALE-120), com o que cada conta tem e as duas ações que
 * existem sobre ela.
 *
 * Os números não são enfeite: são o que a confirmação de apagar usa para dizer
 * o que se perde ANTES de perder — as fichas somem com a conta, as mesas mudam
 * de dono.
 */
export function PlayersPanel(props: PlayersPanelProps) {
  return (
    <FramedPanel>
      <h2 class="font-heading text-sm uppercase tracking-[0.18em] text-grimorio-gold/80">
        Jogadores ({props.users.length})
      </h2>
      <ul class="mt-3 divide-y divide-grimorio-iron/60">
        <For each={props.users}>
          {(user) => (
            <PlayerRow
              user={user}
              isSelf={user.id === props.currentUserId}
              onResetPassword={() => props.onResetPassword(user)}
              onDelete={() => props.onDelete(user)}
            />
          )}
        </For>
      </ul>
    </FramedPanel>
  )
}

function PlayerRow(props: {
  user: AdminUser
  isSelf: boolean
  onResetPassword: () => void
  onDelete: () => Promise<void>
}) {
  const name = () => props.user.name || props.user.email
  return (
    <li class="flex flex-wrap items-center gap-x-3 gap-y-1 py-2">
      <span class="min-w-0 flex-1">
        <span class="block truncate font-heading text-sm tracking-wide">{name()}</span>
        <span class="block truncate text-xs text-muted-foreground">{props.user.email}</span>
      </span>
      <span class="text-xs text-muted-foreground">{belongings(props.user)}</span>
      <span class="flex items-center gap-1">
        <Button
          variant="outline"
          size="sm"
          aria-label={`Redefinir a senha de ${name()}`}
          onClick={props.onResetPassword}
        >
          <KeyRound aria-hidden="true" class="size-4" />
        </Button>
        <Show when={!props.isSelf}>
          <ConfirmDialog
            title={`Apagar a conta de ${name()}?`}
            description={deletionCost(props.user)}
            confirmLabel="Apagar conta"
            destructive
            onConfirm={() => void props.onDelete()}
            trigger={(open) => (
              <Button
                variant="destructive"
                size="sm"
                aria-label={`Apagar a conta de ${name()}`}
                onClick={open}
              >
                <Trash2 aria-hidden="true" class="size-4" />
              </Button>
            )}
          />
        </Show>
      </span>
    </li>
  )
}

/** "admin · você" e o que a conta tem, na mesma linha do olhar. */
function belongings(user: AdminUser): string {
  const parts = [plural(user.campaigns, 'mesa', 'mesas'), plural(user.characters, 'ficha', 'fichas')]
  return user.isAdmin ? `admin · ${parts.join(' · ')}` : parts.join(' · ')
}

/**
 * O texto da confirmação diz o preço EXATO desta conta. Um aviso genérico
 * ("esta ação não pode ser desfeita") não distingue apagar uma conta vazia de
 * apagar a do jogador que mestra duas campanhas.
 */
function deletionCost(user: AdminUser): string {
  const fichas = `${plural(user.characters, 'ficha', 'fichas')} vão junto`
  if (user.campaigns === 0) return `As ${fichas}. Não há mesas para transferir.`
  return `As ${fichas}, e ${plural(user.campaigns, 'mesa passa', 'mesas passam')} para você.`
}

function plural(count: number, one: string, many: string): string {
  return `${count} ${count === 1 ? one : many}`
}
