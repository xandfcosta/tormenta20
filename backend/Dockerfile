# syntax=docker/dockerfile:1
#
# API image. BUILD CONTEXT = repo ROOT (the pnpm workspace + t20-data are
# needed), so build/run it via docker-compose or:
#   docker build -f backend/Dockerfile -t t20-api .
#
# Multi-stage: a build stage compiles t20-data + the Nest app + the Prisma
# client, then `pnpm deploy` produces a self-contained /app (workspace deps
# copied as real files, native better-sqlite3 included). Prisma migrations run
# on boot via the entrypoint.

FROM node:22-slim AS build
ENV PNPM_HOME=/pnpm PATH=/pnpm:$PATH
RUN corepack enable && corepack prepare pnpm@10.33.2 --activate
# Native toolchain for better-sqlite3 (compiled during install).
RUN apt-get update \
 && apt-get install -y --no-install-recommends python3 make g++ ca-certificates \
 && rm -rf /var/lib/apt/lists/*
WORKDIR /repo

# Source is copied before install because t20-data's `prepare` script builds it
# (tsc) during `pnpm install`, and better-sqlite3 compiles its native addon then
# too — both need the tree present.
COPY pnpm-workspace.yaml package.json pnpm-lock.yaml ./
COPY t20-data t20-data
COPY backend backend
RUN --mount=type=cache,id=pnpm,target=/pnpm/store \
    pnpm install --frozen-lockfile --filter backend --filter @tormenta20/t20-data

# Generate the Prisma client, compile the API (t20-data already built by prepare).
RUN pnpm --filter @tormenta20/t20-data build \
 && pnpm --filter backend exec prisma generate \
 && pnpm --filter backend build

# Self-contained deploy dir: real @tormenta20/t20-data, native better-sqlite3,
# and the Prisma CLI (kept for `migrate deploy` at runtime).
RUN pnpm --filter backend deploy --legacy /app

FROM node:22-slim AS runtime
WORKDIR /app
COPY --from=build /app /app
COPY backend/docker-entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh && mkdir -p /data
# SQLite lives on a mounted volume; migrations apply against it on boot.
ENV DATABASE_URL=file:/data/app.db
EXPOSE 3000
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
