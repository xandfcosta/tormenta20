import { useQueryClient } from '@tanstack/react-query'
import { useForm } from '@tanstack/react-form'
import { useState } from 'react'
import { z } from 'zod'
import { CalendarClock } from 'lucide-react'
import { Button } from '@/shared/ui/button'
import { Card, CardContent, CardHeader } from '@/shared/ui/card'
import { Field, FieldError, FieldLabel } from '@/shared/ui/field'
import { Input } from '@/shared/ui/input'
import { SectionHeading } from '@/shared/ui/section-heading'
import { Textarea } from '@/shared/ui/textarea'
import { ApiError, api } from '@/shared/api/api'
import type { Campaign } from '@/shared/api/api'
import { applyServerErrors } from '@/shared/lib/form-errors'
import { campaignQueryOptions, campaignsQueryOptions } from '@/entities/campaign/queries'

// Mirrors the create form (campaign-new-page) so edit + create validate alike.
const campaignEditSchema = z.object({
  name: z.string().trim().min(1, 'Nome é obrigatório').max(120, 'Máximo 120 caracteres'),
  description: z.string().max(2000, 'Máximo 2000 caracteres'),
})

export function CampaignHeaderCard({ campaign }: { campaign: Campaign }) {
  const qc = useQueryClient()
  const [editing, setEditing] = useState(false)
  const [formError, setFormError] = useState<string | null>(null)

  const form = useForm({
    defaultValues: {
      name: campaign.name,
      description: campaign.description ?? '',
    },
    validators: { onSubmit: campaignEditSchema },
    onSubmit: async ({ value, formApi }) => {
      setFormError(null)
      try {
        await api.campaigns.update(campaign.id, {
          name: value.name.trim(),
          description: value.description,
        })
        qc.invalidateQueries({ queryKey: campaignsQueryOptions.queryKey })
        qc.invalidateQueries({
          queryKey: campaignQueryOptions(campaign.id).queryKey,
        })
        setEditing(false)
      } catch (e) {
        if (!applyServerErrors(formApi, e)) {
          setFormError(e instanceof ApiError ? e.message : 'Erro ao salvar')
        }
      }
    },
  })

  const cancel = () => {
    setEditing(false)
    setFormError(null)
    form.reset()
  }

  if (!editing) {
    return (
      <Card>
        <CardHeader className="flex flex-row items-start justify-between gap-4">
          <div className="space-y-1">
            <SectionHeading as="h2">{campaign.name}</SectionHeading>
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
      <CardContent>
        <form
          className="space-y-4"
          onSubmit={(e) => {
            e.preventDefault()
            e.stopPropagation()
            form.handleSubmit()
          }}
        >
          <form.Field
            name="name"
            validators={{ onChange: campaignEditSchema.shape.name }}
          >
            {(f) => {
              const invalid = f.state.meta.isTouched && !f.state.meta.isValid
              return (
                <Field data-invalid={invalid}>
                  <FieldLabel htmlFor={f.name}>Nome</FieldLabel>
                  <Input
                    id={f.name}
                    value={f.state.value}
                    onChange={(e) => f.handleChange(e.target.value)}
                    onBlur={f.handleBlur}
                    aria-invalid={invalid}
                    required
                  />
                  {invalid && <FieldError errors={f.state.meta.errors} />}
                </Field>
              )
            }}
          </form.Field>
          <form.Field
            name="description"
            validators={{ onChange: campaignEditSchema.shape.description }}
          >
            {(f) => {
              const invalid = f.state.meta.isTouched && !f.state.meta.isValid
              return (
                <Field data-invalid={invalid}>
                  <FieldLabel htmlFor={f.name}>Descrição</FieldLabel>
                  <Textarea
                    id={f.name}
                    value={f.state.value}
                    onChange={(e) => f.handleChange(e.target.value)}
                    onBlur={f.handleBlur}
                    rows={6}
                    aria-invalid={invalid}
                  />
                  {invalid && <FieldError errors={f.state.meta.errors} />}
                </Field>
              )
            }}
          </form.Field>
          {formError && <p className="text-sm text-destructive">{formError}</p>}
          <div className="flex justify-end gap-2">
            <Button type="button" variant="outline" onClick={cancel}>
              Cancelar
            </Button>
            <form.Subscribe
              selector={(s) => [s.canSubmit, s.isSubmitting] as const}
              children={([canSubmit, isSubmitting]) => (
                <Button type="submit" disabled={!canSubmit || isSubmitting}>
                  {isSubmitting ? 'Salvando…' : 'Salvar'}
                </Button>
              )}
            />
          </div>
        </form>
      </CardContent>
    </Card>
  )
}
