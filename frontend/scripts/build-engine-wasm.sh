#!/usr/bin/env bash
# Build the engine-go rules engine to WASM + copy Go's JS glue into the front's
# public/ so Vite serves them as static assets. The front loads /engine/t20.wasm
# lazily and runs the SAME Go rules as the server — single source of truth, no
# TS derive duplication (project_front_decouple_catalog Fase 3 → WASM).
#
# Artifacts are gitignored; run this before dev/build (or in CI) whenever the
# engine changes. Needs the Go toolchain.
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"        # frontend/
engine_dir="$(cd "$here/../engine-go" && pwd)"
out_dir="$here/public/engine"
mkdir -p "$out_dir"

goroot="$(cd "$engine_dir" && go env GOROOT)"
glue="$goroot/lib/wasm/wasm_exec.js"
[ -f "$glue" ] || glue="$goroot/misc/wasm/wasm_exec.js"       # older Go layout

echo "building engine-go → wasm…"
(cd "$engine_dir" && GOOS=js GOARCH=wasm go build -o "$out_dir/t20.wasm" ./cmd/wasm)
cp "$glue" "$out_dir/wasm_exec.js"

raw=$(du -h "$out_dir/t20.wasm" | cut -f1)
gz=$(gzip -c "$out_dir/t20.wasm" | wc -c | awk '{printf "%.2f MB gz", $1/1048576}')
echo "ok: public/engine/t20.wasm ($raw raw / $gz) + wasm_exec.js"
