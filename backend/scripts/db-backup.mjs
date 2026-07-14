// Consistent online backup of the SQLite database. Uses `VACUUM INTO`, which
// takes a read snapshot without blocking the running API and is safe regardless
// of journal mode (WAL/DELETE) — unlike a raw `cp` that can miss the -wal file.
//
// Run from the repo root: `pnpm db:backup` (executes in the backend workspace).
// Source: DATABASE_URL (file:) or ../data/app.db. Output: ../backups/app-<ts>.db
import Database from 'better-sqlite3'
import { mkdirSync } from 'node:fs'
import { resolve } from 'node:path'

const repoRoot = resolve(process.cwd(), '..')

function sourcePath() {
  const url = process.env.DATABASE_URL
  if (url?.startsWith('file:')) return resolve(process.cwd(), url.slice('file:'.length))
  return resolve(repoRoot, 'data/app.db')
}

const src = sourcePath()
const backupsDir = resolve(repoRoot, 'backups')
mkdirSync(backupsDir, { recursive: true })

const stamp = new Date().toISOString().replace(/[:.]/g, '-').slice(0, 19)
const out = resolve(backupsDir, `app-${stamp}.db`)

const db = new Database(src, { fileMustExist: true })
db.exec(`VACUUM INTO '${out.replace(/'/g, "''")}'`)
db.close()

console.log(`✔ backup written: ${out}`)
