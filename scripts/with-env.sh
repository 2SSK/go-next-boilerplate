#!/usr/bin/env bash
# Runs a command with .env loaded into its environment: ./scripts/with-env.sh <cmd> [args...]
#
# The file is read by the shell, never by Make, so values reach programs exactly as
# written. Two rules, both matching Docker Compose (the other reader of .env):
#   - wrap any value that contains spaces, #, $, or quotes in single quotes:
#       PGPASSWORD='p@ss $word #1'
#   - a variable already set in the environment wins over the file:
#       PGPORT=55432 make dev
# ENV_FILE overrides the file (the env tests use it).
set -euo pipefail
cd "$(dirname "$0")/.."

env_file=${ENV_FILE:-.env}
if [[ -f $env_file ]]; then
  # Remember what the caller set, load the file, then put the caller's values back.
  declare -A caller
  while IFS='=' read -r -d '' key value; do
    caller[$key]=$value
  done < <(env -0)

  set -a
  # shellcheck disable=SC1090
  . "$env_file"
  set +a

  for key in "${!caller[@]}"; do
    export "$key=${caller[$key]}"
  done
fi
exec "$@"
