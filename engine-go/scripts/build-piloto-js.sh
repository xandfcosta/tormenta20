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
exec ./node_modules/.bin/vite build --config vite.piloto.config.ts --logLevel warn
