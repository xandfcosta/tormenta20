import {
  createFileRoute,
  redirect,
  useNavigate,
} from '@tanstack/react-router'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useForm } from '@tanstack/react-form'
import { useState } from 'react'
import { z } from 'zod'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import {
  Field,
  FieldError,
  FieldGroup,
  FieldLabel,
} from '@/components/ui/field'
import { ApiError, api } from '@/lib/api'
import { applyServerErrors } from '@/lib/form-errors'
import {
  campaignsQueryOptions,
  charactersQueryOptions,
  meQueryOptions,
} from '@/lib/queries'

/**
 * Self-join page. Post-OC1 (whole-flow audit) the backend requires the
 * caller to own the character being added to a campaign — meaning the
 * old "GM adds any character" UX no longer works. This page is the
 * player-side of the pair: they type the campaign id the GM gave them,
 * pick one of their own characters, and submit.
 */
export const Route = createFileRoute('/campaigns/join')({
  beforeLoad: async ({ context, location }) => {
    const user = await context.queryClient.ensureQueryData(meQueryOptions)
    if (!user) {
      throw redirect({ to: '/login', search: { redirect: location.href } })
    }
    await context.queryClient.ensureQueryData(charactersQueryOptions)
  },
  component: JoinCampaignPage,
})

const joinSchema = z.object({
  campaignId: z.number().int().positive(),
  characterId: z.number().int().positive(),
})

type JoinFormValues = z.infer<typeof joinSchema>

const defaults: JoinFormValues = {
  campaignId: 0,
  characterId: 0,
}

function JoinCampaignPage() {
  const navigate = useNavigate()
  const qc = useQueryClient()
  const characters = useQuery(charactersQueryOptions)
  const [formError, setFormError] = useState<string | null>(null)

  const form = useForm({
    defaultValues: defaults,
    validators: { onSubmit: joinSchema },
    onSubmit: async ({ value, formApi }) => {
      setFormError(null)
      try {
        await api.members.add(value.campaignId, {
          characterId: value.characterId,
          role: 'player',
        })
        qc.invalidateQueries({ queryKey: campaignsQueryOptions.queryKey })
        qc.invalidateQueries({
          queryKey: ['characters', value.characterId, 'campaigns'],
        })
        await navigate({
          to: '/campaigns/$id',
          params: { id: String(value.campaignId) },
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

  const noCharacters = (characters.data?.length ?? 0) === 0

  return (
    <form
      className="mx-auto h-full max-w-2xl space-y-6 overflow-y-auto p-6"
      onSubmit={(e) => {
        e.preventDefault()
        e.stopPropagation()
        form.handleSubmit()
      }}
    >
      <h1 className="text-3xl font-semibold">Entrar em campanha</h1>

      <Card>
        <CardHeader>
          <CardTitle>Identificação</CardTitle>
        </CardHeader>
        <CardContent>
          <FieldGroup className="grid gap-4">
            <form.Field name="campaignId">
              {(f) => {
                const invalid = f.state.meta.isTouched && !f.state.meta.isValid
                return (
                  <Field data-invalid={invalid}>
                    <FieldLabel htmlFor={f.name}>ID da campanha</FieldLabel>
                    <Input
                      id={f.name}
                      type="number"
                      inputMode="numeric"
                      min={1}
                      value={f.state.value || ''}
                      onChange={(e) => f.handleChange(Number(e.target.value))}
                      onBlur={f.handleBlur}
                      aria-invalid={invalid}
                      placeholder="Ex.: 42"
                      required
                    />
                    <p className="text-xs text-muted-foreground">
                      O mestre da mesa envia esse número.
                    </p>
                    {invalid && <FieldError errors={f.state.meta.errors} />}
                  </Field>
                )
              }}
            </form.Field>

            <form.Field name="characterId">
              {(f) => {
                const invalid = f.state.meta.isTouched && !f.state.meta.isValid
                return (
                  <Field data-invalid={invalid}>
                    <FieldLabel htmlFor={f.name}>Personagem</FieldLabel>
                    <select
                      id={f.name}
                      className="rounded-md border border-input bg-background px-3 py-2 text-sm"
                      value={f.state.value || ''}
                      onChange={(e) => f.handleChange(Number(e.target.value))}
                      onBlur={f.handleBlur}
                      aria-invalid={invalid}
                      required
                    >
                      <option value="" disabled>
                        Selecione um personagem
                      </option>
                      {characters.data?.map((c) => (
                        <option key={c.id} value={c.id}>
                          {c.name} — Nível {c.level}
                        </option>
                      ))}
                    </select>
                    {invalid && <FieldError errors={f.state.meta.errors} />}
                  </Field>
                )
              }}
            </form.Field>

            {noCharacters && (
              <p className="text-sm text-muted-foreground">
                Você ainda não tem personagens. Crie um primeiro.
              </p>
            )}
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
            <Button
              type="submit"
              disabled={isSubmitting || !canSubmit || noCharacters}
            >
              {isSubmitting ? 'Entrando…' : 'Entrar'}
            </Button>
          )}
        />
      </div>
    </form>
  )
}
