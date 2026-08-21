import type { LucideIcon } from 'lucide-solid'
import { Show, createSignal } from 'solid-js'
import { Dynamic } from 'solid-js/web'
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

export type SingleUseLinkDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  icon: LucideIcon
  title: string
  description: string
  /** Rótulo do botão que gera o primeiro link ("Gerar convite"). */
  createLabel: string
  /** Nome acessível do campo — há mais de um tipo de link no app. */
  linkLabel: string
  /** A linha sob o campo, explicando o que ESTE link faz. */
  caption: string
  /** Cunha o token. O servidor é quem decide se o chamador pode. */
  onCreate: () => Promise<string>
  toUrl: (token: string) => string
  onCopy: (text: string) => Promise<void>
}

/**
 * Gerar um link de uso único, mostrar e copiar (ALE-120). Dois recursos têm
 * exatamente esta forma — o convite de conta e a redefinição de senha — e eles
 * vivem em features diferentes, que não podem se importar: por isso a casca
 * mora em `shared/ui` e cada uma traz o seu texto e a sua URL.
 *
 * Controlado de fora porque abre a partir de um menu popover: um
 * `DialogTrigger` lá dentro fecharia o próprio abridor.
 *
 * O token vive SÓ no estado deste componente, nunca no cache de query — cada
 * geração é um link novo, e um valor cacheado mostraria um que não é o que se
 * acabou de criar.
 *
 * @example <SingleUseLinkDialog title="Convidar jogador" toUrl={(t) => `/register?convite=${t}`} … />
 */
export function SingleUseLinkDialog(props: SingleUseLinkDialogProps) {
  const [token, setToken] = createSignal<string | null>(null)
  const [error, setError] = createSignal<string | null>(null)
  const [pending, setPending] = createSignal(false)

  const url = () => {
    const value = token()
    return value ? props.toUrl(value) : null
  }

  const create = async () => {
    setPending(true)
    try {
      setToken(await props.onCreate())
      setError(null)
    } catch (failure) {
      setError(toSubmitFailure(failure).formError ?? 'Erro ao gerar o link')
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
            <Dynamic component={props.icon} aria-hidden="true" class="size-5" />
            {props.title}
          </DialogTitle>
          <DialogDescription>{props.description}</DialogDescription>
        </DialogHeader>

        <Show
          when={url()}
          fallback={
            <div class="rounded-sm border border-dashed p-4 text-center text-sm text-muted-foreground">
              Nenhum link gerado. Clique em "{props.createLabel}" para criar.
            </div>
          }
        >
          {(link) => (
            <CopyLinkRow url={link()} label={props.linkLabel} onCopy={props.onCopy}>
              {props.caption}
            </CopyLinkRow>
          )}
        </Show>

        <Show when={error()}>{(message) => <p class="text-sm text-destructive">{message()}</p>}</Show>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Fechar
          </Button>
          <Button disabled={pending()} onClick={create}>
            <Dynamic component={props.icon} aria-hidden="true" class="mr-1 size-4" />
            {pending() ? 'Gerando…' : token() ? 'Gerar outro' : props.createLabel}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
