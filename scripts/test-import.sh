#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

cd "$ROOT_DIR/back"

if [ -f .env ]; then
  set -a
  # shellcheck disable=SC1091
  source .env
  set +a
fi

if [ -z "${DATABASE_URL:-}" ]; then
  echo "DATABASE_URL 未设置，无法清空数据库。"
  exit 1
fi

# Ensure psql in PATH for Homebrew installs
if ! command -v psql >/dev/null 2>&1 && command -v brew >/dev/null 2>&1; then
  pg_prefix="$(brew --prefix postgresql@15 2>/dev/null || true)"
  if [ -n "$pg_prefix" ] && [ -d "$pg_prefix/bin" ]; then
    export PATH="$pg_prefix/bin:$PATH"
  fi
fi

if ! command -v psql >/dev/null 2>&1; then
  echo "psql 未找到，请先安装 PostgreSQL。"
  exit 1
fi

# Clear database
psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"

# Import course (FDS) and papers

go run ./cmd/import_courses --path "$ROOT_DIR/data/course" --only-name "数据结构基础"

go run ./cmd/import_papers --path "$ROOT_DIR/data/papers/CS1018F" --course-id "CS1018F"

go run ./cmd/seed_admin --username "admin" --password "admin@123" --email "admin@example.com" --nickname "admin"
