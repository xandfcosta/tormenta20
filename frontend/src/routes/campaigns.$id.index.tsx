import { createFileRoute } from '@tanstack/solid-router'
import { entregaAPorta } from './-guards'

type BuscaDaCronica = { tab?: string }

/**
 * A crônica saiu da SPA (ALE-255): `/campaigns/$id` agora é
 * `/piloto/campanhas/{id}`, renderizada pelo servidor.
 *
 * O `?tab=` viaja junto, e isso é o ponto: ele já era ENDEREÇO nesta tela — a
 * versão em React precisava espelhá-lo num `useState` com dois efeitos e um
 * debounce de 250ms para a troca não travar. No servidor o parâmetro é o
 * estado, então perder o `tab` aqui quebraria todo link de seção compartilhado.
 *
 * `/campaigns/$id/sessions/$sid` continua na SPA: a mesa ao vivo é a próxima
 * fatia, e é ela que os botões desta cena continuam abrindo.
 */
export const Route = createFileRoute('/campaigns/$id/')({
  validateSearch: (busca: Record<string, unknown>): BuscaDaCronica => ({
    tab: typeof busca.tab === 'string' ? busca.tab : undefined,
  }),
  beforeLoad: ({ params, search }) =>
    entregaAPorta(`/piloto/campanhas/${params.id}`, search.tab ? { tab: search.tab } : {}),
  component: () => null,
})
