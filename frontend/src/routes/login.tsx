import { createFileRoute } from '@tanstack/solid-router'
import { z } from 'zod'
import { entregaAPorta } from './-guards'

const searchSchema = z.object({ redirect: z.string().optional() })

/**
 * A porta saiu da SPA (ALE-229): quem é `/login` agora é `/piloto/entrar`,
 * renderizada pelo servidor.
 *
 * A rota fica porque o CAMINHO é público — está em links, em favoritos e no
 * hábito de quem usa a mesa. Ela virou o que sobrou dela: um encaminhamento
 * que preserva o `?redirect=`.
 *
 * Sem `beforeLoad` perguntando pela sessão: quem decide se já há login é a
 * própria porta, e perguntar aqui seria uma ida à rede para descobrir algo que
 * o servidor já sabe ao desenhar.
 */
export const Route = createFileRoute('/login')({
  validateSearch: searchSchema,
  beforeLoad: ({ search }) =>
    entregaAPorta('/piloto/entrar', search.redirect ? { redirect: search.redirect } : {}),
  component: () => null,
})
