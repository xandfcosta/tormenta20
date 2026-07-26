import { Field, FieldError, FieldLabel } from '@/shared/ui/field'
import { NumberInput } from '@/shared/ui/number-input'

// eslint-disable-next-line @typescript-eslint/no-explicit-any
type FormApi = any
// eslint-disable-next-line @typescript-eslint/no-explicit-any
type FieldApi = any

const signed = (n: number) => (n >= 0 ? `+${n}` : `${n}`)

export function NumberField({
  form,
  name,
  label,
  min,
  max,
  raceDelta,
}: {
  form: FormApi
  name: string
  label: string
  min?: number
  max?: number
  /** Racial attribute bonus for this field. Shown as a read-only hint with the
   *  resulting value (`base + delta`); the editable field keeps the base. */
  raceDelta?: number
}) {
  return (
    <form.Field name={name}>
      {(f: FieldApi) => {
        const invalid = f.state.meta.isTouched && !f.state.meta.isValid
        const base = f.state.value as number
        return (
          <Field data-invalid={invalid}>
            <FieldLabel htmlFor={f.name}>{label}</FieldLabel>
            <NumberInput
              id={f.name}
              min={min}
              max={max}
              value={base}
              onChange={(v) => f.handleChange(v)}
              onBlur={f.handleBlur}
              aria-invalid={invalid}
            />
            {raceDelta ? (
              <p className="text-[11px] text-muted-foreground">
                {signed(raceDelta)} raça →{' '}
                <span className="font-semibold text-foreground">
                  {base + raceDelta}
                </span>
              </p>
            ) : null}
            {invalid && <FieldError errors={f.state.meta.errors} />}
          </Field>
        )
      }}
    </form.Field>
  )
}
