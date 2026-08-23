import { createFileRoute } from '@tanstack/solid-router'
import { entregaAPorta } from './-guards'

type RegisterSearch = { convite?: string }

/**
 * Encaminhamento para a porta (ALE-229). O `?convite=` atravessa intacto porque
 * ele É o fluxo: o link de uso único que o administrador entrega não tem nada
 * para digitar, e um recarregamento tem de continuar valendo (ALE-120).
 *
 * A regra de "sem convite a tela não abre" mudou de lugar, não de dono: era o
 * `beforeLoad` desta rota, agora é o handler que desenha a tela. Ela nunca foi
 * a trava de verdade — o servidor responde 403 para registro sem convite
 * usável, e a isenção do `ADMIN_EMAILS` é por E-MAIL.
 */
export const Route = createFileRoute('/register')({
  validateSearch: (search: Record<string, unknown>): RegisterSearch =>
    typeof search.convite === 'string' ? { convite: search.convite } : {},
  beforeLoad: ({ search }) =>
    entregaAPorta('/piloto/criar-conta', search.convite ? { convite: search.convite } : {}),
  component: () => null,
})
