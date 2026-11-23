#!/usr/bin/env bash
# Reprova quando há função que NADA alcança, nem o app nem os testes.
#
# O `-test` conta os testes como raiz de propósito: o ajudante que só um teste usa está vivo, e sem a flag o
# `deadcode` o acusaria junto. O que sobra com ela é código que ninguém chama — o que ficou de uma rota, de uma
# tela ou da SPA que saiu.
#
# O `deadcode` sai com 0 mesmo quando acha alguma coisa, e por isso quem reprova é a saída não vazia. O portão
# local (`scripts/ci-local.sh`) e a CI (`ci.yml`) chamam este mesmo script, para não haver duas cópias do
# comando.
set -euo pipefail
cd "$(dirname "$0")/.."
found=$(go tool deadcode -test ./...)
if [ -n "$found" ]; then
  echo "$found"
  echo
  echo "Função que nada alcança: apague-a, ou, se ela deveria ter chamador, ligue-o."
  exit 1
fi
