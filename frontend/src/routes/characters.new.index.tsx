import { createFileRoute, redirect } from '@tanstack/solid-router'

// /characters/new names no step, so it sends the player to the first one. The
// wizard always lives at an addressable step — there is no stepless Forja.
export const Route = createFileRoute('/characters/new/')({
  beforeLoad: () => {
    throw redirect({ to: '/characters/new/$step', params: { step: 'raca' } })
  },
})
