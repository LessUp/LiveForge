#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT_DIR}"

if [[ -f ".env.local" ]]; then
  # 允许用户在 .env.local 中覆盖环境变量
  # shellcheck disable=SC1091
  set -a
  source ".env.local"
  set +a
fi

mkdir -p "${ROOT_DIR}/records" "${ROOT_DIR}/.gocache" "${ROOT_DIR}/.gomodcache"

export GOCACHE="${ROOT_DIR}/.gocache"
export GOMODCACHE="${ROOT_DIR}/.gomodcache"
export HTTP_ADDR="${HTTP_ADDR:-:8080}"
export ALLOWED_ORIGIN="${ALLOWED_ORIGIN:-*}"
export RECORD_DIR="${RECORD_DIR:-${ROOT_DIR}/records}"

GO_BIN="${GO_BIN:-go}"

echo "[start] 使用 HTTP_ADDR=${HTTP_ADDR} RECORD_DIR=${RECORD_DIR}"

if [[ "${SKIP_TIDY:-0}" != "1" ]]; then
  echo "[start] 运行 go mod tidy（设置 SKIP_TIDY=1 可跳过）"
  "${GO_BIN}" mod tidy
fi

exec "${GO_BIN}" run ./cmd/server
