#!/usr/bin/env bash
#
# Põe a `main` na produção, com a mesa fora do ar pelo menor tempo possível:
#
#   pnpm deploy:prod
#
# A ORDEM é a regra: tudo que pode falhar sem derrubar ninguém vem ANTES de parar o
# servidor. Teste vermelho ou compilação quebrada terminam aqui com a mesa intacta.
#
# O desligamento é o `SIGTERM`: o servidor termina as gravações em curso, corta os
# fluxos ao vivo e fecha o banco. Os navegadores reconectam sozinhos quando ele volta,
# e a página aberta segue com o CSS e o JS da versão anterior até alguém recarregar.
#
# Se a versão nova não responder, a anterior volta a subir. O BANCO não volta sozinho:
# uma migração que já rodou pode não servir à versão anterior, e restaurar apagaria o
# que a mesa gravou. Quem decide é uma pessoa, e o caminho do backup sai na mensagem.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENGINE_DIR="$ROOT_DIR/engine-go"
ENV_FILE="$ENGINE_DIR/.env.production"
BINARY="$ENGINE_DIR/bin/t20-api"
LOG_FILE="$ENGINE_DIR/data/prod.log"

die() {
  echo "deploy: $*" >&2
  exit 1
}

say() {
  echo "deploy: $*"
}

[ -f "$ENV_FILE" ] || die "$ENV_FILE não existe; copie o .env.production.example"
PORT="$(sed -n 's/^PORT=//p' "$ENV_FILE")"
PORT="${PORT:-3001}"

# listening_pid devolve quem escuta na porta da produção, ou nada.
listening_pid() {
  ss -ltnpH "sport = :$PORT" | sed -nE 's/.*pid=([0-9]+).*/\1/p' | head -n 1
}

# require_production_or_free recusa derrubar o que não é a produção: na mesma porta
# pode estar um dev, e matar por porta sem conferir derruba o processo errado.
require_production_or_free() {
  local pid="$1" executable
  [ -n "$pid" ] || return 0
  executable="$(readlink "/proc/$pid/exe" || true)"
  # ` (deleted)` é o binário trocado em disco com o processo ainda rodando.
  [ "${executable% (deleted)}" = "$BINARY" ] && tr '\0' '\n' <"/proc/$pid/environ" | grep -qx 'APP_ENV=production' ||
    die "a porta $PORT é do processo $pid ($executable), que não é a produção; não vou derrubar"
}

# stop_server manda o SIGTERM e espera. O servidor tem prazo próprio de 10s para
# encerrar, então passar disso é defeito, e o deploy para sem forçar nada.
stop_server() {
  local pid="$1"
  kill -TERM "$pid"
  for _ in $(seq 1 150); do
    [ -d "/proc/$pid" ] || return 0
    sleep 0.1
  done
  die "o processo $pid não encerrou em 15s; nada foi trocado"
}

# start_and_wait sobe o binário e espera a saúde responder. Falha se ele morrer ou se
# não responder a tempo.
start_and_wait() {
  local pid
  (
    cd "$ENGINE_DIR"
    APP_ENV=production nohup "$BINARY" >>"$LOG_FILE" 2>&1 </dev/null &
    echo $! >"$ENGINE_DIR/bin/t20-api.pid"
  )
  pid="$(cat "$ENGINE_DIR/bin/t20-api.pid")"
  rm -f "$ENGINE_DIR/bin/t20-api.pid"
  for _ in $(seq 1 100); do
    [ -d "/proc/$pid" ] || return 1
    if curl -sf -o /dev/null "http://localhost:$PORT/health"; then
      return 0
    fi
    sleep 0.2
  done
  kill -TERM "$pid" 2>/dev/null || true
  return 1
}

# Falha cedo, antes dos minutos dos testes.
require_production_or_free "$(listening_pid)"

cd "$ROOT_DIR"
[ "$(git rev-parse --abbrev-ref HEAD)" = "main" ] || die "o deploy sai da main, e este checkout está em $(git rev-parse --abbrev-ref HEAD)"
[ -z "$(git -c core.fsmonitor=false status --porcelain --untracked-files=no)" ] || die "há mudança sem commit; a produção tem de ser exatamente um commit da main"
git pull --ff-only --quiet
say "versão $(git rev-parse --short HEAD)"

say "rodando os testes"
(cd "$ENGINE_DIR" && go test ./...) >/dev/null || die "testes vermelhos; rode 'go test ./...' em engine-go. A mesa segue no ar"

say "compilando"
# Sem `pnpm build`: a folha e as ilhas são versionadas, e a CI exige que batam com a
# fonte. Regenerá-las aqui só sujaria o checkout.
(cd "$ENGINE_DIR" && go build -o "$BINARY.next" ./cmd/api) || die "a compilação falhou. A mesa segue no ar"

# A porta pode ter mudado de dono durante os testes.
RUNNING_PID="$(listening_pid)"
require_production_or_free "$RUNNING_PID"

if [ -n "$RUNNING_PID" ]; then
  say "encerrando o processo $RUNNING_PID"
  stop_server "$RUNNING_PID"
fi

mkdir -p "$(dirname "$LOG_FILE")"
HAD_PREVIOUS=""
[ -f "$BINARY" ] && HAD_PREVIOUS="yes"

DB_BACKUP="sem backup: o banco ainda não existia"
if BACKUP_OUTPUT="$(bash "$ROOT_DIR/scripts/backup-db.sh" production 2>&1)"; then
  DB_BACKUP="$BACKUP_OUTPUT"
elif ! grep -q 'não existe' <<<"$BACKUP_OUTPUT"; then
  # Sem backup a troca não acontece. A versão que estava no ar volta.
  [ -n "$HAD_PREVIOUS" ] && [ -n "$RUNNING_PID" ] && start_and_wait || true
  die "o backup falhou, e a versão anterior voltou: $BACKUP_OUTPUT"
fi
say "$DB_BACKUP"

[ -n "$HAD_PREVIOUS" ] && cp -p "$BINARY" "$BINARY.prev"
mv "$BINARY.next" "$BINARY"

if start_and_wait; then
  say "no ar: $(grep -a 'players can open' "$LOG_FILE" | tail -n 1 | sed 's/.*players can open //' || true)"
  exit 0
fi

if [ -z "$HAD_PREVIOUS" ]; then
  die "a versão nova não respondeu, e não há anterior para voltar. Veja $LOG_FILE"
fi
mv "$BINARY.prev" "$BINARY"
if start_and_wait; then
  die "a versão nova não respondeu, e a anterior voltou. Veja $LOG_FILE. Banco: $DB_BACKUP"
fi
die "nem a versão nova nem a anterior responderam: a mesa está FORA DO AR. Veja $LOG_FILE. Banco: $DB_BACKUP"
