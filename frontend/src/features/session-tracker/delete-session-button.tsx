import { useQueryClient } from '@tanstack/solid-query'
import { useNavigate } from '@tanstack/solid-router'
import { Trash2 } from 'lucide-solid'
import { createSignal } from 'solid-js'
import { campaignSessionsQueryOptions } from '@/entities/session/queries'
import { api } from '@/shared/api/api'
import { Button } from '@/shared/ui/button'
import { ConfirmDialog } from '@/shared/ui/confirm-dialog'
import { toast } from '@/shared/ui/sonner'

/** Deletes the session and leaves the match — GM-only, irreversible. */
export function DeleteSessionButton(props: {
  campaignId: number
  sessionId: number
  sessionNumber: number
}) {
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  const [pending, setPending] = createSignal(false)

  const remove = async () => {
    setPending(true)
    try {
      await api.sessions.remove(props.campaignId, props.sessionId)
      queryClient.invalidateQueries({
        queryKey: campaignSessionsQueryOptions(props.campaignId).queryKey,
      })
      await navigate({ to: '/campaigns/$id', params: { id: String(props.campaignId) } })
    } catch {
      // The dialog is already gone by now, so this one is a toast.
      toast.error('Falha ao excluir a sessão')
    } finally {
      setPending(false)
    }
  }

  return (
    <ConfirmDialog
      title={`Excluir sessão ${props.sessionNumber}?`}
      description="Esta ação não pode ser desfeita."
      confirmLabel="Excluir"
      onConfirm={() => void remove()}
      trigger={(open) => (
        // Contorno, e não `destructive`: numa faixa de ajuste o vermelho cheio
        // fazia da ação mais rara e irreversível o elemento mais forte da tela.
        // O vermelho vive na CONFIRMAÇÃO, que é onde a decisão acontece.
        <Button
          variant="outline"
          size="sm"
          disabled={pending()}
          onClick={open}
          class="text-destructive hover:text-destructive"
        >
          <Trash2 aria-hidden="true" class="mr-1 size-3.5" /> Excluir
        </Button>
      )}
    />
  )
}
