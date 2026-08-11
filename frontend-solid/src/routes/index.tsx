import { createFileRoute, redirect } from '@tanstack/solid-router'
import { meQueryOptions } from '@/entities/user/queries'
import { FoundationHub } from '@/pages/home/foundation-hub'

export const Route = createFileRoute('/')({
  beforeLoad: async ({ context, location }) => {
    const user = await context.queryClient.ensureQueryData(meQueryOptions)
    if (!user) throw redirect({ to: '/login', search: { redirect: location.href } })
  },
  component: FoundationHub,
})
