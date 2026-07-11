import {
  createFileRoute,
  redirect,
  useNavigate,
} from '@tanstack/react-router'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useForm } from '@tanstack/react-form'
import { useState } from 'react'
import { z } from 'zod'
import { Button } from '@/shared/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui/card'
import { Input } from '@/shared/ui/input'
import { PageChrome } from '@/shared/ui/page-chrome'
import { SectionHeading } from '@/shared/ui/section-heading'
import {
  Field,
  FieldError,
  FieldGroup,
  FieldLabel,
} from '@/shared/ui/field'
import { ApiError, api } from '@/shared/api/api'
import type { CampaignInvitePreview } from '@/shared/api/api'
import { applyServerErrors } from '@/shared/lib/form-errors'
import { campaignsQueryOptions } from '@/entities/campaign/queries'
import { charactersQueryOptions } from '@/entities/character/queries'
import { meQueryOptions } from '@/entities/user/queries'
/**
 * Self-join page. Post-OC1 the caller must own the character being
 * added. Two entry paths:
 *   1. Manual: player types the campaign id the GM shared.
 *   2. Invite: `?token=…` search param resolves to a campaign via the
 *      public `/invites/:token` endpoint, pre-fills the id, hides the
 *      manual field, and threads the token through the mutation so
 *      the server can verify.
 */
const joinSearchSchema = z.object({
  token: z.string().min(1).optional(),
})

export const Route = createFileRoute('/campaigns/join')({
  validateSearch: joinSearchSchema,
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
  const { token } = Route.useSearch()

  const invitePreview = useQuery<CampaignInvitePreview>({
    queryKey: ['invites', token] as const,
    queryFn: () => api.invites.resolve(token as string),
    enabled: !!token,
    retry: false,
  })

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
          ...(token ? { inviteToken: token } : {}),
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

  // When the invite resolves, fill in the campaignId so the user just
  // picks a character and submits. Setting it once via a flag avoids
  // clobbering manual edits if the user later tweaks the form.
  const [tokenApplied, setTokenApplied] = useState(false)
  if (
    token &&
    invitePreview.data &&
    !tokenApplied &&
    form.state.values.campaignId === 0
  ) {
    form.setFieldValue('campaignId', invitePreview.data.campaignId)
    setTokenApplied(true)
  }

  const noCharacters = (characters.data?.length ?? 0) === 0
  const inviteInvalid = !!token && invitePreview.isError
  const inviteLoading = !!token && invitePreview.isLoading

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
          Entrar em campanha
        </SectionHeading>

        {token && (
          <Card>
            <CardHeader>
              <CardTitle className="font-display tracking-wide">
                Convite
              </CardTitle>
            </CardHeader>
            <CardContent className="text-sm">
              {inviteLoading && (
                <p className="text-muted-foreground">
                  Verificando convite…
                </p>
              )}
              {inviteInvalid && (
                <p className="text-destructive">
                  Convite inválido ou expirado. Peça um novo link ao mestre.
                </p>
              )}
              {invitePreview.data && (
                <p>
                  Você foi convidado para{' '}
                  <span className="font-semibold text-[color:var(--primary)]">
                    {invitePreview.data.campaignName}
                  </span>
                  .
                </p>
              )}
            </CardContent>
          </Card>
        )}

        <Card>
          <CardHeader>
            <CardTitle className="font-display tracking-wide">
              Identificação
            </CardTitle>
          </CardHeader>
          <CardContent>
            <FieldGroup className="grid gap-4">
              {!token && (
                <form.Field name="campaignId">
                  {(f) => {
                    const invalid =
                      f.state.meta.isTouched && !f.state.meta.isValid
                    return (
                      <Field data-invalid={invalid}>
                        <FieldLabel htmlFor={f.name}>ID da campanha</FieldLabel>
                        <Input
                          id={f.name}
                          type="number"
                          inputMode="numeric"
                          min={1}
                          value={f.state.value || ''}
                          onChange={(e) =>
                            f.handleChange(Number(e.target.value))
                          }
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
              )}

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
                disabled={
                  isSubmitting ||
                  !canSubmit ||
                  noCharacters ||
                  inviteInvalid ||
                  inviteLoading
                }
              >
                {isSubmitting ? 'Entrando…' : 'Entrar'}
              </Button>
            )}
          />
        </div>
      </form>
    </PageChrome>
  )
}
