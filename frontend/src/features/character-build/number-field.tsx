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
            <div className="flex items-stretch gap-2">
              <div className="flex flex-col items-center gap-0.5">
                <NumberInput
                  id={f.name}
                  className="w-20 sm:w-24"
                  min={min}
                  max={max}
                  value={base}
                  onChange={(v) => f.handleChange(v)}
                  onBlur={f.handleBlur}
                  aria-invalid={invalid}
                />
                <span className="text-[10px] uppercase tracking-wide text-muted-foreground">
                  base
                </span>
              </div>
              {raceDelta ? (
                <>
                  <span
                    aria-hidden
                    className="self-center text-sm text-muted-foreground"
                  >
                    =
                  </span>
                  <div className="flex min-w-16 flex-col items-center justify-center rounded-md border border-border bg-card px-3">
                    <span className="font-display text-2xl font-semibold leading-none text-foreground tabular-nums">
                      {base + raceDelta}
                    </span>
                    <span
                      className="mt-0.5 text-[10px] tabular-nums"
                      style={{
                        color:
                          raceDelta >= 0
                            ? 'var(--hp-full)'
                            : 'var(--hp-hurt)',
                      }}
                    >
                      {signed(raceDelta)} raça
                    </span>
                  </div>
                </>
              ) : null}
            </div>
            {invalid && <FieldError errors={f.state.meta.errors} />}
          </Field>
        )
      }}
    </form.Field>
  )
}
