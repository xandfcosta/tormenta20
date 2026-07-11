import { createFileRoute, redirect } from '@tanstack/react-router'
import { meQueryOptions } from '@/entities/user/queries'
/**
 * Public invite landing. `join/<token>` is the shareable URL the GM
 * copies from the InviteButton. Two cases:
 *   1. Unauth → redirect to /login with `redirect=/campaigns/join?token=…`
 *      so post-login the user lands on the token-aware join form.
 *   2. Auth → redirect straight to `/campaigns/join?token=…`.
 *
 * All actual validation + campaign preview happens inside
 * `/campaigns/join` — this file is just a router shim. Keeping it thin
 * means the token flow can evolve without a second copy of the form.
 */
export const Route = createFileRoute('/join/$token')({
  beforeLoad: async ({ context, params }) => {
    const user = await context.queryClient.ensureQueryData(meQueryOptions)
    const target = `/campaigns/join?token=${encodeURIComponent(params.token)}`
    if (!user) {
      throw redirect({ to: '/login', search: { redirect: target } })
    }
    throw redirect({
      to: '/campaigns/join',
      search: { token: params.token },
    })
  },
  component: () => null,
})
