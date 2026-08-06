/**
 * Node engine server — stdlib `node:http`, no framework, so the Go-vs-Node
 * benchmark compares runtime + engine, not Fastify-vs-net/http. Mirror server
 * in engine-go/cmd/server exposes the identical contract.
 *
 *   POST /sheet   body = CharacterInput JSON  → 200 ComputedSheet JSON
 *   GET  /health  → 200 "ok"
 *
 * Run:  node bench/server-node/server.mjs [port]   (default 3001)
 * The engine comes from the built CJS dist so no bundler/tsx is in the path.
 */
import { createServer } from 'node:http'
import { createRequire } from 'node:module'
import { fileURLToPath } from 'node:url'
import { dirname, join } from 'node:path'

const require = createRequire(import.meta.url)
const here = dirname(fileURLToPath(import.meta.url))
const { computeCharacterSheet } = require(
  join(here, '..', '..', 't20-data', 'dist', 'index.js'),
)

const PORT = Number(process.argv[2] ?? 3001)

function readBody(req) {
  return new Promise((resolve, reject) => {
    const chunks = []
    req.on('data', (c) => chunks.push(c))
    req.on('end', () => resolve(Buffer.concat(chunks).toString('utf8')))
    req.on('error', reject)
  })
}

const server = createServer(async (req, res) => {
  if (req.method === 'GET' && req.url === '/health') {
    res.writeHead(200, { 'content-type': 'text/plain' })
    res.end('ok')
    return
  }
  if (req.method === 'POST' && req.url === '/sheet') {
    try {
      const input = JSON.parse(await readBody(req))
      const sheet = computeCharacterSheet(input)
      const body = JSON.stringify(sheet)
      res.writeHead(200, { 'content-type': 'application/json' })
      res.end(body)
    } catch (err) {
      res.writeHead(400, { 'content-type': 'application/json' })
      res.end(JSON.stringify({ error: String(err?.message ?? err) }))
    }
    return
  }
  res.writeHead(404, { 'content-type': 'text/plain' })
  res.end('not found')
})

server.listen(PORT, () => {
  // eslint-disable-next-line no-console -- CLI banner, plain text by design
  console.log(`node engine server on :${PORT}`)
})
