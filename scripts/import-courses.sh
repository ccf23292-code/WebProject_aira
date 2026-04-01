#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

cd "$ROOT_DIR/back"

if [ -z "${DATABASE_URL:-}" ] && [ ! -f .env ]; then
  cat <<'MSG'
DATABASE_URL 未设置。
请在 back/.env 中配置：
  DATABASE_URL=postgres://USER:PASSWORD@HOST:PORT/DBNAME?sslmode=disable
或在当前 shell 中 export DATABASE_URL=... 后再启动。
MSG
  exit 1
fi

go run ./cmd/import_courses --path "$ROOT_DIR/data/course"
