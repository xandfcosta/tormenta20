/**
 * Named fake for the `fetch` seam (CLAUDE.md: fake classes, not inline stubs).
 * Records every call so a test can assert the URL, method and body the client
 * actually sent, and replies with whatever the test queued.
 *
 * @example
 * const http = new FakeFetch([FakeFetch.json({ id: 1 })])
 * const api = createApiClient(http.fetch)
 */
export class FakeFetch {
  readonly calls: Array<{ url: string; init: RequestInit | undefined }> = []
  private readonly queue: Response[]

  constructor(responses: Response[]) {
    this.queue = [...responses]
  }

  /** A 200 with a JSON body. */
  static json(body: unknown, status = 200): Response {
    return new Response(JSON.stringify(body), {
      status,
      headers: { 'Content-Type': 'application/json' },
    })
  }

  /** An error reply shaped like the backend's (message + fieldErrors). */
  static error(status: number, body: unknown): Response {
    return FakeFetch.json(body, status)
  }

  /** A 204 No Content. */
  static empty(): Response {
    return new Response(null, { status: 204 })
  }

  readonly fetch: typeof globalThis.fetch = async (input, init) => {
    this.calls.push({ url: String(input), init })
    const next = this.queue.shift()
    if (!next) {
      throw new Error(
        `FakeFetch: chamada sem resposta na fila para ${String(input)} — enfileire uma Response por chamada esperada`,
      )
    }
    return next
  }

  /** The single call made, failing loudly when the count isn't exactly one. */
  get onlyCall(): { url: string; init: RequestInit | undefined } {
    if (this.calls.length !== 1) {
      throw new Error(`FakeFetch: esperava 1 chamada, houve ${this.calls.length}`)
    }
    return this.calls[0]
  }
}
