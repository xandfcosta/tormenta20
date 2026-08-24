import { createFileRoute } from '@tanstack/solid-router'
import { entregaAPorta } from './-guards'

/**
 * A FERRAMENTA VIAJA JUNTO, e é esse o ponto: `/gm/encontros` tem de abrir os
 * encontros do servidor, não a primeira ferramenta. É o mesmo cuidado do `?tab=`
 * da crônica — o slug é o endereço, e perdê-lo quebraria todo link já
 * compartilhado.
 *
 * O slug NÃO é validado aqui. Ele era, porque uma URL torta desenharia palco em
 * branco; agora quem valida é o servidor, e ele já cai na primeira ferramenta
 * quando não reconhece — validar dos dois lados seria a mesma regra em dois
 * lugares, que é o que esta migração passou quatorze fatias tirando.
 */
export const Route = createFileRoute('/gm/$tool')({
  beforeLoad: ({ params }) => entregaAPorta(`/piloto/mestre/${params.tool}`),
  component: () => null,
})
