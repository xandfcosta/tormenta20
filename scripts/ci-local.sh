#!/usr/bin/env bash
#
# Roda aqui o que o `.github/workflows/ci.yml` confere, mais o `vet` e o `gofmt` do guia.
#
#   pnpm ci:local              # tudo, com o e2e
#   pnpm ci:local --sem-e2e    # para iterar
#   bash scripts/ci-local.sh   # em worktree: o pnpm recusa o `node_modules` em symlink
#
# Cada passo dá o próprio veredito, e um que falha não impede os outros: uma cadeia com `&&` para no
# primeiro e deixa o resto parecendo aprovado. Por isso também os binários são chamados direto, como nos
# scripts de CSS e de JS, e o build vai para um arquivo temporário: `pnpm build` regravaria o
# `bin/t20-api`, que é o binário da produção.
#
# A CI compara o artefato com o commit, num clone limpo. Aqui pode haver trabalho em andamento, então a
# comparação é do antes com o depois de regenerar: o que mudou estava desatualizado.
set -u

cd "$(dirname "$0")/.." || exit 2
RUN_E2E=1
for argument in "$@"; do
  case "$argument" in
    --sem-e2e) RUN_E2E=0 ;;
    *) echo "opção desconhecida: $argument (a única é --sem-e2e)" >&2; exit 2 ;;
  esac
done

LOGS="$(mktemp -d)"
FAILED=()

# changed_files_hashes é a impressão de tudo o que difere do commit, arquivo por arquivo.
changed_files_hashes() {
  git -c core.fsmonitor=false ls-files --modified --others --exclude-standard | sort -u | while read -r file; do
    if [ -e "$file" ]; then sha256sum "$file"; else echo "apagado  $file"; fi
  done
}

record_failure() {
  FAILED+=("$1")
  echo "FALHOU  $1"
}

# run_step roda um comando; se ele falhar, mostra o fim da saída e guarda o caminho da inteira.
run_step() {
  local name="$1" log="$LOGS/$(echo "$1" | tr -c 'a-z0-9' '-').log"
  shift
  if "$@" >"$log" 2>&1; then
    echo "ok      $name"
    return
  fi
  record_failure "$name"
  tail -n 30 "$log"
  echo "        saída inteira: $log"
}

# regenerated_step roda um gerador, e reprova se ele mudou algum arquivo.
regenerated_step() {
  local name="$1" before after
  shift
  before="$(changed_files_hashes)"
  run_step "$name" "$@"
  after="$(changed_files_hashes)"
  if [ "$before" != "$after" ]; then
    record_failure "$name: arquivo desatualizado, já regenerado aqui. Confira e commite:"
    diff <(echo "$before") <(echo "$after") | grep '^>' | awk '{print "        " $NF}'
  fi
}

# Os guardas varrem `git ls-files`: arquivo fora do índice não é medido, e a suíte fica verde sem ele.
OUTSIDE="$(git -c core.fsmonitor=false ls-files --others --exclude-standard)"
if [ -n "$OUTSIDE" ]; then
  record_failure "arquivo fora do índice, que os guardas não medem. Rode git add:"
  echo "$OUTSIDE" | sed 's/^/        /'
else
  echo "ok      nenhum arquivo fora do índice"
fi

regenerated_step "ilhas de JS" ./engine-go/scripts/build-js.sh
regenerated_step "ícones" node engine-go/scripts/gen-icons-templ.mjs
regenerated_step "templ" bash -c 'cd engine-go && go tool templ generate'
regenerated_step "folha de estilo" ./engine-go/scripts/build-css.sh
regenerated_step "biome" ./node_modules/.bin/biome lint --write .

UNFORMATTED="$(cd engine-go && gofmt -l .)"
if [ -n "$UNFORMATTED" ]; then
  record_failure "gofmt:"
  echo "$UNFORMATTED" | sed 's/^/        /'
else
  echo "ok      gofmt"
fi
run_step "go vet" bash -c 'cd engine-go && go vet ./...'
run_step "go test" bash -c 'cd engine-go && go test -count=1 ./...'
run_step "build" bash -c "cd engine-go && go build -o '$LOGS/t20-api' ./cmd/api"
run_step "typecheck" bash -c 'cd engine-go && ./node_modules/.bin/tsc --noEmit -p tsconfig.json'
if [ "$RUN_E2E" = 1 ]; then
  run_step "e2e" bash -c 'cd e2e && npx playwright test'
else
  echo "pulado  e2e"
fi

echo
if [ "${#FAILED[@]}" -gt 0 ]; then
  echo "A CI reprovaria em ${#FAILED[@]} passo(s)."
  exit 1
fi
if [ "$RUN_E2E" = 1 ]; then
  echo "Todos os passos passaram."
else
  echo "Todos os passos passaram, menos o e2e, que não rodou."
fi
