import { createFileRoute, redirect, useNavigate } from '@tanstack/react-router'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useForm } from '@tanstack/react-form'
import { useState } from 'react'
import { z } from 'zod'
import { Button } from '@/shared/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui/card'
import { Input } from '@/shared/ui/input'
import { NumberInput } from '@/shared/ui/number-input'
import { Combobox } from '@/shared/ui/combobox'
import { Badge } from '@/shared/ui/badge'
import { PageChrome } from '@/shared/ui/page-chrome'
import { SectionHeading } from '@/shared/ui/section-heading'
import { ClassEntryRow } from '@/features/character-build/class-entry-row'
import { NumberField } from '@/features/character-build/number-field'
import {
  Field,
  FieldDescription,
  FieldError,
  FieldGroup,
  FieldLabel,
} from '@/shared/ui/field'
import { ApiError, api } from '@/shared/api/api'
import type { CreateCharacterInput } from '@/shared/api/api'
import { applyServerErrors } from '@/shared/lib/form-errors'
import {
  ATTRIBUTE_KEYS,
  attributePresetForClass,
} from '@tormenta20/t20-data'
import { characterOptionsQueryOptions, charactersQueryOptions } from '@/entities/character/queries'
import { meQueryOptions } from '@/entities/user/queries'
export const Route = createFileRoute('/characters/new')({
  beforeLoad: async ({ context, location }) => {
    const user = await context.queryClient.ensureQueryData(meQueryOptions)
    if (!user) throw redirect({ to: '/login', search: { redirect: location.href } })
  },
  loader: ({ context }) =>
    context.queryClient.ensureQueryData(characterOptionsQueryOptions),
  component: NewCharacterPage,
})

const classEntrySchema = z.object({
  className: z.string().min(1, 'Choose a class'),
  level: z.number().int().min(1).max(20),
})

const characterSchema = z
  .object({
    name: z.string().min(1, 'Name is required'),
    races: z.array(z.string()).min(1, 'Select at least one race'),
    origin: z.string().min(1, 'Origin is required'),
    classes: z.array(classEntrySchema).min(1, 'Add at least one class'),
    god: z.string().optional(),
    hpMax: z.number().int().min(1),
    hpCurrent: z.number().int().min(0),
    mpMax: z.number().int().min(0),
    mpCurrent: z.number().int().min(0),
    strength: z.number().int().min(-5).max(10),
    dexterity: z.number().int().min(-5).max(10),
    constitution: z.number().int().min(-5).max(10),
    intelligence: z.number().int().min(-5).max(10),
    wisdom: z.number().int().min(-5).max(10),
    charisma: z.number().int().min(-5).max(10),
    size: z.string().min(1, 'Size is required'),
    displacement: z.number().int().min(0).max(120),
  })
  .superRefine((v, ctx) => {
    if (v.hpCurrent > v.hpMax) {
      ctx.addIssue({
        code: 'custom',
        path: ['hpCurrent'],
        message: 'HP current cannot exceed HP max',
      })
    }
    if (v.mpCurrent > v.mpMax) {
      ctx.addIssue({
        code: 'custom',
        path: ['mpCurrent'],
        message: 'MP current cannot exceed MP max',
      })
    }
    const seen = new Set<string>()
    v.classes.forEach((entry, index) => {
      if (!entry.className) return
      if (seen.has(entry.className)) {
        ctx.addIssue({
          code: 'custom',
          path: ['classes', index, 'className'],
          message: `Class "${entry.className}" already added — combine levels in one entry instead`,
        })
      } else {
        seen.add(entry.className)
      }
    })
  })

type CharacterFormValues = z.infer<typeof characterSchema>

const defaults: CharacterFormValues = {
  name: '',
  races: [],
  origin: '',
  classes: [],
  god: '',
  hpMax: 10,
  hpCurrent: 10,
  mpMax: 0,
  mpCurrent: 0,
  strength: 0,
  dexterity: 0,
  constitution: 0,
  intelligence: 0,
  wisdom: 0,
  charisma: 0,
  size: 'Médio',
  displacement: 9,
}

const toOptions = (values: string[]) => values.map((v) => ({ value: v, label: v }))

