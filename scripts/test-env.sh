#!/usr/bin/env bash
# Proves .env values reach every reader unchanged: the shell loader
# (scripts/with-env.sh) and both Compose services. Run by `make test`.
set -euo pipefail
cd "$(dirname "$0")/.."

want='p@ss $word #1 "dq" /?&=:% end'
tmp=$(mktemp)
# A separate project name keeps the test away from the real dev database
compose=(docker compose -p go-next-env-test --env-file "$tmp")
trap '"${compose[@]}" down -v --remove-orphans >/dev/null 2>&1; rm -f "$tmp"' EXIT
cat >"$tmp" <<ENV
PGUSER=app
PGPASSWORD='$want'
PGDATABASE=app
ENV

fail=0
check() { # name got
  if [[ $2 == "$want" ]]; then
    echo "  ok    $1"
  else
    echo "  FAIL  $1: got [$2]"
    fail=1
  fi
}

check "with-env.sh" "$(ENV_FILE=$tmp ./scripts/with-env.sh printenv PGPASSWORD)"

# What a container really receives (the image is small and cached after first use)
check "compose db container" \
  "$("${compose[@]}" run --rm --no-deps -T --entrypoint printenv db POSTGRES_PASSWORD 2>/dev/null)"

# The api image is built from source, so read its environment from the rendered
# config instead. Compose escapes "$" as "$$" there; undo that one escape.
config=$("${compose[@]}" --profile full config --format json)
check "compose api environment" \
  "$(jq -r '.services.api.environment.PGPASSWORD' <<<"$config" | sed 's/\$\$/$/g')"

exit "$fail"
