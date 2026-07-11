import { createFileRoute, redirect } from '@tanstack/react-router'
import { meQueryOptions } from '@/entities/user/queries'
import { HomePage } from '@/pages/home/home-page'

export const Route = createFileRoute('/')({
  // Logged-in users land on their campaign list (the app's home base per
  // the UX model); the marketing hero in HomePage is for anonymous visitors.
  beforeLoad: async ({ context }) => {
    const user = await context.queryClient.ensureQueryData(meQueryOptions)
    if (user) throw redirect({ to: '/campaigns' })
  },
  component: HomePage,
})
