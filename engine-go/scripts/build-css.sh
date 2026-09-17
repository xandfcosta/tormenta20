#!/usr/bin/env bash
# A folha das cenas em Datastar (ALE-219).
#
# Roda a partir de `engine-go/`, que é onde a folha e as dependências dela
# passaram a morar (ALE-272, fatia 10c). Antes ela rodava de `frontend/`, porque
# a folha da casa importava a da SPA inteira — com a SPA saindo, o `index.css`
# veio para `serve/web/assets/src` e este pacote ganhou o próprio `node_modules`.
#
# O resultado é embutido no binário pelo `go:embed`, então ESTE SCRIPT PRECISA
# RODAR ANTES DO `go build`.
set -euo pipefail
cd "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# `pnpm exec` roda uma checagem de dependências que aborta o build quando algum
# script de instalação está pendente de aprovação — irrelevante para gerar CSS.
# O binário direto não tem essa porta.
exec ./node_modules/.bin/tailwindcss \
  --input ./serve/web/assets/app.src.css \
  --output ./serve/web/assets/static/app.css \
  --minify
