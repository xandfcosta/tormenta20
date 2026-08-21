import { RefreshCw, Share2 } from 'lucide-solid'
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
  DialogTrigger,
} from '@/shared/ui/dialog'

export type InviteDialogProps = {
  /** Mints a fresh token, invalidating whatever was shared before. */
  onRotate: () => Promise<string>
  onCopy: (text: string) => Promise<void>
}

/**
 * Convite dialog — the GM opens it, mints a token, copies the URL and shares it
 * with a player. The token lives ONLY in this component's state, never in the
 * query cache: the DB is the source of truth and rotating invalidates the
 * previous value, so a cached copy would hand out a dead link.
 *
 * Both effects are injected, so the test drives it with no network or
 * `navigator.clipboard`.
 *
 * @example <InviteDialog onRotate={rotate} onCopy={copyToClipboard} />
 */
export function InviteDialog(props: InviteDialogProps) {
  const [open, setOpen] = createSignal(false)
  const [token, setToken] = createSignal<string | null>(null)
  const [error, setError] = createSignal<string | null>(null)
  const [pending, setPending] = createSignal(false)

  const inviteUrl = () => {
    const value = token()
    return value ? `${window.location.origin}/join/${value}` : null
  }

  const rotate = async () => {
    setPending(true)
    try {
      setToken(await props.onRotate())
      setError(null)
    } catch (failure) {
      setError(toSubmitFailure(failure).formError ?? 'Erro ao gerar convite')
    } finally {
      setPending(false)
    }
  }

  const onOpenChange = (next: boolean) => {
    setOpen(next)
    if (next) return
    setToken(null)
    setError(null)
  }

  return (
    <Dialog open={open()} onOpenChange={onOpenChange}>
      <DialogTrigger as={Button} variant="outline" size="sm">
        <Share2 aria-hidden="true" class="mr-1 size-4" /> Convite
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle class="flex items-center gap-2">
            <Share2 aria-hidden="true" class="size-5" />
            Convite para a campanha
          </DialogTitle>
          <DialogDescription>
            Envie o link abaixo para um jogador. Ao entrar, ele escolhe um personagem próprio e é
            adicionado automaticamente à mesa. Rotacionar invalida o link anterior.
          </DialogDescription>
        </DialogHeader>

        <Show when={inviteUrl()} fallback={<NoLinkYet />}>
          {(url) => (
            <CopyLinkRow url={url()} label="Link de convite" onCopy={props.onCopy}>
              Rotacionar invalida este link e gera um novo.
            </CopyLinkRow>
          )}
        </Show>

        <Show when={error()}>{(message) => <p class="text-sm text-destructive">{message()}</p>}</Show>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Fechar
          </Button>
          <Button disabled={pending()} onClick={rotate}>
            <Show
              when={token()}
              fallback={<Share2 aria-hidden="true" class="mr-1 size-4" />}
            >
              <RefreshCw aria-hidden="true" class="mr-1 size-4" />
            </Show>
            {pending() ? 'Gerando…' : token() ? 'Rotacionar convite' : 'Gerar link'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

/** Nothing minted yet — the dialog's resting state. */
function NoLinkYet() {
  return (
    <div class="rounded-sm border border-dashed p-4 text-center text-sm text-muted-foreground">
      Nenhum link gerado. Clique em "Gerar link" para criar.
    </div>
  )
}

/** Wires the dialog to the backend and the system clipboard. */
export function InviteButton(props: { campaignId: number }) {
  const rotate = async () => (await api.campaigns.rotateInvite(props.campaignId)).token
  return <InviteDialog onRotate={rotate} onCopy={copyToClipboard} />
}
