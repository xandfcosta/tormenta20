import { describe, expect, it } from 'vitest'
import { ApiError, createApiClient } from './api'
import { FakeFetch } from './fake-fetch'

const ALICE = { id: 1, email: 'mestre@t20.local', name: 'Mestre' }

describe('createApiClient', () => {
  it('faz POST em /api/auth/login com credenciais em JSON e cookie de sessão', async () => {
    const http = new FakeFetch([FakeFetch.json(ALICE)])
    const api = createApiClient(http.fetch)

    const user = await api.auth.login({ email: ALICE.email, password: 'segredo' })

    expect(user).toEqual(ALICE)
    const { url, init } = http.onlyCall
    expect(url).toBe('/api/auth/login')
    expect(init?.method).toBe('POST')
    // The Go backend sets an httpOnly session cookie — without this the login
    // "works" and every later request is 401.
    expect(init?.credentials).toBe('include')
    expect(JSON.parse(String(init?.body))).toEqual({
      email: ALICE.email,
      password: 'segredo',
    })
  })

  it('devolve o usuário logado em /auth/me', async () => {
    const http = new FakeFetch([FakeFetch.json(ALICE)])
    const api = createApiClient(http.fetch)

    await expect(api.auth.me()).resolves.toEqual(ALICE)
    expect(http.onlyCall.url).toBe('/api/auth/me')
  })

  it('204 no logout vira undefined, sem tentar parsear JSON', async () => {
    const http = new FakeFetch([FakeFetch.empty()])
    const api = createApiClient(http.fetch)

    await expect(api.auth.logout()).resolves.toBeUndefined()
  })

  it('lança ApiError com status, mensagem e fieldErrors do backend', async () => {
    const http = new FakeFetch([
      FakeFetch.error(401, {
        message: 'Credenciais inválidas',
        fieldErrors: { password: ['Senha incorreta'] },
      }),
    ])
    const api = createApiClient(http.fetch)

    const err = await api.auth.login({ email: ALICE.email, password: 'x' }).catch((e) => e)

    expect(err).toBeInstanceOf(ApiError)
    expect(err.status).toBe(401)
    expect(err.message).toBe('Credenciais inválidas')
    expect(err.fieldErrors).toEqual({ password: ['Senha incorreta'] })
  })

  it('junta a mensagem quando o backend manda uma lista', async () => {
    const http = new FakeFetch([FakeFetch.error(400, { message: ['E-mail inválido', 'Senha curta'] })])
    const api = createApiClient(http.fetch)

    const err = await api.auth.register({ email: 'x', password: 'y' }).catch((e) => e)

    expect(err.message).toBe('E-mail inválido; Senha curta')
  })

  it('cai no status HTTP quando o corpo do erro não é JSON', async () => {
    const http = new FakeFetch([new Response('<html>502</html>', { status: 502 })])
    const api = createApiClient(http.fetch)

    const err = await api.auth.me().catch((e) => e)

    expect(err).toBeInstanceOf(ApiError)
    expect(err.status).toBe(502)
    expect(err.message).toContain('502')
    expect(err.fieldErrors).toEqual({})
  })
})
