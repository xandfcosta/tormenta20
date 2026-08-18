#!/usr/bin/env bash
# Pré-comprime o build para o servidor não gastar CPU durante a mesa (ALE-153).
#
# O `spaHandler` do Go serve o irmão `.br`/`.gz` quando o navegador aceita, e o
# arquivo cru quando não existe irmão — então rodar isto é opcional para o app
# funcionar, e obrigatório para ele não mandar 4,9 MB por carga fria.
#
# Comprimir aqui, e não a cada requisição, porque a máquina que serve a mesa é a
# do mestre: brotli -q11 custa ~8s UMA vez no build e zero por jogador que entra.
#
# brotli é opcional de propósito: nem toda máquina de build o tem, e o gzip do
# stdlib já corta ~72%. Sem brotli o script avisa e segue.
set -euo pipefail

DIST="${1:-$(dirname "$0")/../dist}"
[ -d "$DIST" ] || { echo "precompress: $DIST não existe — rode o build antes" >&2; exit 1; }

command -v brotli >/dev/null 2>&1 && TEM_BROTLI=1 || TEM_BROTLI=0
[ "$TEM_BROTLI" = 1 ] || echo "precompress: brotli ausente; só gzip (perde ~25% a mais de redução)"

cru=0
comprimido=0
# Só o que comprime: png/jpg/woff2 já são formatos comprimidos, e um .gz deles
# sai MAIOR que o original — servir isso seria pagar CPU para gastar banda.
while IFS= read -r -d '' arquivo; do
  # Abaixo de 1 KiB o cabeçalho de compressão come o ganho.
  [ "$(stat -c%s "$arquivo")" -ge 1024 ] || continue
  gzip -9 -kf "$arquivo"
  [ "$TEM_BROTLI" = 1 ] && brotli -q 11 -f "$arquivo"
  cru=$((cru + $(stat -c%s "$arquivo")))
  menor="$arquivo.gz"
  [ "$TEM_BROTLI" = 1 ] && menor="$arquivo.br"
  comprimido=$((comprimido + $(stat -c%s "$menor")))
done < <(find "$DIST" -type f \( -name '*.js' -o -name '*.css' -o -name '*.html' \
  -o -name '*.svg' -o -name '*.json' -o -name '*.wasm' -o -name '*.map' \) -print0)

echo "precompress: $(numfmt --to=iec $cru) → $(numfmt --to=iec $comprimido) no fio"
