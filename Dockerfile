# A MESA NUMA IMAGEM (ALE-273).
#
# Continua sendo UM processo servindo tudo — SPA, API e o fluxo ao vivo na mesma
# porta (ALE-101). O que o contêiner acrescenta é o deploy reproduzível e um
# lugar só para o banco viver; não há proxy na frente, e a compressão das cenas
# acontece dentro do Go (`plataforma.Gzip`).
#
# # Por que UM estágio de build e não dois
#
# O `pnpm build` do front tem um `prebuild` que compila o motor para WASM, e para
# isso ele precisa do Go. Separar "estágio do node" de "estágio do Go" obrigaria
# a pular o hook e a montar o `.wasm` na mão — que é exatamente o defeito que o
# `engine-go/CLAUDE.md` registra: o hook sumiu uma vez, o `vite build` continuou
# passando, e o app quebrava só em RUNTIME lendo campos de undefined. Um estágio
# com as duas ferramentas mantém o caminho de build igual ao do CI.

# ─── build ───────────────────────────────────────────────────────────────────
FROM golang:1.26-bookworm AS build

# `brotli` porque o `postbuild` do front pré-comprime o `dist` (ALE-153) e sem
# ele o script AVISA e segue só com gzip — a imagem sairia 25% mais gorda no fio
# sem nada falhar.
RUN apt-get update \
    && apt-get install -y --no-install-recommends brotli ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Node pela distribuição oficial, e pnpm pelo corepack — as MESMAS versões que o
# `.github/workflows/ci.yml` fixa. Divergir aqui é construir um artefato que o CI
# nunca viu.
ENV NODE_VERSION=22 PNPM_VERSION=10.33.2
RUN curl -fsSL "https://deb.nodesource.com/setup_${NODE_VERSION}.x" | bash - \
    && apt-get install -y --no-install-recommends nodejs \
    && rm -rf /var/lib/apt/lists/* \
    && corepack enable \
    && corepack prepare "pnpm@${PNPM_VERSION}" --activate

WORKDIR /src

# As DEPENDÊNCIAS primeiro, em camada própria: elas mudam muito menos que o
# código, e assim um commit de `.templ` não rebaixa o `pnpm install` inteiro.
COPY pnpm-lock.yaml pnpm-workspace.yaml package.json ./
COPY frontend/package.json frontend/
COPY engine-go/package.json engine-go/
COPY e2e/package.json e2e/
RUN pnpm install --frozen-lockfile

COPY engine-go/go.mod engine-go/go.sum engine-go/
RUN cd engine-go && go mod download

COPY . .

# A SPA sai daqui com o `.wasm`, os assets e os irmãos `.br`/`.gz`.
RUN pnpm --filter frontend build

# `CGO_ENABLED=0` porque o driver de SQLite é o `modernc.org/sqlite`, que é Go
# puro: o binário sai estático e a imagem final não precisa de libc.
RUN cd engine-go && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/t20-api ./cmd/api

# ─── runtime ─────────────────────────────────────────────────────────────────
#
# `distroless/static` e não `alpine`: sem shell, sem gerenciador de pacotes e sem
# libc, porque o binário é estático e não precisa de nenhum dos três. O que sobra
# é o app, e a superfície de ataque de uma mesa exposta na LAN encolhe junto.
FROM gcr.io/distroless/static-debian12:nonroot AS runtime

WORKDIR /app

# Os certificados vêm da imagem base. Atravessam o binário, a SPA e o catálogo.
COPY --from=build /out/t20-api /app/t20-api
COPY --from=build /src/frontend/dist /app/frontend/dist

# O CATÁLOGO NÃO É OPCIONAL, e essa é a linha mais fácil de esquecer aqui.
#
# Sem o arquivo, o `primeCatalogs` devolve nulo, escreve "mutation validators
# disabled" numa linha de log e o app SOBE — degradado. A Defesa vira travessão,
# os cartões de arma somem e a aba de Combate fica vazia. É falha em silêncio com
# cara de app quebrado, e é por isso que ele viaja dentro da imagem em vez de
# depender de um caminho relativo que só existe na árvore do repositório.
COPY --from=build /src/engine-go/parity/_catalogs.json /app/catalogs.json

# O BANCO e os BACKUPS moram em volume (ver `docker-compose.yml`).
ENV APP_ENV=production \
    PORT=3001 \
    STATIC_DIR=/app/frontend/dist \
    DATABASE_URL=file:/app/data/t20.db \
    CATALOG_PATH=/app/catalogs.json \
    BACKUP_DIR=/app/backups

EXPOSE 3001

# `nonroot` já vem da tag da base — o processo não roda como root, e o volume do
# banco precisa ser gravável por ele (uid 65532).
USER nonroot:nonroot

ENTRYPOINT ["/app/t20-api"]
