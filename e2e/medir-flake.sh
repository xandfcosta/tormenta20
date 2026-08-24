#!/usr/bin/env bash
# Mede a taxa de vermelho de UM arquivo de spec, com corridas INDEPENDENTES (ALE-238).
#
# Por que restaurar o banco e reiniciar o servidor a cada corrida, e não é
# zelo: a primeira versão deste script não fazia isso e se ENVENENOU. A corrida
# 2 falhou (o diálogo do bestiário não fechou) e deixou um "Ogro" na fila da
# sessão 4; da corrida 3 em diante TODAS morreram em strict mode com "Ogro" e
# "Ogro 2", cada uma acumulando mais. O resultado foi 9 vermelhos em 10, e
# nenhum deles media o que eu queria medir.
#
# É a armadilha que o repositório já documenta em dois lugares — asserção que
# depende do estado do combate mede o BANCO, não o app — e a forma nova dela:
# num laço de medição, a primeira falha CONTAMINA todas as seguintes, então o
# número sai alto e parece sinal forte. Falha que se propaga é pior que falha
# que se repete, porque a segunda ao menos é honesta.
#
# `set -u` porque erro de digitação em variável tem de estourar em vez de contar
# verde em silêncio.
set -u

# Nada de caminho fixo: a raiz sai do próprio git, e as portas do ambiente —
# cada worktree sobe a bancada dela, e um script com a porta cravada mede o
# servidor da árvore errada. Já custou uma sessão inteira depurando um 404 que
# era o proxy apontando para outro lugar.
RAIZ=$(git rev-parse --show-toplevel)
BANCO="$RAIZ/engine-go/data/t20-dev.db"
MOLDE=${MOLDE:-/tmp/molde-flake-$(basename "$RAIZ").db}
BINARIO=${BINARIO:-/tmp/api-flake-$(basename "$RAIZ")}
PORTA_API=${PORTA_API:?exporte PORTA_API, a da SUA bancada}
PORTA_WEB=${PORTA_WEB:?exporte PORTA_WEB, a do SEU vite}
EVIDENCIA=${EVIDENCIA:-/tmp/flake-evidencia}

# O molde é feito UMA vez, do banco como ele está agora. Refazê-lo a cada
# corrida herdaria a sujeira da anterior, que é o defeito que este script
# existe para não ter.
if [ ! -f "$MOLDE" ]; then
  sqlite3 "$BANCO" "VACUUM INTO '$MOLDE'" || exit 1
  echo "molde criado em $MOLDE"
fi
[ -x "$BINARIO" ] || (cd "$RAIZ/engine-go" && go build -o "$BINARIO" ./cmd/api) || exit 1

cd "$RAIZ/e2e" || exit 1

ARQUIVO=${1:?uso: medir-flake.sh <spec> <n>}
N=${2:-8}

para_a_api() {
  local pid
  pid=$(ss -ltnp 2>/dev/null | /usr/bin/grep ":$PORTA_API " | /usr/bin/grep -oP 'pid=\K[0-9]+' | head -1)
  [ -n "$pid" ] || return 0
  # Confere o CWD antes de matar: a sessão vizinha também roda Go nesta máquina,
  # e padrão largo mira nos irmãos.
  case "$(readlink "/proc/$pid/cwd" 2>/dev/null)" in
    "$RAIZ"/*) kill "$pid" ;;
    *) echo "RECUSO matar o pid $pid — não é da minha worktree"; exit 1 ;;
  esac
  for _ in $(seq 1 40); do
    ss -ltn 2>/dev/null | /usr/bin/grep -q ":$PORTA_API " || return 0
    sleep 0.25
  done
}

sobe_a_api() {
  # `cd` para o engine-go é obrigatório e não é estilo: o `DATABASE_URL` é
  # RELATIVO (`./data/t20-dev.db`), então subir daqui faria o servidor criar um
  # banco novo em `e2e/data/` e migrar do zero — o teste rodaria contra uma seed
  # vazia e falharia por um motivo que não existe no produto. Aconteceu.
  (
    cd "$RAIZ/engine-go" || exit 1
    PORT=$PORTA_API \
      CORS_ORIGIN="http://localhost:$PORTA_WEB,http://[::1]:$PORTA_WEB,http://127.0.0.1:$PORTA_WEB" \
      "$BINARIO" > "/tmp/api-flake.log" 2>&1 &
  )
  # Com pausa: sem ela o laço gasta as 100 voltas em milissegundos e declara
  # "não subiu" sobre um servidor que estava subindo.
  for _ in $(seq 1 60); do
    ss -ltn 2>/dev/null | /usr/bin/grep -q ":$PORTA_API " && break
    sleep 0.5
  done
  # Escutar a porta NÃO basta: o que os testes usam é o caminho INTEIRO, e o
  # Vite mantém sockets vivos para o alvo antigo. Depois de eu derrubar a API, a
  # primeira requisição pelo proxy pode morrer num socket obsoleto — e o sintoma
  # não parece bancada: é "Conectado" que não aparece, ou o próprio login
  # estourando, em testes DIFERENTES a cada corrida. Foi o que produziu 5
  # vermelhos espalhados numa medição minha, que eu quase reportei como taxa.
  #
  # A espera é pelo caminho que o teste percorre, não pelo que é fácil de medir.
  for _ in $(seq 1 60); do
    if [ "$(curl -s -o /dev/null -w '%{http_code}' "http://localhost:$PORTA_WEB/api/catalog/spells")" = "200" ]; then
      return 0
    fi
    sleep 0.5
  done
  echo "a api não respondeu PELO VITE"; exit 1
}

vermelhos=0
for i in $(seq 1 "$N"); do
  para_a_api
  # `-wal`/`-shm` vão junto: restaurar só o arquivo principal deixaria escrita
  # pendente da corrida anterior voltando por cima do molde.
  rm -f "$BANCO" "$BANCO-wal" "$BANCO-shm"
  cp "$MOLDE" "$BANCO"
  sobe_a_api

  saida=$(E2E_BASE_URL="http://localhost:$PORTA_WEB" E2E_SERVIDOR_EXTERNO=1 \
    npx playwright test "$ARQUIVO" --reporter=line 2>&1)
  codigo=$?
  linha=$(echo "$saida" | /usr/bin/grep -E "passed|failed" | tail -1)
  if [ "$codigo" -ne 0 ]; then
    vermelhos=$((vermelhos + 1))
    echo "corrida $i: VERMELHO  ($linha)"
    echo "$saida" | /usr/bin/grep -E "^\s+[0-9]+\) |Error:|session.spec.ts:[0-9]+:[0-9]+$" | head -6
    # GUARDA a evidência fora do `test-results`, que a PRÓXIMA corrida limpa.
    # Sem isto a corrida 3 apaga o artefato da 2, e sobra o número sem a causa —
    # que é o buraco que a ALE-244 existiu para tapar e que este laço reabria.
    destino="$EVIDENCIA/corrida-$i"
    mkdir -p "$destino"
    cp -r test-results/. "$destino/" 2>/dev/null
    echo "         evidência em $destino"
  else
    echo "corrida $i: verde     ($linha)"
  fi
done

echo "=== $ARQUIVO: $vermelhos vermelho(s) em $N (corridas independentes) ==="
