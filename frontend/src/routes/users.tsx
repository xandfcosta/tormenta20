import { createFileRoute, redirect } from '@tanstack/react-router'
import { meQueryOptions, usersQueryOptions } from '@/entities/user/queries'
import { UsersPage } from '@/pages/users/users-page'

export const Route = createFileRoute('/users')({
  beforeLoad: async ({ context, location }) => {
    const user = await context.queryClient.ensureQueryData(meQueryOptions)
    if (!user) throw redirect({ to: '/login', search: { redirect: location.href } })
  },
  loader: ({ context }) => context.queryClient.ensureQueryData(usersQueryOptions),
  component: UsersPage,
})
