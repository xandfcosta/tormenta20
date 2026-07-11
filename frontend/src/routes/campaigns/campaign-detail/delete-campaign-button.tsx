import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { useState } from 'react'
import { Trash2 } from 'lucide-react'
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
import { api } from '@/shared/api/api'
import type { Campaign } from '@/shared/api/api'
import { campaignsQueryOptions } from '@/entities/campaign/queries'
export function DeleteCampaignButton({ campaign }: { campaign: Campaign }) {
  const qc = useQueryClient()
  const navigate = useNavigate()
  const [open, setOpen] = useState(false)
  const mutation = useMutation({
    mutationFn: () => api.campaigns.delete(campaign.id),
    onSuccess: async () => {
      qc.invalidateQueries({ queryKey: campaignsQueryOptions.queryKey })
      setOpen(false)
      await navigate({ to: '/campaigns' })
    },
  })

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button variant="destructive" size="sm">
          <Trash2 className="mr-1 size-3.5" /> Excluir campanha
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Excluir "{campaign.name}"?</DialogTitle>
          <DialogDescription>
            Todas as sessões e membros da campanha serão removidos. Esta ação
            não pode ser desfeita.
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button variant="outline" onClick={() => setOpen(false)}>
            Cancelar
          </Button>
          <Button
            variant="destructive"
            disabled={mutation.isPending}
            onClick={() => mutation.mutate()}
          >
            {mutation.isPending ? 'Excluindo…' : 'Excluir'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
