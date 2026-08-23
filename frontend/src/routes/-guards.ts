import { redirect } from '@tanstack/solid-router'
import type { QueryClient } from '@tanstack/solid-query'
import { meQueryOptions } from '@/entities/user/queries'

type GuardArgs = {
  context: { queryClient: QueryClient }
  location: { href: string }
}

/**
 * Entrega o visitante à PORTA, que desde a ALE-229 é renderizada pelo servidor.
 *
 * Não é `redirect()` do TanStack e não pode ser: aquilo navega DENTRO da SPA, e
 * a porta deixou de ser uma rota dela. Sair do bundle exige navegação de
 * verdade.
 *
 * O `Promise` que nunca resolve é deliberado. Sem ele o `beforeLoad` termina, o
 * roteador segue e desenha a tela protegida no quadro entre a atribuição e a
 * troca de documento — um lampejo de tela vazia, ou pior, de tela com dados de
 * ninguém. Parar aqui é dizer "esta navegação não vai terminar", que é a
 * verdade.
 */
export function entregaAPorta(caminho: string, busca: Record<string, string> = {}): Promise<never> {
  const query = new URLSearchParams(busca).toString()
  window.location.href = query ? `${caminho}?${query}` : caminho
  return new Promise<never>(() => {})
}

/**
 * Route guard for the authenticated app: resolves the session once and entrega o
 * visitante anônimo à porta do servidor, lembrando para onde ele ia.
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
  if (!user) await entregaAPorta('/piloto/entrar', { redirect: location.href })
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
