#!/usr/bin/env bash
# Runs a command with .env loaded into its environment: ./scripts/with-env.sh <cmd> [args...]
#
# The file is read by the shell, never by Make, so values reach programs exactly as
# written. One rule covers both readers of .env (this script and Docker Compose):
# wrap any value that contains spaces, #, $, or quotes in single quotes.
#   PGPASSWORD='p@ss $word #1'
# ENV_FILE overrides the file (the env tests use it).
set -euo pipefail
cd "$(dirname "$0")/.."

env_file=${ENV_FILE:-.env}
if [[ -f $env_file ]]; then
  set -a
  # shellcheck disable=SC1090
  . "$env_file"
  set +a
fi
exec "$@"
