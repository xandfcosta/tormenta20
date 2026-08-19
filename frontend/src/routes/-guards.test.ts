import { QueryClient } from '@tanstack/solid-query'
import { describe, expect, it } from 'vitest'
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

describe('requireSession', () => {
  it('deixa passar quem tem sessão', async () => {
    await expect(requireSession(guardArgs(makeUser()))).resolves.toBeUndefined()
  })

  // O destino tem de viajar junto: sem ele, entrar joga na home e quem clicou o
  // link da mesa perde o caminho.
  it('manda o anônimo para o login LEMBRANDO para onde ia', async () => {
    const redirect = await caught(() => requireSession(guardArgs(null, '/campaigns/7/sessions/4')))
    expect(redirect.to).toBe('/login')
    expect(redirect.search).toEqual({ redirect: '/campaigns/7/sessions/4' })
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

  // Anônimo cai no login, não na home: quem não entrou ainda tem para onde ir.
  it('manda o anônimo para o login', async () => {
    const redirect = await caught(() => requireAdmin(guardArgs(null)))
    expect(redirect.to).toBe('/login')
  })
})
