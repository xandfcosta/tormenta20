import { createFileRoute, redirect } from '@tanstack/react-router'
import { meQueryOptions } from '@/entities/user/queries'
import { RegisterPage } from '@/pages/auth/register-page'

export const Route = createFileRoute('/register')({
  beforeLoad: async ({ context }) => {
    const user = await context.queryClient.ensureQueryData(meQueryOptions)
    if (user) throw redirect({ to: '/' })
  },
  component: RegisterPage,
})
