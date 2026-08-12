import { FieldFrame, isInvalid } from './field-frame'
import { Textarea } from './textarea'

export type TextAreaFieldProps = {
  /** Doubles as the textarea's `id` and `name`, so the label associates by `for`. */
  name: string
  label: string
  value: string
  onInput: (value: string) => void
  rows?: number
  /** Validation messages; a non-empty list flips the field to the error style. */
  errors?: string[]
  hint?: string
}

/**
 * `TextField`'s multi-line twin — same frame, same validation grammar, a
 * textarea in the middle. Used for prose fields (a campaign's description).
 *
 * @example
 * <TextAreaField name="descricao" label="Descrição" rows={6}
 *   value={description()} onInput={setDescription} />
 */
export function TextAreaField(props: TextAreaFieldProps) {
  return (
    <FieldFrame name={props.name} label={props.label} errors={props.errors} hint={props.hint}>
      <Textarea
        id={props.name}
        name={props.name}
        rows={props.rows ?? 6}
        value={props.value}
        onInput={(e) => props.onInput(e.currentTarget.value)}
        aria-invalid={isInvalid(props.errors)}
      />
    </FieldFrame>
  )
}
