import { createFileRoute, redirect } from '@tanstack/react-router'
import { z } from 'zod'
import { meQueryOptions } from '@/entities/user/queries'
import { LoginPage } from '@/pages/auth/login-page'

const searchSchema = z.object({ redirect: z.string().optional() })

export const Route = createFileRoute('/login')({
  validateSearch: searchSchema,
  beforeLoad: async ({ context, search }) => {
    const user = await context.queryClient.ensureQueryData(meQueryOptions)
    if (user) throw redirect({ to: search.redirect ?? '/' })
  },
  component: LoginPage,
})
