#!/usr/bin/env bash
# A folha do piloto Datastar (ALE-219).
#
# Roda a partir de `frontend/` porque a folha importa a da SPA, e essa faz
# `@import "tailwindcss"` — a resolução do Node precisa enxergar o
# `node_modules` de lá. O resultado é embutido no binário pelo `go:embed`, então
# ESTE SCRIPT PRECISA RODAR ANTES DO `go build`, como o `build-engine-wasm.sh`.
#
# É o segundo pipeline de CSS do repositório, e isso é um custo conhecido do
# piloto, não um descuido: a SPA compila o dela pelo plugin do Vite.
set -euo pipefail
raiz="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$raiz/frontend"
# `pnpm exec` roda uma checagem de dependências que aborta o build quando algum
# script de instalação está pendente de aprovação — irrelevante para gerar CSS.
# O binário direto não tem essa porta.
exec ./node_modules/.bin/tailwindcss \
  --input "$raiz/engine-go/api/mesa/mesa.src.css" \
  --output "$raiz/engine-go/api/mesa/static/mesa.css" \
  --minify
