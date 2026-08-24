import { Outlet, createFileRoute } from '@tanstack/solid-router'

/**
 * A Mesa do Mestre saiu da SPA (ALE-264): as quatro ferramentas são cenas do
 * servidor em `/piloto/mestre/{ferramenta}`. As rotas ficam como
 * ENCAMINHAMENTO, porque `/gm` é o endereço que os mestres têm salvo.
 *
 * ESTA rota não encaminha, e a razão custou uma medição: o `beforeLoad` do pai
 * roda ANTES do filho, então encaminhar aqui atropelava a `$tool` e
 * `/gm/encontros` abria o BESTIÁRIO. Quem encaminha são as filhas — a `index`
 * para a primeira ferramenta, a `$tool` levando o slug junto.
 *
 * Ela continua sendo um layout com Outlet pelo motivo de sempre: `/gm` tem
 * filhas, e uma tela aqui engoliria a saída ([[reference_tanstack_nested_routes]]).
 */
export const Route = createFileRoute('/gm')({
  component: () => <Outlet />,
})
