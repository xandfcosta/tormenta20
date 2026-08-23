import { createFileRoute } from '@tanstack/solid-router'
import { entregaAPorta } from './-guards'

/**
 * A cena de personagens saiu da SPA (ALE-239): `/characters` agora é
 * `/piloto/personagens`, renderizada pelo servidor.
 *
 * Só o ÍNDICE. `/characters/new` (a Forja) e `/characters/$id` (a ficha)
 * continuam na SPA e continuam sendo o destino dos botões da cena nova.
 *
 * O desvio é do CLIENTE, e não do mux como o da raiz (ALE-231), pela mesma
 * razão da virada de campanhas (ALE-234): em desenvolvimento a API do Go é
 * montada em `/`, e `GET /characters` É o endpoint JSON do elenco —
 * interceptar esse caminho no servidor derrubaria a própria API que a ficha
 * consome. Quem cai aqui é link antigo ou favorito; o Hub aponta direto.
 */
export const Route = createFileRoute('/characters/')({
  beforeLoad: () => entregaAPorta('/piloto/personagens'),
  component: () => null,
})
