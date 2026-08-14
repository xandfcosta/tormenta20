import { queryOptions } from '@tanstack/solid-query'
import { api } from '@/shared/api/api'

/**
 * As leituras da tela de administração (ALE-120). Todas atrás de `requireAdmin`
 * no servidor — o que a UI decide é o que MOSTRAR, nunca quem pode.
 *
 * A chave começa com `admin` para que uma ação que muda o mundo (apagar conta,
 * gerar convite) invalide as quatro de uma vez sem enumerá-las.
 */
export const adminUsersQueryOptions = queryOptions({
  queryKey: ['admin', 'users'] as const,
  queryFn: api.admin.users,
})

export const adminInvitesQueryOptions = queryOptions({
  queryKey: ['admin', 'invites'] as const,
  queryFn: api.admin.invites,
})

export const adminStatusQueryOptions = queryOptions({
  queryKey: ['admin', 'status'] as const,
  queryFn: api.admin.status,
})

export const adminBackupsQueryOptions = queryOptions({
  queryKey: ['admin', 'backups'] as const,
  queryFn: api.admin.backups,
})
