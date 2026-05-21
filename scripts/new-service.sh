#!/usr/bin/env bash
# new-service.sh — scaffold a new kris-<name> service based on kris-alpha.
#
# Usage:
#   scripts/new-service.sh <name> <grpc_port> <http_port> <other_port>
#
# Example:
#   scripts/new-service.sh worker 50054 8086 8087
#
# Output:
#   kris-<name>/        compile-ready scaffold, default middleware chain only
#   go.work             updated with the new module
#
# After:
#   cd kris-<name> && make build
#   ./bin/<name> -grpc=:50054 -http=:8086 -other=:8087
#   curl http://127.0.0.1:8087/healthz
set -euo pipefail

usage() {
  echo "usage: $0 <name> <grpc_port> <http_port> <other_port>" >&2
  echo "example: $0 worker 50054 8086 8087" >&2
  exit 1
}

[ "$#" -eq 4 ] || usage

NAME="$1"
GRPC_PORT="$2"
HTTP_PORT="$3"
OTHER_PORT="$4"

[[ "$NAME" =~ ^[a-z][a-z0-9_-]*$ ]] || { echo "error: name must be lowercase [a-z0-9_-]" >&2; exit 1; }
[[ "$GRPC_PORT" =~ ^[0-9]+$ ]] && [[ "$HTTP_PORT" =~ ^[0-9]+$ ]] && [[ "$OTHER_PORT" =~ ^[0-9]+$ ]] || {
  echo "error: ports must be numeric" >&2; exit 1;
}

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
TEMPLATE="$REPO_ROOT/kris-alpha"
TARGET="$REPO_ROOT/kris-$NAME"

[ -d "$TEMPLATE" ] || { echo "error: template missing at $TEMPLATE" >&2; exit 1; }
[ -e "$TARGET" ] && { echo "error: $TARGET already exists" >&2; exit 1; }

echo "==> Cloning template $TEMPLATE -> $TARGET"
rsync -a \
  --exclude=bin \
  --exclude='go.sum' \
  "$TEMPLATE/" "$TARGET/"

echo "==> Renaming cmd/alpha -> cmd/$NAME"
mv "$TARGET/cmd/alpha" "$TARGET/cmd/$NAME"

echo "==> Rewriting identifiers (incl. helm chart name + tpl)"
# NB: portable sed — avoid \b (BSD sed treats it literally).
# Disambiguate bare "alpha" via explicit surrounding chars / line anchors.
find "$TARGET" -type f \( \
    -name "*.go" -o -name "Makefile" -o -name "go.mod" \
    -o -name "*.yaml" -o -name "*.yml" -o -name "*.md" \
    -o -name "*.tpl" -o -name "*.txt" \
    -o -name "Dockerfile" -o -name ".gitignore" \
  \) -print0 |
while IFS= read -r -d '' f; do
  sed -i.bak \
    -e "s|github.com/kris/go-infrastructure/kris-alpha|github.com/kris/go-infrastructure/kris-$NAME|g" \
    -e "s|kris-alpha|kris-$NAME|g" \
    -e "s|kris/alpha|kris/$NAME|g" \
    -e "s|\"alpha\"|\"$NAME\"|g" \
    -e "s|cmd/alpha|cmd/$NAME|g" \
    -e "s|./cmd/alpha|./cmd/$NAME|g" \
    -e "s|/alpha\"|/$NAME\"|g" \
    -e "s|/alpha$|/$NAME|g" \
    -e "s|/alpha |/$NAME |g" \
    -e "s|:= alpha$|:= $NAME|g" \
    "$f"
  rm -f "$f.bak"
done

echo "==> Patching ports (main.go + Dockerfile + helm values)"
patch_ports() {
  local f="$1"
  [ -f "$f" ] || return 0
  sed -i.bak \
    -e "s|:50051|:$GRPC_PORT|g" \
    -e "s|:8080|:$HTTP_PORT|g" \
    -e "s|:8081|:$OTHER_PORT|g" \
    -e "s|EXPOSE 50051|EXPOSE $GRPC_PORT|g" \
    -e "s|EXPOSE 8080|EXPOSE $HTTP_PORT|g" \
    -e "s|EXPOSE 8081|EXPOSE $OTHER_PORT|g" \
    -e "s|grpc: 50051|grpc: $GRPC_PORT|g" \
    -e "s|http: 8080|http: $HTTP_PORT|g" \
    -e "s|sidecar: 8081|sidecar: $OTHER_PORT|g" \
    "$f"
  rm -f "$f.bak"
}
patch_ports "$TARGET/cmd/$NAME/main.go"
patch_ports "$TARGET/Dockerfile"
patch_ports "$TARGET/helm-charts/values.yaml"

echo "==> Adding to go.work"
if ! grep -q "kris-$NAME" "$REPO_ROOT/go.work"; then
  awk -v name="kris-$NAME" '
    /^use \(/ { print; print "\t./"name; next }
    { print }
  ' "$REPO_ROOT/go.work" > "$REPO_ROOT/go.work.tmp" && mv "$REPO_ROOT/go.work.tmp" "$REPO_ROOT/go.work"
fi

echo "==> go mod tidy + build"
cd "$TARGET"
go mod tidy
go build -o "./bin/$NAME" "./cmd/$NAME"

echo ""
echo "============================================"
echo "[ok] Service kris-$NAME ready"
echo "  layout : $TARGET"
echo "  ports  : grpc=$GRPC_PORT http=$HTTP_PORT other=$OTHER_PORT"
echo "  run    : ./kris-$NAME/bin/$NAME"
echo "  probe  : curl http://127.0.0.1:$OTHER_PORT/healthz"
echo "============================================"
