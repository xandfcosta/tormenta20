import { useQuery, useQueryClient } from '@tanstack/solid-query'
import { useNavigate } from '@tanstack/solid-router'
import { UserPlus } from 'lucide-solid'
import { Show, createSignal } from 'solid-js'
import {
  adminBackupsQueryOptions,
  adminInvitesQueryOptions,
  adminStatusQueryOptions,
  adminUsersQueryOptions,
} from '@/entities/admin/queries'
import { meQueryOptions } from '@/entities/user/queries'
import { InvitePlayer } from '@/features/account-invite/invite-player-dialog'
import { OpenInvitesPanel } from '@/features/admin/open-invites-panel'
import { PlayersPanel } from '@/features/admin/players-panel'
import { ResetLink } from '@/features/admin/reset-link-dialog'
import { ServerPanel } from '@/features/admin/server-panel'
import { type AdminUser, api } from '@/shared/api/api'
import { SceneShell } from '@/shared/layout/scene-shell'
import { copyToClipboard } from '@/shared/lib/clipboard'
import { createSfx } from '@/shared/lib/sfx'
import { useUi } from '@/shared/stores/ui-context'
import { Button } from '@/shared/ui/button'
import { SkeletonCardGrid } from '@/shared/ui/skeleton'

/**
 * Administração (ALE-120) — quem está na mesa, os convites em aberto e o
 * estado do servidor.
 *
 * A cena é composição: os três painéis são features e não se conhecem. Quem
 * pode ver isto é o SERVIDOR que decide (`requireAdmin`); a rota só evita
 * mostrar uma tela que responderia 403 inteira.
 */
export function AdminPage() {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const sfx = createSfx(useUi())

  const me = useQuery(() => meQueryOptions)
  const users = useQuery(() => adminUsersQueryOptions)
  const invites = useQuery(() => adminInvitesQueryOptions)
  const status = useQuery(() => adminStatusQueryOptions)
  const backups = useQuery(() => adminBackupsQueryOptions)

  const [inviting, setInviting] = createSignal(false)
  const [resetting, setResetting] = createSignal<AdminUser | null>(null)

  // Uma ação de admin muda mais de uma leitura (apagar conta mexe em usuários,
  // mesas e status), então a invalidação é do prefixo — enumerar as quatro é o
  // que deixa uma para trás na próxima que for adicionada.
  const refresh = () => queryClient.invalidateQueries({ queryKey: ['admin'] })

  const deleteUser = async (user: AdminUser) => {
    await api.admin.deleteUser(user.id)
    await refresh()
    await queryClient.invalidateQueries({ queryKey: ['campaigns'] })
  }

  const backup = async () => {
    await api.admin.createBackup()
    await refresh()
  }

  return (
    <SceneShell
      dense
      title="Administração"
      onBack={() => {
        sfx('select')
        navigate({ to: '/' })
      }}
      onEnter={() => sfx('transition')}
      headerRight={
        <Button size="sm" onClick={() => setInviting(true)}>
          <UserPlus aria-hidden="true" class="size-4 sm:mr-1" />
          <span class="hidden sm:inline">Convidar</span>
        </Button>
      }
    >
      <Show when={users.isLoading} fallback={null}>
        <SkeletonCardGrid count={3} />
      </Show>

      {/* A cena ocupa a largura que recebe (guia do front): uma coluna no
          telefone, e duas a partir de lg, com os jogadores — a lista que
          cresce — na coluna maior. */}
      <div class="grid w-full items-start gap-4 lg:grid-cols-[3fr_2fr]">
        <Show when={users.data}>
          {(list) => (
            <PlayersPanel
              users={list()}
              currentUserId={me.data?.id ?? 0}
              onResetPassword={setResetting}
              onDelete={deleteUser}
            />
          )}
        </Show>

        <div class="flex flex-col gap-4">
          <Show when={invites.data}>
            {(list) => <OpenInvitesPanel invites={list()} onCopy={copyToClipboard} />}
          </Show>

          <Show when={status.data}>
            {(current) => (
              <ServerPanel status={current()} backups={backups.data ?? []} onBackup={backup} />
            )}
          </Show>
        </div>
      </div>

      <InvitePlayer
        open={inviting()}
        onOpenChange={(open) => {
          setInviting(open)
          if (!open) void refresh()
        }}
      />
      <ResetLink user={resetting()} onClose={() => setResetting(null)} />
    </SceneShell>
  )
}
