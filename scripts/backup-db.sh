#!/usr/bin/env bash
#
# Backup do banco (ALE-119). Snapshot consistente MESMO com a mesa rodando:
#
#   pnpm db:backup              # produção — o banco da mesa
#   pnpm db:backup dev          # o de desenvolvimento
#
# Usa o `.backup` do sqlite3, NUNCA `cp`. Com WAL ligado (é o nosso caso, ver
# db.Open), o arquivo `.db` sozinho não contém as transações que ainda estão no
# `-wal`: copiá-lo enquanto o servidor escreve produz um banco velho, e a cópia
# não dá erro nenhum — o estrago só aparece na hora de restaurar. O `.backup`
# usa a API de backup online do SQLite, que lê um snapshot coerente das duas
# partes e espera quem estiver escrevendo.
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENGINE_DIR="$REPO_ROOT/engine-go"
BACKUP_DIR="${BACKUP_DIR:-$REPO_ROOT/backups}"

die() {
  echo "backup: $*" >&2
  exit 1
}

# resolve_env normaliza o apelido do ambiente para o sufixo do arquivo `.env`.
resolve_env() {
  case "${1:-production}" in
    prod | production) echo "production" ;;
    dev | development) echo "development" ;;
    *) die "ambiente $1 desconhecido; use 'dev' ou 'prod'" ;;
  esac
}

# database_path lê o DATABASE_URL do arquivo do ambiente e devolve o caminho do
# banco. O env do processo vence o arquivo, a MESMA regra do servidor (api/
# envfile.go) — o loader Go não serve aqui porque backup se faz com o servidor
# parado também. Só o formato que os nossos arquivos usam: `CHAVE=valor` cru.
database_path() {
  local env_file="$1" url path
  url="${DATABASE_URL:-$(sed -n 's/^DATABASE_URL=//p' "$env_file")}"
  [ -n "$url" ] || die "DATABASE_URL não está em $env_file nem no ambiente"
  path="${url#file:}"
  # Os caminhos são relativos a engine-go/, que é o CWD do servidor.
  echo "$ENGINE_DIR/${path#./}"
}

APP_ENV="$(resolve_env "${1:-}")"
ENV_FILE="$ENGINE_DIR/.env.$APP_ENV"
[ -f "$ENV_FILE" ] || die "$ENV_FILE não existe — copie o .env.production.example"

DB_PATH="$(database_path "$ENV_FILE")"
[ -f "$DB_PATH" ] || die "banco $DB_PATH não existe (o servidor de $APP_ENV já subiu alguma vez?)"

mkdir -p "$BACKUP_DIR"
OUT="$BACKUP_DIR/t20-$APP_ENV-$(date +%Y%m%d-%H%M%S).db"

sqlite3 "$DB_PATH" ".timeout 5000" ".backup '$OUT'"

# Uma cópia que não abre não é backup, e descobrir isso no dia da restauração é
# tarde: o integrity_check custa milissegundos num banco desta escala.
CHECK="$(sqlite3 "$OUT" "PRAGMA integrity_check;")"
[ "$CHECK" = "ok" ] || die "a cópia saiu corrompida ($CHECK) — $OUT"

echo "backup de $APP_ENV: $OUT ($(du -h "$OUT" | cut -f1))"
echo "restaurar: pare o servidor, depois"
echo "  cp '$OUT' '$DB_PATH' && rm -f '$DB_PATH-wal' '$DB_PATH-shm'"
