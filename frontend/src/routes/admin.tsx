import { createFileRoute } from '@tanstack/solid-router'
import { entregaAPorta } from './-guards'

/**
 * A tela de administração saiu da SPA (ALE-242): `/admin` agora é
 * `/piloto/admin`, renderizada pelo servidor.
 *
 * Sem `requireAdmin` aqui, e a ausência é deliberada: a rota do servidor tem o
 * MESMO `requireAdmin`, e ele é a fronteira de verdade. Repetir a checagem no
 * cliente só adiantaria a recusa em uma tela — e para isso ela teria de
 * resolver a sessão antes de sair do bundle, o que é justamente o trabalho que
 * esta virada existe para não fazer.
 */
export const Route = createFileRoute('/admin')({
  beforeLoad: () => entregaAPorta('/piloto/admin'),
  component: () => null,
})
