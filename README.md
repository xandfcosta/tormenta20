# tormenta20

pnpm workspace monorepo.

```
backend/   NestJS + Prisma 7 (sqlite via @prisma/adapter-better-sqlite3) — LEGADO
engine-go/ Go: API (:3001), motor de regras e o build WASM do mesmo motor
frontend/  Vite + SolidJS + TanStack Router/Query + Kobalte (Tailwind v4, CSS variables)
t20-data/  catálogos e regras puras, compartilhados pelo front e pelo motor
e2e/       Playwright, fora do frontend de propósito
```

O frontend era React até o cutover (ALE-76); a migração para SolidJS está
contada em [MIGRATION.md](MIGRATION.md).

## Setup

```bash
pnpm install
pnpm --filter backend exec prisma migrate dev
```

## Dev

```bash
pnpm dev                 # backend + frontend in parallel
pnpm dev:backend         # NestJS only (port 3000)
pnpm dev:frontend        # Vite only  (port 5173, /api proxied to :3000)
```

## Testes

```bash
pnpm test                # unit: t20-data + backend + frontend (NÃO inclui o e2e)
pnpm test:e2e            # Playwright — exige os dois servers de pé (:5173 e :3001)
```

O e2e ficou FORA do `pnpm test` de propósito (ALE-93): `pnpm -r` roda os pacotes
em paralelo, e disputar CPU com os 1227 vitest do front derruba o Playwright por
timeout — e o job `ci`, que roda `pnpm test`, não instala browser nem sobe a API.
No CI o e2e tem job próprio.

## Build

```bash
pnpm build
```

## Prisma

```bash
pnpm prisma:generate
pnpm prisma:migrate
```

Schema: `backend/prisma/schema.prisma`. Generated client: `backend/src/generated/prisma`. SQLite file: `backend/prisma/dev.db` (path from `DATABASE_URL` in `backend/.env`).
