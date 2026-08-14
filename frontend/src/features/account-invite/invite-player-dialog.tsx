import { UserPlus } from 'lucide-solid'
import { Show, createSignal } from 'solid-js'
import { api } from '@/shared/api/api'
import { copyToClipboard } from '@/shared/lib/clipboard'
import { toSubmitFailure } from '@/shared/lib/form-errors'
import { Button } from '@/shared/ui/button'
import { CopyLinkRow } from '@/shared/ui/copy-link-row'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/shared/ui/dialog'

export type InvitePlayerDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  /** Mints a single-use token. Admin only — the server is the gate. */
  onCreate: () => Promise<string>
  onCopy: (text: string) => Promise<void>
}

/**
 * Convite de CONTA (ALE-120) — the admin mints a link and hands it to whoever
 * is joining the table; that person opens it and picks their OWN password, so
 * the admin never sees a password.
 *
 * Controlled from outside because it opens from the Hub's quick menu, which is
 * a popover: a `DialogTrigger` inside it would close its own opener.
 *
 * The token lives only in this component's state, never in the query cache —
 * each mint is a NEW link, so a cached one would show a link that is not the
 * one just created.
 *
 * @example <InvitePlayerDialog open={open()} onOpenChange={setOpen} … />
 */
export function InvitePlayerDialog(props: InvitePlayerDialogProps) {
  const [token, setToken] = createSignal<string | null>(null)
  const [error, setError] = createSignal<string | null>(null)
  const [pending, setPending] = createSignal(false)

  const inviteUrl = () => {
    const value = token()
    return value
      ? `${window.location.origin}/register?convite=${encodeURIComponent(value)}`
      : null
  }

  const create = async () => {
    setPending(true)
    try {
      setToken(await props.onCreate())
      setError(null)
    } catch (failure) {
      setError(toSubmitFailure(failure).formError ?? 'Erro ao gerar convite')
    } finally {
      setPending(false)
    }
  }

  const onOpenChange = (next: boolean) => {
    props.onOpenChange(next)
    if (next) return
    setToken(null)
    setError(null)
  }

  return (
    <Dialog open={props.open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle class="flex items-center gap-2">
            <UserPlus aria-hidden="true" class="size-5" />
            Convidar jogador
          </DialogTitle>
          <DialogDescription>
            Gere um link e envie para quem vai entrar. Ele vale uma vez só, expira em 7 dias, e a
            pessoa escolhe a própria senha.
          </DialogDescription>
        </DialogHeader>

        <Show
          when={inviteUrl()}
          fallback={
            <div class="rounded-md border border-dashed p-4 text-center text-sm text-muted-foreground">
              Nenhum link gerado. Clique em "Gerar convite" para criar.
            </div>
          }
        >
          {(url) => (
            <CopyLinkRow url={url()} label="Link de convite" onCopy={props.onCopy}>
              Cada convite serve para UMA conta. Gere outro para o próximo jogador.
            </CopyLinkRow>
          )}
        </Show>

        <Show when={error()}>{(message) => <p class="text-sm text-destructive">{message()}</p>}</Show>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Fechar
          </Button>
          <Button disabled={pending()} onClick={create}>
            <UserPlus aria-hidden="true" class="mr-1 size-4" />
            {pending() ? 'Gerando…' : token() ? 'Gerar outro' : 'Gerar convite'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
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
