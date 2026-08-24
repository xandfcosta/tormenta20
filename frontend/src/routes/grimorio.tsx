import { createFileRoute } from '@tanstack/solid-router'
import { entregaAPorta } from './-guards'

/**
 * A folha de especificação saiu da SPA (ALE-251): `/grimorio` agora é
 * `/piloto/grimorio`, renderizada pelo servidor.
 *
 * Ela mede o que desenha, e isso exigiu a primeira ilha de JS da migração que é
 * NECESSÁRIA e não conveniente — `getComputedStyle` e canvas não existem no
 * servidor. E ela ganhou uma coluna a mais: cada peça aparece nas DUAS versões,
 * medida, para a divergência entre os stacks saltar aos olhos em vez de
 * depender de alguém lembrar de comparar.
 */
export const Route = createFileRoute('/grimorio')({
  beforeLoad: () => entregaAPorta('/piloto/grimorio'),
  component: () => null,
})
