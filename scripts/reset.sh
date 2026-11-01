#!/usr/bin/env bash
# Removes the todo demo and leaves the base app: GET /, /livez, /readyz, /docs,
# and the landing page. Run through `make reset`.
#
# A feature is removed in two steps:
#   1. delete the files and folders it owns (FEATURE_PATHS)
#   2. delete the wiring it added to shared files, which sits between
#      "feature:todo:begin" and "feature:todo:end" comment lines, or on a
#      single line ending in "feature:todo"
set -euo pipefail
cd "$(dirname "$0")/.."

FEATURE=todo
FEATURE_PATHS=(
  internal/models/todo.go
  internal/models/todo_test.go
  internal/store/todo.go
  internal/store/todo_integration_test.go
  internal/handlers/todo.go
  internal/handlers/todo_test.go
  internal/database/migrations/0002_todos.sql
  web/app/todos
  web/components/todos
  web/lib/todos.ts
  web/e2e/todos.spec.ts
)

mapfile -t marked < <(grep -rlE "feature:${FEATURE}(:begin|:end)?\b" \
  --include='*.go' --include='*.ts' --include='*.tsx' --include='*.yaml' \
  --exclude-dir=node_modules --exclude-dir=.next --exclude-dir=.git . || true)

present=()
for p in "${FEATURE_PATHS[@]}"; do [[ -e $p ]] && present+=("$p"); done

if [[ ! -d web/node_modules ]]; then
  echo "web/node_modules is missing: run \`make setup\` first (reset regenerates the TypeScript client)." >&2
  exit 1
fi

if [[ ${#present[@]} -eq 0 && ${#marked[@]} -eq 0 ]]; then
  echo "Already at base: no '${FEATURE}' feature left to remove."
  exit 0
fi

for p in "${present[@]}"; do
  rm -rf -- "$p"
  echo "  removed  $p"
done

for f in "${marked[@]}"; do
  sed -i \
    -e "/feature:${FEATURE}:begin/,/feature:${FEATURE}:end/d" \
    -e "/feature:${FEATURE}\$/d" \
    -e "/feature:${FEATURE} \*\/}\$/d" \
    "$f"
  echo "  unwired  ${f#./}"
done

# Next.js caches generated route types for pages that no longer exist
rm -rf web/.next

# The spec lost the feature's paths and schemas; regenerate both sides from it
go generate ./api/
(cd web && npx openapi-typescript ../api/openapi.yaml -o lib/api/schema.ts --silent)

gofmt -w cmd internal api
go mod tidy
go build ./...
# Removing lines can leave code Prettier would reflow (a one-item array, say)
(cd web && npx prettier --write --log-level=warn .)
echo
echo "Done. The '${FEATURE}' feature is gone and the API still builds."
echo "Its table is still in your local database; 'make db-reset' starts from a clean one."
