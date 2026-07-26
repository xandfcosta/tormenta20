import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui/card'
import { Combobox } from '@/shared/ui/combobox'
import { Input } from '@/shared/ui/input'
import { NumberInput } from '@/shared/ui/number-input'
import {
  Field,
  FieldDescription,
  FieldError,
  FieldGroup,
  FieldLabel,
} from '@/shared/ui/field'
import {
  type FieldApi,
  useCreationWizard,
} from '@/features/character-build/creation-wizard-context'
import { toOptions } from '@/features/character-build/wizard-steps'

export function IdentidadeStep() {
  const { form, options } = useCreationWizard()
  return (
    <Card>
      <CardHeader>
        <CardTitle className="font-display tracking-wide">Identidade</CardTitle>
      </CardHeader>
      <CardContent>
        <FieldGroup className="grid gap-4 sm:grid-cols-2 sm:gap-6">
          <form.Field name="name">
            {(f: FieldApi) => {
              const invalid = f.state.meta.isTouched && !f.state.meta.isValid
              return (
                <Field data-invalid={invalid} className="sm:col-span-2">
                  <FieldLabel htmlFor={f.name}>Nome</FieldLabel>
                  <Input
                    id={f.name}
                    value={f.state.value as string}
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

          <form.Field name="god">
            {(f: FieldApi) => (
              <Field>
                <FieldLabel htmlFor={f.name}>Deus (opcional)</FieldLabel>
                <Combobox
                  id={f.name}
                  options={toOptions(options.gods)}
                  value={(f.state.value as string) ?? ''}
                  onChange={f.handleChange}
                  placeholder="Nenhum"
                  searchPlaceholder="Buscar deuses…"
                  emptyMessage="Nenhum deus encontrado."
                  allowClear
                  clearLabel="Nenhum"
                />
              </Field>
            )}
          </form.Field>

          <form.Field name="size">
            {(f: FieldApi) => {
              const invalid = f.state.meta.isTouched && !f.state.meta.isValid
              return (
                <Field data-invalid={invalid}>
                  <FieldLabel htmlFor={f.name}>Tamanho</FieldLabel>
                  <Combobox
                    id={f.name}
                    options={toOptions(options.sizes)}
                    value={f.state.value as string}
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
            {(f: FieldApi) => {
              const invalid = f.state.meta.isTouched && !f.state.meta.isValid
              return (
                <Field data-invalid={invalid}>
                  <FieldLabel htmlFor={f.name}>Deslocamento (m)</FieldLabel>
                  <NumberInput
                    id={f.name}
                    min={0}
                    max={120}
                    step={1}
                    value={f.state.value as number}
                    onChange={(v) => f.handleChange(v)}
                    onBlur={f.handleBlur}
                    aria-invalid={invalid}
                  />
                  <FieldDescription>
                    Padrão: 9 metros por ação de movimento.
                  </FieldDescription>
                  {invalid && <FieldError errors={f.state.meta.errors} />}
                </Field>
              )
            }}
          </form.Field>
        </FieldGroup>
      </CardContent>
    </Card>
  )
}
