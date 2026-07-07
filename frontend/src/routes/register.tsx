import { createFileRoute, Link, redirect, useNavigate } from '@tanstack/react-router'
import { useQueryClient } from '@tanstack/react-query'
import { useForm } from '@tanstack/react-form'
import { useState } from 'react'
import { z } from 'zod'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader } from '@/components/ui/card'
import { Field, FieldError, FieldGroup, FieldLabel } from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { PageChrome } from '@/components/ui/page-chrome'
import { SectionHeading } from '@/components/ui/section-heading'
import { ApiError, api } from '@/lib/api'
import { applyServerErrors } from '@/lib/form-errors'
import { meQueryOptions } from '@/lib/queries'

const registerSchema = z
  .object({
    email: z.email('E-mail inválido'),
    name: z.string(),
    password: z.string().min(8, 'A senha precisa ter ao menos 8 caracteres'),
    confirm: z.string().min(1, 'Confirme sua senha'),
  })
  .refine((v) => v.password === v.confirm, {
    path: ['confirm'],
    message: 'As senhas não conferem',
  })

export const Route = createFileRoute('/register')({
  beforeLoad: async ({ context }) => {
    const user = await context.queryClient.ensureQueryData(meQueryOptions)
    if (user) throw redirect({ to: '/' })
  },
  component: RegisterPage,
})

function RegisterPage() {
  const navigate = useNavigate()
  const qc = useQueryClient()
  const [formError, setFormError] = useState<string | null>(null)

  const form = useForm({
    defaultValues: { email: '', name: '', password: '', confirm: '' },
    validators: { onSubmit: registerSchema },
    onSubmit: async ({ value, formApi }) => {
      setFormError(null)
      try {
        const user = await api.auth.register({
          email: value.email,
          password: value.password,
          name: value.name || undefined,
        })
        qc.setQueryData(meQueryOptions.queryKey, user)
        await navigate({ to: '/' })
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
    <PageChrome width="compact" className="pt-10 sm:pt-16">
      <Card className="border-border/60 bg-card/80">
        <CardHeader className="gap-2">
          <SectionHeading as="h1" variant="kallyadranoch">
            Criar conta
          </SectionHeading>
          <CardDescription>Junte-se à mesa.</CardDescription>
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
            <FieldGroup>
              <form.Field name="email">
                {(f) => {
                  const invalid = f.state.meta.isTouched && !f.state.meta.isValid
                  return (
                    <Field data-invalid={invalid}>
                      <FieldLabel htmlFor={f.name}>E-mail</FieldLabel>
                      <Input
                        id={f.name}
                        name={f.name}
                        type="email"
                        autoComplete="email"
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

              <form.Field name="name">
                {(f) => {
                  const invalid = f.state.meta.isTouched && !f.state.meta.isValid
                  return (
                    <Field data-invalid={invalid}>
                      <FieldLabel htmlFor={f.name}>Nome (opcional)</FieldLabel>
                      <Input
                        id={f.name}
                        name={f.name}
                        value={f.state.value}
                        onChange={(e) => f.handleChange(e.target.value)}
                        onBlur={f.handleBlur}
                        aria-invalid={invalid}
                      />
                      {invalid && <FieldError errors={f.state.meta.errors} />}
                    </Field>
                  )
                }}
              </form.Field>

              <form.Field name="password">
                {(f) => {
                  const invalid = f.state.meta.isTouched && !f.state.meta.isValid
                  return (
                    <Field data-invalid={invalid}>
                      <FieldLabel htmlFor={f.name}>Senha</FieldLabel>
                      <Input
                        id={f.name}
                        name={f.name}
                        type="password"
                        autoComplete="new-password"
                        value={f.state.value}
                        onChange={(e) => f.handleChange(e.target.value)}
                        onBlur={f.handleBlur}
                        minLength={8}
                        aria-invalid={invalid}
                        required
                      />
                      {invalid && <FieldError errors={f.state.meta.errors} />}
                    </Field>
                  )
                }}
              </form.Field>

              <form.Field name="confirm">
                {(f) => {
                  const invalid = f.state.meta.isTouched && !f.state.meta.isValid
                  return (
                    <Field data-invalid={invalid}>
                      <FieldLabel htmlFor={f.name}>Confirmar senha</FieldLabel>
                      <Input
                        id={f.name}
                        name={f.name}
                        type="password"
                        autoComplete="new-password"
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
            </FieldGroup>

            {formError && <p className="text-sm text-destructive">{formError}</p>}

            <form.Subscribe
              selector={(s) => [s.isSubmitting, s.canSubmit] as const}
              children={([isSubmitting, canSubmit]) => (
                <Button type="submit" className="w-full" disabled={isSubmitting || !canSubmit}>
                  {isSubmitting ? 'Criando…' : 'Criar conta'}
                </Button>
              )}
            />

            <p className="text-center text-sm text-muted-foreground">
              Já tem conta?{' '}
              <Link to="/login" className="underline">Entrar</Link>
            </p>
          </form>
        </CardContent>
      </Card>
    </PageChrome>
  )
}
