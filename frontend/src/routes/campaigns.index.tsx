import { createFileRoute } from '@tanstack/solid-router'
import { entregaAPorta } from './-guards'

/**
 * A cena de campanhas saiu da SPA (ALE-234): `/campaigns` agora é
 * `/piloto/campanhas`, renderizada pelo servidor.
 *
 * Só o ÍNDICE. `/campaigns/new`, `/campaigns/join` e `/campaigns/$id`
 * continuam na SPA e continuam sendo o destino dos botões da cena nova.
 *
 * O desvio é do CLIENTE aqui, e não do mux como o da raiz (ALE-231), porque em
 * desenvolvimento a API do Go é montada em `/` — `GET /campaigns` é o endpoint
 * JSON da lista. Interceptar esse caminho no servidor derrubaria a própria API
 * que a SPA usa. O caminho comum não paga por isso: o Hub aponta direto para
 * `/piloto/campanhas`, então quem chega aqui é link antigo ou favorito.
 */
export const Route = createFileRoute('/campaigns/')({
  beforeLoad: () => entregaAPorta('/piloto/campanhas'),
  component: () => null,
})
