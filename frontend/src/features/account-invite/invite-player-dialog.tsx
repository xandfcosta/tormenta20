import { UserPlus } from 'lucide-solid'
import { api } from '@/shared/api/api'
import { copyToClipboard } from '@/shared/lib/clipboard'
import { SingleUseLinkDialog } from '@/shared/ui/single-use-link-dialog'

export type InvitePlayerDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  /** Cunha um token de uso único. Só admin — o servidor é o portão. */
  onCreate: () => Promise<string>
  onCopy: (text: string) => Promise<void>
}

/**
 * Convite de CONTA (ALE-120) — o admin gera um link e entrega a quem vai
 * entrar; a pessoa abre e escolhe a PRÓPRIA senha, então o admin nunca vê
 * senha nenhuma.
 *
 * @example <InvitePlayerDialog open={open()} onOpenChange={setOpen} … />
 */
export function InvitePlayerDialog(props: InvitePlayerDialogProps) {
  return (
    <SingleUseLinkDialog
      open={props.open}
      onOpenChange={props.onOpenChange}
      icon={UserPlus}
      title="Convidar jogador"
      description="Gere um link e envie para quem vai entrar. Ele vale uma vez só, expira em 7 dias, e a pessoa escolhe a própria senha."
      createLabel="Gerar convite"
      linkLabel="Link de convite"
      caption="Cada convite serve para UMA conta. Gere outro para o próximo jogador."
      onCreate={props.onCreate}
      toUrl={(token) => `${window.location.origin}/register?convite=${encodeURIComponent(token)}`}
      onCopy={props.onCopy}
    />
  )
}

/** Wires the dialog to the backend and the system clipboard. */
export function InvitePlayer(props: { open: boolean; onOpenChange: (open: boolean) => void }) {
  return (
    <InvitePlayerDialog
      open={props.open}
      onOpenChange={props.onOpenChange}
      onCreate={async () => (await api.accountInvites.create()).token}
      onCopy={copyToClipboard}
    />
  )
}
