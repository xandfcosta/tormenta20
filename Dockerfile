# A MESA NUMA IMAGEM (ALE-273).
#
# Continua sendo UM processo servindo tudo — as cenas, a API e o fluxo ao vivo na
# mesma porta (ALE-101). O que o contêiner acrescenta é o deploy reproduzível e
# um lugar só para o banco viver; não há proxy na frente, e a compressão das
# cenas acontece dentro do Go (`plataforma.Gzip`).
#
# # Por que UM estágio de build e não dois
#
# O binário embute a folha e as ilhas de JS do piloto (`go:embed`), e quem as
# compila é o `prebuild` do `engine-go` — Node. Separar "estágio do node" de
# "estágio do Go" obrigaria a passar os artefatos de um para o outro na mão, que
# é a forma de o binário sair com uma folha velha sem nada falhar. Um estágio com
# as duas ferramentas mantém o caminho de build igual ao do CI.
#
# Ele já teve um segundo motivo, maior: o `pnpm build` da SPA compilava o motor
# para WASM e precisava do Go. A SPA saiu na ALE-272 e o WASM com ela; o que
# sobra é o de cima.

# ─── build ───────────────────────────────────────────────────────────────────
FROM golang:1.26-bookworm AS build

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates \
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
COPY engine-go/package.json engine-go/
COPY e2e/package.json e2e/
RUN pnpm install --frozen-lockfile

COPY engine-go/go.mod engine-go/go.sum engine-go/
RUN cd engine-go && go mod download

COPY . .

# A FOLHA e as ILHAS do piloto, que o `go:embed` leva para dentro do binário.
# Elas são versionadas, então o build não depende deste passo para compilar — ele
# existe para a imagem não sair com um artefato mais velho que a fonte.
RUN cd engine-go && pnpm run prebuild

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

# Os certificados vêm da imagem base. Só o binário e o catálogo viajam: as cenas,
# a folha, as ilhas e as fontes estão DENTRO do binário desde que a SPA saiu.
COPY --from=build /out/t20-api /app/t20-api

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
    DATABASE_URL=file:/app/data/t20.db \
    CATALOG_PATH=/app/catalogs.json \
    BACKUP_DIR=/app/backups

EXPOSE 3001

# `nonroot` já vem da tag da base — o processo não roda como root, e o volume do
# banco precisa ser gravável por ele (uid 65532).
USER nonroot:nonroot

ENTRYPOINT ["/app/t20-api"]
