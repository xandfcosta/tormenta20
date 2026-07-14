#!/bin/sh
# Apply pending migrations against the (volume-backed) SQLite DB, then start the
# API. `pnpm deploy` doesn't create the .bin/prisma symlink, so call the CLI
# entry directly. Fails fast if a migration errors.
set -e

echo "[entrypoint] prisma migrate deploy (DATABASE_URL=$DATABASE_URL)"
node node_modules/prisma/build/index.js migrate deploy

echo "[entrypoint] starting API"
exec node dist/main
