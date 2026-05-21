# tormenta20

pnpm workspace monorepo.

```
backend/   NestJS + Prisma 7 (sqlite via @prisma/adapter-better-sqlite3)
frontend/  Vite + React + TanStack Router/Query + Zustand + Shadcn (Tailwind v4, CSS variables)
```

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
