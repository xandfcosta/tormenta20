#!/usr/bin/env bash
# Gera os ícones do app instalado a partir do `favicon.svg` (ALE-118).
#
# Existe porque o Chrome só OFERECE instalar quando o manifest aponta para um
# ícone de 192 e um de 512 que realmente carregam — um `href` quebrado não dá
# erro em lugar nenhum, o prompt de instalação simplesmente nunca aparece. E o
# iPhone ignora o manifest inteiro para o ícone: ele usa o `apple-touch-icon`, e
# sem ele a Tela de Início ganha um PRINT da página.
#
# Os PNG são COMMITADOS: `public/` é servido cru, e o build de produção não pode
# depender do `rsvg-convert` estar instalado na máquina do dono da mesa. Isto
# aqui roda à mão quando o desenho do ícone mudar.
#
# Por que três formatos e não um:
#   any       — fundo transparente, que é o que o sistema espera poder compor;
#   maskable  — o Android RECORTA o ícone na forma do lançador (círculo,
#               squircle…), então o desenho fica em 55% do centro sobre o fundo
#               do grimório. Um ícone transparente marcado como maskable vira um
#               raio cortado dentro de um círculo branco;
#   apple     — o iOS compõe sobre PRETO e aplica a própria máscara arredondada,
#               então ele também precisa de fundo opaco.
set -euo pipefail

AQUI="$(cd "$(dirname "$0")" && pwd)"
PUBLIC="$AQUI/../public"
SVG="$PUBLIC/favicon.svg"
DEST="$PUBLIC/icons"
# O fundo é o `--grimorio-bg` (oklch(0.15 0.01 305)) convertido para sRGB: o
# manifest e o iOS querem uma cor sólida, e a splash do Android usa a mesma.
FUNDO="#0c0a0f"

for prog in rsvg-convert magick; do
  command -v "$prog" >/dev/null 2>&1 || { echo "build-app-icons: falta $prog" >&2; exit 1; }
done
mkdir -p "$DEST"

# desenhar <lado> <proporção-do-desenho> <fundo|none> <saída>
desenhar() {
  local lado="$1" proporcao="$2" fundo="$3" saida="$4"
  local interno=$(( lado * proporcao / 100 ))
  rsvg-convert -w "$interno" -h "$interno" --keep-aspect-ratio "$SVG" -o /tmp/t20-icone.png
  magick -size "${lado}x${lado}" "xc:$fundo" /tmp/t20-icone.png -gravity center -composite \
    -strip "PNG32:$saida"
  rm -f /tmp/t20-icone.png
}

desenhar 192 88 none "$DEST/icon-192.png"
desenhar 512 88 none "$DEST/icon-512.png"
desenhar 512 55 "$FUNDO" "$DEST/icon-maskable-512.png"
desenhar 180 66 "$FUNDO" "$DEST/apple-touch-icon.png"

echo "build-app-icons: $(ls -1 "$DEST" | tr '\n' ' ')"
