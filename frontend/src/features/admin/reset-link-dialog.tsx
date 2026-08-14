import { KeyRound } from 'lucide-solid'
import type { AdminUser } from '@/shared/api/api'
import { api } from '@/shared/api/api'
import { copyToClipboard } from '@/shared/lib/clipboard'
import { SingleUseLinkDialog } from '@/shared/ui/single-use-link-dialog'

export type ResetLinkDialogProps = {
  /** A conta que o link abre; `null` fecha o diálogo. */
  user: AdminUser | null
  onClose: () => void
  onCreate: (userId: number) => Promise<string>
  onCopy: (text: string) => Promise<void>
}

/**
 * Link de redefinição de senha (ALE-120). O admin gera e entrega; quem recebe
 * escolhe a própria senha — o admin nunca vê nem digita a senha de ninguém, que
 * foi a decisão do dono quando isto foi desenhado.
 *
 * Vale 24h, contra os 7 dias do convite: este link abre uma conta que JÁ
 * existe, então um esquecido numa conversa vale mais para um estranho.
 *
 * @example <ResetLinkDialog user={target()} onClose={() => setTarget(null)} … />
 */
export function ResetLinkDialog(props: ResetLinkDialogProps) {
  const name = () => props.user?.name || props.user?.email || ''
  return (
    <SingleUseLinkDialog
      open={props.user !== null}
      onOpenChange={(open) => {
        if (!open) props.onClose()
      }}
      icon={KeyRound}
      title="Redefinir senha"
      description={`Envie o link para ${name()}. Vale uma vez só, expira em 24 horas, e quem recebe escolhe a própria senha — você não digita senha nenhuma.`}
      createLabel="Gerar link"
      linkLabel="Link de redefinição"
      caption="Enquanto o link não for usado, a senha atual continua valendo."
      onCreate={() => props.onCreate(props.user?.id ?? 0)}
      toUrl={(token) => `${window.location.origin}/redefinir-senha?token=${encodeURIComponent(token)}`}
      onCopy={props.onCopy}
    />
  )
}

/** Wires the dialog to the backend and the system clipboard. */
export function ResetLink(props: { user: AdminUser | null; onClose: () => void }) {
  return (
    <ResetLinkDialog
      user={props.user}
      onClose={props.onClose}
      onCreate={async (userId) => (await api.admin.passwordReset(userId)).token}
      onCopy={copyToClipboard}
    />
  )
}
