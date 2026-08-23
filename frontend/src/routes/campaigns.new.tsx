import { createFileRoute } from '@tanstack/solid-router'
import { entregaAPorta } from './-guards'

/**
 * A folha em branco saiu da SPA (ALE-246): `/campaigns/new` agora é
 * `/piloto/campanhas/nova`, renderizada pelo servidor.
 *
 * Ela é um formulário que NAVEGA ao dar certo, então a versão do servidor é um
 * `<form method="post">` sem sinal nenhum — a mesma decisão da porta (ALE-229).
 *
 * `/campaigns/$id` e a edição da crônica continuam na SPA, e é por isso que o
 * `CampaignForm` e o `campaign-schema` ficam: eles são compartilhados com
 * "editar campanha", e apagá-los agora derrubaria a tela que ainda usa.
 */
export const Route = createFileRoute('/campaigns/new')({
  beforeLoad: () => entregaAPorta('/piloto/campanhas/nova'),
  component: () => null,
})
