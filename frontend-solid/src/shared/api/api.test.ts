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

  // As mutações de escrita da campanha (ALE-79). O verbo importa tanto quanto o
  // caminho: um PATCH virando POST cria uma segunda campanha em vez de editar.
  it('edita a campanha com PATCH e só os campos mandados', async () => {
    const http = new FakeFetch([FakeFetch.json({ id: 1, name: 'Novo nome' })])
    const api = createApiClient(http.fetch)

    await api.campaigns.update(1, { name: 'Novo nome', description: '' })

    const { url, init } = http.onlyCall
    expect(url).toBe('/api/campaigns/1')
    expect(init?.method).toBe('PATCH')
    expect(JSON.parse(String(init?.body))).toEqual({ name: 'Novo nome', description: '' })
  })

  it('exclui a campanha com DELETE e sem corpo', async () => {
    const http = new FakeFetch([FakeFetch.json({ id: 1 })])
    const api = createApiClient(http.fetch)

    await expect(api.campaigns.delete(1)).resolves.toEqual({ id: 1 })
    const { url, init } = http.onlyCall
    expect(url).toBe('/api/campaigns/1')
    expect(init?.method).toBe('DELETE')
    expect(init?.body).toBeUndefined()
  })

  it('rotaciona o convite com POST e devolve o token novo', async () => {
    const http = new FakeFetch([FakeFetch.json({ campaignId: 1, token: 'abc123' })])
    const api = createApiClient(http.fetch)

    await expect(api.campaigns.rotateInvite(1)).resolves.toEqual({
      campaignId: 1,
      token: 'abc123',
    })
    const { url, init } = http.onlyCall
    expect(url).toBe('/api/campaigns/1/invite')
    expect(init?.method).toBe('POST')
  })

  it('cria a sessão sob a campanha, com o número que o chamador derivou', async () => {
    const http = new FakeFetch([FakeFetch.json({ id: 9, sessionNumber: 3 })])
    const api = createApiClient(http.fetch)

    await api.sessions.create(1, { sessionNumber: 3 })

    const { url, init } = http.onlyCall
    expect(url).toBe('/api/campaigns/1/sessions')
    expect(init?.method).toBe('POST')
    expect(JSON.parse(String(init?.body))).toEqual({ sessionNumber: 3 })
  })

  // O id do DELETE é o do MEMBRO, não o do personagem — trocar os dois remove
  // a pessoa errada da mesa.
  it('remove o membro pelo id de membro', async () => {
    const http = new FakeFetch([FakeFetch.json({ id: 5 })])
    const api = createApiClient(http.fetch)

    await api.members.remove(1, 5)

    const { url, init } = http.onlyCall
    expect(url).toBe('/api/campaigns/1/members/5')
    expect(init?.method).toBe('DELETE')
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
