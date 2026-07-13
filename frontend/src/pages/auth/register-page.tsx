import { Link, useNavigate } from '@tanstack/react-router'
import { useQueryClient } from '@tanstack/react-query'
import { useForm } from '@tanstack/react-form'
import { useState } from 'react'
import { z } from 'zod'
import { Button } from '@/shared/ui/button'
import { Field, FieldError, FieldGroup, FieldLabel } from '@/shared/ui/field'
import { Input } from '@/shared/ui/input'
import { ApiError, api } from '@/shared/api/api'
import { applyServerErrors } from '@/shared/lib/form-errors'
import { meQueryOptions } from '@/entities/user/queries'
import { AuthShell } from './auth-shell'

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


export function RegisterPage() {
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
    <AuthShell
      title="Criar conta"
      subtitle="Junte-se à mesa."
      footer={
        <>
          Já tem conta?{' '}
          <Link to="/login" className="underline underline-offset-4">
            Entrar
          </Link>
        </>
      }
    >
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
            <Button
              type="submit"
              className="w-full"
              disabled={isSubmitting || !canSubmit}
            >
              {isSubmitting ? 'Criando…' : 'Criar conta'}
            </Button>
          )}
        />
      </form>
    </AuthShell>
  )
}
