import { useMatches } from '@tanstack/solid-router'
import { type StepSlug, isStepSlug } from './wizard-steps'

/**
 * The step being answered, read from the matched route. The URL is the single
 * source: a deep link opens the step it names, the back button walks the
 * wizard, and no signal can drift from the address bar.
 *
 * Falls back to the first step for anything unrecognised — the route guard
 * already redirects those, and a scene must never render a blank stage.
 *
 * @example const current = createCurrentStep(); goTo(stepAt(current(), 1))
 */
export function createCurrentStep(): () => StepSlug {
  const matches = useMatches()
  return () => {
    const params = matches().at(-1)?.params as { step?: string } | undefined
    const slug = params?.step
    return slug && isStepSlug(slug) ? slug : 'raca'
  }
}
