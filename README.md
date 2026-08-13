# tormenta20

pnpm workspace monorepo.

```
engine-go/ Go: API (:3001), motor de regras, e o mesmo motor compilado pra WASM
frontend/  Vite + SolidJS + TanStack Router/Query + Kobalte (Tailwind v4, CSS variables)
t20-data/  catálogos e regras puras, compartilhados pelo front e pelo motor
e2e/       Playwright, fora do frontend de propósito
```

O frontend era React até o cutover (ALE-76); a migração para SolidJS está
contada em [MIGRATION.md](MIGRATION.md). O backend NestJS foi removido — o Go é
o backend, e migra o próprio banco (goose embutido) ao subir.

## Setup

```bash
pnpm install
```

Não há passo de migração: `db.Open` aplica as migrações embutidas na partida.

## Dev

```bash
pnpm dev                 # API Go (:3001) + Vite (:5173) juntos
pnpm dev:frontend        # só o Vite (/api e /socket.io proxiados pra :3001)
```

O `predev` do frontend compila `engine-go` → WASM antes de subir o Vite, porque
o app deriva a ficha pelo MESMO motor do servidor. Precisa do toolchain Go.

## Testes

```bash
pnpm test                # unit: t20-data + frontend (NÃO inclui o e2e)
pnpm test:e2e            # Playwright — exige os dois servers de pé (:5173 e :3001)
cd engine-go && go test ./...
```

O e2e ficou FORA do `pnpm test` de propósito (ALE-93): `pnpm -r` roda os pacotes
em paralelo, e disputar CPU com os vitest do front derruba o Playwright por
timeout — e o job `ci`, que roda `pnpm test`, não instala browser nem sobe a API.
No CI o e2e tem job próprio.

## Build e produção

```bash
pnpm build               # compila o WASM, o t20-data e o SPA
pnpm start               # build do front + UM binário servindo tudo
```

**A stack de produção é um processo só.** Com `STATIC_DIR` apontando para
`frontend/dist`, o `cmd/api` serve o SPA (com fallback pras rotas de cliente), os
assets, `/api/*` e o `/socket.io/` na mesma porta. Não há nginx, não há
docker-compose e não há segundo runtime pra manter — foi por isso que os dois
saíram quando o Nest saiu.

Variáveis (defaults em `engine-go/api/config.go`):

| var | default | o que faz |
|---|---|---|
| `PORT` | `3001` | porta do servidor |
| `DATABASE_URL` | `file:./t20-go.db` | arquivo SQLite; migra sozinho ao abrir |
| `JWT_SECRET` | — | **troque**: assina os JWT de sessão |
| `COOKIE_SECURE` | `false` | ligue quando houver TLS na frente |
| `STATIC_DIR` | vazio | o `dist` do front; vazio = modo dev (o Vite serve) |
| `CATALOG_PATH` | `parity/_catalogs.json` | catálogos dos validadores de mutação |
