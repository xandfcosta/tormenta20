import { createFileRoute } from '@tanstack/solid-router'
import { entregaAPorta } from './-guards'

/**
 * O Hub saiu da SPA (ALE-231): `/` agora é `/piloto/`, renderizada pelo
 * servidor.
 *
 * A rota fica porque `/` é a raiz do app — está no favorito de todo mundo e é
 * para onde metade das cenas volta. Ela virou encaminhamento, como as três da
 * porta (ALE-229).
 *
 * Sem guarda de sessão aqui: quem decide é o `requirePagina` do servidor, que
 * manda o anônimo para a porta lembrando o caminho. Perguntar antes seria uma
 * ida à rede para descobrir o que o próximo passo já descobre.
 */
export const Route = createFileRoute('/')({
  beforeLoad: () => entregaAPorta('/piloto/'),
  component: () => null,
})
