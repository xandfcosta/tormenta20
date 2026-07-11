import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { io } from 'socket.io-client'
import { connectSession } from './realtime'

/**
 * Realtime module unit tests. The hook itself is more useful with
 * component tests (Testing Library), but here we cover the factory
 * that wires socket.io-client — asserting the origin, credentials,
 * auto-connect flag, and token override behavior.
 */

vi.mock('socket.io-client', () => ({
  io: vi.fn(),
}))

const mockedIo = io as unknown as ReturnType<typeof vi.fn>

beforeEach(() => {
  mockedIo.mockReset()
  mockedIo.mockReturnValue({ id: 'fake-socket' })
})

afterEach(() => {
  vi.restoreAllMocks()
})

describe('connectSession', () => {
  it('opens against window.location.origin with cookie credentials', () => {
    connectSession()
    expect(mockedIo).toHaveBeenCalledTimes(1)
    const [origin, options] = mockedIo.mock.calls[0]!
    expect(origin).toBe(window.location.origin)
    expect(options).toMatchObject({
      withCredentials: true,
      autoConnect: false,
    })
    expect(options.transports).toEqual(['websocket', 'polling'])
  })

  it('forwards the auth token when provided', () => {
    connectSession('jwt-abc')
    const [, options] = mockedIo.mock.calls[0]!
    expect(options.auth).toEqual({ token: 'jwt-abc' })
  })

  it('omits auth when no token is passed', () => {
    connectSession()
    const [, options] = mockedIo.mock.calls[0]!
    expect(options.auth).toBeUndefined()
  })
})
