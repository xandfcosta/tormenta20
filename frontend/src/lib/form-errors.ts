import { ApiError } from './api'

/**
 * Maps backend ApiError.fieldErrors (dotted paths -> string[]) onto a
 * TanStack Form instance via setFieldMeta. Returns true if any field
 * error was applied — caller can use that to decide whether to surface
 * a top-level form error.
 *
 * `form` is typed loosely because TanStack Form's setFieldMeta is
 * generic over a literal union of field paths that varies per form;
 * we deliberately bypass that since server paths are dynamic.
 */
export function applyServerErrors(
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  form: { setFieldMeta: (name: any, updater: (prev: any) => any) => void },
  error: unknown,
): boolean {
  if (!(error instanceof ApiError)) return false
  const entries = Object.entries(error.fieldErrors)
  if (!entries.length) return false
  for (const [path, msgs] of entries) {
    const errors = msgs.map((message) => ({ message }))
    form.setFieldMeta(normalizePath(path), (prev) => ({
      ...prev,
      errors,
      errorMap: { ...(prev?.errorMap ?? {}), onServer: errors },
    }))
  }
  return true
}

// backend returns "classes.0.className" — TanStack Form keys are "classes[0].className"
function normalizePath(path: string): string {
  return path.replace(/\.(\d+)(?=\.|$)/g, '[$1]')
}
