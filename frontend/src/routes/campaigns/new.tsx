import {
  createFileRoute,
  redirect,
  useNavigate,
} from '@tanstack/react-router'
import { useQueryClient } from '@tanstack/react-query'
import { useForm } from '@tanstack/react-form'
import { useState } from 'react'
import { z } from 'zod'
import { Button } from '@/shared/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui/card'
import { Input } from '@/shared/ui/input'
import { PageChrome } from '@/shared/ui/page-chrome'
import { SectionHeading } from '@/shared/ui/section-heading'
import { Textarea } from '@/shared/ui/textarea'
import {
  Field,
  FieldError,
  FieldGroup,
  FieldLabel,
} from '@/shared/ui/field'
import { ApiError, api } from '@/shared/api/api'
import type { CreateCampaignInput } from '@/shared/api/api'
import { applyServerErrors } from '@/shared/lib/form-errors'
import { campaignsQueryOptions, meQueryOptions } from '@/shared/lib/queries'

export const Route = createFileRoute('/campaigns/new')({
  beforeLoad: async ({ context, location }) => {
    const user = await context.queryClient.ensureQueryData(meQueryOptions)
    if (!user) throw redirect({ to: '/login', search: { redirect: location.href } })
  },
  component: NewCampaignPage,
})

const campaignSchema = z.object({
  name: z.string().min(1, 'Nome é obrigatório').max(120),
  description: z.string().max(2000).optional(),
})

type CampaignFormValues = z.infer<typeof campaignSchema>

const defaults: CampaignFormValues = { name: '', description: '' }

function NewCampaignPage() {
  const navigate = useNavigate()
  const qc = useQueryClient()
  const [formError, setFormError] = useState<string | null>(null)

  const form = useForm({
    defaultValues: defaults,
    validators: { onSubmit: campaignSchema },
    onSubmit: async ({ value, formApi }) => {
      setFormError(null)
      const payload: CreateCampaignInput = {
        name: value.name,
        ...(value.description ? { description: value.description } : {}),
      }
      try {
        const created = await api.campaigns.create(payload)
        qc.invalidateQueries({ queryKey: campaignsQueryOptions.queryKey })
        await navigate({
          to: '/campaigns/$id',
          params: { id: String(created.id) },
        })
      } catch (e) {
        if (!applyServerErrors(formApi, e) && e instanceof ApiError) {
          setFormError(e.message)
        } else if (!(e instanceof ApiError)) {
          setFormError('Erro inesperado. Tente novamente.')
        }
      }
    },
  })

  return (
    <PageChrome width="compact">
      <form
        className="space-y-6"
        onSubmit={(e) => {
          e.preventDefault()
          e.stopPropagation()
          form.handleSubmit()
        }}
      >
        <SectionHeading variant="aharadak" as="h1">
          Nova campanha
        </SectionHeading>

        <Card>
          <CardHeader>
            <CardTitle className="font-display tracking-wide">
              Identificação
            </CardTitle>
          </CardHeader>
          <CardContent>
            <FieldGroup className="grid gap-4">
              <form.Field name="name">
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

              <form.Field name="description">
                {(f) => {
                  const invalid = f.state.meta.isTouched && !f.state.meta.isValid
                  return (
                    <Field data-invalid={invalid}>
                      <FieldLabel htmlFor={f.name}>Descrição (opcional)</FieldLabel>
                      <Textarea
                        id={f.name}
                        value={f.state.value ?? ''}
                        onChange={(e) => f.handleChange(e.target.value)}
                        onBlur={f.handleBlur}
                        aria-invalid={invalid}
                        rows={6}
                      />
                      {invalid && <FieldError errors={f.state.meta.errors} />}
                    </Field>
                  )
                }}
              </form.Field>
            </FieldGroup>
          </CardContent>
        </Card>

        {formError && (
          <p className="text-sm text-destructive">{formError}</p>
        )}

        <div className="flex justify-end gap-2">
          <Button
            type="button"
            variant="outline"
            onClick={() => navigate({ to: '/campaigns' })}
          >
            Cancelar
          </Button>
          <form.Subscribe
            selector={(s) => [s.isSubmitting, s.canSubmit] as const}
            children={([isSubmitting, canSubmit]) => (
              <Button type="submit" disabled={isSubmitting || !canSubmit}>
                {isSubmitting ? 'Criando…' : 'Criar campanha'}
              </Button>
            )}
          />
        </div>
      </form>
    </PageChrome>
  )
}
