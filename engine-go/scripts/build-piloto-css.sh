#!/usr/bin/env bash
# A folha do piloto Datastar (ALE-219).
#
# Roda a partir de `engine-go/`, que é onde a folha e as dependências dela
# passaram a morar (ALE-272, fatia 10c). Antes ela rodava de `frontend/`, porque
# a folha do piloto importava a da SPA inteira — com a SPA saindo, o `index.css`
# veio para `api/piloto/src` e este pacote ganhou o próprio `node_modules`.
#
# O resultado é embutido no binário pelo `go:embed`, então ESTE SCRIPT PRECISA
# RODAR ANTES DO `go build`.
set -euo pipefail
cd "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# `pnpm exec` roda uma checagem de dependências que aborta o build quando algum
# script de instalação está pendente de aprovação — irrelevante para gerar CSS.
# O binário direto não tem essa porta.
exec ./node_modules/.bin/tailwindcss \
  --input ./api/piloto/piloto.src.css \
  --output ./api/piloto/static/piloto.css \
  --minify
