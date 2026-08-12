import { createFileRoute, redirect } from '@tanstack/solid-router'
import { requireSession } from './-guards'

/**
 * The shareable invite URL — this is what `InviteDialog` mints and the GM sends
 * (ALE-79). A router shim, nothing else: it hands the token to
 * `/campaigns/join`, where the preview and the form live, so the join flow
 * exists in exactly one place.
 *
 * `requireSession` runs first, so an anonymous visitor goes to /login
 * remembering THIS url — after logging in they land back here and continue
 * into the form, invite intact.
 */
export const Route = createFileRoute('/join/$token')({
  beforeLoad: async (args) => {
    await requireSession(args)
    throw redirect({ to: '/campaigns/join', search: { token: args.params.token } })
  },
  component: () => null,
})
