import { createFileRoute, redirect, useNavigate } from '@tanstack/react-router'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useForm } from '@tanstack/react-form'
import { useState } from 'react'
import { z } from 'zod'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { NumberInput } from '@/components/ui/number-input'
import { Combobox } from '@/components/ui/combobox'
import { Badge } from '@/components/ui/badge'
import {
  Field,
  FieldDescription,
  FieldError,
  FieldGroup,
  FieldLabel,
} from '@/components/ui/field'
import { ApiError, api } from '@/lib/api'
import type { CreateCharacterInput } from '@/lib/api'
import { applyServerErrors } from '@/lib/form-errors'
import {
  ATTRIBUTE_KEYS,
  attributePresetForClass,
} from '@tormenta20/t20-data'
import {
  characterOptionsQueryOptions,
  charactersQueryOptions,
  meQueryOptions,
} from '@/lib/queries'

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

  if (options.isLoading) return <p className="p-6">Loading options…</p>
  if (!options.data) return <p className="p-6 text-destructive">Failed to load options</p>
  const opts = options.data

  return (
    <form
      className="mx-auto h-full max-w-3xl space-y-6 overflow-y-auto p-6"
      onSubmit={(e) => {
        e.preventDefault()
        e.stopPropagation()
        form.handleSubmit()
      }}
    >
      <h1 className="text-3xl font-semibold">New character</h1>

      <Card>
        <CardHeader><CardTitle>Identity</CardTitle></CardHeader>
        <CardContent>
          <FieldGroup className="grid gap-4 sm:grid-cols-2 sm:gap-6">
            <form.Field name="name">
              {(f) => {
                const invalid = f.state.meta.isTouched && !f.state.meta.isValid
                return (
                  <Field data-invalid={invalid} className="sm:col-span-2">
                    <FieldLabel htmlFor={f.name}>Name</FieldLabel>
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
                    <FieldLabel htmlFor={f.name}>Origin</FieldLabel>
                    <Combobox
                      id={f.name}
                      options={toOptions(opts.origins)}
                      value={f.state.value}
                      onChange={f.handleChange}
                      placeholder="Select origin"
                      searchPlaceholder="Search origins…"
                      emptyMessage="No origin matches."
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
                    <FieldLabel htmlFor={f.name}>God (optional)</FieldLabel>
                    <Combobox
                      id={f.name}
                      options={toOptions(opts.gods)}
                      value={f.state.value ?? ''}
                      onChange={f.handleChange}
                      placeholder="None"
                      searchPlaceholder="Search gods…"
                      emptyMessage="No god matches."
                      allowClear
                      clearLabel="None"
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
                      placeholder="Select size"
                      searchPlaceholder="Search sizes…"
                      emptyMessage="No size matches."
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
        <CardHeader><CardTitle>Races</CardTitle></CardHeader>
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
                    <FieldDescription>Select at least one race.</FieldDescription>
                  ) : null}
                </Field>
              )
            }}
          </form.Field>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Classes (multi-class supported)</CardTitle>
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
                    + Add class
                  </Button>
                  {arrayInvalid ? (
                    <FieldError errors={classesField.state.meta.errors} />
                  ) : !items.length ? (
                    <FieldDescription>Add at least one class.</FieldDescription>
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
        <CardHeader><CardTitle>Vitals</CardTitle></CardHeader>
        <CardContent>
          <FieldGroup className="grid gap-4 sm:grid-cols-4 sm:gap-6">
            <NumberField form={form} name="hpMax" label="HP max" min={1} />
            <NumberField form={form} name="hpCurrent" label="HP current" min={0} />
            <NumberField form={form} name="mpMax" label="MP max" min={0} />
            <NumberField form={form} name="mpCurrent" label="MP current" min={0} />
          </FieldGroup>
        </CardContent>
      </Card>

      <Card>
        <CardHeader><CardTitle>Attributes</CardTitle></CardHeader>
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
          Cancel
        </Button>
        <form.Subscribe
          selector={(s) => [s.isSubmitting, s.canSubmit] as const}
          children={([isSubmitting, canSubmit]) => (
            <Button type="submit" disabled={isSubmitting || !canSubmit}>
              {isSubmitting ? 'Creating…' : 'Create character'}
            </Button>
          )}
        />
      </div>
    </form>
  )
}

// TanStack Form's API type is heavily generic — `any` here keeps row
// helpers usable without leaking 10 type parameters across the file.
// eslint-disable-next-line @typescript-eslint/no-explicit-any
type FormApi = any
// eslint-disable-next-line @typescript-eslint/no-explicit-any
type FieldApi = any

function ClassEntryRow({
  index,
  classOptions,
  form,
  onRemove,
  onPrimaryClassPicked,
}: {
  index: number
  classOptions: string[]
  form: FormApi
  onRemove: () => void
  onPrimaryClassPicked?: (className: string) => void
}) {
  return (
    <div className="space-y-1.5">
      <div className="flex items-end gap-2">
        <form.Field name={`classes[${index}].className`}>
          {(f: FieldApi) => {
            const invalid = f.state.meta.isTouched && !f.state.meta.isValid
            return (
              <Field data-invalid={invalid} className="flex-1">
                <FieldLabel htmlFor={f.name}>Class</FieldLabel>
                <Combobox
                  id={f.name}
                  options={toOptions(classOptions)}
                  value={f.state.value}
                  onChange={(v: string) => {
                    f.handleChange(v)
                    if (v && onPrimaryClassPicked) onPrimaryClassPicked(v)
                  }}
                  placeholder="Select class"
                  searchPlaceholder="Search classes…"
                  emptyMessage="No class matches."
                />
                {invalid && <FieldError errors={f.state.meta.errors} />}
              </Field>
            )
          }}
        </form.Field>
        <form.Field name={`classes[${index}].level`}>
          {(f: FieldApi) => {
            const invalid = f.state.meta.isTouched && !f.state.meta.isValid
            return (
              <Field data-invalid={invalid} className="w-24">
                <FieldLabel htmlFor={f.name}>Level</FieldLabel>
                <NumberInput
                  id={f.name}
                  min={1}
                  max={20}
                  value={f.state.value}
                  onChange={(v) => f.handleChange(v)}
                  onBlur={f.handleBlur}
                  aria-invalid={invalid}
                />
                {invalid && <FieldError errors={f.state.meta.errors} />}
              </Field>
            )
          }}
        </form.Field>
        <Button type="button" variant="outline" size="sm" onClick={onRemove}>
          Remove
        </Button>
      </div>
    </div>
  )
}

type NumericFieldName = {
  [K in keyof CharacterFormValues]: CharacterFormValues[K] extends number ? K : never
}[keyof CharacterFormValues]

function NumberField({
  form,
  name,
  label,
  min,
  max,
}: {
  form: FormApi
  name: NumericFieldName
  label: string
  min?: number
  max?: number
}) {
  return (
    <form.Field name={name}>
      {(f: FieldApi) => {
        const invalid = f.state.meta.isTouched && !f.state.meta.isValid
        return (
          <Field data-invalid={invalid}>
            <FieldLabel htmlFor={f.name}>{label}</FieldLabel>
            <NumberInput
              id={f.name}
              min={min}
              max={max}
              value={f.state.value as number}
              onChange={(v) => f.handleChange(v)}
              onBlur={f.handleBlur}
              aria-invalid={invalid}
            />
            {invalid && <FieldError errors={f.state.meta.errors} />}
          </Field>
        )
      }}
    </form.Field>
  )
}
