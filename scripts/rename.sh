#!/usr/bin/env bash
# Re-brands a fresh clone: Go module path, app name, compose project, and the
# web package. Run through `make rename NAME=myapp MODULE=github.com/me/myapp`.
set -euo pipefail
cd "$(dirname "$0")/.."

NAME=${1:?usage: rename.sh <name> <module>}
MODULE=${2:?usage: rename.sh <name> <module>}

if [[ ! $NAME =~ ^[a-z][a-z0-9-]*$ ]]; then
  echo "NAME must be lowercase letters, digits, and dashes (got '$NAME')" >&2
  exit 1
fi

OLD_MODULE=$(go list -m)
# "my-app" -> "My App" for display
TITLE=$(echo "$NAME" | sed -E 's/-/ /g; s/\b(.)/\u\1/g')

go mod edit -module "$MODULE"
grep -rl --include='*.go' "\"$OLD_MODULE/" cmd internal | xargs -r sed -i "s#\"$OLD_MODULE/#\"$MODULE/#g"

sed -i -E "s/^const AppName = \".*\"/const AppName = \"$NAME\"/" internal/router/router.go
sed -i -E "s/^  title: .* API$/  title: $NAME API/" api/openapi.yaml
sed -i -E "s/^name: .*/name: $NAME/" docker-compose.yml
sed -i "s|- ${OLD_MODULE}\$|- ${MODULE}|" .golangci.yml
sed -i -E "0,/name: \".*\",/s//name: \"$TITLE\",/" web/config/site.ts
(cd web && npm pkg set name="$NAME-web")
# The lockfile records the package name twice; set it directly (no network, no re-resolve)
node -e '
  const fs = require("fs"), f = "web/package-lock.json", lock = JSON.parse(fs.readFileSync(f));
  lock.name = lock.packages[""].name = process.argv[1];
  fs.writeFileSync(f, JSON.stringify(lock, null, 2) + "\n");
' "$NAME-web"

gofmt -w cmd internal
go build ./...
echo "Renamed to $NAME ($MODULE). Update the repo link in web/config/site.ts."
