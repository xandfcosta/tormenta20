#!/usr/bin/env bash
# Build the engine-go rules engine to WASM + copy Go's JS glue into public/ so
# Vite serves them as static assets. The app loads /engine/t20.wasm lazily and
# runs the SAME Go rules as the server — single source of truth, no TS derive
# duplication.
#
# Artifacts are gitignored, so this must run before dev/build and in CI. It is
# wired as the `predev`/`prebuild` hook of this package; the hook went missing in
# the SolidJS cutover (6e8d895) and nothing complained, because `vite build`
# succeeds WITHOUT the wasm and the app only breaks at runtime — the synchronous
# engine accessors return nothing and the sheet reads fields off undefined.
#
# Needs the Go toolchain.
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"        # frontend/
repo="$(cd "$here/.." && pwd)"
engine_dir="$(cd "$repo/engine-go" && pwd)"
out_dir="$here/public/engine"

goroot="$(cd "$engine_dir" && go env GOROOT)"
glue="$goroot/lib/wasm/wasm_exec.js"
[ -f "$glue" ] || glue="$goroot/misc/wasm/wasm_exec.js"       # older Go layout

mkdir -p "$out_dir"

echo "building engine-go → wasm…"
(cd "$engine_dir" && GOOS=js GOARCH=wasm go build -o "$out_dir/t20.wasm" ./cmd/wasm)
cp "$glue" "$out_dir/wasm_exec.js"

raw=$(du -h "$out_dir/t20.wasm" | cut -f1)
gz=$(gzip -c "$out_dir/t20.wasm" | wc -c | awk '{printf "%.2f MB gz", $1/1048576}')
echo "ok: t20.wasm ($raw raw / $gz) + wasm_exec.js → public/engine"
