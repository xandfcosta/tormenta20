# syntax=docker/dockerfile:1
#
# Frontend image. BUILD CONTEXT = repo ROOT (the Vite build bundles t20-data
# source). Compiles the SPA (emitting .br precompressed assets), then serves it
# from a brotli-enabled nginx that also reverse-proxies the API + socket.io.

FROM node:22-slim AS build
ENV PNPM_HOME=/pnpm PATH=/pnpm:$PATH
RUN corepack enable && corepack prepare pnpm@10.33.2 --activate
WORKDIR /repo

# Source copied before install: t20-data's `prepare` (tsc) runs during install.
COPY pnpm-workspace.yaml package.json pnpm-lock.yaml ./
COPY t20-data t20-data
COPY frontend frontend
RUN --mount=type=cache,id=pnpm,target=/pnpm/store \
    pnpm install --frozen-lockfile --filter frontend --filter @tormenta20/t20-data

# `vite build` emits dist/**.br (see vite.config.ts compression plugin).
RUN pnpm --filter frontend build

# ngx_brotli-enabled nginx so brotli_static can serve the precompressed .br.
# (Stock nginx has no brotli module; this image ships it preloaded.)
FROM fholzer/nginx-brotli:latest AS runtime
COPY frontend/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /repo/frontend/dist /usr/share/nginx/html
EXPOSE 80
