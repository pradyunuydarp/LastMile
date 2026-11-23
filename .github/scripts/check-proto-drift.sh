#!/usr/bin/env bash
# Fail if gen/go is out of date with api/*.proto.
#
#   .github/scripts/check-proto-drift.sh      # needs protoc, protoc-gen-go, protoc-gen-go-grpc on PATH
#
# Regenerates every stub into a temp dir (the same command as `make proto`)
# and compares it with the committed gen/go. The committed files were made
# with different protoc releases, so the one header line that records the
# protoc version ("// \tprotoc  vX" / "// - protoc  vX") is ignored. The
# plugin version lines are NOT ignored: those must match exactly.
set -euo pipefail
cd "$(dirname "$0")/../.."

work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

mkdir -p "$work/new" # protoc does not create its output directory
protoc --version
protoc -I . --go_out=module=lastmile:"$work/new" --go-grpc_out=module=lastmile:"$work/new" api/*.proto \
  || { echo "::error::protoc failed"; exit 1; }

# Copy both trees with the protoc-version header line removed.
normalize() { # src dst
  (cd "$1" && find . -type f -name '*.go' -print0) | while IFS= read -r -d '' f; do
    mkdir -p "$2/$(dirname "$f")"
    sed -E '/^\/\/[[:space:]]+(-[[:space:]]+)?protoc[[:space:]]+(v|\()/d' "$1/$f" >"$2/$f"
  done
}
normalize gen/go "$work/committed"
normalize "$work/new/gen/go" "$work/generated"

if (cd "$work" && diff -ru committed generated) >"$work/diff.txt"; then
  echo "gen/go is up to date with api/*.proto ($(find gen/go -name '*.go' | wc -l | tr -d ' ') files)"
  exit 0
fi

echo "::error title=Generated code is stale::gen/go does not match api/*.proto. Run 'make proto' and commit the result."
cat "$work/diff.txt"
exit 1
