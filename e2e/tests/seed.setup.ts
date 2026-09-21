import { execFileSync } from 'node:child_process'
import { existsSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { expect, test as semear } from '@playwright/test'

/**
 * A SEMENTE DO BANCO DO E2E.
 *
 * A suíte roda contra um banco PRÓPRIO (`data/e2e.db`), apagado e recriado a
 * cada corrida, e não contra o banco de desenvolvimento — um tabuleiro aberto à
 * mão ou um NPC deixado na fila fazem specs sem relação nenhuma com a mudança
 * ficarem vermelhos logo depois de um commit, com CARA DE REGRESSÃO. É a pior
 * forma de um defeito de ambiente aparecer.
 *
 * POR QUE AQUI E NÃO NO `webServer`: a seed é SÓ INSERT (`seed.sql` não tem
 * DDL), então ela precisa de um banco já MIGRADO — e quem migra é a API ao
 * abrir o arquivo. A ordem obrigatória é servidor de pé, depois seed, e o
 * `webServer` do Playwright não tem onde encaixar um passo entre subir e o
 * primeiro teste. É a mesma ordem que o CI já pratica, pelo mesmo motivo (ver
 * o comentário do job no `ci.yml`).
 *
 * Não roda quando o servidor é EXTERNO: ali quem semeia é o CI, e aplicar
 * `seed.sql` duas vezes estoura nas chaves primárias.
 */
const AQUI = dirname(fileURLToPath(import.meta.url))
const ENGINE = resolve(AQUI, '../../engine-go')
const BANCO = resolve(ENGINE, 'data/e2e.db')
const SEED = resolve(ENGINE, 'seed.sql')

semear('semear o banco do e2e', async () => {
  expect(existsSync(BANCO), `a API não criou ${BANCO} — ela subiu com outro DATABASE_URL?`).toBe(
    true,
  )
  expect(existsSync(SEED), `seed.sql não está em ${SEED}`).toBe(true)

  // `.timeout 5000` porque o servidor está de pé segurando a mesma conexão: o
  // SQLite deixa outro escritor entrar, mas ele espera. É a mesma linha do CI.
  execFileSync('sqlite3', [BANCO, '.timeout 5000', `.read ${SEED}`], { stdio: 'pipe' })

  // O CONTROLE, e ele não é zelo: sem ele a suíte inteira falharia depois, no
  // login, com "credenciais inválidas" — que manda procurar defeito na
  // autenticação quando o problema é um banco vazio. Uma linha aqui troca
  // trinta falhas confusas por uma frase.
  const users = execFileSync('sqlite3', [BANCO, 'select count(*) from users'], {
    encoding: 'utf8',
  }).trim()
  expect(
    Number(users),
    'o banco do e2e ficou sem usuários depois da seed — o login de todos os specs falharia',
  ).toBeGreaterThan(0)
})
