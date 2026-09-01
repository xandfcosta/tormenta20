#!/usr/bin/env bash
# O módulo JS do piloto Datastar (ALE-231).
#
# Mesma forma e mesmo motivo do `build-piloto-css.sh`: o fonte agora é daqui
# (`api/piloto/src/cena.ts`) e o produto é embutido no binário pelo `go:embed`.
# Ele vivia em `frontend/` enquanto a SPA existia, porque o driver de teclado, o
# som e as peças eram compartilhados — a ALE-272 (fatia 10c) trouxe as fontes
# para cá junto com as dependências.
#
# ESTE SCRIPT PRECISA RODAR ANTES DO `go build`, como o do CSS.
set -euo pipefail
cd "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# O binário direto, e não `pnpm exec`, pelo mesmo motivo do CSS: o `pnpm exec`
# roda uma checagem de dependências que aborta quando há script de instalação
# pendente de aprovação, e isso é irrelevante para gerar um bundle.
./node_modules/.bin/vite build --config vite.piloto.config.ts --logLevel warn

# O WORKER do pdf.js é COPIADO e não empacotado (ALE-264): ele é um segundo
# ponto de entrada, carregado pelo navegador com `new Worker(url)`, e o Vite em
# modo `lib` não emite worker de dependência. São 1,3 MB que só a cena do leitor
# pede — e é justamente por rodar fora da thread principal que a página do livro
# desenha sem travar a cena.
cp node_modules/pdfjs-dist/build/pdf.worker.min.mjs \
   ./api/piloto/static/pdf.worker.js
