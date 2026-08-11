#!/usr/bin/env bash
# Build the engine-go rules engine to WASM + copy Go's JS glue into each front's
# public/ so Vite serves them as static assets. A front loads /engine/t20.wasm
# lazily and runs the SAME Go rules as the server — single source of truth, no
# TS derive duplication (project_front_decouple_catalog Fase 3 → WASM).
#
# Emits into BOTH fronts while the SolidJS migration runs (ALE-63): the Solid app
# derives the sheet through the same engine, and without the artifact its
# synchronous accessors return nothing — the sheet then reads `.expertises` off
# undefined instead of failing loudly (ALE-83). Drop the React target at cutover.
#
# Artifacts are gitignored; run this before dev/build (or in CI) whenever the
# engine changes. Needs the Go toolchain.
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"        # frontend/
repo="$(cd "$here/.." && pwd)"
engine_dir="$(cd "$repo/engine-go" && pwd)"
targets=("$here/public/engine" "$repo/frontend-solid/public/engine")

goroot="$(cd "$engine_dir" && go env GOROOT)"
glue="$goroot/lib/wasm/wasm_exec.js"
[ -f "$glue" ] || glue="$goroot/misc/wasm/wasm_exec.js"       # older Go layout

primary="${targets[0]}"
mkdir -p "$primary"

echo "building engine-go → wasm…"
(cd "$engine_dir" && GOOS=js GOARCH=wasm go build -o "$primary/t20.wasm" ./cmd/wasm)
cp "$glue" "$primary/wasm_exec.js"

# Same bytes in every front — built once, copied, so the two can never drift.
for out_dir in "${targets[@]:1}"; do
  [ -d "$(dirname "$out_dir")" ] || continue
  mkdir -p "$out_dir"
  cp "$primary/t20.wasm" "$primary/wasm_exec.js" "$out_dir/"
done

raw=$(du -h "$primary/t20.wasm" | cut -f1)
gz=$(gzip -c "$primary/t20.wasm" | wc -c | awk '{printf "%.2f MB gz", $1/1048576}')
echo "ok: t20.wasm ($raw raw / $gz) + wasm_exec.js → ${#targets[@]} front(s)"
