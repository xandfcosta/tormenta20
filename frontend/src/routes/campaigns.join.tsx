import { createFileRoute } from '@tanstack/solid-router'
import { entregaAPorta } from './-guards'

type BuscaDoConvite = { token?: string }

/**
 * A carta de convite saiu da SPA (ALE-249): `/campaigns/join` agora é
 * `/piloto/campanhas/entrar`, renderizada pelo servidor.
 *
 * O `token` viaja junto, e isso não é detalhe: o `/join/$token` — que é a URL
 * que o mestre ENVIA — encaminha para cá, então perder o token aqui quebraria
 * todo link de convite já compartilhado.
 *
 * A cena nova resolve o convite no SERVIDOR, então o nome da mesa já vem na
 * primeira resposta. Some o estado de "carregando" que existia aqui — não
 * porque foi escondido, porque não existe mais.
 */
export const Route = createFileRoute('/campaigns/join')({
  validateSearch: (busca: Record<string, unknown>): BuscaDoConvite => ({
    token: typeof busca.token === 'string' ? busca.token : undefined,
  }),
  beforeLoad: ({ search }) =>
    entregaAPorta('/piloto/campanhas/entrar', search.token ? { token: search.token } : {}),
  component: () => null,
})
