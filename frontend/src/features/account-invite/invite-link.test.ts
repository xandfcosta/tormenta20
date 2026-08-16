import { describe, expect, it } from 'vitest'
import { Route as RegisterRoute } from '@/routes/register'
import { inviteRegisterUrl } from './invite-link'

/**
 * O convite atravessa TRÊS lugares — quem monta o link (o diálogo do Hub e o
 * painel do admin) e quem o lê (a rota `/register`) — e nada amarrava os três.
 * Cada um citava `?convite=` por conta própria: renomear a chave em um deles
 * deixa o admin copiando um link que a tela de registro ignora em silêncio, e o
 * jogador cai num formulário sem convite sem entender por quê.
 *
 * Aqui o link é MONTADO por quem monta e LIDO por quem lê, no mesmo teste.
 */
describe('o link de convite que o admin copia é o que o registro entende', () => {
  // Lê pelo MESMO `validateSearch` da rota: reimplementar a leitura aqui só
  // provaria que o teste concorda consigo mesmo.
  const validar = RegisterRoute.options.validateSearch as unknown as (
    search: Record<string, unknown>,
  ) => { convite?: string }

  const ler = (url: string) =>
    validar(Object.fromEntries(new URL(url, 'http://mesa.local').searchParams))

  it('o token sai inteiro do outro lado', () => {
    const url = inviteRegisterUrl('http://mesa.local', 'tok-abc-123')

    expect(ler(url)).toEqual({ convite: 'tok-abc-123' })
  })

  // Token com caractere especial: sem escapar na montagem, o `+` vira espaço na
  // leitura e o convite morre por um caractere.
  it('token com caractere especial sobrevive à ida e à volta', () => {
    const token = 'a+b/c=d e'

    expect(ler(inviteRegisterUrl('http://mesa.local', token))).toEqual({ convite: token })
  })

  // Sem convite a chave é AUSENTE, não presente-e-indefinida: uma chave
  // obrigatória forçaria todo `<Link to="/register">` a passar `search`.
  it('sem convite, a chave nem existe', () => {
    expect(ler('/register')).toEqual({})
  })
})
