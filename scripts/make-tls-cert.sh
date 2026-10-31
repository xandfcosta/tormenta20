#!/usr/bin/env bash
# O CERTIFICADO AUTOASSINADO da mesa (ALE-523), em ./tls, para o nginx do
# compose.
#
#   scripts/make-tls-cert.sh                 # os IPs desta máquina
#   scripts/make-tls-cert.sh 192.168.1.36    # um IP escrito à mão
#
# Ele vale para os IPs da rede de casa e para `localhost`: o navegador confere
# o endereço digitado contra o certificado, e um certificado sem o IP da mesa
# é recusado mesmo depois de aceitar o aviso. Cada aparelho mostra o aviso de
# "conexão não segura" na primeira vez — é o preço de ninguém assinar por ele.
#
# Trocou o IP da máquina? Rode de novo e reinicie o nginx:
#   docker compose restart nginx
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
dir="$root/tls"
mkdir -p "$dir"

ips=("$@")
# Os IPv4 globais da máquina, pelo `ip` e não pelo `hostname -I`, que nem toda
# distribuição traz. As pontes do Docker (172.x) entram junto e não atrapalham.
if [ ${#ips[@]} -eq 0 ]; then
  mapfile -t ips < <(ip -4 -o addr show scope global | awk '{split($4, a, "/"); print a[1]}')
fi

san="DNS:localhost,IP:127.0.0.1"
for ip in "${ips[@]}"; do
  san="$san,IP:$ip"
done

openssl req -x509 -newkey rsa:2048 -nodes -days 825 \
  -keyout "$dir/key.pem" -out "$dir/cert.pem" \
  -subj "/CN=Tormenta 20" -addext "subjectAltName=$san" 2>/dev/null
chmod 644 "$dir/cert.pem"
chmod 640 "$dir/key.pem"

echo "certificado em $dir/cert.pem, válido para: $san"
