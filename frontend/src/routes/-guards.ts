import { redirect } from '@tanstack/solid-router'
import type { QueryClient } from '@tanstack/solid-query'
import { meQueryOptions } from '@/entities/user/queries'

type GuardArgs = {
  context: { queryClient: QueryClient }
  location: { href: string }
}

/**
 * Route guard for the authenticated app: resolves the session once and sends
 * anonymous visitors to /login, remembering where they were headed.
 *
 * Shared so every scene guards the same way — the React app repeated this
 * `ensureQueryData` + redirect block in each route file, and a scene that
 * forgot it was silently public.
 *
 * The `-guards` filename prefix keeps TanStack's file-route generator from
 * treating this module as a route.
 *
 * @example export const Route = createFileRoute('/gm')({ beforeLoad: requireSession, ... })
 */
export async function requireSession({ context, location }: GuardArgs): Promise<void> {
  const user = await context.queryClient.ensureQueryData(meQueryOptions)
  if (!user) throw redirect({ to: '/login', search: { redirect: location.href } })
}

/**
 * Guarda das telas de administração: exige sessão E o papel.
 *
 * O `isAdmin` vem do servidor a cada `/auth/me` e é derivado do `ADMIN_EMAILS`
 * — não há coluna no banco para ficar velha (ALE-120). Isto é UX, não
 * segurança: quem chegar aqui na mão leva 403 de todo endpoint da tela, que é
 * onde a fronteira mora de verdade.
 *
 * @example export const Route = createFileRoute('/admin')({ beforeLoad: requireAdmin, ... })
 */
export async function requireAdmin(args: GuardArgs): Promise<void> {
  await requireSession(args)
  const user = await args.context.queryClient.ensureQueryData(meQueryOptions)
  if (!user?.isAdmin) throw redirect({ to: '/' })
}
