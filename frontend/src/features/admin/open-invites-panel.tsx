import { For, Show } from 'solid-js'
import type { AccountInvite } from '@/shared/api/api'
import { CopyLinkRow } from '@/shared/ui/copy-link-row'
import { inviteRegisterUrl } from '@/features/account-invite/invite-link'
import { FramedPanel } from '@/shared/ui/framed-panel'

export type OpenInvitesPanelProps = {
  invites: AccountInvite[]
  onCopy: (text: string) => Promise<void>
}

/**
 * Os convites já entregues e ainda válidos (ALE-120). Existe para o admin
 * COPIAR de novo em vez de cunhar um segundo: um jogador que perdeu a mensagem
 * não precisa de outro link, precisa do mesmo.
 */
export function OpenInvitesPanel(props: OpenInvitesPanelProps) {
  return (
    <FramedPanel>
      <h2 class="font-heading text-sm uppercase tracking-[0.18em] text-grimorio-gold/80">
        Convites abertos ({props.invites.length})
      </h2>
      <Show
        when={props.invites.length > 0}
        fallback={
          <p class="mt-3 text-sm text-muted-foreground">
            Nenhum convite pendente. Todo mundo que foi convidado já entrou.
          </p>
        }
      >
        <ul class="mt-3 space-y-3">
          <For each={props.invites}>
            {(invite) => (
              <li>
                <CopyLinkRow
                  url={inviteRegisterUrl(window.location.origin, invite.token)}
                  label={`Link de convite ${invite.token.slice(0, 6)}`}
                  onCopy={props.onCopy}
                >
                  Expira em {expiryLabel(invite.expiresAt)}.
                </CopyLinkRow>
              </li>
            )}
          </For>
        </ul>
      </Show>
    </FramedPanel>
  )
}

/**
 * "3 dias" diz mais que uma data: o que importa é quanto ainda dá para esperar.
 *
 * Arredonda em vez de truncar porque um convite recém-criado, com 7 dias menos
 * alguns segundos, anunciava "6 dias" — o número que o admin lê tem de bater
 * com o que ele acabou de prometer ao jogador.
 */
function expiryLabel(iso: string): string {
  const remaining = new Date(iso).getTime() - Date.now()
  if (Number.isNaN(remaining)) return iso
  const days = remaining / 86_400_000
  if (days >= 1) return Math.round(days) === 1 ? '1 dia' : `${Math.round(days)} dias`
  const hours = Math.max(1, Math.round(remaining / 3_600_000))
  return hours === 1 ? '1 hora' : `${hours} horas`
}
