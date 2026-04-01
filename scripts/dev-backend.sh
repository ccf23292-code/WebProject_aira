#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

cd "$ROOT_DIR/back"

detect_os() {
  local os
  os="$(uname -s | tr '[:upper:]' '[:lower:]')"
  case "$os" in
    darwin*) echo "mac" ;;
    linux*) echo "linux" ;;
    *) echo "unknown" ;;
  esac
}

ensure_command() {
  local cmd="$1"
  if ! command -v "$cmd" >/dev/null 2>&1; then
    return 1
  fi
  return 0
}

ensure_go() {
  if ensure_command go; then
    return 0
  fi

  local os
  os="$(detect_os)"
  echo "Go 未安装，开始自动安装..."
  if [ "$os" = "mac" ]; then
    if ! ensure_command brew; then
      echo "请先安装 Homebrew：https://brew.sh/"
      exit 1
    fi
    brew install go
  elif [ "$os" = "linux" ]; then
    if ensure_command apt-get; then
      sudo apt-get update
      sudo apt-get install -y golang-go
    else
      echo "未检测到 apt-get，请手动安装 Go。"
      exit 1
    fi
  else
    echo "未知系统，请手动安装 Go。"
    exit 1
  fi
}

ensure_postgres() {
  # Ensure psql is on PATH for Homebrew installs.
  if ! ensure_command psql && [ "$(detect_os)" = "mac" ] && ensure_command brew; then
    local pg_prefix
    pg_prefix="$(brew --prefix postgresql@15 2>/dev/null || true)"
    if [ -n "$pg_prefix" ] && [ -d "$pg_prefix/bin" ]; then
      export PATH="$pg_prefix/bin:$PATH"
    fi
  fi

  if ! ensure_command psql; then
    local os
    os="$(detect_os)"
    echo "PostgreSQL 未安装，开始自动安装..."
    if [ "$os" = "mac" ]; then
      if ! ensure_command brew; then
        echo "请先安装 Homebrew：https://brew.sh/"
        exit 1
      fi
      brew install postgresql@15
      brew services start postgresql@15
      # Refresh PATH after install
      local pg_prefix
      pg_prefix="$(brew --prefix postgresql@15 2>/dev/null || true)"
      if [ -n "$pg_prefix" ] && [ -d "$pg_prefix/bin" ]; then
        export PATH="$pg_prefix/bin:$PATH"
      fi
    elif [ "$os" = "linux" ]; then
      if ensure_command apt-get; then
        sudo apt-get update
        sudo apt-get install -y postgresql
        sudo systemctl enable --now postgresql
      else
        echo "未检测到 apt-get，请手动安装 PostgreSQL。"
        exit 1
      fi
    else
      echo "未知系统，请手动安装 PostgreSQL。"
      exit 1
    fi
  fi

  if ! ensure_command psql; then
    echo "无法找到 psql，请确认 PostgreSQL 已正确安装并在 PATH 中。"
    exit 1
  fi

  # Ensure service is running
  local os
  os="$(detect_os)"
  if [ "$os" = "mac" ] && ensure_command brew; then
    brew services start postgresql@15 >/dev/null 2>&1 || true
  elif [ "$os" = "linux" ] && ensure_command systemctl; then
    sudo systemctl start postgresql >/dev/null 2>&1 || true
  fi

  # Ensure role and database exist
  if [ "$os" = "linux" ]; then
    sudo -u postgres psql -v ON_ERROR_STOP=1 -d postgres -tAc "SELECT 1 FROM pg_roles WHERE rolname='postgres'" | grep -q 1 \
      || sudo -u postgres psql -v ON_ERROR_STOP=1 -d postgres -c "CREATE ROLE postgres LOGIN PASSWORD 'postgres';"
    sudo -u postgres psql -v ON_ERROR_STOP=1 -d postgres -c "ALTER ROLE postgres WITH PASSWORD 'postgres';"
    sudo -u postgres psql -v ON_ERROR_STOP=1 -d postgres -tAc "SELECT 1 FROM pg_database WHERE datname='airaweb'" | grep -q 1 \
      || sudo -u postgres psql -v ON_ERROR_STOP=1 -d postgres -c "CREATE DATABASE airaweb OWNER postgres;"
  else
    psql -v ON_ERROR_STOP=1 -d postgres -tAc "SELECT 1 FROM pg_roles WHERE rolname='postgres'" | grep -q 1 \
      || psql -v ON_ERROR_STOP=1 -d postgres -c "CREATE ROLE postgres LOGIN PASSWORD 'postgres';"
    psql -v ON_ERROR_STOP=1 -d postgres -c "ALTER ROLE postgres WITH PASSWORD 'postgres';"
    psql -v ON_ERROR_STOP=1 -d postgres -tAc "SELECT 1 FROM pg_database WHERE datname='airaweb'" | grep -q 1 \
      || psql -v ON_ERROR_STOP=1 -d postgres -c "CREATE DATABASE airaweb OWNER postgres;"
  fi
}

if [ -z "${DATABASE_URL:-}" ] && [ ! -f .env ]; then
  cat <<'MSG'
DATABASE_URL 未设置。
请在 back/.env 中配置：
  DATABASE_URL=postgres://USER:PASSWORD@HOST:PORT/DBNAME?sslmode=disable
或在当前 shell 中 export DATABASE_URL=... 后再启动。
MSG
  exit 1
fi

ensure_go
ensure_postgres

go run .
