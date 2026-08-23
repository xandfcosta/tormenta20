import { createFileRoute } from '@tanstack/solid-router'
import { entregaAPorta } from './-guards'

type ResetSearch = { token?: string }

/**
 * Encaminhamento para a porta (ALE-229). O caminho é o que vai escrito no link
 * que o administrador manda, então ele não pode sumir — quem clicar num link
 * antigo tem de chegar à tela nova.
 */
export const Route = createFileRoute('/redefinir-senha')({
  validateSearch: (search: Record<string, unknown>): ResetSearch =>
    typeof search.token === 'string' ? { token: search.token } : {},
  beforeLoad: ({ search }) =>
    entregaAPorta('/piloto/redefinir-senha', search.token ? { token: search.token } : {}),
  component: () => null,
})
