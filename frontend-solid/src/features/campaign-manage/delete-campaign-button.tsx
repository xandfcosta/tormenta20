import { useQueryClient } from '@tanstack/solid-query'
import { useNavigate } from '@tanstack/solid-router'
import { Trash2 } from 'lucide-solid'
import { createSignal } from 'solid-js'
import { campaignsQueryOptions } from '@/entities/campaign/queries'
import { type Campaign, api } from '@/shared/api/api'
import { Button } from '@/shared/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/shared/ui/dialog'

export type DeleteCampaignDialogProps = {
  campaignName: string
  onConfirm: () => Promise<void>
}

/**
 * Confirmation gate for deleting a chronicle. The trigger only opens the
 * dialog — nothing irreversible happens until the second click, and the title
 * names the campaign so the GM sees WHICH one is about to burn.
 *
 * @example <DeleteCampaignDialog campaignName={c.name} onConfirm={remove} />
 */
export function DeleteCampaignDialog(props: DeleteCampaignDialogProps) {
  const [open, setOpen] = createSignal(false)
  const [pending, setPending] = createSignal(false)

  const confirm = async () => {
    setPending(true)
    try {
      await props.onConfirm()
      setOpen(false)
    } finally {
      setPending(false)
    }
  }

  return (
    <Dialog open={open()} onOpenChange={setOpen}>
      <DialogTrigger as={Button} variant="destructive" size="sm">
        <Trash2 aria-hidden="true" class="mr-1 size-3.5" /> Excluir campanha
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Excluir "{props.campaignName}"?</DialogTitle>
          <DialogDescription>
            Todas as sessões e membros da campanha serão removidos. Esta ação não pode ser desfeita.
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button variant="outline" onClick={() => setOpen(false)}>
            Cancelar
          </Button>
          <Button variant="destructive" disabled={pending()} onClick={confirm}>
            {pending() ? 'Excluindo…' : 'Excluir'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

/** Wires the confirmation to the backend, then leaves for the book of chronicles. */
export function DeleteCampaignButton(props: { campaign: Campaign }) {
  const queryClient = useQueryClient()
  const navigate = useNavigate()

  const remove = async () => {
    await api.campaigns.delete(props.campaign.id)
    await queryClient.invalidateQueries({ queryKey: campaignsQueryOptions.queryKey })
    await navigate({ to: '/campaigns' })
  }

  return <DeleteCampaignDialog campaignName={props.campaign.name} onConfirm={remove} />
}
