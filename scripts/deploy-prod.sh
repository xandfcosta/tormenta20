#!/usr/bin/env bash
#
# Põe a `main` na produção, com a mesa fora do ar pelo menor tempo possível:
#
#   pnpm deploy:prod                # a `main` atual
#   pnpm deploy:prod --voltar       # a versão anterior, sem testes
#   pnpm deploy:prod --com-migracao # aceita migração nova, que não volta atrás
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

MODE="deploy"
ALLOW_MIGRATION=""
for argument in "$@"; do
  case "$argument" in
    --voltar) MODE="rollback" ;;
    --com-migracao) ALLOW_MIGRATION="yes" ;;
    *) die "opção $argument desconhecida; use --voltar ou --com-migracao" ;;
  esac
done

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

# binary_revision lê o commit que o `go build` carimbou no binário, ou nada.
binary_revision() {
  [ -f "$1" ] || return 0
  go version -m "$1" | sed -n 's/.*vcs\.revision=//p'
}

# short_revision encurta o commit para a mensagem.
short_revision() {
  echo "${1:0:8}"
}

# migrations_between lista as migrações que mudaram entre dois commits. Commit vazio é
# binário sem carimbo: sem como comparar, a pergunta não tem resposta, e não lista nada.
migrations_between() {
  [ -n "$1" ] && [ -n "$2" ] || return 0
  git -C "$ROOT_DIR" diff --name-only "$1" "$2" -- engine-go/infra/db/migrations
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

# release_answers confere que quem responde é a versão esperada, e não um processo velho na
# porta, e que uma cena de verdade abre. O `/entrar` é a única sem sessão: ele prova que o
# app desenha uma página, mas não diz nada sobre a Mesa.
release_answers() {
  local expected="$1" health
  health="$(curl -sf "http://localhost:$PORT/health")" || return 1
  [ -z "$expected" ] || grep -q "\"version\":\"$expected\"" <<<"$health" || return 1
  curl -sf -o /dev/null "http://localhost:$PORT/entrar"
}

# take_backup faz o backup e devolve a mensagem dele. Banco que ainda não existe não é erro.
take_backup() {
  local output
  if output="$(bash "$ROOT_DIR/scripts/backup-db.sh" production 2>&1)"; then
    echo "$output"
  elif grep -q 'não existe' <<<"$output"; then
    echo "sem backup: o banco ainda não existia"
  else
    echo "$output" >&2
    return 1
  fi
}

# stop_whatever_answers derruba o que subiu e não serviu. Ele já passou pela conferência
# de quem é, porque foi este comando que o subiu.
stop_whatever_answers() {
  local pid
  pid="$(listening_pid)"
  [ -z "$pid" ] || stop_server "$pid"
}

# Falha cedo, antes dos minutos dos testes.
require_production_or_free "$(listening_pid)"
CURRENT_REVISION="$(binary_revision "$BINARY")"

if [ "$MODE" = "rollback" ]; then
  [ -f "$BINARY.prev" ] || die "não há versão anterior guardada em $BINARY.prev"
  PREVIOUS_REVISION="$(binary_revision "$BINARY.prev")"
  # A versão anterior pode não entender o banco que a atual migrou. Restaurar o backup
  # apagaria o que a mesa gravou, e essa decisão não é deste comando.
  if [ -n "$(migrations_between "$PREVIOUS_REVISION" "$CURRENT_REVISION")" ]; then
    die "a versão atual trouxe migração, e a anterior pode não entender o banco. Nada foi trocado. Corrija para a frente, ou restaure um backup de $ROOT_DIR/backups à mão"
  fi
  say "voltando de $(short_revision "$CURRENT_REVISION") para $(short_revision "$PREVIOUS_REVISION")"
  RUNNING_PID="$(listening_pid)"
  [ -z "$RUNNING_PID" ] || stop_server "$RUNNING_PID"
  DB_BACKUP="$(take_backup)" || die "o backup falhou, e a mesa está parada. Suba de novo com pnpm start"
  say "$DB_BACKUP"
  # Troca, e não apaga: a versão de que se voltou continua à mão.
  mv "$BINARY" "$BINARY.swap"
  mv "$BINARY.prev" "$BINARY"
  mv "$BINARY.swap" "$BINARY.prev"
  if start_and_wait && release_answers "$PREVIOUS_REVISION"; then
    say "no ar na versão anterior: $(grep -a 'players can open' "$LOG_FILE" | tail -n 1 | sed 's/.*players can open //' || true)"
    exit 0
  fi
  die "a versão anterior não respondeu: a mesa está FORA DO AR. Veja $LOG_FILE"
fi

cd "$ROOT_DIR"
[ "$(git rev-parse --abbrev-ref HEAD)" = "main" ] || die "o deploy sai da main, e este checkout está em $(git rev-parse --abbrev-ref HEAD)"
[ -z "$(git -c core.fsmonitor=false status --porcelain --untracked-files=no)" ] || die "há mudança sem commit; a produção tem de ser exatamente um commit da main"
git pull --ff-only --quiet
NEW_REVISION="$(git rev-parse HEAD)"
say "de $(short_revision "$CURRENT_REVISION") para $(short_revision "$NEW_REVISION")"

NEW_MIGRATIONS="$(migrations_between "$CURRENT_REVISION" "$NEW_REVISION")"
if [ -n "$NEW_MIGRATIONS" ] && [ -z "$ALLOW_MIGRATION" ]; then
  die "esta versão traz migração, que não volta atrás com a mesa no meio de um jogo. A mesa segue no ar. Para subir mesmo assim, use --com-migracao: $NEW_MIGRATIONS"
fi

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

if ! DB_BACKUP="$(take_backup)"; then
  # Sem backup a troca não acontece. A versão que estava no ar volta.
  [ -n "$HAD_PREVIOUS" ] && [ -n "$RUNNING_PID" ] && start_and_wait || true
  die "o backup falhou, e a versão anterior voltou"
fi
say "$DB_BACKUP"

[ -n "$HAD_PREVIOUS" ] && cp -p "$BINARY" "$BINARY.prev"
mv "$BINARY.next" "$BINARY"

if start_and_wait && release_answers "$NEW_REVISION"; then
  say "no ar: $(grep -a 'players can open' "$LOG_FILE" | tail -n 1 | sed 's/.*players can open //' || true)"
  exit 0
fi

# Pode ter subido e reprovado na conferência, e a porta tem de ficar livre para a anterior.
stop_whatever_answers
if [ -z "$HAD_PREVIOUS" ]; then
  die "a versão nova não respondeu, e não há anterior para voltar. Veja $LOG_FILE"
fi
mv "$BINARY.prev" "$BINARY"
if start_and_wait; then
  die "a versão nova não respondeu, e a anterior voltou. Veja $LOG_FILE. Banco: $DB_BACKUP"
fi
die "nem a versão nova nem a anterior responderam: a mesa está FORA DO AR. Veja $LOG_FILE. Banco: $DB_BACKUP"
