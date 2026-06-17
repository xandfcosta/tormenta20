import { describe, expect, it, vi } from 'vitest'
import { ApiError } from './api'
import { applyServerErrors } from './form-errors'

/**
 * `applyServerErrors` is the bridge between backend ApiError.fieldErrors
 * (dotted-path keys, string[] values) and TanStack Form's setFieldMeta
 * (bracketed-array path keys). Critical because every form submit relies
 * on it for per-field error display.
 */

// applyServerErrors only reads form.setFieldMeta — model that with a
// loose mock typed as the same signature the production code expects.
type SetFieldMeta = (name: unknown, updater: (prev: unknown) => unknown) => void

type FakeForm = {
  setFieldMeta: ReturnType<typeof vi.fn<SetFieldMeta>>
}

function fakeForm(): FakeForm {
  return { setFieldMeta: vi.fn<SetFieldMeta>() }
}

describe('applyServerErrors — ApiError dispatch', () => {
  it('returns false for non-ApiError errors', () => {
    const form = fakeForm()
    expect(applyServerErrors(form, new Error('plain'))).toBe(false)
    expect(form.setFieldMeta).not.toHaveBeenCalled()
  })

  it('returns false for null / undefined / strings', () => {
    const form = fakeForm()
    expect(applyServerErrors(form, null)).toBe(false)
    expect(applyServerErrors(form, undefined)).toBe(false)
    expect(applyServerErrors(form, 'boom')).toBe(false)
  })

  it('returns false when ApiError carries no field errors', () => {
    const form = fakeForm()
    const err = new ApiError(400, 'Validation failed', {})
    expect(applyServerErrors(form, err)).toBe(false)
    expect(form.setFieldMeta).not.toHaveBeenCalled()
  })

  it('returns true and calls setFieldMeta once per field', () => {
    const form = fakeForm()
    const err = new ApiError(400, 'Validation failed', {
      hpCurrent: ['HP current cannot exceed HP max'],
      mpCurrent: ['MP current cannot exceed MP max'],
    })
    expect(applyServerErrors(form, err)).toBe(true)
    expect(form.setFieldMeta).toHaveBeenCalledTimes(2)
  })

  it('passes message objects to the updater', () => {
    const form = fakeForm()
    const err = new ApiError(400, 'Validation failed', {
      name: ['Name is required', 'Pick another'],
    })
    applyServerErrors(form, err)
    const [path, updater] = form.setFieldMeta.mock.calls[0]!
    expect(path).toBe('name')
    const prev = { errors: [], errorMap: {} }
    const next = (updater as (p: typeof prev) => unknown)(prev) as {
      errors: { message: string }[]
      errorMap: { onServer: { message: string }[] }
    }
    expect(next.errors).toEqual([
      { message: 'Name is required' },
      { message: 'Pick another' },
    ])
    expect(next.errorMap.onServer).toEqual(next.errors)
  })

  it('preserves prior errorMap entries when applying onServer errors', () => {
    const form = fakeForm()
    const err = new ApiError(400, 'Validation failed', {
      slots: ['Must be a multiple of 0.5'],
    })
    applyServerErrors(form, err)
    const updater = form.setFieldMeta.mock.calls[0]![1] as (
      p: { errorMap: Record<string, unknown> },
    ) => { errorMap: Record<string, unknown> }
    const next = updater({ errorMap: { onChange: ['existing'] } })
    expect(next.errorMap.onChange).toEqual(['existing'])
    expect(next.errorMap.onServer).toBeDefined()
  })
})

describe('applyServerErrors — path normalization', () => {
  it('rewrites "classes.0.className" to "classes[0].className"', () => {
    const form = fakeForm()
    const err = new ApiError(400, 'Validation failed', {
      'classes.0.className': ['Already added'],
    })
    applyServerErrors(form, err)
    expect(form.setFieldMeta).toHaveBeenCalledWith(
      'classes[0].className',
      expect.any(Function),
    )
  })

  it('rewrites trailing numeric index "items.3"', () => {
    const form = fakeForm()
    const err = new ApiError(400, 'Validation failed', {
      'items.3': ['Bad item'],
    })
    applyServerErrors(form, err)
    expect(form.setFieldMeta).toHaveBeenCalledWith(
      'items[3]',
      expect.any(Function),
    )
  })

  it('leaves non-indexed paths untouched', () => {
    const form = fakeForm()
    const err = new ApiError(400, 'Validation failed', {
      'classChoices.Clérigo.devoto': ['Unknown deus id "x"'],
    })
    applyServerErrors(form, err)
    expect(form.setFieldMeta).toHaveBeenCalledWith(
      'classChoices.Clérigo.devoto',
      expect.any(Function),
    )
  })

  it('rewrites multiple indices in one path', () => {
    const form = fakeForm()
    const err = new ApiError(400, 'Validation failed', {
      'a.0.b.2.c': ['nope'],
    })
    applyServerErrors(form, err)
    expect(form.setFieldMeta).toHaveBeenCalledWith(
      'a[0].b[2].c',
      expect.any(Function),
    )
  })
})
