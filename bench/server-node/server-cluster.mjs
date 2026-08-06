/**
 * Clustered Node engine server — forks one worker per CPU so Node uses all
 * cores, the fair counterpart to Go's multicore net/http. Same POST /sheet
 * contract as server.mjs; the OS load-balances connections across workers
 * (cluster shared-socket). Run: node server-cluster.mjs [port] [workers].
 */
import cluster from 'node:cluster'
import { availableParallelism } from 'node:os'
import { createServer } from 'node:http'
import { createRequire } from 'node:module'
import { fileURLToPath } from 'node:url'
import { dirname, join } from 'node:path'

const PORT = Number(process.argv[2] ?? 3004)
const WORKERS = Number(process.argv[3] ?? availableParallelism())

if (cluster.isPrimary) {
  for (let i = 0; i < WORKERS; i++) cluster.fork()
  // eslint-disable-next-line no-console -- CLI banner
  console.log(`node cluster on :${PORT} — ${WORKERS} workers`)
} else {
  const require = createRequire(import.meta.url)
  const here = dirname(fileURLToPath(import.meta.url))
  const { computeCharacterSheet } = require(
    join(here, '..', '..', 't20-data', 'dist', 'index.js'),
  )
  const readBody = (req) =>
    new Promise((resolve, reject) => {
      const chunks = []
      req.on('data', (c) => chunks.push(c))
      req.on('end', () => resolve(Buffer.concat(chunks).toString('utf8')))
      req.on('error', reject)
    })
  createServer(async (req, res) => {
    if (req.method === 'GET' && req.url === '/health') {
      res.writeHead(200, { 'content-type': 'text/plain' })
      res.end('ok')
      return
    }
    if (req.method === 'POST' && req.url === '/sheet') {
      try {
        const sheet = computeCharacterSheet(JSON.parse(await readBody(req)))
        res.writeHead(200, { 'content-type': 'application/json' })
        res.end(JSON.stringify(sheet))
      } catch (err) {
        res.writeHead(400, { 'content-type': 'application/json' })
        res.end(JSON.stringify({ error: String(err?.message ?? err) }))
      }
      return
    }
    res.writeHead(404)
    res.end('not found')
  }).listen(PORT)
}
