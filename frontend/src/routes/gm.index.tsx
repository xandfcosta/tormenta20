import { createFileRoute } from '@tanstack/solid-router'
import { entregaAPorta } from './-guards'

/** `/gm/` sozinho abria a primeira ferramenta; agora encaminha para ela. */
export const Route = createFileRoute('/gm/')({
  beforeLoad: () => entregaAPorta('/piloto/mestre/bestiario'),
  component: () => null,
})
