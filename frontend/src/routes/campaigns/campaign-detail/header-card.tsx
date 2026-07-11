import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { CalendarClock } from 'lucide-react'
import { Button } from '@/shared/ui/button'
import { Card, CardContent, CardHeader } from '@/shared/ui/card'
import { Input } from '@/shared/ui/input'
import { SectionHeading } from '@/shared/ui/section-heading'
import { Textarea } from '@/shared/ui/textarea'
import { ApiError, api } from '@/shared/api/api'
import type { Campaign } from '@/shared/api/api'
import {
  campaignQueryOptions,
  campaignsQueryOptions,
} from '@/shared/lib/queries'

export function CampaignHeaderCard({ campaign }: { campaign: Campaign }) {
  const qc = useQueryClient()
  const [editing, setEditing] = useState(false)
  const [name, setName] = useState(campaign.name)
  const [description, setDescription] = useState(campaign.description ?? '')
  const [error, setError] = useState<string | null>(null)

  const mutation = useMutation({
    mutationFn: () =>
      api.campaigns.update(campaign.id, { name, description }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: campaignsQueryOptions.queryKey })
      qc.invalidateQueries({
        queryKey: campaignQueryOptions(campaign.id).queryKey,
      })
      setEditing(false)
      setError(null)
    },
    onError: (e) => {
      setError(e instanceof ApiError ? e.message : 'Erro ao salvar')
    },
  })

  if (!editing) {
    return (
      <Card>
        <CardHeader className="flex flex-row items-start justify-between gap-4">
          <div className="space-y-1">
            <SectionHeading variant="aharadak" as="h1">
              {campaign.name}
            </SectionHeading>
            <p className="flex items-center gap-1 text-xs text-muted-foreground">
              <CalendarClock className="size-3" />
              Criada em{' '}
              {new Date(campaign.createdAt).toLocaleDateString('pt-BR')}
            </p>
          </div>
          <Button variant="outline" size="sm" onClick={() => setEditing(true)}>
            Editar
          </Button>
        </CardHeader>
        {campaign.description && (
          <CardContent className="text-sm">
            <p className="whitespace-pre-line text-muted-foreground">
              {campaign.description}
            </p>
          </CardContent>
        )}
      </Card>
    )
  }

  return (
    <Card>
      <CardHeader>
        <SectionHeading as="h2">Editar campanha</SectionHeading>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="space-y-1.5">
          <label className="text-sm font-medium" htmlFor="campaign-name">
            Nome
          </label>
          <Input
            id="campaign-name"
            value={name}
            onChange={(e) => setName(e.target.value)}
            required
          />
        </div>
        <div className="space-y-1.5">
          <label
            className="text-sm font-medium"
            htmlFor="campaign-description"
          >
            Descrição
          </label>
          <Textarea
            id="campaign-description"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            rows={6}
          />
        </div>
        {error && <p className="text-sm text-destructive">{error}</p>}
        <div className="flex justify-end gap-2">
          <Button
            variant="outline"
            onClick={() => {
              setEditing(false)
              setName(campaign.name)
              setDescription(campaign.description ?? '')
              setError(null)
            }}
          >
            Cancelar
          </Button>
          <Button
            disabled={mutation.isPending || !name.trim()}
            onClick={() => mutation.mutate()}
          >
            {mutation.isPending ? 'Salvando…' : 'Salvar'}
          </Button>
        </div>
      </CardContent>
    </Card>
  )
}
