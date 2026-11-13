#!/usr/bin/env bash
# O bundle do Datastar que o `go:embed` leva para dentro do binário (ALE-370).
#
# Ele é VENDORIZADO — baixado e commitado —, e não uma dependência do
# `package.json`. O motivo é que ele não passa pelo Vite: o `layout.templ` o
# referencia com um `<script type="module" src>` escrito à mão, e o `build-js.sh`
# nem o enxerga. Um `pnpm add` poria o arquivo em `node_modules/`, onde o
# `go:embed` não alcança.
#
# ESTE SCRIPT EXISTE PARA A PROCEDÊNCIA, e não para a compilação: até a ALE-370
# nada no repositório dizia de ONDE o arquivo tinha vindo nem QUAL versão ele
# era — só a primeira linha do próprio bundle, que é fácil de não olhar. Ele não
# entra no `prebuild`: subir de versão é ato deliberado, como regenerar o
# oráculo, porque o reconciliador do Datastar é o que mantém as cenas vivas.
#
# Depois de rodar: `go test ./...` AQUI e o Playwright INTEIRO em `e2e/`. Os
# guardas que importam são `ferir duas vezes seguidas não apaga a piscada da
# primeira` (ALE-322), que prende o morph, e o `live-stream-reconnects.spec.ts`,
# que prende a reconexão do fluxo longo — as duas coisas que uma versão nova
# pode mudar em silêncio. E leia o diff das FONTES entre as tags, pelo `.js.map`.
set -euo pipefail
cd "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# A ÚNICA linha a editar para subir de versão.
DATASTAR_VERSION=v1.0.5

# `bundles/datastar.js` da tag, e não o do CDN nem o `-aliased`: o aliased renomeia
# os atributos `data-*` por um prefixo próprio, e todas as cenas escrevem `data-`.
SOURCE="https://raw.githubusercontent.com/starfederation/datastar/${DATASTAR_VERSION}/bundles/datastar.js"
TARGET="serve/web/assets/static/datastar.js"

curl -fsSL "$SOURCE" -o "$TARGET.tmp"

# O bundle abre com `// Datastar vX.Y.Z`. Conferir isso é o que impede o arquivo
# de divergir da versão fixada aqui em cima — uma tag movida, um redirecionamento,
# um download truncado.
EXPECTED="// Datastar ${DATASTAR_VERSION}"
FOUND=$(head -n 1 "$TARGET.tmp")
if [ "$FOUND" != "$EXPECTED" ]; then
  rm -f "$TARGET.tmp"
  echo "o bundle baixado se diz '${FOUND}' e esta casa fixou '${EXPECTED}' — confira ${SOURCE}" >&2
  exit 1
fi

mv "$TARGET.tmp" "$TARGET"
echo "${TARGET}: ${DATASTAR_VERSION} (${SOURCE})"
