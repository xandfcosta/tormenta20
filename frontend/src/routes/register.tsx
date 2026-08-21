import { createFileRoute, redirect } from '@tanstack/solid-router'
import { meQueryOptions } from '@/entities/user/queries'
import { RegisterPage } from '@/pages/auth/register-page'

type RegisterSearch = { convite?: string }

/**
 * `?convite=` carrega o link de uso único que o administrador entregou
 * (ALE-120), e SEM ele esta rota nem abre: quem chega a `/register` de mão
 * vazia volta para o login, onde a frase explica que a mesa é por convite.
 *
 * A porta já era fechada — o servidor responde 403 para registro sem convite
 * usável, e o convite vale exatamente uma vez —, mas a tela ficava ABERTA e
 * parecia um cadastro comum. Fechar a rota faz a UI dizer a mesma coisa que o
 * servidor faz.
 *
 * **O caminho do dono numa instalação nova**: `/register?convite=dono`. O token
 * pode ser qualquer coisa porque ele não é lido para um endereço de
 * `ADMIN_EMAILS` — a isenção é por E-MAIL e não por "instalação vazia", de
 * propósito, senão "o primeiro que registrar ganha a coroa". Quem decide
 * continua sendo o servidor; a rota só deixa de convidar estranhos a tentar.
 */
export const Route = createFileRoute('/register')({
  // The key is ABSENT when there is no invite, not present-and-undefined: a
  // required key would force every <Link to="/register"> to pass `search`.
  validateSearch: (search: Record<string, unknown>): RegisterSearch =>
    typeof search.convite === 'string' ? { convite: search.convite } : {},
  beforeLoad: async ({ context, search }) => {
    const user = await context.queryClient.ensureQueryData(meQueryOptions)
    if (user) throw redirect({ to: '/' })
    if (!search.convite) throw redirect({ to: '/login' })
  },
  component: RegisterPage,
})
