import { createFileRoute, redirect } from '@tanstack/solid-router'
import { meQueryOptions } from '@/entities/user/queries'
import { RegisterPage } from '@/pages/auth/register-page'

type RegisterSearch = { convite?: string }

/**
 * `?convite=` carries the single-use link the admin handed out (ALE-120). It is
 * optional on purpose: the ADMIN_EMAILS address has no invite to hold when it
 * creates the first account on a fresh machine, so the form always submits and
 * the SERVER decides — the UI never guesses who is allowed.
 */
export const Route = createFileRoute('/register')({
  // The key is ABSENT when there is no invite, not present-and-undefined: a
  // required key would force every <Link to="/register"> to pass `search`.
  validateSearch: (search: Record<string, unknown>): RegisterSearch =>
    typeof search.convite === 'string' ? { convite: search.convite } : {},
  beforeLoad: async ({ context }) => {
    const user = await context.queryClient.ensureQueryData(meQueryOptions)
    if (user) throw redirect({ to: '/' })
  },
  component: RegisterPage,
})
