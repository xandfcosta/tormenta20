#!/usr/bin/env bash
# O módulo JS do piloto Datastar (ALE-231).
#
# Mesma forma e mesmo motivo do `build-piloto-css.sh`: o fonte é o da SPA
# (`frontend/src/piloto/cena.ts`, que importa `shared/lib/*`), e o produto é
# embutido no binário pelo `go:embed`. UMA fonte, dois consumidores — duplicar
# o driver de teclado seria a armadilha de divergência que a casca e os botões
# já mostraram.
#
# ESTE SCRIPT PRECISA RODAR ANTES DO `go build`, como o do CSS e o do WASM.
set -euo pipefail
raiz="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$raiz/frontend"
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
   "$raiz/engine-go/api/piloto/static/pdf.worker.js"
