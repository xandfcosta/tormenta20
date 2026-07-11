import { Field, FieldError, FieldLabel } from '@/shared/ui/field'
import { NumberInput } from '@/shared/ui/number-input'

// eslint-disable-next-line @typescript-eslint/no-explicit-any
type FormApi = any
// eslint-disable-next-line @typescript-eslint/no-explicit-any
type FieldApi = any

export function NumberField({
  form,
  name,
  label,
  min,
  max,
}: {
  form: FormApi
  name: string
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
