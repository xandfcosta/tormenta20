import { Button } from '@/shared/ui/button'
import { Combobox } from '@/shared/ui/combobox'
import { Field, FieldError, FieldLabel } from '@/shared/ui/field'
import { NumberInput } from '@/shared/ui/number-input'

// TanStack Form's API type is heavily generic — `any` here keeps row
// helpers usable without leaking 10 type parameters across the file.
// eslint-disable-next-line @typescript-eslint/no-explicit-any
type FormApi = any
// eslint-disable-next-line @typescript-eslint/no-explicit-any
type FieldApi = any

const toOptions = (values: string[]) =>
  values.map((v) => ({ value: v, label: v }))

export function ClassEntryRow({
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
                <FieldLabel htmlFor={f.name}>Classe</FieldLabel>
                <Combobox
                  id={f.name}
                  options={toOptions(classOptions)}
                  value={f.state.value}
                  onChange={(v: string) => {
                    f.handleChange(v)
                    if (v && onPrimaryClassPicked) onPrimaryClassPicked(v)
                  }}
                  placeholder="Selecionar classe"
                  searchPlaceholder="Buscar classes…"
                  emptyMessage="Nenhuma classe encontrada."
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
                <FieldLabel htmlFor={f.name}>Nível</FieldLabel>
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
          Remover
        </Button>
      </div>
    </div>
  )
}
