import { createFileRoute, redirect } from '@tanstack/solid-router'
import { isStepSlug } from '@/features/character-build/wizard-steps'
import { ForgeStep } from '@/pages/characters/forge/forge-step'

export const Route = createFileRoute('/characters/new/$step')({
  // A hand-typed or stale URL lands on the first step instead of an empty
  // stage — the slug is data from outside, so it is checked before it is used.
  beforeLoad: ({ params }) => {
    if (!isStepSlug(params.step)) {
      throw redirect({ to: '/characters/new/$step', params: { step: 'raca' } })
    }
  },
  component: ForgeStep,
})