function NewCharacterPage() {
  const navigate = useNavigate()
  const qc = useQueryClient()
  const options = useQuery(characterOptionsQueryOptions)
  const [formError, setFormError] = useState<string | null>(null)

  const form = useForm({
    defaultValues: defaults,
    validators: { onSubmit: characterSchema },
    onSubmit: async ({ value, formApi }) => {
      setFormError(null)
      const payload: CreateCharacterInput = {
        ...value,
        god: value.god ? value.god : undefined,
      }
      try {
        const created = await api.characters.create(payload)
        qc.invalidateQueries({ queryKey: charactersQueryOptions.queryKey })
        await navigate({ to: '/characters/$id', params: { id: String(created.id) } })
      } catch (e) {
        if (!applyServerErrors(formApi, e) && e instanceof ApiError) {
          setFormError(e.message)
        } else if (!(e instanceof ApiError)) {
          setFormError('Unexpected error. Try again.')
        }
      }
    },
  })

  if (options.isLoading)
    return (
      <PageChrome>
        <p>Carregando opções…</p>
      </PageChrome>
    )
  if (!options.data)
    return (
      <PageChrome>
        <p className="text-destructive">Falha ao carregar opções</p>
      </PageChrome>
    )
  const opts = options.data

  return (
    <PageChrome>
      <form
        className="space-y-6"
        onSubmit={(e) => {
          e.preventDefault()
          e.stopPropagation()
          form.handleSubmit()
        }}
      >
      <SectionHeading variant="kallyadranoch" as="h1">
        Novo personagem
      </SectionHeading>

      <Card>
        <CardHeader>
          <CardTitle className="font-display tracking-wide">Identidade</CardTitle>
        </CardHeader>
        <CardContent>
          <FieldGroup className="grid gap-4 sm:grid-cols-2 sm:gap-6">
            <form.Field name="name">
              {(f) => {
                const invalid = f.state.meta.isTouched && !f.state.meta.isValid
                return (
                  <Field data-invalid={invalid} className="sm:col-span-2">
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

            <form.Field name="origin">
              {(f) => {
                const invalid = f.state.meta.isTouched && !f.state.meta.isValid
                return (
                  <Field data-invalid={invalid}>
                    <FieldLabel htmlFor={f.name}>Origem</FieldLabel>
                    <Combobox
                      id={f.name}
                      options={toOptions(opts.origins)}
                      value={f.state.value}
                      onChange={f.handleChange}
                      placeholder="Selecionar origem"
                      searchPlaceholder="Buscar origens…"
                      emptyMessage="Nenhuma origem encontrada."
                    />
                    {invalid && <FieldError errors={f.state.meta.errors} />}
                  </Field>
                )
              }}
            </form.Field>

            <form.Field name="god">
              {(f) => {
                const invalid = f.state.meta.isTouched && !f.state.meta.isValid
                return (
                  <Field data-invalid={invalid}>
                    <FieldLabel htmlFor={f.name}>Deus (opcional)</FieldLabel>
                    <Combobox
                      id={f.name}
                      options={toOptions(opts.gods)}
                      value={f.state.value ?? ''}
                      onChange={f.handleChange}
                      placeholder="Nenhum"
                      searchPlaceholder="Buscar deuses…"
                      emptyMessage="Nenhum deus encontrado."
                      allowClear
                      clearLabel="Nenhum"
                    />
                    {invalid && <FieldError errors={f.state.meta.errors} />}
                  </Field>
                )
              }}
            </form.Field>

            <form.Field name="size">
              {(f) => {
                const invalid = f.state.meta.isTouched && !f.state.meta.isValid
                return (
                  <Field data-invalid={invalid}>
                    <FieldLabel htmlFor={f.name}>Tamanho</FieldLabel>
                    <Combobox
                      id={f.name}
                      options={toOptions(opts.sizes)}
                      value={f.state.value}
                      onChange={f.handleChange}
                      placeholder="Selecionar tamanho"
                      searchPlaceholder="Buscar tamanhos…"
                      emptyMessage="Nenhum tamanho encontrado."
                    />
                    <FieldDescription>Padrão: Médio.</FieldDescription>
                    {invalid && <FieldError errors={f.state.meta.errors} />}
                  </Field>
                )
              }}
            </form.Field>

            <form.Field name="displacement">
              {(f) => {
                const invalid = f.state.meta.isTouched && !f.state.meta.isValid
                return (
                  <Field data-invalid={invalid}>
                    <FieldLabel htmlFor={f.name}>Deslocamento (m)</FieldLabel>
                    <NumberInput
                      id={f.name}
                      min={0}
                      max={120}
                      step={1}
                      value={f.state.value}
                      onChange={(v) => f.handleChange(v)}
                      onBlur={f.handleBlur}
                      aria-invalid={invalid}
                    />
                    <FieldDescription>Padrão: 9 metros por ação de movimento.</FieldDescription>
                    {invalid && <FieldError errors={f.state.meta.errors} />}
                  </Field>
                )
              }}
            </form.Field>
          </FieldGroup>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="font-display tracking-wide">Raças</CardTitle>
        </CardHeader>
        <CardContent>
          <form.Field name="races" mode="array">
            {(racesField) => {
              const value = racesField.state.value
              const invalid = racesField.state.meta.isTouched && !racesField.state.meta.isValid
              return (
                <Field data-invalid={invalid}>
                  <div className="flex flex-wrap gap-2">
                    {opts.races.map((r) => {
                      const active = value.includes(r)
                      return (
                        <button
                          type="button"
                          key={r}
                          onClick={() => {
                            const next = active
                              ? value.filter((x) => x !== r)
                              : [...value, r]
                            racesField.handleChange(next)
                          }}
                          onBlur={racesField.handleBlur}
                        >
                          <Badge variant={active ? 'default' : 'outline'}>{r}</Badge>
                        </button>
                      )
                    })}
                  </div>
                  {invalid ? (
                    <FieldError errors={racesField.state.meta.errors} />
                  ) : !value.length ? (
                    <FieldDescription>Selecione ao menos uma raça.</FieldDescription>
                  ) : null}
                </Field>
              )
            }}
          </form.Field>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="font-display tracking-wide">
            Classes (multiclasse suportada)
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          <form.Field name="classes" mode="array">
            {(classesField) => {
              const items = classesField.state.value
              const arrayInvalid =
                classesField.state.meta.isTouched && !classesField.state.meta.isValid
              return (
                <>
                  {items.map((_, i) => (
                    <ClassEntryRow
                      key={i}
                      index={i}
                      classOptions={opts.classes}
                      form={form}
                      onRemove={() => classesField.removeValue(i)}
                      onPrimaryClassPicked={
                        i === 0 ? applyClassAttributePreset : undefined
                      }
                    />
                  ))}
                  <Button
                    type="button"
                    variant="outline"
                    onClick={() => {
                      const first = opts.classes[0]
                      classesField.pushValue({ className: first, level: 1 })
                      if (items.length === 0) applyClassAttributePreset(first)
                    }}
                  >
                    + Adicionar classe
                  </Button>
                  {arrayInvalid ? (
                    <FieldError errors={classesField.state.meta.errors} />
                  ) : !items.length ? (
                    <FieldDescription>Adicione ao menos uma classe.</FieldDescription>
                  ) : null}
                  {items.length > 1 && (
                    <p className="text-xs text-amber-700 dark:text-amber-300">
                      ⚠ Multiclasse no nível 1 não é padrão das regras (T20
                      adquire-se via Poder de Multiclasse em níveis mais
                      altos). O preset de atributos usa apenas a primeira
                      classe.
                    </p>
                  )}
                </>
              )

              function applyClassAttributePreset(className: string) {
                const preset = attributePresetForClass(className)
                if (!preset) return
                for (const attr of ATTRIBUTE_KEYS) {
                  form.setFieldValue(attr, preset[attr])
                }
              }
            }}
          </form.Field>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="font-display tracking-wide">Vitalidade</CardTitle>
        </CardHeader>
        <CardContent>
          <FieldGroup className="grid gap-4 sm:grid-cols-4 sm:gap-6">
            <NumberField form={form} name="hpMax" label="PV máx" min={1} />
            <NumberField form={form} name="hpCurrent" label="PV atual" min={0} />
            <NumberField form={form} name="mpMax" label="PM máx" min={0} />
            <NumberField form={form} name="mpCurrent" label="PM atual" min={0} />
          </FieldGroup>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="font-display tracking-wide">Atributos</CardTitle>
        </CardHeader>
        <CardContent>
          <FieldGroup className="grid gap-4 sm:grid-cols-3 sm:gap-6 lg:grid-cols-6">
            <NumberField form={form} name="strength" label="Força" min={-5} max={10} />
            <NumberField form={form} name="dexterity" label="Destreza" min={-5} max={10} />
            <NumberField form={form} name="constitution" label="Constituição" min={-5} max={10} />
            <NumberField form={form} name="intelligence" label="Inteligência" min={-5} max={10} />
            <NumberField form={form} name="wisdom" label="Sabedoria" min={-5} max={10} />
            <NumberField form={form} name="charisma" label="Carisma" min={-5} max={10} />
          </FieldGroup>
        </CardContent>
      </Card>

      {formError && (
        <p className="text-sm text-destructive">{formError}</p>
      )}

      <div className="flex justify-end gap-2">
        <Button type="button" variant="outline" onClick={() => navigate({ to: '/characters' })}>
          Cancelar
        </Button>
        <form.Subscribe
          selector={(s) => [s.isSubmitting, s.canSubmit] as const}
          children={([isSubmitting, canSubmit]) => (
            <Button type="submit" disabled={isSubmitting || !canSubmit}>
              {isSubmitting ? 'Criando…' : 'Criar personagem'}
            </Button>
          )}
        />
      </div>
      </form>
    </PageChrome>
  )
}

