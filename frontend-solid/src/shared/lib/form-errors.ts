import { ApiError } from '@/shared/api/api'

export type FieldErrors = Record<string, string[]>

export type SubmitFailure = {
  /** Per-field messages the backend attributed to specific inputs. */
  fieldErrors: FieldErrors
  /** A single message for the whole form, when no field owns the problem. */
  formError: string | null
}

/**
 * Turns a rejected submit into something a form can render: per-field messages
 * when the backend said which field is wrong, otherwise one form-level message.
 *
 * A non-`ApiError` (network down, a bug) never reaches the user raw — it
 * becomes a generic line, because "Failed to fetch" is not an answer.
 *
 * @example setFieldErrors(toSubmitFailure(err).fieldErrors)
 */
export function toSubmitFailure(error: unknown): SubmitFailure {
  if (!(error instanceof ApiError)) {
    return { fieldErrors: {}, formError: 'Erro inesperado. Tente novamente.' }
  }
  const hasFieldErrors = Object.keys(error.fieldErrors).length > 0
  return {
    fieldErrors: error.fieldErrors,
    formError: hasFieldErrors ? null : error.message,
  }
}
