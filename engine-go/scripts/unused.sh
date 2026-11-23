#!/usr/bin/env bash
# Reprova quando há tipo, variável, constante ou campo que NADA usa — o que o `deadcode` (que só enxerga função)
# não vê. Quem responde é a regra U1000 do `staticcheck`, que conta os testes como uso: o que só um teste usa está
# vivo.
#
# Ele roda por `go run …@versão`, e não como ferramenta do `go.mod`: o `go get -tool` do `staticcheck` subia junto
# as dependências DIRETAS do app (`x/crypto`, `x/net`, `x/text`), e trocar o que vai no binário não é o trabalho de
# um passo do portão. A versão fica escrita aqui. O portão local (`scripts/ci-local.sh`) e a CI (`ci.yml`) chamam
# este mesmo script.
set -euo pipefail
cd "$(dirname "$0")/.."
go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 -checks U1000 ./...
