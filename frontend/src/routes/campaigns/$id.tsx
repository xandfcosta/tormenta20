import { createFileRoute, Outlet, redirect } from '@tanstack/react-router'
import {
  campaignMembersQueryOptions,
  campaignQueryOptions,
} from '@/entities/campaign/queries'
import { campaignSessionsQueryOptions } from '@/entities/session/queries'
import { meQueryOptions } from '@/entities/user/queries'

/**
 * Layout route for `/campaigns/$id`. Renders only an <Outlet/> so its
 * children — the detail index and the nested session tracker
 * (`/campaigns/$id/sessions/$sid`) — each own the full screen. Without
 * this, the detail page sat here as the component and swallowed the
 * outlet, so navigating to a session changed the URL but not the view.
 * Shared campaign data is prefetched here so both children hydrate warm.
 */
export const Route = createFileRoute('/campaigns/$id')({
  beforeLoad: async ({ context, location }) => {
    const user = await context.queryClient.ensureQueryData(meQueryOptions)
    if (!user)
      throw redirect({ to: '/login', search: { redirect: location.href } })
  },
  loader: ({ context, params }) => {
    const id = Number(params.id)
    return Promise.all([
      context.queryClient.ensureQueryData(campaignQueryOptions(id)),
      context.queryClient.ensureQueryData(campaignSessionsQueryOptions(id)),
      context.queryClient.ensureQueryData(campaignMembersQueryOptions(id)),
    ])
  },
  component: Outlet,
})
