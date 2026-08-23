import { QueryClient } from '@tanstack/solid-query'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { meQueryOptions } from '@/entities/user/queries'
import type { AuthUser } from '@/shared/api/api'
import { requireAdmin, requireSession } from './-guards'

/**
 * Os dois guardas decidem 12 rotas e nunca tinham executado sob teste. O desfecho
 * que se nota: abrir um link direto de uma cena, cair no login e — depois de
 * entrar — parar na home em vez do lugar para onde se ia.
 */
function guardArgs(user: AuthUser | null, href = '/campaigns/7') {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  queryClient.setQueryData(meQueryOptions.queryKey, user)
  return { context: { queryClient }, location: { href } }
}

function makeUser(overrides: Partial<AuthUser> = {}): AuthUser {
  return { id: 1, email: 'a@t20.local', isAdmin: false, ...overrides } as AuthUser
}

/**
 * O redirect do TanStack é LANÇADO, não devolvido, e o destino mora em
 * `.options` — ler o objeto cru devolveria `undefined` e o teste passaria por
 * engano em cima de qualquer coisa lançada.
 */
async function caught(run: () => Promise<void>): Promise<{ to?: string; search?: unknown }> {
  try {
    await run()
  } catch (thrown) {
    const options = (thrown as { options?: { to?: string; search?: unknown } }).options
    if (!options) throw new Error(`lançou algo que não é redirect: ${String(thrown)}`)
    return options
  }
  throw new Error('nada foi lançado: o guarda deixou passar')
}

/**
 * Desde a ALE-229 o anônimo não é redirecionado DENTRO da SPA: a porta virou
 * página do servidor, e sair do bundle é uma navegação de verdade
 * (`window.location.href`). O guarda devolve uma promessa que nunca resolve, e
 * essa é a garantia — se ela resolvesse, o roteador seguiria e desenharia a
 * tela protegida no quadro entre a atribuição e a troca de documento.
 *
 * Por isso o teste NÃO espera a promessa: esperar é o modo de falha.
 */
async function paraOndeFoi(run: () => Promise<void>): Promise<string> {
  const ida = vi.fn()
  vi.spyOn(window, 'location', 'get').mockReturnValue({
    set href(url: string) {
      ida(url)
    },
  } as unknown as Location)

  // Assentou = resolveu OU rejeitou. As duas contam: com a promessa resolvida o
  // roteador segue e desenha; com ela rejeitada o TanStack trata como redirect
  // interno e some com a navegação para fora. A primeira versão deste
  // verificador só olhava o `then`, e por isso o `requireAdmin` passava verde
  // sabotado — ele rejeita depois de o `requireSession` devolver.
  let assentou = ''
  void run().then(
    () => {
      assentou = 'resolveu'
    },
    (e) => {
      assentou = `rejeitou (${String(e)})`
    },
  )
  await new Promise((r) => setTimeout(r, 0))

  expect(assentou, 'o guarda assentou: a navegação para fora da SPA não é o que aconteceu').toBe(
    '',
  )
  expect(ida, 'o guarda não navegou para lugar nenhum').toHaveBeenCalledTimes(1)
  return ida.mock.calls[0]?.[0] as string
}

afterEach(() => {
  vi.restoreAllMocks()
})

describe('requireSession', () => {
  it('deixa passar quem tem sessão', async () => {
    await expect(requireSession(guardArgs(makeUser()))).resolves.toBeUndefined()
  })

  // O destino tem de viajar junto: sem ele, entrar joga na home e quem clicou o
  // link da mesa perde o caminho.
  it('entrega o anônimo à porta LEMBRANDO para onde ia', async () => {
    const url = await paraOndeFoi(() =>
      requireSession(guardArgs(null, '/campaigns/7/sessions/4')),
    )
    expect(url).toBe('/piloto/entrar?redirect=%2Fcampaigns%2F7%2Fsessions%2F4')
  })
})

describe('requireAdmin', () => {
  it('deixa passar o admin', async () => {
    await expect(requireAdmin(guardArgs(makeUser({ isAdmin: true })))).resolves.toBeUndefined()
  })

  it('manda o usuário comum para a home, não para o login', async () => {
    const redirect = await caught(() => requireAdmin(guardArgs(makeUser({ isAdmin: false }))))
    expect(redirect.to).toBe('/')
  })

  // Anônimo vai para a PORTA, não para a home: quem não entrou ainda tem para
  // onde ir, e mandá-lo à home o faria bater na mesma parede de novo.
  it('entrega o anônimo à porta, não à home', async () => {
    expect(await paraOndeFoi(() => requireAdmin(guardArgs(null)))).toContain('/piloto/entrar')
  })
})
