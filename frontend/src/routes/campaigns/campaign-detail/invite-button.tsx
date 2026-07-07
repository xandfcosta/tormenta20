import { useMutation } from '@tanstack/react-query'
import { useState } from 'react'
import { Check, Copy, RefreshCw, Share2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { ApiError, api } from '@/lib/api'

/**
 * Convite dialog — GM opens, clicks "Gerar link" to mint a fresh token,
 * copies the URL, shares with the player. Rotating replaces the token
 * (any previously-shared link 404s from that moment). Token is only
 * ever held in local component state — never cached in React Query
 * because the source of truth is the DB and rotation invalidates the
 * previous value.
 */
export function InviteButton({ campaignId }: { campaignId: number }) {
  const [open, setOpen] = useState(false)
  const [token, setToken] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [copied, setCopied] = useState(false)

  const mutation = useMutation({
    mutationFn: () => api.campaigns.rotateInvite(campaignId),
    onSuccess: (data) => {
      setToken(data.token)
      setError(null)
      setCopied(false)
    },
    onError: (e) => {
      setError(e instanceof ApiError ? e.message : 'Erro ao gerar convite')
    },
  })

  const inviteUrl = token
    ? `${window.location.origin}/join/${token}`
    : null

  const copy = async () => {
    if (!inviteUrl) return
    await navigator.clipboard.writeText(inviteUrl)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  return (
    <Dialog
      open={open}
      onOpenChange={(o) => {
        setOpen(o)
        if (!o) {
          setToken(null)
          setError(null)
          setCopied(false)
        }
      }}
    >
      <DialogTrigger asChild>
        <Button variant="outline" size="sm">
          <Share2 className="mr-1 size-4" /> Convite
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2 font-display tracking-wide">
            <Share2 className="size-5 text-[color:var(--primary)]" />
            Convite para a campanha
          </DialogTitle>
          <DialogDescription>
            Envie o link abaixo para um jogador. Ao entrar, ele escolhe
            um personagem próprio e é adicionado automaticamente à mesa.
            Rotacionar invalida o link anterior.
          </DialogDescription>
        </DialogHeader>

        {token ? (
          <div className="space-y-3">
            <div className="flex items-center gap-2">
              <Input
                readOnly
                value={inviteUrl ?? ''}
                aria-label="Link de convite"
                className="font-mono text-xs"
              />
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={copy}
                aria-label="Copiar link"
              >
                {copied ? (
                  <Check className="size-4" />
                ) : (
                  <Copy className="size-4" />
                )}
              </Button>
            </div>
            <p className="text-xs text-muted-foreground">
              Rotacionar invalida este link e gera um novo.
            </p>
          </div>
        ) : (
          <div className="rounded-md border border-dashed p-4 text-center text-sm text-muted-foreground">
            Nenhum link gerado. Clique em "Gerar link" para criar.
          </div>
        )}

        {error && <p className="text-sm text-destructive">{error}</p>}

        <DialogFooter>
          <Button variant="outline" onClick={() => setOpen(false)}>
            Fechar
          </Button>
          <Button
            disabled={mutation.isPending}
            onClick={() => mutation.mutate()}
          >
            {token ? (
              <RefreshCw className="mr-1 size-4" />
            ) : (
              <Share2 className="mr-1 size-4" />
            )}
            {mutation.isPending
              ? 'Gerando…'
              : token
                ? 'Rotacionar convite'
                : 'Gerar link'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
